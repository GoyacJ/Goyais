// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package registry

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"goyais/internal/command"
)

type Repository interface {
	GetCapabilityForAccess(ctx context.Context, req command.RequestContext, capabilityID string) (Capability, error)
	GetAlgorithmForAccess(ctx context.Context, req command.RequestContext, algorithmID string) (Algorithm, error)
	ListCapabilities(ctx context.Context, params ListParams) (CapabilityListResult, error)
	ListAlgorithms(ctx context.Context, params ListParams) (AlgorithmListResult, error)
	ListProviders(ctx context.Context, params ListParams) (ProviderListResult, error)
	HasPermission(ctx context.Context, req command.RequestContext, resourceType, resourceID, permission string, now time.Time) (bool, error)
}

func NewRepository(dbDriver string, db *sql.DB) (Repository, error) {
	switch strings.ToLower(strings.TrimSpace(dbDriver)) {
	case "sqlite":
		return NewSQLiteRepository(db), nil
	case "postgres":
		return NewPostgresRepository(db), nil
	default:
		return nil, fmt.Errorf("unsupported registry repository driver: %s", dbDriver)
	}
}
