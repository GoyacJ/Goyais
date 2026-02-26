package httpapi

import (
	"net/http"
	"strconv"
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
		"content": "first",
	}, authHeaders)
	if first.Code != http.StatusCreated {
		t.Fatalf("expected first message 201, got %d (%s)", first.Code, first.Body.String())
	}

	second := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/messages", map[string]any{
		"content": "second",
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
	if controlRes.Code != http.StatusOK {
		t.Fatalf("expected run control deny 200, got %d (%s)", controlRes.Code, controlRes.Body.String())
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
	if stateByExecutionID[runID] != string(ExecutionStateCancelled) {
		t.Fatalf("expected run %s to be cancelled, got %q", runID, stateByExecutionID[runID])
	}
}

func TestRunControlEndpoint_StopPropagatesToWorkerControlPoll(t *testing.T) {
	router := NewRouter()
	workspaceID := createRemoteWorkspace(t, router, "Remote Run Control Stop Poll", "http://127.0.0.1:9121", false)
	token := loginRemoteWorkspace(t, router, workspaceID, "run_control_stop_user", "pw", RoleDeveloper, true)
	authHeaders := map[string]string{"Authorization": "Bearer " + token}
	internalHeaders := internalAuthHeaders(t)

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
		"content": "stop this run",
	}, authHeaders)
	if messageRes.Code != http.StatusCreated {
		t.Fatalf("expected create execution 201, got %d (%s)", messageRes.Code, messageRes.Body.String())
	}
	messagePayload := map[string]any{}
	mustDecodeJSON(t, messageRes.Body.Bytes(), &messagePayload)
	runID := messagePayload["execution"].(map[string]any)["id"].(string)

	claimRes := performJSONRequest(t, router, http.MethodPost, "/internal/executions/claim", map[string]any{
		"worker_id": "worker-stop-propagation-1",
	}, internalHeaders)
	if claimRes.Code != http.StatusOK {
		t.Fatalf("expected claim 200, got %d (%s)", claimRes.Code, claimRes.Body.String())
	}
	claimPayload := map[string]any{}
	mustDecodeJSON(t, claimRes.Body.Bytes(), &claimPayload)
	if claimed, _ := claimPayload["claimed"].(bool); !claimed {
		t.Fatalf("expected claimed=true, got %#v", claimPayload)
	}

	controlRes := performJSONRequest(t, router, http.MethodPost, "/v1/runs/"+runID+"/control", map[string]any{
		"action": "stop",
	}, authHeaders)
	if controlRes.Code != http.StatusOK {
		t.Fatalf("expected run control stop 200, got %d (%s)", controlRes.Code, controlRes.Body.String())
	}
	controlPayload := map[string]any{}
	mustDecodeJSON(t, controlRes.Body.Bytes(), &controlPayload)
	if controlPayload["state"] != string(ExecutionStateCancelled) {
		t.Fatalf("expected cancelled state, got %#v", controlPayload["state"])
	}

	pollRes := performJSONRequest(t, router, http.MethodGet, "/internal/executions/"+runID+"/control?after_seq=0&wait_ms=0", nil, internalHeaders)
	if pollRes.Code != http.StatusOK {
		t.Fatalf("expected control poll 200, got %d (%s)", pollRes.Code, pollRes.Body.String())
	}
	pollPayload := map[string]any{}
	mustDecodeJSON(t, pollRes.Body.Bytes(), &pollPayload)
	commands := pollPayload["commands"].([]any)
	if len(commands) != 1 {
		t.Fatalf("expected 1 control command, got %d (%#v)", len(commands), commands)
	}
	command := commands[0].(map[string]any)
	if command["type"] != string(ExecutionControlCommandTypeStop) {
		t.Fatalf("expected command type stop, got %#v", command["type"])
	}
	payload := command["payload"].(map[string]any)
	if payload["action"] != "stop" {
		t.Fatalf("expected control payload action stop, got %#v", payload["action"])
	}
	if payload["source"] != "run_control" {
		t.Fatalf("expected control payload source run_control, got %#v", payload["source"])
	}

	lastSeq := int(pollPayload["last_seq"].(float64))
	if lastSeq <= 0 {
		t.Fatalf("expected last_seq > 0, got %d", lastSeq)
	}

	rePollRes := performJSONRequest(t, router, http.MethodGet, "/internal/executions/"+runID+"/control?after_seq="+strconv.Itoa(lastSeq)+"&wait_ms=0", nil, internalHeaders)
	if rePollRes.Code != http.StatusOK {
		t.Fatalf("expected control repoll 200, got %d (%s)", rePollRes.Code, rePollRes.Body.String())
	}
	rePollPayload := map[string]any{}
	mustDecodeJSON(t, rePollRes.Body.Bytes(), &rePollPayload)
	rePollCommands := rePollPayload["commands"].([]any)
	if len(rePollCommands) != 0 {
		t.Fatalf("expected no incremental commands after last_seq, got %#v", rePollCommands)
	}
}
