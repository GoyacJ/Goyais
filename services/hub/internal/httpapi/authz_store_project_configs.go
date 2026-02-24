package httpapi

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"
)

func (s *authzStore) getProjectConfig(projectID string) (ProjectConfig, bool, error) {
	row := s.db.QueryRow(
		`SELECT project_id, model_ids_json, default_model_id, rule_ids_json, skill_ids_json, mcp_ids_json, updated_at
		 FROM project_configs
		 WHERE project_id=?`,
		strings.TrimSpace(projectID),
	)
	var (
		item           ProjectConfig
		modelIDsJSON   string
		defaultModelID sql.NullString
		ruleIDsJSON    string
		skillIDsJSON   string
		mcpIDsJSON     string
	)
	if err := row.Scan(
		&item.ProjectID,
		&modelIDsJSON,
		&defaultModelID,
		&ruleIDsJSON,
		&skillIDsJSON,
		&mcpIDsJSON,
		&item.UpdatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
			return ProjectConfig{}, false, nil
		}
		return ProjectConfig{}, false, err
	}
	modelIDs, err := decodeJSONStringArray(modelIDsJSON)
	if err != nil {
		return ProjectConfig{}, false, err
	}
	ruleIDs, err := decodeJSONStringArray(ruleIDsJSON)
	if err != nil {
		return ProjectConfig{}, false, err
	}
	skillIDs, err := decodeJSONStringArray(skillIDsJSON)
	if err != nil {
		return ProjectConfig{}, false, err
	}
	mcpIDs, err := decodeJSONStringArray(mcpIDsJSON)
	if err != nil {
		return ProjectConfig{}, false, err
	}

	item.ModelIDs = modelIDs
	item.RuleIDs = ruleIDs
	item.SkillIDs = skillIDs
	item.MCPIDs = mcpIDs
	item.DefaultModelID = nullStringToPointer(defaultModelID)
	item = normalizeProjectConfigForStorage(item)
	return item, true, nil
}

func (s *authzStore) upsertProjectConfig(workspaceID string, input ProjectConfig) (ProjectConfig, error) {
	config := normalizeProjectConfigForStorage(input)
	workspaceID = strings.TrimSpace(workspaceID)

	modelIDsJSON, err := encodeJSONStringArray(config.ModelIDs)
	if err != nil {
		return ProjectConfig{}, err
	}
	ruleIDsJSON, err := encodeJSONStringArray(config.RuleIDs)
	if err != nil {
		return ProjectConfig{}, err
	}
	skillIDsJSON, err := encodeJSONStringArray(config.SkillIDs)
	if err != nil {
		return ProjectConfig{}, err
	}
	mcpIDsJSON, err := encodeJSONStringArray(config.MCPIDs)
	if err != nil {
		return ProjectConfig{}, err
	}

	_, err = s.db.Exec(
		`INSERT INTO project_configs(project_id, workspace_id, model_ids_json, default_model_id, rule_ids_json, skill_ids_json, mcp_ids_json, updated_at)
		 VALUES(?,?,?,?,?,?,?,?)
		 ON CONFLICT(project_id) DO UPDATE SET
		   workspace_id=excluded.workspace_id,
		   model_ids_json=excluded.model_ids_json,
		   default_model_id=excluded.default_model_id,
		   rule_ids_json=excluded.rule_ids_json,
		   skill_ids_json=excluded.skill_ids_json,
		   mcp_ids_json=excluded.mcp_ids_json,
		   updated_at=excluded.updated_at`,
		config.ProjectID,
		workspaceID,
		modelIDsJSON,
		nullWhenEmpty(derefString(config.DefaultModelID)),
		ruleIDsJSON,
		skillIDsJSON,
		mcpIDsJSON,
		config.UpdatedAt,
	)
	if err != nil {
		return ProjectConfig{}, err
	}
	return config, nil
}

func (s *authzStore) listWorkspaceProjectConfigItems(workspaceID string) ([]workspaceProjectConfigItem, error) {
	rows, err := s.db.Query(
		`SELECT
			p.id,
			p.name,
			p.default_model_id,
			p.updated_at,
			c.model_ids_json,
			c.default_model_id,
			c.rule_ids_json,
			c.skill_ids_json,
			c.mcp_ids_json,
			c.updated_at
		FROM projects p
		LEFT JOIN project_configs c ON c.project_id = p.id
		WHERE p.workspace_id=?
		ORDER BY lower(p.name) ASC`,
		strings.TrimSpace(workspaceID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]workspaceProjectConfigItem, 0)
	for rows.Next() {
		var (
			projectID             string
			projectName           string
			projectDefaultModelID sql.NullString
			projectUpdatedAt      string
			modelIDsJSON          sql.NullString
			configDefaultModelID  sql.NullString
			ruleIDsJSON           sql.NullString
			skillIDsJSON          sql.NullString
			mcpIDsJSON            sql.NullString
			configUpdatedAt       sql.NullString
		)
		if err := rows.Scan(
			&projectID,
			&projectName,
			&projectDefaultModelID,
			&projectUpdatedAt,
			&modelIDsJSON,
			&configDefaultModelID,
			&ruleIDsJSON,
			&skillIDsJSON,
			&mcpIDsJSON,
			&configUpdatedAt,
		); err != nil {
			return nil, err
		}

		config := ProjectConfig{
			ProjectID: projectID,
			ModelIDs:  []string{},
			RuleIDs:   []string{},
			SkillIDs:  []string{},
			MCPIDs:    []string{},
			UpdatedAt: strings.TrimSpace(projectUpdatedAt),
		}
		if strings.TrimSpace(config.UpdatedAt) == "" {
			config.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		}

		if modelIDsJSON.Valid {
			modelIDs, decodeErr := decodeJSONStringArray(modelIDsJSON.String)
			if decodeErr != nil {
				return nil, decodeErr
			}
			config.ModelIDs = modelIDs
			config.DefaultModelID = nullStringToPointer(configDefaultModelID)
			if ruleIDsJSON.Valid {
				ruleIDs, decodeErr := decodeJSONStringArray(ruleIDsJSON.String)
				if decodeErr != nil {
					return nil, decodeErr
				}
				config.RuleIDs = ruleIDs
			}
			if skillIDsJSON.Valid {
				skillIDs, decodeErr := decodeJSONStringArray(skillIDsJSON.String)
				if decodeErr != nil {
					return nil, decodeErr
				}
				config.SkillIDs = skillIDs
			}
			if mcpIDsJSON.Valid {
				mcpIDs, decodeErr := decodeJSONStringArray(mcpIDsJSON.String)
				if decodeErr != nil {
					return nil, decodeErr
				}
				config.MCPIDs = mcpIDs
			}
			if configUpdatedAt.Valid && strings.TrimSpace(configUpdatedAt.String) != "" {
				config.UpdatedAt = strings.TrimSpace(configUpdatedAt.String)
			}
		} else {
			defaultModel := strings.TrimSpace(projectDefaultModelID.String)
			if defaultModel != "" {
				config.ModelIDs = []string{defaultModel}
				config.DefaultModelID = &defaultModel
			}
		}

		config = normalizeProjectConfigForStorage(config)
		items = append(items, workspaceProjectConfigItem{
			ProjectID:   projectID,
			ProjectName: projectName,
			Config:      config,
		})
	}
	return items, rows.Err()
}

func normalizeProjectConfigForStorage(input ProjectConfig) ProjectConfig {
	now := time.Now().UTC().Format(time.RFC3339)
	item := input
	item.ProjectID = strings.TrimSpace(item.ProjectID)
	item.ModelIDs = sanitizeIDList(item.ModelIDs)
	item.RuleIDs = sanitizeIDList(item.RuleIDs)
	item.SkillIDs = sanitizeIDList(item.SkillIDs)
	item.MCPIDs = sanitizeIDList(item.MCPIDs)
	if item.DefaultModelID != nil {
		value := strings.TrimSpace(*item.DefaultModelID)
		if value == "" {
			item.DefaultModelID = nil
		} else {
			item.DefaultModelID = &value
		}
	}
	if item.DefaultModelID != nil && !containsTrimmed(item.ModelIDs, *item.DefaultModelID) {
		item.ModelIDs = append(item.ModelIDs, *item.DefaultModelID)
	}
	if strings.TrimSpace(item.UpdatedAt) == "" {
		item.UpdatedAt = now
	}
	return item
}

func sanitizeIDList(items []string) []string {
	output := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		output = append(output, value)
	}
	return output
}

func containsTrimmed(items []string, target string) bool {
	normalizedTarget := strings.TrimSpace(target)
	for _, item := range items {
		if strings.TrimSpace(item) == normalizedTarget {
			return true
		}
	}
	return false
}

func encodeJSONStringArray(items []string) (string, error) {
	encoded, err := json.Marshal(sanitizeIDList(items))
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func decodeJSONStringArray(raw string) ([]string, error) {
	source := strings.TrimSpace(raw)
	if source == "" {
		return []string{}, nil
	}
	output := []string{}
	if err := json.Unmarshal([]byte(source), &output); err != nil {
		return nil, err
	}
	return sanitizeIDList(output), nil
}

func nullStringToPointer(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	trimmed := strings.TrimSpace(value.String)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
