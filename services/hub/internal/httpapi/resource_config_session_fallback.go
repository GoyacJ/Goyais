package httpapi

import (
	"context"
	"strings"

	"goyais/services/hub/internal/domain"
)

func applyResourceConfigDeletionEffects(state *AppState, workspaceID string, deleted ResourceConfig) error {
	normalizedWorkspaceID := strings.TrimSpace(workspaceID)
	normalizedConfigID := strings.TrimSpace(deleted.ID)
	if state == nil || normalizedWorkspaceID == "" || normalizedConfigID == "" {
		return nil
	}

	type affectedSession struct {
		sessionID string
		session   Conversation
	}
	affected := make([]affectedSession, 0)
	state.mu.RLock()
	for sessionID, conversation := range state.conversations {
		if strings.TrimSpace(conversation.WorkspaceID) != normalizedWorkspaceID {
			continue
		}
		switch deleted.Type {
		case ResourceTypeModel:
			if strings.TrimSpace(conversation.ModelConfigID) != normalizedConfigID {
				continue
			}
		case ResourceTypeRule:
			if !containsString(conversation.RuleIDs, normalizedConfigID) {
				continue
			}
		case ResourceTypeSkill:
			if !containsString(conversation.SkillIDs, normalizedConfigID) {
				continue
			}
		case ResourceTypeMCP:
			if !containsString(conversation.MCPIDs, normalizedConfigID) {
				continue
			}
		default:
			continue
		}
		affected = append(affected, affectedSession{sessionID: sessionID, session: conversation})
	}
	state.mu.RUnlock()

	if len(affected) == 0 {
		return nil
	}

	service := newResourceConfigDomainService(state)
	candidates := make([]domain.AffectedSessionResources, 0, len(affected))
	sessionsByID := make(map[string]Conversation, len(affected))
	for _, item := range affected {
		sessionsByID[item.sessionID] = item.session
		project, exists, err := getProjectFromStore(state, item.session.ProjectID)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		projectConfig, err := getProjectConfigFromStore(state, project)
		if err != nil {
			return err
		}

		candidates = append(candidates, domain.AffectedSessionResources{
			Session:       toDomainSessionResourceState(item.session),
			ProjectConfig: toDomainProjectResourceConfig(projectConfig),
		})
	}

	plans, err := service.PlanDeletedResource(context.Background(), domain.PlanDeletedResourceRequest{
		WorkspaceID:      domain.WorkspaceID(normalizedWorkspaceID),
		DeletedConfig:    toDomainResourceConfig(deleted),
		AffectedSessions: candidates,
		Timestamp:        nowUTC(),
	})
	if err != nil {
		return err
	}

	for _, plan := range plans {
		sessionID := strings.TrimSpace(string(plan.Session.SessionID))
		updatedSession := fromDomainSessionResourceState(plan.Session, sessionsByID[sessionID])
		state.mu.Lock()
		state.conversations[sessionID] = updatedSession
		state.mu.Unlock()
		if err := replaceSessionResourceSnapshots(state, sessionID, fromDomainSessionResourceSnapshots(plan.Snapshots)); err != nil {
			return err
		}
		event := fromDomainResourceEvent(plan.Event)
		if strings.TrimSpace(event.EventID) == "" {
			event.EventID = "wev_" + randomHex(8)
		}
		emitWorkspaceResourceEvent(state, event)
	}

	syncExecutionDomainBestEffort(state)
	return nil
}

func resolveProjectFallbackModelConfigID(
	state *AppState,
	workspaceID string,
	project Project,
	projectConfig ProjectConfig,
	deletedConfigID string,
) string {
	candidates := make([]string, 0, len(projectConfig.ModelConfigIDs)+2)
	if defaultID := strings.TrimSpace(derefString(projectConfig.DefaultModelConfigID)); defaultID != "" {
		candidates = append(candidates, defaultID)
	}
	if defaultID := strings.TrimSpace(project.DefaultModelConfigID); defaultID != "" {
		candidates = append(candidates, defaultID)
	}
	candidates = append(candidates, sanitizeIDList(projectConfig.ModelConfigIDs)...)
	for _, candidate := range candidates {
		normalizedCandidate := strings.TrimSpace(candidate)
		if normalizedCandidate == "" || normalizedCandidate == strings.TrimSpace(deletedConfigID) {
			continue
		}
		item, exists, err := getWorkspaceEnabledModelConfigByID(state, workspaceID, normalizedCandidate)
		if err != nil || !exists || item.Model == nil {
			continue
		}
		return normalizedCandidate
	}
	return ""
}

func markSessionResourceSnapshotDeprecated(
	items []SessionResourceSnapshot,
	sessionID string,
	deleted ResourceConfig,
	fallbackResourceID string,
	snapshotAt string,
) []SessionResourceSnapshot {
	out := cloneSessionResourceSnapshots(items)
	updated := false
	for index := range out {
		if strings.TrimSpace(out[index].ResourceConfigID) != strings.TrimSpace(deleted.ID) {
			continue
		}
		out[index].IsDeprecated = true
		out[index].SnapshotAt = strings.TrimSpace(snapshotAt)
		if strings.TrimSpace(fallbackResourceID) != "" {
			out[index].FallbackResourceID = toStringPtr(fallbackResourceID)
		} else {
			out[index].FallbackResourceID = nil
		}
		updated = true
	}
	if !updated {
		fallbackPtr := (*string)(nil)
		if strings.TrimSpace(fallbackResourceID) != "" {
			fallbackPtr = toStringPtr(fallbackResourceID)
		}
		out = append(out, SessionResourceSnapshot{
			SessionID:          strings.TrimSpace(sessionID),
			ResourceConfigID:   strings.TrimSpace(deleted.ID),
			ResourceType:       deleted.Type,
			ResourceVersion:    deleted.Version,
			IsDeprecated:       true,
			FallbackResourceID: fallbackPtr,
			SnapshotAt:         strings.TrimSpace(snapshotAt),
			CapturedConfig:     deleted,
		})
	}
	return out
}

func mergeSessionResourceSnapshots(current []SessionResourceSnapshot, incoming []SessionResourceSnapshot) []SessionResourceSnapshot {
	if len(incoming) == 0 {
		return cloneSessionResourceSnapshots(current)
	}
	indexByID := make(map[string]int, len(current))
	out := cloneSessionResourceSnapshots(current)
	for index, item := range out {
		indexByID[strings.TrimSpace(item.ResourceConfigID)] = index
	}
	for _, item := range incoming {
		key := strings.TrimSpace(item.ResourceConfigID)
		if existingIndex, exists := indexByID[key]; exists {
			out[existingIndex] = item
			continue
		}
		indexByID[key] = len(out)
		out = append(out, item)
	}
	return out
}

func removeStringID(items []string, target string) []string {
	normalizedTarget := strings.TrimSpace(target)
	if normalizedTarget == "" {
		return sanitizeIDList(items)
	}
	out := make([]string, 0, len(items))
	for _, item := range sanitizeIDList(items) {
		if strings.TrimSpace(item) == normalizedTarget {
			continue
		}
		out = append(out, item)
	}
	return out
}
