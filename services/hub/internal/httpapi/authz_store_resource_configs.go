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
	normalizeResourceConfigForStorage(&item)
	return item, true, nil
}

func (s *authzStore) deleteResourceConfig(workspaceID string, configID string) error {
	result, err := s.db.Exec(
		`DELETE FROM resource_configs WHERE workspace_id=? AND id=?`,
		strings.TrimSpace(workspaceID),
		strings.TrimSpace(configID),
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
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
	if item.Model != nil {
		model := *item.Model
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
