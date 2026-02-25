package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestConversationEventsSSE_EmitsRunSemantics(t *testing.T) {
	router := NewRouter()
	workspaceID := createRemoteWorkspace(t, router, "Remote SSE", "http://127.0.0.1:9130", false)
	token := loginRemoteWorkspace(t, router, workspaceID, "sse_user", "pw", RoleDeveloper, true)
	authHeaders := map[string]string{"Authorization": "Bearer " + token}

	projectRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/import", map[string]any{
		"workspace_id":   workspaceID,
		"directory_path": "/tmp/sse-run-alpha",
	}, authHeaders)
	if projectRes.Code != http.StatusCreated {
		t.Fatalf("expected import project 201, got %d (%s)", projectRes.Code, projectRes.Body.String())
	}
	projectPayload := map[string]any{}
	mustDecodeJSON(t, projectRes.Body.Bytes(), &projectPayload)
	projectID := projectPayload["id"].(string)

	conversationRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/"+projectID+"/conversations", map[string]any{
		"workspace_id": workspaceID,
		"name":         "SSEConv",
	}, authHeaders)
	if conversationRes.Code != http.StatusCreated {
		t.Fatalf("expected create conversation 201, got %d (%s)", conversationRes.Code, conversationRes.Body.String())
	}
	conversationPayload := map[string]any{}
	mustDecodeJSON(t, conversationRes.Body.Bytes(), &conversationPayload)
	conversationID := conversationPayload["id"].(string)

	messageRes := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/messages", map[string]any{
		"content": "trigger sse",
	}, authHeaders)
	if messageRes.Code != http.StatusCreated {
		t.Fatalf("expected message 201, got %d (%s)", messageRes.Code, messageRes.Body.String())
	}
	messagePayload := map[string]any{}
	mustDecodeJSON(t, messageRes.Body.Bytes(), &messagePayload)
	executionID := messagePayload["execution"].(map[string]any)["id"].(string)

	req := httptest.NewRequest(http.MethodGet, "/v1/conversations/"+conversationID+"/events", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	ctx, cancel := context.WithCancel(req.Context())
	defer cancel()
	req = req.WithContext(ctx)

	res := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		router.ServeHTTP(res, req)
		close(done)
	}()

	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(res.Body.String(), "data: ") {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for SSE handler to exit")
	}

	lines := strings.Split(res.Body.String(), "\n")
	var eventLine string
	for _, line := range lines {
		if strings.HasPrefix(line, "data: ") {
			eventLine = strings.TrimSpace(strings.TrimPrefix(line, "data: "))
			break
		}
	}
	if eventLine == "" {
		t.Fatalf("expected SSE data line, got body: %s", res.Body.String())
	}

	payload := map[string]any{}
	if err := json.Unmarshal([]byte(eventLine), &payload); err != nil {
		t.Fatalf("failed to decode SSE payload: %v (%s)", err, eventLine)
	}

	if payload["type"] != "run_queued" {
		t.Fatalf("expected run_queued event type, got %#v", payload["type"])
	}
	if payload["session_id"] != conversationID {
		t.Fatalf("expected session_id=%s, got %#v", conversationID, payload["session_id"])
	}
	if payload["run_id"] != executionID {
		t.Fatalf("expected run_id=%s, got %#v", executionID, payload["run_id"])
	}
}
