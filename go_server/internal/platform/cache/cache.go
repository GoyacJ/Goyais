// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package cache

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Provider interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Del(ctx context.Context, key string) error
	Ping(ctx context.Context) error
	Name() string
}

type Config struct {
	Provider      string
	RedisAddr     string
	RedisPassword string
}

func New(cfg Config) (Provider, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Provider)) {
	case "", "memory":
		return NewMemoryProvider(), nil
	case "redis":
		return NewRedisProvider(strings.TrimSpace(cfg.RedisAddr), strings.TrimSpace(cfg.RedisPassword)), nil
	default:
		return nil, errors.New("unsupported cache provider: " + cfg.Provider)
	}
}

func validateKey(key string) error {
	if strings.TrimSpace(key) == "" {
		return fmt.Errorf("cache key cannot be empty")
	}
	return nil
}
