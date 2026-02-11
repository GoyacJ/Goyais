// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package vector

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type SearchResult struct {
	ID       string            `json:"id"`
	Score    float64           `json:"score"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type Provider interface {
	Upsert(ctx context.Context, namespace, id string, values []float64, metadata map[string]string) error
	Search(ctx context.Context, namespace string, query []float64, topK int) ([]SearchResult, error)
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
	case "", "sqlite":
		return NewSQLiteProvider(), nil
	case "redis_stack":
		return NewRedisStackProvider(strings.TrimSpace(cfg.RedisAddr), strings.TrimSpace(cfg.RedisPassword)), nil
	default:
		return nil, errors.New("unsupported vector provider: " + cfg.Provider)
	}
}

func validateVectorInput(namespace, id string, values []float64) error {
	if strings.TrimSpace(namespace) == "" {
		return fmt.Errorf("vector namespace cannot be empty")
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("vector id cannot be empty")
	}
	if len(values) == 0 {
		return fmt.Errorf("vector values cannot be empty")
	}
	return nil
}

func validateSearchInput(namespace string, query []float64) error {
	if strings.TrimSpace(namespace) == "" {
		return fmt.Errorf("vector namespace cannot be empty")
	}
	if len(query) == 0 {
		return fmt.Errorf("vector query cannot be empty")
	}
	return nil
}
