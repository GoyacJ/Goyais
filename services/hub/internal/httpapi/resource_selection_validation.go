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

	for _, modelID := range sanitizeIDList(config.ModelIDs) {
		if err := validateWorkspaceResourceReference(state, workspaceID, modelID, ResourceTypeModel); err != nil {
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
	if config.DefaultModelID != nil {
		defaultModelID := strings.TrimSpace(*config.DefaultModelID)
		if defaultModelID != "" {
			if !containsString(config.ModelIDs, defaultModelID) {
				return fmt.Errorf("default_model_id must be included in model_ids")
			}
			if err := validateWorkspaceResourceReference(state, workspaceID, defaultModelID, ResourceTypeModel); err != nil {
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
	modelID string,
	ruleIDs []string,
	skillIDs []string,
	mcpIDs []string,
) error {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return fmt.Errorf("model_id cannot be empty")
	}
	if !containsString(projectConfig.ModelIDs, modelID) {
		return fmt.Errorf("model_id must be included in project model_ids")
	}
	if err := validateWorkspaceResourceReference(state, workspaceID, modelID, ResourceTypeModel); err != nil {
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
