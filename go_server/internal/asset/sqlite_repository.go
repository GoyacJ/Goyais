package asset

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

func (r *SQLiteRepository) Create(ctx context.Context, in CreateInput) (Asset, error) {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	visibility := strings.ToUpper(strings.TrimSpace(in.Visibility))
	if visibility == "" {
		visibility = command.VisibilityPrivate
	}
	metadata := in.Metadata
	if len(metadata) == 0 {
		metadata = json.RawMessage(`{}`)
	}
	assetID := newID("ast")
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO assets(id, tenant_id, workspace_id, owner_id, visibility, acl_json, name, type, mime, size, uri, hash, metadata_json, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		assetID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
		in.Context.OwnerID,
		visibility,
		"[]",
		strings.TrimSpace(in.Name),
		strings.TrimSpace(in.Type),
		strings.TrimSpace(in.Mime),
		in.Size,
		strings.TrimSpace(in.URI),
		strings.TrimSpace(in.Hash),
		string(metadata),
		StatusReady,
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
	)
	if err != nil {
		return Asset{}, fmt.Errorf("insert asset: %w", err)
	}
	return r.getByID(ctx, in.Context, assetID)
}

func (r *SQLiteRepository) GetForAccess(ctx context.Context, req command.RequestContext, id string) (Asset, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, workspace_id, owner_id, visibility, acl_json, name, type, mime, size, uri, hash, metadata_json, status, created_at, updated_at
		 FROM assets
		 WHERE id = ? AND tenant_id = ? AND workspace_id = ?`,
		id, req.TenantID, req.WorkspaceID,
	)
	item, err := scanAsset(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Asset{}, ErrNotFound
	}
	if err != nil {
		return Asset{}, fmt.Errorf("query asset for access: %w", err)
	}
	return item, nil
}

func (r *SQLiteRepository) List(ctx context.Context, params ListParams) (ListResult, error) {
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

	if strings.TrimSpace(params.Cursor) != "" {
		createdAt, id, err := command.DecodeCursor(params.Cursor)
		if err != nil {
			return ListResult{}, ErrInvalidCursor
		}
		rows, err := r.db.QueryContext(ctx,
			`SELECT c.id, c.tenant_id, c.workspace_id, c.owner_id, c.visibility, c.acl_json, c.name, c.type, c.mime, c.size, c.uri, c.hash, c.metadata_json, c.status, c.created_at, c.updated_at
			 FROM assets c
			 WHERE c.tenant_id = ? AND c.workspace_id = ?
			   AND (
			     c.owner_id = ?
			     OR c.visibility = 'WORKSPACE'
			     OR EXISTS (
			       SELECT 1
			       FROM acl_entries a
			       WHERE a.tenant_id = c.tenant_id
			         AND a.workspace_id = c.workspace_id
			         AND a.resource_type = 'asset'
			         AND a.resource_id = c.id
			         AND a.subject_type = 'user'
			         AND a.subject_id = ?
			         AND (a.expires_at IS NULL OR a.expires_at >= ?)
			         AND EXISTS (SELECT 1 FROM json_each(a.permissions) p WHERE p.value = 'READ')
			     )
			   )
			   AND ((c.created_at < ?) OR (c.created_at = ? AND c.id < ?))
			 ORDER BY c.created_at DESC, c.id DESC
			 LIMIT ?`,
			params.Context.TenantID,
			params.Context.WorkspaceID,
			params.Context.OwnerID,
			params.Context.UserID,
			now,
			createdAt.Format(time.RFC3339Nano),
			createdAt.Format(time.RFC3339Nano),
			id,
			pageSize,
		)
		if err != nil {
			return ListResult{}, fmt.Errorf("list assets by cursor: %w", err)
		}
		defer rows.Close()
		items, err := scanAssets(rows)
		if err != nil {
			return ListResult{}, err
		}
		nextCursor := ""
		if len(items) == pageSize {
			last := items[len(items)-1]
			nextCursor, err = command.EncodeCursor(last.CreatedAt, last.ID)
			if err != nil {
				return ListResult{}, fmt.Errorf("encode next cursor: %w", err)
			}
		}
		return ListResult{Items: items, NextCursor: nextCursor, UsedCursor: true}, nil
	}

	offset := (page - 1) * pageSize
	rows, err := r.db.QueryContext(ctx,
		`SELECT c.id, c.tenant_id, c.workspace_id, c.owner_id, c.visibility, c.acl_json, c.name, c.type, c.mime, c.size, c.uri, c.hash, c.metadata_json, c.status, c.created_at, c.updated_at
		 FROM assets c
		 WHERE c.tenant_id = ? AND c.workspace_id = ?
		   AND (
		     c.owner_id = ?
		     OR c.visibility = 'WORKSPACE'
		     OR EXISTS (
		       SELECT 1
		       FROM acl_entries a
		       WHERE a.tenant_id = c.tenant_id
		         AND a.workspace_id = c.workspace_id
		         AND a.resource_type = 'asset'
		         AND a.resource_id = c.id
		         AND a.subject_type = 'user'
		         AND a.subject_id = ?
		         AND (a.expires_at IS NULL OR a.expires_at >= ?)
		         AND EXISTS (SELECT 1 FROM json_each(a.permissions) p WHERE p.value = 'READ')
		     )
		   )
		 ORDER BY c.created_at DESC, c.id DESC
		 LIMIT ? OFFSET ?`,
		params.Context.TenantID,
		params.Context.WorkspaceID,
		params.Context.OwnerID,
		params.Context.UserID,
		now,
		pageSize,
		offset,
	)
	if err != nil {
		return ListResult{}, fmt.Errorf("list assets by page: %w", err)
	}
	defer rows.Close()
	items, err := scanAssets(rows)
	if err != nil {
		return ListResult{}, err
	}
	var total int64
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(1)
		 FROM assets c
		 WHERE c.tenant_id = ? AND c.workspace_id = ?
		   AND (
		     c.owner_id = ?
		     OR c.visibility = 'WORKSPACE'
		     OR EXISTS (
		       SELECT 1
		       FROM acl_entries a
		       WHERE a.tenant_id = c.tenant_id
		         AND a.workspace_id = c.workspace_id
		         AND a.resource_type = 'asset'
		         AND a.resource_id = c.id
		         AND a.subject_type = 'user'
		         AND a.subject_id = ?
		         AND (a.expires_at IS NULL OR a.expires_at >= ?)
		         AND EXISTS (SELECT 1 FROM json_each(a.permissions) p WHERE p.value = 'READ')
		     )
		   )`,
		params.Context.TenantID,
		params.Context.WorkspaceID,
		params.Context.OwnerID,
		params.Context.UserID,
		now,
	).Scan(&total); err != nil {
		return ListResult{}, fmt.Errorf("count assets: %w", err)
	}
	return ListResult{Items: items, Total: total}, nil
}

func (r *SQLiteRepository) Update(ctx context.Context, in UpdateInput) (Asset, error) {
	current, err := r.GetForAccess(ctx, in.Context, in.AssetID)
	if err != nil {
		return Asset{}, err
	}
	if current.Status == StatusDeleted {
		return Asset{}, ErrNotFound
	}

	name := current.Name
	if in.Name != nil {
		name = strings.TrimSpace(*in.Name)
	}
	visibility := current.Visibility
	if in.Visibility != nil {
		visibility = strings.ToUpper(strings.TrimSpace(*in.Visibility))
	}
	metadata := current.MetadataJSON
	if in.MetadataSet {
		metadata = in.Metadata
	}
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	_, err = r.db.ExecContext(ctx,
		`UPDATE assets
		 SET name = ?, visibility = ?, metadata_json = ?, updated_at = ?
		 WHERE id = ? AND tenant_id = ? AND workspace_id = ?`,
		name,
		visibility,
		string(metadata),
		now.Format(time.RFC3339Nano),
		in.AssetID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
	)
	if err != nil {
		return Asset{}, fmt.Errorf("update asset: %w", err)
	}
	return r.GetForAccess(ctx, in.Context, in.AssetID)
}

func (r *SQLiteRepository) Delete(ctx context.Context, req command.RequestContext, id string, now time.Time) (Asset, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	result, err := r.db.ExecContext(ctx,
		`UPDATE assets
		 SET status = ?, updated_at = ?
		 WHERE id = ? AND tenant_id = ? AND workspace_id = ?`,
		StatusDeleted,
		now.Format(time.RFC3339Nano),
		id,
		req.TenantID,
		req.WorkspaceID,
	)
	if err != nil {
		return Asset{}, fmt.Errorf("delete asset: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return Asset{}, fmt.Errorf("delete asset rows affected: %w", err)
	}
	if affected == 0 {
		return Asset{}, ErrNotFound
	}
	return r.GetForAccess(ctx, req, id)
}

func (r *SQLiteRepository) ListLineage(ctx context.Context, req command.RequestContext, assetID string) ([]LineageEdge, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, tenant_id, workspace_id, source_asset_id, target_asset_id, run_id, step_id, relation, created_at
		 FROM asset_lineage
		 WHERE tenant_id = ? AND workspace_id = ? AND (target_asset_id = ? OR source_asset_id = ?)
		 ORDER BY created_at DESC, id DESC`,
		req.TenantID,
		req.WorkspaceID,
		assetID,
		assetID,
	)
	if err != nil {
		return nil, fmt.Errorf("list asset lineage: %w", err)
	}
	defer rows.Close()
	return scanSQLiteLineageEdges(rows)
}

func (r *SQLiteRepository) HasPermission(ctx context.Context, req command.RequestContext, assetID, permission string, now time.Time) (bool, error) {
	if strings.TrimSpace(assetID) == "" || strings.TrimSpace(permission) == "" {
		return false, nil
	}
	permission = strings.ToUpper(strings.TrimSpace(permission))
	nowRaw := now.UTC().Format(time.RFC3339Nano)
	if allowed, err := r.hasACLPermission(ctx, req.TenantID, req.WorkspaceID, assetID, "user", req.UserID, permission, nowRaw); err != nil {
		return false, err
	} else if allowed {
		return true, nil
	}
	for _, role := range req.Roles {
		role = strings.TrimSpace(role)
		if role == "" {
			continue
		}
		allowed, err := r.hasACLPermission(ctx, req.TenantID, req.WorkspaceID, assetID, "role", role, permission, nowRaw)
		if err != nil {
			return false, err
		}
		if allowed {
			return true, nil
		}
	}
	return false, nil
}

func (r *SQLiteRepository) hasACLPermission(
	ctx context.Context,
	tenantID, workspaceID, assetID, subjectType, subjectID, permission, nowRaw string,
) (bool, error) {
	if strings.TrimSpace(subjectID) == "" {
		return false, nil
	}
	var marker int
	err := r.db.QueryRowContext(ctx,
		`SELECT 1
		 FROM acl_entries a
		 WHERE a.tenant_id = ?
		   AND a.workspace_id = ?
		   AND a.resource_type = 'asset'
		   AND a.resource_id = ?
		   AND a.subject_type = ?
		   AND a.subject_id = ?
		   AND (a.expires_at IS NULL OR a.expires_at >= ?)
		   AND EXISTS (SELECT 1 FROM json_each(a.permissions) p WHERE p.value = ?)
		 LIMIT 1`,
		tenantID,
		workspaceID,
		assetID,
		subjectType,
		subjectID,
		nowRaw,
		permission,
	).Scan(&marker)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query asset acl permission: %w", err)
	}
	return true, nil
}

func (r *SQLiteRepository) getByID(ctx context.Context, req command.RequestContext, id string) (Asset, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, tenant_id, workspace_id, owner_id, visibility, acl_json, name, type, mime, size, uri, hash, metadata_json, status, created_at, updated_at
		 FROM assets
		 WHERE id = ? AND tenant_id = ? AND workspace_id = ? AND owner_id = ?`,
		id,
		req.TenantID,
		req.WorkspaceID,
		req.OwnerID,
	)
	item, err := scanAsset(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Asset{}, ErrNotFound
	}
	if err != nil {
		return Asset{}, fmt.Errorf("query asset by id: %w", err)
	}
	return item, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanAssets(rows *sql.Rows) ([]Asset, error) {
	items := make([]Asset, 0)
	for rows.Next() {
		item, err := scanAsset(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate assets: %w", err)
	}
	return items, nil
}

func scanAsset(row rowScanner) (Asset, error) {
	var (
		item         Asset
		aclRaw       string
		metadataRaw  string
		createdAtRaw string
		updatedAtRaw string
	)
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.WorkspaceID,
		&item.OwnerID,
		&item.Visibility,
		&aclRaw,
		&item.Name,
		&item.Type,
		&item.Mime,
		&item.Size,
		&item.URI,
		&item.Hash,
		&metadataRaw,
		&item.Status,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		return Asset{}, err
	}
	if strings.TrimSpace(aclRaw) == "" {
		aclRaw = "[]"
	}
	if strings.TrimSpace(metadataRaw) == "" {
		metadataRaw = "{}"
	}
	item.ACLJSON = json.RawMessage(aclRaw)
	item.MetadataJSON = json.RawMessage(metadataRaw)
	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return Asset{}, fmt.Errorf("parse asset created_at: %w", err)
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, updatedAtRaw)
	if err != nil {
		return Asset{}, fmt.Errorf("parse asset updated_at: %w", err)
	}
	item.CreatedAt = createdAt
	item.UpdatedAt = updatedAt
	return item, nil
}

func scanSQLiteLineageEdges(rows *sql.Rows) ([]LineageEdge, error) {
	edges := make([]LineageEdge, 0)
	for rows.Next() {
		var (
			edge         LineageEdge
			sourceID     sql.NullString
			runID        sql.NullString
			stepID       sql.NullString
			createdAtRaw string
		)
		if err := rows.Scan(
			&edge.ID,
			&edge.TenantID,
			&edge.WorkspaceID,
			&sourceID,
			&edge.TargetAssetID,
			&runID,
			&stepID,
			&edge.Relation,
			&createdAtRaw,
		); err != nil {
			return nil, err
		}
		createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
		if err != nil {
			return nil, fmt.Errorf("parse lineage created_at: %w", err)
		}
		edge.CreatedAt = createdAt.UTC()
		if sourceID.Valid {
			edge.SourceAssetID = sourceID.String
		}
		if runID.Valid {
			edge.RunID = runID.String
		}
		if stepID.Valid {
			edge.StepID = stepID.String
		}
		edges = append(edges, edge)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate asset lineage: %w", err)
	}
	return edges, nil
}
