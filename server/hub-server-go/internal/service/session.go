package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/goyais/hub/internal/middleware"
)

// SessionSummary is the API-facing session shape.
type SessionSummary struct {
	SessionID         string  `json:"session_id"`
	WorkspaceID       string  `json:"workspace_id"`
	ProjectID         string  `json:"project_id"`
	Title             string  `json:"title"`
	Mode              string  `json:"mode"`
	ModelConfigID     *string `json:"model_config_id,omitempty"`
	SkillSetIDs       string  `json:"skill_set_ids"`
	MCPConnectorIDs   string  `json:"mcp_connector_ids"`
	UseWorktree       bool    `json:"use_worktree"`
	ActiveExecutionID *string `json:"active_execution_id,omitempty"`
	Status            string  `json:"status"`
	CreatedBy         string  `json:"created_by"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
	ArchivedAt        *string `json:"archived_at,omitempty"`
}

type CreateSessionInput struct {
	ProjectID       string  `json:"project_id"`
	Title           string  `json:"title"`
	Mode            string  `json:"mode"`
	ModelConfigID   *string `json:"model_config_id"`
	SkillSetIDs     string  `json:"skill_set_ids"`
	MCPConnectorIDs string  `json:"mcp_connector_ids"`
	UseWorktree     *bool   `json:"use_worktree"`
}

type UpdateSessionInput struct {
	Title           *string `json:"title"`
	Mode            *string `json:"mode"`
	ModelConfigID   *string `json:"model_config_id"`
	SkillSetIDs     *string `json:"skill_set_ids"`
	MCPConnectorIDs *string `json:"mcp_connector_ids"`
	UseWorktree     *bool   `json:"use_worktree"`
}

type SessionService struct {
	db *sql.DB
}

func NewSessionService(db *sql.DB) *SessionService {
	return &SessionService{db: db}
}

func (s *SessionService) List(ctx context.Context, workspaceID, projectID string) ([]SessionSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT session_id, workspace_id, project_id, title, mode, model_config_id,
		       skill_set_ids, mcp_connector_ids, use_worktree,
		       active_execution_id, status, created_by, created_at, updated_at, archived_at
		FROM sessions
		WHERE project_id = ? AND workspace_id = ? AND archived_at IS NULL
		ORDER BY updated_at DESC`, projectID, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSessions(rows)
}

func (s *SessionService) Get(ctx context.Context, workspaceID, sessionID string) (*SessionSummary, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT session_id, workspace_id, project_id, title, mode, model_config_id,
		       skill_set_ids, mcp_connector_ids, use_worktree,
		       active_execution_id, status, created_by, created_at, updated_at, archived_at
		FROM sessions WHERE session_id = ? AND workspace_id = ?`, sessionID, workspaceID)
	sess, err := scanSession(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return sess, err
}

func (s *SessionService) Create(ctx context.Context, workspaceID string, in CreateSessionInput) (*SessionSummary, error) {
	user := middleware.UserFromCtx(ctx)
	if user == nil {
		return nil, fmt.Errorf("unauthenticated")
	}

	// Validate project belongs to workspace
	var count int
	if err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(1) FROM projects WHERE project_id = ? AND workspace_id = ?`,
		in.ProjectID, workspaceID).Scan(&count); err != nil || count == 0 {
		return nil, fmt.Errorf("project not found in workspace")
	}

	mode := in.Mode
	if mode == "" {
		mode = "agent"
	}
	skillSetIDs := in.SkillSetIDs
	if skillSetIDs == "" {
		skillSetIDs = "[]"
	}
	mcpConnectorIDs := in.MCPConnectorIDs
	if mcpConnectorIDs == "" {
		mcpConnectorIDs = "[]"
	}
	useWorktree := 1
	if in.UseWorktree != nil && !*in.UseWorktree {
		useWorktree = 0
	}
	title := in.Title
	if title == "" {
		title = "New Session"
	}

	sessionID := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339Nano)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (
			session_id, workspace_id, project_id, title, mode,
			model_config_id, skill_set_ids, mcp_connector_ids, use_worktree,
			status, created_by, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'idle', ?, ?, ?)`,
		sessionID, workspaceID, in.ProjectID, title, mode,
		in.ModelConfigID, skillSetIDs, mcpConnectorIDs, useWorktree,
		user.UserID, now, now)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, workspaceID, sessionID)
}

func (s *SessionService) Update(ctx context.Context, workspaceID, sessionID string, in UpdateSessionInput) (*SessionSummary, error) {
	// Build dynamic update — only set provided fields
	_, err := s.db.ExecContext(ctx, `
		UPDATE sessions SET
			title            = COALESCE(?, title),
			mode             = COALESCE(?, mode),
			model_config_id  = COALESCE(?, model_config_id),
			skill_set_ids    = COALESCE(?, skill_set_ids),
			mcp_connector_ids = COALESCE(?, mcp_connector_ids),
			updated_at       = datetime('now')
		WHERE session_id = ? AND workspace_id = ?`,
		in.Title, in.Mode, in.ModelConfigID, in.SkillSetIDs, in.MCPConnectorIDs,
		sessionID, workspaceID)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, workspaceID, sessionID)
}

func (s *SessionService) Delete(ctx context.Context, workspaceID, sessionID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM sessions
		WHERE session_id = ? AND workspace_id = ?`, sessionID, workspaceID)
	return err
}

// WorkspaceService handles workspace listing.
type WorkspaceService struct {
	db *sql.DB
}

func NewWorkspaceService(db *sql.DB) *WorkspaceService {
	return &WorkspaceService{db: db}
}

func (s *WorkspaceService) List(ctx context.Context) ([]map[string]any, error) {
	user := middleware.UserFromCtx(ctx)
	if user == nil {
		return nil, fmt.Errorf("unauthenticated")
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT w.workspace_id, w.name, w.slug, w.kind, r.name AS role_name
		FROM workspaces w
		JOIN workspace_members wm ON wm.workspace_id = w.workspace_id
		JOIN roles r ON r.role_id = wm.role_id
		WHERE wm.user_id = ? AND wm.status = 'active'
		ORDER BY w.name`, user.UserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var wsID, name, slug, kind, roleName string
		if err := rows.Scan(&wsID, &name, &slug, &kind, &roleName); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"workspace_id": wsID,
			"name":         name,
			"slug":         slug,
			"kind":         kind,
			"role_name":    roleName,
		})
	}
	return out, nil
}

// --- scan helpers ---

func scanSessions(rows *sql.Rows) ([]SessionSummary, error) {
	var out []SessionSummary
	for rows.Next() {
		s, err := scanSessionRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *s)
	}
	return out, rows.Err()
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSession(row rowScanner) (*SessionSummary, error) {
	return scanSessionRow(row)
}

func scanSessionRow(row rowScanner) (*SessionSummary, error) {
	var s SessionSummary
	var modelConfigID, activeExecutionID, archivedAt sql.NullString
	var useWorktree int
	if err := row.Scan(
		&s.SessionID, &s.WorkspaceID, &s.ProjectID, &s.Title, &s.Mode,
		&modelConfigID, &s.SkillSetIDs, &s.MCPConnectorIDs, &useWorktree,
		&activeExecutionID, &s.Status, &s.CreatedBy, &s.CreatedAt, &s.UpdatedAt, &archivedAt,
	); err != nil {
		return nil, err
	}
	s.UseWorktree = useWorktree == 1
	if modelConfigID.Valid {
		s.ModelConfigID = &modelConfigID.String
	}
	if activeExecutionID.Valid {
		s.ActiveExecutionID = &activeExecutionID.String
	}
	if archivedAt.Valid {
		s.ArchivedAt = &archivedAt.String
	}
	return &s, nil
}
