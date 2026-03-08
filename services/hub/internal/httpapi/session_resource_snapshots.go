package httpapi

import (
	"fmt"
	"strings"
)

func loadSessionResourceSnapshots(state *AppState, sessionID string) ([]SessionResourceSnapshot, error) {
	normalizedSessionID := strings.TrimSpace(sessionID)
	if state == nil || normalizedSessionID == "" {
		return []SessionResourceSnapshot{}, nil
	}

	state.mu.RLock()
	if items, exists := state.sessionResourceSnapshots[normalizedSessionID]; exists {
		state.mu.RUnlock()
		return cloneSessionResourceSnapshots(items), nil
	}
	state.mu.RUnlock()

	if state.authz == nil {
		return []SessionResourceSnapshot{}, nil
	}
	items, err := state.authz.listSessionResourceSnapshots(normalizedSessionID)
	if err != nil {
		return nil, err
	}
	state.mu.Lock()
	state.sessionResourceSnapshots[normalizedSessionID] = cloneSessionResourceSnapshots(items)
	state.mu.Unlock()
	return cloneSessionResourceSnapshots(items), nil
}

func replaceSessionResourceSnapshots(state *AppState, sessionID string, snapshots []SessionResourceSnapshot) error {
	normalizedSessionID := strings.TrimSpace(sessionID)
	if state == nil || normalizedSessionID == "" {
		return nil
	}

	items := cloneSessionResourceSnapshots(snapshots)
	for index := range items {
		items[index].SessionID = normalizedSessionID
		items[index].ResourceConfigID = strings.TrimSpace(items[index].ResourceConfigID)
		items[index].ResourceType = ResourceType(strings.TrimSpace(string(items[index].ResourceType)))
		items[index].ResourceVersion = normalizeResourceConfigVersion(items[index].ResourceVersion)
		items[index].SnapshotAt = strings.TrimSpace(items[index].SnapshotAt)
	}

	if state.authz != nil {
		if err := state.authz.replaceSessionResourceSnapshots(normalizedSessionID, items); err != nil {
			return err
		}
	}
	state.mu.Lock()
	state.sessionResourceSnapshots[normalizedSessionID] = cloneSessionResourceSnapshots(items)
	state.mu.Unlock()
	return nil
}

func deleteSessionResourceSnapshots(state *AppState, sessionID string) error {
	normalizedSessionID := strings.TrimSpace(sessionID)
	if state == nil || normalizedSessionID == "" {
		return nil
	}
	if state.authz != nil {
		if err := state.authz.deleteSessionResourceSnapshots(normalizedSessionID); err != nil {
			return err
		}
	}
	state.mu.Lock()
	delete(state.sessionResourceSnapshots, normalizedSessionID)
	state.mu.Unlock()
	return nil
}

func captureSessionResourceSnapshots(
	state *AppState,
	sessionID string,
	workspaceID string,
	modelConfigID string,
	ruleIDs []string,
	skillIDs []string,
	mcpIDs []string,
	snapshotAt string,
) ([]SessionResourceSnapshot, error) {
	items := make([]SessionResourceSnapshot, 0, 1+len(ruleIDs)+len(skillIDs)+len(mcpIDs))
	appendSnapshot := func(configID string, expectedType ResourceType) error {
		normalizedConfigID := strings.TrimSpace(configID)
		if normalizedConfigID == "" {
			return nil
		}
		item, exists, err := loadWorkspaceResourceConfigRaw(state, workspaceID, normalizedConfigID)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("resource config %s does not exist", normalizedConfigID)
		}
		if item.Type != expectedType {
			return fmt.Errorf("resource config %s type mismatch: expected %s", normalizedConfigID, expectedType)
		}
		items = append(items, SessionResourceSnapshot{
			SessionID:        strings.TrimSpace(sessionID),
			ResourceConfigID: normalizedConfigID,
			ResourceType:     expectedType,
			ResourceVersion:  item.Version,
			IsDeprecated:     false,
			SnapshotAt:       strings.TrimSpace(snapshotAt),
			CapturedConfig:   item,
		})
		return nil
	}

	if err := appendSnapshot(modelConfigID, ResourceTypeModel); err != nil {
		return nil, err
	}
	for _, ruleID := range sanitizeIDList(ruleIDs) {
		if err := appendSnapshot(ruleID, ResourceTypeRule); err != nil {
			return nil, err
		}
	}
	for _, skillID := range sanitizeIDList(skillIDs) {
		if err := appendSnapshot(skillID, ResourceTypeSkill); err != nil {
			return nil, err
		}
	}
	for _, mcpID := range sanitizeIDList(mcpIDs) {
		if err := appendSnapshot(mcpID, ResourceTypeMCP); err != nil {
			return nil, err
		}
	}
	return items, nil
}

func resolveSessionResourceConfig(
	state *AppState,
	sessionID string,
	workspaceID string,
	configID string,
	expectedType ResourceType,
) (ResourceConfig, bool, error) {
	normalizedSessionID := strings.TrimSpace(sessionID)
	normalizedWorkspaceID := strings.TrimSpace(workspaceID)
	normalizedConfigID := strings.TrimSpace(configID)
	if normalizedWorkspaceID == "" || normalizedConfigID == "" {
		return ResourceConfig{}, false, nil
	}

	if normalizedSessionID != "" {
		items, err := loadSessionResourceSnapshots(state, normalizedSessionID)
		if err != nil {
			return ResourceConfig{}, false, err
		}
		for _, item := range items {
			if strings.TrimSpace(item.ResourceConfigID) != normalizedConfigID {
				continue
			}
			if item.ResourceType != expectedType {
				continue
			}
			if item.IsDeprecated {
				if item.FallbackResourceID != nil && strings.TrimSpace(*item.FallbackResourceID) != "" {
					return resolveSessionResourceConfig(state, normalizedSessionID, normalizedWorkspaceID, *item.FallbackResourceID, expectedType)
				}
				return ResourceConfig{}, false, nil
			}
			return item.CapturedConfig, true, nil
		}
	}

	item, exists, err := loadWorkspaceResourceConfigRaw(state, normalizedWorkspaceID, normalizedConfigID)
	if err != nil {
		return ResourceConfig{}, false, err
	}
	if !exists || item.Type != expectedType {
		return ResourceConfig{}, false, nil
	}
	return item, true, nil
}

func resolveSessionResourceConfigs(
	state *AppState,
	sessionID string,
	workspaceID string,
	ids []string,
	expectedType ResourceType,
) ([]ResourceConfig, error) {
	if len(ids) == 0 {
		return []ResourceConfig{}, nil
	}
	items := make([]ResourceConfig, 0, len(ids))
	for _, rawID := range sanitizeIDList(ids) {
		item, exists, err := resolveSessionResourceConfig(state, sessionID, workspaceID, rawID, expectedType)
		if err != nil {
			return nil, err
		}
		if !exists {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func cloneSessionResourceSnapshots(items []SessionResourceSnapshot) []SessionResourceSnapshot {
	if len(items) == 0 {
		return []SessionResourceSnapshot{}
	}
	result := make([]SessionResourceSnapshot, 0, len(items))
	for _, item := range items {
		copyItem := item
		copyItem.FallbackResourceID = cloneStringPointer(item.FallbackResourceID)
		result = append(result, copyItem)
	}
	return result
}
