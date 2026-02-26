package httpapi

import (
	"fmt"
	"strings"
)

func validateProjectConfigResourceReferences(state *AppState, workspaceID string, config ProjectConfig) error {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return fmt.Errorf("workspace_id is required")
	}

	for _, modelConfigID := range sanitizeIDList(config.ModelConfigIDs) {
		if err := validateWorkspaceModelConfigReference(state, workspaceID, modelConfigID); err != nil {
			return err
		}
	}
	for _, ruleID := range sanitizeIDList(config.RuleIDs) {
		if err := validateWorkspaceResourceReference(state, workspaceID, ruleID, ResourceTypeRule); err != nil {
			return err
		}
	}
	for _, skillID := range sanitizeIDList(config.SkillIDs) {
		if err := validateWorkspaceResourceReference(state, workspaceID, skillID, ResourceTypeSkill); err != nil {
			return err
		}
	}
	for _, mcpID := range sanitizeIDList(config.MCPIDs) {
		if err := validateWorkspaceResourceReference(state, workspaceID, mcpID, ResourceTypeMCP); err != nil {
			return err
		}
	}
	if config.DefaultModelConfigID != nil {
		defaultModelConfigID := strings.TrimSpace(*config.DefaultModelConfigID)
		if defaultModelConfigID != "" {
			if !containsString(config.ModelConfigIDs, defaultModelConfigID) {
				return fmt.Errorf("default_model_config_id must be included in model_config_ids")
			}
			if err := validateWorkspaceModelConfigReference(state, workspaceID, defaultModelConfigID); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateConversationResourceSelection(
	state *AppState,
	workspaceID string,
	projectConfig ProjectConfig,
	modelConfigID string,
	ruleIDs []string,
	skillIDs []string,
	mcpIDs []string,
) error {
	modelConfigID = strings.TrimSpace(modelConfigID)
	if modelConfigID == "" {
		return fmt.Errorf("model_config_id cannot be empty")
	}
	if !containsString(projectConfig.ModelConfigIDs, modelConfigID) {
		return fmt.Errorf("model_config_id must be included in project model_config_ids")
	}
	if err := validateWorkspaceModelConfigReference(state, workspaceID, modelConfigID); err != nil {
		return err
	}

	for _, ruleID := range sanitizeIDList(ruleIDs) {
		if !containsString(projectConfig.RuleIDs, ruleID) {
			return fmt.Errorf("rule_id %s is not allowed by project config", ruleID)
		}
		if err := validateWorkspaceResourceReference(state, workspaceID, ruleID, ResourceTypeRule); err != nil {
			return err
		}
	}
	for _, skillID := range sanitizeIDList(skillIDs) {
		if !containsString(projectConfig.SkillIDs, skillID) {
			return fmt.Errorf("skill_id %s is not allowed by project config", skillID)
		}
		if err := validateWorkspaceResourceReference(state, workspaceID, skillID, ResourceTypeSkill); err != nil {
			return err
		}
	}
	for _, mcpID := range sanitizeIDList(mcpIDs) {
		if !containsString(projectConfig.MCPIDs, mcpID) {
			return fmt.Errorf("mcp_id %s is not allowed by project config", mcpID)
		}
		if err := validateWorkspaceResourceReference(state, workspaceID, mcpID, ResourceTypeMCP); err != nil {
			return err
		}
	}
	return nil
}

func validateWorkspaceResourceReference(state *AppState, workspaceID string, configID string, expectedType ResourceType) error {
	item, exists, err := loadWorkspaceResourceConfigRaw(state, strings.TrimSpace(workspaceID), strings.TrimSpace(configID))
	if err != nil {
		return fmt.Errorf("failed to load resource config %s: %w", configID, err)
	}
	if !exists {
		return fmt.Errorf("resource config %s does not exist", configID)
	}
	if item.Type != expectedType {
		return fmt.Errorf("resource config %s type mismatch: expected %s", configID, expectedType)
	}
	if !item.Enabled {
		return fmt.Errorf("resource config %s is disabled", configID)
	}
	return nil
}

func validateWorkspaceModelConfigReference(state *AppState, workspaceID string, modelConfigID string) error {
	workspaceID = strings.TrimSpace(workspaceID)
	modelConfigID = strings.TrimSpace(modelConfigID)
	if workspaceID == "" || modelConfigID == "" {
		return fmt.Errorf("resource config %s does not exist", modelConfigID)
	}
	if _, exists, err := getWorkspaceEnabledModelConfigByID(state, workspaceID, modelConfigID); err != nil {
		return fmt.Errorf("failed to load resource config %s: %w", modelConfigID, err)
	} else if exists {
		return nil
	}

	item, exists, err := loadWorkspaceResourceConfigRaw(state, workspaceID, modelConfigID)
	if err != nil {
		return fmt.Errorf("failed to load resource config %s: %w", modelConfigID, err)
	}
	if !exists {
		return fmt.Errorf("resource config %s does not exist", modelConfigID)
	}
	if item.Type != ResourceTypeModel {
		return fmt.Errorf("resource config %s type mismatch: expected %s", modelConfigID, ResourceTypeModel)
	}
	if !item.Enabled {
		return fmt.Errorf("resource config %s is disabled", modelConfigID)
	}
	return fmt.Errorf("resource config %s is invalid", modelConfigID)
}
