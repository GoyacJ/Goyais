package httpapi

import (
	"database/sql"
	"sort"
	"strings"
	"time"
)

type projectResourceBinding struct {
	ProjectID        string
	ResourceConfigID string
	ResourceType     ResourceType
	BindingIndex     int
	IsDefault        bool
	CreatedAt        string
	UpdatedAt        string
}

func (s *authzStore) listProjectResourceBindings(projectID string) ([]projectResourceBinding, error) {
	rows, err := s.db.Query(
		`SELECT project_id, resource_config_id, resource_type, binding_index, is_default, created_at, updated_at
		 FROM project_resource_bindings
		 WHERE project_id=?
		 ORDER BY resource_type ASC, binding_index ASC, resource_config_id ASC`,
		strings.TrimSpace(projectID),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]projectResourceBinding, 0)
	for rows.Next() {
		var (
			item         projectResourceBinding
			isDefaultInt int
		)
		if err := rows.Scan(
			&item.ProjectID,
			&item.ResourceConfigID,
			&item.ResourceType,
			&item.BindingIndex,
			&isDefaultInt,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		item.IsDefault = parseBoolInt(isDefaultInt)
		items = append(items, item)
	}
	return items, rows.Err()
}

func replaceProjectResourceBindingsTx(tx *sql.Tx, projectID string, bindings []projectResourceBinding) error {
	normalizedProjectID := strings.TrimSpace(projectID)
	if _, err := tx.Exec(`DELETE FROM project_resource_bindings WHERE project_id=?`, normalizedProjectID); err != nil {
		return err
	}
	for _, binding := range bindings {
		if _, err := tx.Exec(
			`INSERT INTO project_resource_bindings(
				project_id,
				resource_config_id,
				resource_type,
				binding_index,
				is_default,
				created_at,
				updated_at
			) VALUES(?,?,?,?,?,?,?)`,
			normalizedProjectID,
			strings.TrimSpace(binding.ResourceConfigID),
			string(binding.ResourceType),
			binding.BindingIndex,
			boolToInt(binding.IsDefault),
			binding.CreatedAt,
			binding.UpdatedAt,
		); err != nil {
			return err
		}
	}
	return nil
}

func buildProjectResourceBindings(input ProjectConfig) []projectResourceBinding {
	config := normalizeProjectConfigForStorage(input)
	now := strings.TrimSpace(config.UpdatedAt)
	if now == "" {
		now = time.Now().UTC().Format(time.RFC3339)
	}

	bindings := make([]projectResourceBinding, 0, len(config.ModelConfigIDs)+len(config.RuleIDs)+len(config.SkillIDs)+len(config.MCPIDs))
	appendBindings := func(resourceType ResourceType, ids []string, defaultResourceID string) {
		for index, resourceConfigID := range sanitizeIDList(ids) {
			bindings = append(bindings, projectResourceBinding{
				ProjectID:        config.ProjectID,
				ResourceConfigID: resourceConfigID,
				ResourceType:     resourceType,
				BindingIndex:     index,
				IsDefault:        resourceType == ResourceTypeModel && resourceConfigID == defaultResourceID,
				CreatedAt:        now,
				UpdatedAt:        now,
			})
		}
	}

	defaultModelConfigID := strings.TrimSpace(derefString(config.DefaultModelConfigID))
	appendBindings(ResourceTypeModel, config.ModelConfigIDs, defaultModelConfigID)
	appendBindings(ResourceTypeRule, config.RuleIDs, "")
	appendBindings(ResourceTypeSkill, config.SkillIDs, "")
	appendBindings(ResourceTypeMCP, config.MCPIDs, "")
	return bindings
}

func applyProjectResourceBindings(input ProjectConfig, bindings []projectResourceBinding) ProjectConfig {
	if len(bindings) == 0 {
		return normalizeProjectConfigForStorage(input)
	}

	config := input
	config.ModelConfigIDs = []string{}
	config.RuleIDs = []string{}
	config.SkillIDs = []string{}
	config.MCPIDs = []string{}

	sort.SliceStable(bindings, func(i, j int) bool {
		if bindings[i].ResourceType == bindings[j].ResourceType {
			if bindings[i].BindingIndex == bindings[j].BindingIndex {
				return bindings[i].ResourceConfigID < bindings[j].ResourceConfigID
			}
			return bindings[i].BindingIndex < bindings[j].BindingIndex
		}
		return string(bindings[i].ResourceType) < string(bindings[j].ResourceType)
	})

	defaultModelConfigID := ""
	for _, binding := range bindings {
		resourceConfigID := strings.TrimSpace(binding.ResourceConfigID)
		if resourceConfigID == "" {
			continue
		}
		switch binding.ResourceType {
		case ResourceTypeModel:
			config.ModelConfigIDs = append(config.ModelConfigIDs, resourceConfigID)
			if binding.IsDefault && defaultModelConfigID == "" {
				defaultModelConfigID = resourceConfigID
			}
		case ResourceTypeRule:
			config.RuleIDs = append(config.RuleIDs, resourceConfigID)
		case ResourceTypeSkill:
			config.SkillIDs = append(config.SkillIDs, resourceConfigID)
		case ResourceTypeMCP:
			config.MCPIDs = append(config.MCPIDs, resourceConfigID)
		}
	}

	switch {
	case strings.TrimSpace(defaultModelConfigID) != "":
		config.DefaultModelConfigID = toStringPtr(defaultModelConfigID)
	case containsTrimmed(config.ModelConfigIDs, derefString(config.DefaultModelConfigID)):
	default:
		if len(config.ModelConfigIDs) > 0 {
			config.DefaultModelConfigID = toStringPtr(config.ModelConfigIDs[0])
		} else {
			config.DefaultModelConfigID = nil
		}
	}

	return normalizeProjectConfigForStorage(config)
}
