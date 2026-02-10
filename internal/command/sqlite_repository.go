package command

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) Create(ctx context.Context, in CreateInput) (CreateResult, error) {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	if in.IdempotencyKey == "" {
		cmd, err := r.insertCommand(ctx, in.Context, in.CommandType, in.Payload, in.Visibility, now)
		if err != nil {
			return CreateResult{}, err
		}
		return CreateResult{Command: cmd, Reused: false}, nil
	}

	conn, err := r.db.Conn(ctx)
	if err != nil {
		return CreateResult{}, fmt.Errorf("open sqlite conn: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, "BEGIN IMMEDIATE"); err != nil {
		return CreateResult{}, fmt.Errorf("begin immediate tx: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()

	active, existingHash, existingCommandID, err := r.lookupActiveIdempotency(ctx, conn, in, now)
	if err != nil {
		return CreateResult{}, err
	}

	if active {
		if existingHash == in.RequestHash {
			cmd, err := r.getByIDFromConn(ctx, conn, in.Context, existingCommandID)
			if err != nil {
				return CreateResult{}, err
			}
			if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
				return CreateResult{}, fmt.Errorf("commit tx: %w", err)
			}
			committed = true
			return CreateResult{Command: cmd, Reused: true}, nil
		}
		return CreateResult{}, &IdempotencyConflictError{ExistingCommandID: existingCommandID}
	}

	cmd, err := r.insertCommandFromConn(ctx, conn, in.Context, in.CommandType, in.Payload, in.Visibility, now)
	if err != nil {
		return CreateResult{}, err
	}

	expiresAt := now.Add(in.TTL)
	if err := r.upsertIdempotencyFromConn(ctx, conn, in, cmd.ID, expiresAt, now); err != nil {
		return CreateResult{}, err
	}

	if _, err := conn.ExecContext(ctx, "COMMIT"); err != nil {
		return CreateResult{}, fmt.Errorf("commit tx: %w", err)
	}
	committed = true

	return CreateResult{Command: cmd, Reused: false}, nil
}

func (r *SQLiteRepository) Get(ctx context.Context, req RequestContext, id string) (Command, error) {
	return r.getByID(ctx, req, id)
}

func (r *SQLiteRepository) GetForAccess(ctx context.Context, req RequestContext, id string) (Command, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, owner_id, visibility, acl_json, command_type, payload, status, result, error_code, message_key, accepted_at, finished_at, created_at, updated_at
		 FROM commands
		 WHERE id = ? AND tenant_id = ? AND workspace_id = ?`,
		id,
		req.TenantID,
		req.WorkspaceID,
	)

	cmd, err := scanCommand(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Command{}, ErrNotFound
	}
	if err != nil {
		return Command{}, fmt.Errorf("query command for access: %w", err)
	}
	return cmd, nil
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
	visibility := VisibilityWorkspace

	if params.Cursor != "" {
		createdAt, id, err := DecodeCursor(params.Cursor)
		if err != nil {
			return ListResult{}, ErrInvalidCursor
		}

		rows, err := r.db.QueryContext(
			ctx,
			`SELECT c.id, c.tenant_id, c.workspace_id, c.owner_id, c.visibility, c.acl_json, c.command_type, c.payload, c.status, c.result, c.error_code, c.message_key, c.accepted_at, c.finished_at, c.created_at, c.updated_at
			 FROM commands c
			 WHERE c.tenant_id = ? AND c.workspace_id = ?
			   AND (
			     c.owner_id = ?
			     OR c.visibility = ?
			     OR EXISTS (
			       SELECT 1
			       FROM acl_entries a
			       WHERE a.tenant_id = c.tenant_id
			         AND a.workspace_id = c.workspace_id
			         AND a.resource_type = 'command'
			         AND a.resource_id = c.id
			         AND a.subject_type = 'user'
			         AND a.subject_id = ?
			         AND (a.expires_at IS NULL OR a.expires_at >= ?)
			         AND EXISTS (
			           SELECT 1 FROM json_each(a.permissions) p WHERE p.value = 'READ'
			         )
			     )
			   )
			   AND ((c.created_at < ?) OR (c.created_at = ? AND c.id < ?))
			 ORDER BY c.created_at DESC, c.id DESC
			 LIMIT ?`,
			params.Context.TenantID,
			params.Context.WorkspaceID,
			params.Context.OwnerID,
			visibility,
			params.Context.UserID,
			now,
			createdAt.Format(time.RFC3339Nano),
			createdAt.Format(time.RFC3339Nano),
			id,
			pageSize,
		)
		if err != nil {
			return ListResult{}, fmt.Errorf("list commands by cursor: %w", err)
		}
		defer rows.Close()

		items, err := scanCommands(rows)
		if err != nil {
			return ListResult{}, err
		}

		nextCursor := ""
		if len(items) == pageSize {
			last := items[len(items)-1]
			nextCursor, err = EncodeCursor(last.CreatedAt, last.ID)
			if err != nil {
				return ListResult{}, fmt.Errorf("encode next cursor: %w", err)
			}
		}

		return ListResult{
			Items:      items,
			NextCursor: nextCursor,
			UsedCursor: true,
		}, nil
	}

	offset := (page - 1) * pageSize
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT c.id, c.tenant_id, c.workspace_id, c.owner_id, c.visibility, c.acl_json, c.command_type, c.payload, c.status, c.result, c.error_code, c.message_key, c.accepted_at, c.finished_at, c.created_at, c.updated_at
		 FROM commands c
		 WHERE c.tenant_id = ? AND c.workspace_id = ?
		   AND (
		     c.owner_id = ?
		     OR c.visibility = ?
		     OR EXISTS (
		       SELECT 1
		       FROM acl_entries a
		       WHERE a.tenant_id = c.tenant_id
		         AND a.workspace_id = c.workspace_id
		         AND a.resource_type = 'command'
		         AND a.resource_id = c.id
		         AND a.subject_type = 'user'
		         AND a.subject_id = ?
		         AND (a.expires_at IS NULL OR a.expires_at >= ?)
		         AND EXISTS (
		           SELECT 1 FROM json_each(a.permissions) p WHERE p.value = 'READ'
		         )
		     )
		   )
		 ORDER BY c.created_at DESC, c.id DESC
		 LIMIT ? OFFSET ?`,
		params.Context.TenantID,
		params.Context.WorkspaceID,
		params.Context.OwnerID,
		visibility,
		params.Context.UserID,
		now,
		pageSize,
		offset,
	)
	if err != nil {
		return ListResult{}, fmt.Errorf("list commands by page: %w", err)
	}
	defer rows.Close()

	items, err := scanCommands(rows)
	if err != nil {
		return ListResult{}, err
	}

	var total int64
	if err := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(1)
		 FROM commands c
		 WHERE c.tenant_id = ? AND c.workspace_id = ?
		   AND (
		     c.owner_id = ?
		     OR c.visibility = ?
		     OR EXISTS (
		       SELECT 1
		       FROM acl_entries a
		       WHERE a.tenant_id = c.tenant_id
		         AND a.workspace_id = c.workspace_id
		         AND a.resource_type = 'command'
		         AND a.resource_id = c.id
		         AND a.subject_type = 'user'
		         AND a.subject_id = ?
		         AND (a.expires_at IS NULL OR a.expires_at >= ?)
		         AND EXISTS (
		           SELECT 1 FROM json_each(a.permissions) p WHERE p.value = 'READ'
		         )
		     )
		   )`,
		params.Context.TenantID,
		params.Context.WorkspaceID,
		params.Context.OwnerID,
		visibility,
		params.Context.UserID,
		now,
	).Scan(&total); err != nil {
		return ListResult{}, fmt.Errorf("count commands: %w", err)
	}

	return ListResult{
		Items: items,
		Total: total,
	}, nil
}

func (r *SQLiteRepository) HasCommandPermission(ctx context.Context, req RequestContext, commandID, permission string, now time.Time) (bool, error) {
	if strings.TrimSpace(commandID) == "" || strings.TrimSpace(permission) == "" {
		return false, nil
	}

	var marker int
	err := r.db.QueryRowContext(
		ctx,
		`SELECT 1
		 FROM acl_entries a
		 WHERE a.tenant_id = ?
		   AND a.workspace_id = ?
		   AND a.resource_type = 'command'
		   AND a.resource_id = ?
		   AND a.subject_type = 'user'
		   AND a.subject_id = ?
		   AND (a.expires_at IS NULL OR a.expires_at >= ?)
		   AND EXISTS (
		     SELECT 1 FROM json_each(a.permissions) p WHERE p.value = ?
		   )
		 LIMIT 1`,
		req.TenantID,
		req.WorkspaceID,
		commandID,
		req.UserID,
		now.UTC().Format(time.RFC3339Nano),
		strings.ToUpper(permission),
	).Scan(&marker)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query acl permission: %w", err)
	}
	return true, nil
}

func (r *SQLiteRepository) GetShareResource(ctx context.Context, req RequestContext, resourceType, resourceID string) (ShareResource, error) {
	resourceType = strings.ToLower(strings.TrimSpace(resourceType))
	resourceID = strings.TrimSpace(resourceID)
	if resourceID == "" {
		return ShareResource{}, ErrInvalidShareRequest
	}

	query := ""
	switch resourceType {
	case "command":
		query = `SELECT tenant_id, workspace_id, owner_id, visibility
		         FROM commands
		         WHERE id = ? AND tenant_id = ? AND workspace_id = ?`
	case "asset":
		query = `SELECT tenant_id, workspace_id, owner_id, visibility
		         FROM assets
		         WHERE id = ? AND tenant_id = ? AND workspace_id = ?`
	default:
		return ShareResource{}, ErrInvalidShareRequest
	}

	resource := ShareResource{
		ResourceType: resourceType,
		ResourceID:   resourceID,
	}
	err := r.db.QueryRowContext(
		ctx,
		query,
		resourceID,
		req.TenantID,
		req.WorkspaceID,
	).Scan(
		&resource.TenantID,
		&resource.WorkspaceID,
		&resource.OwnerID,
		&resource.Visibility,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return ShareResource{}, ErrNotFound
	}
	if err != nil {
		return ShareResource{}, fmt.Errorf("query share resource %s: %w", resourceType, err)
	}
	return resource, nil
}

func (r *SQLiteRepository) HasShareResourcePermission(
	ctx context.Context,
	req RequestContext,
	resourceType,
	resourceID,
	permission string,
	now time.Time,
) (bool, error) {
	resourceType = strings.ToLower(strings.TrimSpace(resourceType))
	resourceID = strings.TrimSpace(resourceID)
	permission = strings.ToUpper(strings.TrimSpace(permission))
	if resourceID == "" || permission == "" {
		return false, nil
	}
	switch resourceType {
	case "command", "asset":
	default:
		return false, ErrInvalidShareRequest
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
		   AND EXISTS (
		     SELECT 1 FROM json_each(a.permissions) p WHERE p.value = ?
		   )
		 LIMIT 1`,
		req.TenantID,
		req.WorkspaceID,
		resourceType,
		resourceID,
		req.UserID,
		now.UTC().Format(time.RFC3339Nano),
		permission,
	).Scan(&marker)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query share resource permission: %w", err)
	}
	return true, nil
}

func (r *SQLiteRepository) CreateShare(ctx context.Context, in ShareCreateInput) (Share, error) {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	permissionsRaw, err := json.Marshal(in.Permissions)
	if err != nil {
		return Share{}, fmt.Errorf("marshal permissions: %w", err)
	}

	shareID := newID("shr")
	var expiresAtValue interface{}
	if in.ExpiresAt != nil {
		expiresAtValue = in.ExpiresAt.UTC().Format(time.RFC3339Nano)
	}

	_, err = r.db.ExecContext(
		ctx,
		`INSERT INTO acl_entries(id, tenant_id, workspace_id, resource_type, resource_id, subject_type, subject_id, permissions, expires_at, created_by, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		shareID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
		in.ResourceType,
		in.ResourceID,
		in.SubjectType,
		in.SubjectID,
		string(permissionsRaw),
		expiresAtValue,
		in.Context.UserID,
		now.Format(time.RFC3339Nano),
	)
	if err != nil {
		return Share{}, fmt.Errorf("insert share: %w", err)
	}

	return r.getShareByID(ctx, in.Context, shareID)
}

func (r *SQLiteRepository) ListShares(ctx context.Context, params ShareListParams) (ShareListResult, error) {
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

	if params.Cursor != "" {
		createdAt, id, err := DecodeCursor(params.Cursor)
		if err != nil {
			return ShareListResult{}, ErrInvalidCursor
		}

		rows, err := r.db.QueryContext(
			ctx,
			`SELECT id, tenant_id, workspace_id, resource_type, resource_id, subject_type, subject_id, permissions, expires_at, created_by, created_at
			 FROM acl_entries
			 WHERE tenant_id = ? AND workspace_id = ?
			   AND ((created_at < ?) OR (created_at = ? AND id < ?))
			 ORDER BY created_at DESC, id DESC
			 LIMIT ?`,
			params.Context.TenantID,
			params.Context.WorkspaceID,
			createdAt.Format(time.RFC3339Nano),
			createdAt.Format(time.RFC3339Nano),
			id,
			pageSize,
		)
		if err != nil {
			return ShareListResult{}, fmt.Errorf("list shares by cursor: %w", err)
		}
		defer rows.Close()

		items, err := scanShares(rows)
		if err != nil {
			return ShareListResult{}, err
		}

		nextCursor := ""
		if len(items) == pageSize {
			last := items[len(items)-1]
			nextCursor, err = EncodeCursor(last.CreatedAt, last.ID)
			if err != nil {
				return ShareListResult{}, fmt.Errorf("encode next share cursor: %w", err)
			}
		}

		return ShareListResult{
			Items:      items,
			NextCursor: nextCursor,
			UsedCursor: true,
		}, nil
	}

	offset := (page - 1) * pageSize
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, resource_type, resource_id, subject_type, subject_id, permissions, expires_at, created_by, created_at
		 FROM acl_entries
		 WHERE tenant_id = ? AND workspace_id = ?
		 ORDER BY created_at DESC, id DESC
		 LIMIT ? OFFSET ?`,
		params.Context.TenantID,
		params.Context.WorkspaceID,
		pageSize,
		offset,
	)
	if err != nil {
		return ShareListResult{}, fmt.Errorf("list shares by page: %w", err)
	}
	defer rows.Close()

	items, err := scanShares(rows)
	if err != nil {
		return ShareListResult{}, err
	}

	var total int64
	if err := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(1) FROM acl_entries WHERE tenant_id = ? AND workspace_id = ?`,
		params.Context.TenantID,
		params.Context.WorkspaceID,
	).Scan(&total); err != nil {
		return ShareListResult{}, fmt.Errorf("count shares: %w", err)
	}

	return ShareListResult{
		Items: items,
		Total: total,
	}, nil
}

func (r *SQLiteRepository) DeleteShare(ctx context.Context, req RequestContext, shareID string) error {
	result, err := r.db.ExecContext(
		ctx,
		`DELETE FROM acl_entries
		 WHERE id = ? AND tenant_id = ? AND workspace_id = ? AND created_by = ?`,
		shareID,
		req.TenantID,
		req.WorkspaceID,
		req.UserID,
	)
	if err != nil {
		return fmt.Errorf("delete share: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete share rows affected: %w", err)
	}
	if affected == 0 {
		return ErrShareNotFound
	}
	return nil
}

func (r *SQLiteRepository) AppendCommandEvent(ctx context.Context, req RequestContext, commandID, eventType string, payload []byte) error {
	if len(payload) == 0 {
		payload = []byte("{}")
	}
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO command_events(id, command_id, event_type, payload, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		newID("cevt"),
		commandID,
		eventType,
		string(payload),
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert command event: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) AppendAuditEvent(ctx context.Context, req RequestContext, commandID, eventType, decision, reason string, payload []byte) error {
	payload = buildAuditPayload(req, payload)
	traceID := strings.TrimSpace(req.TraceID)
	if traceID == "" {
		traceID = newID("trace")
	}
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO audit_events(id, tenant_id, workspace_id, user_id, trace_id, command_id, event_type, resource_type, resource_id, decision, reason, payload, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		newID("aevt"),
		req.TenantID,
		req.WorkspaceID,
		req.UserID,
		traceID,
		commandID,
		eventType,
		"command",
		commandID,
		decision,
		reason,
		string(payload),
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}
	return nil
}

func buildAuditPayload(req RequestContext, payload []byte) []byte {
	var base any
	if len(payload) > 0 {
		_ = json.Unmarshal(payload, &base)
	}
	if base == nil {
		base = map[string]any{}
	}

	ctxPayload := map[string]any{
		"roles":         req.Roles,
		"policyVersion": req.PolicyVersion,
	}
	if strings.TrimSpace(req.TraceID) != "" {
		ctxPayload["traceId"] = strings.TrimSpace(req.TraceID)
	}

	out := map[string]any{
		"context": ctxPayload,
		"data":    base,
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return []byte(`{"context":{},"data":{}}`)
	}
	return raw
}

func (r *SQLiteRepository) SetStatus(ctx context.Context, req RequestContext, commandID, status string, result []byte, errorCode, messageKey string, finishedAt *time.Time) (Command, error) {
	updatedAt := time.Now().UTC().Format(time.RFC3339Nano)

	var finishedAtVal interface{}
	if finishedAt != nil {
		finishedAtVal = finishedAt.UTC().Format(time.RFC3339Nano)
	}

	var resultVal interface{}
	if len(result) > 0 {
		resultVal = string(result)
	}

	var errorCodeVal interface{}
	if errorCode != "" {
		errorCodeVal = errorCode
	}

	var messageKeyVal interface{}
	if messageKey != "" {
		messageKeyVal = messageKey
	}

	_, err := r.db.ExecContext(
		ctx,
		`UPDATE commands
		 SET status = ?,
		     result = COALESCE(?, result),
		     error_code = ?,
		     message_key = ?,
		     finished_at = ?,
		     updated_at = ?
		 WHERE id = ? AND tenant_id = ? AND workspace_id = ? AND owner_id = ?`,
		status,
		resultVal,
		errorCodeVal,
		messageKeyVal,
		finishedAtVal,
		updatedAt,
		commandID,
		req.TenantID,
		req.WorkspaceID,
		req.OwnerID,
	)
	if err != nil {
		return Command{}, fmt.Errorf("update command status: %w", err)
	}

	return r.Get(ctx, req, commandID)
}

func (r *SQLiteRepository) lookupActiveIdempotency(ctx context.Context, conn *sql.Conn, in CreateInput, now time.Time) (bool, string, string, error) {
	var requestHash, commandID string
	err := conn.QueryRowContext(
		ctx,
		`SELECT request_hash, command_id
		 FROM command_idempotency
		 WHERE tenant_id = ? AND workspace_id = ? AND owner_id = ? AND idempotency_key = ? AND expires_at >= ?`,
		in.Context.TenantID,
		in.Context.WorkspaceID,
		in.Context.OwnerID,
		in.IdempotencyKey,
		now.Format(time.RFC3339Nano),
	).Scan(&requestHash, &commandID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, "", "", nil
	}
	if err != nil {
		return false, "", "", fmt.Errorf("query idempotency mapping: %w", err)
	}
	return true, requestHash, commandID, nil
}

func (r *SQLiteRepository) upsertIdempotencyFromConn(ctx context.Context, conn *sql.Conn, in CreateInput, commandID string, expiresAt, now time.Time) error {
	_, err := conn.ExecContext(
		ctx,
		`INSERT INTO command_idempotency(tenant_id, workspace_id, owner_id, idempotency_key, request_hash, command_id, expires_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(tenant_id, workspace_id, owner_id, idempotency_key)
		 DO UPDATE SET request_hash = excluded.request_hash,
		               command_id = excluded.command_id,
		               expires_at = excluded.expires_at,
		               created_at = excluded.created_at`,
		in.Context.TenantID,
		in.Context.WorkspaceID,
		in.Context.OwnerID,
		in.IdempotencyKey,
		in.RequestHash,
		commandID,
		expiresAt.UTC().Format(time.RFC3339Nano),
		now.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("upsert idempotency mapping: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) insertCommand(ctx context.Context, req RequestContext, commandType string, payload []byte, visibility string, now time.Time) (Command, error) {
	cmdID := newID("cmd")
	if visibility == "" {
		visibility = VisibilityPrivate
	}
	if _, err := r.db.ExecContext(
		ctx,
		`INSERT INTO commands(id, tenant_id, workspace_id, owner_id, visibility, acl_json, command_type, payload, status, accepted_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		cmdID,
		req.TenantID,
		req.WorkspaceID,
		req.OwnerID,
		visibility,
		"[]",
		commandType,
		string(payload),
		StatusAccepted,
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
	); err != nil {
		return Command{}, fmt.Errorf("insert command: %w", err)
	}

	return r.getByID(ctx, req, cmdID)
}

func (r *SQLiteRepository) insertCommandFromConn(ctx context.Context, conn *sql.Conn, req RequestContext, commandType string, payload []byte, visibility string, now time.Time) (Command, error) {
	cmdID := newID("cmd")
	if visibility == "" {
		visibility = VisibilityPrivate
	}
	if _, err := conn.ExecContext(
		ctx,
		`INSERT INTO commands(id, tenant_id, workspace_id, owner_id, visibility, acl_json, command_type, payload, status, accepted_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		cmdID,
		req.TenantID,
		req.WorkspaceID,
		req.OwnerID,
		visibility,
		"[]",
		commandType,
		string(payload),
		StatusAccepted,
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
		now.Format(time.RFC3339Nano),
	); err != nil {
		return Command{}, fmt.Errorf("insert command in tx: %w", err)
	}

	return r.getByIDFromConn(ctx, conn, req, cmdID)
}

func (r *SQLiteRepository) getByID(ctx context.Context, req RequestContext, id string) (Command, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, owner_id, visibility, acl_json, command_type, payload, status, result, error_code, message_key, accepted_at, finished_at, created_at, updated_at
		 FROM commands
		 WHERE id = ? AND tenant_id = ? AND workspace_id = ? AND owner_id = ?`,
		id,
		req.TenantID,
		req.WorkspaceID,
		req.OwnerID,
	)

	cmd, err := scanCommand(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Command{}, ErrNotFound
	}
	if err != nil {
		return Command{}, fmt.Errorf("query command: %w", err)
	}
	return cmd, nil
}

func (r *SQLiteRepository) getByIDFromConn(ctx context.Context, conn *sql.Conn, req RequestContext, id string) (Command, error) {
	row := conn.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, owner_id, visibility, acl_json, command_type, payload, status, result, error_code, message_key, accepted_at, finished_at, created_at, updated_at
		 FROM commands
		 WHERE id = ? AND tenant_id = ? AND workspace_id = ? AND owner_id = ?`,
		id,
		req.TenantID,
		req.WorkspaceID,
		req.OwnerID,
	)

	cmd, err := scanCommand(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Command{}, ErrNotFound
	}
	if err != nil {
		return Command{}, fmt.Errorf("query command from tx: %w", err)
	}
	return cmd, nil
}

func (r *SQLiteRepository) getShareByID(ctx context.Context, req RequestContext, id string) (Share, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, resource_type, resource_id, subject_type, subject_id, permissions, expires_at, created_by, created_at
		 FROM acl_entries
		 WHERE id = ? AND tenant_id = ? AND workspace_id = ?`,
		id,
		req.TenantID,
		req.WorkspaceID,
	)

	share, err := scanShare(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Share{}, ErrShareNotFound
	}
	if err != nil {
		return Share{}, fmt.Errorf("query share: %w", err)
	}
	return share, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanCommands(rows *sql.Rows) ([]Command, error) {
	items := make([]Command, 0)
	for rows.Next() {
		cmd, err := scanCommand(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, cmd)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate commands: %w", err)
	}
	return items, nil
}

func scanCommand(row rowScanner) (Command, error) {
	var (
		cmd           Command
		aclRaw        string
		payloadRaw    string
		resultRaw     sql.NullString
		errorCodeRaw  sql.NullString
		messageKeyRaw sql.NullString
		acceptedAtRaw string
		finishedAtRaw sql.NullString
		createdAtRaw  string
		updatedAtRaw  string
	)

	if err := row.Scan(
		&cmd.ID,
		&cmd.TenantID,
		&cmd.WorkspaceID,
		&cmd.OwnerID,
		&cmd.Visibility,
		&aclRaw,
		&cmd.CommandType,
		&payloadRaw,
		&cmd.Status,
		&resultRaw,
		&errorCodeRaw,
		&messageKeyRaw,
		&acceptedAtRaw,
		&finishedAtRaw,
		&createdAtRaw,
		&updatedAtRaw,
	); err != nil {
		return Command{}, err
	}

	cmd.ACLJSON = json.RawMessage(aclRaw)
	cmd.Payload = json.RawMessage(payloadRaw)
	if resultRaw.Valid {
		cmd.Result = json.RawMessage(resultRaw.String)
	}
	if errorCodeRaw.Valid {
		cmd.ErrorCode = errorCodeRaw.String
	}
	if messageKeyRaw.Valid {
		cmd.MessageKey = messageKeyRaw.String
	}

	acceptedAt, err := time.Parse(time.RFC3339Nano, acceptedAtRaw)
	if err != nil {
		return Command{}, fmt.Errorf("parse accepted_at: %w", err)
	}
	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return Command{}, fmt.Errorf("parse created_at: %w", err)
	}
	updatedAt, err := time.Parse(time.RFC3339Nano, updatedAtRaw)
	if err != nil {
		return Command{}, fmt.Errorf("parse updated_at: %w", err)
	}
	cmd.AcceptedAt = acceptedAt
	cmd.CreatedAt = createdAt
	cmd.UpdatedAt = updatedAt

	if finishedAtRaw.Valid {
		v, err := time.Parse(time.RFC3339Nano, finishedAtRaw.String)
		if err != nil {
			return Command{}, fmt.Errorf("parse finished_at: %w", err)
		}
		cmd.FinishedAt = &v
	}

	return cmd, nil
}

func scanShares(rows *sql.Rows) ([]Share, error) {
	items := make([]Share, 0)
	for rows.Next() {
		item, err := scanShare(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate shares: %w", err)
	}
	return items, nil
}

func scanShare(row rowScanner) (Share, error) {
	var (
		item         Share
		permissions  string
		expiresAtRaw sql.NullString
		createdAtRaw string
	)

	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.WorkspaceID,
		&item.ResourceType,
		&item.ResourceID,
		&item.SubjectType,
		&item.SubjectID,
		&permissions,
		&expiresAtRaw,
		&item.CreatedBy,
		&createdAtRaw,
	); err != nil {
		return Share{}, err
	}

	if err := json.Unmarshal([]byte(permissions), &item.Permissions); err != nil {
		return Share{}, fmt.Errorf("unmarshal permissions: %w", err)
	}

	if expiresAtRaw.Valid && strings.TrimSpace(expiresAtRaw.String) != "" {
		expiresAt, err := time.Parse(time.RFC3339Nano, expiresAtRaw.String)
		if err != nil {
			return Share{}, fmt.Errorf("parse expires_at: %w", err)
		}
		item.ExpiresAt = &expiresAt
	}

	createdAt, err := time.Parse(time.RFC3339Nano, createdAtRaw)
	if err != nil {
		return Share{}, fmt.Errorf("parse share created_at: %w", err)
	}
	item.CreatedAt = createdAt

	return item, nil
}
