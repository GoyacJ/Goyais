// Copyright (c) 2026 Ysmjjsy
// Author: Goyais Team
// SPDX-License-Identifier: MIT

package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestConversationStopHandlerUsesRepositoryWhenConversationAndExecutionMapMissing(t *testing.T) {
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
	conversationID := "conv_stop_repo_" + randomHex(4)
	executionID := "exec_stop_repo_" + randomHex(4)

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:                conversationID,
		WorkspaceID:       localWorkspaceID,
		ProjectID:         "proj_stop_repo_" + randomHex(4),
		Name:              "Stop Repository",
		QueueState:        QueueStateRunning,
		DefaultMode:       PermissionModeDefault,
		ModelConfigID:     "rc_model_stop_repo",
		ActiveExecutionID: ptrString(executionID),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	state.executions[executionID] = Execution{
		ID:             executionID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_stop_repo_" + randomHex(4),
		State:          RunStateExecuting,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     0,
		TraceID:        "tr_stop_repo_" + randomHex(4),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.conversationExecutionOrder[conversationID] = []string{executionID}
	state.mu.Unlock()

	syncExecutionDomainBestEffort(state)

	state.mu.Lock()
	state.conversations = map[string]Conversation{}
	state.executions = map[string]Execution{}
	state.conversationExecutionOrder = map[string][]string{}
	state.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/v1/sessions/"+conversationID+"/stop", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("session_id", conversationID)
	res := httptest.NewRecorder()
	ConversationStopHandler(state).ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected conversation stop 200 with repository seeds, got %d (%s)", res.Code, res.Body.String())
	}

	state.mu.RLock()
	conversation := state.conversations[conversationID]
	execution, executionExists := state.executions[executionID]
	state.mu.RUnlock()

	if conversation.ActiveExecutionID != nil {
		t.Fatalf("expected active execution to be cleared, got %q", *conversation.ActiveExecutionID)
	}
	if !executionExists {
		t.Fatalf("expected execution seed to be hydrated into state map")
	}
	if execution.State != RunStateCancelled {
		t.Fatalf("expected execution state cancelled, got %s", execution.State)
	}
}

func TestConversationExportHandlerUsesRepositoryWhenConversationMapMissing(t *testing.T) {
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
	conversationID := "conv_export_repo_" + randomHex(4)

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:            conversationID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     "proj_export_repo_" + randomHex(4),
		Name:          "Export Repository",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_export_repo",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.conversationMessages[conversationID] = []ConversationMessage{
		{
			ID:             "msg_export_repo_" + randomHex(4),
			ConversationID: conversationID,
			Role:           MessageRoleUser,
			Content:        "hello export",
			CreatedAt:      now,
		},
	}
	state.mu.Unlock()

	syncExecutionDomainBestEffort(state)

	state.mu.Lock()
	state.conversations = map[string]Conversation{}
	state.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/"+conversationID+"/export?format=markdown", nil)
	req.SetPathValue("session_id", conversationID)
	res := httptest.NewRecorder()
	ConversationExportHandler(state).ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected conversation export 200 with repository seed, got %d (%s)", res.Code, res.Body.String())
	}
	body := res.Body.String()
	if !strings.Contains(body, "Export Repository") {
		t.Fatalf("expected markdown export to contain conversation title, got %q", body)
	}
	if !strings.Contains(body, "hello export") {
		t.Fatalf("expected markdown export to contain message content, got %q", body)
	}
}
