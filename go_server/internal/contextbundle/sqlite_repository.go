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

func (r *SQLiteRepository) ListBundles(ctx context.Context, params ListParams) (ListResult, error) {
	page := params.Page
	if page <= 0 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	accessClause, accessArgs := buildSQLiteAccessClause(params.Context, now, "READ")
	baseFilter := `FROM context_bundles b
		WHERE b.tenant_id = ? AND b.workspace_id = ?
		  AND (` + accessClause + `)`

	baseArgs := []any{params.Context.TenantID, params.Context.WorkspaceID}
	baseArgs = append(baseArgs, accessArgs...)

	if strings.TrimSpace(params.Cursor) != "" {
		cursorAt, cursorID, err := command.DecodeCursor(params.Cursor)
		if err != nil {
			return ListResult{}, ErrInvalidCursor
		}

		args := make([]any, 0, len(baseArgs)+4)
		args = append(args, baseArgs...)
		args = append(args, cursorAt.UTC().Format(time.RFC3339Nano), cursorAt.UTC().Format(time.RFC3339Nano), cursorID, pageSize)
		rows, err := r.db.QueryContext(
			ctx,
			`SELECT b.id, b.tenant_id, b.workspace_id, b.owner_id, b.visibility, b.acl_json,
			        b.scope_type, b.scope_id, b.facts, b.summaries, b.refs, b.embeddings_index_refs, b.timeline,
			        b.created_at, b.updated_at
			 `+baseFilter+`
			   AND ((b.created_at < ?) OR (b.created_at = ? AND b.id < ?))
			 ORDER BY b.created_at DESC, b.id DESC
			 LIMIT ?`,
			args...,
		)
		if err != nil {
			return ListResult{}, fmt.Errorf("list context bundles by cursor: %w", err)
		}
		defer rows.Close()
		items, err := scanSQLiteBundles(rows)
		if err != nil {
			return ListResult{}, err
		}
		nextCursor := ""
		if len(items) == pageSize {
			last := items[len(items)-1]
			nextCursor, err = command.EncodeCursor(last.CreatedAt, last.ID)
			if err != nil {
				return ListResult{}, fmt.Errorf("encode context bundle cursor: %w", err)
			}
		}
		return ListResult{Items: items, NextCursor: nextCursor, UsedCursor: true}, nil
	}

	offset := (page - 1) * pageSize
	args := make([]any, 0, len(baseArgs)+2)
	args = append(args, baseArgs...)
	args = append(args, pageSize, offset)
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT b.id, b.tenant_id, b.workspace_id, b.owner_id, b.visibility, b.acl_json,
		        b.scope_type, b.scope_id, b.facts, b.summaries, b.refs, b.embeddings_index_refs, b.timeline,
		        b.created_at, b.updated_at
		 `+baseFilter+`
		 ORDER BY b.created_at DESC, b.id DESC
		 LIMIT ? OFFSET ?`,
		args...,
	)
	if err != nil {
		return ListResult{}, fmt.Errorf("list context bundles by page: %w", err)
	}
	defer rows.Close()

	items, err := scanSQLiteBundles(rows)
	if err != nil {
		return ListResult{}, err
	}

	countQuery := `SELECT COUNT(1) ` + baseFilter
	var total int64
	if err := r.db.QueryRowContext(ctx, countQuery, baseArgs...).Scan(&total); err != nil {
		return ListResult{}, fmt.Errorf("count context bundles: %w", err)
	}

	return ListResult{Items: items, Total: total}, nil
}

func (r *SQLiteRepository) GetBundleForAccess(ctx context.Context, req command.RequestContext, bundleID string) (Bundle, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, owner_id, visibility, acl_json,
		        scope_type, scope_id, facts, summaries, refs, embeddings_index_refs, timeline,
		        created_at, updated_at
		 FROM context_bundles
		 WHERE id = ? AND tenant_id = ? AND workspace_id = ?`,
		bundleID,
		req.TenantID,
		req.WorkspaceID,
	)
	item, err := scanSQLiteBundle(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Bundle{}, ErrNotFound
	}
	if err != nil {
		return Bundle{}, fmt.Errorf("query context bundle: %w", err)
	}
	return item, nil
}

func (r *SQLiteRepository) HasBundlePermission(ctx context.Context, req command.RequestContext, bundleID, permission string, now time.Time) (bool, error) {
	if strings.TrimSpace(bundleID) == "" || strings.TrimSpace(permission) == "" {
		return false, nil
	}
	permission = strings.ToUpper(strings.TrimSpace(permission))
	nowRaw := now.UTC().Format(time.RFC3339Nano)

	if allowed, err := r.hasSubjectPermission(ctx, req.TenantID, req.WorkspaceID, bundleID, "user", req.UserID, permission, nowRaw); err != nil {
		return false, err
	} else if allowed {
		return true, nil
	}
	for _, role := range req.Roles {
		role = strings.TrimSpace(role)
		if role == "" {
			continue
		}
		allowed, err := r.hasSubjectPermission(ctx, req.TenantID, req.WorkspaceID, bundleID, "role", role, permission, nowRaw)
		if err != nil {
			return false, err
		}
		if allowed {
			return true, nil
		}
	}
	return false, nil
}

func (r *SQLiteRepository) hasSubjectPermission(
	ctx context.Context,
	tenantID, workspaceID, bundleID, subjectType, subjectID, permission, nowRaw string,
) (bool, error) {
	if strings.TrimSpace(subjectID) == "" {
		return false, nil
	}
	var marker int
	err := r.db.QueryRowContext(
		ctx,
		`SELECT 1
		 FROM acl_entries a
		 WHERE a.tenant_id = ?
		   AND a.workspace_id = ?
		   AND a.resource_type = 'context_bundle'
		   AND a.resource_id = ?
		   AND a.subject_type = ?
		   AND a.subject_id = ?
		   AND (a.expires_at IS NULL OR a.expires_at >= ?)
		   AND EXISTS (
		     SELECT 1 FROM json_each(a.permissions) p WHERE UPPER(COALESCE(p.value, '')) = ?
		   )
		 LIMIT 1`,
		tenantID,
		workspaceID,
		bundleID,
		subjectType,
		subjectID,
		nowRaw,
		permission,
	).Scan(&marker)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query context bundle acl permission: %w", err)
	}
	return true, nil
}

func (r *SQLiteRepository) UpsertBundle(ctx context.Context, in RebuildInput) (Bundle, error) {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if len(in.Facts) == 0 {
		in.Facts = json.RawMessage(`{}`)
	}
	if len(in.Summaries) == 0 {
		in.Summaries = json.RawMessage(`{}`)
	}
	if len(in.Refs) == 0 {
		in.Refs = json.RawMessage(`{}`)
	}
	if len(in.EmbeddingsIndexRefs) == 0 {
		in.EmbeddingsIndexRefs = json.RawMessage(`[]`)
	}
	if len(in.Timeline) == 0 {
		in.Timeline = json.RawMessage(`[]`)
	}
	id := newID("cb")
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO context_bundles(
			id, tenant_id, workspace_id, owner_id, visibility, acl_json,
			scope_type, scope_id, facts, summaries, refs, embeddings_index_refs, timeline,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(tenant_id, workspace_id, owner_id, scope_type, scope_id)
		DO UPDATE SET
		  visibility = excluded.visibility,
		  facts = excluded.facts,
		  summaries = excluded.summaries,
		  refs = excluded.refs,
		  embeddings_index_refs = excluded.embeddings_index_refs,
		  timeline = excluded.timeline,
		  updated_at = excluded.updated_at`,
		id,
		in.Context.TenantID,
		in.Context.WorkspaceID,
		in.Context.OwnerID,
		in.Visibility,
		"[]",
		in.ScopeType,
		in.ScopeID,
		string(in.Facts),
		string(in.Summaries),
		string(in.Refs),
		string(in.EmbeddingsIndexRefs),
		string(in.Timeline),
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
	)
	if err != nil {
		return Bundle{}, fmt.Errorf("upsert context bundle: %w", err)
	}

	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, owner_id, visibility, acl_json,
		        scope_type, scope_id, facts, summaries, refs, embeddings_index_refs, timeline,
		        created_at, updated_at
		 FROM context_bundles
		 WHERE tenant_id = ? AND workspace_id = ? AND owner_id = ? AND scope_type = ? AND scope_id = ?`,
		in.Context.TenantID,
		in.Context.WorkspaceID,
		in.Context.OwnerID,
		in.ScopeType,
		in.ScopeID,
	)
	item, err := scanSQLiteBundle(row)
	if err != nil {
		return Bundle{}, fmt.Errorf("query upserted context bundle: %w", err)
	}
	return item, nil
}

func buildSQLiteAccessClause(req command.RequestContext, nowRaw, permission string) (string, []any) {
	permission = strings.ToUpper(strings.TrimSpace(permission))
	args := []any{req.OwnerID, nowRaw, req.UserID}
	aclSubjects := `(a.subject_type = 'user' AND a.subject_id = ?)`
	roles := make([]string, 0, len(req.Roles))
	for _, role := range req.Roles {
		role = strings.TrimSpace(role)
		if role == "" {
			continue
		}
		roles = append(roles, role)
	}
	if len(roles) > 0 {
		placeholders := make([]string, 0, len(roles))
		for _, role := range roles {
			placeholders = append(placeholders, "?")
			args = append(args, role)
		}
		aclSubjects += ` OR (a.subject_type = 'role' AND a.subject_id IN (` + strings.Join(placeholders, ",") + `))`
	}

	clause := `b.owner_id = ?
			OR b.visibility = 'WORKSPACE'
			OR EXISTS (
			  SELECT 1
			  FROM acl_entries a
			  WHERE a.tenant_id = b.tenant_id
			    AND a.workspace_id = b.workspace_id
			    AND a.resource_type = 'context_bundle'
			    AND a.resource_id = b.id
			    AND (a.expires_at IS NULL OR a.expires_at >= ?)
			    AND EXISTS (
			      SELECT 1 FROM json_each(a.permissions) p WHERE UPPER(COALESCE(p.value, '')) = '` + permission + `'
			    )
			    AND (` + aclSubjects + `)
			)`
	return clause, args
}

type sqliteRowScanner interface {
	Scan(dest ...any) error
}

func scanSQLiteBundles(rows *sql.Rows) ([]Bundle, error) {
	items := make([]Bundle, 0)
	for rows.Next() {
		item, err := scanSQLiteBundle(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate context bundles: %w", err)
	}
	return items, nil
}

func scanSQLiteBundle(row sqliteRowScanner) (Bundle, error) {
	var (
		item                Bundle
		aclRaw              string
		factsRaw            string
		summariesRaw        string
		refsRaw             string
		embeddingsRaw       string
		timelineRaw         string
		createdAtRaw        string
		updatedAtRaw        string
	)
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.WorkspaceID,
		&item.OwnerID,
		&item.Visibility,
		&aclRaw,
		&item.ScopeType,
		&item.ScopeID,
		&factsRaw,
		&summariesRaw,
		&refsRaw,
		&embeddingsRaw,
		&timelineRaw,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		return Bundle{}, err
	}
	if strings.TrimSpace(aclRaw) == "" {
		aclRaw = "[]"
	}
	if strings.TrimSpace(factsRaw) == "" {
		factsRaw = "{}"
	}
	if strings.TrimSpace(summariesRaw) == "" {
		summariesRaw = "{}"
	}
	if strings.TrimSpace(refsRaw) == "" {
		refsRaw = "{}"
	}
	if strings.TrimSpace(embeddingsRaw) == "" {
		embeddingsRaw = "[]"
	}
	if strings.TrimSpace(timelineRaw) == "" {
		timelineRaw = "[]"
	}
	item.ACLJSON = json.RawMessage(aclRaw)
	item.FactsJSON = json.RawMessage(factsRaw)
	item.SummariesJSON = json.RawMessage(summariesRaw)
	item.RefsJSON = json.RawMessage(refsRaw)
	item.EmbeddingsIndexRefsJSON = json.RawMessage(embeddingsRaw)
	item.TimelineJSON = json.RawMessage(timelineRaw)

	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return Bundle{}, fmt.Errorf("parse context bundle created_at: %w", err)
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, updatedAtRaw)
	if err != nil {
		return Bundle{}, fmt.Errorf("parse context bundle updated_at: %w", err)
	}
	item.CreatedAt = createdAt
	item.UpdatedAt = updatedAt
	return item, nil
}
