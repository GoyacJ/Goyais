package cache

import (
	"context"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisProvider struct {
	client *redis.Client
}

func NewRedisProvider(addr, password string) *RedisProvider {
	if strings.TrimSpace(addr) == "" {
		addr = "127.0.0.1:6379"
	}
	client := redis.NewClient(&redis.Options{
		Addr:     strings.TrimSpace(addr),
		Password: strings.TrimSpace(password),
	})
	return &RedisProvider{client: client}
}

func (p *RedisProvider) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if err := validateKey(key); err != nil {
		return nil, false, err
	}
	raw, err := p.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return raw, true, nil
}

func (p *RedisProvider) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if err := validateKey(key); err != nil {
		return err
	}
	return p.client.Set(ctx, key, value, ttl).Err()
}

func (p *RedisProvider) Del(ctx context.Context, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	return p.client.Del(ctx, key).Err()
}

func (p *RedisProvider) Ping(ctx context.Context) error {
	return p.client.Ping(ctx).Err()
}

func (p *RedisProvider) Name() string {
	return "redis"
}
