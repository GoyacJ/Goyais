package httpapi

import (
	"context"
	"strings"

	"goyais/services/hub/internal/domain"
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
	service := newResourceConfigDomainService(state)
	items, err := service.CaptureSessionSnapshots(context.Background(), domain.CaptureSessionSnapshotsRequest{
		SessionID:     domain.SessionID(strings.TrimSpace(sessionID)),
		WorkspaceID:   domain.WorkspaceID(strings.TrimSpace(workspaceID)),
		ModelConfigID: strings.TrimSpace(modelConfigID),
		RuleIDs:       append([]string{}, ruleIDs...),
		SkillIDs:      append([]string{}, skillIDs...),
		MCPIDs:        append([]string{}, mcpIDs...),
		SnapshotAt:    strings.TrimSpace(snapshotAt),
	})
	if err != nil {
		return nil, err
	}
	return fromDomainSessionResourceSnapshots(items), nil
}

func resolveSessionResourceConfig(
	state *AppState,
	sessionID string,
	workspaceID string,
	configID string,
	expectedType ResourceType,
) (ResourceConfig, bool, error) {
	service := newResourceConfigDomainService(state)
	item, exists, err := service.ResolveSessionResourceConfig(
		context.Background(),
		domain.SessionID(strings.TrimSpace(sessionID)),
		domain.WorkspaceID(strings.TrimSpace(workspaceID)),
		strings.TrimSpace(configID),
		domain.ResourceType(strings.TrimSpace(string(expectedType))),
	)
	if err != nil || !exists {
		return ResourceConfig{}, exists, err
	}
	return fromDomainResourceConfig(item), true, nil
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
