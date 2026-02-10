package vector

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

type redisVectorPayload struct {
	Values   []float64         `json:"values"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type RedisStackProvider struct {
	client *redis.Client
}

func NewRedisStackProvider(addr string) *RedisStackProvider {
	if strings.TrimSpace(addr) == "" {
		addr = "127.0.0.1:6379"
	}
	client := redis.NewClient(&redis.Options{
		Addr: strings.TrimSpace(addr),
	})
	return &RedisStackProvider{client: client}
}

func (p *RedisStackProvider) Upsert(ctx context.Context, namespace, id string, values []float64, metadata map[string]string) error {
	if err := validateVectorInput(namespace, id, values); err != nil {
		return err
	}
	raw, err := json.Marshal(redisVectorPayload{
		Values:   values,
		Metadata: metadata,
	})
	if err != nil {
		return fmt.Errorf("marshal vector payload: %w", err)
	}
	return p.client.Set(ctx, vectorKey(namespace, id), raw, 0).Err()
}

func (p *RedisStackProvider) Search(ctx context.Context, namespace string, query []float64, topK int) ([]SearchResult, error) {
	if err := validateSearchInput(namespace, query); err != nil {
		return nil, err
	}
	if topK <= 0 {
		topK = 10
	}

	keys, err := p.scanKeys(ctx, namespace)
	if err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(keys))
	for _, key := range keys {
		raw, err := p.client.Get(ctx, key).Bytes()
		if err == redis.Nil {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read vector key %s: %w", key, err)
		}
		var payload redisVectorPayload
		if err := json.Unmarshal(raw, &payload); err != nil {
			continue
		}
		score, ok := cosineSimilarity(query, payload.Values)
		if !ok {
			continue
		}
		id := strings.TrimPrefix(key, vectorKeyPrefix(namespace))
		results = append(results, SearchResult{
			ID:       id,
			Score:    score,
			Metadata: copyMetadata(payload.Metadata),
		})
	}

	sortSearchResults(results)
	if len(results) > topK {
		results = results[:topK]
	}
	return results, nil
}

func (p *RedisStackProvider) Ping(ctx context.Context) error {
	return p.client.Ping(ctx).Err()
}

func (p *RedisStackProvider) Name() string {
	return "redis_stack"
}

func (p *RedisStackProvider) scanKeys(ctx context.Context, namespace string) ([]string, error) {
	pattern := vectorKeyPrefix(namespace) + "*"
	cursor := uint64(0)
	keys := make([]string, 0)
	for {
		batch, nextCursor, err := p.client.Scan(ctx, cursor, pattern, 200).Result()
		if err != nil {
			return nil, fmt.Errorf("scan redis vectors: %w", err)
		}
		keys = append(keys, batch...)
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return keys, nil
}

func vectorKey(namespace, id string) string {
	return vectorKeyPrefix(namespace) + strings.TrimSpace(id)
}

func vectorKeyPrefix(namespace string) string {
	return "goyais:vector:" + strings.TrimSpace(namespace) + ":"
}
