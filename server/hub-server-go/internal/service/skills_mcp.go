package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/goyais/hub/internal/middleware"
)

// ─────────────────────────────────────────────────────────────────────────────
// Skill Sets
// ─────────────────────────────────────────────────────────────────────────────

type SkillSetSummary struct {
	SkillSetID  string  `json:"skill_set_id"`
	WorkspaceID string  `json:"workspace_id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	CreatedBy   string  `json:"created_by"`
	CreatedAt   string  `json:"created_at"`
}

type CreateSkillSetInput struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type UpdateSkillSetInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

// SkillSummary is one skill within a skill set.
type SkillSummary struct {
	SkillID    string `json:"skill_id"`
	SkillSetID string `json:"skill_set_id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	ConfigJSON string `json:"config_json"`
	CreatedAt  string `json:"created_at"`
}

type CreateSkillInput struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	ConfigJSON string `json:"config_json"`
}

type SkillSetService struct {
	db *sql.DB
}

func NewSkillSetService(db *sql.DB) *SkillSetService {
	return &SkillSetService{db: db}
}

func (s *SkillSetService) List(ctx context.Context, workspaceID string) ([]SkillSetSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT skill_set_id, workspace_id, name, description, created_by, created_at
		FROM skill_sets WHERE workspace_id = ?
		ORDER BY name`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SkillSetSummary
	for rows.Next() {
		var ss SkillSetSummary
		var desc sql.NullString
		if err := rows.Scan(&ss.SkillSetID, &ss.WorkspaceID, &ss.Name, &desc, &ss.CreatedBy, &ss.CreatedAt); err != nil {
			return nil, err
		}
		if desc.Valid {
			ss.Description = &desc.String
		}
		out = append(out, ss)
	}
	return out, rows.Err()
}

func (s *SkillSetService) Create(ctx context.Context, workspaceID string, in CreateSkillSetInput) (*SkillSetSummary, error) {
	user := middleware.UserFromCtx(ctx)
	if user == nil {
		return nil, fmt.Errorf("unauthenticated")
	}
	if in.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO skill_sets (skill_set_id, workspace_id, name, description, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		id, workspaceID, in.Name, in.Description, user.UserID, now)
	if err != nil {
		return nil, err
	}
	return s.get(ctx, workspaceID, id)
}

func (s *SkillSetService) Update(ctx context.Context, workspaceID, skillSetID string, in UpdateSkillSetInput) (*SkillSetSummary, error) {
	_, err := s.db.ExecContext(ctx, `
		UPDATE skill_sets SET
			name        = COALESCE(?, name),
			description = COALESCE(?, description)
		WHERE skill_set_id = ? AND workspace_id = ?`,
		in.Name, in.Description, skillSetID, workspaceID)
	if err != nil {
		return nil, err
	}
	return s.get(ctx, workspaceID, skillSetID)
}

func (s *SkillSetService) Delete(ctx context.Context, workspaceID, skillSetID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM skill_sets WHERE skill_set_id = ? AND workspace_id = ?`,
		skillSetID, workspaceID)
	return err
}

func (s *SkillSetService) get(ctx context.Context, workspaceID, skillSetID string) (*SkillSetSummary, error) {
	var ss SkillSetSummary
	var desc sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT skill_set_id, workspace_id, name, description, created_by, created_at
		FROM skill_sets WHERE skill_set_id = ? AND workspace_id = ?`,
		skillSetID, workspaceID).Scan(
		&ss.SkillSetID, &ss.WorkspaceID, &ss.Name, &desc, &ss.CreatedBy, &ss.CreatedAt)
	if err != nil {
		return nil, err
	}
	if desc.Valid {
		ss.Description = &desc.String
	}
	return &ss, nil
}

// Skills within a skill set ─────────────────────────────────────────────────

func (s *SkillSetService) ListSkills(ctx context.Context, skillSetID string) ([]SkillSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT skill_id, skill_set_id, name, type, config_json, created_at
		FROM skills WHERE skill_set_id = ?
		ORDER BY name`, skillSetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SkillSummary
	for rows.Next() {
		var sk SkillSummary
		if err := rows.Scan(&sk.SkillID, &sk.SkillSetID, &sk.Name, &sk.Type, &sk.ConfigJSON, &sk.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, sk)
	}
	return out, rows.Err()
}

func (s *SkillSetService) CreateSkill(ctx context.Context, skillSetID string, in CreateSkillInput) (*SkillSummary, error) {
	if in.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	validTypes := map[string]bool{"tool_combo": true, "template": true, "custom": true}
	if !validTypes[in.Type] {
		return nil, fmt.Errorf("type must be tool_combo, template, or custom")
	}
	configJSON := in.ConfigJSON
	if configJSON == "" {
		configJSON = "{}"
	}

	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO skills (skill_id, skill_set_id, name, type, config_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		id, skillSetID, in.Name, in.Type, configJSON, now)
	if err != nil {
		return nil, err
	}
	var sk SkillSummary
	_ = s.db.QueryRowContext(ctx, `
		SELECT skill_id, skill_set_id, name, type, config_json, created_at
		FROM skills WHERE skill_id = ?`, id).Scan(
		&sk.SkillID, &sk.SkillSetID, &sk.Name, &sk.Type, &sk.ConfigJSON, &sk.CreatedAt)
	return &sk, nil
}

func (s *SkillSetService) DeleteSkill(ctx context.Context, skillID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM skills WHERE skill_id = ?`, skillID)
	return err
}

// ─────────────────────────────────────────────────────────────────────────────
// MCP Connectors
// ─────────────────────────────────────────────────────────────────────────────

type MCPConnectorSummary struct {
	ConnectorID string `json:"connector_id"`
	WorkspaceID string `json:"workspace_id"`
	Name        string `json:"name"`
	Transport   string `json:"transport"`
	Endpoint    string `json:"endpoint"`
	SecretRef   string `json:"secret_ref,omitempty"`
	ConfigJSON  string `json:"config_json"`
	Enabled     bool   `json:"enabled"`
	CreatedBy   string `json:"created_by"`
	CreatedAt   string `json:"created_at"`
}

type CreateMCPConnectorInput struct {
	Name      string `json:"name"`
	Transport string `json:"transport"`
	Endpoint  string `json:"endpoint"`
	SecretRef string `json:"secret_ref"`
}

type UpdateMCPConnectorInput struct {
	Name      *string `json:"name"`
	Transport *string `json:"transport"`
	Endpoint  *string `json:"endpoint"`
	SecretRef *string `json:"secret_ref"`
	Enabled   *bool   `json:"enabled"`
}

type MCPConnectorService struct {
	db *sql.DB
}

func NewMCPConnectorService(db *sql.DB) *MCPConnectorService {
	return &MCPConnectorService{db: db}
}

func (s *MCPConnectorService) List(ctx context.Context, workspaceID string) ([]MCPConnectorSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT connector_id, workspace_id, name, transport, endpoint, COALESCE(secret_ref,''),
		       config_json, enabled, created_by, created_at
		FROM mcp_connectors WHERE workspace_id = ?
		ORDER BY name`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MCPConnectorSummary
	for rows.Next() {
		var mc MCPConnectorSummary
		var enabled int
		if err := rows.Scan(
			&mc.ConnectorID, &mc.WorkspaceID, &mc.Name, &mc.Transport, &mc.Endpoint,
			&mc.SecretRef, &mc.ConfigJSON, &enabled, &mc.CreatedBy, &mc.CreatedAt,
		); err != nil {
			return nil, err
		}
		mc.Enabled = enabled == 1
		out = append(out, mc)
	}
	return out, rows.Err()
}

func (s *MCPConnectorService) Create(ctx context.Context, workspaceID string, in CreateMCPConnectorInput) (*MCPConnectorSummary, error) {
	user := middleware.UserFromCtx(ctx)
	if user == nil {
		return nil, fmt.Errorf("unauthenticated")
	}
	if in.Name == "" || in.Endpoint == "" {
		return nil, fmt.Errorf("name and endpoint are required")
	}
	validTransports := map[string]bool{"stdio": true, "sse": true, "streamable_http": true}
	if !validTransports[in.Transport] {
		return nil, fmt.Errorf("transport must be stdio, sse, or streamable_http")
	}

	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO mcp_connectors
			(connector_id, workspace_id, name, transport, endpoint, secret_ref, config_json, enabled, created_by, created_at)
		VALUES (?, ?, ?, ?, ?, ?, '{}', 1, ?, ?)`,
		id, workspaceID, in.Name, in.Transport, in.Endpoint, in.SecretRef, user.UserID, now)
	if err != nil {
		return nil, err
	}
	return s.get(ctx, workspaceID, id)
}

func (s *MCPConnectorService) Update(ctx context.Context, workspaceID, connectorID string, in UpdateMCPConnectorInput) (*MCPConnectorSummary, error) {
	enabledVal := sql.NullInt64{}
	if in.Enabled != nil {
		enabledVal = sql.NullInt64{Valid: true}
		if *in.Enabled {
			enabledVal.Int64 = 1
		}
	}
	_, err := s.db.ExecContext(ctx, `
		UPDATE mcp_connectors SET
			name       = COALESCE(?, name),
			transport  = COALESCE(?, transport),
			endpoint   = COALESCE(?, endpoint),
			secret_ref = COALESCE(?, secret_ref),
			enabled    = CASE WHEN ? THEN ? ELSE enabled END
		WHERE connector_id = ? AND workspace_id = ?`,
		in.Name, in.Transport, in.Endpoint, in.SecretRef,
		enabledVal.Valid, enabledVal.Int64,
		connectorID, workspaceID)
	if err != nil {
		return nil, err
	}
	return s.get(ctx, workspaceID, connectorID)
}

func (s *MCPConnectorService) Delete(ctx context.Context, workspaceID, connectorID string) error {
	_, err := s.db.ExecContext(ctx, `
		DELETE FROM mcp_connectors WHERE connector_id = ? AND workspace_id = ?`,
		connectorID, workspaceID)
	return err
}

func (s *MCPConnectorService) get(ctx context.Context, workspaceID, connectorID string) (*MCPConnectorSummary, error) {
	var mc MCPConnectorSummary
	var enabled int
	err := s.db.QueryRowContext(ctx, `
		SELECT connector_id, workspace_id, name, transport, endpoint, COALESCE(secret_ref,''),
		       config_json, enabled, created_by, created_at
		FROM mcp_connectors WHERE connector_id = ? AND workspace_id = ?`,
		connectorID, workspaceID).Scan(
		&mc.ConnectorID, &mc.WorkspaceID, &mc.Name, &mc.Transport, &mc.Endpoint,
		&mc.SecretRef, &mc.ConfigJSON, &enabled, &mc.CreatedBy, &mc.CreatedAt)
	if err != nil {
		return nil, err
	}
	mc.Enabled = enabled == 1
	return &mc, nil
}
