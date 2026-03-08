package domain

import (
	"context"
	"testing"
)

type resourceConfigRepositoryStub struct {
	configsByWorkspace map[WorkspaceID]map[string]ResourceConfig
	sessionSnapshots   map[SessionID][]SessionResourceSnapshot
}

func (s *resourceConfigRepositoryStub) GetResourceConfig(_ context.Context, workspaceID WorkspaceID, configID string) (ResourceConfig, bool, error) {
	if s == nil {
		return ResourceConfig{}, false, nil
	}
	items := s.configsByWorkspace[workspaceID]
	item, exists := items[configID]
	return item, exists, nil
}

func (s *resourceConfigRepositoryStub) ListSessionResourceSnapshots(_ context.Context, sessionID SessionID) ([]SessionResourceSnapshot, error) {
	if s == nil {
		return []SessionResourceSnapshot{}, nil
	}
	items := s.sessionSnapshots[sessionID]
	out := make([]SessionResourceSnapshot, 0, len(items))
	for _, item := range items {
		out = append(out, item)
	}
	return out, nil
}

func TestResourceConfigServiceValidateSessionSelectionRejectsDisallowedRule(t *testing.T) {
	repository := &resourceConfigRepositoryStub{
		configsByWorkspace: map[WorkspaceID]map[string]ResourceConfig{
			"ws_1": {
				"model_allowed": {ID: "model_allowed", WorkspaceID: "ws_1", Type: ResourceTypeModel, Enabled: true},
				"rule_disallowed": {ID: "rule_disallowed", WorkspaceID: "ws_1", Type: ResourceTypeRule, Enabled: true},
			},
		},
	}
	service := NewResourceConfigService(repository)

	err := service.ValidateSessionSelection(context.Background(), ValidateSessionSelectionRequest{
		WorkspaceID:   "ws_1",
		ProjectConfig: ProjectResourceConfig{ModelConfigIDs: []string{"model_allowed"}, RuleIDs: []string{"rule_allowed"}},
		ModelConfigID: "model_allowed",
		RuleIDs:       []string{"rule_disallowed"},
	})
	if err == nil {
		t.Fatal("expected validation error for disallowed rule")
	}
}

func TestResourceConfigServiceCaptureSessionSnapshotsIncludesAllSelectedResources(t *testing.T) {
	repository := &resourceConfigRepositoryStub{
		configsByWorkspace: map[WorkspaceID]map[string]ResourceConfig{
			"ws_1": {
				"model_1": {ID: "model_1", WorkspaceID: "ws_1", Type: ResourceTypeModel, Enabled: true, Version: 2},
				"rule_1":  {ID: "rule_1", WorkspaceID: "ws_1", Type: ResourceTypeRule, Enabled: true, Version: 3},
				"skill_1": {ID: "skill_1", WorkspaceID: "ws_1", Type: ResourceTypeSkill, Enabled: true, Version: 4},
				"mcp_1":   {ID: "mcp_1", WorkspaceID: "ws_1", Type: ResourceTypeMCP, Enabled: true, Version: 5},
			},
		},
	}
	service := NewResourceConfigService(repository)

	items, err := service.CaptureSessionSnapshots(context.Background(), CaptureSessionSnapshotsRequest{
		SessionID:      "sess_1",
		WorkspaceID:    "ws_1",
		ModelConfigID:  "model_1",
		RuleIDs:        []string{"rule_1"},
		SkillIDs:       []string{"skill_1"},
		MCPIDs:         []string{"mcp_1"},
		SnapshotAt:     "2026-03-08T10:00:00Z",
	})
	if err != nil {
		t.Fatalf("capture session snapshots failed: %v", err)
	}
	if len(items) != 4 {
		t.Fatalf("expected 4 captured resources, got %d", len(items))
	}
	if items[0].ResourceConfigID != "model_1" || items[0].ResourceVersion != 2 {
		t.Fatalf("unexpected model snapshot %#v", items[0])
	}
}

func TestResourceConfigServicePlanDeletedResourceMarksSnapshotsDeprecatedAndFallsBack(t *testing.T) {
	repository := &resourceConfigRepositoryStub{
		configsByWorkspace: map[WorkspaceID]map[string]ResourceConfig{
			"ws_1": {
				"model_deleted":  {ID: "model_deleted", WorkspaceID: "ws_1", Type: ResourceTypeModel, Enabled: false, Version: 6},
				"model_fallback": {ID: "model_fallback", WorkspaceID: "ws_1", Type: ResourceTypeModel, Enabled: true, Version: 7},
				"rule_1":         {ID: "rule_1", WorkspaceID: "ws_1", Type: ResourceTypeRule, Enabled: true, Version: 2},
			},
		},
		sessionSnapshots: map[SessionID][]SessionResourceSnapshot{
			"sess_1": {
				{
					SessionID:        "sess_1",
					ResourceConfigID: "model_deleted",
					ResourceType:     ResourceTypeModel,
					ResourceVersion:  6,
					SnapshotAt:       "2026-03-08T10:00:00Z",
					CapturedConfig:   ResourceConfig{ID: "model_deleted", WorkspaceID: "ws_1", Type: ResourceTypeModel, Enabled: false, Version: 6},
				},
				{
					SessionID:        "sess_1",
					ResourceConfigID: "rule_1",
					ResourceType:     ResourceTypeRule,
					ResourceVersion:  2,
					SnapshotAt:       "2026-03-08T10:00:00Z",
					CapturedConfig:   ResourceConfig{ID: "rule_1", WorkspaceID: "ws_1", Type: ResourceTypeRule, Enabled: true, Version: 2},
				},
			},
		},
	}
	service := NewResourceConfigService(repository)

	plans, err := service.PlanDeletedResource(context.Background(), PlanDeletedResourceRequest{
		WorkspaceID: "ws_1",
		DeletedConfig: ResourceConfig{
			ID:          "model_deleted",
			WorkspaceID: "ws_1",
			Type:        ResourceTypeModel,
			Version:     6,
		},
		AffectedSessions: []AffectedSessionResources{
			{
				Session: SessionResourceState{
					SessionID:     "sess_1",
					WorkspaceID:   "ws_1",
					ProjectID:     "proj_1",
					ModelConfigID: "model_deleted",
					RuleIDs:       []string{"rule_1"},
				},
				ProjectConfig: ProjectResourceConfig{
					ProjectID:            "proj_1",
					ModelConfigIDs:       []string{"model_deleted", "model_fallback"},
					DefaultModelConfigID: toStringPointer("model_fallback"),
					RuleIDs:              []string{"rule_1"},
				},
			},
		},
		Timestamp: "2026-03-08T11:00:00Z",
	})
	if err != nil {
		t.Fatalf("plan deleted resource failed: %v", err)
	}
	if len(plans) != 1 {
		t.Fatalf("expected one affected session plan, got %d", len(plans))
	}
	plan := plans[0]
	if plan.Session.ModelConfigID != "model_fallback" {
		t.Fatalf("expected fallback model, got %#v", plan.Session)
	}
	var deprecatedFound bool
	for _, snapshot := range plan.Snapshots {
		if snapshot.ResourceConfigID == "model_deleted" {
			deprecatedFound = snapshot.IsDeprecated
			if snapshot.FallbackResourceID == nil || *snapshot.FallbackResourceID != "model_fallback" {
				t.Fatalf("expected fallback snapshot metadata, got %#v", snapshot)
			}
		}
	}
	if !deprecatedFound {
		t.Fatalf("expected deleted model snapshot to be deprecated, got %#v", plan.Snapshots)
	}
	if plan.Event.Type != ResourceEventTypeSnapshotDeprecated {
		t.Fatalf("expected snapshot deprecated event, got %#v", plan.Event)
	}
}
