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
		DefaultMode: ConversationModeAgent,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	state.projects[otherProjectID] = Project{
		ID:          otherProjectID,
		WorkspaceID: workspaceID,
		Name:        "Other",
		RepoPath:    "/tmp/other",
		DefaultMode: ConversationModeAgent,
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
		DefaultMode:   ConversationModeAgent,
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
		DefaultMode:   ConversationModeAgent,
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
		Type:           ExecutionEventTypeExecutionStarted,
		Timestamp:      now,
	}}
	state.executionEvents[otherConversationID] = []ExecutionEvent{{
		EventID:        "evt_other",
		ConversationID: otherConversationID,
		ExecutionID:    otherExecutionID,
		Type:           ExecutionEventTypeExecutionStarted,
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
		State:          ExecutionStateExecuting,
		Mode:           ConversationModeAgent,
		ModelID:        oldModelID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.executions[otherExecutionID] = Execution{
		ID:             otherExecutionID,
		WorkspaceID:    workspaceID,
		ConversationID: otherConversationID,
		State:          ExecutionStateExecuting,
		Mode:           ConversationModeAgent,
		ModelID:        otherModelID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.executionDiffs[targetExecutionID] = []DiffItem{{ID: "diff_target", Path: "a.txt", ChangeType: "modified"}}
	state.executionDiffs[otherExecutionID] = []DiffItem{{ID: "diff_other", Path: "b.txt", ChangeType: "modified"}}

	cancelled := make(chan struct{}, 1)
	state.orchestrator.mu.Lock()
	state.orchestrator.active[targetExecutionID] = func() {
		select {
		case cancelled <- struct{}{}:
		default:
		}
	}
	state.orchestrator.mu.Unlock()

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
	case <-cancelled:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("expected running execution cancellation")
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
		DefaultMode: ConversationModeAgent,
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
	mux.HandleFunc("/v1/projects/{project_id}/conversations", ProjectConversationsHandler(state))
	res := performJSONRequest(t, mux, http.MethodPost, "/v1/projects/"+projectID+"/conversations", map[string]any{
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
