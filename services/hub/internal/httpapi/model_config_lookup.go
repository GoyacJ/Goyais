package httpapi

import "strings"

func getWorkspaceEnabledModelConfigByID(
	state *AppState,
	workspaceID string,
	modelConfigID string,
) (ResourceConfig, bool, error) {
	normalizedWorkspaceID := strings.TrimSpace(workspaceID)
	normalizedModelConfigID := strings.TrimSpace(modelConfigID)
	if normalizedWorkspaceID == "" || normalizedModelConfigID == "" {
		return ResourceConfig{}, false, nil
	}

	item, exists, err := loadWorkspaceResourceConfigRaw(state, normalizedWorkspaceID, normalizedModelConfigID)
	if err != nil {
		return ResourceConfig{}, false, err
	}
	if !exists || item.Type != ResourceTypeModel || !item.Enabled || item.Model == nil {
		return ResourceConfig{}, false, nil
	}
	if strings.TrimSpace(item.Model.ModelID) == "" {
		return ResourceConfig{}, false, nil
	}
	return item, true, nil
}

func buildModelDisplayName(config ResourceConfig) string {
	name := strings.TrimSpace(config.Name)
	if name != "" {
		return name
	}
	if config.Model == nil {
		return strings.TrimSpace(config.ID)
	}
	modelID := strings.TrimSpace(config.Model.ModelID)
	if modelID == "" {
		return strings.TrimSpace(config.ID)
	}
	vendor := strings.TrimSpace(string(config.Model.Vendor))
	if vendor == "" {
		return modelID
	}
	return vendor + " / " + modelID
}
