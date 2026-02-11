// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 Goya
// Author: Goya
// Created: 2026-02-11
// Version: v1.0.0
// Description: Goyais source file.

package algorithm

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"goyais/internal/command"
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) CreateRun(ctx context.Context, in CreateRunInput) (Run, error) {
	algorithmID := strings.TrimSpace(in.AlgorithmID)
	workflowRunID := strings.TrimSpace(in.WorkflowRunID)
	if algorithmID == "" || workflowRunID == "" {
		return Run{}, ErrInvalidRequest
	}
	status := strings.TrimSpace(in.Status)
	if status == "" {
		return Run{}, ErrInvalidRequest
	}

	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	visibility := strings.TrimSpace(in.Visibility)
	if visibility == "" {
		visibility = command.VisibilityPrivate
	}

	outputsRaw := in.Outputs
	if len(outputsRaw) == 0 {
		outputsRaw = json.RawMessage(`{}`)
	}
	assetIDsRaw, err := json.Marshal(normalizeAssetIDs(in.AssetIDs))
	if err != nil {
		return Run{}, fmt.Errorf("marshal algorithm asset ids: %w", err)
	}

	runID := newID("alg_run")
	if _, err := r.db.ExecContext(
		ctx,
		`INSERT INTO algorithm_runs(
			id, tenant_id, workspace_id, owner_id, visibility, acl_json,
			algorithm_id, workflow_run_id, command_id, outputs, asset_ids, status, error_code, message_key,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		runID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
		in.Context.OwnerID,
		visibility,
		"[]",
		algorithmID,
		workflowRunID,
		nullIfEmpty(in.CommandID),
		string(outputsRaw),
		string(assetIDsRaw),
		status,
		nullIfEmpty(in.ErrorCode),
		nullIfEmpty(in.MessageKey),
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
	); err != nil {
		return Run{}, fmt.Errorf("insert algorithm run: %w", err)
	}

	return Run{
		ID:            runID,
		TenantID:      in.Context.TenantID,
		WorkspaceID:   in.Context.WorkspaceID,
		OwnerID:       in.Context.OwnerID,
		Visibility:    visibility,
		ACLJSON:       json.RawMessage(`[]`),
		AlgorithmID:   algorithmID,
		WorkflowRunID: workflowRunID,
		CommandID:     strings.TrimSpace(in.CommandID),
		OutputsJSON:   outputsRaw,
		AssetIDsJSON:  json.RawMessage(assetIDsRaw),
		Status:        status,
		ErrorCode:     strings.TrimSpace(in.ErrorCode),
		MessageKey:    strings.TrimSpace(in.MessageKey),
		CreatedAt:     now,
		UpdatedAt:     now,
	}, nil
}

func normalizeAssetIDs(input []string) []string {
	if len(input) == 0 {
		return []string{}
	}
	result := make([]string, 0, len(input))
	seen := make(map[string]struct{}, len(input))
	for _, item := range input {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	if len(result) == 0 {
		return []string{}
	}
	return result
}

func nullIfEmpty(raw string) any {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	return value
}
