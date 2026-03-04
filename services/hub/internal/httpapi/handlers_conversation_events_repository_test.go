// Copyright (c) 2026 Ysmjjsy
// Author: Goyais Team
// SPDX-License-Identifier: MIT

package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestConversationEventsHandlerUsesRepositoryWhenConversationMapMissing(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	}()

	state := NewAppState(store)
	now := time.Now().UTC().Format(time.RFC3339)
	conversationID := "conv_events_repo_" + randomHex(4)
	executionID := "exec_events_repo_" + randomHex(4)

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:            conversationID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     "proj_events_repo_" + randomHex(4),
		Name:          "Events Repository",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_events_repo",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.executions[executionID] = Execution{
		ID:             executionID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_events_repo_" + randomHex(4),
		State:          RunStateQueued,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     0,
		TraceID:        "tr_events_repo_" + randomHex(4),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	appendExecutionEventLocked(state, ExecutionEvent{
		ExecutionID:    executionID,
		ConversationID: conversationID,
		Type:           RunEventTypeExecutionStarted,
		Timestamp:      now,
		Payload: map[string]any{
			"source": "repository_test",
		},
	})
	state.mu.Unlock()

	syncExecutionDomainBestEffort(state)

	state.mu.Lock()
	state.conversations = map[string]Conversation{}
	state.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/"+conversationID+"/events", nil)
	req.SetPathValue("session_id", conversationID)
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)
	res := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		ConversationEventsHandler(state).ServeHTTP(res, req)
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
		t.Fatal("timed out waiting for conversation events handler to exit")
	}

	body := res.Body.String()
	if !strings.Contains(body, "\"session_id\":\""+conversationID+"\"") {
		t.Fatalf("expected SSE payload to contain session_id %q, got %s", conversationID, body)
	}
	if !strings.Contains(body, "\"run_id\":\""+executionID+"\"") {
		t.Fatalf("expected SSE payload to contain run_id %q, got %s", executionID, body)
	}

	state.mu.RLock()
	_, hydrated := state.conversations[conversationID]
	state.mu.RUnlock()
	if !hydrated {
		t.Fatalf("expected conversation seed to be hydrated from repository")
	}
}
