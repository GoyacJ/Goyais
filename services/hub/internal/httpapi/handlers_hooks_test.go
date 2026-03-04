package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHookRoutesAreRegistered(t *testing.T) {
	router := NewRouter()

	postRes := performJSONRequest(t, router, http.MethodPost, "/v1/hooks/policies", map[string]any{
		"id":           "policy_test_deny_write",
		"scope":        "global",
		"event":        "pre_tool_use",
		"handler_type": "agent",
		"tool_name":    "Write",
		"enabled":      true,
		"decision": map[string]any{
			"action": "deny",
			"reason": "blocked by test policy",
		},
	}, nil)
	if postRes.Code != http.StatusOK {
		t.Fatalf("expected 200 for hooks policy upsert, got %d (%s)", postRes.Code, postRes.Body.String())
	}

	listRes := performJSONRequest(t, router, http.MethodGet, "/v1/hooks/policies", nil, nil)
	if listRes.Code != http.StatusOK {
		t.Fatalf("expected 200 for hooks policy list, got %d (%s)", listRes.Code, listRes.Body.String())
	}
	listPayload := HookPolicyListResponse{}
	mustDecodeJSON(t, listRes.Body.Bytes(), &listPayload)
	if len(listPayload.Items) == 0 {
		t.Fatalf("expected at least one hook policy, got %#v", listPayload)
	}

	executionRes := performJSONRequest(t, router, http.MethodGet, "/v1/hooks/executions/missing_run", nil, nil)
	if executionRes.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for missing run hook executions, got %d (%s)", executionRes.Code, executionRes.Body.String())
	}
}

func TestHookExecutionsHandlerListsRecordsForRunConversation(t *testing.T) {
	state := NewAppState(nil)
	now := time.Now().UTC().Format(time.RFC3339)
	conversationID := "conv_hook_exec"
	runID := "exec_hook_seed"

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:            conversationID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     "proj_hook_exec",
		Name:          "Hook Exec",
		QueueState:    QueueStateRunning,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_1",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.executions[runID] = Execution{
		ID:             runID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_hook_exec",
		State:          RunStateExecuting,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     0,
		TraceID:        "tr_hook_exec",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	appendHookExecutionRecordLocked(state, HookExecutionRecord{
		ID:             "hook_exec_1",
		RunID:          runID,
		TaskID:         runID,
		ConversationID: conversationID,
		Event:          HookEventTypePreToolUse,
		ToolName:       "Write",
		PolicyID:       "policy_deny_write",
		Decision: HookDecision{
			Action: HookDecisionActionDeny,
			Reason: "blocked by test",
		},
		Timestamp: now,
	})
	state.mu.Unlock()

	handler := HookExecutionsHandler(state)
	req := httptest.NewRequest(http.MethodGet, "/v1/hooks/executions/"+runID, nil)
	req.SetPathValue("run_id", runID)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", res.Code, res.Body.String())
	}

	payload := HookExecutionListResponse{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	if len(payload.Items) != 1 {
		t.Fatalf("expected one hook execution record, got %#v", payload)
	}
	if payload.Items[0].RunID != runID || payload.Items[0].ToolName != "Write" {
		t.Fatalf("unexpected hook execution payload: %#v", payload.Items[0])
	}
}

func TestHooksPoliciesHandlerPostPersistsToStore(t *testing.T) {
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
	handler := HooksPoliciesHandler(state)

	body, err := json.Marshal(map[string]any{
		"id":           "policy_persist",
		"scope":        "project",
		"event":        "pre_tool_use",
		"handler_type": "agent",
		"tool_name":    "Write",
		"workspace_id": "ws_local",
		"project_id":   "proj_persist",
		"enabled":      true,
		"decision": map[string]any{
			"action": "deny",
			"reason": "persist test",
		},
	})
	if err != nil {
		t.Fatalf("marshal request body failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/hooks/policies", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", res.Code, res.Body.String())
	}

	loaded, err := store.loadExecutionDomainSnapshot()
	if err != nil {
		t.Fatalf("load execution domain snapshot failed: %v", err)
	}
	if len(loaded.HookPolicies) != 1 {
		t.Fatalf("expected 1 persisted hook policy, got %#v", loaded.HookPolicies)
	}
	if loaded.HookPolicies[0].ID != "policy_persist" || loaded.HookPolicies[0].Decision.Action != HookDecisionActionDeny {
		t.Fatalf("unexpected persisted hook policy: %#v", loaded.HookPolicies[0])
	}
	if loaded.HookPolicies[0].WorkspaceID != "ws_local" || loaded.HookPolicies[0].ProjectID != "proj_persist" || loaded.HookPolicies[0].ConversationID != "" {
		t.Fatalf("expected explicit scope binding fields persisted, got %#v", loaded.HookPolicies[0])
	}
}

func TestHooksPoliciesHandlerRejectsProjectScopeWithoutProjectID(t *testing.T) {
	router := NewRouter()
	res := performJSONRequest(t, router, http.MethodPost, "/v1/hooks/policies", map[string]any{
		"id":           "policy_project_missing_binding",
		"scope":        "project",
		"event":        "pre_tool_use",
		"handler_type": "agent",
		"tool_name":    "Write",
		"enabled":      true,
		"decision": map[string]any{
			"action": "deny",
			"reason": "project binding required",
		},
	}, nil)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for project scope without project_id, got %d (%s)", res.Code, res.Body.String())
	}
	errPayload := StandardError{}
	mustDecodeJSON(t, res.Body.Bytes(), &errPayload)
	if errPayload.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %#v", errPayload)
	}
	if scopeValue, ok := errPayload.Details["scope"].(string); !ok || scopeValue != "project" {
		t.Fatalf("expected scope detail project, got %#v", errPayload.Details)
	}
	if projectValue, ok := errPayload.Details["project_id"].(string); !ok || projectValue != "" {
		t.Fatalf("expected empty project_id detail, got %#v", errPayload.Details)
	}
	if validationError, ok := errPayload.Details["validation_error"].(string); !ok || validationError != "scope=project requires project_id" {
		t.Fatalf("expected project validation_error detail, got %#v", errPayload.Details)
	}
}

func TestHooksPoliciesHandlerRejectsLocalScopeWithoutConversationID(t *testing.T) {
	router := NewRouter()
	res := performJSONRequest(t, router, http.MethodPost, "/v1/hooks/policies", map[string]any{
		"id":           "policy_local_missing_binding",
		"scope":        "local",
		"event":        "pre_tool_use",
		"handler_type": "agent",
		"tool_name":    "Write",
		"enabled":      true,
		"decision": map[string]any{
			"action": "deny",
			"reason": "conversation binding required",
		},
	}, nil)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for local scope without conversation_id, got %d (%s)", res.Code, res.Body.String())
	}
	errPayload := StandardError{}
	mustDecodeJSON(t, res.Body.Bytes(), &errPayload)
	if errPayload.Code != "VALIDATION_ERROR" {
		t.Fatalf("expected VALIDATION_ERROR, got %#v", errPayload)
	}
	if scopeValue, ok := errPayload.Details["scope"].(string); !ok || scopeValue != "local" {
		t.Fatalf("expected scope detail local, got %#v", errPayload.Details)
	}
	if conversationValue, ok := errPayload.Details["conversation_id"].(string); !ok || conversationValue != "" {
		t.Fatalf("expected empty conversation_id detail, got %#v", errPayload.Details)
	}
	if validationError, ok := errPayload.Details["validation_error"].(string); !ok || validationError != "scope=local requires conversation_id" {
		t.Fatalf("expected local validation_error detail, got %#v", errPayload.Details)
	}
}

func TestHooksPoliciesHandlerRejectsGlobalScopeWithProjectBinding(t *testing.T) {
	router := NewRouter()
	res := performJSONRequest(t, router, http.MethodPost, "/v1/hooks/policies", map[string]any{
		"id":           "policy_global_invalid_project_binding",
		"scope":        "global",
		"event":        "pre_tool_use",
		"handler_type": "agent",
		"tool_name":    "Write",
		"project_id":   "proj_should_not_exist",
		"enabled":      true,
		"decision": map[string]any{
			"action": "deny",
			"reason": "global should not bind project",
		},
	}, nil)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for global scope with project_id, got %d (%s)", res.Code, res.Body.String())
	}
	errPayload := StandardError{}
	mustDecodeJSON(t, res.Body.Bytes(), &errPayload)
	if validationError, ok := errPayload.Details["validation_error"].(string); !ok || validationError != "scope=global does not allow project_id" {
		t.Fatalf("expected global validation_error detail, got %#v", errPayload.Details)
	}
}

func TestHooksPoliciesHandlerRejectsLocalScopeWithProjectBinding(t *testing.T) {
	router := NewRouter()
	res := performJSONRequest(t, router, http.MethodPost, "/v1/hooks/policies", map[string]any{
		"id":              "policy_local_invalid_project_binding",
		"scope":           "local",
		"event":           "pre_tool_use",
		"handler_type":    "agent",
		"tool_name":       "Write",
		"project_id":      "proj_should_not_exist",
		"conversation_id": "conv_valid",
		"enabled":         true,
		"decision": map[string]any{
			"action": "deny",
			"reason": "local should not bind project",
		},
	}, nil)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for local scope with project_id, got %d (%s)", res.Code, res.Body.String())
	}
	errPayload := StandardError{}
	mustDecodeJSON(t, res.Body.Bytes(), &errPayload)
	if validationError, ok := errPayload.Details["validation_error"].(string); !ok || validationError != "scope=local does not allow project_id" {
		t.Fatalf("expected local-project validation_error detail, got %#v", errPayload.Details)
	}
}

func TestHooksPoliciesHandlerAcceptsConfigChangeEvent(t *testing.T) {
	router := NewRouter()
	res := performJSONRequest(t, router, http.MethodPost, "/v1/hooks/policies", map[string]any{
		"id":           "policy_config_change_allow",
		"scope":        "global",
		"event":        "config_change",
		"handler_type": "agent",
		"enabled":      true,
		"decision": map[string]any{
			"action": "allow",
			"reason": "allow config change event",
		},
	}, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200 for config_change hook policy, got %d (%s)", res.Code, res.Body.String())
	}
	payload := HookPolicy{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	if payload.Event != HookEventType("config_change") {
		t.Fatalf("expected event config_change, got %#v", payload)
	}
}
