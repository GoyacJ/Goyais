package httpapi

import "strings"

type tokenUsageTotals struct {
	Input  int
	Output int
	Total  int
}

type tokenUsageAggregate struct {
	projectTotals        map[string]tokenUsageTotals
	projectModelTotals   map[string]map[string]tokenUsageTotals
	workspaceModelTotals map[string]map[string]tokenUsageTotals
}

func computeTokenUsageAggregateLocked(state *AppState) tokenUsageAggregate {
	aggregate := tokenUsageAggregate{
		projectTotals:        map[string]tokenUsageTotals{},
		projectModelTotals:   map[string]map[string]tokenUsageTotals{},
		workspaceModelTotals: map[string]map[string]tokenUsageTotals{},
	}

	for _, execution := range state.executions {
		conversation, exists := state.conversations[execution.ConversationID]
		if !exists {
			continue
		}

		usage := tokenUsageTotals{
			Input:  normalizeTokenCount(execution.TokensIn),
			Output: normalizeTokenCount(execution.TokensOut),
		}
		usage.Total = usage.Input + usage.Output
		if usage.Total <= 0 {
			continue
		}

		projectID := strings.TrimSpace(conversation.ProjectID)
		workspaceID := strings.TrimSpace(conversation.WorkspaceID)
		if projectID != "" {
			aggregate.projectTotals[projectID] = addTokenUsage(aggregate.projectTotals[projectID], usage)
		}

		modelConfigID := resolveExecutionModelConfigID(execution)
		if modelConfigID == "" {
			continue
		}
		if projectID != "" {
			aggregate.projectModelTotals[projectID] = addTokenUsageByModelConfigID(aggregate.projectModelTotals[projectID], modelConfigID, usage)
		}
		if workspaceID != "" {
			aggregate.workspaceModelTotals[workspaceID] = addTokenUsageByModelConfigID(aggregate.workspaceModelTotals[workspaceID], modelConfigID, usage)
		}
	}

	return aggregate
}

func resolveExecutionModelConfigID(execution Execution) string {
	if execution.ResourceProfileSnapshot != nil {
		if fromResourceProfile := strings.TrimSpace(execution.ResourceProfileSnapshot.ModelConfigID); fromResourceProfile != "" {
			return fromResourceProfile
		}
	}
	return strings.TrimSpace(execution.ModelSnapshot.ConfigID)
}

func addTokenUsage(current tokenUsageTotals, incoming tokenUsageTotals) tokenUsageTotals {
	return tokenUsageTotals{
		Input:  current.Input + incoming.Input,
		Output: current.Output + incoming.Output,
		Total:  current.Total + incoming.Total,
	}
}

func addTokenUsageByModelConfigID(
	current map[string]tokenUsageTotals,
	modelConfigID string,
	incoming tokenUsageTotals,
) map[string]tokenUsageTotals {
	if current == nil {
		current = map[string]tokenUsageTotals{}
	}
	current[modelConfigID] = addTokenUsage(current[modelConfigID], incoming)
	return current
}

func normalizeTokenCount(input int) int {
	if input <= 0 {
		return 0
	}
	return input
}

func toModelTokenUsage(totals tokenUsageTotals) ModelTokenUsage {
	return ModelTokenUsage{
		TokensInTotal:  totals.Input,
		TokensOutTotal: totals.Output,
		TokensTotal:    totals.Total,
	}
}
