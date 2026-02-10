package cache

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestRedisProviderIntegration(t *testing.T) {
	addr := firstNonEmpty("GOYAIS_IT_REDIS_ADDR", "GOYAIS_CACHE_REDIS_ADDR")
	password := firstNonEmpty("GOYAIS_IT_REDIS_PASSWORD", "GOYAIS_CACHE_REDIS_PASSWORD")
	if strings.TrimSpace(addr) == "" {
		t.Skip("set GOYAIS_IT_REDIS_ADDR to enable redis integration test")
	}

	provider := NewRedisProvider(strings.TrimSpace(addr), strings.TrimSpace(password))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := provider.Ping(ctx); err != nil {
		t.Fatalf("ping redis: %v", err)
	}

	key := "it-cache-" + time.Now().UTC().Format("20060102150405.000000000")
	value := []byte("ok")
	if err := provider.Set(ctx, key, value, 10*time.Second); err != nil {
		t.Fatalf("set redis key: %v", err)
	}

	got, ok, err := provider.Get(ctx, key)
	if err != nil {
		t.Fatalf("get redis key: %v", err)
	}
	if !ok || string(got) != string(value) {
		t.Fatalf("unexpected redis value ok=%v value=%s", ok, string(got))
	}

	if err := provider.Del(ctx, key); err != nil {
		t.Fatalf("del redis key: %v", err)
	}
}

func firstNonEmpty(keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(os.Getenv(key))
		if value != "" {
			return value
		}
	}
	return ""
}
