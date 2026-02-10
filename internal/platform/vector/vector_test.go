package vector

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
)

func TestSQLiteProviderUpsertAndSearch(t *testing.T) {
	p := NewSQLiteProvider()
	ctx := context.Background()

	if err := p.Upsert(ctx, "ns1", "a", []float64{1, 0}, map[string]string{"k": "a"}); err != nil {
		t.Fatalf("upsert a: %v", err)
	}
	if err := p.Upsert(ctx, "ns1", "b", []float64{0, 1}, map[string]string{"k": "b"}); err != nil {
		t.Fatalf("upsert b: %v", err)
	}

	results, err := p.Search(ctx, "ns1", []float64{0.9, 0.1}, 2)
	if err != nil {
		t.Fatalf("search vectors: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results got=%d", len(results))
	}
	if results[0].ID != "a" {
		t.Fatalf("expected first result a got=%s", results[0].ID)
	}
	if results[0].Score <= results[1].Score {
		t.Fatalf("expected descending score order")
	}
}

func TestRedisStackProviderUpsertAndSearch(t *testing.T) {
	redisServer := miniredis.RunT(t)
	p := NewRedisStackProvider(redisServer.Addr())
	ctx := context.Background()

	if err := p.Ping(ctx); err != nil {
		t.Fatalf("ping redis stack provider: %v", err)
	}
	if err := p.Upsert(ctx, "ns2", "a", []float64{1, 0}, map[string]string{"kind": "a"}); err != nil {
		t.Fatalf("upsert redis a: %v", err)
	}
	if err := p.Upsert(ctx, "ns2", "b", []float64{0, 1}, map[string]string{"kind": "b"}); err != nil {
		t.Fatalf("upsert redis b: %v", err)
	}

	results, err := p.Search(ctx, "ns2", []float64{0.8, 0.2}, 2)
	if err != nil {
		t.Fatalf("search redis vectors: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results got=%d", len(results))
	}
	if results[0].ID != "a" {
		t.Fatalf("expected first result a got=%s", results[0].ID)
	}
}
