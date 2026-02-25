package httpapi

import (
	"net/http"
	"testing"
)

func TestInternalRoutesRejectWhenTokenNotConfigured(t *testing.T) {
	t.Setenv("HUB_INTERNAL_TOKEN", "")
	t.Setenv("GOYAIS_ALLOW_INSECURE_INTERNAL_TOKEN", "")

	router := NewRouter()
	res := performJSONRequest(t, router, http.MethodPost, "/internal/executions/claim", map[string]any{
		"worker_id": "worker-test-1",
	}, map[string]string{"X-Internal-Token": defaultHubInternalToken})
	if res.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 when internal token is not configured, got %d (%s)", res.Code, res.Body.String())
	}
}

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

	internalHeaders := internalAuthHeaders(t)
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

	controlRes := performJSONRequest(t, router, http.MethodGet, "/internal/executions/"+executionID+"/control?after_seq=0&wait_ms=0", nil, internalHeaders)
	if controlRes.Code != http.StatusOK {
		t.Fatalf("expected control poll 200, got %d (%s)", controlRes.Code, controlRes.Body.String())
	}
	controlPayload := map[string]any{}
	mustDecodeJSON(t, controlRes.Body.Bytes(), &controlPayload)
	commands := controlPayload["commands"].([]any)
	if len(commands) != 0 {
		t.Fatalf("expected no control command by default, got %#v", commands)
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
				"payload": map[string]any{
					"content": "done",
					"usage": map[string]any{
						"input_tokens":  31,
						"output_tokens": 12,
						"total_tokens":  43,
					},
				},
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
	if got := int(items[0].(map[string]any)["tokens_in"].(float64)); got != 31 {
		t.Fatalf("expected tokens_in=31, got %d", got)
	}
	if got := int(items[0].(map[string]any)["tokens_out"].(float64)); got != 12 {
		t.Fatalf("expected tokens_out=12, got %d", got)
	}
}

func TestInternalExecutionClaimHydratesModelSnapshotFromConfig(t *testing.T) {
	router := NewRouter()
	workspaceID := createRemoteWorkspace(t, router, "Remote Worker Model Snapshot", "http://127.0.0.1:9101", false)
	token := loginRemoteWorkspace(t, router, workspaceID, "worker_model_user", "pw", RoleDeveloper, true)
	authHeaders := map[string]string{"Authorization": "Bearer " + token}
	internalHeaders := internalAuthHeaders(t)

	projectRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/import", map[string]any{
		"workspace_id":   workspaceID,
		"directory_path": "/tmp/worker-model-snapshot",
	}, authHeaders)
	if projectRes.Code != http.StatusCreated {
		t.Fatalf("expected import project 201, got %d (%s)", projectRes.Code, projectRes.Body.String())
	}
	projectPayload := map[string]any{}
	mustDecodeJSON(t, projectRes.Body.Bytes(), &projectPayload)
	projectID := projectPayload["id"].(string)

	modelConfigRes := performJSONRequest(t, router, http.MethodPost, "/v1/workspaces/"+workspaceID+"/resource-configs", map[string]any{
		"type":    "model",
		"enabled": true,
		"model": map[string]any{
			"vendor":   "MiniMax",
			"model_id": "MiniMax-M2.5",
			"api_key":  "minimax-key",
		},
	}, authHeaders)
	if modelConfigRes.Code != http.StatusCreated {
		t.Fatalf("expected create model config 201, got %d (%s)", modelConfigRes.Code, modelConfigRes.Body.String())
	}
	modelConfigPayload := map[string]any{}
	mustDecodeJSON(t, modelConfigRes.Body.Bytes(), &modelConfigPayload)
	modelConfigID := modelConfigPayload["id"].(string)

	putConfigRes := performJSONRequest(t, router, http.MethodPut, "/v1/projects/"+projectID+"/config", map[string]any{
		"project_id":       projectID,
		"model_ids":        []string{modelConfigID},
		"default_model_id": modelConfigID,
		"rule_ids":         []string{},
		"skill_ids":        []string{},
		"mcp_ids":          []string{},
		"updated_at":       "",
	}, authHeaders)
	if putConfigRes.Code != http.StatusOK {
		t.Fatalf("expected put project config 200, got %d (%s)", putConfigRes.Code, putConfigRes.Body.String())
	}

	conversationRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/"+projectID+"/conversations", map[string]any{
		"workspace_id": workspaceID,
		"name":         "ClaimWithModelConfig",
	}, authHeaders)
	if conversationRes.Code != http.StatusCreated {
		t.Fatalf("expected create conversation 201, got %d (%s)", conversationRes.Code, conversationRes.Body.String())
	}
	conversationPayload := map[string]any{}
	mustDecodeJSON(t, conversationRes.Body.Bytes(), &conversationPayload)
	conversationID := conversationPayload["id"].(string)

	messageRes := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/messages", map[string]any{
		"content":  "ping minimax",
		"model_id": modelConfigID,
	}, authHeaders)
	if messageRes.Code != http.StatusCreated {
		t.Fatalf("expected create execution 201, got %d (%s)", messageRes.Code, messageRes.Body.String())
	}

	claimRes := performJSONRequest(t, router, http.MethodPost, "/internal/executions/claim", map[string]any{
		"worker_id": "worker-model-1",
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
	executionPayload := executionEnvelope["execution"].(map[string]any)
	modelSnapshot := executionPayload["model_snapshot"].(map[string]any)
	if got := modelSnapshot["config_id"]; got != modelConfigID {
		t.Fatalf("expected model_snapshot.config_id=%s, got %#v", modelConfigID, got)
	}
	if got := modelSnapshot["vendor"]; got != "MiniMax" {
		t.Fatalf("expected model_snapshot.vendor MiniMax, got %#v", got)
	}
	if got := modelSnapshot["model_id"]; got != "MiniMax-M2.5" {
		t.Fatalf("expected model_snapshot.model_id MiniMax-M2.5, got %#v", got)
	}
	params, ok := modelSnapshot["params"].(map[string]any)
	if !ok {
		t.Fatalf("expected model_snapshot.params to exist, got %#v", modelSnapshot["params"])
	}
	if got := params["api_key"]; got != "minimax-key" {
		t.Fatalf("expected model_snapshot.params.api_key minimax-key, got %#v", got)
	}

	publicExecutionsRes := performJSONRequest(t, router, http.MethodGet, "/v1/executions?conversation_id="+conversationID, nil, authHeaders)
	if publicExecutionsRes.Code != http.StatusOK {
		t.Fatalf("expected execution list 200, got %d (%s)", publicExecutionsRes.Code, publicExecutionsRes.Body.String())
	}
	publicListPayload := map[string]any{}
	mustDecodeJSON(t, publicExecutionsRes.Body.Bytes(), &publicListPayload)
	items, ok := publicListPayload["items"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("expected execution list entries, got %#v", publicListPayload)
	}
	publicExecution := items[0].(map[string]any)
	publicSnapshot := publicExecution["model_snapshot"].(map[string]any)
	if publicParams, ok := publicSnapshot["params"].(map[string]any); ok {
		if _, exists := publicParams["api_key"]; exists {
			t.Fatalf("expected public execution snapshot without api_key, got %#v", publicParams)
		}
	}
}
