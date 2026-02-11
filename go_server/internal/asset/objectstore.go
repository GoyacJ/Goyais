// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package asset

import (
	"context"
	"fmt"
	"strings"
	"time"

	"goyais/internal/command"
)

type ObjectStore interface {
	Put(ctx context.Context, req command.RequestContext, hash string, data []byte, now time.Time) (string, error)
	Get(ctx context.Context, uri string) ([]byte, error)
	Delete(ctx context.Context, uri string) error
	Ping(ctx context.Context) error
	Provider() string
}

type NotImplementedStore struct {
	provider string
}

type ObjectStoreOptions struct {
	Provider  string
	LocalRoot string
	Bucket    string
	Endpoint  string
	AccessKey string
	SecretKey string
	Region    string
	UseSSL    bool
}

func NewObjectStore(options ObjectStoreOptions) ObjectStore {
	switch strings.ToLower(strings.TrimSpace(options.Provider)) {
	case "", "local":
		return NewLocalStore(options.LocalRoot)
	case "minio", "s3":
		return NewS3CompatibleStore(options)
	default:
		return &NotImplementedStore{provider: options.Provider}
	}
}

func (s *NotImplementedStore) Put(context.Context, command.RequestContext, string, []byte, time.Time) (string, error) {
	return "", fmt.Errorf("%w: object_store.provider=%s", ErrNotImplemented, s.provider)
}

func (s *NotImplementedStore) Get(context.Context, string) ([]byte, error) {
	return nil, fmt.Errorf("%w: object_store.provider=%s", ErrNotImplemented, s.provider)
}

func (s *NotImplementedStore) Delete(context.Context, string) error {
	return fmt.Errorf("%w: object_store.provider=%s", ErrNotImplemented, s.provider)
}

func (s *NotImplementedStore) Ping(context.Context) error {
	return fmt.Errorf("%w: object_store.provider=%s", ErrNotImplemented, s.provider)
}

func (s *NotImplementedStore) Provider() string {
	return strings.ToLower(strings.TrimSpace(s.provider))
}
