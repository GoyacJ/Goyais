package httpapi

import (
	"net/http"
	"testing"
)

func TestInternalExecutionClaimAndControlFlow(t *testing.T) {
	router := NewRouter()
	workspaceID := createRemoteWorkspace(t, router, "Remote Worker Flow", "http://127.0.0.1:9100", false)
	token := loginRemoteWorkspace(t, router, workspaceID, "worker_flow_user", "pw", RoleDeveloper, true)
	authHeaders := map[string]string{"Authorization": "Bearer " + token}

	projectRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/import", map[string]any{
		"workspace_id":   workspaceID,
		"directory_path": "/tmp/worker-claim-flow",
	}, authHeaders)
	if projectRes.Code != http.StatusCreated {
		t.Fatalf("expected import project 201, got %d (%s)", projectRes.Code, projectRes.Body.String())
	}
	projectPayload := map[string]any{}
	mustDecodeJSON(t, projectRes.Body.Bytes(), &projectPayload)
	projectID := projectPayload["id"].(string)

	conversationRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/"+projectID+"/conversations", map[string]any{
		"workspace_id": workspaceID,
		"name":         "ClaimFlow",
	}, authHeaders)
	if conversationRes.Code != http.StatusCreated {
		t.Fatalf("expected create conversation 201, got %d (%s)", conversationRes.Code, conversationRes.Body.String())
	}
	conversationPayload := map[string]any{}
	mustDecodeJSON(t, conversationRes.Body.Bytes(), &conversationPayload)
	conversationID := conversationPayload["id"].(string)

	messageRes := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/messages", map[string]any{
		"content": "read README and explain",
	}, authHeaders)
	if messageRes.Code != http.StatusCreated {
		t.Fatalf("expected create execution 201, got %d (%s)", messageRes.Code, messageRes.Body.String())
	}
	messagePayload := map[string]any{}
	mustDecodeJSON(t, messageRes.Body.Bytes(), &messagePayload)
	executionID := messagePayload["execution"].(map[string]any)["id"].(string)

	internalHeaders := map[string]string{"X-Internal-Token": defaultHubInternalToken}
	claimRes := performJSONRequest(t, router, http.MethodPost, "/internal/executions/claim", map[string]any{
		"worker_id": "worker-test-1",
	}, internalHeaders)
	if claimRes.Code != http.StatusOK {
		t.Fatalf("expected claim 200, got %d (%s)", claimRes.Code, claimRes.Body.String())
	}
	claimPayload := map[string]any{}
	mustDecodeJSON(t, claimRes.Body.Bytes(), &claimPayload)
	if claimed, _ := claimPayload["claimed"].(bool); !claimed {
		t.Fatalf("expected claimed=true, got %#v", claimPayload)
	}
	executionEnvelope := claimPayload["execution"].(map[string]any)
	if executionEnvelope["content"] == "" {
		t.Fatalf("expected claim envelope with content")
	}

	confirmRes := performJSONRequest(t, router, http.MethodPost, "/v1/executions/"+executionID+"/confirm", map[string]any{
		"decision": "approve",
	}, authHeaders)
	if confirmRes.Code != http.StatusOK {
		t.Fatalf("expected confirm 200, got %d (%s)", confirmRes.Code, confirmRes.Body.String())
	}

	controlRes := performJSONRequest(t, router, http.MethodGet, "/internal/executions/"+executionID+"/control?after_seq=0&wait_ms=0", nil, internalHeaders)
	if controlRes.Code != http.StatusOK {
		t.Fatalf("expected control poll 200, got %d (%s)", controlRes.Code, controlRes.Body.String())
	}
	controlPayload := map[string]any{}
	mustDecodeJSON(t, controlRes.Body.Bytes(), &controlPayload)
	commands := controlPayload["commands"].([]any)
	if len(commands) == 0 {
		t.Fatalf("expected confirm command in control poll")
	}
	first := commands[0].(map[string]any)
	if first["type"] != "confirm" {
		t.Fatalf("expected command type confirm, got %#v", first["type"])
	}

	eventBatchRes := performJSONRequest(t, router, http.MethodPost, "/internal/executions/"+executionID+"/events/batch", map[string]any{
		"events": []map[string]any{
			{
				"event_id":        "evt_started",
				"execution_id":    executionID,
				"conversation_id": conversationID,
				"type":            "execution_started",
				"sequence":        1,
				"queue_index":     0,
				"payload":         map[string]any{},
			},
			{
				"event_id":        "evt_done",
				"execution_id":    executionID,
				"conversation_id": conversationID,
				"type":            "execution_done",
				"sequence":        2,
				"queue_index":     0,
				"payload":         map[string]any{"content": "done"},
			},
		},
	}, internalHeaders)
	if eventBatchRes.Code != http.StatusAccepted {
		t.Fatalf("expected events batch 202, got %d (%s)", eventBatchRes.Code, eventBatchRes.Body.String())
	}

	executionsRes := performJSONRequest(t, router, http.MethodGet, "/v1/executions?conversation_id="+conversationID, nil, authHeaders)
	if executionsRes.Code != http.StatusOK {
		t.Fatalf("expected execution list 200, got %d", executionsRes.Code)
	}
	listPayload := map[string]any{}
	mustDecodeJSON(t, executionsRes.Body.Bytes(), &listPayload)
	items := listPayload["items"].([]any)
	if len(items) == 0 {
		t.Fatalf("expected at least one execution")
	}
	state := items[0].(map[string]any)["state"]
	if state != "completed" {
		t.Fatalf("expected completed execution state, got %#v", state)
	}
}
