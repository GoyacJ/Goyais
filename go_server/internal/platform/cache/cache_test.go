package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

func TestMemoryProviderSetGetDel(t *testing.T) {
	p := NewMemoryProvider()
	ctx := context.Background()

	if err := p.Set(ctx, "k1", []byte("v1"), 0); err != nil {
		t.Fatalf("set cache: %v", err)
	}
	got, ok, err := p.Get(ctx, "k1")
	if err != nil {
		t.Fatalf("get cache: %v", err)
	}
	if !ok || string(got) != "v1" {
		t.Fatalf("unexpected cache get value ok=%v value=%s", ok, string(got))
	}

	if err := p.Del(ctx, "k1"); err != nil {
		t.Fatalf("delete cache key: %v", err)
	}
	_, ok, err = p.Get(ctx, "k1")
	if err != nil {
		t.Fatalf("get deleted key: %v", err)
	}
	if ok {
		t.Fatalf("expected deleted key not found")
	}
}

func TestMemoryProviderTTL(t *testing.T) {
	p := NewMemoryProvider()
	ctx := context.Background()

	if err := p.Set(ctx, "ttl-key", []byte("v"), 25*time.Millisecond); err != nil {
		t.Fatalf("set ttl key: %v", err)
	}

	time.Sleep(50 * time.Millisecond)
	_, ok, err := p.Get(ctx, "ttl-key")
	if err != nil {
		t.Fatalf("get ttl key: %v", err)
	}
	if ok {
		t.Fatalf("expected key to expire")
	}
}

func TestRedisProviderSetGetDel(t *testing.T) {
	redisServer := miniredis.RunT(t)
	p := NewRedisProvider(redisServer.Addr(), "")
	ctx := context.Background()

	if err := p.Ping(ctx); err != nil {
		t.Fatalf("ping redis: %v", err)
	}
	if err := p.Set(ctx, "k", []byte("v"), time.Second); err != nil {
		t.Fatalf("set redis key: %v", err)
	}
	got, ok, err := p.Get(ctx, "k")
	if err != nil {
		t.Fatalf("get redis key: %v", err)
	}
	if !ok || string(got) != "v" {
		t.Fatalf("unexpected redis value ok=%v value=%s", ok, string(got))
	}
	if err := p.Del(ctx, "k"); err != nil {
		t.Fatalf("del redis key: %v", err)
	}
	_, ok, err = p.Get(ctx, "k")
	if err != nil {
		t.Fatalf("get deleted redis key: %v", err)
	}
	if ok {
		t.Fatalf("expected redis key deleted")
	}
}
