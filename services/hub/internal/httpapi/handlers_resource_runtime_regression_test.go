package httpapi

import (
	"net/http"
	"strings"
	"testing"
)

func TestResourceConfigTestEndpoint_ModelProbeMissingAPIKeyRegression(t *testing.T) {
	router := NewRouter()
	workspaceID := createRemoteWorkspace(t, router, "Remote Resource Runtime Regression", "http://127.0.0.1:9132", false)
	token := loginRemoteWorkspace(t, router, workspaceID, "resource_runtime_user", "pw", RoleDeveloper, true)
	authHeaders := map[string]string{"Authorization": "Bearer " + token}

	createRes := performJSONRequest(t, router, http.MethodPost, "/v1/workspaces/"+workspaceID+"/resource-configs", map[string]any{
		"type":    "model",
		"enabled": true,
		"model": map[string]any{
			"vendor":   "OpenAI",
			"model_id": "gpt-5.3",
		},
	}, authHeaders)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected create model config 201, got %d (%s)", createRes.Code, createRes.Body.String())
	}
	createPayload := map[string]any{}
	mustDecodeJSON(t, createRes.Body.Bytes(), &createPayload)
	configID := createPayload["id"].(string)

	testRes := performJSONRequest(t, router, http.MethodPost, "/v1/workspaces/"+workspaceID+"/resource-configs/"+configID+"/test", map[string]any{}, authHeaders)
	if testRes.Code != http.StatusOK {
		t.Fatalf("expected model test 200, got %d (%s)", testRes.Code, testRes.Body.String())
	}
	testPayload := map[string]any{}
	mustDecodeJSON(t, testRes.Body.Bytes(), &testPayload)

	if testPayload["status"] != "failed" {
		t.Fatalf("expected failed probe status, got %#v", testPayload["status"])
	}
	if testPayload["error_code"] != "missing_api_key" {
		t.Fatalf("expected missing_api_key, got %#v", testPayload["error_code"])
	}
	if !strings.Contains(strings.ToLower(asString(testPayload["message"])), "api_key") {
		t.Fatalf("expected api_key message, got %#v", testPayload["message"])
	}
}

func TestResourceConfigCreateRejectsOutOfRangeRuntimeTimeout(t *testing.T) {
	router := NewRouter()
	workspaceID := createRemoteWorkspace(t, router, "Remote Resource Timeout Range", "http://127.0.0.1:9133", false)
	token := loginRemoteWorkspace(t, router, workspaceID, "resource_runtime_user", "pw", RoleDeveloper, true)
	authHeaders := map[string]string{"Authorization": "Bearer " + token}

	createRes := performJSONRequest(t, router, http.MethodPost, "/v1/workspaces/"+workspaceID+"/resource-configs", map[string]any{
		"type": "model",
		"model": map[string]any{
			"vendor":   "OpenAI",
			"model_id": "gpt-5.3",
			"runtime": map[string]any{
				"request_timeout_ms": 999,
			},
		},
	}, authHeaders)
	if createRes.Code != http.StatusBadRequest {
		t.Fatalf("expected create model config 400 for invalid runtime timeout, got %d (%s)", createRes.Code, createRes.Body.String())
	}
}

func TestResourceConfigPatchRejectsOutOfRangeRuntimeTimeout(t *testing.T) {
	router := NewRouter()
	workspaceID := createRemoteWorkspace(t, router, "Remote Resource Timeout Patch", "http://127.0.0.1:9134", false)
	token := loginRemoteWorkspace(t, router, workspaceID, "resource_runtime_user", "pw", RoleDeveloper, true)
	authHeaders := map[string]string{"Authorization": "Bearer " + token}

	createRes := performJSONRequest(t, router, http.MethodPost, "/v1/workspaces/"+workspaceID+"/resource-configs", map[string]any{
		"type": "model",
		"model": map[string]any{
			"vendor":   "OpenAI",
			"model_id": "gpt-5.3",
			"api_key":  "sk-test",
		},
	}, authHeaders)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected create model config 201, got %d (%s)", createRes.Code, createRes.Body.String())
	}
	payload := map[string]any{}
	mustDecodeJSON(t, createRes.Body.Bytes(), &payload)
	configID := strings.TrimSpace(asString(payload["id"]))
	if configID == "" {
		t.Fatalf("expected model config id")
	}

	patchRes := performJSONRequest(t, router, http.MethodPatch, "/v1/workspaces/"+workspaceID+"/resource-configs/"+configID, map[string]any{
		"model": map[string]any{
			"vendor":   "OpenAI",
			"model_id": "gpt-5.3",
			"runtime": map[string]any{
				"request_timeout_ms": 120001,
			},
		},
	}, authHeaders)
	if patchRes.Code != http.StatusBadRequest {
		t.Fatalf("expected patch model config 400 for invalid runtime timeout, got %d (%s)", patchRes.Code, patchRes.Body.String())
	}
}
