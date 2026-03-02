package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRunControlEndpoint_DenyQueuedRun(t *testing.T) {
	router := NewRouter()
	workspaceID := createRemoteWorkspace(t, router, "Remote Run Control", "http://127.0.0.1:9120", false)
	token := loginRemoteWorkspace(t, router, workspaceID, "run_control_user", "pw", RoleDeveloper, true)
	authHeaders := map[string]string{"Authorization": "Bearer " + token}

	projectRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/import", map[string]any{
		"workspace_id":   workspaceID,
		"directory_path": "/tmp/run-control-alpha",
	}, authHeaders)
	if projectRes.Code != http.StatusCreated {
		t.Fatalf("expected import project 201, got %d (%s)", projectRes.Code, projectRes.Body.String())
	}
	projectPayload := map[string]any{}
	mustDecodeJSON(t, projectRes.Body.Bytes(), &projectPayload)
	projectID := projectPayload["id"].(string)
	modelConfigID := createModelResourceConfigForTest(t, router, workspaceID, authHeaders, "OpenAI", "gpt-5.3")
	bindProjectConfigWithModelForTest(t, router, projectID, modelConfigID, authHeaders)

	conversationRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/"+projectID+"/conversations", map[string]any{
		"workspace_id": workspaceID,
		"name":         "RunControlConv",
	}, authHeaders)
	if conversationRes.Code != http.StatusCreated {
		t.Fatalf("expected create conversation 201, got %d (%s)", conversationRes.Code, conversationRes.Body.String())
	}
	conversationPayload := map[string]any{}
	mustDecodeJSON(t, conversationRes.Body.Bytes(), &conversationPayload)
	conversationID := conversationPayload["id"].(string)

	first := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/input/submit", map[string]any{
		"raw_input":       "first",
		"model_config_id": modelConfigID,
	}, authHeaders)
	if first.Code != http.StatusCreated {
		t.Fatalf("expected first message 201, got %d (%s)", first.Code, first.Body.String())
	}

	second := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/input/submit", map[string]any{
		"raw_input":       "second",
		"model_config_id": modelConfigID,
	}, authHeaders)
	if second.Code != http.StatusCreated {
		t.Fatalf("expected second message 201, got %d (%s)", second.Code, second.Body.String())
	}
	secondPayload := map[string]any{}
	mustDecodeJSON(t, second.Body.Bytes(), &secondPayload)
	runID := secondPayload["execution"].(map[string]any)["id"].(string)

	controlRes := performJSONRequest(t, router, http.MethodPost, "/v1/runs/"+runID+"/control", map[string]any{
		"action": "deny",
	}, authHeaders)
	if controlRes.Code != http.StatusOK && controlRes.Code != http.StatusConflict {
		t.Fatalf("expected run control deny 200/409, got %d (%s)", controlRes.Code, controlRes.Body.String())
	}

	executionsRes := performJSONRequest(t, router, http.MethodGet, "/v1/executions?conversation_id="+conversationID, nil, authHeaders)
	if executionsRes.Code != http.StatusOK {
		t.Fatalf("expected list executions 200, got %d (%s)", executionsRes.Code, executionsRes.Body.String())
	}
	listPayload := map[string]any{}
	mustDecodeJSON(t, executionsRes.Body.Bytes(), &listPayload)

	items := listPayload["items"].([]any)
	stateByExecutionID := map[string]string{}
	for _, raw := range items {
		item := raw.(map[string]any)
		stateByExecutionID[item["id"].(string)] = item["state"].(string)
	}
	if stateByExecutionID[runID] != string(ExecutionStateCancelled) && stateByExecutionID[runID] != string(ExecutionStateFailed) {
		t.Fatalf("expected run %s to be cancelled/failed, got %q", runID, stateByExecutionID[runID])
	}
}

func TestRunControlEndpoint_StopTransitionsRun(t *testing.T) {
	router := NewRouter()
	workspaceID := createRemoteWorkspace(t, router, "Remote Run Control Stop Poll", "http://127.0.0.1:9121", false)
	token := loginRemoteWorkspace(t, router, workspaceID, "run_control_stop_user", "pw", RoleDeveloper, true)
	authHeaders := map[string]string{"Authorization": "Bearer " + token}

	projectRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/import", map[string]any{
		"workspace_id":   workspaceID,
		"directory_path": "/tmp/run-control-stop-poll",
	}, authHeaders)
	if projectRes.Code != http.StatusCreated {
		t.Fatalf("expected import project 201, got %d (%s)", projectRes.Code, projectRes.Body.String())
	}
	projectPayload := map[string]any{}
	mustDecodeJSON(t, projectRes.Body.Bytes(), &projectPayload)
	projectID := projectPayload["id"].(string)
	modelConfigID := createModelResourceConfigForTest(t, router, workspaceID, authHeaders, "OpenAI", "gpt-5.3")
	bindProjectConfigWithModelForTest(t, router, projectID, modelConfigID, authHeaders)

	conversationRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/"+projectID+"/conversations", map[string]any{
		"workspace_id": workspaceID,
		"name":         "RunControlStopPollConv",
	}, authHeaders)
	if conversationRes.Code != http.StatusCreated {
		t.Fatalf("expected create conversation 201, got %d (%s)", conversationRes.Code, conversationRes.Body.String())
	}
	conversationPayload := map[string]any{}
	mustDecodeJSON(t, conversationRes.Body.Bytes(), &conversationPayload)
	conversationID := conversationPayload["id"].(string)

	messageRes := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/input/submit", map[string]any{
		"raw_input":       "stop this run",
		"model_config_id": modelConfigID,
	}, authHeaders)
	if messageRes.Code != http.StatusCreated {
		t.Fatalf("expected create execution 201, got %d (%s)", messageRes.Code, messageRes.Body.String())
	}
	messagePayload := map[string]any{}
	mustDecodeJSON(t, messageRes.Body.Bytes(), &messagePayload)
	runID := messagePayload["execution"].(map[string]any)["id"].(string)

	controlRes := performJSONRequest(t, router, http.MethodPost, "/v1/runs/"+runID+"/control", map[string]any{
		"action": "stop",
	}, authHeaders)
	if controlRes.Code != http.StatusOK && controlRes.Code != http.StatusConflict {
		t.Fatalf("expected run control stop 200/409, got %d (%s)", controlRes.Code, controlRes.Body.String())
	}
	controlPayload := map[string]any{}
	mustDecodeJSON(t, controlRes.Body.Bytes(), &controlPayload)
	if controlRes.Code == http.StatusOK && controlPayload["state"] != string(ExecutionStateCancelled) {
		t.Fatalf("expected cancelled state on successful stop, got %#v", controlPayload["state"])
	}
}

func TestRunControlEndpoint_DenyConfirmingRunResumesExecution(t *testing.T) {
	state := NewAppState(nil)
	handler := RunControlHandler(state)

	now := time.Now().UTC().Format(time.RFC3339)
	conversationID := "conv_confirming_" + randomHex(4)
	executionID := "exec_confirming_" + randomHex(4)
	activeExecutionID := executionID

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:                conversationID,
		WorkspaceID:       localWorkspaceID,
		ProjectID:         "proj_" + randomHex(4),
		Name:              "Confirming Run",
		QueueState:        QueueStateRunning,
		DefaultMode:       PermissionModeDefault,
		ModelConfigID:     "rc_model_test",
		ActiveExecutionID: &activeExecutionID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	state.executions[executionID] = Execution{
		ID:             executionID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_" + randomHex(4),
		State:          ExecutionStateConfirming,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     0,
		TraceID:        "tr_" + randomHex(4),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.conversationExecutionOrder[conversationID] = []string{executionID}
	state.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/v1/runs/"+executionID+"/control", strings.NewReader(`{"action":"deny"}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("run_id", executionID)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected run control deny for confirming run 200, got %d (%s)", res.Code, res.Body.String())
	}

	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	if got := strings.TrimSpace(asString(payload["state"])); got != string(ExecutionStateExecuting) {
		t.Fatalf("expected state executing after deny confirming run, got %q", got)
	}
}

func TestRunControlEndpoint_AnswerAwaitingInputTransitionsToExecuting(t *testing.T) {
	state := NewAppState(nil)
	handler := RunControlHandler(state)

	now := time.Now().UTC().Format(time.RFC3339)
	conversationID := "conv_awaiting_" + randomHex(4)
	executionID := "exec_awaiting_" + randomHex(4)
	activeExecutionID := executionID

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:                conversationID,
		WorkspaceID:       localWorkspaceID,
		ProjectID:         "proj_" + randomHex(4),
		Name:              "Awaiting Input Run",
		QueueState:        QueueStateRunning,
		DefaultMode:       PermissionModeDefault,
		ModelConfigID:     "rc_model_test",
		ActiveExecutionID: &activeExecutionID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	state.executions[executionID] = Execution{
		ID:             executionID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_" + randomHex(4),
		State:          ExecutionStateAwaitingInput,
		Mode:           PermissionModePlan,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModePlan,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     0,
		TraceID:        "tr_" + randomHex(4),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.pendingUserQuestions[executionID] = pendingUserQuestion{
		QuestionID: "q_choose_mode",
		Question:   "Choose mode",
		Options: []map[string]any{
			{"id": "opt_default", "label": "Default"},
			{"id": "opt_plan", "label": "Plan"},
		},
		AllowText: false,
		Required:  true,
	}
	state.conversationExecutionOrder[conversationID] = []string{executionID}
	state.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/v1/runs/"+executionID+"/control", strings.NewReader(`{"action":"answer","answer":{"question_id":"q_choose_mode","selected_option_id":"opt_plan"}}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("run_id", executionID)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected run control answer 200, got %d (%s)", res.Code, res.Body.String())
	}

	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	if got := strings.TrimSpace(asString(payload["state"])); got != string(ExecutionStateExecuting) {
		t.Fatalf("expected state executing after answer, got %q", got)
	}

	state.mu.RLock()
	_, pendingExists := state.pendingUserQuestions[executionID]
	state.mu.RUnlock()
	if pendingExists {
		t.Fatalf("expected pending question cleared for execution %s", executionID)
	}
}

func TestRunControlEndpoint_AnswerRejectsMismatchedQuestionID(t *testing.T) {
	state := NewAppState(nil)
	handler := RunControlHandler(state)

	now := time.Now().UTC().Format(time.RFC3339)
	conversationID := "conv_awaiting_mismatch_" + randomHex(4)
	executionID := "exec_awaiting_mismatch_" + randomHex(4)
	activeExecutionID := executionID

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:                conversationID,
		WorkspaceID:       localWorkspaceID,
		ProjectID:         "proj_" + randomHex(4),
		Name:              "Awaiting Input Mismatch",
		QueueState:        QueueStateRunning,
		DefaultMode:       PermissionModeDefault,
		ModelConfigID:     "rc_model_test",
		ActiveExecutionID: &activeExecutionID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	state.executions[executionID] = Execution{
		ID:             executionID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_" + randomHex(4),
		State:          ExecutionStateAwaitingInput,
		Mode:           PermissionModePlan,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModePlan,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     0,
		TraceID:        "tr_" + randomHex(4),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.pendingUserQuestions[executionID] = pendingUserQuestion{
		QuestionID: "q_expected",
		Question:   "Choose one option",
		Options: []map[string]any{
			{"id": "opt_a", "label": "A"},
		},
		AllowText: true,
		Required:  true,
	}
	state.conversationExecutionOrder[conversationID] = []string{executionID}
	state.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/v1/runs/"+executionID+"/control", strings.NewReader(`{"action":"answer","answer":{"question_id":"q_other","selected_option_id":"opt_a"}}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("run_id", executionID)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected run control answer mismatch 400, got %d (%s)", res.Code, res.Body.String())
	}

	state.mu.RLock()
	execution := state.executions[executionID]
	_, pendingExists := state.pendingUserQuestions[executionID]
	state.mu.RUnlock()
	if execution.State != ExecutionStateAwaitingInput {
		t.Fatalf("expected execution state awaiting_input after mismatch, got %s", execution.State)
	}
	if !pendingExists {
		t.Fatalf("expected pending question to remain after mismatch")
	}
}

func TestRunControlEndpoint_AnswerRejectsInvalidOptionAndDuplicateAnswer(t *testing.T) {
	state := NewAppState(nil)
	handler := RunControlHandler(state)

	now := time.Now().UTC().Format(time.RFC3339)
	conversationID := "conv_awaiting_option_" + randomHex(4)
	executionID := "exec_awaiting_option_" + randomHex(4)
	activeExecutionID := executionID

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:                conversationID,
		WorkspaceID:       localWorkspaceID,
		ProjectID:         "proj_" + randomHex(4),
		Name:              "Awaiting Input Option Validation",
		QueueState:        QueueStateRunning,
		DefaultMode:       PermissionModeDefault,
		ModelConfigID:     "rc_model_test",
		ActiveExecutionID: &activeExecutionID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	state.executions[executionID] = Execution{
		ID:             executionID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_" + randomHex(4),
		State:          ExecutionStateAwaitingInput,
		Mode:           PermissionModePlan,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModePlan,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     0,
		TraceID:        "tr_" + randomHex(4),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.pendingUserQuestions[executionID] = pendingUserQuestion{
		QuestionID: "q_option",
		Question:   "Choose an option",
		Options: []map[string]any{
			{"id": "opt_valid", "label": "Valid"},
		},
		AllowText: true,
		Required:  true,
	}
	state.conversationExecutionOrder[conversationID] = []string{executionID}
	state.mu.Unlock()

	invalidReq := httptest.NewRequest(http.MethodPost, "/v1/runs/"+executionID+"/control", strings.NewReader(`{"action":"answer","answer":{"question_id":"q_option","selected_option_id":"opt_invalid"}}`))
	invalidReq.Header.Set("Content-Type", "application/json")
	invalidReq.SetPathValue("run_id", executionID)
	invalidRes := httptest.NewRecorder()
	handler.ServeHTTP(invalidRes, invalidReq)
	if invalidRes.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid option 400, got %d (%s)", invalidRes.Code, invalidRes.Body.String())
	}

	validReq := httptest.NewRequest(http.MethodPost, "/v1/runs/"+executionID+"/control", strings.NewReader(`{"action":"answer","answer":{"question_id":"q_option","selected_option_id":"opt_valid"}}`))
	validReq.Header.Set("Content-Type", "application/json")
	validReq.SetPathValue("run_id", executionID)
	validRes := httptest.NewRecorder()
	handler.ServeHTTP(validRes, validReq)
	if validRes.Code != http.StatusOK {
		t.Fatalf("expected valid answer 200, got %d (%s)", validRes.Code, validRes.Body.String())
	}

	duplicateReq := httptest.NewRequest(http.MethodPost, "/v1/runs/"+executionID+"/control", strings.NewReader(`{"action":"answer","answer":{"question_id":"q_option","selected_option_id":"opt_valid"}}`))
	duplicateReq.Header.Set("Content-Type", "application/json")
	duplicateReq.SetPathValue("run_id", executionID)
	duplicateRes := httptest.NewRecorder()
	handler.ServeHTTP(duplicateRes, duplicateReq)
	if duplicateRes.Code != http.StatusConflict {
		t.Fatalf("expected duplicate answer 409, got %d (%s)", duplicateRes.Code, duplicateRes.Body.String())
	}
}

func TestRunControlEndpoint_StopQueuedRunEmitsTaskCancelledEvent(t *testing.T) {
	state := NewAppState(nil)
	handler := RunControlHandler(state)

	now := time.Now().UTC().Format(time.RFC3339)
	conversationID := "conv_stop_queued_" + randomHex(4)
	executionID := "exec_stop_queued_" + randomHex(4)

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:            conversationID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     "proj_" + randomHex(4),
		Name:          "Stop Queued Run",
		QueueState:    QueueStateQueued,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_test",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.executions[executionID] = Execution{
		ID:             executionID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_" + randomHex(4),
		State:          ExecutionStateQueued,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     0,
		TraceID:        "tr_" + randomHex(4),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.conversationExecutionOrder[conversationID] = []string{executionID}
	state.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/v1/runs/"+executionID+"/control", strings.NewReader(`{"action":"stop"}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("run_id", executionID)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected stop queued run 200, got %d (%s)", res.Code, res.Body.String())
	}

	state.mu.RLock()
	execution := state.executions[executionID]
	events := append([]ExecutionEvent{}, state.executionEvents[conversationID]...)
	state.mu.RUnlock()
	if execution.State != ExecutionStateCancelled {
		t.Fatalf("expected execution cancelled, got %s", execution.State)
	}

	foundTaskCancelled := false
	for _, event := range events {
		if event.Type != ExecutionEventTypeTaskCancelled {
			continue
		}
		if strings.TrimSpace(asString(event.Payload["task_id"])) != executionID {
			continue
		}
		if strings.TrimSpace(asString(event.Payload["source"])) != "run_control" {
			t.Fatalf("expected task_cancelled source run_control, got %#v", event.Payload)
		}
		foundTaskCancelled = true
	}
	if !foundTaskCancelled {
		t.Fatalf("expected task_cancelled event, got %#v", events)
	}
}

func TestRunControlEndpoint_DenyQueuedRunEmitsTaskCancelledEvent(t *testing.T) {
	state := NewAppState(nil)
	handler := RunControlHandler(state)

	now := time.Now().UTC().Format(time.RFC3339)
	conversationID := "conv_deny_queued_" + randomHex(4)
	executionID := "exec_deny_queued_" + randomHex(4)

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:            conversationID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     "proj_" + randomHex(4),
		Name:          "Deny Queued Run",
		QueueState:    QueueStateQueued,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_test",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.executions[executionID] = Execution{
		ID:             executionID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_" + randomHex(4),
		State:          ExecutionStateQueued,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     0,
		TraceID:        "tr_" + randomHex(4),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.conversationExecutionOrder[conversationID] = []string{executionID}
	state.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/v1/runs/"+executionID+"/control", strings.NewReader(`{"action":"deny"}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("run_id", executionID)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected deny queued run 200, got %d (%s)", res.Code, res.Body.String())
	}

	state.mu.RLock()
	execution := state.executions[executionID]
	events := append([]ExecutionEvent{}, state.executionEvents[conversationID]...)
	state.mu.RUnlock()
	if execution.State != ExecutionStateCancelled {
		t.Fatalf("expected execution cancelled, got %s", execution.State)
	}

	foundTaskCancelled := false
	for _, event := range events {
		if event.Type != ExecutionEventTypeTaskCancelled {
			continue
		}
		if strings.TrimSpace(asString(event.Payload["task_id"])) != executionID {
			continue
		}
		if strings.TrimSpace(asString(event.Payload["source"])) != "run_control" {
			t.Fatalf("expected task_cancelled source run_control, got %#v", event.Payload)
		}
		if strings.TrimSpace(asString(event.Payload["action"])) != "deny" {
			t.Fatalf("expected task_cancelled action deny, got %#v", event.Payload)
		}
		foundTaskCancelled = true
	}
	if !foundTaskCancelled {
		t.Fatalf("expected task_cancelled event, got %#v", events)
	}
}

func TestRunControlEndpoint_StopEmitsHookStopRecord(t *testing.T) {
	state := NewAppState(nil)
	handler := RunControlHandler(state)

	now := time.Now().UTC().Format(time.RFC3339)
	conversationID := "conv_stop_hook_" + randomHex(4)
	executionID := "exec_stop_hook_" + randomHex(4)
	activeExecutionID := executionID

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:                conversationID,
		WorkspaceID:       localWorkspaceID,
		ProjectID:         "proj_" + randomHex(4),
		Name:              "Stop Hook Run",
		QueueState:        QueueStateRunning,
		DefaultMode:       PermissionModeDefault,
		ModelConfigID:     "rc_model_test",
		ActiveExecutionID: &activeExecutionID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	state.executions[executionID] = Execution{
		ID:             executionID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_" + randomHex(4),
		State:          ExecutionStateExecuting,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     0,
		TraceID:        "tr_" + randomHex(4),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.conversationExecutionOrder[conversationID] = []string{executionID}
	state.hookPolicies["policy_stop_deny"] = HookPolicy{
		ID:          "policy_stop_deny",
		Scope:       HookScopeGlobal,
		Event:       HookEventTypeStop,
		HandlerType: HookHandlerTypePlugin,
		Enabled:     true,
		Decision: HookDecision{
			Action: HookDecisionActionDeny,
			Reason: "test stop hook deny",
		},
		UpdatedAt: now,
	}
	state.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/v1/runs/"+executionID+"/control", strings.NewReader(`{"action":"stop"}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("run_id", executionID)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected run control stop 200, got %d (%s)", res.Code, res.Body.String())
	}

	state.mu.RLock()
	records := append([]HookExecutionRecord{}, state.hookExecutionRecords[conversationID]...)
	state.mu.RUnlock()

	foundHookRecord := false
	for _, record := range records {
		if record.RunID != executionID || record.Event != HookEventTypeStop {
			continue
		}
		if record.PolicyID != "policy_stop_deny" || record.Decision.Action != HookDecisionActionDeny {
			t.Fatalf("unexpected stop hook record: %#v", record)
		}
		foundHookRecord = true
	}
	if !foundHookRecord {
		t.Fatalf("expected stop hook record for run %s, got %#v", executionID, records)
	}
}
