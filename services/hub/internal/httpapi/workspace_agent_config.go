package httpapi

import (
	"strings"

	"goyais/services/hub/internal/agent/tools/catalog"
)

const (
	defaultWorkspaceAgentMaxModelTurns = 24
	minWorkspaceAgentMaxModelTurns     = 4
	maxWorkspaceAgentMaxModelTurns     = 64
	defaultWorkspacePromptBudgetChars  = 16000
	defaultWorkspaceSearchThresholdPct = 10
	defaultWorkspaceMCPResultLimit     = 20
	defaultWorkspaceSubagentMaxTurns   = 8
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
			DefaultMode:  PermissionModeDefault,
			BuiltinTools: catalog.BuiltinToolNames(),
			CapabilityBudgets: WorkspaceAgentCapabilityBudgets{
				PromptBudgetChars:      defaultWorkspacePromptBudgetChars,
				SearchThresholdPercent: defaultWorkspaceSearchThresholdPct,
			},
			MCPSearch: WorkspaceAgentMCPSearchConfig{
				Enabled:     true,
				ResultLimit: defaultWorkspaceMCPResultLimit,
			},
			OutputStyle: "default",
			SubagentDefaults: WorkspaceAgentSubagentDefaults{
				MaxTurns: defaultWorkspaceSubagentMaxTurns,
			},
			FeatureFlags: WorkspaceAgentFeatureFlags{
				EnableToolSearch:      true,
				EnableCapabilityGraph: true,
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
	normalized.DefaultMode = NormalizePermissionMode(string(normalized.DefaultMode))
	normalized.BuiltinTools = sanitizeBuiltinTools(normalized.BuiltinTools)
	if len(normalized.BuiltinTools) == 0 {
		normalized.BuiltinTools = catalog.BuiltinToolNames()
	}
	normalized.CapabilityBudgets.PromptBudgetChars = clampInt(
		normalized.CapabilityBudgets.PromptBudgetChars,
		256,
		512000,
		defaultWorkspacePromptBudgetChars,
	)
	normalized.CapabilityBudgets.SearchThresholdPercent = clampInt(
		normalized.CapabilityBudgets.SearchThresholdPercent,
		1,
		100,
		defaultWorkspaceSearchThresholdPct,
	)
	if normalized.MCPSearch.ResultLimit <= 0 {
		normalized.MCPSearch.ResultLimit = defaultWorkspaceMCPResultLimit
	}
	if strings.TrimSpace(normalized.OutputStyle) == "" {
		normalized.OutputStyle = "default"
	}
	normalized.SubagentDefaults.MaxTurns = clampInt(
		normalized.SubagentDefaults.MaxTurns,
		1,
		64,
		defaultWorkspaceSubagentMaxTurns,
	)
	normalized.SubagentDefaults.AllowedTools = sanitizeBuiltinTools(normalized.SubagentDefaults.AllowedTools)
	if !normalized.MCPSearch.Enabled && input.MCPSearch == (WorkspaceAgentMCPSearchConfig{}) {
		normalized.MCPSearch.Enabled = true
	}
	if !normalized.FeatureFlags.EnableToolSearch && input.FeatureFlags == (WorkspaceAgentFeatureFlags{}) {
		normalized.FeatureFlags.EnableToolSearch = true
	}
	if !normalized.FeatureFlags.EnableCapabilityGraph && input.FeatureFlags == (WorkspaceAgentFeatureFlags{}) {
		normalized.FeatureFlags.EnableCapabilityGraph = true
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
		MaxModelTurns:     normalized.Execution.MaxModelTurns,
		ShowProcessTrace:  normalized.Display.ShowProcessTrace,
		TraceDetailLevel:  normalized.Display.TraceDetailLevel,
		DefaultMode:       normalized.DefaultMode,
		BuiltinTools:      append([]string{}, normalized.BuiltinTools...),
		CapabilityBudgets: normalized.CapabilityBudgets,
		MCPSearch:         normalized.MCPSearch,
		OutputStyle:       strings.TrimSpace(normalized.OutputStyle),
		SubagentDefaults:  WorkspaceAgentSubagentDefaults{MaxTurns: normalized.SubagentDefaults.MaxTurns, AllowedTools: append([]string{}, normalized.SubagentDefaults.AllowedTools...)},
		FeatureFlags:      normalized.FeatureFlags,
	}
}

func cloneExecutionAgentConfigSnapshot(input *ExecutionAgentConfigSnapshot) *ExecutionAgentConfigSnapshot {
	if input == nil {
		return nil
	}
	copyValue := *input
	copyValue.BuiltinTools = append([]string{}, input.BuiltinTools...)
	copyValue.SubagentDefaults = WorkspaceAgentSubagentDefaults{
		MaxTurns:     input.SubagentDefaults.MaxTurns,
		AllowedTools: append([]string{}, input.SubagentDefaults.AllowedTools...),
	}
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

func sanitizeBuiltinTools(input []string) []string {
	if len(input) == 0 {
		return nil
	}
	allowed := map[string]struct{}{}
	for _, item := range catalog.BuiltinToolNames() {
		allowed[strings.TrimSpace(item)] = struct{}{}
	}
	out := make([]string, 0, len(input))
	seen := map[string]struct{}{}
	for _, item := range input {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := allowed[trimmed]; !ok {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}
