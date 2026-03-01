package httpapi

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConversationRollbackHandlerFallsBackWithoutSnapshot(t *testing.T) {
	state := NewAppState(nil)
	now := "2026-03-01T00:00:00Z"
	projectID := "proj_rollback_fallback"
	conversationID := "conv_rollback_fallback"
	executionOneID := "exec_rollback_fallback_1"
	executionTwoID := "exec_rollback_fallback_2"
	state.mu.Lock()
	state.projects[projectID] = Project{
		ID:                   projectID,
		WorkspaceID:          localWorkspaceID,
		Name:                 "Rollback Fallback Project",
		RepoPath:             t.TempDir(),
		IsGit:                false,
		DefaultModelConfigID: "rc_model_1",
		DefaultMode:          PermissionModeDefault,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	state.conversations[conversationID] = Conversation{
		ID:            conversationID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     projectID,
		Name:          "Rollback Fallback Conversation",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_1",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	queueIndexZero := 0
	queueIndexOne := 1
	canRollback := true
	state.conversationMessages[conversationID] = []ConversationMessage{
		{
			ID:             "msg_rollback_target",
			ConversationID: conversationID,
			Role:           MessageRoleUser,
			Content:        "first",
			QueueIndex:     &queueIndexZero,
			CanRollback:    &canRollback,
			CreatedAt:      now,
		},
		{
			ID:             "msg_rollback_target_assistant",
			ConversationID: conversationID,
			Role:           MessageRoleAssistant,
			Content:        "first answer",
			QueueIndex:     &queueIndexZero,
			CreatedAt:      now,
		},
		{
			ID:             "msg_rollback_extra",
			ConversationID: conversationID,
			Role:           MessageRoleUser,
			Content:        "second",
			QueueIndex:     &queueIndexOne,
			CanRollback:    &canRollback,
			CreatedAt:      now,
		},
	}
	state.executions[executionOneID] = Execution{
		ID:             executionOneID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_rollback_target",
		State:          ExecutionStateCompleted,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     queueIndexZero,
		TraceID:        "tr_rollback_fallback_1",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.executions[executionTwoID] = Execution{
		ID:             executionTwoID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_rollback_extra",
		State:          ExecutionStateCompleted,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     queueIndexOne,
		TraceID:        "tr_rollback_fallback_2",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.conversationExecutionOrder[conversationID] = []string{executionOneID, executionTwoID}
	state.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/v2/conversations/"+conversationID+"/rollback", strings.NewReader(`{"message_id":"msg_rollback_target"}`))
	req.SetPathValue("conversation_id", conversationID)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	ConversationRollbackHandler(state).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected fallback rollback 200, got %d (%s)", recorder.Code, recorder.Body.String())
	}
	state.mu.RLock()
	defer state.mu.RUnlock()
	if len(state.conversationMessages[conversationID]) != 1 {
		t.Fatalf("expected rollback to keep only target user message, got %#v", state.conversationMessages[conversationID])
	}
	if state.conversationMessages[conversationID][0].ID != "msg_rollback_target" {
		t.Fatalf("expected remaining message to be rollback target, got %#v", state.conversationMessages[conversationID][0])
	}
	if len(state.conversationExecutionOrder[conversationID]) != 1 || state.conversationExecutionOrder[conversationID][0] != executionOneID {
		t.Fatalf("expected rollback to keep first execution only, got %#v", state.conversationExecutionOrder[conversationID])
	}
}

func TestConversationRollbackHandlerSkipsGitRestoreWhenRepoIsNotGit(t *testing.T) {
	repoPath := t.TempDir()
	filePath := filepath.Join(repoPath, "src", "main.ts")
	writeTestFile(t, filePath, []byte("console.log('changed')\n"))

	state := NewAppState(nil)
	now := "2026-03-01T00:00:00Z"
	projectID := "proj_rollback_non_git_marked_git"
	conversationID := "conv_rollback_non_git_marked_git"
	executionKeepID := "exec_rollback_non_git_keep"
	executionDropID := "exec_rollback_non_git_drop"
	queueIndexZero := 0
	queueIndexOne := 1
	canRollback := true

	state.mu.Lock()
	state.projects[projectID] = Project{
		ID:                   projectID,
		WorkspaceID:          localWorkspaceID,
		Name:                 "Rollback Non Git Marked Git",
		RepoPath:             repoPath,
		IsGit:                true,
		DefaultModelConfigID: "rc_model_1",
		DefaultMode:          PermissionModeDefault,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	state.conversations[conversationID] = Conversation{
		ID:            conversationID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     projectID,
		Name:          "Rollback Non Git Marked Git Conversation",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_1",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.conversationMessages[conversationID] = []ConversationMessage{
		{
			ID:             "msg_rollback_non_git_target",
			ConversationID: conversationID,
			Role:           MessageRoleUser,
			Content:        "first",
			QueueIndex:     &queueIndexZero,
			CanRollback:    &canRollback,
			CreatedAt:      now,
		},
		{
			ID:             "msg_rollback_non_git_next",
			ConversationID: conversationID,
			Role:           MessageRoleUser,
			Content:        "second",
			QueueIndex:     &queueIndexOne,
			CanRollback:    &canRollback,
			CreatedAt:      now,
		},
	}
	state.executions[executionKeepID] = Execution{
		ID:             executionKeepID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_rollback_non_git_target",
		State:          ExecutionStateCompleted,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     queueIndexZero,
		TraceID:        "tr_rollback_non_git_keep",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.executions[executionDropID] = Execution{
		ID:             executionDropID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_rollback_non_git_next",
		State:          ExecutionStateCompleted,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     queueIndexOne,
		TraceID:        "tr_rollback_non_git_drop",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.executionDiffs[executionDropID] = []DiffItem{{
		ID:         "diff_rollback_non_git_drop",
		Path:       "src/main.ts",
		ChangeType: "modified",
		Summary:    "non git rollback update",
	}}
	state.conversationExecutionOrder[conversationID] = []string{executionKeepID, executionDropID}
	state.conversationSnapshots[conversationID] = []ConversationSnapshot{
		{
			ID:                     "snap_rollback_non_git_target",
			ConversationID:         conversationID,
			RollbackPointMessageID: "msg_rollback_non_git_target",
			QueueState:             QueueStateIdle,
			InspectorState:         ConversationInspector{Tab: "diff"},
			Messages: []ConversationMessage{
				{
					ID:             "msg_rollback_non_git_target",
					ConversationID: conversationID,
					Role:           MessageRoleUser,
					Content:        "first",
					QueueIndex:     &queueIndexZero,
					CanRollback:    &canRollback,
					CreatedAt:      now,
				},
			},
			ExecutionIDs: []string{executionKeepID},
			CreatedAt:    now,
		},
	}
	state.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/v2/conversations/"+conversationID+"/rollback", strings.NewReader(`{"message_id":"msg_rollback_non_git_target"}`))
	req.SetPathValue("conversation_id", conversationID)
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	ConversationRollbackHandler(state).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected rollback 200, got %d (%s)", recorder.Code, recorder.Body.String())
	}
	state.mu.RLock()
	defer state.mu.RUnlock()
	if len(state.conversationExecutionOrder[conversationID]) != 1 || state.conversationExecutionOrder[conversationID][0] != executionKeepID {
		t.Fatalf("expected rollback to keep first execution only, got %#v", state.conversationExecutionOrder[conversationID])
	}
}

func writeTestFile(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for %s failed: %v", path, err)
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write %s failed: %v", path, err)
	}
}
