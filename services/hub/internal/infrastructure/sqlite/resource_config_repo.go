package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"goyais/services/hub/internal/domain"
)

type ResourceConfigRepository struct {
	db *sql.DB
}

func NewResourceConfigRepository(db *sql.DB) ResourceConfigRepository {
	return ResourceConfigRepository{db: db}
}

func (r ResourceConfigRepository) GetResourceConfig(ctx context.Context, workspaceID domain.WorkspaceID, configID string) (domain.ResourceConfig, bool, error) {
	if r.db == nil {
		return domain.ResourceConfig{}, false, nil
	}
	row := r.db.QueryRowContext(
		ctx,
		`SELECT payload_json
		   FROM resource_configs
		  WHERE workspace_id=? AND id=?`,
		strings.TrimSpace(string(workspaceID)),
		strings.TrimSpace(configID),
	)
	var payload string
	if err := row.Scan(&payload); err != nil {
		if err == sql.ErrNoRows {
			return domain.ResourceConfig{}, false, nil
		}
		return domain.ResourceConfig{}, false, err
	}
	item, err := decodeDomainResourceConfig(payload)
	if err != nil {
		return domain.ResourceConfig{}, false, err
	}
	if item.IsDeleted {
		return domain.ResourceConfig{}, false, nil
	}
	return item, true, nil
}

func (r ResourceConfigRepository) ListSessionResourceSnapshots(ctx context.Context, sessionID domain.SessionID) ([]domain.SessionResourceSnapshot, error) {
	if r.db == nil {
		return []domain.SessionResourceSnapshot{}, nil
	}
	rows, err := r.db.QueryContext(
		ctx,
		`SELECT session_id, resource_config_id, resource_type, resource_version, is_deprecated, fallback_resource_id, payload_json, snapshot_at
		   FROM session_resource_snapshots
		  WHERE session_id=?
		  ORDER BY snapshot_at ASC, resource_config_id ASC`,
		strings.TrimSpace(string(sessionID)),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]domain.SessionResourceSnapshot, 0)
	for rows.Next() {
		var (
			item               domain.SessionResourceSnapshot
			isDeprecated       int
			fallbackResourceID sql.NullString
			payloadJSON        string
		)
		if err := rows.Scan(
			&item.SessionID,
			&item.ResourceConfigID,
			&item.ResourceType,
			&item.ResourceVersion,
			&isDeprecated,
			&fallbackResourceID,
			&payloadJSON,
			&item.SnapshotAt,
		); err != nil {
			return nil, err
		}
		item.IsDeprecated = isDeprecated != 0
		if fallbackResourceID.Valid && strings.TrimSpace(fallbackResourceID.String) != "" {
			value := strings.TrimSpace(fallbackResourceID.String)
			item.FallbackResourceID = &value
		}
		capturedConfig, err := decodeDomainResourceConfig(payloadJSON)
		if err != nil {
			return nil, err
		}
		item.CapturedConfig = capturedConfig
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

type domainResourceConfigPayload struct {
	ID          string                 `json:"id"`
	WorkspaceID string                 `json:"workspace_id"`
	Type        domain.ResourceType    `json:"type"`
	Name        string                 `json:"name"`
	Enabled     bool                   `json:"enabled"`
	Version     int                    `json:"version"`
	IsDeleted   bool                   `json:"is_deleted"`
	DeletedAt   *string                `json:"deleted_at"`
	Model       *domainModelSpec       `json:"model"`
	Rule        *domainRuleSpec        `json:"rule"`
	Skill       *domainSkillSpec       `json:"skill"`
	MCP         *domainMCPConfig       `json:"mcp"`
	TokensInTotal  int                 `json:"tokens_in_total"`
	TokensOutTotal int                 `json:"tokens_out_total"`
	TokensTotal    int                 `json:"tokens_total"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
}

type domainModelSpec struct {
	Vendor         string              `json:"vendor"`
	ModelID        string              `json:"model_id"`
	BaseURL        string              `json:"base_url"`
	BaseURLKey     string              `json:"base_url_key"`
	APIKey         string              `json:"api_key"`
	APIKeyMasked   string              `json:"api_key_masked"`
	TokenThreshold *int                `json:"token_threshold"`
	Runtime        *domainModelRuntime `json:"runtime"`
	Params         map[string]any      `json:"params"`
}

type domainModelRuntime struct {
	RequestTimeoutMS *int `json:"request_timeout_ms"`
}

type domainRuleSpec struct {
	Content string `json:"content"`
}

type domainSkillSpec struct {
	Content string `json:"content"`
}

type domainMCPConfig struct {
	Transport       string            `json:"transport"`
	Endpoint        string            `json:"endpoint"`
	Command         string            `json:"command"`
	Env             map[string]string `json:"env"`
	Status          string            `json:"status"`
	Tools           []string          `json:"tools"`
	LastError       string            `json:"last_error"`
	LastConnectedAt string            `json:"last_connected_at"`
}

func decodeDomainResourceConfig(payload string) (domain.ResourceConfig, error) {
	raw := domainResourceConfigPayload{}
	if err := json.Unmarshal([]byte(payload), &raw); err != nil {
		return domain.ResourceConfig{}, err
	}
	item := domain.ResourceConfig{
		ID:          strings.TrimSpace(raw.ID),
		WorkspaceID: domain.WorkspaceID(strings.TrimSpace(raw.WorkspaceID)),
		Type:        domain.ResourceType(strings.TrimSpace(string(raw.Type))),
		Name:        strings.TrimSpace(raw.Name),
		Enabled:     raw.Enabled,
		Version:     raw.Version,
		IsDeleted:   raw.IsDeleted,
		DeletedAt:   cloneOptionalString(raw.DeletedAt),
		TokensInTotal:  raw.TokensInTotal,
		TokensOutTotal: raw.TokensOutTotal,
		TokensTotal:    raw.TokensTotal,
		CreatedAt:   strings.TrimSpace(raw.CreatedAt),
		UpdatedAt:   strings.TrimSpace(raw.UpdatedAt),
	}
	if raw.Model != nil {
		item.Model = &domain.ModelSpec{
			Vendor:         strings.TrimSpace(raw.Model.Vendor),
			ModelID:        strings.TrimSpace(raw.Model.ModelID),
			BaseURL:        strings.TrimSpace(raw.Model.BaseURL),
			BaseURLKey:     strings.TrimSpace(raw.Model.BaseURLKey),
			APIKey:         strings.TrimSpace(raw.Model.APIKey),
			APIKeyMasked:   strings.TrimSpace(raw.Model.APIKeyMasked),
			TokenThreshold: cloneIntPointer(raw.Model.TokenThreshold),
			Params:         cloneAnyMap(raw.Model.Params),
		}
		if raw.Model.Runtime != nil {
			item.Model.Runtime = &domain.ModelRuntimeSpec{
				RequestTimeoutMS: cloneIntPointer(raw.Model.Runtime.RequestTimeoutMS),
			}
		}
	}
	if raw.Rule != nil {
		item.Rule = &domain.RuleSpec{Content: raw.Rule.Content}
	}
	if raw.Skill != nil {
		item.Skill = &domain.SkillSpec{Content: raw.Skill.Content}
	}
	if raw.MCP != nil {
		item.MCP = &domain.MCPConfig{
			Transport:       strings.TrimSpace(raw.MCP.Transport),
			Endpoint:        strings.TrimSpace(raw.MCP.Endpoint),
			Command:         strings.TrimSpace(raw.MCP.Command),
			Env:             cloneStringMap(raw.MCP.Env),
			Status:          strings.TrimSpace(raw.MCP.Status),
			Tools:           append([]string{}, raw.MCP.Tools...),
			LastError:       strings.TrimSpace(raw.MCP.LastError),
			LastConnectedAt: strings.TrimSpace(raw.MCP.LastConnectedAt),
		}
	}
	return item, nil
}

func cloneOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	copyValue := trimmed
	return &copyValue
}

func cloneIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func cloneAnyMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
