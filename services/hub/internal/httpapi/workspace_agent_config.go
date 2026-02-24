package httpapi

import "strings"

const (
	defaultWorkspaceAgentMaxModelTurns = 24
	minWorkspaceAgentMaxModelTurns     = 4
	maxWorkspaceAgentMaxModelTurns     = 64
)

func defaultWorkspaceAgentConfig(workspaceID string, updatedAt string) WorkspaceAgentConfig {
	return normalizeWorkspaceAgentConfig(
		workspaceID,
		WorkspaceAgentConfig{
			WorkspaceID: strings.TrimSpace(workspaceID),
			Execution: WorkspaceAgentExecutionConfig{
				MaxModelTurns: defaultWorkspaceAgentMaxModelTurns,
			},
			Display: WorkspaceAgentDisplayConfig{
				ShowProcessTrace: true,
				TraceDetailLevel: WorkspaceAgentTraceDetailLevelVerbose,
			},
			UpdatedAt: updatedAt,
		},
		updatedAt,
	)
}

func normalizeWorkspaceAgentConfig(workspaceID string, input WorkspaceAgentConfig, updatedAt string) WorkspaceAgentConfig {
	normalizedWorkspaceID := strings.TrimSpace(workspaceID)
	if normalizedWorkspaceID == "" {
		normalizedWorkspaceID = strings.TrimSpace(input.WorkspaceID)
	}

	normalized := input
	normalized.WorkspaceID = normalizedWorkspaceID
	normalized.Execution.MaxModelTurns = clampInt(
		normalized.Execution.MaxModelTurns,
		minWorkspaceAgentMaxModelTurns,
		maxWorkspaceAgentMaxModelTurns,
		defaultWorkspaceAgentMaxModelTurns,
	)
	switch normalized.Display.TraceDetailLevel {
	case WorkspaceAgentTraceDetailLevelBasic, WorkspaceAgentTraceDetailLevelVerbose:
	default:
		normalized.Display.TraceDetailLevel = WorkspaceAgentTraceDetailLevelVerbose
	}
	if strings.TrimSpace(updatedAt) == "" {
		normalized.UpdatedAt = nowUTC()
	} else {
		normalized.UpdatedAt = strings.TrimSpace(updatedAt)
	}
	return normalized
}

func toExecutionAgentConfigSnapshot(config WorkspaceAgentConfig) *ExecutionAgentConfigSnapshot {
	normalized := normalizeWorkspaceAgentConfig(config.WorkspaceID, config, config.UpdatedAt)
	return &ExecutionAgentConfigSnapshot{
		MaxModelTurns:    normalized.Execution.MaxModelTurns,
		ShowProcessTrace: normalized.Display.ShowProcessTrace,
		TraceDetailLevel: normalized.Display.TraceDetailLevel,
	}
}

func cloneExecutionAgentConfigSnapshot(input *ExecutionAgentConfigSnapshot) *ExecutionAgentConfigSnapshot {
	if input == nil {
		return nil
	}
	copyValue := *input
	return &copyValue
}

func clampInt(value int, minValue int, maxValue int, defaultValue int) int {
	if value == 0 {
		value = defaultValue
	}
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}
