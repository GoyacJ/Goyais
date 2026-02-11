// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package vector

import (
	"context"
	"math"
	"sync"
)

type sqliteEntry struct {
	values   []float64
	metadata map[string]string
}

type SQLiteProvider struct {
	mu   sync.RWMutex
	data map[string]map[string]sqliteEntry
}

func NewSQLiteProvider() *SQLiteProvider {
	return &SQLiteProvider{
		data: make(map[string]map[string]sqliteEntry),
	}
}

func (p *SQLiteProvider) Upsert(_ context.Context, namespace, id string, values []float64, metadata map[string]string) error {
	if err := validateVectorInput(namespace, id, values); err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.data[namespace]; !ok {
		p.data[namespace] = make(map[string]sqliteEntry)
	}
	p.data[namespace][id] = sqliteEntry{
		values:   append([]float64(nil), values...),
		metadata: copyMetadata(metadata),
	}
	return nil
}

func (p *SQLiteProvider) Search(_ context.Context, namespace string, query []float64, topK int) ([]SearchResult, error) {
	if err := validateSearchInput(namespace, query); err != nil {
		return nil, err
	}
	if topK <= 0 {
		topK = 10
	}

	p.mu.RLock()
	entries := p.data[namespace]
	p.mu.RUnlock()

	results := make([]SearchResult, 0, len(entries))
	for id, entry := range entries {
		score, ok := cosineSimilarity(query, entry.values)
		if !ok {
			continue
		}
		results = append(results, SearchResult{
			ID:       id,
			Score:    score,
			Metadata: copyMetadata(entry.metadata),
		})
	}

	sortSearchResults(results)
	if len(results) > topK {
		results = results[:topK]
	}
	return results, nil
}

func (p *SQLiteProvider) Ping(context.Context) error {
	return nil
}

func (p *SQLiteProvider) Name() string {
	return "sqlite"
}

func copyMetadata(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cosineSimilarity(query, value []float64) (float64, bool) {
	if len(query) != len(value) {
		return 0, false
	}
	var (
		dot   float64
		normQ float64
		normV float64
	)
	for idx := range query {
		dot += query[idx] * value[idx]
		normQ += query[idx] * query[idx]
		normV += value[idx] * value[idx]
	}
	if normQ == 0 || normV == 0 {
		return 0, false
	}
	return dot / (math.Sqrt(normQ) * math.Sqrt(normV)), true
}
