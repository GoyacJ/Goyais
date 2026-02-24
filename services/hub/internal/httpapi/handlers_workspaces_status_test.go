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

func TestDeriveConversationStatusLockedMappings(t *testing.T) {
	state := NewAppState(nil)
	workspace := state.CreateRemoteWorkspace(CreateWorkspaceRequest{
		Name:     "Remote Status Mapping",
		HubURL:   "http://127.0.0.1:9782",
		AuthMode: AuthModePasswordOrToken,
	})

	cases := []struct {
		name   string
		states []ExecutionState
		want   ConversationStatus
	}{
		{name: "running from executing", states: []ExecutionState{ExecutionStateExecuting}, want: ConversationStatusRunning},
		{name: "queued from pending", states: []ExecutionState{ExecutionStatePending}, want: ConversationStatusQueued},
		{name: "done from completed", states: []ExecutionState{ExecutionStateCompleted}, want: ConversationStatusDone},
		{name: "error from failed", states: []ExecutionState{ExecutionStateFailed}, want: ConversationStatusError},
		{name: "stopped from cancelled", states: []ExecutionState{ExecutionStateCancelled}, want: ConversationStatusStopped},
		{name: "stopped from empty", states: nil, want: ConversationStatusStopped},
	}

	for index, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			conversationID := "conv_status_" + randomHex(6)
			now := time.Now().UTC().Add(time.Duration(index) * time.Second).Format(time.RFC3339)
			conversation := Conversation{
				ID:           conversationID,
				WorkspaceID:  workspace.ID,
				ProjectID:    "proj_" + randomHex(4),
				Name:         "Status Conversation",
				QueueState:   QueueStateIdle,
				DefaultMode:  ConversationModeAgent,
				ModelID:      "gpt-5.3",
				BaseRevision: 0,
				CreatedAt:    now,
				UpdatedAt:    now,
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
					Mode:           ConversationModeAgent,
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
