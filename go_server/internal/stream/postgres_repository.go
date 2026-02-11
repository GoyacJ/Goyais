package stream

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

func (r *PostgresRepository) CreateStream(ctx context.Context, in CreateStreamInput) (Stream, error) {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if len(in.State) == 0 {
		in.State = json.RawMessage(`{}`)
	}
	streamID := newID("str")
	if _, err := r.db.ExecContext(
		ctx,
		`INSERT INTO streaming_assets(
			id, tenant_id, workspace_id, owner_id, visibility, acl_json,
			path, protocol, source, endpoints_json, state_json, status,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9, $10::jsonb, $11::jsonb, $12, $13, $14)`,
		streamID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
		in.Context.OwnerID,
		in.Visibility,
		"[]",
		in.Path,
		in.Protocol,
		in.Source,
		"{}",
		string(in.State),
		StreamStatusOnline,
		now,
		now,
	); err != nil {
		return Stream{}, fmt.Errorf("insert stream: %w", err)
	}
	return r.GetStreamForAccess(ctx, in.Context, streamID)
}

func (r *PostgresRepository) GetStreamForAccess(ctx context.Context, req command.RequestContext, streamID string) (Stream, error) {
	return r.getStreamForAccess(ctx, req, streamID, false)
}

func (r *PostgresRepository) getStreamForAccess(
	ctx context.Context,
	req command.RequestContext,
	streamID string,
	includeDeleted bool,
) (Stream, error) {
	deletedFilter := "AND COALESCE(state_json->>'deleted', 'false') <> 'true'"
	if includeDeleted {
		deletedFilter = ""
	}
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, owner_id, visibility, acl_json::text, path, protocol, source, endpoints_json::text, state_json::text, status, created_at, updated_at
		 FROM streaming_assets
		 WHERE id = $1 AND tenant_id = $2 AND workspace_id = $3 `+deletedFilter,
		streamID,
		req.TenantID,
		req.WorkspaceID,
	)
	item, err := scanPostgresStream(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Stream{}, ErrStreamNotFound
	}
	if err != nil {
		return Stream{}, fmt.Errorf("query stream: %w", err)
	}
	return item, nil
}

func (r *PostgresRepository) ListStreams(ctx context.Context, params StreamListParams) (StreamListResult, error) {
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

	baseFilter := `FROM streaming_assets s
		WHERE s.tenant_id = $1 AND s.workspace_id = $2
		  AND COALESCE(s.state_json->>'deleted', 'false') <> 'true'
		  AND (
		    s.owner_id = $3
		    OR s.visibility = 'WORKSPACE'
		    OR EXISTS (
		      SELECT 1 FROM acl_entries a
		      WHERE a.tenant_id = s.tenant_id
		        AND a.workspace_id = s.workspace_id
		        AND a.resource_type = 'streaming_asset'
		        AND a.resource_id = s.id
		        AND a.subject_type = 'user'
		        AND a.subject_id = $4
		        AND (a.expires_at IS NULL OR a.expires_at >= $5)
		        AND a.permissions @> jsonb_build_array('READ')
		    )
		  )`

	if strings.TrimSpace(params.Cursor) != "" {
		cursorAt, cursorID, err := command.DecodeCursor(params.Cursor)
		if err != nil {
			return StreamListResult{}, ErrInvalidCursor
		}
		rows, err := r.db.QueryContext(
			ctx,
			`SELECT s.id, s.tenant_id, s.workspace_id, s.owner_id, s.visibility, s.acl_json::text, s.path, s.protocol, s.source, s.endpoints_json::text, s.state_json::text, s.status, s.created_at, s.updated_at
			 `+baseFilter+`
			   AND ((s.created_at < $6) OR (s.created_at = $7 AND s.id < $8))
			 ORDER BY s.created_at DESC, s.id DESC
			 LIMIT $9`,
			params.Context.TenantID,
			params.Context.WorkspaceID,
			params.Context.OwnerID,
			params.Context.UserID,
			now,
			cursorAt.UTC(),
			cursorAt.UTC(),
			cursorID,
			pageSize,
		)
		if err != nil {
			return StreamListResult{}, fmt.Errorf("list streams by cursor: %w", err)
		}
		defer rows.Close()

		items, err := scanPostgresStreams(rows)
		if err != nil {
			return StreamListResult{}, err
		}

		nextCursor := ""
		if len(items) == pageSize {
			last := items[len(items)-1]
			nextCursor, err = command.EncodeCursor(last.CreatedAt, last.ID)
			if err != nil {
				return StreamListResult{}, fmt.Errorf("encode stream cursor: %w", err)
			}
		}

		return StreamListResult{
			Items:      items,
			NextCursor: nextCursor,
			UsedCursor: true,
		}, nil
	}

	offset := (page - 1) * pageSize
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT s.id, s.tenant_id, s.workspace_id, s.owner_id, s.visibility, s.acl_json::text, s.path, s.protocol, s.source, s.endpoints_json::text, s.state_json::text, s.status, s.created_at, s.updated_at
		 `+baseFilter+`
		 ORDER BY s.created_at DESC, s.id DESC
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
		return StreamListResult{}, fmt.Errorf("list streams by page: %w", err)
	}
	defer rows.Close()

	items, err := scanPostgresStreams(rows)
	if err != nil {
		return StreamListResult{}, err
	}

	var total int64
	if err := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(1) `+baseFilter,
		params.Context.TenantID,
		params.Context.WorkspaceID,
		params.Context.OwnerID,
		params.Context.UserID,
		now,
	).Scan(&total); err != nil {
		return StreamListResult{}, fmt.Errorf("count streams: %w", err)
	}

	return StreamListResult{
		Items: items,
		Total: total,
	}, nil
}

func (r *PostgresRepository) UpdateStreamStatus(ctx context.Context, in UpdateStreamStatusInput) (Stream, error) {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if len(in.State) == 0 {
		in.State = json.RawMessage(`{}`)
	}
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE streaming_assets
		 SET status = $1, state_json = $2::jsonb, updated_at = $3
		 WHERE id = $4 AND tenant_id = $5 AND workspace_id = $6`,
		in.Status,
		string(in.State),
		now,
		in.StreamID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
	)
	if err != nil {
		return Stream{}, fmt.Errorf("update stream status: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return Stream{}, fmt.Errorf("update stream rows affected: %w", err)
	}
	if affected == 0 {
		if _, err := r.getStreamForAccess(ctx, in.Context, in.StreamID, true); err != nil {
			return Stream{}, err
		}
	}
	return r.getStreamForAccess(ctx, in.Context, in.StreamID, true)
}

func (r *PostgresRepository) UpsertStreamAuthRule(ctx context.Context, in UpsertStreamAuthRuleInput) error {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	ruleStatus := strings.TrimSpace(in.Status)
	if ruleStatus == "" {
		ruleStatus = "active"
	}
	ruleID := newID("sar")
	if _, err := r.db.ExecContext(
		ctx,
		`INSERT INTO stream_auth_rules(
			id, tenant_id, workspace_id, stream_id, rule, status, updated_by, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8, $9)
		ON CONFLICT(tenant_id, workspace_id, stream_id) DO UPDATE SET
			rule = EXCLUDED.rule,
			status = EXCLUDED.status,
			updated_by = EXCLUDED.updated_by,
			updated_at = EXCLUDED.updated_at`,
		ruleID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
		in.StreamID,
		string(in.Rule),
		ruleStatus,
		in.Context.UserID,
		now,
		now,
	); err != nil {
		return fmt.Errorf("upsert stream auth rule: %w", err)
	}
	return nil
}

func (r *PostgresRepository) CreateRecording(ctx context.Context, in CreateRecordingInput) (Recording, error) {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	active, err := r.GetActiveRecording(ctx, in.Context, in.StreamID)
	if err == nil {
		return active, nil
	}
	if !errors.Is(err, ErrRecordingNotFound) {
		return Recording{}, err
	}

	recordingID := newID("rec")
	if _, err := r.db.ExecContext(
		ctx,
		`INSERT INTO stream_recordings(
			id, stream_id, tenant_id, workspace_id, owner_id, visibility,
			status, asset_id, error_code, message_key, started_at, finished_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		recordingID,
		in.StreamID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
		in.Context.OwnerID,
		command.VisibilityPrivate,
		RecordingStatusRecording,
		nil,
		nil,
		nil,
		now,
		nil,
		now,
		now,
	); err != nil {
		return Recording{}, fmt.Errorf("insert stream recording: %w", err)
	}

	return r.GetActiveRecording(ctx, in.Context, in.StreamID)
}

func (r *PostgresRepository) GetActiveRecording(ctx context.Context, req command.RequestContext, streamID string) (Recording, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, stream_id, tenant_id, workspace_id, owner_id, visibility, status, asset_id, error_code, message_key, started_at, finished_at, created_at, updated_at
		 FROM stream_recordings
		 WHERE stream_id = $1 AND tenant_id = $2 AND workspace_id = $3 AND status IN ($4, $5, $6)
		 ORDER BY created_at DESC, id DESC
		 LIMIT 1`,
		streamID,
		req.TenantID,
		req.WorkspaceID,
		RecordingStatusStarting,
		RecordingStatusRecording,
		RecordingStatusStopping,
	)
	item, err := scanPostgresRecording(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Recording{}, ErrRecordingNotFound
	}
	if err != nil {
		return Recording{}, fmt.Errorf("query active recording: %w", err)
	}
	return item, nil
}

func (r *PostgresRepository) CompleteRecording(ctx context.Context, in CompleteRecordingInput) (Recording, error) {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE stream_recordings
		 SET status = $1, asset_id = $2, finished_at = $3, updated_at = $4, error_code = NULL, message_key = NULL
		 WHERE id = $5 AND tenant_id = $6 AND workspace_id = $7`,
		RecordingStatusSucceeded,
		in.AssetID,
		now,
		now,
		in.RecordingID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
	)
	if err != nil {
		return Recording{}, fmt.Errorf("complete recording: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return Recording{}, fmt.Errorf("complete recording rows affected: %w", err)
	}
	if affected == 0 {
		return Recording{}, ErrRecordingNotFound
	}

	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, stream_id, tenant_id, workspace_id, owner_id, visibility, status, asset_id, error_code, message_key, started_at, finished_at, created_at, updated_at
		 FROM stream_recordings
		 WHERE id = $1 AND tenant_id = $2 AND workspace_id = $3`,
		in.RecordingID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
	)
	item, err := scanPostgresRecording(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Recording{}, ErrRecordingNotFound
	}
	if err != nil {
		return Recording{}, fmt.Errorf("query recording after complete: %w", err)
	}
	return item, nil
}

func (r *PostgresRepository) CreateLineage(ctx context.Context, in CreateLineageInput) (string, error) {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	lineageID := newID("lin")
	if _, err := r.db.ExecContext(
		ctx,
		`INSERT INTO asset_lineage(
			id, tenant_id, workspace_id, source_asset_id, target_asset_id, run_id, step_id, relation, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		lineageID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
		nil,
		in.TargetAssetID,
		nil,
		in.StepID,
		in.Relation,
		now,
	); err != nil {
		return "", fmt.Errorf("insert asset lineage: %w", err)
	}
	return lineageID, nil
}

func (r *PostgresRepository) HasPermission(
	ctx context.Context,
	req command.RequestContext,
	resourceType string,
	resourceID string,
	permission string,
	now time.Time,
) (bool, error) {
	if strings.TrimSpace(resourceID) == "" || strings.TrimSpace(permission) == "" {
		return false, nil
	}
	var marker int
	err := r.db.QueryRowContext(
		ctx,
		`SELECT 1
		 FROM acl_entries a
		 WHERE a.tenant_id = $1
		   AND a.workspace_id = $2
		   AND a.resource_type = $3
		   AND a.resource_id = $4
		   AND a.subject_type = 'user'
		   AND a.subject_id = $5
		   AND (a.expires_at IS NULL OR a.expires_at >= $6)
		   AND a.permissions @> jsonb_build_array($7::text)
		 LIMIT 1`,
		req.TenantID,
		req.WorkspaceID,
		resourceType,
		resourceID,
		req.UserID,
		now.UTC(),
		strings.ToUpper(strings.TrimSpace(permission)),
	).Scan(&marker)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query stream permission: %w", err)
	}
	return true, nil
}

type postgresRowScanner interface {
	Scan(dest ...any) error
}

func scanPostgresStreams(rows *sql.Rows) ([]Stream, error) {
	items := make([]Stream, 0)
	for rows.Next() {
		item, err := scanPostgresStream(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate streams: %w", err)
	}
	return items, nil
}

func scanPostgresStream(row postgresRowScanner) (Stream, error) {
	var (
		item         Stream
		aclRaw       string
		endpointsRaw string
		stateRaw     string
		createdAt    time.Time
		updatedAt    time.Time
	)
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.WorkspaceID,
		&item.OwnerID,
		&item.Visibility,
		&aclRaw,
		&item.Path,
		&item.Protocol,
		&item.Source,
		&endpointsRaw,
		&stateRaw,
		&item.Status,
		&createdAt,
		&updatedAt,
	); err != nil {
		return Stream{}, err
	}
	if strings.TrimSpace(aclRaw) == "" {
		aclRaw = "[]"
	}
	if strings.TrimSpace(endpointsRaw) == "" {
		endpointsRaw = "{}"
	}
	if strings.TrimSpace(stateRaw) == "" {
		stateRaw = "{}"
	}
	item.ACLJSON = json.RawMessage(aclRaw)
	item.EndpointsJSON = json.RawMessage(endpointsRaw)
	item.StateJSON = json.RawMessage(stateRaw)
	item.CreatedAt = createdAt.UTC()
	item.UpdatedAt = updatedAt.UTC()
	return item, nil
}

func scanPostgresRecording(row postgresRowScanner) (Recording, error) {
	var (
		item       Recording
		assetID    sql.NullString
		errorCode  sql.NullString
		messageKey sql.NullString
		startedAt  time.Time
		finishedAt sql.NullTime
		createdAt  time.Time
		updatedAt  time.Time
	)
	if err := row.Scan(
		&item.ID,
		&item.StreamID,
		&item.TenantID,
		&item.WorkspaceID,
		&item.OwnerID,
		&item.Visibility,
		&item.Status,
		&assetID,
		&errorCode,
		&messageKey,
		&startedAt,
		&finishedAt,
		&createdAt,
		&updatedAt,
	); err != nil {
		return Recording{}, err
	}
	if assetID.Valid {
		item.AssetID = assetID.String
	}
	if errorCode.Valid {
		item.ErrorCode = errorCode.String
	}
	if messageKey.Valid {
		item.MessageKey = messageKey.String
	}
	item.StartedAt = startedAt.UTC()
	item.CreatedAt = createdAt.UTC()
	item.UpdatedAt = updatedAt.UTC()
	if finishedAt.Valid {
		f := finishedAt.Time.UTC()
		item.FinishedAt = &f
	}
	return item, nil
}
