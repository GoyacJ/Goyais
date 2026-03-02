package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRunTaskRoutesAreRegistered(t *testing.T) {
	router := NewRouter()

	cases := []struct {
		method string
		path   string
		body   map[string]any
	}{
		{method: http.MethodGet, path: "/v1/runs/missing_run/graph"},
		{method: http.MethodGet, path: "/v1/runs/missing_run/tasks"},
		{method: http.MethodGet, path: "/v1/runs/missing_run/tasks/missing_task"},
		{method: http.MethodPost, path: "/v1/runs/missing_run/tasks/missing_task/control", body: map[string]any{"action": "cancel"}},
	}
	for _, tc := range cases {
		res := performJSONRequest(t, router, tc.method, tc.path, tc.body, nil)
		if res.Code != http.StatusNotFound {
			t.Fatalf("%s %s expected 404, got %d (%s)", tc.method, tc.path, res.Code, res.Body.String())
		}
		payload := StandardError{}
		mustDecodeJSON(t, res.Body.Bytes(), &payload)
		if payload.Code != "RUN_NOT_FOUND" {
			t.Fatalf("%s %s expected RUN_NOT_FOUND, got %#v", tc.method, tc.path, payload.Code)
		}
	}
}

func TestRunTaskHandlersExposeGraphListAndDetail(t *testing.T) {
	state, runID, taskIDs := seedRunTaskGraphState()
	graphHandler := RunGraphHandler(state)
	listHandler := RunTasksHandler(state)
	detailHandler := RunTaskByIDHandler(state)

	graphReq := httptest.NewRequest(http.MethodGet, "/v1/runs/"+runID+"/graph", nil)
	graphReq.SetPathValue("run_id", runID)
	graphRes := httptest.NewRecorder()
	graphHandler.ServeHTTP(graphRes, graphReq)
	if graphRes.Code != http.StatusOK {
		t.Fatalf("graph endpoint expected 200, got %d (%s)", graphRes.Code, graphRes.Body.String())
	}
	graphPayload := AgentGraph{}
	mustDecodeJSON(t, graphRes.Body.Bytes(), &graphPayload)
	if graphPayload.RunID != runID || graphPayload.MaxParallelism != 1 || len(graphPayload.Tasks) != 3 {
		t.Fatalf("unexpected graph payload: %#v", graphPayload)
	}
	if graphPayload.Tasks[0].TaskID != taskIDs[0] || graphPayload.Tasks[0].State != TaskStateRunning {
		t.Fatalf("unexpected graph task payload: %#v", graphPayload.Tasks[0])
	}
	if len(graphPayload.Edges) != 2 || graphPayload.Edges[0].FromTaskID != taskIDs[0] || graphPayload.Edges[0].ToTaskID != taskIDs[1] {
		t.Fatalf("unexpected graph edges payload: %#v", graphPayload.Edges)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/v1/runs/"+runID+"/tasks?state=running", nil)
	listReq.SetPathValue("run_id", runID)
	listRes := httptest.NewRecorder()
	listHandler.ServeHTTP(listRes, listReq)
	if listRes.Code != http.StatusOK {
		t.Fatalf("tasks endpoint expected 200, got %d (%s)", listRes.Code, listRes.Body.String())
	}
	listPayload := RunTaskListResponse{}
	mustDecodeJSON(t, listRes.Body.Bytes(), &listPayload)
	if len(listPayload.Items) != 1 || listPayload.Items[0].TaskID != taskIDs[0] {
		t.Fatalf("unexpected tasks payload: %#v", listPayload)
	}

	targetTaskID := taskIDs[2]
	detailReq := httptest.NewRequest(http.MethodGet, "/v1/runs/"+runID+"/tasks/"+targetTaskID, nil)
	detailReq.SetPathValue("run_id", runID)
	detailReq.SetPathValue("task_id", targetTaskID)
	detailRes := httptest.NewRecorder()
	detailHandler.ServeHTTP(detailRes, detailReq)
	if detailRes.Code != http.StatusOK {
		t.Fatalf("task detail endpoint expected 200, got %d (%s)", detailRes.Code, detailRes.Body.String())
	}
	taskPayload := TaskNode{}
	mustDecodeJSON(t, detailRes.Body.Bytes(), &taskPayload)
	if taskPayload.TaskID != targetTaskID || taskPayload.RunID != runID || taskPayload.State != TaskStateCompleted {
		t.Fatalf("unexpected task detail payload: %#v", taskPayload)
	}
	if len(taskPayload.DependsOn) != 1 || taskPayload.DependsOn[0] != taskIDs[1] {
		t.Fatalf("unexpected task detail depends_on: %#v", taskPayload.DependsOn)
	}
}

func TestRunTaskControlHandlerDelegatesToRunControl(t *testing.T) {
	state, runID, taskIDs := seedRunTaskGraphState()
	targetTaskID := taskIDs[1]

	handler := RunTaskControlHandler(state)
	req := httptest.NewRequest(http.MethodPost, "/v1/runs/"+runID+"/tasks/"+targetTaskID+"/control", strings.NewReader(`{"action":"cancel"}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("run_id", runID)
	req.SetPathValue("task_id", targetTaskID)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("task control endpoint expected 200, got %d (%s)", res.Code, res.Body.String())
	}
	payload := TaskControlResponse{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	if !payload.OK || payload.RunID != runID || payload.TaskID != targetTaskID {
		t.Fatalf("unexpected task control payload: %#v", payload)
	}
	if payload.State == "" || payload.PreviousState == "" {
		t.Fatalf("expected task control state fields present, got %#v", payload)
	}

	state.mu.RLock()
	updated := state.executions[targetTaskID]
	state.mu.RUnlock()
	if updated.State != ExecutionStateCancelled {
		t.Fatalf("expected delegated stop to cancel execution, got %s", updated.State)
	}
}

func TestRunTaskGraphBuildUsesExecutionEventMetadata(t *testing.T) {
	state, runID, taskIDs := seedRunTaskGraphState()
	state.mu.Lock()
	conversationID := state.executions[runID].ConversationID
	now := time.Now().UTC().Format(time.RFC3339)
	state.executionEvents[conversationID] = []ExecutionEvent{
		{
			EventID:        "evt_meta_1",
			ExecutionID:    runID,
			ConversationID: conversationID,
			TraceID:        "tr_meta_1",
			Sequence:       1,
			QueueIndex:     0,
			Type:           ExecutionEventTypeThinkingDelta,
			Timestamp:      now,
			Payload: map[string]any{
				"max_parallelism": 2,
			},
		},
		{
			EventID:        "evt_meta_2",
			ExecutionID:    taskIDs[1],
			ConversationID: conversationID,
			TraceID:        "tr_meta_2",
			Sequence:       2,
			QueueIndex:     1,
			Type:           ExecutionEventTypeThinkingDelta,
			Timestamp:      now,
			Payload: map[string]any{
				"depends_on":  []any{taskIDs[0]},
				"retry_count": 1,
				"max_retries": 3,
				"task_artifact": map[string]any{
					"kind":    "diff",
					"uri":     "file:///tmp/patch.diff",
					"summary": "changes prepared",
					"metadata": map[string]any{
						"files": 3,
					},
				},
				"last_error": "tool timeout",
			},
		},
		{
			EventID:        "evt_meta_3",
			ExecutionID:    taskIDs[2],
			ConversationID: conversationID,
			TraceID:        "tr_meta_3",
			Sequence:       3,
			QueueIndex:     2,
			Type:           ExecutionEventTypeThinkingDelta,
			Timestamp:      now,
			Payload: map[string]any{
				"task": map[string]any{
					"depends_on": []any{taskIDs[0]},
				},
			},
		},
	}
	state.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/v1/runs/"+runID+"/graph", nil)
	req.SetPathValue("run_id", runID)
	res := httptest.NewRecorder()
	RunGraphHandler(state).ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("graph endpoint expected 200, got %d (%s)", res.Code, res.Body.String())
	}

	graphPayload := AgentGraph{}
	mustDecodeJSON(t, res.Body.Bytes(), &graphPayload)
	if graphPayload.MaxParallelism != 2 {
		t.Fatalf("expected max_parallelism from event metadata, got %#v", graphPayload.MaxParallelism)
	}
	if len(graphPayload.Edges) != 2 {
		t.Fatalf("expected explicit dependency edges, got %#v", graphPayload.Edges)
	}

	taskByID := map[string]TaskNode{}
	for _, item := range graphPayload.Tasks {
		taskByID[item.TaskID] = item
	}
	childTask := taskByID[taskIDs[1]]
	if childTask.RetryCount != 1 || childTask.MaxRetries != 3 {
		t.Fatalf("expected retry metadata applied, got %#v", childTask)
	}
	if len(childTask.DependsOn) != 1 || childTask.DependsOn[0] != taskIDs[0] {
		t.Fatalf("expected depends_on from event metadata, got %#v", childTask.DependsOn)
	}
	if childTask.State != TaskStateBlocked {
		t.Fatalf("expected child blocked while dependency running, got %#v", childTask.State)
	}
	if childTask.Artifact == nil || childTask.Artifact.Kind != "diff" || childTask.Artifact.URI != "file:///tmp/patch.diff" {
		t.Fatalf("expected artifact metadata applied, got %#v", childTask.Artifact)
	}
	if childTask.LastError == nil || strings.TrimSpace(*childTask.LastError) != "tool timeout" {
		t.Fatalf("expected last_error metadata applied, got %#v", childTask.LastError)
	}
}

func TestRunTaskGraphBuildUsesStructuredTaskEvents(t *testing.T) {
	state, runID, taskIDs := seedRunTaskGraphState()
	state.mu.Lock()
	conversationID := state.executions[runID].ConversationID
	now := time.Now().UTC().Format(time.RFC3339)
	state.executionEvents[conversationID] = []ExecutionEvent{
		{
			EventID:        "evt_task_cfg",
			ExecutionID:    runID,
			ConversationID: conversationID,
			TraceID:        "tr_task_cfg",
			Sequence:       1,
			QueueIndex:     0,
			Type:           ExecutionEventTypeTaskGraphConfigured,
			Timestamp:      now,
			Payload: map[string]any{
				"max_parallelism": 2,
			},
		},
		{
			EventID:        "evt_task_dep",
			ExecutionID:    runID,
			ConversationID: conversationID,
			TraceID:        "tr_task_dep",
			Sequence:       2,
			QueueIndex:     0,
			Type:           ExecutionEventTypeTaskDependenciesUpdated,
			Timestamp:      now,
			Payload: map[string]any{
				"task_id":    taskIDs[1],
				"depends_on": []any{taskIDs[0]},
				"priority":   5,
			},
		},
		{
			EventID:        "evt_task_retry",
			ExecutionID:    runID,
			ConversationID: conversationID,
			TraceID:        "tr_task_retry",
			Sequence:       3,
			QueueIndex:     0,
			Type:           ExecutionEventTypeTaskRetryPolicyUpdated,
			Timestamp:      now,
			Payload: map[string]any{
				"task_id":     taskIDs[1],
				"retry_count": 2,
				"max_retries": 5,
			},
		},
		{
			EventID:        "evt_task_artifact",
			ExecutionID:    runID,
			ConversationID: conversationID,
			TraceID:        "tr_task_artifact",
			Sequence:       4,
			QueueIndex:     0,
			Type:           ExecutionEventTypeTaskArtifactEmitted,
			Timestamp:      now,
			Payload: map[string]any{
				"task_id": taskIDs[1],
				"artifact": map[string]any{
					"kind":    "report",
					"uri":     "file:///tmp/report.json",
					"summary": "artifact ready",
					"metadata": map[string]any{
						"size": 12,
					},
				},
			},
		},
		{
			EventID:        "evt_task_failed",
			ExecutionID:    runID,
			ConversationID: conversationID,
			TraceID:        "tr_task_failed",
			Sequence:       5,
			QueueIndex:     0,
			Type:           ExecutionEventTypeTaskFailed,
			Timestamp:      now,
			Payload: map[string]any{
				"task_id":       taskIDs[1],
				"error_message": "lint failed",
			},
		},
	}
	state.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/v1/runs/"+runID+"/graph", nil)
	req.SetPathValue("run_id", runID)
	res := httptest.NewRecorder()
	RunGraphHandler(state).ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("graph endpoint expected 200, got %d (%s)", res.Code, res.Body.String())
	}

	graphPayload := AgentGraph{}
	mustDecodeJSON(t, res.Body.Bytes(), &graphPayload)
	if graphPayload.MaxParallelism != 2 {
		t.Fatalf("expected max_parallelism from structured task event, got %#v", graphPayload.MaxParallelism)
	}

	taskByID := map[string]TaskNode{}
	for _, item := range graphPayload.Tasks {
		taskByID[item.TaskID] = item
	}
	childTask := taskByID[taskIDs[1]]
	if childTask.RetryCount != 2 || childTask.MaxRetries != 5 {
		t.Fatalf("expected retry policy from structured task event, got %#v", childTask)
	}
	if len(childTask.DependsOn) != 1 || childTask.DependsOn[0] != taskIDs[0] {
		t.Fatalf("expected depends_on from structured task event, got %#v", childTask.DependsOn)
	}
	if childTask.Artifact == nil || childTask.Artifact.Kind != "report" || childTask.Artifact.URI != "file:///tmp/report.json" {
		t.Fatalf("expected artifact from structured task event, got %#v", childTask.Artifact)
	}
	if childTask.LastError == nil || strings.TrimSpace(*childTask.LastError) != "lint failed" {
		t.Fatalf("expected last_error from structured task event, got %#v", childTask.LastError)
	}
}

func TestRunTaskGraphBuildUsesStructuredTaskLifecycleEvents(t *testing.T) {
	state, runID, taskIDs := seedRunTaskGraphState()
	state.mu.Lock()
	conversationID := state.executions[runID].ConversationID
	now := time.Now().UTC().Format(time.RFC3339)
	state.executionEvents[conversationID] = []ExecutionEvent{
		{
			EventID:        "evt_task_started",
			ExecutionID:    runID,
			ConversationID: conversationID,
			TraceID:        "tr_task_started",
			Sequence:       1,
			QueueIndex:     0,
			Type:           ExecutionEventTypeTaskStarted,
			Timestamp:      now,
			Payload: map[string]any{
				"task_id": taskIDs[1],
			},
		},
		{
			EventID:        "evt_task_cancelled",
			ExecutionID:    runID,
			ConversationID: conversationID,
			TraceID:        "tr_task_cancelled",
			Sequence:       2,
			QueueIndex:     0,
			Type:           ExecutionEventTypeTaskCancelled,
			Timestamp:      now,
			Payload: map[string]any{
				"task_id": taskIDs[0],
				"reason":  "stop",
			},
		},
		{
			EventID:        "evt_task_completed",
			ExecutionID:    runID,
			ConversationID: conversationID,
			TraceID:        "tr_task_completed",
			Sequence:       3,
			QueueIndex:     0,
			Type:           ExecutionEventTypeTaskCompleted,
			Timestamp:      now,
			Payload: map[string]any{
				"task_id": taskIDs[2],
			},
		},
	}
	state.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/v1/runs/"+runID+"/graph", nil)
	req.SetPathValue("run_id", runID)
	res := httptest.NewRecorder()
	RunGraphHandler(state).ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("graph endpoint expected 200, got %d (%s)", res.Code, res.Body.String())
	}

	graphPayload := AgentGraph{}
	mustDecodeJSON(t, res.Body.Bytes(), &graphPayload)
	taskByID := map[string]TaskNode{}
	for _, item := range graphPayload.Tasks {
		taskByID[item.TaskID] = item
	}

	if taskByID[taskIDs[0]].State != TaskStateCancelled {
		t.Fatalf("expected root task cancelled from structured lifecycle event, got %#v", taskByID[taskIDs[0]].State)
	}
	if taskByID[taskIDs[1]].State != TaskStateRunning {
		t.Fatalf("expected child task running from structured lifecycle event, got %#v", taskByID[taskIDs[1]].State)
	}
	if taskByID[taskIDs[2]].State != TaskStateCompleted {
		t.Fatalf("expected third task completed from structured lifecycle event, got %#v", taskByID[taskIDs[2]].State)
	}
}

func seedRunTaskGraphState() (*AppState, string, []string) {
	state := NewAppState(nil)
	now := time.Now().UTC().Format(time.RFC3339)
	runID := "exec_task_root_" + randomHex(4)
	secondTaskID := "exec_task_child_" + randomHex(4)
	thirdTaskID := "exec_task_done_" + randomHex(4)
	conversationID := "conv_task_" + randomHex(4)
	activeExecutionID := runID

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:                conversationID,
		WorkspaceID:       localWorkspaceID,
		ProjectID:         "proj_task_" + randomHex(4),
		Name:              "RunTaskConversation",
		QueueState:        QueueStateRunning,
		DefaultMode:       PermissionModeDefault,
		ModelConfigID:     "rc_model_1",
		ActiveExecutionID: &activeExecutionID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	state.executions[runID] = Execution{
		ID:             runID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_task_" + randomHex(4),
		State:          ExecutionStateExecuting,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     0,
		TraceID:        "tr_task_" + randomHex(4),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.executions[secondTaskID] = Execution{
		ID:             secondTaskID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_task_" + randomHex(4),
		State:          ExecutionStateQueued,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     1,
		TraceID:        "tr_task_" + randomHex(4),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.executions[thirdTaskID] = Execution{
		ID:             thirdTaskID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_task_" + randomHex(4),
		State:          ExecutionStateCompleted,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     2,
		TraceID:        "tr_task_" + randomHex(4),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.conversationExecutionOrder[conversationID] = []string{runID, secondTaskID, thirdTaskID}
	state.mu.Unlock()
	return state, runID, []string{runID, secondTaskID, thirdTaskID}
}
