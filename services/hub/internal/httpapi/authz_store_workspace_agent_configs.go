package httpapi

import (
	"database/sql"
	"encoding/json"
	"strings"
)

func (s *authzStore) getWorkspaceAgentConfig(workspaceID string) (WorkspaceAgentConfig, bool, error) {
	normalizedWorkspaceID := strings.TrimSpace(workspaceID)
	if normalizedWorkspaceID == "" {
		return WorkspaceAgentConfig{}, false, nil
	}

	var (
		configJSON string
		updatedAt  string
	)
	err := s.db.QueryRow(
		`SELECT config_json, updated_at FROM workspace_agent_configs WHERE workspace_id=?`,
		normalizedWorkspaceID,
	).Scan(&configJSON, &updatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return WorkspaceAgentConfig{}, false, nil
		}
		return WorkspaceAgentConfig{}, false, err
	}

	item := WorkspaceAgentConfig{}
	if strings.TrimSpace(configJSON) != "" {
		if unmarshalErr := json.Unmarshal([]byte(configJSON), &item); unmarshalErr != nil {
			return WorkspaceAgentConfig{}, false, unmarshalErr
		}
	}
	item.WorkspaceID = normalizedWorkspaceID
	item.UpdatedAt = strings.TrimSpace(updatedAt)
	normalized := normalizeWorkspaceAgentConfig(normalizedWorkspaceID, item, item.UpdatedAt)
	return normalized, true, nil
}

func (s *authzStore) upsertWorkspaceAgentConfig(workspaceID string, input WorkspaceAgentConfig) (WorkspaceAgentConfig, error) {
	normalizedWorkspaceID := strings.TrimSpace(workspaceID)
	if normalizedWorkspaceID == "" {
		return WorkspaceAgentConfig{}, nil
	}

	normalized := normalizeWorkspaceAgentConfig(normalizedWorkspaceID, input, nowUTC())
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return WorkspaceAgentConfig{}, err
	}
	_, err = s.db.Exec(
		`INSERT INTO workspace_agent_configs(workspace_id, config_json, updated_at)
		 VALUES(?,?,?)
		 ON CONFLICT(workspace_id) DO UPDATE SET
		   config_json=excluded.config_json,
		   updated_at=excluded.updated_at`,
		normalized.WorkspaceID,
		string(encoded),
		normalized.UpdatedAt,
	)
	if err != nil {
		return WorkspaceAgentConfig{}, err
	}
	return normalized, nil
}

func (s *authzStore) ensureWorkspaceAgentConfig(workspaceID string) (WorkspaceAgentConfig, error) {
	existing, exists, err := s.getWorkspaceAgentConfig(workspaceID)
	if err != nil {
		return WorkspaceAgentConfig{}, err
	}
	if exists {
		return existing, nil
	}
	defaultConfig := defaultWorkspaceAgentConfig(strings.TrimSpace(workspaceID), nowUTC())
	return s.upsertWorkspaceAgentConfig(defaultConfig.WorkspaceID, defaultConfig)
}
