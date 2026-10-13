package auth

import (
	"context"
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"strings"
	"testing"
	"time"
)

type testCepheidBlacklist struct {
	revoked map[string]bool
	err     error
	ttl     time.Duration
}

func (b *testCepheidBlacklist) IsRevoked(_ context.Context, id string) (bool, error) {
	return b.revoked[id], b.err
}
func (b *testCepheidBlacklist) Revoke(_ context.Context, id string, ttl time.Duration) error {
	if b.err != nil {
		return b.err
	}
	b.revoked[id] = true
	b.ttl = ttl
	return nil
}
func TestCepheidTokenLifecycle(t *testing.T) {
	b := &testCepheidBlacklist{revoked: map[string]bool{}}
	tokens, err := NewCepheidTokens(strings.Repeat("a", 32), time.Hour, b)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := tokens.Issue(17)
	if err != nil {
		t.Fatal(err)
	}
	claims, id, err := tokens.Validate(context.Background(), encoded)
	if err != nil || id != 17 {
		t.Fatalf("id=%d err=%v", id, err)
	}
	if err := tokens.Revoke(context.Background(), claims); err != nil {
		t.Fatal(err)
	}
	if b.ttl <= 0 || b.ttl > time.Hour {
		t.Fatalf("invalid TTL %v", b.ttl)
	}
	if _, _, err := tokens.Validate(context.Background(), encoded); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("revoked token accepted: %v", err)
	}
	b.revoked = map[string]bool{}
	b.err = errors.New("redis down")
	if _, _, err := tokens.Validate(context.Background(), encoded); err == nil || errors.Is(err, ErrInvalidToken) {
		t.Fatalf("redis failure not distinguished: %v", err)
	}
}
func TestInvalidCepheidTokens(t *testing.T) {
	b := &testCepheidBlacklist{revoked: map[string]bool{}}
	secret := strings.Repeat("a", 32)
	tokens, _ := NewCepheidTokens(secret, time.Hour, b)
	base := jwt.RegisteredClaims{Subject: "17", ID: "test", IssuedAt: jwt.NewNumericDate(time.Now()), ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))}
	cases := []struct {
		name   string
		claims jwt.RegisteredClaims
		method jwt.SigningMethod
		key    []byte
	}{
		{"wrong signature", base, jwt.SigningMethodHS256, []byte(strings.Repeat("b", 32))},
		{"wrong algorithm", base, jwt.SigningMethodHS384, []byte(secret)},
	}
	expired := base
	expired.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-time.Minute))
	noExpiry := base
	noExpiry.ExpiresAt = nil
	noIssued := base
	noIssued.IssuedAt = nil
	noID := base
	noID.ID = ""
	noSubject := base
	noSubject.Subject = ""
	badSubject := base
	badSubject.Subject = "0"
	future := base
	future.IssuedAt = jwt.NewNumericDate(time.Now().Add(time.Hour))
	for name, c := range map[string]jwt.RegisteredClaims{"expired": expired, "no expiry": noExpiry, "no issued": noIssued, "no jti": noID, "no subject": noSubject, "zero subject": badSubject, "future issued": future} {
		cases = append(cases, struct {
			name   string
			claims jwt.RegisteredClaims
			method jwt.SigningMethod
			key    []byte
		}{name, c, jwt.SigningMethodHS256, []byte(secret)})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := jwt.NewWithClaims(tc.method, tc.claims).SignedString(tc.key)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := tokens.Validate(context.Background(), encoded); !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("expected invalid token, got %v", err)
			}
		})
	}
	if _, _, err := tokens.Validate(context.Background(), "not-a-jwt"); !errors.Is(err, ErrInvalidToken) {
		t.Fatal(err)
	}
}
