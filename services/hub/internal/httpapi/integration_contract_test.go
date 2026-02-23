package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRemoteConnectionsEndpointReturnsUnifiedShape(t *testing.T) {
	targetRouter := NewRouter()
	targetServer := httptest.NewServer(targetRouter)
	defer targetServer.Close()

	router := NewRouter()

	res := performJSONRequest(t, router, http.MethodPost, "/v1/workspaces/remote-connections", map[string]any{
		"name":     "Remote Contract",
		"hub_url":  targetServer.URL,
		"username": "alice",
		"password": "pw",
	}, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", res.Code, res.Body.String())
	}

	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)

	workspace, ok := payload["workspace"].(map[string]any)
	if !ok {
		t.Fatalf("expected workspace object, got %#v", payload["workspace"])
	}
	if workspace["mode"] != string(WorkspaceModeRemote) {
		t.Fatalf("expected remote mode, got %#v", workspace["mode"])
	}

	connection, ok := payload["connection"].(map[string]any)
	if !ok {
		t.Fatalf("expected connection object, got %#v", payload["connection"])
	}
	if strings.TrimSpace(connection["workspace_id"].(string)) == "" {
		t.Fatalf("expected connection.workspace_id to be present")
	}
	if connection["hub_url"] != targetServer.URL {
		t.Fatalf("expected connection.hub_url, got %#v", connection["hub_url"])
	}
	if connection["username"] != "alice" {
		t.Fatalf("expected connection.username, got %#v", connection["username"])
	}

	if strings.TrimSpace(payload["access_token"].(string)) == "" {
		t.Fatalf("expected access_token to be present")
	}
}

func TestProjectConversationFlowWithCursorPagination(t *testing.T) {
	router := NewRouter()
	workspaceID := createRemoteWorkspace(t, router, "Remote Flow", "http://127.0.0.1:9982", false)

	projectRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/import", map[string]any{
		"workspace_id":   workspaceID,
		"directory_path": "/tmp/repo-alpha",
	}, nil)
	if projectRes.Code != http.StatusCreated {
		t.Fatalf("expected import project 201, got %d (%s)", projectRes.Code, projectRes.Body.String())
	}
	project := map[string]any{}
	mustDecodeJSON(t, projectRes.Body.Bytes(), &project)
	projectID := project["id"].(string)

	projectRes2 := performJSONRequest(t, router, http.MethodPost, "/v1/projects", map[string]any{
		"workspace_id": workspaceID,
		"name":         "beta",
		"repo_path":    "/tmp/repo-beta",
		"is_git":       true,
	}, nil)
	if projectRes2.Code != http.StatusCreated {
		t.Fatalf("expected create project 201, got %d (%s)", projectRes2.Code, projectRes2.Body.String())
	}

	page1 := performJSONRequest(t, router, http.MethodGet, "/v1/projects?workspace_id="+workspaceID+"&limit=1", nil, nil)
	if page1.Code != http.StatusOK {
		t.Fatalf("expected projects page1 200, got %d", page1.Code)
	}
	page1Payload := map[string]any{}
	mustDecodeJSON(t, page1.Body.Bytes(), &page1Payload)
	items := page1Payload["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected 1 project in page1, got %d", len(items))
	}
	nextCursor, ok := page1Payload["next_cursor"].(string)
	if !ok || strings.TrimSpace(nextCursor) == "" {
		t.Fatalf("expected next_cursor in page1, got %#v", page1Payload["next_cursor"])
	}

	page2 := performJSONRequest(t, router, http.MethodGet, "/v1/projects?workspace_id="+workspaceID+"&limit=1&cursor="+nextCursor, nil, nil)
	if page2.Code != http.StatusOK {
		t.Fatalf("expected projects page2 200, got %d", page2.Code)
	}
	page2Payload := map[string]any{}
	mustDecodeJSON(t, page2.Body.Bytes(), &page2Payload)
	if len(page2Payload["items"].([]any)) == 0 {
		t.Fatalf("expected projects page2 to have data")
	}

	conversationRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/"+projectID+"/conversations", map[string]any{
		"workspace_id": workspaceID,
		"name":         "Conv Main",
	}, nil)
	if conversationRes.Code != http.StatusCreated {
		t.Fatalf("expected create conversation 201, got %d (%s)", conversationRes.Code, conversationRes.Body.String())
	}
	conversation := map[string]any{}
	mustDecodeJSON(t, conversationRes.Body.Bytes(), &conversation)
	conversationID := conversation["id"].(string)

	msg1 := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/messages", map[string]any{
		"content":  "hello",
		"mode":     "agent",
		"model_id": "gpt-4.1",
	}, nil)
	if msg1.Code != http.StatusCreated {
		t.Fatalf("expected first message 201, got %d (%s)", msg1.Code, msg1.Body.String())
	}
	msg1Payload := map[string]any{}
	mustDecodeJSON(t, msg1.Body.Bytes(), &msg1Payload)
	exec1 := msg1Payload["execution"].(map[string]any)
	messageID := exec1["message_id"].(string)

	msg2 := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/messages", map[string]any{
		"content":  "second",
		"mode":     "agent",
		"model_id": "gpt-4.1",
	}, nil)
	if msg2.Code != http.StatusCreated {
		t.Fatalf("expected second message 201, got %d (%s)", msg2.Code, msg2.Body.String())
	}
	msg2Payload := map[string]any{}
	mustDecodeJSON(t, msg2.Body.Bytes(), &msg2Payload)
	exec2 := msg2Payload["execution"].(map[string]any)
	if exec2["state"] != "queued" {
		t.Fatalf("expected second execution queued, got %#v", exec2["state"])
	}

	stopRes := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/stop", map[string]any{}, nil)
	if stopRes.Code != http.StatusOK {
		t.Fatalf("expected stop 200, got %d (%s)", stopRes.Code, stopRes.Body.String())
	}

	rollbackRes := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/rollback", map[string]any{
		"message_id": messageID,
	}, nil)
	if rollbackRes.Code != http.StatusOK {
		t.Fatalf("expected rollback 200, got %d (%s)", rollbackRes.Code, rollbackRes.Body.String())
	}

	exportRes := performJSONRequest(t, router, http.MethodGet, "/v1/conversations/"+conversationID+"/export?format=markdown", nil, nil)
	if exportRes.Code != http.StatusOK {
		t.Fatalf("expected export 200, got %d (%s)", exportRes.Code, exportRes.Body.String())
	}
	if !strings.Contains(exportRes.Body.String(), "# Conversation") {
		t.Fatalf("expected markdown export body, got %s", exportRes.Body.String())
	}
}

func TestShareApproveRequiresApproverAndProducesAudit(t *testing.T) {
	targetRouter := NewRouter()
	targetServer := httptest.NewServer(targetRouter)
	defer targetServer.Close()

	router := NewRouter()
	workspaceID := createRemoteWorkspace(t, router, "Remote Share", targetServer.URL, false)

	importRes := performJSONRequest(t, router, http.MethodPost, "/v1/workspaces/"+workspaceID+"/resource-imports", map[string]any{
		"resource_type": "model",
		"source_id":     "model-src-1",
	}, nil)
	if importRes.Code != http.StatusCreated {
		t.Fatalf("expected import resource 201, got %d (%s)", importRes.Code, importRes.Body.String())
	}
	resourcePayload := map[string]any{}
	mustDecodeJSON(t, importRes.Body.Bytes(), &resourcePayload)
	resourceID := resourcePayload["id"].(string)

	shareRes := performJSONRequest(t, router, http.MethodPost, "/v1/workspaces/"+workspaceID+"/share-requests", map[string]any{
		"resource_id": resourceID,
	}, nil)
	if shareRes.Code != http.StatusCreated {
		t.Fatalf("expected share request 201, got %d (%s)", shareRes.Code, shareRes.Body.String())
	}
	sharePayload := map[string]any{}
	mustDecodeJSON(t, shareRes.Body.Bytes(), &sharePayload)
	requestID := sharePayload["id"].(string)

	developerLogin := performJSONRequest(t, router, http.MethodPost, "/v1/auth/login", map[string]any{
		"workspace_id": workspaceID,
		"username":     "dev",
		"password":     "pw",
	}, map[string]string{
		"X-Role":                     string(RoleDeveloper),
		internalForwardedLoginHeader: "1",
	})
	if developerLogin.Code != http.StatusOK {
		t.Fatalf("expected dev login 200, got %d (%s)", developerLogin.Code, developerLogin.Body.String())
	}
	developerLoginPayload := LoginResponse{}
	mustDecodeJSON(t, developerLogin.Body.Bytes(), &developerLoginPayload)

	approveDenied := performJSONRequest(t, router, http.MethodPost, "/v1/share-requests/"+requestID+"/approve", map[string]any{}, map[string]string{
		"Authorization": "Bearer " + developerLoginPayload.AccessToken,
	})
	if approveDenied.Code != http.StatusForbidden {
		t.Fatalf("expected dev approve denied 403, got %d (%s)", approveDenied.Code, approveDenied.Body.String())
	}

	approverLogin := performJSONRequest(t, router, http.MethodPost, "/v1/auth/login", map[string]any{
		"workspace_id": workspaceID,
		"username":     "approver",
		"password":     "pw",
	}, map[string]string{
		"X-Role":                     string(RoleApprover),
		internalForwardedLoginHeader: "1",
	})
	if approverLogin.Code != http.StatusOK {
		t.Fatalf("expected approver login 200, got %d (%s)", approverLogin.Code, approverLogin.Body.String())
	}
	approverLoginPayload := LoginResponse{}
	mustDecodeJSON(t, approverLogin.Body.Bytes(), &approverLoginPayload)

	approveRes := performJSONRequest(t, router, http.MethodPost, "/v1/share-requests/"+requestID+"/approve", map[string]any{}, map[string]string{
		"Authorization": "Bearer " + approverLoginPayload.AccessToken,
	})
	if approveRes.Code != http.StatusOK {
		t.Fatalf("expected approver approve 200, got %d (%s)", approveRes.Code, approveRes.Body.String())
	}

	auditRes := performJSONRequest(t, router, http.MethodGet, "/v1/admin/audit?workspace_id="+workspaceID+"&limit=5", nil, nil)
	if auditRes.Code != http.StatusOK {
		t.Fatalf("expected audit list 200, got %d (%s)", auditRes.Code, auditRes.Body.String())
	}
	auditPayload := map[string]any{}
	mustDecodeJSON(t, auditRes.Body.Bytes(), &auditPayload)
	if len(auditPayload["items"].([]any)) == 0 {
		t.Fatalf("expected audit items")
	}
}
