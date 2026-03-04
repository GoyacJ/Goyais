package httpapi

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestProjectConfigHandlerPutPurgesProjectConversationHistory(t *testing.T) {
	state := NewAppState(nil)
	now := time.Now().UTC().Format(time.RFC3339)

	workspaceID := localWorkspaceID
	targetProjectID := "proj_target"
	otherProjectID := "proj_other"
	targetConversationID := "conv_target"
	otherConversationID := "conv_other"
	targetExecutionID := "exec_target"
	otherExecutionID := "exec_other"

	state.projects[targetProjectID] = Project{
		ID:          targetProjectID,
		WorkspaceID: workspaceID,
		Name:        "Target",
		RepoPath:    "/tmp/target",
		DefaultMode: PermissionModeDefault,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	state.projects[otherProjectID] = Project{
		ID:          otherProjectID,
		WorkspaceID: workspaceID,
		Name:        "Other",
		RepoPath:    "/tmp/other",
		DefaultMode: PermissionModeDefault,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	oldModelID := "rc_model_old"
	newModelID := "rc_model_new"
	otherModelID := "rc_model_other"

	if _, err := saveWorkspaceResourceConfig(state, ResourceConfig{
		ID:          oldModelID,
		WorkspaceID: workspaceID,
		Type:        ResourceTypeModel,
		Enabled:     true,
		Model: &ModelSpec{
			Vendor:  ModelVendorOpenAI,
			ModelID: "gpt-5.3",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("save old model config failed: %v", err)
	}
	if _, err := saveWorkspaceResourceConfig(state, ResourceConfig{
		ID:          newModelID,
		WorkspaceID: workspaceID,
		Type:        ResourceTypeModel,
		Enabled:     true,
		Model: &ModelSpec{
			Vendor:  ModelVendorOpenAI,
			ModelID: "gpt-5.4",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("save new model config failed: %v", err)
	}
	if _, err := saveWorkspaceResourceConfig(state, ResourceConfig{
		ID:          otherModelID,
		WorkspaceID: workspaceID,
		Type:        ResourceTypeModel,
		Enabled:     true,
		Model: &ModelSpec{
			Vendor:  ModelVendorOpenAI,
			ModelID: "gpt-4.1",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}); err != nil {
		t.Fatalf("save other model config failed: %v", err)
	}

	state.projectConfigs[targetProjectID] = ProjectConfig{
		ProjectID:            targetProjectID,
		ModelConfigIDs:       []string{oldModelID},
		DefaultModelConfigID: &oldModelID,
		RuleIDs:              []string{},
		SkillIDs:             []string{},
		MCPIDs:               []string{},
		UpdatedAt:            now,
	}
	state.projectConfigs[otherProjectID] = ProjectConfig{
		ProjectID:            otherProjectID,
		ModelConfigIDs:       []string{otherModelID},
		DefaultModelConfigID: &otherModelID,
		RuleIDs:              []string{},
		SkillIDs:             []string{},
		MCPIDs:               []string{},
		UpdatedAt:            now,
	}

	state.conversations[targetConversationID] = Conversation{
		ID:            targetConversationID,
		WorkspaceID:   workspaceID,
		ProjectID:     targetProjectID,
		Name:          "Target Conversation",
		QueueState:    QueueStateRunning,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: oldModelID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.conversations[otherConversationID] = Conversation{
		ID:            otherConversationID,
		WorkspaceID:   workspaceID,
		ProjectID:     otherProjectID,
		Name:          "Other Conversation",
		QueueState:    QueueStateRunning,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: otherModelID,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	state.conversationMessages[targetConversationID] = []ConversationMessage{{
		ID:             "msg_target",
		ConversationID: targetConversationID,
		Role:           MessageRoleUser,
		Content:        "target message",
		CreatedAt:      now,
	}}
	state.conversationMessages[otherConversationID] = []ConversationMessage{{
		ID:             "msg_other",
		ConversationID: otherConversationID,
		Role:           MessageRoleUser,
		Content:        "other message",
		CreatedAt:      now,
	}}

	state.conversationSnapshots[targetConversationID] = []ConversationSnapshot{{
		ID:                     "snap_target",
		ConversationID:         targetConversationID,
		RollbackPointMessageID: "msg_target",
		QueueState:             QueueStateRunning,
		InspectorState:         ConversationInspector{Tab: "diff"},
		Messages:               []ConversationMessage{{ID: "msg_target", ConversationID: targetConversationID, Role: MessageRoleUser, Content: "target message", CreatedAt: now}},
		ExecutionIDs:           []string{targetExecutionID},
		CreatedAt:              now,
	}}
	state.conversationSnapshots[otherConversationID] = []ConversationSnapshot{{
		ID:                     "snap_other",
		ConversationID:         otherConversationID,
		RollbackPointMessageID: "msg_other",
		QueueState:             QueueStateRunning,
		InspectorState:         ConversationInspector{Tab: "diff"},
		Messages:               []ConversationMessage{{ID: "msg_other", ConversationID: otherConversationID, Role: MessageRoleUser, Content: "other message", CreatedAt: now}},
		ExecutionIDs:           []string{otherExecutionID},
		CreatedAt:              now,
	}}

	state.conversationExecutionOrder[targetConversationID] = []string{targetExecutionID}
	state.conversationExecutionOrder[otherConversationID] = []string{otherExecutionID}
	state.executionEvents[targetConversationID] = []ExecutionEvent{{
		EventID:        "evt_target",
		ConversationID: targetConversationID,
		ExecutionID:    targetExecutionID,
		Type:           RunEventTypeExecutionStarted,
		Timestamp:      now,
	}}
	state.executionEvents[otherConversationID] = []ExecutionEvent{{
		EventID:        "evt_other",
		ConversationID: otherConversationID,
		ExecutionID:    otherExecutionID,
		Type:           RunEventTypeExecutionStarted,
		Timestamp:      now,
	}}
	state.conversationEventSeq[targetConversationID] = 7
	state.conversationEventSeq[otherConversationID] = 3

	targetSub := make(chan ExecutionEvent, 1)
	state.conversationEventSubs[targetConversationID] = map[string]chan ExecutionEvent{"sub_target": targetSub}
	state.conversationEventSubs[otherConversationID] = map[string]chan ExecutionEvent{"sub_other": make(chan ExecutionEvent, 1)}

	state.executions[targetExecutionID] = Execution{
		ID:             targetExecutionID,
		WorkspaceID:    workspaceID,
		ConversationID: targetConversationID,
		State:          RunStateExecuting,
		Mode:           PermissionModeDefault,
		ModelID:        oldModelID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.executions[otherExecutionID] = Execution{
		ID:             otherExecutionID,
		WorkspaceID:    workspaceID,
		ConversationID: otherConversationID,
		State:          RunStateExecuting,
		Mode:           PermissionModeDefault,
		ModelID:        otherModelID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.executionDiffs[targetExecutionID] = []DiffItem{{ID: "diff_target", Path: "a.txt", ChangeType: "modified"}}
	state.executionDiffs[otherExecutionID] = []DiffItem{{ID: "diff_other", Path: "b.txt", ChangeType: "modified"}}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/projects/{project_id}/config", ProjectConfigHandler(state))

	putRes := performJSONRequest(t, mux, http.MethodPut, "/v1/projects/"+targetProjectID+"/config", map[string]any{
		"project_id":              targetProjectID,
		"model_config_ids":        []string{newModelID},
		"default_model_config_id": newModelID,
		"rule_ids":                []string{},
		"skill_ids":               []string{},
		"mcp_ids":                 []string{},
		"updated_at":              "",
	}, nil)
	if putRes.Code != http.StatusOK {
		t.Fatalf("expected put project config 200, got %d (%s)", putRes.Code, putRes.Body.String())
	}

	updatedConfig, ok := state.projectConfigs[targetProjectID]
	if !ok {
		t.Fatalf("expected updated project config to exist")
	}
	if got := derefString(updatedConfig.DefaultModelConfigID); got != newModelID {
		t.Fatalf("expected updated default_model_config_id %s, got %s", newModelID, got)
	}

	if _, exists := state.conversations[targetConversationID]; exists {
		t.Fatalf("expected target conversation removed")
	}
	if _, exists := state.conversationMessages[targetConversationID]; exists {
		t.Fatalf("expected target conversation messages removed")
	}
	if _, exists := state.conversationSnapshots[targetConversationID]; exists {
		t.Fatalf("expected target conversation snapshots removed")
	}
	if _, exists := state.conversationExecutionOrder[targetConversationID]; exists {
		t.Fatalf("expected target execution order removed")
	}
	if _, exists := state.executionEvents[targetConversationID]; exists {
		t.Fatalf("expected target execution events removed")
	}
	if _, exists := state.conversationEventSeq[targetConversationID]; exists {
		t.Fatalf("expected target event seq removed")
	}
	if _, exists := state.conversationEventSubs[targetConversationID]; exists {
		t.Fatalf("expected target subscribers removed")
	}
	if _, exists := state.executions[targetExecutionID]; exists {
		t.Fatalf("expected target execution removed")
	}
	if _, exists := state.executionDiffs[targetExecutionID]; exists {
		t.Fatalf("expected target execution diff removed")
	}

	if _, exists := state.conversations[otherConversationID]; !exists {
		t.Fatalf("expected other project conversation preserved")
	}
	if _, exists := state.executions[otherExecutionID]; !exists {
		t.Fatalf("expected other project execution preserved")
	}
	if _, exists := state.conversationEventSubs[otherConversationID]; !exists {
		t.Fatalf("expected other project subscribers preserved")
	}

	select {
	case _, ok := <-targetSub:
		if ok {
			t.Fatalf("expected target subscriber channel closed")
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("expected target subscriber channel to close")
	}
}

func TestProjectConversationsHandlerPostDoesNotFallbackToCatalogDefaultModel(t *testing.T) {
	state := NewAppState(nil)
	now := time.Now().UTC().Format(time.RFC3339)
	projectID := "proj_no_default_model"
	workspaceID := localWorkspaceID

	state.projects[projectID] = Project{
		ID:          projectID,
		WorkspaceID: workspaceID,
		Name:        "No Default Model Project",
		RepoPath:    "/tmp/no-default",
		DefaultMode: PermissionModeDefault,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	state.projectConfigs[projectID] = ProjectConfig{
		ProjectID:      projectID,
		ModelConfigIDs: []string{},
		RuleIDs:        []string{},
		SkillIDs:       []string{},
		MCPIDs:         []string{},
		UpdatedAt:      now,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/projects/{project_id}/sessions", ProjectConversationsHandler(state))
	res := performJSONRequest(t, mux, http.MethodPost, "/v1/projects/"+projectID+"/sessions", map[string]any{
		"name": "Conversation without model",
	}, nil)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected create conversation 201, got %d (%s)", res.Code, res.Body.String())
	}

	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	if got := strings.TrimSpace(asString(payload["model_config_id"])); got != "" {
		t.Fatalf("expected empty model_config_id when project has no default, got %q", got)
	}
}

func TestProjectConversationsHandlerGetIncludesTokenUsageTotals(t *testing.T) {
	state := NewAppState(nil)
	now := time.Now().UTC().Format(time.RFC3339)
	projectID := "proj_usage_sidebar"
	conversationID := "conv_usage_sidebar"
	executionOneID := "exec_usage_sidebar_1"
	executionTwoID := "exec_usage_sidebar_2"

	state.projects[projectID] = Project{
		ID:          projectID,
		WorkspaceID: localWorkspaceID,
		Name:        "Usage Sidebar",
		RepoPath:    "/tmp/usage-sidebar",
		DefaultMode: PermissionModeDefault,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	state.projectConfigs[projectID] = ProjectConfig{
		ProjectID:      projectID,
		ModelConfigIDs: []string{},
		RuleIDs:        []string{},
		SkillIDs:       []string{},
		MCPIDs:         []string{},
		UpdatedAt:      now,
	}
	state.conversations[conversationID] = Conversation{
		ID:            conversationID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     projectID,
		Name:          "Conversation Usage",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_1",
		RuleIDs:       []string{},
		SkillIDs:      []string{},
		MCPIDs:        []string{},
		BaseRevision:  0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.conversationExecutionOrder[conversationID] = []string{executionOneID, executionTwoID}
	state.executions[executionOneID] = Execution{
		ID:             executionOneID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_usage_sidebar_1",
		State:          RunStateCompleted,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot: ModelSnapshot{
			ModelID: "gpt-5",
		},
		TokensIn:                10,
		TokensOut:               20,
		ProjectRevisionSnapshot: 0,
		QueueIndex:              1,
		TraceID:                 "tr_usage_sidebar_1",
		CreatedAt:               now,
		UpdatedAt:               now,
	}
	state.executions[executionTwoID] = Execution{
		ID:             executionTwoID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_usage_sidebar_2",
		State:          RunStateCompleted,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot: ModelSnapshot{
			ModelID: "gpt-5",
		},
		TokensIn:                3,
		TokensOut:               7,
		ProjectRevisionSnapshot: 0,
		QueueIndex:              2,
		TraceID:                 "tr_usage_sidebar_2",
		CreatedAt:               now,
		UpdatedAt:               now,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/projects/{project_id}/sessions", ProjectConversationsHandler(state))
	res := performJSONRequest(t, mux, http.MethodGet, "/v1/projects/"+projectID+"/sessions", nil, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("expected list conversations 200, got %d (%s)", res.Code, res.Body.String())
	}

	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	items := payload["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected one conversation, got %d", len(items))
	}
	conversation := items[0].(map[string]any)
	if got := int(conversation["tokens_in_total"].(float64)); got != 13 {
		t.Fatalf("expected tokens_in_total 13, got %d", got)
	}
	if got := int(conversation["tokens_out_total"].(float64)); got != 27 {
		t.Fatalf("expected tokens_out_total 27, got %d", got)
	}
	if got := int(conversation["tokens_total"].(float64)); got != 40 {
		t.Fatalf("expected tokens_total 40, got %d", got)
	}
}

func TestProjectConversationsHandlerGetUsesRepositoryWhenConversationMapMissing(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	}()

	state := NewAppState(store)
	now := time.Now().UTC().Format(time.RFC3339)
	projectID := "proj_usage_sidebar_repo_" + randomHex(4)
	conversationID := "conv_usage_sidebar_repo_" + randomHex(4)
	executionID := "exec_usage_sidebar_repo_" + randomHex(4)

	project := Project{
		ID:          projectID,
		WorkspaceID: localWorkspaceID,
		Name:        "Usage Sidebar Repository",
		RepoPath:    "/tmp/usage-sidebar-repository",
		DefaultMode: PermissionModeDefault,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if _, err := saveProjectToStore(state, project); err != nil {
		t.Fatalf("save project failed: %v", err)
	}
	if _, err := saveProjectConfigToStore(state, localWorkspaceID, ProjectConfig{
		ProjectID:      projectID,
		ModelConfigIDs: []string{},
		RuleIDs:        []string{},
		SkillIDs:       []string{},
		MCPIDs:         []string{},
		UpdatedAt:      now,
	}); err != nil {
		t.Fatalf("save project config failed: %v", err)
	}

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:            conversationID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     projectID,
		Name:          "Conversation Usage Repository",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_repo",
		BaseRevision:  0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.conversationExecutionOrder[conversationID] = []string{executionID}
	state.executions[executionID] = Execution{
		ID:             executionID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_usage_sidebar_repo_" + randomHex(4),
		State:          RunStateCompleted,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot: ModelSnapshot{
			ModelID: "gpt-5",
		},
		TokensIn:                4,
		TokensOut:               6,
		ProjectRevisionSnapshot: 0,
		QueueIndex:              1,
		TraceID:                 "tr_usage_sidebar_repo_" + randomHex(4),
		CreatedAt:               now,
		UpdatedAt:               now,
	}
	state.mu.Unlock()

	syncExecutionDomainBestEffort(state)

	state.mu.Lock()
	state.conversations = map[string]Conversation{}
	state.executions = map[string]Execution{}
	state.conversationExecutionOrder = map[string][]string{}
	state.mu.Unlock()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/projects/{project_id}/sessions", ProjectConversationsHandler(state))
	res := performJSONRequest(t, mux, http.MethodGet, "/v1/projects/"+projectID+"/sessions", nil, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("expected list conversations 200 from repository, got %d (%s)", res.Code, res.Body.String())
	}

	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	items, ok := payload["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected exactly one conversation item, got %#v", payload["items"])
	}
	conversation, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("expected conversation object, got %#v", items[0])
	}
	if got := int(conversation["tokens_in_total"].(float64)); got != 4 {
		t.Fatalf("expected tokens_in_total 4, got %d", got)
	}
	if got := int(conversation["tokens_out_total"].(float64)); got != 6 {
		t.Fatalf("expected tokens_out_total 6, got %d", got)
	}
	if got := int(conversation["tokens_total"].(float64)); got != 10 {
		t.Fatalf("expected tokens_total 10, got %d", got)
	}
}
