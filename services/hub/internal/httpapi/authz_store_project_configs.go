package httpapi

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"
)

func (s *authzStore) getProjectConfig(projectID string) (ProjectConfig, bool, error) {
	row := s.db.QueryRow(
		`SELECT project_id, model_config_ids_json, default_model_config_id, token_threshold, model_token_thresholds_json, rule_ids_json, skill_ids_json, mcp_ids_json, updated_at
		 FROM project_configs
		 WHERE project_id=?`,
		strings.TrimSpace(projectID),
	)
	var (
		item                 ProjectConfig
		modelConfigIDsJSON   string
		defaultModelConfigID sql.NullString
		tokenThreshold       sql.NullInt64
		modelThresholdsJSON  sql.NullString
		ruleIDsJSON          string
		skillIDsJSON         string
		mcpIDsJSON           string
	)
	if err := row.Scan(
		&item.ProjectID,
		&modelConfigIDsJSON,
		&defaultModelConfigID,
		&tokenThreshold,
		&modelThresholdsJSON,
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
	modelConfigIDs, err := decodeJSONStringArray(modelConfigIDsJSON)
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
	modelTokenThresholds, err := decodeJSONIntMap(modelThresholdsJSON.String)
	if err != nil {
		return ProjectConfig{}, false, err
	}

	item.ModelConfigIDs = modelConfigIDs
	item.RuleIDs = ruleIDs
	item.SkillIDs = skillIDs
	item.MCPIDs = mcpIDs
	item.DefaultModelConfigID = nullStringToPointer(defaultModelConfigID)
	item.TokenThreshold = nullInt64ToPositiveIntPointer(tokenThreshold)
	item.ModelTokenThresholds = modelTokenThresholds
	item = normalizeProjectConfigForStorage(item)
	return item, true, nil
}

func (s *authzStore) upsertProjectConfig(workspaceID string, input ProjectConfig) (ProjectConfig, error) {
	config := normalizeProjectConfigForStorage(input)
	workspaceID = strings.TrimSpace(workspaceID)

	modelConfigIDsJSON, err := encodeJSONStringArray(config.ModelConfigIDs)
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
	modelTokenThresholdsJSON, err := encodeJSONIntMap(config.ModelTokenThresholds)
	if err != nil {
		return ProjectConfig{}, err
	}

	_, err = s.db.Exec(
		`INSERT INTO project_configs(project_id, workspace_id, model_config_ids_json, default_model_config_id, token_threshold, model_token_thresholds_json, rule_ids_json, skill_ids_json, mcp_ids_json, updated_at)
		 VALUES(?,?,?,?,?,?,?,?,?,?)
		 ON CONFLICT(project_id) DO UPDATE SET
		   workspace_id=excluded.workspace_id,
		   model_config_ids_json=excluded.model_config_ids_json,
		   default_model_config_id=excluded.default_model_config_id,
		   token_threshold=excluded.token_threshold,
		   model_token_thresholds_json=excluded.model_token_thresholds_json,
		   rule_ids_json=excluded.rule_ids_json,
		   skill_ids_json=excluded.skill_ids_json,
		   mcp_ids_json=excluded.mcp_ids_json,
		   updated_at=excluded.updated_at`,
		config.ProjectID,
		workspaceID,
		modelConfigIDsJSON,
		nullWhenEmpty(derefString(config.DefaultModelConfigID)),
		config.TokenThreshold,
		modelTokenThresholdsJSON,
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
			p.default_model_config_id,
			p.updated_at,
			c.model_config_ids_json,
			c.default_model_config_id,
			c.token_threshold,
			c.model_token_thresholds_json,
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
			projectID                   string
			projectName                 string
			projectDefaultModelConfigID sql.NullString
			projectUpdatedAt            string
			modelConfigIDsJSON          sql.NullString
			configDefaultModelConfigID  sql.NullString
			configTokenThreshold        sql.NullInt64
			configModelThresholdsJSON   sql.NullString
			ruleIDsJSON                 sql.NullString
			skillIDsJSON                sql.NullString
			mcpIDsJSON                  sql.NullString
			configUpdatedAt             sql.NullString
		)
		if err := rows.Scan(
			&projectID,
			&projectName,
			&projectDefaultModelConfigID,
			&projectUpdatedAt,
			&modelConfigIDsJSON,
			&configDefaultModelConfigID,
			&configTokenThreshold,
			&configModelThresholdsJSON,
			&ruleIDsJSON,
			&skillIDsJSON,
			&mcpIDsJSON,
			&configUpdatedAt,
		); err != nil {
			return nil, err
		}

		config := ProjectConfig{
			ProjectID:            projectID,
			ModelConfigIDs:       []string{},
			ModelTokenThresholds: map[string]int{},
			RuleIDs:              []string{},
			SkillIDs:             []string{},
			MCPIDs:               []string{},
			UpdatedAt:            strings.TrimSpace(projectUpdatedAt),
		}
		if strings.TrimSpace(config.UpdatedAt) == "" {
			config.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		}

		if modelConfigIDsJSON.Valid {
			modelConfigIDs, decodeErr := decodeJSONStringArray(modelConfigIDsJSON.String)
			if decodeErr != nil {
				return nil, decodeErr
			}
			config.ModelConfigIDs = modelConfigIDs
			config.DefaultModelConfigID = nullStringToPointer(configDefaultModelConfigID)
			config.TokenThreshold = nullInt64ToPositiveIntPointer(configTokenThreshold)
			if configModelThresholdsJSON.Valid {
				modelThresholds, decodeErr := decodeJSONIntMap(configModelThresholdsJSON.String)
				if decodeErr != nil {
					return nil, decodeErr
				}
				config.ModelTokenThresholds = modelThresholds
			}
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
			defaultModelConfigID := strings.TrimSpace(projectDefaultModelConfigID.String)
			if defaultModelConfigID != "" {
				config.ModelConfigIDs = []string{defaultModelConfigID}
				config.DefaultModelConfigID = &defaultModelConfigID
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
	item.ModelConfigIDs = sanitizeIDList(item.ModelConfigIDs)
	item.TokenThreshold = normalizeOptionalPositiveThreshold(item.TokenThreshold)
	item.ModelTokenThresholds = sanitizeModelTokenThresholds(item.ModelTokenThresholds, item.ModelConfigIDs)
	item.RuleIDs = sanitizeIDList(item.RuleIDs)
	item.SkillIDs = sanitizeIDList(item.SkillIDs)
	item.MCPIDs = sanitizeIDList(item.MCPIDs)
	if item.DefaultModelConfigID != nil {
		value := strings.TrimSpace(*item.DefaultModelConfigID)
		if value == "" {
			item.DefaultModelConfigID = nil
		} else {
			item.DefaultModelConfigID = &value
		}
	}
	if item.DefaultModelConfigID != nil && !containsTrimmed(item.ModelConfigIDs, *item.DefaultModelConfigID) {
		item.ModelConfigIDs = append(item.ModelConfigIDs, *item.DefaultModelConfigID)
	}
	item.ModelTokenThresholds = sanitizeModelTokenThresholds(item.ModelTokenThresholds, item.ModelConfigIDs)
	if strings.TrimSpace(item.UpdatedAt) == "" {
		item.UpdatedAt = now
	}
	return item
}

func sanitizeModelTokenThresholds(input map[string]int, allowedModelConfigIDs []string) map[string]int {
	allowed := map[string]struct{}{}
	for _, id := range sanitizeIDList(allowedModelConfigIDs) {
		allowed[id] = struct{}{}
	}
	output := map[string]int{}
	for key, value := range input {
		modelConfigID := strings.TrimSpace(key)
		if modelConfigID == "" || value <= 0 {
			continue
		}
		if _, ok := allowed[modelConfigID]; !ok {
			continue
		}
		output[modelConfigID] = value
	}
	return output
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

func encodeJSONIntMap(input map[string]int) (string, error) {
	encoded, err := json.Marshal(input)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func decodeJSONIntMap(raw string) (map[string]int, error) {
	source := strings.TrimSpace(raw)
	if source == "" {
		return map[string]int{}, nil
	}
	output := map[string]int{}
	if err := json.Unmarshal([]byte(source), &output); err != nil {
		return nil, err
	}
	normalized := map[string]int{}
	for key, value := range output {
		normalizedKey := strings.TrimSpace(key)
		if normalizedKey == "" || value <= 0 {
			continue
		}
		normalized[normalizedKey] = value
	}
	return normalized, nil
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

func nullInt64ToPositiveIntPointer(value sql.NullInt64) *int {
	if !value.Valid || value.Int64 <= 0 {
		return nil
	}
	converted := int(value.Int64)
	return &converted
}
