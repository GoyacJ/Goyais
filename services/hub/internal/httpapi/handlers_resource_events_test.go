package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestWorkspaceResourceEventsHandlerReplaysBacklog(t *testing.T) {
	router := NewRouter()
	workspaceID := createRemoteWorkspace(t, router, "Remote Resource Events", "http://127.0.0.1:9141", false)
	token := loginRemoteWorkspace(t, router, workspaceID, "resource_events_user", "pw", RoleAdmin, true)
	authHeaders := map[string]string{"Authorization": "Bearer " + token}

	createRes := performJSONRequest(t, router, http.MethodPost, "/v1/workspaces/"+workspaceID+"/resource-configs", map[string]any{
		"type": "rule",
		"name": "Review Rule",
		"rule": map[string]any{"content": "always review"},
	}, authHeaders)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("expected resource config create 201, got %d (%s)", createRes.Code, createRes.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/workspaces/"+workspaceID+"/resource-events", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)
	recorder := httptest.NewRecorder()
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	router.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected resource events 200, got %d (%s)", recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); !strings.Contains(got, "text/event-stream") {
		t.Fatalf("expected text/event-stream content type, got %q", got)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"type":"resource_config_created"`) {
		t.Fatalf("expected create event in SSE backlog, got %s", body)
	}
}
