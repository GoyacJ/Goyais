// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package asset

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"goyais/internal/command"
)

type Repository interface {
	Create(ctx context.Context, in CreateInput) (Asset, error)
	GetForAccess(ctx context.Context, req command.RequestContext, id string) (Asset, error)
	List(ctx context.Context, params ListParams) (ListResult, error)
	Update(ctx context.Context, in UpdateInput) (Asset, error)
	Delete(ctx context.Context, req command.RequestContext, id string, now time.Time) (Asset, error)
	ListLineage(ctx context.Context, req command.RequestContext, assetID string) ([]LineageEdge, error)
	HasPermission(ctx context.Context, req command.RequestContext, assetID, permission string, now time.Time) (bool, error)
}

func NewRepository(dbDriver string, db *sql.DB) (Repository, error) {
	switch strings.ToLower(strings.TrimSpace(dbDriver)) {
	case "sqlite":
		return NewSQLiteRepository(db), nil
	case "postgres":
		return NewPostgresRepository(db), nil
	default:
		return nil, fmt.Errorf("unsupported asset repository driver: %s", dbDriver)
	}
}
