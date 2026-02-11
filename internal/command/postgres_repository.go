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

type PostgresRepository struct {
	db *sql.DB
}

func NewPostgresRepository(db *sql.DB) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(ctx context.Context, in CreateInput) (CreateResult, error) {
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

	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return CreateResult{}, fmt.Errorf("begin postgres tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	active, existingHash, existingCommandID, err := r.lookupActiveIdempotency(ctx, tx, in, now)
	if err != nil {
		return CreateResult{}, err
	}

	if active {
		if existingHash == in.RequestHash {
			cmd, err := r.getByIDFromTx(ctx, tx, in.Context, existingCommandID)
			if err != nil {
				return CreateResult{}, err
			}
			if err := tx.Commit(); err != nil {
				return CreateResult{}, fmt.Errorf("commit postgres tx: %w", err)
			}
			committed = true
			return CreateResult{Command: cmd, Reused: true}, nil
		}
		return CreateResult{}, &IdempotencyConflictError{ExistingCommandID: existingCommandID}
	}

	cmd, err := r.insertCommandFromTx(ctx, tx, in.Context, in.CommandType, in.Payload, in.Visibility, now)
	if err != nil {
		return CreateResult{}, err
	}
	expiresAt := now.Add(in.TTL)
	if err := r.upsertIdempotencyFromTx(ctx, tx, in, cmd.ID, expiresAt, now); err != nil {
		return CreateResult{}, err
	}

	if err := tx.Commit(); err != nil {
		return CreateResult{}, fmt.Errorf("commit postgres tx: %w", err)
	}
	committed = true
	return CreateResult{Command: cmd, Reused: false}, nil
}

func (r *PostgresRepository) Get(ctx context.Context, req RequestContext, id string) (Command, error) {
	return r.getByID(ctx, req, id)
}

func (r *PostgresRepository) GetForAccess(ctx context.Context, req RequestContext, id string) (Command, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT c.id, c.tenant_id, c.workspace_id, c.owner_id, c.visibility, c.acl_json::text, c.command_type, c.payload::text, c.status, c.result::text, c.error_code, c.message_key, c.accepted_at, c.finished_at, c.created_at, c.updated_at,
		        COALESCE((SELECT ae.trace_id FROM audit_events ae WHERE ae.command_id = c.id ORDER BY ae.created_at DESC LIMIT 1), '') AS trace_id
		 FROM commands c
		 WHERE c.id = $1 AND c.tenant_id = $2 AND c.workspace_id = $3`,
		id,
		req.TenantID,
		req.WorkspaceID,
	)

	cmd, err := scanPostgresCommand(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Command{}, ErrNotFound
	}
	if err != nil {
		return Command{}, fmt.Errorf("query command for access: %w", err)
	}
	return cmd, nil
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

	if params.Cursor != "" {
		createdAt, id, err := DecodeCursor(params.Cursor)
		if err != nil {
			return ListResult{}, ErrInvalidCursor
		}

		rows, err := r.db.QueryContext(
			ctx,
			`SELECT c.id, c.tenant_id, c.workspace_id, c.owner_id, c.visibility, c.acl_json::text, c.command_type, c.payload::text, c.status, c.result::text, c.error_code, c.message_key, c.accepted_at, c.finished_at, c.created_at, c.updated_at,
			        COALESCE((SELECT ae.trace_id FROM audit_events ae WHERE ae.command_id = c.id ORDER BY ae.created_at DESC LIMIT 1), '') AS trace_id
			 FROM commands c
			 WHERE c.tenant_id = $1 AND c.workspace_id = $2
			   AND (
			     c.owner_id = $3
			     OR c.visibility = 'WORKSPACE'
			     OR EXISTS (
			       SELECT 1
			       FROM acl_entries a
			       WHERE a.tenant_id = c.tenant_id
			         AND a.workspace_id = c.workspace_id
			         AND a.resource_type = 'command'
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
			return ListResult{}, fmt.Errorf("list commands by cursor: %w", err)
		}
		defer rows.Close()

		items, err := scanPostgresCommands(rows)
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
		`SELECT c.id, c.tenant_id, c.workspace_id, c.owner_id, c.visibility, c.acl_json::text, c.command_type, c.payload::text, c.status, c.result::text, c.error_code, c.message_key, c.accepted_at, c.finished_at, c.created_at, c.updated_at,
		        COALESCE((SELECT ae.trace_id FROM audit_events ae WHERE ae.command_id = c.id ORDER BY ae.created_at DESC LIMIT 1), '') AS trace_id
		 FROM commands c
		 WHERE c.tenant_id = $1 AND c.workspace_id = $2
		   AND (
		     c.owner_id = $3
		     OR c.visibility = 'WORKSPACE'
		     OR EXISTS (
		       SELECT 1
		       FROM acl_entries a
		       WHERE a.tenant_id = c.tenant_id
		         AND a.workspace_id = c.workspace_id
		         AND a.resource_type = 'command'
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
		return ListResult{}, fmt.Errorf("list commands by page: %w", err)
	}
	defer rows.Close()

	items, err := scanPostgresCommands(rows)
	if err != nil {
		return ListResult{}, err
	}

	var total int64
	if err := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(1)
		 FROM commands c
		 WHERE c.tenant_id = $1 AND c.workspace_id = $2
		   AND (
		     c.owner_id = $3
		     OR c.visibility = 'WORKSPACE'
		     OR EXISTS (
		       SELECT 1
		       FROM acl_entries a
		       WHERE a.tenant_id = c.tenant_id
		         AND a.workspace_id = c.workspace_id
		         AND a.resource_type = 'command'
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
		return ListResult{}, fmt.Errorf("count commands: %w", err)
	}

	return ListResult{
		Items: items,
		Total: total,
	}, nil
}

func (r *PostgresRepository) HasCommandPermission(ctx context.Context, req RequestContext, commandID, permission string, now time.Time) (bool, error) {
	if strings.TrimSpace(commandID) == "" || strings.TrimSpace(permission) == "" {
		return false, nil
	}
	permission = strings.ToUpper(strings.TrimSpace(permission))
	if allowed, err := r.hasACLPermission(ctx, req.TenantID, req.WorkspaceID, "command", commandID, "user", req.UserID, permission, now.UTC()); err != nil {
		return false, err
	} else if allowed {
		return true, nil
	}
	for _, role := range req.Roles {
		role = strings.TrimSpace(role)
		if role == "" {
			continue
		}
		allowed, err := r.hasACLPermission(ctx, req.TenantID, req.WorkspaceID, "command", commandID, "role", role, permission, now.UTC())
		if err != nil {
			return false, err
		}
		if allowed {
			return true, nil
		}
	}
	return false, nil
}

func (r *PostgresRepository) GetShareResource(ctx context.Context, req RequestContext, resourceType, resourceID string) (ShareResource, error) {
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
		         WHERE id = $1 AND tenant_id = $2 AND workspace_id = $3`
	case "asset":
		query = `SELECT tenant_id, workspace_id, owner_id, visibility
		         FROM assets
		         WHERE id = $1 AND tenant_id = $2 AND workspace_id = $3`
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

func (r *PostgresRepository) HasShareResourcePermission(
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
	if allowed, err := r.hasACLPermission(ctx, req.TenantID, req.WorkspaceID, resourceType, resourceID, "user", req.UserID, permission, now.UTC()); err != nil {
		return false, err
	} else if allowed {
		return true, nil
	}
	for _, role := range req.Roles {
		role = strings.TrimSpace(role)
		if role == "" {
			continue
		}
		allowed, err := r.hasACLPermission(ctx, req.TenantID, req.WorkspaceID, resourceType, resourceID, "role", role, permission, now.UTC())
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
	tenantID, workspaceID, resourceType, resourceID, subjectType, subjectID, permission string,
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
		   AND a.resource_type = $3
		   AND a.resource_id = $4
		   AND a.subject_type = $5
		   AND a.subject_id = $6
		   AND (a.expires_at IS NULL OR a.expires_at >= $7)
		   AND a.permissions @> jsonb_build_array($8::text)
		 LIMIT 1`,
		tenantID,
		workspaceID,
		resourceType,
		resourceID,
		subjectType,
		subjectID,
		now.UTC(),
		permission,
	).Scan(&marker)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query acl permission: %w", err)
	}
	return true, nil
}

func (r *PostgresRepository) CreateShare(ctx context.Context, in ShareCreateInput) (Share, error) {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	permissionsRaw, err := json.Marshal(in.Permissions)
	if err != nil {
		return Share{}, fmt.Errorf("marshal permissions: %w", err)
	}

	shareID := newID("shr")
	_, err = r.db.ExecContext(
		ctx,
		`INSERT INTO acl_entries(id, tenant_id, workspace_id, resource_type, resource_id, subject_type, subject_id, permissions, expires_at, created_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9, $10, $11)`,
		shareID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
		in.ResourceType,
		in.ResourceID,
		in.SubjectType,
		in.SubjectID,
		string(permissionsRaw),
		in.ExpiresAt,
		in.Context.UserID,
		now,
	)
	if err != nil {
		return Share{}, fmt.Errorf("insert share: %w", err)
	}

	return r.getShareByID(ctx, in.Context, shareID)
}

func (r *PostgresRepository) ListShares(ctx context.Context, params ShareListParams) (ShareListResult, error) {
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
			`SELECT id, tenant_id, workspace_id, resource_type, resource_id, subject_type, subject_id, permissions::text, expires_at, created_by, created_at
			 FROM acl_entries
			 WHERE tenant_id = $1 AND workspace_id = $2
			   AND ((created_at < $3) OR (created_at = $4 AND id < $5))
			 ORDER BY created_at DESC, id DESC
			 LIMIT $6`,
			params.Context.TenantID,
			params.Context.WorkspaceID,
			createdAt.UTC(),
			createdAt.UTC(),
			id,
			pageSize,
		)
		if err != nil {
			return ShareListResult{}, fmt.Errorf("list shares by cursor: %w", err)
		}
		defer rows.Close()

		items, err := scanPostgresShares(rows)
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
		`SELECT id, tenant_id, workspace_id, resource_type, resource_id, subject_type, subject_id, permissions::text, expires_at, created_by, created_at
		 FROM acl_entries
		 WHERE tenant_id = $1 AND workspace_id = $2
		 ORDER BY created_at DESC, id DESC
		 LIMIT $3 OFFSET $4`,
		params.Context.TenantID,
		params.Context.WorkspaceID,
		pageSize,
		offset,
	)
	if err != nil {
		return ShareListResult{}, fmt.Errorf("list shares by page: %w", err)
	}
	defer rows.Close()

	items, err := scanPostgresShares(rows)
	if err != nil {
		return ShareListResult{}, err
	}

	var total int64
	if err := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(1) FROM acl_entries WHERE tenant_id = $1 AND workspace_id = $2`,
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

func (r *PostgresRepository) DeleteShare(ctx context.Context, req RequestContext, shareID string) error {
	result, err := r.db.ExecContext(
		ctx,
		`DELETE FROM acl_entries
		 WHERE id = $1 AND tenant_id = $2 AND workspace_id = $3 AND created_by = $4`,
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

func (r *PostgresRepository) AppendCommandEvent(ctx context.Context, req RequestContext, commandID, eventType string, payload []byte) error {
	if len(payload) == 0 {
		payload = []byte("{}")
	}
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO command_events(id, command_id, event_type, payload, created_at)
		 VALUES ($1, $2, $3, $4::jsonb, $5)`,
		newID("cevt"),
		commandID,
		eventType,
		string(payload),
		time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("insert command event: %w", err)
	}
	return nil
}

func (r *PostgresRepository) AppendAuditEvent(ctx context.Context, req RequestContext, commandID, eventType, decision, reason string, payload []byte) error {
	payload = buildAuditPayload(req, commandID, eventType, decision, reason, payload)
	traceID := strings.TrimSpace(req.TraceID)
	if traceID == "" {
		traceID = newID("trace")
	}
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO audit_events(id, tenant_id, workspace_id, user_id, trace_id, command_id, event_type, resource_type, resource_id, decision, reason, payload, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12::jsonb, $13)`,
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
		time.Now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}
	return nil
}

func (r *PostgresRepository) SetStatus(ctx context.Context, req RequestContext, commandID, status string, result []byte, errorCode, messageKey string, finishedAt *time.Time) (Command, error) {
	updatedAt := time.Now().UTC()

	var resultVal any
	if len(result) > 0 {
		resultVal = string(result)
	}

	var errorCodeVal any
	if errorCode != "" {
		errorCodeVal = errorCode
	}

	var messageKeyVal any
	if messageKey != "" {
		messageKeyVal = messageKey
	}

	_, err := r.db.ExecContext(
		ctx,
		`UPDATE commands
		 SET status = $1,
		     result = COALESCE($2::jsonb, result),
		     error_code = $3,
		     message_key = $4,
		     finished_at = $5,
		     updated_at = $6
		 WHERE id = $7 AND tenant_id = $8 AND workspace_id = $9 AND owner_id = $10`,
		status,
		resultVal,
		errorCodeVal,
		messageKeyVal,
		finishedAt,
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

func (r *PostgresRepository) lookupActiveIdempotency(ctx context.Context, tx *sql.Tx, in CreateInput, now time.Time) (bool, string, string, error) {
	var requestHash, commandID string
	err := tx.QueryRowContext(
		ctx,
		`SELECT request_hash, command_id
		 FROM command_idempotency
		 WHERE tenant_id = $1 AND workspace_id = $2 AND owner_id = $3 AND idempotency_key = $4 AND expires_at >= $5
		 FOR UPDATE`,
		in.Context.TenantID,
		in.Context.WorkspaceID,
		in.Context.OwnerID,
		in.IdempotencyKey,
		now.UTC(),
	).Scan(&requestHash, &commandID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, "", "", nil
	}
	if err != nil {
		return false, "", "", fmt.Errorf("query idempotency mapping: %w", err)
	}
	return true, requestHash, commandID, nil
}

func (r *PostgresRepository) upsertIdempotencyFromTx(ctx context.Context, tx *sql.Tx, in CreateInput, commandID string, expiresAt, now time.Time) error {
	_, err := tx.ExecContext(
		ctx,
		`INSERT INTO command_idempotency(tenant_id, workspace_id, owner_id, idempotency_key, request_hash, command_id, expires_at, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 ON CONFLICT(tenant_id, workspace_id, owner_id, idempotency_key)
		 DO UPDATE SET request_hash = EXCLUDED.request_hash,
		               command_id = EXCLUDED.command_id,
		               expires_at = EXCLUDED.expires_at,
		               created_at = EXCLUDED.created_at`,
		in.Context.TenantID,
		in.Context.WorkspaceID,
		in.Context.OwnerID,
		in.IdempotencyKey,
		in.RequestHash,
		commandID,
		expiresAt.UTC(),
		now.UTC(),
	)
	if err != nil {
		return fmt.Errorf("upsert idempotency mapping: %w", err)
	}
	return nil
}

func (r *PostgresRepository) insertCommand(ctx context.Context, req RequestContext, commandType string, payload []byte, visibility string, now time.Time) (Command, error) {
	cmdID := newID("cmd")
	if visibility == "" {
		visibility = VisibilityPrivate
	}
	if _, err := r.db.ExecContext(
		ctx,
		`INSERT INTO commands(id, tenant_id, workspace_id, owner_id, visibility, acl_json, command_type, payload, status, accepted_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8::jsonb, $9, $10, $11, $12)`,
		cmdID,
		req.TenantID,
		req.WorkspaceID,
		req.OwnerID,
		visibility,
		"[]",
		commandType,
		string(payload),
		StatusAccepted,
		now.UTC(),
		now.UTC(),
		now.UTC(),
	); err != nil {
		return Command{}, fmt.Errorf("insert command: %w", err)
	}
	return r.getByID(ctx, req, cmdID)
}

func (r *PostgresRepository) insertCommandFromTx(ctx context.Context, tx *sql.Tx, req RequestContext, commandType string, payload []byte, visibility string, now time.Time) (Command, error) {
	cmdID := newID("cmd")
	if visibility == "" {
		visibility = VisibilityPrivate
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO commands(id, tenant_id, workspace_id, owner_id, visibility, acl_json, command_type, payload, status, accepted_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8::jsonb, $9, $10, $11, $12)`,
		cmdID,
		req.TenantID,
		req.WorkspaceID,
		req.OwnerID,
		visibility,
		"[]",
		commandType,
		string(payload),
		StatusAccepted,
		now.UTC(),
		now.UTC(),
		now.UTC(),
	); err != nil {
		return Command{}, fmt.Errorf("insert command in tx: %w", err)
	}
	return r.getByIDFromTx(ctx, tx, req, cmdID)
}

func (r *PostgresRepository) getByID(ctx context.Context, req RequestContext, id string) (Command, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT c.id, c.tenant_id, c.workspace_id, c.owner_id, c.visibility, c.acl_json::text, c.command_type, c.payload::text, c.status, c.result::text, c.error_code, c.message_key, c.accepted_at, c.finished_at, c.created_at, c.updated_at,
		        COALESCE((SELECT ae.trace_id FROM audit_events ae WHERE ae.command_id = c.id ORDER BY ae.created_at DESC LIMIT 1), '') AS trace_id
		 FROM commands c
		 WHERE c.id = $1 AND c.tenant_id = $2 AND c.workspace_id = $3 AND c.owner_id = $4`,
		id,
		req.TenantID,
		req.WorkspaceID,
		req.OwnerID,
	)
	cmd, err := scanPostgresCommand(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Command{}, ErrNotFound
	}
	if err != nil {
		return Command{}, fmt.Errorf("query command: %w", err)
	}
	return cmd, nil
}

func (r *PostgresRepository) getByIDFromTx(ctx context.Context, tx *sql.Tx, req RequestContext, id string) (Command, error) {
	row := tx.QueryRowContext(
		ctx,
		`SELECT c.id, c.tenant_id, c.workspace_id, c.owner_id, c.visibility, c.acl_json::text, c.command_type, c.payload::text, c.status, c.result::text, c.error_code, c.message_key, c.accepted_at, c.finished_at, c.created_at, c.updated_at,
		        COALESCE((SELECT ae.trace_id FROM audit_events ae WHERE ae.command_id = c.id ORDER BY ae.created_at DESC LIMIT 1), '') AS trace_id
		 FROM commands c
		 WHERE c.id = $1 AND c.tenant_id = $2 AND c.workspace_id = $3 AND c.owner_id = $4`,
		id,
		req.TenantID,
		req.WorkspaceID,
		req.OwnerID,
	)
	cmd, err := scanPostgresCommand(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Command{}, ErrNotFound
	}
	if err != nil {
		return Command{}, fmt.Errorf("query command from tx: %w", err)
	}
	return cmd, nil
}

func (r *PostgresRepository) getShareByID(ctx context.Context, req RequestContext, id string) (Share, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, resource_type, resource_id, subject_type, subject_id, permissions::text, expires_at, created_by, created_at
		 FROM acl_entries
		 WHERE id = $1 AND tenant_id = $2 AND workspace_id = $3`,
		id,
		req.TenantID,
		req.WorkspaceID,
	)
	share, err := scanPostgresShare(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Share{}, ErrShareNotFound
	}
	if err != nil {
		return Share{}, fmt.Errorf("query share: %w", err)
	}
	return share, nil
}

func scanPostgresCommands(rows *sql.Rows) ([]Command, error) {
	items := make([]Command, 0)
	for rows.Next() {
		cmd, err := scanPostgresCommand(rows)
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

func scanPostgresCommand(row rowScanner) (Command, error) {
	var (
		cmd           Command
		aclRaw        string
		payloadRaw    string
		resultRaw     sql.NullString
		errorCodeRaw  sql.NullString
		messageKeyRaw sql.NullString
		finishedAtRaw sql.NullTime
		traceIDRaw    string
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
		&cmd.AcceptedAt,
		&finishedAtRaw,
		&cmd.CreatedAt,
		&cmd.UpdatedAt,
		&traceIDRaw,
	); err != nil {
		return Command{}, err
	}

	if strings.TrimSpace(aclRaw) == "" {
		aclRaw = "[]"
	}
	if strings.TrimSpace(payloadRaw) == "" {
		payloadRaw = "{}"
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
	if finishedAtRaw.Valid {
		finishedAt := finishedAtRaw.Time.UTC()
		cmd.FinishedAt = &finishedAt
	}

	cmd.AcceptedAt = cmd.AcceptedAt.UTC()
	cmd.CreatedAt = cmd.CreatedAt.UTC()
	cmd.UpdatedAt = cmd.UpdatedAt.UTC()
	cmd.TraceID = strings.TrimSpace(traceIDRaw)
	return cmd, nil
}

func scanPostgresShares(rows *sql.Rows) ([]Share, error) {
	items := make([]Share, 0)
	for rows.Next() {
		item, err := scanPostgresShare(rows)
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

func scanPostgresShare(row rowScanner) (Share, error) {
	var (
		item        Share
		permissions string
		expiresAt   sql.NullTime
		createdAt   time.Time
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
		&expiresAt,
		&item.CreatedBy,
		&createdAt,
	); err != nil {
		return Share{}, err
	}

	if err := json.Unmarshal([]byte(permissions), &item.Permissions); err != nil {
		return Share{}, fmt.Errorf("unmarshal permissions: %w", err)
	}
	if expiresAt.Valid {
		v := expiresAt.Time.UTC()
		item.ExpiresAt = &v
	}
	item.CreatedAt = createdAt.UTC()
	return item, nil
}
