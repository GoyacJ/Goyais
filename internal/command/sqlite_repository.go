package command

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
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
		cmd, err := r.insertCommand(ctx, in.Context, in.CommandType, in.Payload, now)
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

	cmd, err := r.insertCommandFromConn(ctx, conn, in.Context, in.CommandType, in.Payload, now)
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
	cmd, err := r.getByID(ctx, req, id)
	if err != nil {
		return Command{}, err
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

	if params.Cursor != "" {
		createdAt, id, err := DecodeCursor(params.Cursor)
		if err != nil {
			return ListResult{}, ErrInvalidCursor
		}

		rows, err := r.db.QueryContext(
			ctx,
			`SELECT id, tenant_id, workspace_id, owner_id, visibility, acl_json, command_type, payload, status, result, error_code, message_key, accepted_at, finished_at, created_at, updated_at
			 FROM commands
			 WHERE tenant_id = ? AND workspace_id = ? AND owner_id = ?
			   AND ((created_at < ?) OR (created_at = ? AND id < ?))
			 ORDER BY created_at DESC, id DESC
			 LIMIT ?`,
			params.Context.TenantID,
			params.Context.WorkspaceID,
			params.Context.OwnerID,
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
		`SELECT id, tenant_id, workspace_id, owner_id, visibility, acl_json, command_type, payload, status, result, error_code, message_key, accepted_at, finished_at, created_at, updated_at
		 FROM commands
		 WHERE tenant_id = ? AND workspace_id = ? AND owner_id = ?
		 ORDER BY created_at DESC, id DESC
		 LIMIT ? OFFSET ?`,
		params.Context.TenantID,
		params.Context.WorkspaceID,
		params.Context.OwnerID,
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
		`SELECT COUNT(1) FROM commands WHERE tenant_id = ? AND workspace_id = ? AND owner_id = ?`,
		params.Context.TenantID,
		params.Context.WorkspaceID,
		params.Context.OwnerID,
	).Scan(&total); err != nil {
		return ListResult{}, fmt.Errorf("count commands: %w", err)
	}

	return ListResult{
		Items: items,
		Total: total,
	}, nil
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
	if len(payload) == 0 {
		payload = []byte("{}")
	}
	_, err := r.db.ExecContext(
		ctx,
		`INSERT INTO audit_events(id, tenant_id, workspace_id, user_id, trace_id, command_id, event_type, resource_type, resource_id, decision, reason, payload, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		newID("aevt"),
		req.TenantID,
		req.WorkspaceID,
		req.UserID,
		newID("trace"),
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

func (r *SQLiteRepository) insertCommand(ctx context.Context, req RequestContext, commandType string, payload []byte, now time.Time) (Command, error) {
	cmdID := newID("cmd")
	if _, err := r.db.ExecContext(
		ctx,
		`INSERT INTO commands(id, tenant_id, workspace_id, owner_id, visibility, acl_json, command_type, payload, status, accepted_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		cmdID,
		req.TenantID,
		req.WorkspaceID,
		req.OwnerID,
		"PRIVATE",
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

func (r *SQLiteRepository) insertCommandFromConn(ctx context.Context, conn *sql.Conn, req RequestContext, commandType string, payload []byte, now time.Time) (Command, error) {
	cmdID := newID("cmd")
	if _, err := conn.ExecContext(
		ctx,
		`INSERT INTO commands(id, tenant_id, workspace_id, owner_id, visibility, acl_json, command_type, payload, status, accepted_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		cmdID,
		req.TenantID,
		req.WorkspaceID,
		req.OwnerID,
		"PRIVATE",
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
