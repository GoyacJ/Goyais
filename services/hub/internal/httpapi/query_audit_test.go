package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSessionReadHandlersRecordBusinessAuditEntries(t *testing.T) {
	state := NewAppState(nil)
	now := time.Now().UTC().Format(time.RFC3339)
	projectID := "proj_query_audit"
	sessionID := "conv_query_audit"
	runID := "exec_query_audit"

	state.projects[projectID] = Project{
		ID:          projectID,
		WorkspaceID: localWorkspaceID,
		Name:        "Query Audit Project",
		RepoPath:    t.TempDir(),
		DefaultMode: PermissionModeDefault,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	state.conversations[sessionID] = Conversation{
		ID:            sessionID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     projectID,
		Name:          "Query Audit Session",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_query_audit",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.executionEvents[sessionID] = []ExecutionEvent{{
		EventID:        "evt_query_audit",
		ExecutionID:    runID,
		ConversationID: sessionID,
		TraceID:        "tr_query_audit",
		QueueIndex:     0,
		Type:           RunEventTypeExecutionStarted,
		Timestamp:      now,
	}}
	state.adminAudit = nil

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sessions", ConversationsHandler(state))
	mux.HandleFunc("/v1/sessions/{session_id}", ConversationByIDHandler(state))
	mux.HandleFunc("/v1/sessions/{session_id}/events", ConversationEventsHandler(state))

	listRes := performJSONRequest(t, mux, http.MethodGet, "/v1/sessions?workspace_id="+localWorkspaceID, nil, nil)
	if listRes.Code != http.StatusOK {
		t.Fatalf("expected list 200, got %d (%s)", listRes.Code, listRes.Body.String())
	}
	detailRes := performJSONRequest(t, mux, http.MethodGet, "/v1/sessions/"+sessionID, nil, nil)
	if detailRes.Code != http.StatusOK {
		t.Fatalf("expected detail 200, got %d (%s)", detailRes.Code, detailRes.Body.String())
	}
	eventsReq := httptest.NewRequest(http.MethodGet, "/v1/sessions/"+sessionID+"/events", nil)
	eventsReq.SetPathValue("session_id", sessionID)
	cancelledCtx, cancel := context.WithCancel(eventsReq.Context())
	cancel()
	eventsReq = eventsReq.WithContext(cancelledCtx)
	eventsRes := httptest.NewRecorder()
	ConversationEventsHandler(state).ServeHTTP(eventsRes, eventsReq)
	if eventsRes.Code != http.StatusOK {
		t.Fatalf("expected events 200, got %d (%s)", eventsRes.Code, eventsRes.Body.String())
	}

	state.mu.RLock()
	defer state.mu.RUnlock()
	readAuditCount := 0
	for _, entry := range state.adminAudit {
		if entry.Action == "session.read" {
			readAuditCount++
		}
	}
	if readAuditCount != 3 {
		t.Fatalf("expected 3 session.read business audit entries, got %d (%#v)", readAuditCount, state.adminAudit)
	}
}

func TestSessionDetailAuthorizationAuditUsesSessionIDTarget(t *testing.T) {
	state := NewAppState(nil)
	now := time.Now().UTC().Format(time.RFC3339)
	projectID := "proj_query_authz_target"
	sessionID := "conv_query_authz_target"

	state.projects[projectID] = Project{
		ID:          projectID,
		WorkspaceID: localWorkspaceID,
		Name:        "Query Target Project",
		RepoPath:    t.TempDir(),
		DefaultMode: PermissionModeDefault,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	state.conversations[sessionID] = Conversation{
		ID:            sessionID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     projectID,
		Name:          "Query Target Session",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_query_target",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.adminAudit = nil

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/sessions/{session_id}", ConversationByIDHandler(state))

	detailRes := performJSONRequest(t, mux, http.MethodGet, "/v1/sessions/"+sessionID, nil, nil)
	if detailRes.Code != http.StatusOK {
		t.Fatalf("expected detail 200, got %d (%s)", detailRes.Code, detailRes.Body.String())
	}

	state.mu.RLock()
	defer state.mu.RUnlock()
	for _, entry := range state.adminAudit {
		if entry.Action != "authz.session.read" {
			continue
		}
		if entry.Resource != sessionID {
			t.Fatalf("expected authz.session.read resource %q, got %#v", sessionID, entry)
		}
		return
	}
	t.Fatalf("expected authz.session.read audit entry, got %#v", state.adminAudit)
}
