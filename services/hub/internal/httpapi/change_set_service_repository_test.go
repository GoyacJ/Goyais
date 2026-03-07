// Copyright (c) 2026 Ysmjjsy
// Author: Goyais Team
// SPDX-License-Identifier: MIT

package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestApplyExecutionEventToChangeLedgerLockedUsesRepositoryMessageID(t *testing.T) {
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
	now := "2026-03-04T00:00:00Z"
	projectID := "proj_change_set_repo_msg_" + randomHex(4)
	conversationID := "conv_change_set_repo_msg_" + randomHex(4)
	executionID := "exec_change_set_repo_msg_" + randomHex(4)
	messageID := "msg_change_set_repo_msg_" + randomHex(4)

	state.mu.Lock()
	state.projects[projectID] = Project{
		ID:          projectID,
		WorkspaceID: localWorkspaceID,
		Name:        "ChangeSet Project MessageID",
		RepoPath:    "/tmp/changeset-project-message",
		IsGit:       false,
		DefaultMode: PermissionModeDefault,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	state.conversations[conversationID] = Conversation{
		ID:            conversationID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     projectID,
		Name:          "ChangeSet Conversation MessageID",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_test",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.executions[executionID] = Execution{
		ID:             executionID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      messageID,
		State:          RunStateCompleted,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot: ModelSnapshot{
			ModelID:  "gpt-5.3",
			ConfigID: "rc_model_test",
		},
		QueueIndex: 0,
		TraceID:    "tr_change_set_repo_msg_" + randomHex(4),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	state.mu.Unlock()

	syncExecutionDomainBestEffort(state)

	state.mu.Lock()
	state.executions = map[string]Execution{}
	applyExecutionEventToChangeLedgerLocked(state, ExecutionEvent{
		ExecutionID:    executionID,
		ConversationID: conversationID,
		Type:           RunEventTypeDiffGenerated,
		Timestamp:      now,
		Payload: map[string]any{
			"diff": []any{
				map[string]any{
					"id":          "diff_change_set_repo_msg_1",
					"path":        "README.md",
					"change_type": "modified",
					"summary":     "Update README",
				},
			},
		},
	})
	ledger := state.conversationChangeLedgers[conversationID]
	state.mu.Unlock()

	if ledger == nil || len(ledger.Entries) != 1 {
		t.Fatalf("expected one ledger entry, got %#v", ledger)
	}
	if ledger.Entries[0].MessageID != messageID {
		t.Fatalf("expected ledger entry message_id %q, got %q", messageID, ledger.Entries[0].MessageID)
	}
}

func TestRebuildConversationChangeLedgerFromStateLockedUsesRepositoryExecutionSeed(t *testing.T) {
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
	now := "2026-03-04T00:00:00Z"
	projectID := "proj_change_set_repo_rebuild_" + randomHex(4)
	conversationID := "conv_change_set_repo_rebuild_" + randomHex(4)
	executionID := "exec_change_set_repo_rebuild_" + randomHex(4)
	messageID := "msg_change_set_repo_rebuild_" + randomHex(4)

	state.mu.Lock()
	state.projects[projectID] = Project{
		ID:          projectID,
		WorkspaceID: localWorkspaceID,
		Name:        "ChangeSet Project Rebuild",
		RepoPath:    "/tmp/changeset-project-rebuild",
		IsGit:       false,
		DefaultMode: PermissionModeDefault,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	state.conversations[conversationID] = Conversation{
		ID:            conversationID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     projectID,
		Name:          "ChangeSet Conversation Rebuild",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_test",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.executions[executionID] = Execution{
		ID:             executionID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      messageID,
		State:          RunStateCompleted,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot: ModelSnapshot{
			ModelID:  "gpt-5.3",
			ConfigID: "rc_model_test",
		},
		QueueIndex: 0,
		TraceID:    "tr_change_set_repo_rebuild_" + randomHex(4),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	state.conversationExecutionOrder[conversationID] = []string{executionID}
	state.executionDiffs[executionID] = []DiffItem{
		{
			ID:         "diff_change_set_repo_rebuild_1",
			Path:       "docs/guide.md",
			ChangeType: "modified",
			Summary:    "Update guide",
		},
	}
	state.mu.Unlock()

	syncExecutionDomainBestEffort(state)

	state.mu.Lock()
	state.executions = map[string]Execution{}
	ledger := rebuildConversationChangeLedgerFromStateLocked(state, conversationID)
	state.mu.Unlock()

	if ledger == nil || len(ledger.Entries) != 1 {
		t.Fatalf("expected one ledger entry after rebuild, got %#v", ledger)
	}
	entry := ledger.Entries[0]
	if entry.ExecutionID != executionID {
		t.Fatalf("expected ledger entry execution_id %q, got %q", executionID, entry.ExecutionID)
	}
	if entry.MessageID != messageID {
		t.Fatalf("expected ledger entry message_id %q, got %q", messageID, entry.MessageID)
	}
}

func TestHasMutableExecutionsLockedFallsBackToRepositoryRuns(t *testing.T) {
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
	now := "2026-03-04T00:00:00Z"
	conversationID := "conv_change_set_repo_mutable_" + randomHex(4)
	executionID := "exec_change_set_repo_mutable_" + randomHex(4)

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:            conversationID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     "proj_change_set_repo_mutable_" + randomHex(4),
		Name:          "ChangeSet Conversation Mutable",
		QueueState:    QueueStateQueued,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_test",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.executions[executionID] = Execution{
		ID:             executionID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_change_set_repo_mutable_" + randomHex(4),
		State:          RunStatePending,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot: ModelSnapshot{
			ModelID:  "gpt-5.3",
			ConfigID: "rc_model_test",
		},
		QueueIndex: 0,
		TraceID:    "tr_change_set_repo_mutable_" + randomHex(4),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	state.conversationExecutionOrder[conversationID] = []string{executionID}
	state.mu.Unlock()

	syncExecutionDomainBestEffort(state)

	state.mu.Lock()
	state.executions = map[string]Execution{}
	state.conversationExecutionOrder = map[string][]string{}
	isBusy := hasMutableExecutionsLocked(state, conversationID)
	state.mu.Unlock()

	if !isBusy {
		t.Fatalf("expected mutable execution to be detected from repository fallback")
	}
}

func TestConversationChangeSetHandlerUsesRepositoryAndProjectStoreSeeds(t *testing.T) {
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
	now := "2026-03-04T00:00:00Z"
	projectID := "proj_change_set_repo_handler_" + randomHex(4)
	conversationID := "conv_change_set_repo_handler_" + randomHex(4)

	project, saveErr := saveProjectToStore(state, Project{
		ID:          projectID,
		WorkspaceID: localWorkspaceID,
		Name:        "ChangeSet Handler Project",
		RepoPath:    "/tmp/changeset-handler-project",
		IsGit:       false,
		DefaultMode: PermissionModeDefault,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if saveErr != nil {
		t.Fatalf("save project failed: %v", saveErr)
	}

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:            conversationID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     project.ID,
		Name:          "ChangeSet Handler Conversation",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_test",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.conversationChangeLedgers[conversationID] = &ConversationChangeLedger{
		ConversationID:     conversationID,
		ProjectKind:        "non_git",
		PendingChangeSetID: "cs_handler_repo_seed",
		Entries:            []ChangeEntry{},
	}
	state.mu.Unlock()

	syncExecutionDomainBestEffort(state)

	state.mu.Lock()
	state.conversations = map[string]Conversation{}
	state.projects = map[string]Project{}
	state.mu.Unlock()

	handler := ConversationChangeSetHandler(state)
	req := httptest.NewRequest(http.MethodGet, "/v1/sessions/"+conversationID+"/changeset", nil)
	req.SetPathValue("session_id", conversationID)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected changeset handler 200 with repository/project fallback, got %d (%s)", res.Code, res.Body.String())
	}

	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	if got := strings.TrimSpace(asString(payload["session_id"])); got != conversationID {
		t.Fatalf("expected session_id %q, got %q", conversationID, got)
	}

	state.mu.RLock()
	_, hasConversationSeed := state.conversations[conversationID]
	_, hasProjectSeed := state.projects[projectID]
	state.mu.RUnlock()
	if !hasConversationSeed || !hasProjectSeed {
		t.Fatalf("expected handler to hydrate conversation/project seeds into state maps")
	}
}

func TestConversationChangeSetCommitHandlerUsesRepositoryAndProjectStoreSeeds(t *testing.T) {
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
	now := "2026-03-04T00:00:00Z"
	projectID := "proj_change_set_repo_commit_" + randomHex(4)
	conversationID := "conv_change_set_repo_commit_" + randomHex(4)

	project, saveErr := saveProjectToStore(state, Project{
		ID:          projectID,
		WorkspaceID: localWorkspaceID,
		Name:        "ChangeSet Commit Project",
		RepoPath:    "/tmp/changeset-commit-project",
		IsGit:       false,
		DefaultMode: PermissionModeDefault,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if saveErr != nil {
		t.Fatalf("save project failed: %v", saveErr)
	}

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:            conversationID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     project.ID,
		Name:          "ChangeSet Commit Conversation",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_test",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.conversationChangeLedgers[conversationID] = &ConversationChangeLedger{
		ConversationID:     conversationID,
		ProjectKind:        "non_git",
		PendingChangeSetID: "cs_commit_repo_seed",
		Entries:            []ChangeEntry{},
	}
	state.mu.Unlock()

	syncExecutionDomainBestEffort(state)

	state.mu.Lock()
	state.conversations = map[string]Conversation{}
	state.projects = map[string]Project{}
	state.mu.Unlock()

	handler := ConversationChangeSetCommitHandler(state)
	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/sessions/"+conversationID+"/changeset/commit",
		strings.NewReader(`{"expected_change_set_id":"cs_commit_repo_seed"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("session_id", conversationID)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusConflict {
		t.Fatalf("expected changeset commit 409 for empty entries with repository/project fallback, got %d (%s)", res.Code, res.Body.String())
	}

	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	if got := strings.TrimSpace(asString(payload["code"])); got != "CHANGESET_EMPTY" {
		t.Fatalf("expected CHANGESET_EMPTY, got %q", got)
	}

	state.mu.RLock()
	_, hasConversationSeed := state.conversations[conversationID]
	_, hasProjectSeed := state.projects[projectID]
	state.mu.RUnlock()
	if !hasConversationSeed || !hasProjectSeed {
		t.Fatalf("expected commit handler to hydrate conversation/project seeds into state maps")
	}
}

func TestBuildConversationChangeSetLockedUsesRepositoryAndProjectStoreSeeds(t *testing.T) {
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
	now := "2026-03-04T00:00:00Z"
	projectID := "proj_change_set_repo_build_" + randomHex(4)
	conversationID := "conv_change_set_repo_build_" + randomHex(4)

	project, saveErr := saveProjectToStore(state, Project{
		ID:          projectID,
		WorkspaceID: localWorkspaceID,
		Name:        "ChangeSet Build Project",
		RepoPath:    "/tmp/changeset-build-project",
		IsGit:       false,
		DefaultMode: PermissionModeDefault,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if saveErr != nil {
		t.Fatalf("save project failed: %v", saveErr)
	}

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:            conversationID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     project.ID,
		Name:          "ChangeSet Build Conversation",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_test",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.mu.Unlock()

	syncExecutionDomainBestEffort(state)

	state.mu.Lock()
	state.conversations = map[string]Conversation{}
	state.projects = map[string]Project{}
	changeSet, buildErr := buildConversationChangeSetLocked(state, conversationID)
	_, hasConversationSeed := state.conversations[conversationID]
	_, hasProjectSeed := state.projects[projectID]
	state.mu.Unlock()

	if buildErr != nil {
		t.Fatalf("expected build changeset to use repository/project fallback, got err: %v", buildErr)
	}
	if got := strings.TrimSpace(changeSet.ConversationID); got != conversationID {
		t.Fatalf("expected session_id %q, got %q", conversationID, got)
	}
	if !hasConversationSeed || !hasProjectSeed {
		t.Fatalf("expected buildConversationChangeSetLocked to hydrate conversation/project seeds")
	}
}

func TestResolveConversationProjectKindLockedUsesRepositoryAndProjectStoreSeeds(t *testing.T) {
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
	now := "2026-03-04T00:00:00Z"
	projectID := "proj_change_set_repo_kind_" + randomHex(4)
	conversationID := "conv_change_set_repo_kind_" + randomHex(4)

	project, saveErr := saveProjectToStore(state, Project{
		ID:          projectID,
		WorkspaceID: localWorkspaceID,
		Name:        "ChangeSet Kind Project",
		RepoPath:    "/tmp/changeset-kind-project",
		IsGit:       false,
		DefaultMode: PermissionModeDefault,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if saveErr != nil {
		t.Fatalf("save project failed: %v", saveErr)
	}

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:            conversationID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     project.ID,
		Name:          "ChangeSet Kind Conversation",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_test",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.mu.Unlock()

	syncExecutionDomainBestEffort(state)

	state.mu.Lock()
	state.conversations = map[string]Conversation{}
	state.projects = map[string]Project{}
	kind := resolveConversationProjectKindLocked(state, conversationID)
	_, hasConversationSeed := state.conversations[conversationID]
	_, hasProjectSeed := state.projects[projectID]
	state.mu.Unlock()

	if kind != "non_git" {
		t.Fatalf("expected non_git project kind, got %q", kind)
	}
	if !hasConversationSeed || !hasProjectSeed {
		t.Fatalf("expected resolveConversationProjectKindLocked to hydrate conversation/project seeds")
	}
}
