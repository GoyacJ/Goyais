package httpapi

import (
	"net/http"
	"testing"
	"time"
)

func TestSummarizeConversationTokenUsageLocked(t *testing.T) {
	state := NewAppState(nil)
	conversationID := "conv_usage_1"
	now := time.Now().UTC().Format(time.RFC3339)

	state.conversations[conversationID] = Conversation{
		ID:            conversationID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     "proj_usage_1",
		Name:          "Usage",
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
	state.conversationExecutionOrder[conversationID] = []string{"exec_usage_1", "exec_usage_2", "exec_usage_3"}
	state.executions["exec_usage_1"] = newUsageExecution(conversationID, "exec_usage_1", 3, 5, now)
	state.executions["exec_usage_2"] = newUsageExecution(conversationID, "exec_usage_2", 7, 11, now)
	state.executions["exec_usage_3"] = newUsageExecution(conversationID, "exec_usage_3", 0, 0, now)

	tokensIn, tokensOut, tokensTotal := summarizeConversationTokenUsageLocked(state, conversationID)
	if tokensIn != 10 {
		t.Fatalf("expected tokens_in_total 10, got %d", tokensIn)
	}
	if tokensOut != 16 {
		t.Fatalf("expected tokens_out_total 16, got %d", tokensOut)
	}
	if tokensTotal != 26 {
		t.Fatalf("expected tokens_total 26, got %d", tokensTotal)
	}
}

func TestConversationsHandlerGetIncludesTokenUsage(t *testing.T) {
	state := NewAppState(nil)
	now := time.Now().UTC().Format(time.RFC3339)
	conversationID := "conv_usage_list_1"

	state.conversations[conversationID] = Conversation{
		ID:            conversationID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     "proj_usage_list_1",
		Name:          "List usage",
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
	state.conversationExecutionOrder[conversationID] = []string{"exec_usage_list_1", "exec_usage_list_2"}
	state.executions["exec_usage_list_1"] = newUsageExecution(conversationID, "exec_usage_list_1", 4, 6, now)
	state.executions["exec_usage_list_2"] = newUsageExecution(conversationID, "exec_usage_list_2", 8, 10, now)

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/conversations", ConversationsHandler(state))
	res := performJSONRequest(t, mux, http.MethodGet, "/v1/conversations?workspace_id="+localWorkspaceID, nil, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("expected conversations list 200, got %d (%s)", res.Code, res.Body.String())
	}

	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	items, ok := payload["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected exactly one conversation item, got %#v", payload["items"])
	}
	item, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("expected conversation object, got %#v", items[0])
	}
	if got := int(item["tokens_in_total"].(float64)); got != 12 {
		t.Fatalf("expected tokens_in_total 12, got %d", got)
	}
	if got := int(item["tokens_out_total"].(float64)); got != 16 {
		t.Fatalf("expected tokens_out_total 16, got %d", got)
	}
	if got := int(item["tokens_total"].(float64)); got != 28 {
		t.Fatalf("expected tokens_total 28, got %d", got)
	}
}

func TestConversationsHandlerGetUsesRepositoryWhenConversationAndExecutionMapMissing(t *testing.T) {
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
	conversationID := "conv_usage_list_repo_" + randomHex(4)
	executionID := "exec_usage_list_repo_" + randomHex(4)

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:            conversationID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     "proj_usage_list_repo_" + randomHex(4),
		Name:          "List Usage Repository",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_repo",
		BaseRevision:  0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.conversationExecutionOrder[conversationID] = []string{executionID}
	state.executions[executionID] = newUsageExecution(conversationID, executionID, 6, 7, now)
	state.mu.Unlock()

	syncExecutionDomainBestEffort(state)

	state.mu.Lock()
	state.conversations = map[string]Conversation{}
	state.executions = map[string]Execution{}
	state.conversationExecutionOrder = map[string][]string{}
	state.mu.Unlock()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/conversations", ConversationsHandler(state))
	res := performJSONRequest(t, mux, http.MethodGet, "/v1/conversations?workspace_id="+localWorkspaceID, nil, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("expected conversations list 200, got %d (%s)", res.Code, res.Body.String())
	}

	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	items, ok := payload["items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected exactly one conversation item, got %#v", payload["items"])
	}
	item, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("expected conversation object, got %#v", items[0])
	}
	if got := int(item["tokens_in_total"].(float64)); got != 6 {
		t.Fatalf("expected tokens_in_total 6, got %d", got)
	}
	if got := int(item["tokens_out_total"].(float64)); got != 7 {
		t.Fatalf("expected tokens_out_total 7, got %d", got)
	}
	if got := int(item["tokens_total"].(float64)); got != 13 {
		t.Fatalf("expected tokens_total 13, got %d", got)
	}
}

func TestConversationByIDHandlerGetAndPatchIncludeTokenUsage(t *testing.T) {
	state := NewAppState(nil)
	now := time.Now().UTC().Format(time.RFC3339)
	projectID := "proj_usage_detail_1"
	conversationID := "conv_usage_detail_1"

	state.projects[projectID] = Project{
		ID:          projectID,
		WorkspaceID: localWorkspaceID,
		Name:        "Usage Detail",
		RepoPath:    "/tmp/usage-detail",
		IsGit:       true,
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
		Name:          "Before Rename",
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
	state.conversationExecutionOrder[conversationID] = []string{"exec_usage_detail_1"}
	state.executions["exec_usage_detail_1"] = newUsageExecution(conversationID, "exec_usage_detail_1", 9, 12, now)

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/conversations/{conversation_id}", ConversationByIDHandler(state))

	getRes := performJSONRequest(t, mux, http.MethodGet, "/v1/conversations/"+conversationID, nil, nil)
	if getRes.Code != http.StatusOK {
		t.Fatalf("expected conversation detail 200, got %d (%s)", getRes.Code, getRes.Body.String())
	}
	getPayload := map[string]any{}
	mustDecodeJSON(t, getRes.Body.Bytes(), &getPayload)
	getConversation := getPayload["conversation"].(map[string]any)
	if got := int(getConversation["tokens_in_total"].(float64)); got != 9 {
		t.Fatalf("expected detail tokens_in_total 9, got %d", got)
	}
	if got := int(getConversation["tokens_out_total"].(float64)); got != 12 {
		t.Fatalf("expected detail tokens_out_total 12, got %d", got)
	}
	if got := int(getConversation["tokens_total"].(float64)); got != 21 {
		t.Fatalf("expected detail tokens_total 21, got %d", got)
	}

	patchRes := performJSONRequest(t, mux, http.MethodPatch, "/v1/conversations/"+conversationID, map[string]any{
		"name": "After Rename",
	}, nil)
	if patchRes.Code != http.StatusOK {
		t.Fatalf("expected conversation patch 200, got %d (%s)", patchRes.Code, patchRes.Body.String())
	}
	patched := Conversation{}
	mustDecodeJSON(t, patchRes.Body.Bytes(), &patched)
	if patched.Name != "After Rename" {
		t.Fatalf("expected patched name After Rename, got %q", patched.Name)
	}
	if patched.TokensInTotal != 9 || patched.TokensOutTotal != 12 || patched.TokensTotal != 21 {
		t.Fatalf("expected patched usage totals 9/12/21, got %d/%d/%d", patched.TokensInTotal, patched.TokensOutTotal, patched.TokensTotal)
	}
}

func TestConversationByIDHandlerPatchEmitsConfigChangeHookRecordForActiveExecution(t *testing.T) {
	state := NewAppState(nil)
	now := time.Now().UTC().Format(time.RFC3339)
	projectID := "proj_cfg_change_1"
	conversationID := "conv_cfg_change_1"
	executionID := "exec_cfg_change_1"

	state.projects[projectID] = Project{
		ID:          projectID,
		WorkspaceID: localWorkspaceID,
		Name:        "Config Change",
		RepoPath:    "/tmp/config-change",
		IsGit:       true,
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
		ID:                conversationID,
		WorkspaceID:       localWorkspaceID,
		ProjectID:         projectID,
		Name:              "Config Conversation",
		QueueState:        QueueStateRunning,
		DefaultMode:       PermissionModeDefault,
		ModelConfigID:     "rc_model_1",
		RuleIDs:           []string{},
		SkillIDs:          []string{},
		MCPIDs:            []string{},
		BaseRevision:      0,
		ActiveExecutionID: ptrString(executionID),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	state.executions[executionID] = Execution{
		ID:             executionID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_cfg_change_1",
		State:          RunStateExecuting,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5"},
		QueueIndex:     0,
		TraceID:        "tr_cfg_change_1",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.hookPolicies["policy_config_change_deny"] = HookPolicy{
		ID:          "policy_config_change_deny",
		Scope:       HookScopeGlobal,
		Event:       HookEventTypeConfigChange,
		HandlerType: HookHandlerTypeAgent,
		Enabled:     true,
		Decision: HookDecision{
			Action: HookDecisionActionDeny,
			Reason: "test config change hook deny",
		},
		UpdatedAt: now,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/conversations/{conversation_id}", ConversationByIDHandler(state))

	patchRes := performJSONRequest(t, mux, http.MethodPatch, "/v1/conversations/"+conversationID, map[string]any{
		"mode": "plan",
	}, nil)
	if patchRes.Code != http.StatusOK {
		t.Fatalf("expected conversation patch 200, got %d (%s)", patchRes.Code, patchRes.Body.String())
	}

	state.mu.RLock()
	records := append([]HookExecutionRecord{}, state.hookExecutionRecords[conversationID]...)
	state.mu.RUnlock()

	foundConfigChangeRecord := false
	for _, record := range records {
		if record.RunID != executionID || record.Event != HookEventTypeConfigChange {
			continue
		}
		if record.PolicyID != "policy_config_change_deny" || record.Decision.Action != HookDecisionActionDeny {
			t.Fatalf("unexpected config_change hook record: %#v", record)
		}
		foundConfigChangeRecord = true
	}
	if !foundConfigChangeRecord {
		t.Fatalf("expected config_change hook record for run %s, got %#v", executionID, records)
	}
}

func TestConversationByIDHandlerGetUsesRepositoryWhenExecutionMapMissing(t *testing.T) {
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
	conversationID := "conv_usage_repo_" + randomHex(4)
	executionID := "exec_usage_repo_" + randomHex(4)

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:            conversationID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     "proj_usage_repo_" + randomHex(4),
		Name:          "Repo Usage",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_repo",
		BaseRevision:  0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.conversationExecutionOrder[conversationID] = []string{executionID}
	state.executions[executionID] = newUsageExecution(conversationID, executionID, 5, 8, now)
	state.mu.Unlock()

	syncExecutionDomainBestEffort(state)

	state.mu.Lock()
	state.conversations = map[string]Conversation{}
	state.executions = map[string]Execution{}
	state.conversationExecutionOrder = map[string][]string{}
	state.mu.Unlock()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/conversations/{conversation_id}", ConversationByIDHandler(state))

	res := performJSONRequest(t, mux, http.MethodGet, "/v1/conversations/"+conversationID, nil, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("expected conversation detail 200, got %d (%s)", res.Code, res.Body.String())
	}

	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)

	conversationRaw, ok := payload["conversation"].(map[string]any)
	if !ok {
		t.Fatalf("expected conversation object, got %#v", payload["conversation"])
	}
	if got := int(conversationRaw["tokens_in_total"].(float64)); got != 5 {
		t.Fatalf("expected tokens_in_total 5, got %d", got)
	}
	if got := int(conversationRaw["tokens_out_total"].(float64)); got != 8 {
		t.Fatalf("expected tokens_out_total 8, got %d", got)
	}
	if got := int(conversationRaw["tokens_total"].(float64)); got != 13 {
		t.Fatalf("expected tokens_total 13, got %d", got)
	}

	executionsRaw, ok := payload["executions"].([]any)
	if !ok || len(executionsRaw) != 1 {
		t.Fatalf("expected one execution from repository, got %#v", payload["executions"])
	}
	executionRaw, ok := executionsRaw[0].(map[string]any)
	if !ok {
		t.Fatalf("expected execution object, got %#v", executionsRaw[0])
	}
	if got := executionRaw["id"]; got != executionID {
		t.Fatalf("expected execution id %q, got %#v", executionID, got)
	}
}

func TestConversationByIDHandlerPatchUsesRepositoryWhenConversationMapMissing(t *testing.T) {
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
	projectID := "proj_usage_patch_repo_" + randomHex(4)
	conversationID := "conv_usage_patch_repo_" + randomHex(4)

	project, saveErr := saveProjectToStore(state, Project{
		ID:          projectID,
		WorkspaceID: localWorkspaceID,
		Name:        "Patch Repository Project",
		RepoPath:    "/tmp/patch-repository-project",
		IsGit:       false,
		DefaultMode: PermissionModeDefault,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if saveErr != nil {
		t.Fatalf("save project failed: %v", saveErr)
	}

	_, configErr := saveProjectConfigToStore(state, localWorkspaceID, ProjectConfig{
		ProjectID:      project.ID,
		ModelConfigIDs: []string{},
		RuleIDs:        []string{},
		SkillIDs:       []string{},
		MCPIDs:         []string{},
		UpdatedAt:      now,
	})
	if configErr != nil {
		t.Fatalf("save project config failed: %v", configErr)
	}

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:            conversationID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     project.ID,
		Name:          "Before Repository Patch",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_patch_repo",
		BaseRevision:  0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.mu.Unlock()

	syncExecutionDomainBestEffort(state)

	state.mu.Lock()
	state.conversations = map[string]Conversation{}
	state.mu.Unlock()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/conversations/{conversation_id}", ConversationByIDHandler(state))

	res := performJSONRequest(t, mux, http.MethodPatch, "/v1/conversations/"+conversationID, map[string]any{
		"name": "After Repository Patch",
	}, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("expected conversation patch 200 with repository seed, got %d (%s)", res.Code, res.Body.String())
	}

	patched := Conversation{}
	mustDecodeJSON(t, res.Body.Bytes(), &patched)
	if patched.Name != "After Repository Patch" {
		t.Fatalf("expected patched name After Repository Patch, got %q", patched.Name)
	}

	state.mu.RLock()
	hydrated, exists := state.conversations[conversationID]
	state.mu.RUnlock()
	if !exists {
		t.Fatalf("expected conversation seed to be hydrated into state map")
	}
	if hydrated.Name != "After Repository Patch" {
		t.Fatalf("expected hydrated conversation name After Repository Patch, got %q", hydrated.Name)
	}
}

func newUsageExecution(conversationID string, executionID string, tokensIn int, tokensOut int, now string) Execution {
	return Execution{
		ID:             executionID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_" + executionID,
		State:          RunStateCompleted,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot: ModelSnapshot{
			ModelID: "gpt-5",
		},
		TokensIn:                tokensIn,
		TokensOut:               tokensOut,
		ProjectRevisionSnapshot: 0,
		QueueIndex:              0,
		TraceID:                 "tr_" + executionID,
		CreatedAt:               now,
		UpdatedAt:               now,
	}
}
