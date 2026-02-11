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

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(ctx context.Context, in CreateInput) (Asset, error) {
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
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO assets(id, tenant_id, workspace_id, owner_id, visibility, acl_json, name, type, mime, size, uri, hash, metadata_json, status, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9, $10, $11, $12, $13::jsonb, $14, $15, $16)`,
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
		now.UTC(),
		now.UTC(),
	)
	if err != nil {
		return Asset{}, fmt.Errorf("insert asset: %w", err)
	}
	return r.getByID(ctx, in.Context, assetID)
}

func (r *PostgresRepository) GetForAccess(ctx context.Context, req command.RequestContext, id string) (Asset, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, owner_id, visibility, acl_json::text, name, type, mime, size, uri, hash, metadata_json::text, status, created_at, updated_at
		 FROM assets
		 WHERE id = $1 AND tenant_id = $2 AND workspace_id = $3`,
		id,
		req.TenantID,
		req.WorkspaceID,
	)
	item, err := scanPostgresAsset(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Asset{}, ErrNotFound
	}
	if err != nil {
		return Asset{}, fmt.Errorf("query asset for access: %w", err)
	}
	return item, nil
}

func (r *PostgresRepository) List(ctx context.Context, params ListParams) (ListResult, error) {
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
	now := time.Now().UTC()

	if strings.TrimSpace(params.Cursor) != "" {
		createdAt, id, err := command.DecodeCursor(params.Cursor)
		if err != nil {
			return ListResult{}, ErrInvalidCursor
		}
		rows, err := r.db.QueryContext(
			ctx,
			`SELECT c.id, c.tenant_id, c.workspace_id, c.owner_id, c.visibility, c.acl_json::text, c.name, c.type, c.mime, c.size, c.uri, c.hash, c.metadata_json::text, c.status, c.created_at, c.updated_at
			 FROM assets c
			 WHERE c.tenant_id = $1 AND c.workspace_id = $2
			   AND (
			     c.owner_id = $3
			     OR c.visibility = 'WORKSPACE'
			     OR EXISTS (
			       SELECT 1
			       FROM acl_entries a
			       WHERE a.tenant_id = c.tenant_id
			         AND a.workspace_id = c.workspace_id
			         AND a.resource_type = 'asset'
			         AND a.resource_id = c.id
			         AND a.subject_type = 'user'
			         AND a.subject_id = $4
			         AND (a.expires_at IS NULL OR a.expires_at >= $5)
			         AND a.permissions @> jsonb_build_array('READ')
			     )
			   )
			   AND ((c.created_at < $6) OR (c.created_at = $7 AND c.id < $8))
			 ORDER BY c.created_at DESC, c.id DESC
			 LIMIT $9`,
			params.Context.TenantID,
			params.Context.WorkspaceID,
			params.Context.OwnerID,
			params.Context.UserID,
			now,
			createdAt.UTC(),
			createdAt.UTC(),
			id,
			pageSize,
		)
		if err != nil {
			return ListResult{}, fmt.Errorf("list assets by cursor: %w", err)
		}
		defer rows.Close()

		items, err := scanPostgresAssets(rows)
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
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT c.id, c.tenant_id, c.workspace_id, c.owner_id, c.visibility, c.acl_json::text, c.name, c.type, c.mime, c.size, c.uri, c.hash, c.metadata_json::text, c.status, c.created_at, c.updated_at
		 FROM assets c
		 WHERE c.tenant_id = $1 AND c.workspace_id = $2
		   AND (
		     c.owner_id = $3
		     OR c.visibility = 'WORKSPACE'
		     OR EXISTS (
		       SELECT 1
		       FROM acl_entries a
		       WHERE a.tenant_id = c.tenant_id
		         AND a.workspace_id = c.workspace_id
		         AND a.resource_type = 'asset'
		         AND a.resource_id = c.id
		         AND a.subject_type = 'user'
		         AND a.subject_id = $4
		         AND (a.expires_at IS NULL OR a.expires_at >= $5)
		         AND a.permissions @> jsonb_build_array('READ')
		     )
		   )
		 ORDER BY c.created_at DESC, c.id DESC
		 LIMIT $6 OFFSET $7`,
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

	items, err := scanPostgresAssets(rows)
	if err != nil {
		return ListResult{}, err
	}

	var total int64
	if err := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(1)
		 FROM assets c
		 WHERE c.tenant_id = $1 AND c.workspace_id = $2
		   AND (
		     c.owner_id = $3
		     OR c.visibility = 'WORKSPACE'
		     OR EXISTS (
		       SELECT 1
		       FROM acl_entries a
		       WHERE a.tenant_id = c.tenant_id
		         AND a.workspace_id = c.workspace_id
		         AND a.resource_type = 'asset'
		         AND a.resource_id = c.id
		         AND a.subject_type = 'user'
		         AND a.subject_id = $4
		         AND (a.expires_at IS NULL OR a.expires_at >= $5)
		         AND a.permissions @> jsonb_build_array('READ')
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

func (r *PostgresRepository) Update(ctx context.Context, in UpdateInput) (Asset, error) {
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

	_, err = r.db.ExecContext(
		ctx,
		`UPDATE assets
		 SET name = $1, visibility = $2, metadata_json = $3::jsonb, updated_at = $4
		 WHERE id = $5 AND tenant_id = $6 AND workspace_id = $7`,
		name,
		visibility,
		string(metadata),
		now,
		in.AssetID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
	)
	if err != nil {
		return Asset{}, fmt.Errorf("update asset: %w", err)
	}
	return r.GetForAccess(ctx, in.Context, in.AssetID)
}

func (r *PostgresRepository) Delete(ctx context.Context, req command.RequestContext, id string, now time.Time) (Asset, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE assets
		 SET status = $1, updated_at = $2
		 WHERE id = $3 AND tenant_id = $4 AND workspace_id = $5`,
		StatusDeleted,
		now,
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

func (r *PostgresRepository) ListLineage(ctx context.Context, req command.RequestContext, assetID string) ([]LineageEdge, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, source_asset_id, target_asset_id, run_id, step_id, relation, created_at
		 FROM asset_lineage
		 WHERE tenant_id = $1 AND workspace_id = $2 AND (target_asset_id = $3 OR source_asset_id = $4)
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
	return scanPostgresLineageEdges(rows)
}

func (r *PostgresRepository) HasPermission(ctx context.Context, req command.RequestContext, assetID, permission string, now time.Time) (bool, error) {
	if strings.TrimSpace(assetID) == "" || strings.TrimSpace(permission) == "" {
		return false, nil
	}
	permission = strings.ToUpper(strings.TrimSpace(permission))
	if allowed, err := r.hasACLPermission(ctx, req.TenantID, req.WorkspaceID, assetID, "user", req.UserID, permission, now.UTC()); err != nil {
		return false, err
	} else if allowed {
		return true, nil
	}
	for _, role := range req.Roles {
		role = strings.TrimSpace(role)
		if role == "" {
			continue
		}
		allowed, err := r.hasACLPermission(ctx, req.TenantID, req.WorkspaceID, assetID, "role", role, permission, now.UTC())
		if err != nil {
			return false, err
		}
		if allowed {
			return true, nil
		}
	}
	return false, nil
}

func (r *PostgresRepository) hasACLPermission(
	ctx context.Context,
	tenantID, workspaceID, assetID, subjectType, subjectID, permission string,
	now time.Time,
) (bool, error) {
	if strings.TrimSpace(subjectID) == "" {
		return false, nil
	}
	var marker int
	err := r.db.QueryRowContext(
		ctx,
		`SELECT 1
		 FROM acl_entries a
		 WHERE a.tenant_id = $1
		   AND a.workspace_id = $2
		   AND a.resource_type = 'asset'
		   AND a.resource_id = $3
		   AND a.subject_type = $4
		   AND a.subject_id = $5
		   AND (a.expires_at IS NULL OR a.expires_at >= $6)
		   AND a.permissions @> jsonb_build_array($7::text)
		 LIMIT 1`,
		tenantID,
		workspaceID,
		assetID,
		subjectType,
		subjectID,
		now.UTC(),
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

func (r *PostgresRepository) getByID(ctx context.Context, req command.RequestContext, id string) (Asset, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, owner_id, visibility, acl_json::text, name, type, mime, size, uri, hash, metadata_json::text, status, created_at, updated_at
		 FROM assets
		 WHERE id = $1 AND tenant_id = $2 AND workspace_id = $3 AND owner_id = $4`,
		id,
		req.TenantID,
		req.WorkspaceID,
		req.OwnerID,
	)
	item, err := scanPostgresAsset(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Asset{}, ErrNotFound
	}
	if err != nil {
		return Asset{}, fmt.Errorf("query asset by id: %w", err)
	}
	return item, nil
}

func scanPostgresAssets(rows *sql.Rows) ([]Asset, error) {
	items := make([]Asset, 0)
	for rows.Next() {
		item, err := scanPostgresAsset(rows)
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

func scanPostgresAsset(row rowScanner) (Asset, error) {
	var (
		item      Asset
		aclRaw    string
		metaRaw   string
		createdAt time.Time
		updatedAt time.Time
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
		&metaRaw,
		&item.Status,
		&createdAt,
		&updatedAt,
	); err != nil {
		return Asset{}, err
	}

	if strings.TrimSpace(aclRaw) == "" {
		aclRaw = "[]"
	}
	if strings.TrimSpace(metaRaw) == "" {
		metaRaw = "{}"
	}
	item.ACLJSON = json.RawMessage(aclRaw)
	item.MetadataJSON = json.RawMessage(metaRaw)
	item.CreatedAt = createdAt.UTC()
	item.UpdatedAt = updatedAt.UTC()
	return item, nil
}

func scanPostgresLineageEdges(rows *sql.Rows) ([]LineageEdge, error) {
	edges := make([]LineageEdge, 0)
	for rows.Next() {
		var (
			edge      LineageEdge
			sourceID  sql.NullString
			runID     sql.NullString
			stepID    sql.NullString
			createdAt time.Time
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
			&createdAt,
		); err != nil {
			return nil, err
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
