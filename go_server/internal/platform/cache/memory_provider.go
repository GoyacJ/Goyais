// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package cache

import (
	"context"
	"sync"
	"time"
)

type memoryEntry struct {
	value     []byte
	expiresAt time.Time
}

type MemoryProvider struct {
	mu    sync.RWMutex
	items map[string]memoryEntry
}

func NewMemoryProvider() *MemoryProvider {
	return &MemoryProvider{
		items: make(map[string]memoryEntry),
	}
}

func (p *MemoryProvider) Get(_ context.Context, key string) ([]byte, bool, error) {
	if err := validateKey(key); err != nil {
		return nil, false, err
	}
	now := time.Now().UTC()

	p.mu.RLock()
	entry, ok := p.items[key]
	p.mu.RUnlock()
	if !ok {
		return nil, false, nil
	}
	if !entry.expiresAt.IsZero() && entry.expiresAt.Before(now) {
		p.mu.Lock()
		delete(p.items, key)
		p.mu.Unlock()
		return nil, false, nil
	}
	out := make([]byte, len(entry.value))
	copy(out, entry.value)
	return out, true, nil
}

func (p *MemoryProvider) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	if err := validateKey(key); err != nil {
		return err
	}
	entry := memoryEntry{
		value: append([]byte(nil), value...),
	}
	if ttl > 0 {
		entry.expiresAt = time.Now().UTC().Add(ttl)
	}
	p.mu.Lock()
	p.items[key] = entry
	p.mu.Unlock()
	return nil
}

func (p *MemoryProvider) Del(_ context.Context, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}
	p.mu.Lock()
	delete(p.items, key)
	p.mu.Unlock()
	return nil
}

func (p *MemoryProvider) Ping(context.Context) error {
	return nil
}

func (p *MemoryProvider) Name() string {
	return "memory"
}
