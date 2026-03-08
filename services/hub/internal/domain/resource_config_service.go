package domain

import (
	"context"
	"fmt"
	"strings"
)

type ResourceConfigService struct {
	repository ResourceConfigRepository
}

type ValidateSessionSelectionRequest struct {
	WorkspaceID   WorkspaceID
	ProjectConfig ProjectResourceConfig
	ModelConfigID string
	RuleIDs       []string
	SkillIDs      []string
	MCPIDs        []string
}

type CaptureSessionSnapshotsRequest struct {
	SessionID     SessionID
	WorkspaceID   WorkspaceID
	ModelConfigID string
	RuleIDs       []string
	SkillIDs      []string
	MCPIDs        []string
	SnapshotAt    string
}

type PlanDeletedResourceRequest struct {
	WorkspaceID      WorkspaceID
	DeletedConfig    ResourceConfig
	AffectedSessions []AffectedSessionResources
	Timestamp        string
}

func NewResourceConfigService(repository ResourceConfigRepository) *ResourceConfigService {
	return &ResourceConfigService{repository: repository}
}

func (s *ResourceConfigService) ValidateProjectConfig(ctx context.Context, workspaceID WorkspaceID, config ProjectResourceConfig) error {
	if strings.TrimSpace(string(workspaceID)) == "" {
		return fmt.Errorf("workspace_id is required")
	}
	if err := validateOptionalPositiveThreshold("token_threshold", config.TokenThreshold); err != nil {
		return err
	}
	for _, modelConfigID := range sanitizeIDs(config.ModelConfigIDs) {
		if err := s.validateWorkspaceResourceReference(ctx, workspaceID, modelConfigID, ResourceTypeModel); err != nil {
			return err
		}
	}
	for modelConfigID, threshold := range config.ModelTokenThresholds {
		normalizedModelConfigID := strings.TrimSpace(modelConfigID)
		if normalizedModelConfigID == "" {
			return fmt.Errorf("model_token_thresholds contains empty model_config_id")
		}
		if threshold <= 0 {
			return fmt.Errorf("model_token_thresholds.%s must be a positive integer", normalizedModelConfigID)
		}
		if !containsID(config.ModelConfigIDs, normalizedModelConfigID) {
			return fmt.Errorf("model_token_thresholds.%s must be included in model_config_ids", normalizedModelConfigID)
		}
	}
	for _, ruleID := range sanitizeIDs(config.RuleIDs) {
		if err := s.validateWorkspaceResourceReference(ctx, workspaceID, ruleID, ResourceTypeRule); err != nil {
			return err
		}
	}
	for _, skillID := range sanitizeIDs(config.SkillIDs) {
		if err := s.validateWorkspaceResourceReference(ctx, workspaceID, skillID, ResourceTypeSkill); err != nil {
			return err
		}
	}
	for _, mcpID := range sanitizeIDs(config.MCPIDs) {
		if err := s.validateWorkspaceResourceReference(ctx, workspaceID, mcpID, ResourceTypeMCP); err != nil {
			return err
		}
	}
	if config.DefaultModelConfigID != nil && strings.TrimSpace(*config.DefaultModelConfigID) != "" {
		if !containsID(config.ModelConfigIDs, *config.DefaultModelConfigID) {
			return fmt.Errorf("default_model_config_id must be included in model_config_ids")
		}
		if err := s.validateWorkspaceResourceReference(ctx, workspaceID, *config.DefaultModelConfigID, ResourceTypeModel); err != nil {
			return err
		}
	}
	return nil
}

func (s *ResourceConfigService) ValidateSessionSelection(ctx context.Context, req ValidateSessionSelectionRequest) error {
	modelConfigID := strings.TrimSpace(req.ModelConfigID)
	if modelConfigID == "" {
		return fmt.Errorf("model_config_id cannot be empty")
	}
	if !containsID(req.ProjectConfig.ModelConfigIDs, modelConfigID) {
		return fmt.Errorf("model_config_id must be included in project model_config_ids")
	}
	if err := s.validateWorkspaceResourceReference(ctx, req.WorkspaceID, modelConfigID, ResourceTypeModel); err != nil {
		return err
	}
	for _, ruleID := range sanitizeIDs(req.RuleIDs) {
		if !containsID(req.ProjectConfig.RuleIDs, ruleID) {
			return fmt.Errorf("rule_id %s is not allowed by project config", ruleID)
		}
		if err := s.validateWorkspaceResourceReference(ctx, req.WorkspaceID, ruleID, ResourceTypeRule); err != nil {
			return err
		}
	}
	for _, skillID := range sanitizeIDs(req.SkillIDs) {
		if !containsID(req.ProjectConfig.SkillIDs, skillID) {
			return fmt.Errorf("skill_id %s is not allowed by project config", skillID)
		}
		if err := s.validateWorkspaceResourceReference(ctx, req.WorkspaceID, skillID, ResourceTypeSkill); err != nil {
			return err
		}
	}
	for _, mcpID := range sanitizeIDs(req.MCPIDs) {
		if !containsID(req.ProjectConfig.MCPIDs, mcpID) {
			return fmt.Errorf("mcp_id %s is not allowed by project config", mcpID)
		}
		if err := s.validateWorkspaceResourceReference(ctx, req.WorkspaceID, mcpID, ResourceTypeMCP); err != nil {
			return err
		}
	}
	return nil
}

func (s *ResourceConfigService) CaptureSessionSnapshots(ctx context.Context, req CaptureSessionSnapshotsRequest) ([]SessionResourceSnapshot, error) {
	items := make([]SessionResourceSnapshot, 0, 1+len(req.RuleIDs)+len(req.SkillIDs)+len(req.MCPIDs))
	appendSnapshot := func(configID string, expectedType ResourceType) error {
		normalizedID := strings.TrimSpace(configID)
		if normalizedID == "" {
			return nil
		}
		item, exists, err := s.loadResourceConfig(ctx, req.WorkspaceID, normalizedID, expectedType)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("resource config %s does not exist", normalizedID)
		}
		items = append(items, SessionResourceSnapshot{
			SessionID:        req.SessionID,
			ResourceConfigID: normalizedID,
			ResourceType:     expectedType,
			ResourceVersion:  normalizeResourceVersion(item.Version),
			SnapshotAt:       strings.TrimSpace(req.SnapshotAt),
			CapturedConfig:   item,
		})
		return nil
	}

	if err := appendSnapshot(req.ModelConfigID, ResourceTypeModel); err != nil {
		return nil, err
	}
	for _, ruleID := range sanitizeIDs(req.RuleIDs) {
		if err := appendSnapshot(ruleID, ResourceTypeRule); err != nil {
			return nil, err
		}
	}
	for _, skillID := range sanitizeIDs(req.SkillIDs) {
		if err := appendSnapshot(skillID, ResourceTypeSkill); err != nil {
			return nil, err
		}
	}
	for _, mcpID := range sanitizeIDs(req.MCPIDs) {
		if err := appendSnapshot(mcpID, ResourceTypeMCP); err != nil {
			return nil, err
		}
	}
	return items, nil
}

func (s *ResourceConfigService) ResolveSessionResourceConfig(ctx context.Context, sessionID SessionID, workspaceID WorkspaceID, configID string, expectedType ResourceType) (ResourceConfig, bool, error) {
	normalizedConfigID := strings.TrimSpace(configID)
	if normalizedConfigID == "" {
		return ResourceConfig{}, false, nil
	}
	if sessionID != "" && s != nil && s.repository != nil {
		snapshots, err := s.repository.ListSessionResourceSnapshots(ctx, sessionID)
		if err != nil {
			return ResourceConfig{}, false, err
		}
		for _, snapshot := range snapshots {
			if strings.TrimSpace(snapshot.ResourceConfigID) != normalizedConfigID || snapshot.ResourceType != expectedType {
				continue
			}
			if snapshot.IsDeprecated {
				if snapshot.FallbackResourceID != nil && strings.TrimSpace(*snapshot.FallbackResourceID) != "" {
					return s.ResolveSessionResourceConfig(ctx, sessionID, workspaceID, *snapshot.FallbackResourceID, expectedType)
				}
				return ResourceConfig{}, false, nil
			}
			return snapshot.CapturedConfig, true, nil
		}
	}
	return s.loadResourceConfig(ctx, workspaceID, normalizedConfigID, expectedType)
}

func (s *ResourceConfigService) PlanDeletedResource(ctx context.Context, req PlanDeletedResourceRequest) ([]DeletedResourcePlan, error) {
	plans := make([]DeletedResourcePlan, 0, len(req.AffectedSessions))
	for _, candidate := range req.AffectedSessions {
		snapshots, err := s.repository.ListSessionResourceSnapshots(ctx, candidate.Session.SessionID)
		if err != nil {
			return nil, err
		}
		updatedSession := candidate.Session
		fallbackResourceID := ""
		switch req.DeletedConfig.Type {
		case ResourceTypeModel:
			fallbackResourceID, err = s.resolveFallbackModelConfigID(ctx, req.WorkspaceID, candidate.ProjectConfig, req.DeletedConfig.ID)
			if err != nil {
				return nil, err
			}
			updatedSession.ModelConfigID = fallbackResourceID
		case ResourceTypeRule:
			updatedSession.RuleIDs = removeID(updatedSession.RuleIDs, req.DeletedConfig.ID)
		case ResourceTypeSkill:
			updatedSession.SkillIDs = removeID(updatedSession.SkillIDs, req.DeletedConfig.ID)
		case ResourceTypeMCP:
			updatedSession.MCPIDs = removeID(updatedSession.MCPIDs, req.DeletedConfig.ID)
		}
		updatedSession.UpdatedAt = strings.TrimSpace(req.Timestamp)
		nextSnapshots := markSnapshotsDeprecated(snapshots, updatedSession.SessionID, req.DeletedConfig, fallbackResourceID, updatedSession.UpdatedAt)
		if req.DeletedConfig.Type == ResourceTypeModel && fallbackResourceID != "" {
			fallbackSnapshots, err := s.CaptureSessionSnapshots(ctx, CaptureSessionSnapshotsRequest{
				SessionID:     updatedSession.SessionID,
				WorkspaceID:   req.WorkspaceID,
				ModelConfigID: fallbackResourceID,
				RuleIDs:       updatedSession.RuleIDs,
				SkillIDs:      updatedSession.SkillIDs,
				MCPIDs:        updatedSession.MCPIDs,
				SnapshotAt:    updatedSession.UpdatedAt,
			})
			if err != nil {
				return nil, err
			}
			nextSnapshots = mergeSnapshots(nextSnapshots, fallbackSnapshots)
		}
		plans = append(plans, DeletedResourcePlan{
			Session:   updatedSession,
			Snapshots: nextSnapshots,
			Event: ResourceEvent{
				WorkspaceID:     req.WorkspaceID,
				Type:            ResourceEventTypeSnapshotDeprecated,
				ConfigID:        strings.TrimSpace(req.DeletedConfig.ID),
				ConfigType:      req.DeletedConfig.Type,
				ResourceVersion: normalizeResourceVersion(req.DeletedConfig.Version),
				SessionID:       updatedSession.SessionID,
				Timestamp:       updatedSession.UpdatedAt,
				Payload: map[string]any{
					"fallback_resource_id": strings.TrimSpace(fallbackResourceID),
				},
			},
		})
	}
	return plans, nil
}

func (s *ResourceConfigService) resolveFallbackModelConfigID(ctx context.Context, workspaceID WorkspaceID, projectConfig ProjectResourceConfig, deletedConfigID string) (string, error) {
	candidates := make([]string, 0, len(projectConfig.ModelConfigIDs)+1)
	if projectConfig.DefaultModelConfigID != nil && strings.TrimSpace(*projectConfig.DefaultModelConfigID) != "" {
		candidates = append(candidates, strings.TrimSpace(*projectConfig.DefaultModelConfigID))
	}
	candidates = append(candidates, sanitizeIDs(projectConfig.ModelConfigIDs)...)
	for _, candidate := range candidates {
		if candidate == "" || candidate == strings.TrimSpace(deletedConfigID) {
			continue
		}
		item, exists, err := s.loadResourceConfig(ctx, workspaceID, candidate, ResourceTypeModel)
		if err != nil {
			return "", err
		}
		if exists && item.Enabled && !item.IsDeleted {
			return candidate, nil
		}
	}
	return "", nil
}

func (s *ResourceConfigService) validateWorkspaceResourceReference(ctx context.Context, workspaceID WorkspaceID, configID string, expectedType ResourceType) error {
	item, exists, err := s.loadResourceConfig(ctx, workspaceID, configID, expectedType)
	if err != nil {
		return fmt.Errorf("failed to load resource config %s: %w", strings.TrimSpace(configID), err)
	}
	if !exists {
		return fmt.Errorf("resource config %s does not exist", strings.TrimSpace(configID))
	}
	if !item.Enabled || item.IsDeleted {
		return fmt.Errorf("resource config %s is disabled", strings.TrimSpace(configID))
	}
	return nil
}

func (s *ResourceConfigService) loadResourceConfig(ctx context.Context, workspaceID WorkspaceID, configID string, expectedType ResourceType) (ResourceConfig, bool, error) {
	if s == nil || s.repository == nil {
		return ResourceConfig{}, false, nil
	}
	item, exists, err := s.repository.GetResourceConfig(ctx, workspaceID, strings.TrimSpace(configID))
	if err != nil {
		return ResourceConfig{}, false, err
	}
	if !exists || item.Type != expectedType {
		return ResourceConfig{}, false, nil
	}
	return item, true, nil
}

func markSnapshotsDeprecated(items []SessionResourceSnapshot, sessionID SessionID, deleted ResourceConfig, fallbackResourceID string, snapshotAt string) []SessionResourceSnapshot {
	out := cloneSnapshots(items)
	updated := false
	for index := range out {
		if strings.TrimSpace(out[index].ResourceConfigID) != strings.TrimSpace(deleted.ID) {
			continue
		}
		out[index].IsDeprecated = true
		out[index].SnapshotAt = strings.TrimSpace(snapshotAt)
		out[index].FallbackResourceID = nil
		if strings.TrimSpace(fallbackResourceID) != "" {
			out[index].FallbackResourceID = toStringPointer(fallbackResourceID)
		}
		updated = true
	}
	if !updated {
		var fallback *string
		if strings.TrimSpace(fallbackResourceID) != "" {
			fallback = toStringPointer(fallbackResourceID)
		}
		out = append(out, SessionResourceSnapshot{
			SessionID:          sessionID,
			ResourceConfigID:   strings.TrimSpace(deleted.ID),
			ResourceType:       deleted.Type,
			ResourceVersion:    normalizeResourceVersion(deleted.Version),
			IsDeprecated:       true,
			FallbackResourceID: fallback,
			SnapshotAt:         strings.TrimSpace(snapshotAt),
			CapturedConfig:     deleted,
		})
	}
	return out
}

func mergeSnapshots(current []SessionResourceSnapshot, incoming []SessionResourceSnapshot) []SessionResourceSnapshot {
	out := cloneSnapshots(current)
	indexByID := make(map[string]int, len(out))
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

func cloneSnapshots(items []SessionResourceSnapshot) []SessionResourceSnapshot {
	if len(items) == 0 {
		return []SessionResourceSnapshot{}
	}
	out := make([]SessionResourceSnapshot, 0, len(items))
	for _, item := range items {
		copyItem := item
		copyItem.FallbackResourceID = cloneStringPointer(item.FallbackResourceID)
		out = append(out, copyItem)
	}
	return out
}

func removeID(items []string, target string) []string {
	normalizedTarget := strings.TrimSpace(target)
	out := make([]string, 0, len(items))
	for _, item := range sanitizeIDs(items) {
		if item == normalizedTarget {
			continue
		}
		out = append(out, item)
	}
	return out
}

func sanitizeIDs(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(items))
	seen := map[string]struct{}{}
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
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

func containsID(items []string, target string) bool {
	normalizedTarget := strings.TrimSpace(target)
	for _, item := range sanitizeIDs(items) {
		if item == normalizedTarget {
			return true
		}
	}
	return false
}

func normalizeResourceVersion(value int) int {
	if value <= 0 {
		return 1
	}
	return value
}

func validateOptionalPositiveThreshold(name string, value *int) error {
	if value == nil {
		return nil
	}
	if *value <= 0 {
		return fmt.Errorf("%s must be a positive integer", strings.TrimSpace(name))
	}
	return nil
}

func toStringPointer(value string) *string {
	trimmed := strings.TrimSpace(value)
	return &trimmed
}

func cloneStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}
