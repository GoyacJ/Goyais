package httpapi

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestSessionSubmitUsesSnapshottedResourcesAfterWorkspaceResourceUpdate(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	})

	now := time.Now().UTC().Format(time.RFC3339)
	projectID := "proj_snapshot_runtime"
	modelConfigID := "rc_model_snapshot_runtime"
	ruleID := "rc_rule_snapshot_runtime"

	state := newSessionResourceSnapshotTestState(store, "run_snapshot_runtime_create")
	if _, err := saveProjectToStore(state, Project{
		ID:          projectID,
		WorkspaceID: localWorkspaceID,
		Name:        "Snapshot Runtime Project",
		RepoPath:    t.TempDir(),
		DefaultMode: PermissionModeDefault,
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("save project failed: %v", err)
	}
	if _, err := saveWorkspaceResourceConfig(state, ResourceConfig{
		ID:          modelConfigID,
		WorkspaceID: localWorkspaceID,
		Type:        ResourceTypeModel,
		Enabled:     true,
		Model: &ModelSpec{
			Vendor:  ModelVendorOpenAI,
			ModelID: "gpt-5.1",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("save model config failed: %v", err)
	}
	if _, err := saveWorkspaceResourceConfig(state, ResourceConfig{
		ID:          ruleID,
		WorkspaceID: localWorkspaceID,
		Type:        ResourceTypeRule,
		Name:        "Snapshot Rule",
		Enabled:     true,
		Rule: &RuleSpec{
			Content: "always answer briefly",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("save rule config failed: %v", err)
	}
	if _, err := saveProjectConfigToStore(state, localWorkspaceID, ProjectConfig{
		ProjectID:            projectID,
		ModelConfigIDs:       []string{modelConfigID},
		DefaultModelConfigID: toStringPtr(modelConfigID),
		ModelTokenThresholds: map[string]int{},
		RuleIDs:              []string{ruleID},
		SkillIDs:             []string{},
		MCPIDs:               []string{},
		UpdatedAt:            now,
	}); err != nil {
		t.Fatalf("save project config failed: %v", err)
	}

	mux := newSessionResourceSnapshotTestMux(state)
	createRes := performJSONRequest(t, mux, http.MethodPost, "/v1/projects/"+projectID+"/sessions", map[string]any{
		"name": "Snapshot Runtime Session",
	}, nil)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected create session 201, got %d (%s)", createRes.Code, createRes.Body.String())
	}
	createPayload := map[string]any{}
	mustDecodeJSON(t, createRes.Body.Bytes(), &createPayload)
	sessionID := strings.TrimSpace(asString(createPayload["id"]))
	if sessionID == "" {
		t.Fatalf("expected session id in create payload")
	}

	updatedModel, exists, err := loadWorkspaceResourceConfigRaw(state, localWorkspaceID, modelConfigID)
	if err != nil || !exists {
		t.Fatalf("load updated model config failed: exists=%v err=%v", exists, err)
	}
	updatedModel.Model.ModelID = "gpt-5.2"
	updatedModel.UpdatedAt = time.Now().UTC().Add(time.Second).Format(time.RFC3339)
	if _, err := saveWorkspaceResourceConfig(state, updatedModel); err != nil {
		t.Fatalf("update model config failed: %v", err)
	}

	updatedRule, exists, err := loadWorkspaceResourceConfigRaw(state, localWorkspaceID, ruleID)
	if err != nil || !exists {
		t.Fatalf("load updated rule config failed: exists=%v err=%v", exists, err)
	}
	updatedRule.Rule.Content = "always answer with a long explanation"
	updatedRule.UpdatedAt = time.Now().UTC().Add(2 * time.Second).Format(time.RFC3339)
	if _, err := saveWorkspaceResourceConfig(state, updatedRule); err != nil {
		t.Fatalf("update rule config failed: %v", err)
	}

	restartedState := newSessionResourceSnapshotTestState(store, "run_snapshot_runtime_submit")
	restartedMux := newSessionResourceSnapshotTestMux(restartedState)

	submitRes := performJSONRequest(t, restartedMux, http.MethodPost, "/v1/sessions/"+sessionID+"/runs", map[string]any{
		"raw_input": "describe the current defaults",
	}, nil)
	if submitRes.Code != http.StatusCreated {
		t.Fatalf("expected submit run 201, got %d (%s)", submitRes.Code, submitRes.Body.String())
	}
	submitPayload := map[string]any{}
	mustDecodeJSON(t, submitRes.Body.Bytes(), &submitPayload)
	runPayload, ok := submitPayload["run"].(map[string]any)
	if !ok {
		t.Fatalf("expected run payload, got %#v", submitPayload["run"])
	}
	if got := strings.TrimSpace(asString(runPayload["model_id"])); got != "gpt-5.1" {
		t.Fatalf("expected snapshotted model_id gpt-5.1, got %q", got)
	}
	resourceProfile, ok := runPayload["resource_profile_snapshot"].(map[string]any)
	if !ok {
		t.Fatalf("expected resource_profile_snapshot, got %#v", runPayload["resource_profile_snapshot"])
	}
	if got := strings.TrimSpace(asString(resourceProfile["rules_dsl"])); got != "always answer briefly" {
		t.Fatalf("expected snapshotted rules_dsl, got %q", got)
	}

	detailRes := performJSONRequest(t, restartedMux, http.MethodGet, "/v1/sessions/"+sessionID, nil, nil)
	if detailRes.Code != http.StatusOK {
		t.Fatalf("expected session detail 200, got %d (%s)", detailRes.Code, detailRes.Body.String())
	}
	detailPayload := map[string]any{}
	mustDecodeJSON(t, detailRes.Body.Bytes(), &detailPayload)
	resourceSnapshots, ok := detailPayload["resource_snapshots"].([]any)
	if !ok {
		t.Fatalf("expected resource_snapshots array, got %#v", detailPayload["resource_snapshots"])
	}
	assertSessionResourceSnapshot(t, resourceSnapshots, modelConfigID, string(ResourceTypeModel), 1, false, "")
	assertSessionResourceSnapshot(t, resourceSnapshots, ruleID, string(ResourceTypeRule), 1, false, "")
}

func TestDeleteModelResourceConfigFallsBackSessionAndDeprecatesSnapshot(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	})

	now := time.Now().UTC().Format(time.RFC3339)
	projectID := "proj_snapshot_fallback"
	legacyModelID := "rc_model_legacy"
	fallbackModelID := "rc_model_fallback"

	state := newSessionResourceSnapshotTestState(store, "run_snapshot_fallback_create")
	if _, err := saveProjectToStore(state, Project{
		ID:          projectID,
		WorkspaceID: localWorkspaceID,
		Name:        "Snapshot Fallback Project",
		RepoPath:    t.TempDir(),
		DefaultMode: PermissionModeDefault,
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("save project failed: %v", err)
	}
	if _, err := saveWorkspaceResourceConfig(state, ResourceConfig{
		ID:          legacyModelID,
		WorkspaceID: localWorkspaceID,
		Type:        ResourceTypeModel,
		Enabled:     true,
		Model: &ModelSpec{
			Vendor:  ModelVendorOpenAI,
			ModelID: "gpt-5.legacy",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("save legacy model config failed: %v", err)
	}
	if _, err := saveWorkspaceResourceConfig(state, ResourceConfig{
		ID:          fallbackModelID,
		WorkspaceID: localWorkspaceID,
		Type:        ResourceTypeModel,
		Enabled:     true,
		Model: &ModelSpec{
			Vendor:  ModelVendorOpenAI,
			ModelID: "gpt-5.safe",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("save fallback model config failed: %v", err)
	}
	if _, err := saveProjectConfigToStore(state, localWorkspaceID, ProjectConfig{
		ProjectID:            projectID,
		ModelConfigIDs:       []string{legacyModelID, fallbackModelID},
		DefaultModelConfigID: toStringPtr(fallbackModelID),
		ModelTokenThresholds: map[string]int{},
		RuleIDs:              []string{},
		SkillIDs:             []string{},
		MCPIDs:               []string{},
		UpdatedAt:            now,
	}); err != nil {
		t.Fatalf("save project config failed: %v", err)
	}

	mux := newSessionResourceSnapshotTestMux(state)
	createRes := performJSONRequest(t, mux, http.MethodPost, "/v1/projects/"+projectID+"/sessions", map[string]any{
		"name": "Snapshot Fallback Session",
	}, nil)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected create session 201, got %d (%s)", createRes.Code, createRes.Body.String())
	}
	createPayload := map[string]any{}
	mustDecodeJSON(t, createRes.Body.Bytes(), &createPayload)
	sessionID := strings.TrimSpace(asString(createPayload["id"]))
	if sessionID == "" {
		t.Fatalf("expected session id in create payload")
	}

	patchRes := performJSONRequest(t, mux, http.MethodPatch, "/v1/sessions/"+sessionID, map[string]any{
		"model_config_id": legacyModelID,
	}, nil)
	if patchRes.Code != http.StatusOK {
		t.Fatalf("expected patch session 200, got %d (%s)", patchRes.Code, patchRes.Body.String())
	}

	deleteRes := performJSONRequest(t, mux, http.MethodDelete, "/v1/workspaces/"+localWorkspaceID+"/resource-configs/"+legacyModelID, nil, nil)
	if deleteRes.Code != http.StatusNoContent {
		t.Fatalf("expected delete resource config 204, got %d (%s)", deleteRes.Code, deleteRes.Body.String())
	}
	if !containsWorkspaceResourceEvent(state.workspaceResourceEvents[localWorkspaceID], WorkspaceResourceEventTypeSnapshotDeprecated, legacyModelID, sessionID) {
		t.Fatalf("expected snapshot deprecated workspace event for %s", legacyModelID)
	}

	restartedState := newSessionResourceSnapshotTestState(store, "run_snapshot_fallback_submit")
	restartedMux := newSessionResourceSnapshotTestMux(restartedState)

	detailRes := performJSONRequest(t, restartedMux, http.MethodGet, "/v1/sessions/"+sessionID, nil, nil)
	if detailRes.Code != http.StatusOK {
		t.Fatalf("expected session detail 200, got %d (%s)", detailRes.Code, detailRes.Body.String())
	}
	detailPayload := map[string]any{}
	mustDecodeJSON(t, detailRes.Body.Bytes(), &detailPayload)
	sessionPayload, ok := detailPayload["conversation"].(map[string]any)
	if !ok {
		sessionPayload, ok = detailPayload["session"].(map[string]any)
		if !ok {
			t.Fatalf("expected session object, got %#v", detailPayload["session"])
		}
	}
	if got := strings.TrimSpace(asString(sessionPayload["model_config_id"])); got != fallbackModelID {
		t.Fatalf("expected session model fallback %s, got %q", fallbackModelID, got)
	}

	resourceSnapshots, ok := detailPayload["resource_snapshots"].([]any)
	if !ok {
		t.Fatalf("expected resource_snapshots array, got %#v", detailPayload["resource_snapshots"])
	}
	assertSessionResourceSnapshot(t, resourceSnapshots, legacyModelID, string(ResourceTypeModel), 1, true, fallbackModelID)
	assertSessionResourceSnapshot(t, resourceSnapshots, fallbackModelID, string(ResourceTypeModel), 1, false, "")

	submitRes := performJSONRequest(t, restartedMux, http.MethodPost, "/v1/sessions/"+sessionID+"/runs", map[string]any{
		"raw_input": "use the fallback model",
	}, nil)
	if submitRes.Code != http.StatusCreated {
		t.Fatalf("expected submit run 201, got %d (%s)", submitRes.Code, submitRes.Body.String())
	}
	submitPayload := map[string]any{}
	mustDecodeJSON(t, submitRes.Body.Bytes(), &submitPayload)
	runPayload, ok := submitPayload["run"].(map[string]any)
	if !ok {
		t.Fatalf("expected run payload, got %#v", submitPayload["run"])
	}
	if got := strings.TrimSpace(asString(runPayload["model_id"])); got != "gpt-5.safe" {
		t.Fatalf("expected fallback model_id gpt-5.safe, got %q", got)
	}
}

func TestDeleteToolingResourceConfigRemovesSessionSelectionsAndDeprecatesSnapshots(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	})

	now := time.Now().UTC().Format(time.RFC3339)
	projectID := "proj_snapshot_tooling"
	modelConfigID := "rc_model_tooling"
	ruleID := "rc_rule_tooling"
	skillID := "rc_skill_tooling"
	mcpID := "rc_mcp_tooling"

	state := newSessionResourceSnapshotTestState(store, "run_snapshot_tooling_create")
	if _, err := saveProjectToStore(state, Project{
		ID:          projectID,
		WorkspaceID: localWorkspaceID,
		Name:        "Snapshot Tooling Project",
		RepoPath:    t.TempDir(),
		DefaultMode: PermissionModeDefault,
		CreatedAt:   now,
		UpdatedAt:   now,
	}); err != nil {
		t.Fatalf("save project failed: %v", err)
	}
	if _, err := saveWorkspaceResourceConfig(state, ResourceConfig{
		ID:          modelConfigID,
		WorkspaceID: localWorkspaceID,
		Type:        ResourceTypeModel,
		Enabled:     true,
		Model: &ModelSpec{
			Vendor:  ModelVendorOpenAI,
			ModelID: "gpt-5.tooling",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("save model config failed: %v", err)
	}
	if _, err := saveWorkspaceResourceConfig(state, ResourceConfig{
		ID:          ruleID,
		WorkspaceID: localWorkspaceID,
		Type:        ResourceTypeRule,
		Name:        "Tooling Rule",
		Enabled:     true,
		Rule: &RuleSpec{
			Content: "prefer concise output",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("save rule config failed: %v", err)
	}
	if _, err := saveWorkspaceResourceConfig(state, ResourceConfig{
		ID:          skillID,
		WorkspaceID: localWorkspaceID,
		Type:        ResourceTypeSkill,
		Name:        "Tooling Skill",
		Enabled:     true,
		Skill: &SkillSpec{
			Content: "tooling skill content",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("save skill config failed: %v", err)
	}
	if _, err := saveWorkspaceResourceConfig(state, ResourceConfig{
		ID:          mcpID,
		WorkspaceID: localWorkspaceID,
		Type:        ResourceTypeMCP,
		Name:        "Tooling MCP",
		Enabled:     true,
		MCP: &McpSpec{
			Transport: "stdio",
			Command:   "tooling-mcp",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("save mcp config failed: %v", err)
	}
	if _, err := saveProjectConfigToStore(state, localWorkspaceID, ProjectConfig{
		ProjectID:            projectID,
		ModelConfigIDs:       []string{modelConfigID},
		DefaultModelConfigID: toStringPtr(modelConfigID),
		ModelTokenThresholds: map[string]int{},
		RuleIDs:              []string{ruleID},
		SkillIDs:             []string{skillID},
		MCPIDs:               []string{mcpID},
		UpdatedAt:            now,
	}); err != nil {
		t.Fatalf("save project config failed: %v", err)
	}

	mux := newSessionResourceSnapshotTestMux(state)
	createRes := performJSONRequest(t, mux, http.MethodPost, "/v1/projects/"+projectID+"/sessions", map[string]any{
		"name": "Snapshot Tooling Session",
	}, nil)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected create session 201, got %d (%s)", createRes.Code, createRes.Body.String())
	}
	createPayload := map[string]any{}
	mustDecodeJSON(t, createRes.Body.Bytes(), &createPayload)
	sessionID := strings.TrimSpace(asString(createPayload["id"]))
	if sessionID == "" {
		t.Fatalf("expected session id in create payload")
	}

	for _, configID := range []string{ruleID, skillID, mcpID} {
		deleteRes := performJSONRequest(t, mux, http.MethodDelete, "/v1/workspaces/"+localWorkspaceID+"/resource-configs/"+configID, nil, nil)
		if deleteRes.Code != http.StatusNoContent {
			t.Fatalf("expected delete resource config %s to return 204, got %d (%s)", configID, deleteRes.Code, deleteRes.Body.String())
		}
	}

	restartedState := newSessionResourceSnapshotTestState(store, "run_snapshot_tooling_submit")
	restartedMux := newSessionResourceSnapshotTestMux(restartedState)

	detailRes := performJSONRequest(t, restartedMux, http.MethodGet, "/v1/sessions/"+sessionID, nil, nil)
	if detailRes.Code != http.StatusOK {
		t.Fatalf("expected session detail 200, got %d (%s)", detailRes.Code, detailRes.Body.String())
	}
	detailPayload := map[string]any{}
	mustDecodeJSON(t, detailRes.Body.Bytes(), &detailPayload)
	sessionPayload, ok := detailPayload["conversation"].(map[string]any)
	if !ok {
		sessionPayload, ok = detailPayload["session"].(map[string]any)
		if !ok {
			t.Fatalf("expected session object, got %#v", detailPayload["session"])
		}
	}
	if got := sessionStringSlice(sessionPayload["rule_ids"]); len(got) != 0 {
		t.Fatalf("expected empty rule_ids after delete, got %#v", got)
	}
	if got := sessionStringSlice(sessionPayload["skill_ids"]); len(got) != 0 {
		t.Fatalf("expected empty skill_ids after delete, got %#v", got)
	}
	if got := sessionStringSlice(sessionPayload["mcp_ids"]); len(got) != 0 {
		t.Fatalf("expected empty mcp_ids after delete, got %#v", got)
	}

	resourceSnapshots, ok := detailPayload["resource_snapshots"].([]any)
	if !ok {
		t.Fatalf("expected resource_snapshots array, got %#v", detailPayload["resource_snapshots"])
	}
	assertSessionResourceSnapshot(t, resourceSnapshots, ruleID, string(ResourceTypeRule), 1, true, "")
	assertSessionResourceSnapshot(t, resourceSnapshots, skillID, string(ResourceTypeSkill), 1, true, "")
	assertSessionResourceSnapshot(t, resourceSnapshots, mcpID, string(ResourceTypeMCP), 1, true, "")

	submitRes := performJSONRequest(t, restartedMux, http.MethodPost, "/v1/sessions/"+sessionID+"/runs", map[string]any{
		"raw_input": "submit after tooling deletion",
	}, nil)
	if submitRes.Code != http.StatusCreated {
		t.Fatalf("expected submit run 201, got %d (%s)", submitRes.Code, submitRes.Body.String())
	}
	submitPayload := map[string]any{}
	mustDecodeJSON(t, submitRes.Body.Bytes(), &submitPayload)
	runPayload, ok := submitPayload["run"].(map[string]any)
	if !ok {
		t.Fatalf("expected run payload, got %#v", submitPayload["run"])
	}
	resourceProfile, ok := runPayload["resource_profile_snapshot"].(map[string]any)
	if !ok {
		t.Fatalf("expected resource_profile_snapshot, got %#v", runPayload["resource_profile_snapshot"])
	}
	if got := strings.TrimSpace(asString(resourceProfile["rules_dsl"])); got != "" {
		t.Fatalf("expected empty rules_dsl after rule delete, got %q", got)
	}
	if got := sessionStringSlice(resourceProfile["mcp_ids"]); len(got) != 0 {
		t.Fatalf("expected empty mcp_ids after delete, got %#v", got)
	}
}

func newSessionResourceSnapshotTestState(store *authzStore, runID string) *AppState {
	state := NewAppState(store)
	state.runtimeService = &runtimeBridgeServiceStub{runID: runID}
	state.runtimeEngine = &runtimeEngineSubscribeStub{}
	return state
}

func newSessionResourceSnapshotTestMux(state *AppState) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/projects/{project_id}/sessions", ProjectConversationsHandler(state))
	mux.HandleFunc("/v1/sessions/{session_id}", ConversationByIDHandler(state))
	mux.HandleFunc("/v1/sessions/{session_id}/runs", ConversationInputSubmitHandler(state))
	mux.HandleFunc("/v1/workspaces/{workspace_id}/resource-configs/{config_id}", ResourceConfigByIDHandler(state))
	return mux
}

func assertSessionResourceSnapshot(
	t *testing.T,
	items []any,
	resourceConfigID string,
	resourceType string,
	resourceVersion int,
	isDeprecated bool,
	fallbackResourceID string,
) {
	t.Helper()
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if strings.TrimSpace(asString(item["resource_config_id"])) != strings.TrimSpace(resourceConfigID) {
			continue
		}
		if got := strings.TrimSpace(asString(item["resource_type"])); got != strings.TrimSpace(resourceType) {
			t.Fatalf("expected resource_type %s for %s, got %q", resourceType, resourceConfigID, got)
		}
		if got := int(item["resource_version"].(float64)); got != resourceVersion {
			t.Fatalf("expected resource_version %d for %s, got %d", resourceVersion, resourceConfigID, got)
		}
		if got := item["is_deprecated"].(bool); got != isDeprecated {
			t.Fatalf("expected is_deprecated=%v for %s, got %v", isDeprecated, resourceConfigID, got)
		}
		if got := strings.TrimSpace(asString(item["fallback_resource_id"])); got != strings.TrimSpace(fallbackResourceID) {
			t.Fatalf("expected fallback_resource_id %q for %s, got %q", fallbackResourceID, resourceConfigID, got)
		}
		return
	}
	t.Fatalf("expected resource snapshot for %s in %#v", resourceConfigID, items)
}

func sessionStringSlice(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, strings.TrimSpace(asString(item)))
	}
	return out
}

func containsWorkspaceResourceEvent(
	items []WorkspaceResourceEvent,
	eventType WorkspaceResourceEventType,
	configID string,
	sessionID string,
) bool {
	for _, item := range items {
		if item.Type != eventType {
			continue
		}
		if strings.TrimSpace(item.ConfigID) != strings.TrimSpace(configID) {
			continue
		}
		if strings.TrimSpace(item.SessionID) != strings.TrimSpace(sessionID) {
			continue
		}
		return true
	}
	return false
}
