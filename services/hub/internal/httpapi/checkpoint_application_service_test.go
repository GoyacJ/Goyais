package httpapi

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"goyais/services/hub/internal/domain"
)

type checkpointDomainServiceStub struct {
	listItems       []domain.Checkpoint
	createCheckpoint domain.Checkpoint
	rollbackResult  domain.RollbackResult
}

func (s *checkpointDomainServiceStub) ListSessionCheckpoints(_ context.Context, _ domain.SessionID) ([]domain.Checkpoint, error) {
	return append([]domain.Checkpoint{}, s.listItems...), nil
}

func (s *checkpointDomainServiceStub) CreateCheckpoint(_ context.Context, _ domain.CreateCheckpointRequest) (domain.Checkpoint, error) {
	return s.createCheckpoint, nil
}

func (s *checkpointDomainServiceStub) RollbackToCheckpoint(_ context.Context, _ domain.SessionID, _ string) (domain.RollbackResult, error) {
	return s.rollbackResult, nil
}

func TestCheckpointApplicationServiceAdapterCreatePreservesCheckpointFields(t *testing.T) {
	service := &checkpointDomainServiceStub{
		createCheckpoint: domain.Checkpoint{
			CheckpointID:       "cp_create",
			SessionID:          "sess_create",
			WorkspaceID:        localWorkspaceID,
			ProjectID:          "proj_create",
			Message:            "checkpoint create",
			ProjectKind:        domain.CheckpointProjectKindGit,
			CreatedAt:          "2026-03-08T00:00:00Z",
			ParentCheckpointID: "cp_parent",
			Session: &domain.CheckpointSession{
				ID:            "sess_create",
				WorkspaceID:   localWorkspaceID,
				ProjectID:     "proj_create",
				Name:          "Created Session",
				QueueState:    string(QueueStateIdle),
				DefaultMode:   string(PermissionModeDefault),
				ModelConfigID: "model_create",
			},
		},
	}
	adapter := &checkpointApplicationServiceAdapter{
		service: service,
	}

	checkpoint, err := adapter.CreateSessionCheckpoint(context.Background(), "sess_create", "save")
	if err != nil {
		t.Fatalf("create session checkpoint failed: %v", err)
	}
	if checkpoint.ParentCheckpointID != "cp_parent" {
		t.Fatalf("expected parent checkpoint preserved, got %#v", checkpoint)
	}
	if checkpoint.Session == nil || checkpoint.Session.Name != "Created Session" {
		t.Fatalf("expected checkpoint session details preserved, got %#v", checkpoint.Session)
	}
	if checkpoint.Session.ProjectID != "proj_create" || checkpoint.Session.ModelConfigID != "model_create" {
		t.Fatalf("expected checkpoint session project/model preserved, got %#v", checkpoint.Session)
	}
}

func TestCheckpointApplicationServiceAdapterRollbackPreservesRestoredSessionFields(t *testing.T) {
	activeExecutionID := "exec_rollback"
	service := &checkpointDomainServiceStub{
		rollbackResult: domain.RollbackResult{
			Checkpoint: domain.Checkpoint{
				CheckpointID:       "cp_rollback",
				SessionID:          "sess_rollback",
				WorkspaceID:        localWorkspaceID,
				ProjectID:          "proj_rollback",
				Message:            "checkpoint rollback",
				ProjectKind:        domain.CheckpointProjectKindNonGit,
				CreatedAt:          "2026-03-08T00:01:00Z",
				ParentCheckpointID: "cp_before",
			},
			Session: domain.CheckpointSession{
				ID:                "sess_rollback",
				WorkspaceID:       localWorkspaceID,
				ProjectID:         "proj_rollback",
				Name:              "Restored Session",
				QueueState:        string(QueueStateIdle),
				DefaultMode:       string(PermissionModePlan),
				ModelConfigID:     "model_rollback",
				RuleIDs:           []string{"rule_a"},
				SkillIDs:          []string{"skill_a"},
				MCPIDs:            []string{"mcp_a"},
				BaseRevision:      42,
				ActiveExecutionID: &activeExecutionID,
				TokensInTotal:     7,
				TokensOutTotal:    9,
				TokensTotal:       16,
				CreatedAt:         "2026-03-08T00:00:00Z",
				UpdatedAt:         "2026-03-08T00:02:00Z",
			},
		},
	}
	adapter := &checkpointApplicationServiceAdapter{
		service: service,
	}

	checkpoint, session, err := adapter.RollbackSessionToCheckpoint(context.Background(), "sess_rollback", "cp_rollback")
	if err != nil {
		t.Fatalf("rollback session checkpoint failed: %v", err)
	}
	if checkpoint.ParentCheckpointID != "cp_before" {
		t.Fatalf("expected rollback checkpoint parent preserved, got %#v", checkpoint)
	}
	if session.Name != "Restored Session" || session.ProjectID != "proj_rollback" {
		t.Fatalf("expected restored session details preserved, got %#v", session)
	}
	if session.DefaultMode != PermissionModePlan || session.ModelConfigID != "model_rollback" {
		t.Fatalf("expected restored session mode/model preserved, got %#v", session)
	}
	if session.ActiveExecutionID == nil || *session.ActiveExecutionID != "exec_rollback" {
		t.Fatalf("expected restored session active execution preserved, got %#v", session.ActiveExecutionID)
	}
	if len(session.RuleIDs) != 1 || len(session.SkillIDs) != 1 || len(session.MCPIDs) != 1 {
		t.Fatalf("expected restored session resource selections preserved, got %#v", session)
	}
}

func TestCheckpointRepositoryAdapterRollbackUsesCapturedRuntimeMetadata(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	})

	state := NewAppState(store)
	now := "2026-03-08T00:00:00Z"
	projectID := "proj_runtime_metadata"
	sessionID := "sess_runtime_metadata"
	projectRepo := t.TempDir()

	project, err := saveProjectToStore(state, Project{
		ID:          projectID,
		WorkspaceID: localWorkspaceID,
		Name:        "Runtime Metadata Project",
		RepoPath:    projectRepo,
		DefaultMode: PermissionModeDefault,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("save project failed: %v", err)
	}
	state.conversations[sessionID] = Conversation{
		ID:            sessionID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     project.ID,
		Name:          "Runtime Metadata Session",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModePlan,
		ModelConfigID: "rc_model_runtime",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.conversationSessionIDs[sessionID] = "rt_sess_original"
	state.conversationMessages[sessionID] = []ConversationMessage{
		{
			ID:             "msg_runtime_1",
			ConversationID: sessionID,
			Role:           MessageRoleUser,
			Content:        "first",
			CreatedAt:      now,
		},
		{
			ID:             "msg_runtime_2",
			ConversationID: sessionID,
			Role:           MessageRoleAssistant,
			Content:        "second",
			CreatedAt:      now,
		},
	}

	checkpoint, err := createSessionCheckpoint(state, sessionID, "savepoint")
	if err != nil {
		t.Fatalf("create session checkpoint failed: %v", err)
	}

	mutatedProject, err := saveProjectToStore(state, Project{
		ID:          projectID,
		WorkspaceID: localWorkspaceID,
		Name:        "Runtime Metadata Project",
		RepoPath:    t.TempDir(),
		DefaultMode: PermissionModeDefault,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		t.Fatalf("mutate project failed: %v", err)
	}
	state.projects[projectID] = mutatedProject
	state.conversationSessionIDs[sessionID] = "rt_sess_mutated"
	state.conversationMessages[sessionID] = append(state.conversationMessages[sessionID], ConversationMessage{
		ID:             "msg_runtime_3",
		ConversationID: sessionID,
		Role:           MessageRoleUser,
		Content:        "mutated",
		CreatedAt:      now,
	})

	service := newCheckpointDomainService(state)
	result, err := service.RollbackToCheckpoint(context.Background(), domain.SessionID(sessionID), checkpoint.CheckpointID)
	if err != nil {
		t.Fatalf("rollback checkpoint failed: %v", err)
	}
	if result.Runtime.WorkingDir != projectRepo {
		t.Fatalf("working dir = %q, want %q", result.Runtime.WorkingDir, projectRepo)
	}
	if result.Runtime.HistoryEntries != 2 {
		t.Fatalf("history entries = %d, want 2", result.Runtime.HistoryEntries)
	}
	if got := state.conversationSessionIDs[sessionID]; got != "rt_sess_original" {
		t.Fatalf("runtime session id = %q, want rt_sess_original", got)
	}
}

func TestCheckpointRollbackRestoresNonGitFilesFromExecutionDiffsWhenLedgerMissing(t *testing.T) {
	state := NewAppState(nil)
	now := "2026-03-08T00:00:00Z"
	projectID := "proj_checkpoint_non_git_restore"
	sessionID := "sess_checkpoint_non_git_restore"
	repoPath := t.TempDir()
	filePath := filepath.Join(repoPath, "src", "main.txt")
	beforeContent := []byte("before checkpoint\n")
	afterContent := []byte("after checkpoint\n")

	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filePath, beforeContent, 0o644); err != nil {
		t.Fatalf("write before file failed: %v", err)
	}

	state.projects[projectID] = Project{
		ID:          projectID,
		WorkspaceID: localWorkspaceID,
		Name:        "Checkpoint Non Git Restore",
		RepoPath:    repoPath,
		IsGit:       false,
		DefaultMode: PermissionModeDefault,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	state.conversations[sessionID] = Conversation{
		ID:            sessionID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     projectID,
		Name:          "Checkpoint Non Git Restore Session",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_restore",
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	checkpoint, err := createSessionCheckpoint(state, sessionID, "savepoint")
	if err != nil {
		t.Fatalf("create session checkpoint failed: %v", err)
	}

	if err := os.WriteFile(filePath, afterContent, 0o644); err != nil {
		t.Fatalf("write after file failed: %v", err)
	}

	executionID := "exec_checkpoint_non_git_restore"
	state.executions[executionID] = Execution{
		ID:             executionID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: sessionID,
		MessageID:      "msg_restore",
		State:          RunStateCompleted,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     0,
		TraceID:        "tr_restore",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.conversationExecutionOrder[sessionID] = []string{executionID}
	state.executionDiffs[executionID] = []DiffItem{{
		ID:         "diff_checkpoint_non_git_restore",
		Path:       "src/main.txt",
		ChangeType: "modified",
		Summary:    "restore from diff",
		BeforeBlob: base64.StdEncoding.EncodeToString(beforeContent),
		AfterBlob:  base64.StdEncoding.EncodeToString(afterContent),
	}}
	state.conversationChangeLedgers[sessionID] = &ConversationChangeLedger{
		ConversationID: sessionID,
		ProjectKind:    "non_git",
		Entries:        nil,
	}

	if _, _, _, err := rollbackSessionToCheckpoint(state, sessionID, checkpoint.CheckpointID); err != nil {
		t.Fatalf("rollback checkpoint failed: %v", err)
	}

	restored, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read restored file failed: %v", err)
	}
	if string(restored) != string(beforeContent) {
		t.Fatalf("restored file = %q, want %q", restored, beforeContent)
	}
}
