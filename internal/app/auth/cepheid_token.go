package auth

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"strconv"
	"time"
)

const blacklistPrefix = "cepheids:jwt:blacklist:"

var ErrInvalidToken = errors.New("invalid cepheid token")

type CepheidTokenBlacklist interface {
	IsRevoked(context.Context, string) (bool, error)
	Revoke(context.Context, string, time.Duration) error
}
type RedisCepheidBlacklist struct{ Client *redis.Client }

func (b RedisCepheidBlacklist) IsRevoked(ctx context.Context, id string) (bool, error) {
	n, err := b.Client.Exists(ctx, blacklistPrefix+id).Result()
	return n > 0, err
}
func (b RedisCepheidBlacklist) Revoke(ctx context.Context, id string, ttl time.Duration) error {
	if ttl <= 0 {
		return ErrInvalidToken
	}
	return b.Client.Set(ctx, blacklistPrefix+id, "revoked", ttl).Err()
}

type CepheidTokens struct {
	secret    []byte
	Lifetime  time.Duration
	Blacklist CepheidTokenBlacklist
}

func NewCepheidTokens(secret string, lifetime time.Duration, blacklist CepheidTokenBlacklist) (*CepheidTokens, error) {
	if len(secret) < 32 || lifetime < time.Second || blacklist == nil {
		return nil, errors.New("JWT_SECRET must contain at least 32 bytes; JWT_TTL must be positive; blacklist is required")
	}
	return &CepheidTokens{secret: []byte(secret), Lifetime: lifetime, Blacklist: blacklist}, nil
}
func (t *CepheidTokens) Issue(userID uint) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{Subject: strconv.FormatUint(uint64(userID), 10), ID: uuid.NewString(), IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(t.Lifetime))}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
}
func (t *CepheidTokens) Validate(ctx context.Context, encoded string) (*jwt.RegisteredClaims, uint, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(encoded, claims, func(_ *jwt.Token) (interface{}, error) { return t.secret, nil }, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil || !token.Valid || claims.ID == "" || claims.IssuedAt == nil {
		return nil, 0, ErrInvalidToken
	}
	id, err := strconv.ParseUint(claims.Subject, 10, strconv.IntSize)
	if err != nil || id == 0 {
		return nil, 0, ErrInvalidToken
	}
	revoked, err := t.Blacklist.IsRevoked(ctx, claims.ID)
	if err != nil {
		return nil, 0, fmt.Errorf("cepheid blacklist unavailable: %w", err)
	}
	if revoked {
		return nil, 0, ErrInvalidToken
	}
	return claims, uint(id), nil
}
func (t *CepheidTokens) Revoke(ctx context.Context, claims *jwt.RegisteredClaims) error {
	return t.Blacklist.Revoke(ctx, claims.ID, time.Until(claims.ExpiresAt.Time))
}
