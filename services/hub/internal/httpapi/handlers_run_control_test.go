package httpapi

import (
	"net/http"
	"testing"
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

	first := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/messages", map[string]any{
		"content":         "first",
		"model_config_id": modelConfigID,
	}, authHeaders)
	if first.Code != http.StatusCreated {
		t.Fatalf("expected first message 201, got %d (%s)", first.Code, first.Body.String())
	}

	second := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/messages", map[string]any{
		"content":         "second",
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

	messageRes := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/messages", map[string]any{
		"content":         "stop this run",
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
