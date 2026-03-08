package httpapi

import (
	"context"
	"fmt"
	"strings"

	"goyais/services/hub/internal/domain"
)

func validateProjectConfigResourceReferences(state *AppState, workspaceID string, config ProjectConfig) error {
	service := newResourceConfigDomainService(state)
	return service.ValidateProjectConfig(context.Background(), domain.WorkspaceID(strings.TrimSpace(workspaceID)), toDomainProjectResourceConfig(config))
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
	service := newResourceConfigDomainService(state)
	return service.ValidateSessionSelection(context.Background(), domain.ValidateSessionSelectionRequest{
		WorkspaceID:   domain.WorkspaceID(strings.TrimSpace(workspaceID)),
		ProjectConfig: toDomainProjectResourceConfig(projectConfig),
		ModelConfigID: strings.TrimSpace(modelConfigID),
		RuleIDs:       append([]string{}, ruleIDs...),
		SkillIDs:      append([]string{}, skillIDs...),
		MCPIDs:        append([]string{}, mcpIDs...),
	})
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
