package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWorkspaceStatusHandlerLocalDefaults(t *testing.T) {
	state := NewAppState(nil)
	handler := WorkspaceStatusHandler(state)

	req := httptest.NewRequest(http.MethodGet, "/v1/workspaces/ws_local/status", nil)
	req.SetPathValue("workspace_id", localWorkspaceID)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", res.Code, res.Body.String())
	}

	payload := WorkspaceStatusResponse{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	if payload.WorkspaceID != localWorkspaceID {
		t.Fatalf("expected workspace_id=%s, got %s", localWorkspaceID, payload.WorkspaceID)
	}
	if payload.ConversationStatus != ConversationStatusStopped {
		t.Fatalf("expected stopped conversation status, got %s", payload.ConversationStatus)
	}
	if payload.ConnectionStatus != "connected" {
		t.Fatalf("expected connected, got %s", payload.ConnectionStatus)
	}
	if payload.HubURL != "local://workspace" {
		t.Fatalf("expected local hub url, got %s", payload.HubURL)
	}
	if payload.UserDisplayName != "Local User" {
		t.Fatalf("expected Local User display name, got %q", payload.UserDisplayName)
	}
}

func TestWorkspaceStatusHandlerRemoteRequiresAuth(t *testing.T) {
	router := NewRouter()
	workspaceID := createRemoteWorkspace(t, router, "Remote Status", "http://127.0.0.1:9781", false)

	res := performJSONRequest(t, router, http.MethodGet, "/v1/workspaces/"+workspaceID+"/status", nil, nil)
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d (%s)", res.Code, res.Body.String())
	}
}

func TestWorkspaceStatusHandlerUsesRepositoryWhenExecutionMapMissing(t *testing.T) {
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
	handler := WorkspaceStatusHandler(state)
	now := time.Now().UTC().Format(time.RFC3339)
	conversationID := "conv_status_repo_" + randomHex(4)
	runID := "run_status_repo_" + randomHex(4)

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:                conversationID,
		WorkspaceID:       localWorkspaceID,
		ProjectID:         "proj_status_repo_" + randomHex(4),
		Name:              "Repository Status Conversation",
		QueueState:        QueueStateRunning,
		DefaultMode:       PermissionModeDefault,
		ModelConfigID:     "rc_model_status_repo",
		ActiveExecutionID: ptrString(runID),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	state.conversationExecutionOrder[conversationID] = []string{runID}
	state.executions[runID] = Execution{
		ID:             runID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_status_repo_" + randomHex(4),
		State:          RunStateExecuting,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     0,
		TraceID:        "tr_status_repo_" + randomHex(4),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.mu.Unlock()

	syncExecutionDomainBestEffort(state)

	state.mu.Lock()
	state.conversations = map[string]Conversation{}
	state.executions = map[string]Execution{}
	state.conversationExecutionOrder = map[string][]string{}
	state.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/v1/workspaces/"+localWorkspaceID+"/status?conversation_id="+conversationID, nil)
	req.SetPathValue("workspace_id", localWorkspaceID)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", res.Code, res.Body.String())
	}

	payload := WorkspaceStatusResponse{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	if payload.ConversationID != conversationID {
		t.Fatalf("expected conversation_id %q, got %q", conversationID, payload.ConversationID)
	}
	if payload.ConversationStatus != ConversationStatusRunning {
		t.Fatalf("expected running status from repository, got %s", payload.ConversationStatus)
	}
}

func TestWorkspaceStatusHandlerSelectsConversationFromRepositoryWhenConversationMapMissing(t *testing.T) {
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
	handler := WorkspaceStatusHandler(state)
	now := time.Now().UTC().Format(time.RFC3339)
	conversationID := "conv_status_select_repo_" + randomHex(4)
	runID := "run_status_select_repo_" + randomHex(4)

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:                conversationID,
		WorkspaceID:       localWorkspaceID,
		ProjectID:         "proj_status_select_repo_" + randomHex(4),
		Name:              "Repository Select Conversation",
		QueueState:        QueueStateRunning,
		DefaultMode:       PermissionModeDefault,
		ModelConfigID:     "rc_model_status_select_repo",
		ActiveExecutionID: ptrString(runID),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	state.conversationExecutionOrder[conversationID] = []string{runID}
	state.executions[runID] = Execution{
		ID:             runID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_status_select_repo_" + randomHex(4),
		State:          RunStateExecuting,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     0,
		TraceID:        "tr_status_select_repo_" + randomHex(4),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.mu.Unlock()

	syncExecutionDomainBestEffort(state)

	state.mu.Lock()
	state.conversations = map[string]Conversation{}
	state.executions = map[string]Execution{}
	state.conversationExecutionOrder = map[string][]string{}
	state.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/v1/workspaces/"+localWorkspaceID+"/status", nil)
	req.SetPathValue("workspace_id", localWorkspaceID)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", res.Code, res.Body.String())
	}

	payload := WorkspaceStatusResponse{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	if payload.ConversationID != conversationID {
		t.Fatalf("expected conversation_id %q, got %q", conversationID, payload.ConversationID)
	}
	if payload.ConversationStatus != ConversationStatusRunning {
		t.Fatalf("expected running status from repository-selected conversation, got %s", payload.ConversationStatus)
	}
}

func TestDeriveConversationStatusLockedMappings(t *testing.T) {
	state := NewAppState(nil)
	workspace := state.CreateRemoteWorkspace(CreateWorkspaceRequest{
		Name:     "Remote Status Mapping",
		HubURL:   "http://127.0.0.1:9782",
		AuthMode: AuthModePasswordOrToken,
	})

	cases := []struct {
		name   string
		states []RunState
		want   ConversationStatus
	}{
		{name: "running from executing", states: []RunState{RunStateExecuting}, want: ConversationStatusRunning},
		{name: "running from confirming", states: []RunState{RunStateConfirming}, want: ConversationStatusRunning},
		{name: "queued from pending", states: []RunState{RunStatePending}, want: ConversationStatusQueued},
		{name: "done from completed", states: []RunState{RunStateCompleted}, want: ConversationStatusDone},
		{name: "error from failed", states: []RunState{RunStateFailed}, want: ConversationStatusError},
		{name: "stopped from cancelled", states: []RunState{RunStateCancelled}, want: ConversationStatusStopped},
		{name: "stopped from empty", states: nil, want: ConversationStatusStopped},
	}

	for index, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			conversationID := "conv_status_" + randomHex(6)
			now := time.Now().UTC().Add(time.Duration(index) * time.Second).Format(time.RFC3339)
			conversation := Conversation{
				ID:            conversationID,
				WorkspaceID:   workspace.ID,
				ProjectID:     "proj_" + randomHex(4),
				Name:          "Status Conversation",
				QueueState:    QueueStateIdle,
				DefaultMode:   PermissionModeDefault,
				ModelConfigID: "rc_model_status",
				BaseRevision:  0,
				CreatedAt:     now,
				UpdatedAt:     now,
			}

			state.mu.Lock()
			state.conversations[conversationID] = conversation
			state.conversationExecutionOrder[conversationID] = nil
			for executionIndex, executionState := range tc.states {
				executionID := "exec_status_" + randomHex(6)
				execution := Execution{
					ID:             executionID,
					WorkspaceID:    workspace.ID,
					ConversationID: conversationID,
					MessageID:      "msg_status_" + randomHex(4),
					State:          executionState,
					Mode:           PermissionModeDefault,
					ModelID:        "gpt-5.3",
					QueueIndex:     executionIndex,
					TraceID:        "tr_status_" + randomHex(6),
					CreatedAt:      now,
					UpdatedAt:      now,
				}
				state.executions[executionID] = execution
				state.conversationExecutionOrder[conversationID] = append(state.conversationExecutionOrder[conversationID], executionID)
			}

			got := deriveConversationStatusLocked(state, conversationID)
			state.mu.Unlock()

			if got != tc.want {
				t.Fatalf("expected %s, got %s", tc.want, got)
			}
		})
	}
}
