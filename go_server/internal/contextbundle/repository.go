// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package contextbundle

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"goyais/internal/command"
)

type Repository interface {
	ListBundles(ctx context.Context, params ListParams) (ListResult, error)
	GetBundleForAccess(ctx context.Context, req command.RequestContext, bundleID string) (Bundle, error)
	HasBundlePermission(ctx context.Context, req command.RequestContext, bundleID, permission string, now time.Time) (bool, error)
	UpsertBundle(ctx context.Context, in RebuildInput) (Bundle, error)
}

func NewRepository(dbDriver string, db *sql.DB) (Repository, error) {
	switch strings.ToLower(strings.TrimSpace(dbDriver)) {
	case "sqlite":
		return NewSQLiteRepository(db), nil
	case "postgres":
		return NewPostgresRepository(db), nil
	default:
		return nil, fmt.Errorf("unsupported context bundle repository driver: %s", dbDriver)
	}
}
