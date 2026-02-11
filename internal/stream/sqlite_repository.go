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

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) CreateStream(ctx context.Context, in CreateStreamInput) (Stream, error) {
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
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
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
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
	); err != nil {
		return Stream{}, fmt.Errorf("insert stream: %w", err)
	}
	return r.GetStreamForAccess(ctx, in.Context, streamID)
}

func (r *SQLiteRepository) GetStreamForAccess(ctx context.Context, req command.RequestContext, streamID string) (Stream, error) {
	return r.getStreamForAccess(ctx, req, streamID, false)
}

func (r *SQLiteRepository) getStreamForAccess(
	ctx context.Context,
	req command.RequestContext,
	streamID string,
	includeDeleted bool,
) (Stream, error) {
	deletedFilter := "AND COALESCE(json_extract(state_json, '$.deleted'), 0) != 1"
	if includeDeleted {
		deletedFilter = ""
	}
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, owner_id, visibility, acl_json, path, protocol, source, endpoints_json, state_json, status, created_at, updated_at
		 FROM streaming_assets
		 WHERE id = ? AND tenant_id = ? AND workspace_id = ? `+deletedFilter,
		streamID,
		req.TenantID,
		req.WorkspaceID,
	)
	item, err := scanStream(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Stream{}, ErrStreamNotFound
	}
	if err != nil {
		return Stream{}, fmt.Errorf("query stream: %w", err)
	}
	return item, nil
}

func (r *SQLiteRepository) ListStreams(ctx context.Context, params StreamListParams) (StreamListResult, error) {
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

	baseFilter := `FROM streaming_assets s
		WHERE s.tenant_id = ? AND s.workspace_id = ?
		  AND COALESCE(json_extract(s.state_json, '$.deleted'), 0) != 1
		  AND (
		    s.owner_id = ?
		    OR s.visibility = 'WORKSPACE'
		    OR EXISTS (
		      SELECT 1 FROM acl_entries a
		      WHERE a.tenant_id = s.tenant_id
		        AND a.workspace_id = s.workspace_id
		        AND a.resource_type = 'streaming_asset'
		        AND a.resource_id = s.id
		        AND a.subject_type = 'user'
		        AND a.subject_id = ?
		        AND (a.expires_at IS NULL OR a.expires_at >= ?)
		        AND EXISTS (
		          SELECT 1 FROM json_each(a.permissions) p
		          WHERE UPPER(COALESCE(p.value, '')) = 'READ'
		        )
		    )
		  )`

	if strings.TrimSpace(params.Cursor) != "" {
		cursorAt, cursorID, err := command.DecodeCursor(params.Cursor)
		if err != nil {
			return StreamListResult{}, ErrInvalidCursor
		}
		rows, err := r.db.QueryContext(
			ctx,
			`SELECT s.id, s.tenant_id, s.workspace_id, s.owner_id, s.visibility, s.acl_json, s.path, s.protocol, s.source, s.endpoints_json, s.state_json, s.status, s.created_at, s.updated_at
			 `+baseFilter+`
			   AND ((s.created_at < ?) OR (s.created_at = ? AND s.id < ?))
			 ORDER BY s.created_at DESC, s.id DESC
			 LIMIT ?`,
			params.Context.TenantID,
			params.Context.WorkspaceID,
			params.Context.OwnerID,
			params.Context.UserID,
			now,
			cursorAt.UTC().Format(time.RFC3339Nano),
			cursorAt.UTC().Format(time.RFC3339Nano),
			cursorID,
			pageSize,
		)
		if err != nil {
			return StreamListResult{}, fmt.Errorf("list streams by cursor: %w", err)
		}
		defer rows.Close()

		items, err := scanStreams(rows)
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
		`SELECT s.id, s.tenant_id, s.workspace_id, s.owner_id, s.visibility, s.acl_json, s.path, s.protocol, s.source, s.endpoints_json, s.state_json, s.status, s.created_at, s.updated_at
		 `+baseFilter+`
		 ORDER BY s.created_at DESC, s.id DESC
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
		return StreamListResult{}, fmt.Errorf("list streams by page: %w", err)
	}
	defer rows.Close()

	items, err := scanStreams(rows)
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

func (r *SQLiteRepository) UpdateStreamStatus(ctx context.Context, in UpdateStreamStatusInput) (Stream, error) {
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
		 SET status = ?, state_json = ?, updated_at = ?
		 WHERE id = ? AND tenant_id = ? AND workspace_id = ?`,
		in.Status,
		string(in.State),
		now.Format(time.RFC3339Nano),
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

func (r *SQLiteRepository) UpsertStreamAuthRule(ctx context.Context, in UpsertStreamAuthRuleInput) error {
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
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(tenant_id, workspace_id, stream_id) DO UPDATE SET
			rule = excluded.rule,
			status = excluded.status,
			updated_by = excluded.updated_by,
			updated_at = excluded.updated_at`,
		ruleID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
		in.StreamID,
		string(in.Rule),
		ruleStatus,
		in.Context.UserID,
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
	); err != nil {
		return fmt.Errorf("upsert stream auth rule: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) CreateRecording(ctx context.Context, in CreateRecordingInput) (Recording, error) {
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
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
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
		now.Format(time.RFC3339Nano),
		nil,
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
	); err != nil {
		return Recording{}, fmt.Errorf("insert stream recording: %w", err)
	}

	return r.GetActiveRecording(ctx, in.Context, in.StreamID)
}

func (r *SQLiteRepository) GetActiveRecording(ctx context.Context, req command.RequestContext, streamID string) (Recording, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, stream_id, tenant_id, workspace_id, owner_id, visibility, status, asset_id, error_code, message_key, started_at, finished_at, created_at, updated_at
		 FROM stream_recordings
		 WHERE stream_id = ? AND tenant_id = ? AND workspace_id = ? AND status IN (?, ?, ?)
		 ORDER BY created_at DESC, id DESC
		 LIMIT 1`,
		streamID,
		req.TenantID,
		req.WorkspaceID,
		RecordingStatusStarting,
		RecordingStatusRecording,
		RecordingStatusStopping,
	)
	item, err := scanRecording(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Recording{}, ErrRecordingNotFound
	}
	if err != nil {
		return Recording{}, fmt.Errorf("query active recording: %w", err)
	}
	return item, nil
}

func (r *SQLiteRepository) CompleteRecording(ctx context.Context, in CompleteRecordingInput) (Recording, error) {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE stream_recordings
		 SET status = ?, asset_id = ?, finished_at = ?, updated_at = ?, error_code = NULL, message_key = NULL
		 WHERE id = ? AND tenant_id = ? AND workspace_id = ?`,
		RecordingStatusSucceeded,
		in.AssetID,
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
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
		 WHERE id = ? AND tenant_id = ? AND workspace_id = ?`,
		in.RecordingID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
	)
	item, err := scanRecording(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Recording{}, ErrRecordingNotFound
	}
	if err != nil {
		return Recording{}, fmt.Errorf("query recording after complete: %w", err)
	}
	return item, nil
}

func (r *SQLiteRepository) CreateLineage(ctx context.Context, in CreateLineageInput) (string, error) {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	lineageID := newID("lin")
	if _, err := r.db.ExecContext(
		ctx,
		`INSERT INTO asset_lineage(
			id, tenant_id, workspace_id, source_asset_id, target_asset_id, run_id, step_id, relation, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		lineageID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
		nil,
		in.TargetAssetID,
		nil,
		in.StepID,
		in.Relation,
		now.Format(time.RFC3339Nano),
	); err != nil {
		return "", fmt.Errorf("insert asset lineage: %w", err)
	}
	return lineageID, nil
}

func (r *SQLiteRepository) HasPermission(
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
	row := r.db.QueryRowContext(
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
		   AND EXISTS (
		     SELECT 1 FROM json_each(a.permissions) p
		     WHERE UPPER(COALESCE(p.value, '')) = ?
		   )
		 LIMIT 1`,
		req.TenantID,
		req.WorkspaceID,
		resourceType,
		resourceID,
		req.UserID,
		now.UTC().Format(time.RFC3339Nano),
		strings.ToUpper(strings.TrimSpace(permission)),
	)
	var marker int
	err := row.Scan(&marker)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query stream permission: %w", err)
	}
	return true, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanStreams(rows *sql.Rows) ([]Stream, error) {
	items := make([]Stream, 0)
	for rows.Next() {
		item, err := scanStream(rows)
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

func scanStream(row rowScanner) (Stream, error) {
	var (
		item         Stream
		aclRaw       string
		endpointsRaw string
		stateRaw     string
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
		&item.Path,
		&item.Protocol,
		&item.Source,
		&endpointsRaw,
		&stateRaw,
		&item.Status,
		&createdAtRaw,
		&updatedAtRaw,
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

	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return Stream{}, fmt.Errorf("parse stream created_at: %w", err)
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, updatedAtRaw)
	if err != nil {
		return Stream{}, fmt.Errorf("parse stream updated_at: %w", err)
	}
	item.CreatedAt = createdAt
	item.UpdatedAt = updatedAt
	return item, nil
}

func scanRecording(row rowScanner) (Recording, error) {
	var (
		item          Recording
		assetIDRaw    sql.NullString
		errorCodeRaw  sql.NullString
		messageKeyRaw sql.NullString
		startedAtRaw  string
		finishedAtRaw sql.NullString
		createdAtRaw  string
		updatedAtRaw  string
	)
	if err := row.Scan(
		&item.ID,
		&item.StreamID,
		&item.TenantID,
		&item.WorkspaceID,
		&item.OwnerID,
		&item.Visibility,
		&item.Status,
		&assetIDRaw,
		&errorCodeRaw,
		&messageKeyRaw,
		&startedAtRaw,
		&finishedAtRaw,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		return Recording{}, err
	}
	if assetIDRaw.Valid {
		item.AssetID = assetIDRaw.String
	}
	if errorCodeRaw.Valid {
		item.ErrorCode = errorCodeRaw.String
	}
	if messageKeyRaw.Valid {
		item.MessageKey = messageKeyRaw.String
	}

	startedAt, err := time.Parse(time.RFC3339Nano, startedAtRaw)
	if err != nil {
		return Recording{}, fmt.Errorf("parse recording started_at: %w", err)
	}
	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return Recording{}, fmt.Errorf("parse recording created_at: %w", err)
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, updatedAtRaw)
	if err != nil {
		return Recording{}, fmt.Errorf("parse recording updated_at: %w", err)
	}
	item.StartedAt = startedAt
	item.CreatedAt = createdAt
	item.UpdatedAt = updatedAt
	if finishedAtRaw.Valid && strings.TrimSpace(finishedAtRaw.String) != "" {
		finishedAt, err := time.Parse(time.RFC3339Nano, finishedAtRaw.String)
		if err != nil {
			return Recording{}, fmt.Errorf("parse recording finished_at: %w", err)
		}
		item.FinishedAt = &finishedAt
	}
	return item, nil
}
