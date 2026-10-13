package auth

import (
	"github.com/redis/go-redis/v9"
	"os"
	"time"
)

func CepheidTokensFromEnv() (*CepheidTokens, *redis.Client, error) {
	lifetime := time.Hour
	if raw := os.Getenv("JWT_TTL"); raw != "" {
		value, err := time.ParseDuration(raw)
		if err != nil {
			return nil, nil, err
		}
		lifetime = value
	}
	address := os.Getenv("REDIS_ADDR")
	if address == "" {
		address = "localhost:6379"
	}
	client := redis.NewClient(&redis.Options{Addr: address, Password: os.Getenv("REDIS_PASSWORD"), DialTimeout: time.Second, ReadTimeout: time.Second, WriteTimeout: time.Second, MaxRetries: -1})
	tokens, err := NewCepheidTokens(os.Getenv("JWT_SECRET"), lifetime, RedisCepheidBlacklist{Client: client})
	if err != nil {
		client.Close()
		return nil, nil, err
	}
	return tokens, client, nil
}
