package httpapi

import "strings"

func firstNonEmptyMode(primary ConversationMode, fallback ConversationMode) ConversationMode {
	if strings.TrimSpace(string(primary)) != "" {
		return primary
	}
	return fallback
}

func resolveExecutionModelSnapshot(
	state *AppState,
	workspaceID string,
	modelConfig ResourceConfig,
) (string, ModelSnapshot) {
	if modelConfig.Model == nil {
		return "", ModelSnapshot{}
	}

	modelID := strings.TrimSpace(modelConfig.Model.ModelID)
	if modelID == "" {
		return "", ModelSnapshot{}
	}

	snapshot := ModelSnapshot{
		ConfigID:   strings.TrimSpace(modelConfig.ID),
		Vendor:     strings.TrimSpace(string(modelConfig.Model.Vendor)),
		ModelID:    modelID,
		BaseURLKey: strings.TrimSpace(modelConfig.Model.BaseURLKey),
		Runtime:    cloneModelRuntimeSpec(modelConfig.Model.Runtime),
	}
	if modelConfig.Model.Vendor == ModelVendorLocal {
		snapshot.BaseURL = resolveModelBaseURLForExecution(state, workspaceID, modelConfig.Model)
	}
	if len(modelConfig.Model.Params) > 0 {
		snapshot.Params = cloneMapAny(modelConfig.Model.Params)
	}

	return modelID, snapshot
}

func hydrateExecutionModelSnapshotForWorker(state *AppState, execution Execution) Execution {
	hydrated := execution
	snapshot := cloneModelSnapshot(execution.ModelSnapshot)
	if strings.TrimSpace(snapshot.ModelID) == "" {
		snapshot.ModelID = strings.TrimSpace(execution.ModelID)
	}

	config, exists := loadExecutionModelConfigRaw(state, execution.WorkspaceID, snapshot.ConfigID)
	if !exists || config.Model == nil {
		hydrated.ModelSnapshot = snapshot
		return hydrated
	}

	model := config.Model
	modelID := strings.TrimSpace(model.ModelID)
	if modelID != "" {
		snapshot.ModelID = modelID
		hydrated.ModelID = modelID
	}
	snapshot.ConfigID = strings.TrimSpace(config.ID)
	snapshot.Vendor = strings.TrimSpace(string(model.Vendor))
	snapshot.BaseURLKey = strings.TrimSpace(model.BaseURLKey)
	if model.Vendor == ModelVendorLocal {
		snapshot.BaseURL = resolveModelBaseURLForExecution(state, execution.WorkspaceID, model)
	} else {
		snapshot.BaseURL = ""
	}
	snapshot.Runtime = cloneModelRuntimeSpec(model.Runtime)

	params := cloneMapAny(snapshot.Params)
	if len(model.Params) > 0 {
		for key, value := range model.Params {
			params[key] = value
		}
	}
	apiKey := strings.TrimSpace(model.APIKey)
	if apiKey != "" {
		params["api_key"] = apiKey
	}
	if len(params) > 0 {
		snapshot.Params = params
	}

	hydrated.ModelSnapshot = snapshot
	return hydrated
}

func loadExecutionModelConfigRaw(
	state *AppState,
	workspaceID string,
	configID string,
) (ResourceConfig, bool) {
	item, exists, err := getWorkspaceEnabledModelConfigByID(state, workspaceID, configID)
	if err != nil {
		return ResourceConfig{}, false
	}
	if !exists {
		return ResourceConfig{}, false
	}
	return item, true
}

func selectModelConfigForExecution(
	modelConfigs []ResourceConfig,
	projectConfig ProjectConfig,
	modelSelector string,
) (ResourceConfig, bool) {
	selector := strings.TrimSpace(modelSelector)
	if selector == "" {
		return ResourceConfig{}, false
	}

	if len(projectConfig.ModelConfigIDs) > 0 || strings.TrimSpace(derefString(projectConfig.DefaultModelConfigID)) != "" {
		if !containsTrimmed(projectConfig.ModelConfigIDs, selector) {
			return ResourceConfig{}, false
		}
	}

	for _, item := range modelConfigs {
		if item.Type != ResourceTypeModel || !item.Enabled || item.Model == nil {
			continue
		}
		id := strings.TrimSpace(item.ID)
		if id == "" || strings.TrimSpace(item.Model.ModelID) == "" {
			continue
		}
		if id == selector {
			return item, true
		}
	}

	return ResourceConfig{}, false
}

func resolveModelBaseURLForExecution(state *AppState, workspaceID string, model *ModelSpec) string {
	if model == nil {
		return ""
	}
	vendor := ModelVendorName(strings.TrimSpace(string(model.Vendor)))
	if vendor == ModelVendorLocal {
		if localBaseURL := strings.TrimSpace(model.BaseURL); localBaseURL != "" {
			return localBaseURL
		}
	}

	catalogVendor, exists := state.resolveCatalogVendor(strings.TrimSpace(workspaceID), vendor)
	if !exists {
		return strings.TrimSpace(model.BaseURL)
	}

	if endpointKey := strings.TrimSpace(model.BaseURLKey); endpointKey != "" {
		if value, ok := catalogVendor.BaseURLs[endpointKey]; ok {
			trimmed := strings.TrimSpace(value)
			if trimmed != "" {
				return trimmed
			}
		}
	}
	if baseURL := strings.TrimSpace(catalogVendor.BaseURL); baseURL != "" {
		return baseURL
	}
	return strings.TrimSpace(model.BaseURL)
}
