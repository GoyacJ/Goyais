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

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) ListBundles(ctx context.Context, params ListParams) (ListResult, error) {
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
	accessClause, accessArgs := buildPostgresAccessClause(params.Context, now, "READ", 3)
	baseFilter := `FROM context_bundles b
		WHERE b.tenant_id = $1 AND b.workspace_id = $2
		  AND (` + accessClause + `)`

	baseArgs := []any{params.Context.TenantID, params.Context.WorkspaceID}
	baseArgs = append(baseArgs, accessArgs...)

	if strings.TrimSpace(params.Cursor) != "" {
		cursorAt, cursorID, err := command.DecodeCursor(params.Cursor)
		if err != nil {
			return ListResult{}, ErrInvalidCursor
		}
		placeholder := len(baseArgs) + 1
		query := `SELECT b.id, b.tenant_id, b.workspace_id, b.owner_id, b.visibility, b.acl_json::text,
		        b.scope_type, b.scope_id, b.facts::text, b.summaries::text, b.refs::text, b.embeddings_index_refs::text, b.timeline::text,
		        b.created_at, b.updated_at
		 ` + baseFilter + `
		   AND ((b.created_at < $` + fmt.Sprint(placeholder) + `) OR (b.created_at = $` + fmt.Sprint(placeholder+1) + ` AND b.id < $` + fmt.Sprint(placeholder+2) + `))
		 ORDER BY b.created_at DESC, b.id DESC
		 LIMIT $` + fmt.Sprint(placeholder+3)
		args := append(baseArgs, cursorAt.UTC(), cursorAt.UTC(), cursorID, pageSize)
		rows, err := r.db.QueryContext(ctx, query, args...)
		if err != nil {
			return ListResult{}, fmt.Errorf("list context bundles by cursor: %w", err)
		}
		defer rows.Close()
		items, err := scanPostgresBundles(rows)
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
	query := `SELECT b.id, b.tenant_id, b.workspace_id, b.owner_id, b.visibility, b.acl_json::text,
	        b.scope_type, b.scope_id, b.facts::text, b.summaries::text, b.refs::text, b.embeddings_index_refs::text, b.timeline::text,
	        b.created_at, b.updated_at
	 ` + baseFilter + `
	 ORDER BY b.created_at DESC, b.id DESC
	 LIMIT $` + fmt.Sprint(len(baseArgs)+1) + ` OFFSET $` + fmt.Sprint(len(baseArgs)+2)
	args := append(baseArgs, pageSize, offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return ListResult{}, fmt.Errorf("list context bundles by page: %w", err)
	}
	defer rows.Close()

	items, err := scanPostgresBundles(rows)
	if err != nil {
		return ListResult{}, err
	}

	var total int64
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(1) `+baseFilter, baseArgs...).Scan(&total); err != nil {
		return ListResult{}, fmt.Errorf("count context bundles: %w", err)
	}
	return ListResult{Items: items, Total: total}, nil
}

func (r *PostgresRepository) GetBundleForAccess(ctx context.Context, req command.RequestContext, bundleID string) (Bundle, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, owner_id, visibility, acl_json::text,
		        scope_type, scope_id, facts::text, summaries::text, refs::text, embeddings_index_refs::text, timeline::text,
		        created_at, updated_at
		 FROM context_bundles
		 WHERE id = $1 AND tenant_id = $2 AND workspace_id = $3`,
		bundleID,
		req.TenantID,
		req.WorkspaceID,
	)
	item, err := scanPostgresBundle(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Bundle{}, ErrNotFound
	}
	if err != nil {
		return Bundle{}, fmt.Errorf("query context bundle: %w", err)
	}
	return item, nil
}

func (r *PostgresRepository) HasBundlePermission(ctx context.Context, req command.RequestContext, bundleID, permission string, now time.Time) (bool, error) {
	if strings.TrimSpace(bundleID) == "" || strings.TrimSpace(permission) == "" {
		return false, nil
	}
	permission = strings.ToUpper(strings.TrimSpace(permission))
	if allowed, err := r.hasSubjectPermission(ctx, req.TenantID, req.WorkspaceID, bundleID, "user", req.UserID, permission, now.UTC()); err != nil {
		return false, err
	} else if allowed {
		return true, nil
	}
	for _, role := range req.Roles {
		role = strings.TrimSpace(role)
		if role == "" {
			continue
		}
		allowed, err := r.hasSubjectPermission(ctx, req.TenantID, req.WorkspaceID, bundleID, "role", role, permission, now.UTC())
		if err != nil {
			return false, err
		}
		if allowed {
			return true, nil
		}
	}
	return false, nil
}

func (r *PostgresRepository) hasSubjectPermission(
	ctx context.Context,
	tenantID, workspaceID, bundleID, subjectType, subjectID, permission string,
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
		   AND a.resource_type = 'context_bundle'
		   AND a.resource_id = $3
		   AND a.subject_type = $4
		   AND a.subject_id = $5
		   AND (a.expires_at IS NULL OR a.expires_at >= $6)
		   AND a.permissions @> jsonb_build_array($7::text)
		 LIMIT 1`,
		tenantID,
		workspaceID,
		bundleID,
		subjectType,
		subjectID,
		now,
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

func (r *PostgresRepository) UpsertBundle(ctx context.Context, in RebuildInput) (Bundle, error) {
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
		) VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9::jsonb, $10::jsonb, $11::jsonb, $12::jsonb, $13::jsonb, $14, $15)
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
		now,
		now,
	)
	if err != nil {
		return Bundle{}, fmt.Errorf("upsert context bundle: %w", err)
	}

	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, owner_id, visibility, acl_json::text,
		        scope_type, scope_id, facts::text, summaries::text, refs::text, embeddings_index_refs::text, timeline::text,
		        created_at, updated_at
		 FROM context_bundles
		 WHERE tenant_id = $1 AND workspace_id = $2 AND owner_id = $3 AND scope_type = $4 AND scope_id = $5`,
		in.Context.TenantID,
		in.Context.WorkspaceID,
		in.Context.OwnerID,
		in.ScopeType,
		in.ScopeID,
	)
	item, err := scanPostgresBundle(row)
	if err != nil {
		return Bundle{}, fmt.Errorf("query upserted context bundle: %w", err)
	}
	return item, nil
}

func buildPostgresAccessClause(req command.RequestContext, now time.Time, permission string, start int) (string, []any) {
	permission = strings.ToUpper(strings.TrimSpace(permission))
	args := []any{req.OwnerID, now, req.UserID}
	position := start
	ownerPos := position
	position++
	nowPos := position
	position++
	userPos := position
	position++
	subjectClause := `(a.subject_type = 'user' AND a.subject_id = $` + fmt.Sprint(userPos) + `)`

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
			args = append(args, role)
			placeholders = append(placeholders, "$"+fmt.Sprint(position))
			position++
		}
		subjectClause += ` OR (a.subject_type = 'role' AND a.subject_id IN (` + strings.Join(placeholders, ",") + `))`
	}

	clause := `b.owner_id = $` + fmt.Sprint(ownerPos) + `
			OR b.visibility = 'WORKSPACE'
			OR EXISTS (
			  SELECT 1
			  FROM acl_entries a
			  WHERE a.tenant_id = b.tenant_id
			    AND a.workspace_id = b.workspace_id
			    AND a.resource_type = 'context_bundle'
			    AND a.resource_id = b.id
			    AND (a.expires_at IS NULL OR a.expires_at >= $` + fmt.Sprint(nowPos) + `)
			    AND a.permissions @> jsonb_build_array('` + permission + `')
			    AND (` + subjectClause + `)
			)`
	return clause, args
}

type pgRowScanner interface {
	Scan(dest ...any) error
}

func scanPostgresBundles(rows *sql.Rows) ([]Bundle, error) {
	items := make([]Bundle, 0)
	for rows.Next() {
		item, err := scanPostgresBundle(rows)
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

func scanPostgresBundle(row pgRowScanner) (Bundle, error) {
	var (
		item          Bundle
		aclRaw        string
		factsRaw      string
		summariesRaw  string
		refsRaw       string
		embeddingsRaw string
		timelineRaw   string
		createdAt     time.Time
		updatedAt     time.Time
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
		&createdAt,
		&updatedAt,
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
	item.CreatedAt = createdAt.UTC()
	item.UpdatedAt = updatedAt.UTC()
	return item, nil
}
