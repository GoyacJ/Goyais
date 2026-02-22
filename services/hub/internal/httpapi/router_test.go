package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

func TestHealth(t *testing.T) {
	router := NewRouter()
	res := performJSONRequest(t, router, http.MethodGet, "/health", nil, nil)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}

	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	if payload["ok"] != true {
		t.Fatalf("expected ok=true, got %#v", payload["ok"])
	}
	if payload["version"] != "0.4.0" {
		t.Fatalf("expected version 0.4.0, got %#v", payload["version"])
	}
}

func TestPlaceholderListEndpoints(t *testing.T) {
	router := NewRouter()
	paths := []string{"/v1/projects", "/v1/conversations", "/v1/executions"}
	for _, path := range paths {
		res := performJSONRequest(t, router, http.MethodGet, path, nil, nil)
		if res.Code != http.StatusOK {
			t.Fatalf("%s expected 200, got %d", path, res.Code)
		}
		payload := map[string]any{}
		mustDecodeJSON(t, res.Body.Bytes(), &payload)
		items, ok := payload["items"].([]any)
		if !ok || len(items) != 0 {
			t.Fatalf("%s expected empty items, got %#v", path, payload["items"])
		}
		if payload["next_cursor"] != nil {
			t.Fatalf("%s expected next_cursor=nil, got %#v", path, payload["next_cursor"])
		}
	}
}

func TestWorkspacesContainsLocal(t *testing.T) {
	router := NewRouter()
	res := performJSONRequest(t, router, http.MethodGet, "/v1/workspaces", nil, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}

	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	items := payload["items"].([]any)
	if len(items) == 0 {
		t.Fatalf("expected at least one workspace")
	}

	local := items[0].(map[string]any)
	if local["id"] != localWorkspaceID {
		t.Fatalf("expected first workspace id=%s, got %#v", localWorkspaceID, local["id"])
	}
	if local["mode"] != string(WorkspaceModeLocal) {
		t.Fatalf("expected local mode, got %#v", local["mode"])
	}
}

func TestCreateRemoteWorkspaceAndList(t *testing.T) {
	router := NewRouter()
	workspaceID := createRemoteWorkspace(t, router, "Remote A", "http://127.0.0.1:9876", false)

	res := performJSONRequest(t, router, http.MethodGet, "/v1/workspaces", nil, nil)
	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)

	items := payload["items"].([]any)
	found := false
	for _, item := range items {
		workspace := item.(map[string]any)
		if workspace["id"] == workspaceID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected remote workspace %s in list", workspaceID)
	}
}

func TestLoginValidationAndAuthErrors(t *testing.T) {
	router := NewRouter()
	remoteID := createRemoteWorkspace(t, router, "Remote B", "http://127.0.0.1:9877", false)

	localLogin := performJSONRequest(t, router, http.MethodPost, "/v1/auth/login", map[string]any{
		"workspace_id": localWorkspaceID,
		"token":        "anything",
	}, nil)
	if localLogin.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for local login, got %d", localLogin.Code)
	}

	missingPassword := performJSONRequest(t, router, http.MethodPost, "/v1/auth/login", map[string]any{
		"workspace_id": remoteID,
		"username":     "alice",
	}, nil)
	if missingPassword.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid shape, got %d", missingPassword.Code)
	}

	invalidCred := performJSONRequest(t, router, http.MethodPost, "/v1/auth/login", map[string]any{
		"workspace_id": "ws_target_invalid",
		"username":     "invalid",
		"password":     "invalid",
	}, map[string]string{internalForwardedLoginHeader: "1"})
	if invalidCred.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid credentials, got %d", invalidCred.Code)
	}
}

func TestLoginDisabledReturns403(t *testing.T) {
	router := NewRouter()
	disabledID := createRemoteWorkspace(t, router, "Remote Disabled", "http://127.0.0.1:9878", true)

	res := performJSONRequest(t, router, http.MethodPost, "/v1/auth/login", map[string]any{
		"workspace_id": disabledID,
		"username":     "alice",
		"password":     "pass",
	}, nil)
	if res.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", res.Code)
	}

	errPayload := StandardError{}
	mustDecodeJSON(t, res.Body.Bytes(), &errPayload)
	if errPayload.Code != "LOGIN_DISABLED" {
		t.Fatalf("expected LOGIN_DISABLED, got %s", errPayload.Code)
	}
}

func TestControlProxyLoginCreatesSessionOnlyOnTarget(t *testing.T) {
	targetRouter := NewRouter()
	targetServer := httptest.NewServer(targetRouter)
	defer targetServer.Close()

	controlRouter := NewRouter()
	remoteID := createRemoteWorkspace(t, controlRouter, "Remote Target", targetServer.URL, false)

	loginRes := performJSONRequest(t, controlRouter, http.MethodPost, "/v1/auth/login", map[string]any{
		"workspace_id": remoteID,
		"username":     "bob",
		"password":     "pw",
	}, nil)
	if loginRes.Code != http.StatusOK {
		t.Fatalf("expected 200 login, got %d", loginRes.Code)
	}

	loginPayload := LoginResponse{}
	mustDecodeJSON(t, loginRes.Body.Bytes(), &loginPayload)
	if loginPayload.AccessToken == "" {
		t.Fatalf("expected access token")
	}

	controlMe := performJSONRequest(t, controlRouter, http.MethodGet, "/v1/me", nil, map[string]string{
		"Authorization": "Bearer " + loginPayload.AccessToken,
	})
	if controlMe.Code != http.StatusUnauthorized {
		t.Fatalf("expected control /v1/me to reject remote token, got %d", controlMe.Code)
	}

	targetMe := performJSONRequest(t, targetRouter, http.MethodGet, "/v1/me", nil, map[string]string{
		"Authorization": "Bearer " + loginPayload.AccessToken,
	})
	if targetMe.Code != http.StatusOK {
		t.Fatalf("expected target /v1/me 200, got %d", targetMe.Code)
	}
	mePayload := Me{}
	mustDecodeJSON(t, targetMe.Body.Bytes(), &mePayload)
	if mePayload.WorkspaceID != remoteID {
		t.Fatalf("expected workspace_id=%s, got %s", remoteID, mePayload.WorkspaceID)
	}
	if mePayload.Capabilities.AdminConsole {
		t.Fatalf("expected default developer role with admin_console=false")
	}

	targetAdminDenied := performJSONRequest(t, targetRouter, http.MethodGet, "/v1/admin/ping", nil, map[string]string{
		"Authorization": "Bearer " + loginPayload.AccessToken,
	})
	if targetAdminDenied.Code != http.StatusForbidden {
		t.Fatalf("expected developer token to be forbidden, got %d", targetAdminDenied.Code)
	}

	adminLoginRes := performJSONRequest(t, controlRouter, http.MethodPost, "/v1/auth/login", map[string]any{
		"workspace_id": remoteID,
		"username":     "admin_user",
		"password":     "pw",
	}, map[string]string{"X-Role": string(RoleAdmin)})
	if adminLoginRes.Code != http.StatusOK {
		t.Fatalf("expected 200 admin login, got %d", adminLoginRes.Code)
	}
	adminLoginPayload := LoginResponse{}
	mustDecodeJSON(t, adminLoginRes.Body.Bytes(), &adminLoginPayload)

	targetAdminPing := performJSONRequest(t, targetRouter, http.MethodGet, "/v1/admin/ping", nil, map[string]string{
		"Authorization": "Bearer " + adminLoginPayload.AccessToken,
	})
	if targetAdminPing.Code != http.StatusOK {
		t.Fatalf("expected admin token to pass admin ping, got %d", targetAdminPing.Code)
	}
}

func TestMeWithoutTokenReturnsLocalAdmin(t *testing.T) {
	router := NewRouter()
	res := performJSONRequest(t, router, http.MethodGet, "/v1/me", nil, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}

	payload := Me{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	if payload.WorkspaceID != localWorkspaceID {
		t.Fatalf("expected local workspace, got %s", payload.WorkspaceID)
	}
	if !payload.Capabilities.AdminConsole {
		t.Fatalf("expected local admin capabilities")
	}
}

func TestTraceConsistencyAcrossErrorClasses(t *testing.T) {
	router := NewRouter()
	disabledID := createRemoteWorkspace(t, router, "Trace Remote", "http://127.0.0.1:9879", true)

	testCases := []struct {
		name    string
		method  string
		path    string
		body    map[string]any
		headers map[string]string
		status  int
	}{
		{
			name:   "400",
			method: http.MethodPost,
			path:   "/v1/auth/login",
			body: map[string]any{
				"workspace_id": localWorkspaceID,
				"token":        "anything",
			},
			status: http.StatusBadRequest,
		},
		{
			name:   "401",
			method: http.MethodPost,
			path:   "/v1/auth/login",
			body: map[string]any{
				"workspace_id": "ws_target_invalid",
				"token":        "invalid",
			},
			headers: map[string]string{internalForwardedLoginHeader: "1"},
			status:  http.StatusUnauthorized,
		},
		{
			name:   "403",
			method: http.MethodPost,
			path:   "/v1/auth/login",
			body: map[string]any{
				"workspace_id": disabledID,
				"username":     "alice",
				"password":     "pw",
			},
			status: http.StatusForbidden,
		},
	}

	for _, tc := range testCases {
		headers := map[string]string{TraceHeader: "tr_test_" + tc.name}
		for key, value := range tc.headers {
			headers[key] = value
		}

		res := performJSONRequest(t, router, tc.method, tc.path, tc.body, headers)
		if res.Code != tc.status {
			t.Fatalf("%s expected %d, got %d", tc.name, tc.status, res.Code)
		}

		payload := StandardError{}
		mustDecodeJSON(t, res.Body.Bytes(), &payload)

		headerTrace := res.Header().Get(TraceHeader)
		if headerTrace != headers[TraceHeader] {
			t.Fatalf("%s expected header trace=%s, got %s", tc.name, headers[TraceHeader], headerTrace)
		}
		if payload.TraceID != headers[TraceHeader] {
			t.Fatalf("%s expected body trace=%s, got %s", tc.name, headers[TraceHeader], payload.TraceID)
		}
	}
}

func TestTraceGeneratedWhenMissing(t *testing.T) {
	router := NewRouter()
	res := performJSONRequest(t, router, http.MethodPut, "/v1/projects", nil, nil)
	if res.Code != http.StatusNotImplemented {
		t.Fatalf("expected 501, got %d", res.Code)
	}

	payload := StandardError{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	traceFromHeader := res.Header().Get(TraceHeader)
	if traceFromHeader == "" {
		t.Fatalf("expected generated trace header")
	}
	matched, _ := regexp.MatchString(`^tr_[a-z0-9]+$`, traceFromHeader)
	if !matched {
		t.Fatalf("generated trace format mismatch: %s", traceFromHeader)
	}
	if payload.TraceID != traceFromHeader {
		t.Fatalf("expected trace match, body=%s header=%s", payload.TraceID, traceFromHeader)
	}
}

func createRemoteWorkspace(t *testing.T, router http.Handler, name string, hubURL string, disabled bool) string {
	t.Helper()
	res := performJSONRequest(t, router, http.MethodPost, "/v1/workspaces", map[string]any{
		"name":           name,
		"hub_url":        hubURL,
		"login_disabled": disabled,
	}, nil)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected 201 create workspace, got %d (%s)", res.Code, res.Body.String())
	}

	workspace := Workspace{}
	mustDecodeJSON(t, res.Body.Bytes(), &workspace)
	if workspace.ID == "" {
		t.Fatalf("workspace id should not be empty")
	}
	return workspace.ID
}

func performJSONRequest(t *testing.T, router http.Handler, method string, path string, body map[string]any, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	var bodyReader *bytes.Reader
	if body == nil {
		bodyReader = bytes.NewReader(nil)
	} else {
		encoded, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("failed to marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(encoded)
	}

	req := httptest.NewRequest(method, path, bodyReader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	return res
}

func mustDecodeJSON(t *testing.T, raw []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(raw, target); err != nil {
		t.Fatalf("failed to decode JSON: %v; payload=%s", err, string(raw))
	}
}
