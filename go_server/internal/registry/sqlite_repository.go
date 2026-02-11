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
	"encoding/json"
	"errors"
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

func (r *SQLiteRepository) GetCapabilityForAccess(ctx context.Context, req command.RequestContext, capabilityID string) (Capability, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, owner_id, visibility, acl_json, provider_id, name, kind, version, input_schema, output_schema, required_permissions, egress_policy, status, created_at, updated_at
		 FROM capabilities
		 WHERE id = ? AND tenant_id = ? AND workspace_id = ?`,
		strings.TrimSpace(capabilityID),
		req.TenantID,
		req.WorkspaceID,
	)
	item, err := scanSQLiteCapability(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Capability{}, ErrCapabilityNotFound
	}
	if err != nil {
		return Capability{}, fmt.Errorf("query capability for access: %w", err)
	}
	return item, nil
}

func (r *SQLiteRepository) GetAlgorithmForAccess(ctx context.Context, req command.RequestContext, algorithmID string) (Algorithm, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, owner_id, visibility, acl_json, name, version, template_ref, defaults_json, constraints_json, dependencies_json, status, created_at, updated_at
		 FROM algorithms
		 WHERE id = ? AND tenant_id = ? AND workspace_id = ?`,
		strings.TrimSpace(algorithmID),
		req.TenantID,
		req.WorkspaceID,
	)
	item, err := scanSQLiteAlgorithm(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Algorithm{}, ErrAlgorithmNotFound
	}
	if err != nil {
		return Algorithm{}, fmt.Errorf("query algorithm for access: %w", err)
	}
	return item, nil
}

func (r *SQLiteRepository) ListCapabilities(ctx context.Context, params ListParams) (CapabilityListResult, error) {
	page, pageSize := normalizeListParams(params.Page, params.PageSize)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	filter := sqliteReadableFilter("c")
	resourceType := ResourceTypeCapability

	if strings.TrimSpace(params.Cursor) != "" {
		cursorAt, cursorID, err := command.DecodeCursor(params.Cursor)
		if err != nil {
			return CapabilityListResult{}, ErrInvalidCursor
		}

		rows, err := r.db.QueryContext(
			ctx,
			`SELECT c.id, c.tenant_id, c.workspace_id, c.owner_id, c.visibility, c.acl_json, c.provider_id, c.name, c.kind, c.version, c.input_schema, c.output_schema, c.required_permissions, c.egress_policy, c.status, c.created_at, c.updated_at
			 `+filter+`
			   AND ((c.created_at < ?) OR (c.created_at = ? AND c.id < ?))
			 ORDER BY c.created_at DESC, c.id DESC
			 LIMIT ?`,
			params.Context.TenantID,
			params.Context.WorkspaceID,
			params.Context.OwnerID,
			resourceType,
			params.Context.UserID,
			now,
			cursorAt.Format(time.RFC3339Nano),
			cursorAt.Format(time.RFC3339Nano),
			cursorID,
			pageSize,
		)
		if err != nil {
			return CapabilityListResult{}, fmt.Errorf("list capabilities by cursor: %w", err)
		}
		defer rows.Close()

		items, err := scanSQLiteCapabilities(rows)
		if err != nil {
			return CapabilityListResult{}, err
		}

		nextCursor := ""
		if len(items) == pageSize {
			last := items[len(items)-1]
			nextCursor, err = command.EncodeCursor(last.CreatedAt, last.ID)
			if err != nil {
				return CapabilityListResult{}, fmt.Errorf("encode next capability cursor: %w", err)
			}
		}

		return CapabilityListResult{Items: items, NextCursor: nextCursor, UsedCursor: true}, nil
	}

	offset := (page - 1) * pageSize
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT c.id, c.tenant_id, c.workspace_id, c.owner_id, c.visibility, c.acl_json, c.provider_id, c.name, c.kind, c.version, c.input_schema, c.output_schema, c.required_permissions, c.egress_policy, c.status, c.created_at, c.updated_at
		 `+filter+`
		 ORDER BY c.created_at DESC, c.id DESC
		 LIMIT ? OFFSET ?`,
		params.Context.TenantID,
		params.Context.WorkspaceID,
		params.Context.OwnerID,
		resourceType,
		params.Context.UserID,
		now,
		pageSize,
		offset,
	)
	if err != nil {
		return CapabilityListResult{}, fmt.Errorf("list capabilities by page: %w", err)
	}
	defer rows.Close()

	items, err := scanSQLiteCapabilities(rows)
	if err != nil {
		return CapabilityListResult{}, err
	}

	var total int64
	if err := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(1)
		 `+filter,
		params.Context.TenantID,
		params.Context.WorkspaceID,
		params.Context.OwnerID,
		resourceType,
		params.Context.UserID,
		now,
	).Scan(&total); err != nil {
		return CapabilityListResult{}, fmt.Errorf("count capabilities: %w", err)
	}

	return CapabilityListResult{Items: items, Total: total}, nil
}

func (r *SQLiteRepository) ListAlgorithms(ctx context.Context, params ListParams) (AlgorithmListResult, error) {
	page, pageSize := normalizeListParams(params.Page, params.PageSize)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	filter := sqliteReadableFilter("a")
	resourceType := ResourceTypeAlgorithm

	if strings.TrimSpace(params.Cursor) != "" {
		cursorAt, cursorID, err := command.DecodeCursor(params.Cursor)
		if err != nil {
			return AlgorithmListResult{}, ErrInvalidCursor
		}

		rows, err := r.db.QueryContext(
			ctx,
			`SELECT a.id, a.tenant_id, a.workspace_id, a.owner_id, a.visibility, a.acl_json, a.name, a.version, a.template_ref, a.defaults_json, a.constraints_json, a.dependencies_json, a.status, a.created_at, a.updated_at
			 `+filter+`
			   AND ((a.created_at < ?) OR (a.created_at = ? AND a.id < ?))
			 ORDER BY a.created_at DESC, a.id DESC
			 LIMIT ?`,
			params.Context.TenantID,
			params.Context.WorkspaceID,
			params.Context.OwnerID,
			resourceType,
			params.Context.UserID,
			now,
			cursorAt.Format(time.RFC3339Nano),
			cursorAt.Format(time.RFC3339Nano),
			cursorID,
			pageSize,
		)
		if err != nil {
			return AlgorithmListResult{}, fmt.Errorf("list algorithms by cursor: %w", err)
		}
		defer rows.Close()

		items, err := scanSQLiteAlgorithms(rows)
		if err != nil {
			return AlgorithmListResult{}, err
		}

		nextCursor := ""
		if len(items) == pageSize {
			last := items[len(items)-1]
			nextCursor, err = command.EncodeCursor(last.CreatedAt, last.ID)
			if err != nil {
				return AlgorithmListResult{}, fmt.Errorf("encode next algorithm cursor: %w", err)
			}
		}

		return AlgorithmListResult{Items: items, NextCursor: nextCursor, UsedCursor: true}, nil
	}

	offset := (page - 1) * pageSize
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT a.id, a.tenant_id, a.workspace_id, a.owner_id, a.visibility, a.acl_json, a.name, a.version, a.template_ref, a.defaults_json, a.constraints_json, a.dependencies_json, a.status, a.created_at, a.updated_at
		 `+filter+`
		 ORDER BY a.created_at DESC, a.id DESC
		 LIMIT ? OFFSET ?`,
		params.Context.TenantID,
		params.Context.WorkspaceID,
		params.Context.OwnerID,
		resourceType,
		params.Context.UserID,
		now,
		pageSize,
		offset,
	)
	if err != nil {
		return AlgorithmListResult{}, fmt.Errorf("list algorithms by page: %w", err)
	}
	defer rows.Close()

	items, err := scanSQLiteAlgorithms(rows)
	if err != nil {
		return AlgorithmListResult{}, err
	}

	var total int64
	if err := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(1)
		 `+filter,
		params.Context.TenantID,
		params.Context.WorkspaceID,
		params.Context.OwnerID,
		resourceType,
		params.Context.UserID,
		now,
	).Scan(&total); err != nil {
		return AlgorithmListResult{}, fmt.Errorf("count algorithms: %w", err)
	}

	return AlgorithmListResult{Items: items, Total: total}, nil
}

func (r *SQLiteRepository) ListProviders(ctx context.Context, params ListParams) (ProviderListResult, error) {
	page, pageSize := normalizeListParams(params.Page, params.PageSize)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	filter := sqliteReadableFilter("p")
	resourceType := ResourceTypeCapabilityProvider

	if strings.TrimSpace(params.Cursor) != "" {
		cursorAt, cursorID, err := command.DecodeCursor(params.Cursor)
		if err != nil {
			return ProviderListResult{}, ErrInvalidCursor
		}

		rows, err := r.db.QueryContext(
			ctx,
			`SELECT p.id, p.tenant_id, p.workspace_id, p.owner_id, p.visibility, p.acl_json, p.name, p.provider_type, p.endpoint, p.metadata_json, p.status, p.created_at, p.updated_at
			 `+filter+`
			   AND ((p.created_at < ?) OR (p.created_at = ? AND p.id < ?))
			 ORDER BY p.created_at DESC, p.id DESC
			 LIMIT ?`,
			params.Context.TenantID,
			params.Context.WorkspaceID,
			params.Context.OwnerID,
			resourceType,
			params.Context.UserID,
			now,
			cursorAt.Format(time.RFC3339Nano),
			cursorAt.Format(time.RFC3339Nano),
			cursorID,
			pageSize,
		)
		if err != nil {
			return ProviderListResult{}, fmt.Errorf("list providers by cursor: %w", err)
		}
		defer rows.Close()

		items, err := scanSQLiteProviders(rows)
		if err != nil {
			return ProviderListResult{}, err
		}

		nextCursor := ""
		if len(items) == pageSize {
			last := items[len(items)-1]
			nextCursor, err = command.EncodeCursor(last.CreatedAt, last.ID)
			if err != nil {
				return ProviderListResult{}, fmt.Errorf("encode next provider cursor: %w", err)
			}
		}

		return ProviderListResult{Items: items, NextCursor: nextCursor, UsedCursor: true}, nil
	}

	offset := (page - 1) * pageSize
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT p.id, p.tenant_id, p.workspace_id, p.owner_id, p.visibility, p.acl_json, p.name, p.provider_type, p.endpoint, p.metadata_json, p.status, p.created_at, p.updated_at
		 `+filter+`
		 ORDER BY p.created_at DESC, p.id DESC
		 LIMIT ? OFFSET ?`,
		params.Context.TenantID,
		params.Context.WorkspaceID,
		params.Context.OwnerID,
		resourceType,
		params.Context.UserID,
		now,
		pageSize,
		offset,
	)
	if err != nil {
		return ProviderListResult{}, fmt.Errorf("list providers by page: %w", err)
	}
	defer rows.Close()

	items, err := scanSQLiteProviders(rows)
	if err != nil {
		return ProviderListResult{}, err
	}

	var total int64
	if err := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(1)
		 `+filter,
		params.Context.TenantID,
		params.Context.WorkspaceID,
		params.Context.OwnerID,
		resourceType,
		params.Context.UserID,
		now,
	).Scan(&total); err != nil {
		return ProviderListResult{}, fmt.Errorf("count providers: %w", err)
	}

	return ProviderListResult{Items: items, Total: total}, nil
}

func (r *SQLiteRepository) HasPermission(ctx context.Context, req command.RequestContext, resourceType, resourceID, permission string, now time.Time) (bool, error) {
	if strings.TrimSpace(resourceType) == "" || strings.TrimSpace(resourceID) == "" || strings.TrimSpace(permission) == "" {
		return false, nil
	}

	var marker int
	err := r.db.QueryRowContext(
		ctx,
		`SELECT 1
		 FROM acl_entries a
		 WHERE a.tenant_id = ?
		   AND a.workspace_id = ?
		   AND a.resource_type = ?
		   AND a.resource_id = ?
		   AND a.subject_type = 'user'
		   AND a.subject_id = ?
		   AND (a.expires_at IS NULL OR a.expires_at >= ?)
		   AND EXISTS (SELECT 1 FROM json_each(a.permissions) p WHERE p.value = ?)
		 LIMIT 1`,
		req.TenantID,
		req.WorkspaceID,
		strings.TrimSpace(resourceType),
		strings.TrimSpace(resourceID),
		req.UserID,
		now.UTC().Format(time.RFC3339Nano),
		strings.ToUpper(strings.TrimSpace(permission)),
	).Scan(&marker)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query registry acl permission: %w", err)
	}
	return true, nil
}

func sqliteReadableFilter(alias string) string {
	alias = strings.TrimSpace(alias)
	return `FROM ` + tableForAlias(alias) + ` ` + alias + `
		WHERE ` + alias + `.tenant_id = ? AND ` + alias + `.workspace_id = ?
		  AND (
		    ` + alias + `.owner_id = ?
		    OR ` + alias + `.visibility = 'WORKSPACE'
		    OR EXISTS (
		      SELECT 1 FROM acl_entries acl
		      WHERE acl.tenant_id = ` + alias + `.tenant_id
		        AND acl.workspace_id = ` + alias + `.workspace_id
		        AND acl.resource_type = ?
		        AND acl.resource_id = ` + alias + `.id
		        AND acl.subject_type = 'user'
		        AND acl.subject_id = ?
		        AND (acl.expires_at IS NULL OR acl.expires_at >= ?)
		        AND EXISTS (SELECT 1 FROM json_each(acl.permissions) p WHERE p.value = 'READ')
		    )
		  )`
}

func tableForAlias(alias string) string {
	switch alias {
	case "c":
		return "capabilities"
	case "a":
		return "algorithms"
	case "p":
		return "capability_providers"
	default:
		return "capabilities"
	}
}

func normalizeListParams(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}
	return page, pageSize
}

func scanSQLiteCapabilities(rows *sql.Rows) ([]Capability, error) {
	items := make([]Capability, 0)
	for rows.Next() {
		item, err := scanSQLiteCapability(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate capabilities: %w", err)
	}
	return items, nil
}

func scanSQLiteCapability(row interface{ Scan(dest ...any) error }) (Capability, error) {
	var item Capability
	var providerID sql.NullString
	var aclRaw string
	var inputSchemaRaw string
	var outputSchemaRaw string
	var requiredPermissionsRaw string
	var egressPolicyRaw string
	var createdAt string
	var updatedAt string
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.WorkspaceID,
		&item.OwnerID,
		&item.Visibility,
		&aclRaw,
		&providerID,
		&item.Name,
		&item.Kind,
		&item.Version,
		&inputSchemaRaw,
		&outputSchemaRaw,
		&requiredPermissionsRaw,
		&egressPolicyRaw,
		&item.Status,
		&createdAt,
		&updatedAt,
	); err != nil {
		return Capability{}, err
	}
	if providerID.Valid {
		item.ProviderID = providerID.String
	}
	item.ACLJSON = decodeJSONOrDefault(json.RawMessage(strings.TrimSpace(aclRaw)), "[]")
	item.InputSchemaJSON = decodeJSONOrDefault(json.RawMessage(strings.TrimSpace(inputSchemaRaw)), "{}")
	item.OutputSchemaJSON = decodeJSONOrDefault(json.RawMessage(strings.TrimSpace(outputSchemaRaw)), "{}")
	item.RequiredPermissionsJSON = decodeJSONOrDefault(json.RawMessage(strings.TrimSpace(requiredPermissionsRaw)), "[]")
	item.EgressPolicyJSON = decodeJSONOrDefault(json.RawMessage(strings.TrimSpace(egressPolicyRaw)), "{}")
	createdAtParsed, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Capability{}, fmt.Errorf("parse capability created_at: %w", err)
	}
	updatedAtParsed, err := time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return Capability{}, fmt.Errorf("parse capability updated_at: %w", err)
	}
	item.CreatedAt = createdAtParsed.UTC()
	item.UpdatedAt = updatedAtParsed.UTC()
	return item, nil
}

func scanSQLiteAlgorithms(rows *sql.Rows) ([]Algorithm, error) {
	items := make([]Algorithm, 0)
	for rows.Next() {
		item, err := scanSQLiteAlgorithm(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate algorithms: %w", err)
	}
	return items, nil
}

func scanSQLiteAlgorithm(row interface{ Scan(dest ...any) error }) (Algorithm, error) {
	var item Algorithm
	var aclRaw string
	var defaultsRaw string
	var constraintsRaw string
	var dependenciesRaw string
	var createdAt string
	var updatedAt string
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.WorkspaceID,
		&item.OwnerID,
		&item.Visibility,
		&aclRaw,
		&item.Name,
		&item.Version,
		&item.TemplateRef,
		&defaultsRaw,
		&constraintsRaw,
		&dependenciesRaw,
		&item.Status,
		&createdAt,
		&updatedAt,
	); err != nil {
		return Algorithm{}, err
	}
	item.ACLJSON = decodeJSONOrDefault(json.RawMessage(strings.TrimSpace(aclRaw)), "[]")
	item.DefaultsJSON = decodeJSONOrDefault(json.RawMessage(strings.TrimSpace(defaultsRaw)), "{}")
	item.ConstraintsJSON = decodeJSONOrDefault(json.RawMessage(strings.TrimSpace(constraintsRaw)), "{}")
	item.DependenciesJSON = decodeJSONOrDefault(json.RawMessage(strings.TrimSpace(dependenciesRaw)), "{}")
	createdAtParsed, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return Algorithm{}, fmt.Errorf("parse algorithm created_at: %w", err)
	}
	updatedAtParsed, err := time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return Algorithm{}, fmt.Errorf("parse algorithm updated_at: %w", err)
	}
	item.CreatedAt = createdAtParsed.UTC()
	item.UpdatedAt = updatedAtParsed.UTC()
	return item, nil
}

func scanSQLiteProviders(rows *sql.Rows) ([]CapabilityProvider, error) {
	items := make([]CapabilityProvider, 0)
	for rows.Next() {
		item, err := scanSQLiteProvider(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate providers: %w", err)
	}
	return items, nil
}

func scanSQLiteProvider(row interface{ Scan(dest ...any) error }) (CapabilityProvider, error) {
	var item CapabilityProvider
	var aclRaw string
	var metadataRaw string
	var createdAt string
	var updatedAt string
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.WorkspaceID,
		&item.OwnerID,
		&item.Visibility,
		&aclRaw,
		&item.Name,
		&item.ProviderType,
		&item.Endpoint,
		&metadataRaw,
		&item.Status,
		&createdAt,
		&updatedAt,
	); err != nil {
		return CapabilityProvider{}, err
	}
	item.ACLJSON = decodeJSONOrDefault(json.RawMessage(strings.TrimSpace(aclRaw)), "[]")
	item.MetadataJSON = decodeJSONOrDefault(json.RawMessage(strings.TrimSpace(metadataRaw)), "{}")
	createdAtParsed, err := time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return CapabilityProvider{}, fmt.Errorf("parse provider created_at: %w", err)
	}
	updatedAtParsed, err := time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return CapabilityProvider{}, fmt.Errorf("parse provider updated_at: %w", err)
	}
	item.CreatedAt = createdAtParsed.UTC()
	item.UpdatedAt = updatedAtParsed.UTC()
	return item, nil
}

func decodeJSONOrDefault(raw json.RawMessage, fallback string) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(fallback)
	}
	return raw
}
