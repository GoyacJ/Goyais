package httpapi

import (
	"fmt"
	"strings"
)

func validateExecutionTokenThresholdsLocked(
	state *AppState,
	conversation Conversation,
	projectConfig ProjectConfig,
	modelConfig ResourceConfig,
	modelConfigID string,
) error {
	normalizedProjectID := strings.TrimSpace(conversation.ProjectID)
	normalizedWorkspaceID := strings.TrimSpace(conversation.WorkspaceID)
	normalizedModelConfigID := strings.TrimSpace(modelConfigID)
	if normalizedProjectID == "" || normalizedWorkspaceID == "" || normalizedModelConfigID == "" {
		return nil
	}

	aggregate := computeTokenUsageAggregateLocked(state)

	if threshold, exists := projectConfig.ModelTokenThresholds[normalizedModelConfigID]; exists && threshold > 0 {
		current := aggregate.projectModelTotals[normalizedProjectID][normalizedModelConfigID].Total
		if current >= threshold {
			return fmt.Errorf("project model token threshold reached for %s (%d/%d)", normalizedModelConfigID, current, threshold)
		}
	}
	if projectConfig.TokenThreshold != nil {
		threshold := *projectConfig.TokenThreshold
		current := aggregate.projectTotals[normalizedProjectID].Total
		if current >= threshold {
			return fmt.Errorf("project token threshold reached (%d/%d)", current, threshold)
		}
	}
	if modelConfig.Model != nil && modelConfig.Model.TokenThreshold != nil {
		threshold := *modelConfig.Model.TokenThreshold
		current := aggregate.workspaceModelTotals[normalizedWorkspaceID][normalizedModelConfigID].Total
		if current >= threshold {
			return fmt.Errorf("workspace model token threshold reached for %s (%d/%d)", normalizedModelConfigID, current, threshold)
		}
	}

	return nil
}
