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
	projectConfig ProjectConfig,
	modelSelector string,
	modelConfigs []ResourceConfig,
) (string, ModelSnapshot) {
	selector := strings.TrimSpace(modelSelector)
	if selector == "" {
		return "", ModelSnapshot{}
	}

	selected, exists := selectModelConfigForExecution(modelConfigs, projectConfig, selector)
	if !exists || selected.Model == nil {
		return selector, ModelSnapshot{ModelID: selector}
	}

	modelID := strings.TrimSpace(selected.Model.ModelID)
	if modelID == "" {
		modelID = selector
	}

	snapshot := ModelSnapshot{
		ConfigID:  strings.TrimSpace(selected.ID),
		Vendor:    strings.TrimSpace(string(selected.Model.Vendor)),
		ModelID:   modelID,
		BaseURL:   resolveModelBaseURLForExecution(state, workspaceID, selected.Model),
		TimeoutMS: selected.Model.TimeoutMS,
	}
	if len(selected.Model.Params) > 0 {
		snapshot.Params = cloneMapAny(selected.Model.Params)
	}

	return modelID, snapshot
}

func hydrateExecutionModelSnapshotForWorker(state *AppState, execution Execution) Execution {
	hydrated := execution
	snapshot := cloneModelSnapshot(execution.ModelSnapshot)
	if strings.TrimSpace(snapshot.ModelID) == "" {
		snapshot.ModelID = strings.TrimSpace(execution.ModelID)
	}

	config, exists := loadExecutionModelConfigRaw(state, execution.WorkspaceID, snapshot.ConfigID, snapshot.ModelID)
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
	snapshot.BaseURL = resolveModelBaseURLForExecution(state, execution.WorkspaceID, model)
	if model.TimeoutMS > 0 {
		snapshot.TimeoutMS = model.TimeoutMS
	}

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
	modelID string,
) (ResourceConfig, bool) {
	if id := strings.TrimSpace(configID); id != "" {
		item, exists, err := loadWorkspaceResourceConfigRaw(state, strings.TrimSpace(workspaceID), id)
		if err == nil && exists && item.Type == ResourceTypeModel && item.Model != nil {
			return item, true
		}
	}

	enabled := true
	items, err := listWorkspaceResourceConfigs(state, strings.TrimSpace(workspaceID), resourceConfigQuery{
		Type:    ResourceTypeModel,
		Enabled: &enabled,
	})
	if err != nil {
		return ResourceConfig{}, false
	}

	selected, exists := selectModelConfigForExecution(items, ProjectConfig{}, strings.TrimSpace(modelID))
	if !exists {
		return ResourceConfig{}, false
	}

	item, exists, err := loadWorkspaceResourceConfigRaw(state, strings.TrimSpace(workspaceID), strings.TrimSpace(selected.ID))
	if err != nil || !exists || item.Type != ResourceTypeModel || item.Model == nil {
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

	enabledModels := make([]ResourceConfig, 0, len(modelConfigs))
	byID := map[string]ResourceConfig{}
	byModelID := map[string][]ResourceConfig{}
	for _, item := range modelConfigs {
		if item.Type != ResourceTypeModel || !item.Enabled || item.Model == nil {
			continue
		}
		id := strings.TrimSpace(item.ID)
		modelID := strings.TrimSpace(item.Model.ModelID)
		if id == "" || modelID == "" {
			continue
		}
		enabledModels = append(enabledModels, item)
		byID[id] = item
		byModelID[modelID] = append(byModelID[modelID], item)
	}

	if item, ok := byID[selector]; ok {
		return item, true
	}
	if items := byModelID[selector]; len(items) > 0 {
		if item, ok := pickConfigByProjectPreference(items, projectConfig); ok {
			return item, true
		}
		return items[0], true
	}

	// 兼容历史绑定：若 selector 未命中，尝试按 ProjectConfig 里的优先序反查。
	orderedSelectors := make([]string, 0, len(projectConfig.ModelIDs)+1)
	if projectConfig.DefaultModelID != nil {
		if value := strings.TrimSpace(*projectConfig.DefaultModelID); value != "" {
			orderedSelectors = append(orderedSelectors, value)
		}
	}
	for _, item := range projectConfig.ModelIDs {
		if value := strings.TrimSpace(item); value != "" {
			orderedSelectors = append(orderedSelectors, value)
		}
	}
	for _, item := range orderedSelectors {
		if candidate, ok := byID[item]; ok {
			return candidate, true
		}
		if candidates := byModelID[item]; len(candidates) > 0 {
			return candidates[0], true
		}
	}

	if len(enabledModels) == 0 {
		return ResourceConfig{}, false
	}
	return enabledModels[0], true
}

func pickConfigByProjectPreference(candidates []ResourceConfig, projectConfig ProjectConfig) (ResourceConfig, bool) {
	preferredIDs := make([]string, 0, len(projectConfig.ModelIDs)+1)
	if projectConfig.DefaultModelID != nil {
		if value := strings.TrimSpace(*projectConfig.DefaultModelID); value != "" {
			preferredIDs = append(preferredIDs, value)
		}
	}
	for _, item := range projectConfig.ModelIDs {
		if value := strings.TrimSpace(item); value != "" {
			preferredIDs = append(preferredIDs, value)
		}
	}
	if len(preferredIDs) == 0 {
		return ResourceConfig{}, false
	}

	byID := map[string]ResourceConfig{}
	for _, item := range candidates {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		byID[id] = item
	}
	for _, preferredID := range preferredIDs {
		if item, ok := byID[preferredID]; ok {
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
