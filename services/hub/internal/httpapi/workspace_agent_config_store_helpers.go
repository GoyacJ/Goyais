package httpapi

import "strings"

func loadWorkspaceAgentConfigFromStore(state *AppState, workspaceID string) (WorkspaceAgentConfig, error) {
	normalizedWorkspaceID := strings.TrimSpace(workspaceID)
	if normalizedWorkspaceID == "" {
		return defaultWorkspaceAgentConfig(localWorkspaceID, nowUTC()), nil
	}

	if state.authz != nil {
		return state.authz.ensureWorkspaceAgentConfig(normalizedWorkspaceID)
	}
	return defaultWorkspaceAgentConfig(normalizedWorkspaceID, nowUTC()), nil
}

func saveWorkspaceAgentConfigToStore(
	state *AppState,
	workspaceID string,
	config WorkspaceAgentConfig,
) (WorkspaceAgentConfig, error) {
	normalizedWorkspaceID := strings.TrimSpace(workspaceID)
	if normalizedWorkspaceID == "" {
		return WorkspaceAgentConfig{}, nil
	}
	if state.authz != nil {
		return state.authz.upsertWorkspaceAgentConfig(normalizedWorkspaceID, config)
	}
	return normalizeWorkspaceAgentConfig(normalizedWorkspaceID, config, nowUTC()), nil
}
