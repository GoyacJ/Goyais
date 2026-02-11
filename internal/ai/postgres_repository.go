package ai

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

func (r *PostgresRepository) CreateSession(ctx context.Context, in CreateSessionInput) (Session, error) {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if len(in.Inputs) == 0 {
		in.Inputs = json.RawMessage(`{}`)
	}
	if len(in.Constraints) == 0 {
		in.Constraints = json.RawMessage(`{}`)
	}
	if len(in.Preferences) == 0 {
		in.Preferences = json.RawMessage(`{}`)
	}

	sessionID := newID("sess")
	if _, err := r.db.ExecContext(
		ctx,
		`INSERT INTO ai_sessions(
			id, tenant_id, workspace_id, owner_id, visibility, acl_json,
			title, goal, status, inputs, constraints, preferences,
			archived_at, last_turn_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9, $10::jsonb, $11::jsonb, $12::jsonb, $13, $14, $15, $16)`,
		sessionID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
		in.Context.OwnerID,
		in.Visibility,
		"[]",
		in.Title,
		in.Goal,
		SessionStatusActive,
		string(in.Inputs),
		string(in.Constraints),
		string(in.Preferences),
		nil,
		nil,
		now,
		now,
	); err != nil {
		return Session{}, fmt.Errorf("insert ai session: %w", err)
	}

	return r.GetSessionForAccess(ctx, in.Context, sessionID)
}

func (r *PostgresRepository) ArchiveSession(ctx context.Context, in ArchiveSessionInput) (Session, error) {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	result, err := r.db.ExecContext(
		ctx,
		`UPDATE ai_sessions
		 SET status = $1, archived_at = COALESCE(archived_at, $2), updated_at = $3
		 WHERE id = $4 AND tenant_id = $5 AND workspace_id = $6`,
		SessionStatusArchived,
		now,
		now,
		in.SessionID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
	)
	if err != nil {
		return Session{}, fmt.Errorf("archive ai session: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return Session{}, fmt.Errorf("archive ai session rows affected: %w", err)
	}
	if affected == 0 {
		return Session{}, ErrSessionNotFound
	}
	return r.GetSessionForAccess(ctx, in.Context, in.SessionID)
}

func (r *PostgresRepository) CreateTurn(ctx context.Context, in CreateTurnInput) (SessionTurn, error) {
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return SessionTurn{}, fmt.Errorf("begin create ai turn tx: %w", err)
	}
	defer tx.Rollback()

	var (
		status     string
		visibility string
	)
	if err := tx.QueryRowContext(
		ctx,
		`SELECT status, visibility
		 FROM ai_sessions
		 WHERE id = $1 AND tenant_id = $2 AND workspace_id = $3`,
		in.SessionID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
	).Scan(&status, &visibility); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return SessionTurn{}, ErrSessionNotFound
		}
		return SessionTurn{}, fmt.Errorf("query ai session for turn: %w", err)
	}
	if status != SessionStatusActive {
		return SessionTurn{}, ErrInvalidRequest
	}
	commandIDsJSON := marshalCommandIDsJSON(in.CommandIDs)

	userTurnID := newID("turn")
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO ai_session_turns(
			id, session_id, tenant_id, workspace_id, owner_id, visibility, role, content, command_type, command_ids, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11)`,
		userTurnID,
		in.SessionID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
		in.Context.OwnerID,
		visibility,
		TurnRoleUser,
		in.UserMessage,
		in.CommandType,
		"[]",
		now,
	); err != nil {
		return SessionTurn{}, fmt.Errorf("insert ai user turn: %w", err)
	}

	assistantTurnID := newID("turn")
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO ai_session_turns(
			id, session_id, tenant_id, workspace_id, owner_id, visibility, role, content, command_type, command_ids, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11)`,
		assistantTurnID,
		in.SessionID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
		in.Context.OwnerID,
		visibility,
		TurnRoleAssistant,
		in.AssistantMessage,
		in.CommandType,
		commandIDsJSON,
		now,
	); err != nil {
		return SessionTurn{}, fmt.Errorf("insert ai assistant turn: %w", err)
	}

	if _, err := tx.ExecContext(
		ctx,
		`UPDATE ai_sessions
		 SET last_turn_at = $1, updated_at = $2
		 WHERE id = $3 AND tenant_id = $4 AND workspace_id = $5`,
		now,
		now,
		in.SessionID,
		in.Context.TenantID,
		in.Context.WorkspaceID,
	); err != nil {
		return SessionTurn{}, fmt.Errorf("update ai session last_turn_at: %w", err)
	}

	row := tx.QueryRowContext(
		ctx,
		`SELECT id, session_id, tenant_id, workspace_id, owner_id, visibility, role, content, command_type, command_ids::text, created_at
		 FROM ai_session_turns
		 WHERE id = $1`,
		assistantTurnID,
	)
	turn, err := scanPostgresTurn(row)
	if err != nil {
		return SessionTurn{}, fmt.Errorf("query ai assistant turn: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return SessionTurn{}, fmt.Errorf("commit create ai turn tx: %w", err)
	}
	return turn, nil
}

func (r *PostgresRepository) GetSessionForAccess(ctx context.Context, req command.RequestContext, sessionID string) (Session, error) {
	row := r.db.QueryRowContext(
		ctx,
		`SELECT id, tenant_id, workspace_id, owner_id, visibility, acl_json::text, title, goal, status, inputs::text, constraints::text, preferences::text, archived_at, last_turn_at, created_at, updated_at
		 FROM ai_sessions
		 WHERE id = $1 AND tenant_id = $2 AND workspace_id = $3`,
		sessionID,
		req.TenantID,
		req.WorkspaceID,
	)
	item, err := scanPostgresSession(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Session{}, ErrSessionNotFound
	}
	if err != nil {
		return Session{}, fmt.Errorf("query ai session: %w", err)
	}
	return item, nil
}

func (r *PostgresRepository) ListSessions(ctx context.Context, params SessionListParams) (SessionListResult, error) {
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

	baseFilter := `FROM ai_sessions s
		WHERE s.tenant_id = $1 AND s.workspace_id = $2
		  AND (
		    s.owner_id = $3
		    OR s.visibility = 'WORKSPACE'
		    OR EXISTS (
		      SELECT 1 FROM acl_entries a
		      WHERE a.tenant_id = s.tenant_id
		        AND a.workspace_id = s.workspace_id
		        AND a.resource_type = 'ai_session'
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
			return SessionListResult{}, ErrInvalidCursor
		}
		rows, err := r.db.QueryContext(
			ctx,
			`SELECT s.id, s.tenant_id, s.workspace_id, s.owner_id, s.visibility, s.acl_json::text, s.title, s.goal, s.status, s.inputs::text, s.constraints::text, s.preferences::text, s.archived_at, s.last_turn_at, s.created_at, s.updated_at
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
			return SessionListResult{}, fmt.Errorf("list ai sessions by cursor: %w", err)
		}
		defer rows.Close()

		items, err := scanPostgresSessions(rows)
		if err != nil {
			return SessionListResult{}, err
		}

		nextCursor := ""
		if len(items) == pageSize {
			last := items[len(items)-1]
			nextCursor, err = command.EncodeCursor(last.CreatedAt, last.ID)
			if err != nil {
				return SessionListResult{}, fmt.Errorf("encode ai session cursor: %w", err)
			}
		}

		return SessionListResult{
			Items:      items,
			NextCursor: nextCursor,
			UsedCursor: true,
		}, nil
	}

	offset := (page - 1) * pageSize
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT s.id, s.tenant_id, s.workspace_id, s.owner_id, s.visibility, s.acl_json::text, s.title, s.goal, s.status, s.inputs::text, s.constraints::text, s.preferences::text, s.archived_at, s.last_turn_at, s.created_at, s.updated_at
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
		return SessionListResult{}, fmt.Errorf("list ai sessions by page: %w", err)
	}
	defer rows.Close()

	items, err := scanPostgresSessions(rows)
	if err != nil {
		return SessionListResult{}, err
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
		return SessionListResult{}, fmt.Errorf("count ai sessions: %w", err)
	}

	return SessionListResult{
		Items: items,
		Total: total,
	}, nil
}

func (r *PostgresRepository) ListSessionTurns(ctx context.Context, req command.RequestContext, sessionID string) ([]SessionTurn, error) {
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT id, session_id, tenant_id, workspace_id, owner_id, visibility, role, content, command_type, command_ids::text, created_at
		 FROM ai_session_turns
		 WHERE session_id = $1 AND tenant_id = $2 AND workspace_id = $3
		 ORDER BY created_at ASC, id ASC`,
		sessionID,
		req.TenantID,
		req.WorkspaceID,
	)
	if err != nil {
		return nil, fmt.Errorf("list ai session turns: %w", err)
	}
	defer rows.Close()
	return scanPostgresTurns(rows)
}

func (r *PostgresRepository) HasSessionPermission(
	ctx context.Context,
	req command.RequestContext,
	sessionID string,
	permission string,
	now time.Time,
) (bool, error) {
	if strings.TrimSpace(sessionID) == "" || strings.TrimSpace(permission) == "" {
		return false, nil
	}
	row := r.db.QueryRowContext(
		ctx,
		`SELECT 1
		 FROM acl_entries a
		 WHERE a.tenant_id = $1
		   AND a.workspace_id = $2
		   AND a.resource_type = 'ai_session'
		   AND a.resource_id = $3
		   AND a.subject_type = 'user'
		   AND a.subject_id = $4
		   AND (a.expires_at IS NULL OR a.expires_at >= $5)
		   AND a.permissions @> jsonb_build_array($6)
		 LIMIT 1`,
		req.TenantID,
		req.WorkspaceID,
		sessionID,
		req.UserID,
		now.UTC(),
		strings.ToUpper(strings.TrimSpace(permission)),
	)
	var marker int
	err := row.Scan(&marker)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("query ai session permission: %w", err)
	}
	return true, nil
}

type postgresRowScanner interface {
	Scan(dest ...any) error
}

func scanPostgresSessions(rows *sql.Rows) ([]Session, error) {
	items := make([]Session, 0)
	for rows.Next() {
		item, err := scanPostgresSession(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ai sessions: %w", err)
	}
	return items, nil
}

func scanPostgresSession(row postgresRowScanner) (Session, error) {
	var (
		item           Session
		aclRaw         string
		inputsRaw      string
		constraintsRaw string
		preferencesRaw string
		archivedAtRaw  sql.NullTime
		lastTurnAtRaw  sql.NullTime
	)
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.WorkspaceID,
		&item.OwnerID,
		&item.Visibility,
		&aclRaw,
		&item.Title,
		&item.Goal,
		&item.Status,
		&inputsRaw,
		&constraintsRaw,
		&preferencesRaw,
		&archivedAtRaw,
		&lastTurnAtRaw,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return Session{}, err
	}
	if strings.TrimSpace(aclRaw) == "" {
		aclRaw = "[]"
	}
	if strings.TrimSpace(inputsRaw) == "" {
		inputsRaw = "{}"
	}
	if strings.TrimSpace(constraintsRaw) == "" {
		constraintsRaw = "{}"
	}
	if strings.TrimSpace(preferencesRaw) == "" {
		preferencesRaw = "{}"
	}
	item.ACLJSON = json.RawMessage(aclRaw)
	item.InputsJSON = json.RawMessage(inputsRaw)
	item.ConstraintsJSON = json.RawMessage(constraintsRaw)
	item.PreferencesJSON = json.RawMessage(preferencesRaw)
	if archivedAtRaw.Valid {
		t := archivedAtRaw.Time.UTC()
		item.ArchivedAt = &t
	}
	if lastTurnAtRaw.Valid {
		t := lastTurnAtRaw.Time.UTC()
		item.LastTurnAt = &t
	}
	item.CreatedAt = item.CreatedAt.UTC()
	item.UpdatedAt = item.UpdatedAt.UTC()
	return item, nil
}

func scanPostgresTurns(rows *sql.Rows) ([]SessionTurn, error) {
	items := make([]SessionTurn, 0)
	for rows.Next() {
		item, err := scanPostgresTurn(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ai turns: %w", err)
	}
	return items, nil
}

func scanPostgresTurn(row postgresRowScanner) (SessionTurn, error) {
	var (
		item           SessionTurn
		commandTypeRaw sql.NullString
		commandIDsRaw  string
		createdAt      time.Time
	)
	if err := row.Scan(
		&item.ID,
		&item.SessionID,
		&item.TenantID,
		&item.WorkspaceID,
		&item.OwnerID,
		&item.Visibility,
		&item.Role,
		&item.Content,
		&commandTypeRaw,
		&commandIDsRaw,
		&createdAt,
	); err != nil {
		return SessionTurn{}, err
	}
	if commandTypeRaw.Valid {
		item.CommandType = commandTypeRaw.String
	}
	if strings.TrimSpace(commandIDsRaw) == "" {
		commandIDsRaw = "[]"
	}
	item.CommandIDsJSON = json.RawMessage(commandIDsRaw)
	item.CreatedAt = createdAt.UTC()
	return item, nil
}
