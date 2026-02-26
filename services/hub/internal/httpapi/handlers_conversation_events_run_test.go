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

func TestConversationEventsSSE_ResyncBackfillWhenLastEventMissing(t *testing.T) {
	router := NewRouter()
	workspaceID := createRemoteWorkspace(t, router, "Remote SSE Resync", "http://127.0.0.1:9131", false)
	token := loginRemoteWorkspace(t, router, workspaceID, "sse_resync_user", "pw", RoleDeveloper, true)
	authHeaders := map[string]string{"Authorization": "Bearer " + token}

	projectRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/import", map[string]any{
		"workspace_id":   workspaceID,
		"directory_path": "/tmp/sse-resync-alpha",
	}, authHeaders)
	if projectRes.Code != http.StatusCreated {
		t.Fatalf("expected import project 201, got %d (%s)", projectRes.Code, projectRes.Body.String())
	}
	projectPayload := map[string]any{}
	mustDecodeJSON(t, projectRes.Body.Bytes(), &projectPayload)
	projectID := projectPayload["id"].(string)

	conversationRes := performJSONRequest(t, router, http.MethodPost, "/v1/projects/"+projectID+"/conversations", map[string]any{
		"workspace_id": workspaceID,
		"name":         "SSEResyncConv",
	}, authHeaders)
	if conversationRes.Code != http.StatusCreated {
		t.Fatalf("expected create conversation 201, got %d (%s)", conversationRes.Code, conversationRes.Body.String())
	}
	conversationPayload := map[string]any{}
	mustDecodeJSON(t, conversationRes.Body.Bytes(), &conversationPayload)
	conversationID := conversationPayload["id"].(string)

	messageRes := performJSONRequest(t, router, http.MethodPost, "/v1/conversations/"+conversationID+"/messages", map[string]any{
		"content": "trigger sse resync",
	}, authHeaders)
	if messageRes.Code != http.StatusCreated {
		t.Fatalf("expected message 201, got %d (%s)", messageRes.Code, messageRes.Body.String())
	}
	messagePayload := map[string]any{}
	mustDecodeJSON(t, messageRes.Body.Bytes(), &messagePayload)
	executionID := messagePayload["execution"].(map[string]any)["id"].(string)

	stopRes := performJSONRequest(t, router, http.MethodPost, "/v1/runs/"+executionID+"/control", map[string]any{
		"action": "stop",
	}, authHeaders)
	if stopRes.Code != http.StatusOK && stopRes.Code != http.StatusConflict {
		t.Fatalf("expected stop control 200/409, got %d (%s)", stopRes.Code, stopRes.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/conversations/"+conversationID+"/events?last_event_id=evt_missing_cursor", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)
	res := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		router.ServeHTTP(res, req)
		close(done)
	}()

	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		body := res.Body.String()
		if strings.Contains(body, "\"resync_required\":true") && (strings.Contains(body, "\"type\":\"run_cancelled\"") || strings.Contains(body, "\"type\":\"run_failed\"")) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()

	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for SSE resync handler to exit")
	}

	events := decodeSSEDataLines(t, res.Body.String())
	if len(events) == 0 {
		t.Fatalf("expected SSE events, got body: %s", res.Body.String())
	}

	first := events[0]
	if first["type"] != "run_output_delta" {
		t.Fatalf("expected first event run_output_delta resync marker, got %#v", first["type"])
	}
	firstPayload, ok := first["payload"].(map[string]any)
	if !ok {
		t.Fatalf("expected first event payload map, got %#v", first["payload"])
	}
	if firstPayload["resync_required"] != true {
		t.Fatalf("expected resync_required=true, got %#v", firstPayload["resync_required"])
	}
	if firstPayload["reason"] != "last_event_id_not_found" {
		t.Fatalf("expected reason last_event_id_not_found, got %#v", firstPayload["reason"])
	}
	if firstPayload["last_event_id"] != "evt_missing_cursor" {
		t.Fatalf("expected echoed last_event_id, got %#v", firstPayload["last_event_id"])
	}

	foundTerminal := false
	for _, event := range events {
		if (event["type"] == "run_cancelled" || event["type"] == "run_failed") && event["run_id"] == executionID {
			foundTerminal = true
			break
		}
	}
	if !foundTerminal {
		t.Fatalf("expected terminal run event for %s, got %#v", executionID, events)
	}
}

func decodeSSEDataLines(t *testing.T, body string) []map[string]any {
	t.Helper()

	lines := strings.Split(body, "\n")
	events := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		raw := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
		if raw == "" {
			continue
		}
		payload := map[string]any{}
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			t.Fatalf("failed to decode SSE payload line: %v (%s)", err, raw)
		}
		events = append(events, payload)
	}
	return events
}
