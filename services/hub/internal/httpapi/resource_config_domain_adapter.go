package httpapi

import (
	"context"
	"strings"

	"goyais/services/hub/internal/domain"
	infrasqlite "goyais/services/hub/internal/infrastructure/sqlite"
)

type inMemoryResourceConfigRepository struct {
	state *AppState
}

func newResourceConfigDomainService(state *AppState) *domain.ResourceConfigService {
	if state == nil {
		return domain.NewResourceConfigService(nil)
	}
	if state.authz != nil && state.authz.db != nil {
		return domain.NewResourceConfigService(infrasqlite.NewResourceConfigRepository(state.authz.db))
	}
	return domain.NewResourceConfigService(inMemoryResourceConfigRepository{state: state})
}

func (r inMemoryResourceConfigRepository) GetResourceConfig(_ context.Context, workspaceID domain.WorkspaceID, configID string) (domain.ResourceConfig, bool, error) {
	item, exists, err := loadWorkspaceResourceConfigRaw(r.state, strings.TrimSpace(string(workspaceID)), strings.TrimSpace(configID))
	if err != nil || !exists {
		return domain.ResourceConfig{}, exists, err
	}
	return toDomainResourceConfig(item), true, nil
}

func (r inMemoryResourceConfigRepository) ListSessionResourceSnapshots(_ context.Context, sessionID domain.SessionID) ([]domain.SessionResourceSnapshot, error) {
	items, err := loadSessionResourceSnapshots(r.state, strings.TrimSpace(string(sessionID)))
	if err != nil {
		return nil, err
	}
	return toDomainSessionResourceSnapshots(items), nil
}

func toDomainProjectResourceConfig(input ProjectConfig) domain.ProjectResourceConfig {
	return domain.ProjectResourceConfig{
		ProjectID:            strings.TrimSpace(input.ProjectID),
		ModelConfigIDs:       append([]string{}, input.ModelConfigIDs...),
		DefaultModelConfigID: cloneStringPointer(input.DefaultModelConfigID),
		TokenThreshold:       cloneIntPointer(input.TokenThreshold),
		ModelTokenThresholds: cloneIntMap(input.ModelTokenThresholds),
		RuleIDs:              append([]string{}, input.RuleIDs...),
		SkillIDs:             append([]string{}, input.SkillIDs...),
		MCPIDs:               append([]string{}, input.MCPIDs...),
		UpdatedAt:            strings.TrimSpace(input.UpdatedAt),
	}
}

func toDomainResourceConfig(input ResourceConfig) domain.ResourceConfig {
	item := domain.ResourceConfig{
		ID:          strings.TrimSpace(input.ID),
		WorkspaceID: domain.WorkspaceID(strings.TrimSpace(input.WorkspaceID)),
		Type:        domain.ResourceType(strings.TrimSpace(string(input.Type))),
		Name:        strings.TrimSpace(input.Name),
		Enabled:     input.Enabled,
		Version:     input.Version,
		IsDeleted:   input.IsDeleted,
		DeletedAt:   cloneStringPointer(input.DeletedAt),
		TokensInTotal:  input.TokensInTotal,
		TokensOutTotal: input.TokensOutTotal,
		TokensTotal:    input.TokensTotal,
		CreatedAt:   strings.TrimSpace(input.CreatedAt),
		UpdatedAt:   strings.TrimSpace(input.UpdatedAt),
	}
	if input.Model != nil {
		item.Model = &domain.ModelSpec{
			Vendor:         strings.TrimSpace(string(input.Model.Vendor)),
			ModelID:        strings.TrimSpace(input.Model.ModelID),
			BaseURL:        strings.TrimSpace(input.Model.BaseURL),
			BaseURLKey:     strings.TrimSpace(input.Model.BaseURLKey),
			APIKey:         strings.TrimSpace(input.Model.APIKey),
			APIKeyMasked:   strings.TrimSpace(input.Model.APIKeyMasked),
			TokenThreshold: cloneIntPointer(input.Model.TokenThreshold),
			Params:         cloneMapAny(input.Model.Params),
		}
		if input.Model.Runtime != nil {
			item.Model.Runtime = &domain.ModelRuntimeSpec{
				RequestTimeoutMS: cloneIntPointer(input.Model.Runtime.RequestTimeoutMS),
			}
		}
	}
	if input.Rule != nil {
		item.Rule = &domain.RuleSpec{Content: input.Rule.Content}
	}
	if input.Skill != nil {
		item.Skill = &domain.SkillSpec{Content: input.Skill.Content}
	}
	if input.MCP != nil {
		item.MCP = &domain.MCPConfig{
			Transport:       strings.TrimSpace(input.MCP.Transport),
			Endpoint:        strings.TrimSpace(input.MCP.Endpoint),
			Command:         strings.TrimSpace(input.MCP.Command),
			Env:             cloneStringMap(input.MCP.Env),
			Status:          strings.TrimSpace(input.MCP.Status),
			Tools:           append([]string{}, input.MCP.Tools...),
			LastError:       strings.TrimSpace(input.MCP.LastError),
			LastConnectedAt: strings.TrimSpace(input.MCP.LastConnectedAt),
		}
	}
	return item
}

func fromDomainResourceConfig(input domain.ResourceConfig) ResourceConfig {
	item := ResourceConfig{
		ID:          strings.TrimSpace(input.ID),
		WorkspaceID: strings.TrimSpace(string(input.WorkspaceID)),
		Type:        ResourceType(strings.TrimSpace(string(input.Type))),
		Name:        strings.TrimSpace(input.Name),
		Enabled:     input.Enabled,
		Version:     input.Version,
		IsDeleted:   input.IsDeleted,
		DeletedAt:   cloneStringPointer(input.DeletedAt),
		TokensInTotal:  input.TokensInTotal,
		TokensOutTotal: input.TokensOutTotal,
		TokensTotal:    input.TokensTotal,
		CreatedAt:   strings.TrimSpace(input.CreatedAt),
		UpdatedAt:   strings.TrimSpace(input.UpdatedAt),
	}
	if input.Model != nil {
		item.Model = &ModelSpec{
			Vendor:         ModelVendorName(strings.TrimSpace(input.Model.Vendor)),
			ModelID:        strings.TrimSpace(input.Model.ModelID),
			BaseURL:        strings.TrimSpace(input.Model.BaseURL),
			BaseURLKey:     strings.TrimSpace(input.Model.BaseURLKey),
			APIKey:         strings.TrimSpace(input.Model.APIKey),
			APIKeyMasked:   strings.TrimSpace(input.Model.APIKeyMasked),
			TokenThreshold: cloneIntPointer(input.Model.TokenThreshold),
			Params:         cloneMapAny(input.Model.Params),
		}
		if input.Model.Runtime != nil {
			item.Model.Runtime = &ModelRuntimeSpec{
				RequestTimeoutMS: cloneIntPointer(input.Model.Runtime.RequestTimeoutMS),
			}
		}
	}
	if input.Rule != nil {
		item.Rule = &RuleSpec{Content: input.Rule.Content}
	}
	if input.Skill != nil {
		item.Skill = &SkillSpec{Content: input.Skill.Content}
	}
	if input.MCP != nil {
		item.MCP = &McpSpec{
			Transport:       strings.TrimSpace(input.MCP.Transport),
			Endpoint:        strings.TrimSpace(input.MCP.Endpoint),
			Command:         strings.TrimSpace(input.MCP.Command),
			Env:             cloneStringMap(input.MCP.Env),
			Status:          strings.TrimSpace(input.MCP.Status),
			Tools:           append([]string{}, input.MCP.Tools...),
			LastError:       strings.TrimSpace(input.MCP.LastError),
			LastConnectedAt: strings.TrimSpace(input.MCP.LastConnectedAt),
		}
	}
	return item
}

func toDomainSessionResourceSnapshots(items []SessionResourceSnapshot) []domain.SessionResourceSnapshot {
	out := make([]domain.SessionResourceSnapshot, 0, len(items))
	for _, item := range items {
		out = append(out, domain.SessionResourceSnapshot{
			SessionID:          domain.SessionID(strings.TrimSpace(item.SessionID)),
			ResourceConfigID:   strings.TrimSpace(item.ResourceConfigID),
			ResourceType:       domain.ResourceType(strings.TrimSpace(string(item.ResourceType))),
			ResourceVersion:    item.ResourceVersion,
			IsDeprecated:       item.IsDeprecated,
			FallbackResourceID: cloneStringPointer(item.FallbackResourceID),
			SnapshotAt:         strings.TrimSpace(item.SnapshotAt),
			CapturedConfig:     toDomainResourceConfig(item.CapturedConfig),
		})
	}
	return out
}

func fromDomainSessionResourceSnapshots(items []domain.SessionResourceSnapshot) []SessionResourceSnapshot {
	out := make([]SessionResourceSnapshot, 0, len(items))
	for _, item := range items {
		out = append(out, SessionResourceSnapshot{
			SessionID:          strings.TrimSpace(string(item.SessionID)),
			ResourceConfigID:   strings.TrimSpace(item.ResourceConfigID),
			ResourceType:       ResourceType(strings.TrimSpace(string(item.ResourceType))),
			ResourceVersion:    item.ResourceVersion,
			IsDeprecated:       item.IsDeprecated,
			FallbackResourceID: cloneStringPointer(item.FallbackResourceID),
			SnapshotAt:         strings.TrimSpace(item.SnapshotAt),
			CapturedConfig:     fromDomainResourceConfig(item.CapturedConfig),
		})
	}
	return out
}

func toDomainSessionResourceState(input Conversation) domain.SessionResourceState {
	return domain.SessionResourceState{
		SessionID:     domain.SessionID(strings.TrimSpace(input.ID)),
		WorkspaceID:   domain.WorkspaceID(strings.TrimSpace(input.WorkspaceID)),
		ProjectID:     strings.TrimSpace(input.ProjectID),
		ModelConfigID: strings.TrimSpace(input.ModelConfigID),
		RuleIDs:       append([]string{}, input.RuleIDs...),
		SkillIDs:      append([]string{}, input.SkillIDs...),
		MCPIDs:        append([]string{}, input.MCPIDs...),
		UpdatedAt:     strings.TrimSpace(input.UpdatedAt),
	}
}

func fromDomainSessionResourceState(input domain.SessionResourceState, original Conversation) Conversation {
	updated := original
	updated.ID = strings.TrimSpace(string(input.SessionID))
	updated.WorkspaceID = strings.TrimSpace(string(input.WorkspaceID))
	updated.ProjectID = strings.TrimSpace(input.ProjectID)
	updated.ModelConfigID = strings.TrimSpace(input.ModelConfigID)
	updated.RuleIDs = append([]string{}, input.RuleIDs...)
	updated.SkillIDs = append([]string{}, input.SkillIDs...)
	updated.MCPIDs = append([]string{}, input.MCPIDs...)
	updated.UpdatedAt = strings.TrimSpace(input.UpdatedAt)
	return updated
}

func fromDomainResourceEvent(input domain.ResourceEvent) WorkspaceResourceEvent {
	return WorkspaceResourceEvent{
		EventID:         strings.TrimSpace(input.EventID),
		WorkspaceID:     strings.TrimSpace(string(input.WorkspaceID)),
		Type:            WorkspaceResourceEventType(strings.TrimSpace(string(input.Type))),
		ConfigID:        strings.TrimSpace(input.ConfigID),
		ConfigType:      ResourceType(strings.TrimSpace(string(input.ConfigType))),
		ResourceVersion: input.ResourceVersion,
		SessionID:       strings.TrimSpace(string(input.SessionID)),
		Timestamp:       strings.TrimSpace(input.Timestamp),
		Payload:         cloneMapAny(input.Payload),
	}
}

func cloneIntMap(input map[string]int) map[string]int {
	if len(input) == 0 {
		return map[string]int{}
	}
	out := make(map[string]int, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
