package httpapi

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"
)

type resourceConfigQuery struct {
	Type    ResourceType
	Query   string
	Enabled *bool
}

func (s *authzStore) upsertCatalogRoot(input CatalogRootResponse) error {
	_, err := s.db.Exec(
		`INSERT INTO workspace_catalog_roots(workspace_id, catalog_root, updated_at)
		 VALUES(?,?,?)
		 ON CONFLICT(workspace_id) DO UPDATE SET catalog_root=excluded.catalog_root, updated_at=excluded.updated_at`,
		strings.TrimSpace(input.WorkspaceID),
		strings.TrimSpace(input.CatalogRoot),
		strings.TrimSpace(input.UpdatedAt),
	)
	return err
}

func (s *authzStore) getCatalogRoot(workspaceID string) (CatalogRootResponse, bool, error) {
	row := s.db.QueryRow(
		`SELECT workspace_id, catalog_root, updated_at FROM workspace_catalog_roots WHERE workspace_id=?`,
		strings.TrimSpace(workspaceID),
	)
	item := CatalogRootResponse{}
	if err := row.Scan(&item.WorkspaceID, &item.CatalogRoot, &item.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return CatalogRootResponse{}, false, nil
		}
		return CatalogRootResponse{}, false, err
	}
	return item, true, nil
}

func (s *authzStore) upsertResourceConfig(input ResourceConfig) (ResourceConfig, error) {
	existing, exists, err := s.getResourceConfigWithMode(input.WorkspaceID, input.ID, false)
	if err != nil {
		return ResourceConfig{}, err
	}
	if !exists {
		input.Version = normalizeResourceConfigVersion(input.Version)
		input.IsDeleted = false
		if input.DeletedAt != nil && strings.TrimSpace(*input.DeletedAt) == "" {
			input.DeletedAt = nil
		}
	} else {
		nextVersion := existing.Version + 1
		if input.Version > nextVersion {
			nextVersion = input.Version
		}
		input.Version = normalizeResourceConfigVersion(nextVersion)
		if input.CreatedAt == "" {
			input.CreatedAt = existing.CreatedAt
		}
	}
	encoded, err := encodeResourceConfigPayload(input)
	if err != nil {
		return ResourceConfig{}, err
	}
	_, err = s.db.Exec(
		`INSERT INTO resource_configs(id, workspace_id, type, enabled, payload_json, created_at, updated_at)
		 VALUES(?,?,?,?,?,?,?)
		 ON CONFLICT(id) DO UPDATE SET type=excluded.type, enabled=excluded.enabled, payload_json=excluded.payload_json, updated_at=excluded.updated_at`,
		input.ID,
		input.WorkspaceID,
		string(input.Type),
		boolToInt(input.Enabled),
		encoded,
		input.CreatedAt,
		input.UpdatedAt,
	)
	if err != nil {
		return ResourceConfig{}, err
	}
	return s.getResourceConfig(input.WorkspaceID, input.ID)
}

func (s *authzStore) listResourceConfigs(workspaceID string, query resourceConfigQuery) ([]ResourceConfig, error) {
	clauses := []string{"workspace_id = ?"}
	args := []any{strings.TrimSpace(workspaceID)}
	if query.Type != "" {
		clauses = append(clauses, "type = ?")
		args = append(args, string(query.Type))
	}
	if query.Enabled != nil {
		clauses = append(clauses, "enabled = ?")
		args = append(args, boolToInt(*query.Enabled))
	}
	stmt := `SELECT payload_json FROM resource_configs WHERE ` + strings.Join(clauses, " AND ") + ` ORDER BY created_at ASC`
	rows, err := s.db.Query(stmt, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]ResourceConfig, 0)
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		item, err := decodeResourceConfigPayload(payload, true)
		if err != nil {
			return nil, err
		}
		if item.IsDeleted {
			continue
		}
		if query.Query != "" && !matchesResourceConfigQuery(item, query.Query) {
			continue
		}
		normalizeResourceConfigForStorage(&item)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *authzStore) getResourceConfig(workspaceID string, configID string) (ResourceConfig, error) {
	raw, _, err := s.getResourceConfigWithMode(workspaceID, configID, true)
	return raw, err
}

func (s *authzStore) getResourceConfigRaw(workspaceID string, configID string) (ResourceConfig, bool, error) {
	return s.getResourceConfigWithMode(workspaceID, configID, false)
}

func (s *authzStore) getResourceConfigWithMode(workspaceID string, configID string, redactSecret bool) (ResourceConfig, bool, error) {
	row := s.db.QueryRow(
		`SELECT payload_json FROM resource_configs WHERE workspace_id=? AND id=?`,
		strings.TrimSpace(workspaceID),
		strings.TrimSpace(configID),
	)
	var payload string
	if err := row.Scan(&payload); err != nil {
		if err == sql.ErrNoRows {
			return ResourceConfig{}, false, nil
		}
		return ResourceConfig{}, false, err
	}
	item, err := decodeResourceConfigPayload(payload, redactSecret)
	if err != nil {
		return ResourceConfig{}, false, err
	}
	if item.IsDeleted {
		return ResourceConfig{}, false, nil
	}
	normalizeResourceConfigForStorage(&item)
	return item, true, nil
}

func (s *authzStore) deleteResourceConfig(workspaceID string, configID string) error {
	row := s.db.QueryRow(
		`SELECT payload_json FROM resource_configs WHERE workspace_id=? AND id=?`,
		strings.TrimSpace(workspaceID),
		strings.TrimSpace(configID),
	)
	var payload string
	if err := row.Scan(&payload); err != nil {
		if err == sql.ErrNoRows {
			return sql.ErrNoRows
		}
		return err
	}
	item, err := decodeResourceConfigPayload(payload, false)
	if err != nil {
		return err
	}
	if item.IsDeleted {
		return sql.ErrNoRows
	}
	now := time.Now().UTC().Format(time.RFC3339)
	item.IsDeleted = true
	item.Enabled = false
	item.DeletedAt = &now
	item.UpdatedAt = now
	item.Version = normalizeResourceConfigVersion(item.Version + 1)
	encoded, err := encodeResourceConfigPayload(item)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`UPDATE resource_configs SET enabled=?, payload_json=?, updated_at=? WHERE workspace_id=? AND id=?`,
		boolToInt(item.Enabled),
		encoded,
		item.UpdatedAt,
		strings.TrimSpace(workspaceID),
		strings.TrimSpace(configID),
	)
	return err
}

func (s *authzStore) appendResourceTestLog(item ResourceTestLog) error {
	if strings.TrimSpace(item.ID) == "" {
		item.ID = "rt_" + randomHex(6)
	}
	if strings.TrimSpace(item.CreatedAt) == "" {
		item.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	_, err := s.db.Exec(
		`INSERT INTO resource_test_logs(id, workspace_id, config_id, test_type, result, latency_ms, error_code, details_json, created_at)
		 VALUES(?,?,?,?,?,?,?,?,?)`,
		item.ID,
		item.WorkspaceID,
		item.ConfigID,
		item.TestType,
		item.Result,
		item.LatencyMS,
		item.ErrorCode,
		item.Details,
		item.CreatedAt,
	)
	return err
}

func encodeResourceConfigPayload(input ResourceConfig) (string, error) {
	safe := input
	safe.Version = normalizeResourceConfigVersion(safe.Version)
	if safe.DeletedAt != nil && strings.TrimSpace(*safe.DeletedAt) == "" {
		safe.DeletedAt = nil
	}
	if safe.Model != nil {
		model := *safe.Model
		if strings.TrimSpace(model.APIKey) != "" {
			encrypted, err := encryptSecret(model.APIKey)
			if err != nil {
				return "", err
			}
			model.APIKey = encrypted
		}
		model.APIKeyMasked = ""
		safe.Model = &model
	}

	encoded, err := json.Marshal(safe)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func decodeResourceConfigPayload(payload string, redactSecret bool) (ResourceConfig, error) {
	item := ResourceConfig{}
	if err := json.Unmarshal([]byte(payload), &item); err != nil {
		return ResourceConfig{}, err
	}
	legacy := struct {
		Model *struct {
			TimeoutMS *int `json:"timeout_ms"`
		} `json:"model"`
	}{}
	if err := json.Unmarshal([]byte(payload), &legacy); err != nil {
		return ResourceConfig{}, err
	}
	item.Version = normalizeResourceConfigVersion(item.Version)
	if item.Model != nil {
		model := *item.Model
		if (model.Runtime == nil || model.Runtime.RequestTimeoutMS == nil) && legacy.Model != nil && legacy.Model.TimeoutMS != nil {
			value := *legacy.Model.TimeoutMS
			model.Runtime = &ModelRuntimeSpec{RequestTimeoutMS: &value}
		}
		if strings.TrimSpace(model.APIKey) != "" {
			secret, err := decryptSecret(model.APIKey)
			if err != nil {
				return ResourceConfig{}, err
			}
			if redactSecret {
				model.APIKeyMasked = maskSecret(secret)
				model.APIKey = ""
			} else {
				model.APIKeyMasked = maskSecret(secret)
				model.APIKey = secret
			}
		}
		item.Model = &model
	}
	return item, nil
}

func normalizeResourceConfigVersion(version int) int {
	if version <= 0 {
		return 1
	}
	return version
}
