// Copyright (c) 2026 Ysmjjsy
// Author: Goyais Team
// SPDX-License-Identifier: MIT

package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRunTaskQueryServiceBuildRunTaskGraphFromRepository(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	}()

	state, runID, taskIDs := seedRunTaskGraphStateWithStore(store)
	syncExecutionDomainBestEffort(state)

	service, ok := newRunTaskQueryService(state)
	if !ok {
		t.Fatalf("expected run task query service to be available")
	}
	graph, exists, err := service.BuildRunTaskGraph(context.Background(), runID)
	if err != nil {
		t.Fatalf("build run task graph failed: %v", err)
	}
	if !exists {
		t.Fatalf("expected run task graph to exist for %q", runID)
	}
	if graph.RunID != runID || len(graph.Tasks) != 3 {
		t.Fatalf("unexpected run task graph payload: %#v", graph)
	}

	taskByID := map[string]struct{}{}
	for _, item := range graph.Tasks {
		taskByID[item.TaskID] = struct{}{}
	}
	for _, taskID := range taskIDs {
		if _, exists := taskByID[taskID]; !exists {
			t.Fatalf("expected graph to include task %q, got %#v", taskID, graph.Tasks)
		}
	}
}

func TestRunGraphHandlerUsesRepositoryWhenStateMapMissing(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	}()

	state, runID, _ := seedRunTaskGraphStateWithStore(store)
	syncExecutionDomainBestEffort(state)

	state.mu.Lock()
	state.executions = map[string]Execution{}
	state.executionEvents = map[string][]ExecutionEvent{}
	state.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/v1/runs/"+runID+"/graph", nil)
	req.SetPathValue("run_id", runID)
	res := httptest.NewRecorder()
	RunGraphHandler(state).ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected run graph handler to use repository and return 200, got %d (%s)", res.Code, res.Body.String())
	}
}

func seedRunTaskGraphStateWithStore(store *authzStore) (*AppState, string, []string) {
	state := NewAppState(store)
	now := time.Now().UTC().Format(time.RFC3339)
	runID := "exec_task_root_repo_" + randomHex(4)
	secondTaskID := "exec_task_child_repo_" + randomHex(4)
	thirdTaskID := "exec_task_done_repo_" + randomHex(4)
	conversationID := "conv_task_repo_" + randomHex(4)
	activeExecutionID := runID

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:                conversationID,
		WorkspaceID:       localWorkspaceID,
		ProjectID:         "proj_task_repo_" + randomHex(4),
		Name:              "RunTaskConversationRepo",
		QueueState:        QueueStateRunning,
		DefaultMode:       PermissionModeDefault,
		ModelConfigID:     "rc_model_repo",
		ActiveExecutionID: &activeExecutionID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	state.executions[runID] = Execution{
		ID:             runID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_task_repo_" + randomHex(4),
		State:          RunStateExecuting,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     0,
		TraceID:        "tr_task_repo_" + randomHex(4),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.executions[secondTaskID] = Execution{
		ID:             secondTaskID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_task_repo_" + randomHex(4),
		State:          RunStateQueued,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     1,
		TraceID:        "tr_task_repo_" + randomHex(4),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.executions[thirdTaskID] = Execution{
		ID:             thirdTaskID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_task_repo_" + randomHex(4),
		State:          RunStateCompleted,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     2,
		TraceID:        "tr_task_repo_" + randomHex(4),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.conversationExecutionOrder[conversationID] = []string{runID, secondTaskID, thirdTaskID}
	state.executionEvents[conversationID] = []ExecutionEvent{
		{
			EventID:        "evt_task_repo_dep",
			ExecutionID:    secondTaskID,
			ConversationID: conversationID,
			TraceID:        "tr_task_repo_dep",
			Sequence:       1,
			QueueIndex:     1,
			Type:           RunEventTypeThinkingDelta,
			Timestamp:      now,
			Payload: map[string]any{
				"depends_on": []any{runID},
			},
		},
	}
	state.mu.Unlock()

	return state, runID, []string{runID, secondTaskID, thirdTaskID}
}
