package httpapi

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecutionDiffHandlerReturnsAccumulatedDiffEntries(t *testing.T) {
	state := NewAppState(nil)
	state.mu.Lock()
	state.executions["exec_diff_1"] = Execution{
		ID:             "exec_diff_1",
		WorkspaceID:    localWorkspaceID,
		ConversationID: "conv_diff_1",
		MessageID:      "msg_diff_1",
		State:          ExecutionStateCompleted,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot: ModelSnapshot{
			ModelID: "gpt-5.3",
		},
		QueueIndex: 0,
		TraceID:    "tr_diff_1",
		CreatedAt:  "2026-02-28T00:00:00Z",
		UpdatedAt:  "2026-02-28T00:00:00Z",
	}
	appendExecutionEventLocked(state, ExecutionEvent{
		ExecutionID:    "exec_diff_1",
		ConversationID: "conv_diff_1",
		TraceID:        "tr_diff_1",
		QueueIndex:     0,
		Type:           ExecutionEventTypeDiffGenerated,
		Payload: map[string]any{
			"diff": []any{
				map[string]any{
					"path":          "README.md",
					"change_type":   "modified",
					"summary":       "updated",
					"added_lines":   5,
					"deleted_lines": 2,
				},
			},
		},
	})
	appendExecutionEventLocked(state, ExecutionEvent{
		ExecutionID:    "exec_diff_1",
		ConversationID: "conv_diff_1",
		TraceID:        "tr_diff_1",
		QueueIndex:     0,
		Type:           ExecutionEventTypeDiffGenerated,
		Payload: map[string]any{
			"diff": []any{
				map[string]any{
					"path":        "src/main.ts",
					"change_type": "added",
					"summary":     "created",
				},
			},
		},
	})
	state.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/v1/executions/exec_diff_1/diff", nil)
	req.SetPathValue("execution_id", "exec_diff_1")
	recorder := httptest.NewRecorder()
	ExecutionDiffHandler(state).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", recorder.Code, recorder.Body.String())
	}
	diff := []DiffItem{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &diff); err != nil {
		t.Fatalf("decode diff response failed: %v", err)
	}
	if len(diff) != 2 {
		t.Fatalf("expected accumulated diff entries, got %#v", diff)
	}
	if diff[0].Path != "README.md" || diff[1].Path != "src/main.ts" {
		t.Fatalf("expected ordered accumulated paths, got %#v", diff)
	}
	if diff[0].AddedLines == nil || *diff[0].AddedLines != 5 {
		t.Fatalf("expected README added line count 5, got %#v", diff[0].AddedLines)
	}
	if diff[0].DeletedLines == nil || *diff[0].DeletedLines != 2 {
		t.Fatalf("expected README deleted line count 2, got %#v", diff[0].DeletedLines)
	}
	if diff[1].AddedLines != nil || diff[1].DeletedLines != nil {
		t.Fatalf("expected missing line counts for second diff item, got %+v", diff[1])
	}
}

func TestExecutionDiffHandlerReturnsConversationWideDiffEntries(t *testing.T) {
	state := NewAppState(nil)
	state.mu.Lock()
	state.executions["exec_diff_conv_1"] = Execution{
		ID:             "exec_diff_conv_1",
		WorkspaceID:    localWorkspaceID,
		ConversationID: "conv_diff_group_1",
		MessageID:      "msg_diff_group_1",
		State:          ExecutionStateCompleted,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     0,
		TraceID:        "tr_diff_group_1",
		CreatedAt:      "2026-02-28T00:00:00Z",
		UpdatedAt:      "2026-02-28T00:00:00Z",
	}
	state.executions["exec_diff_conv_2"] = Execution{
		ID:             "exec_diff_conv_2",
		WorkspaceID:    localWorkspaceID,
		ConversationID: "conv_diff_group_1",
		MessageID:      "msg_diff_group_2",
		State:          ExecutionStateCompleted,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     1,
		TraceID:        "tr_diff_group_2",
		CreatedAt:      "2026-02-28T00:00:01Z",
		UpdatedAt:      "2026-02-28T00:00:01Z",
	}
	state.executionDiffs["exec_diff_conv_1"] = []DiffItem{
		{
			ID:         "diff_conv_1",
			Path:       "src/first.ts",
			ChangeType: "modified",
			Summary:    "first",
		},
	}
	state.executionDiffs["exec_diff_conv_2"] = []DiffItem{
		{
			ID:         "diff_conv_2",
			Path:       "src/second.ts",
			ChangeType: "added",
			Summary:    "second",
		},
	}
	state.conversationExecutionOrder["conv_diff_group_1"] = []string{"exec_diff_conv_1", "exec_diff_conv_2"}
	state.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/v1/executions/exec_diff_conv_1/diff", nil)
	req.SetPathValue("execution_id", "exec_diff_conv_1")
	recorder := httptest.NewRecorder()
	ExecutionDiffHandler(state).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", recorder.Code, recorder.Body.String())
	}
	diff := []DiffItem{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &diff); err != nil {
		t.Fatalf("decode diff response failed: %v", err)
	}
	if len(diff) != 2 {
		t.Fatalf("expected conversation-wide diff entries, got %#v", diff)
	}
	if diff[0].Path != "src/first.ts" || diff[1].Path != "src/second.ts" {
		t.Fatalf("expected merged conversation diff paths, got %#v", diff)
	}
}

func TestExecutionDiffHandlerKeepsInternalLineCountsWithoutGitOverride(t *testing.T) {
	repoPath := t.TempDir()
	initGitRepoForDiffCountTest(t, repoPath)

	writeTestFile(t, filepath.Join(repoPath, "src", "tracked.ts"), []byte("line1\nline2\n"))
	runGitCommand(t, repoPath, "add", "src/tracked.ts")
	runGitCommand(t, repoPath, "commit", "-m", "init tracked file")

	writeTestFile(t, filepath.Join(repoPath, "src", "tracked.ts"), []byte("line1\nline2-changed\nline3\n"))
	state := NewAppState(nil)
	now := "2026-02-28T00:00:00Z"
	projectID := "proj_internal_diff"
	conversationID := "conv_internal_diff"
	executionID := "exec_internal_diff"
	added := 9
	deleted := 4

	state.mu.Lock()
	state.projects[projectID] = Project{
		ID:                   projectID,
		WorkspaceID:          localWorkspaceID,
		Name:                 "Internal Diff Project",
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
		Name:          "Internal Diff Conversation",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_1",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.executions[executionID] = Execution{
		ID:             executionID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_internal_diff",
		State:          ExecutionStateCompleted,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     0,
		TraceID:        "tr_internal_diff",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.executionDiffs[executionID] = []DiffItem{
		{
			ID:           "diff_tracked",
			Path:         "src/tracked.ts",
			ChangeType:   "modified",
			Summary:      "tracked update",
			AddedLines:   &added,
			DeletedLines: &deleted,
		},
	}
	state.conversationExecutionOrder[conversationID] = []string{executionID}
	state.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/v1/executions/"+executionID+"/diff", nil)
	req.SetPathValue("execution_id", executionID)
	recorder := httptest.NewRecorder()
	ExecutionDiffHandler(state).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", recorder.Code, recorder.Body.String())
	}
	diff := []DiffItem{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &diff); err != nil {
		t.Fatalf("decode diff response failed: %v", err)
	}
	if len(diff) != 1 {
		t.Fatalf("expected one diff item, got %#v", diff)
	}
	if diff[0].AddedLines == nil || *diff[0].AddedLines != added {
		t.Fatalf("expected added_lines to preserve internal value %d, got %+v", added, diff[0])
	}
	if diff[0].DeletedLines == nil || *diff[0].DeletedLines != deleted {
		t.Fatalf("expected deleted_lines to preserve internal value %d, got %+v", deleted, diff[0])
	}
}

func TestExecutionDiffHandlerDoesNotBackfillLineCountsFromGit(t *testing.T) {
	repoPath := t.TempDir()
	initGitRepoForDiffCountTest(t, repoPath)

	writeTestFile(t, filepath.Join(repoPath, "src", "tracked.ts"), []byte("line1\nline2\n"))
	runGitCommand(t, repoPath, "add", "src/tracked.ts")
	runGitCommand(t, repoPath, "commit", "-m", "init tracked file")
	writeTestFile(t, filepath.Join(repoPath, "src", "tracked.ts"), []byte("line1\nline2-changed\nline3\n"))

	state := NewAppState(nil)
	now := "2026-02-28T00:00:00Z"
	projectID := "proj_internal_diff_nobackfill"
	conversationID := "conv_internal_diff_nobackfill"
	executionID := "exec_internal_diff_nobackfill"

	state.mu.Lock()
	state.projects[projectID] = Project{
		ID:                   projectID,
		WorkspaceID:          localWorkspaceID,
		Name:                 "Internal Diff Project",
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
		Name:          "Internal Diff Conversation",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_1",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.executions[executionID] = Execution{
		ID:             executionID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_internal_diff",
		State:          ExecutionStateCompleted,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     0,
		TraceID:        "tr_internal_diff",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.executionDiffs[executionID] = []DiffItem{
		{
			ID:         "diff_tracked",
			Path:       "src/tracked.ts",
			ChangeType: "modified",
			Summary:    "tracked update",
		},
	}
	state.conversationExecutionOrder[conversationID] = []string{executionID}
	state.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/v1/executions/"+executionID+"/diff", nil)
	req.SetPathValue("execution_id", executionID)
	recorder := httptest.NewRecorder()
	ExecutionDiffHandler(state).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", recorder.Code, recorder.Body.String())
	}
	diff := []DiffItem{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &diff); err != nil {
		t.Fatalf("decode diff response failed: %v", err)
	}
	if len(diff) != 1 {
		t.Fatalf("expected one diff item, got %#v", diff)
	}
	if diff[0].AddedLines != nil || diff[0].DeletedLines != nil {
		t.Fatalf("expected handler to keep missing internal counts, got %+v", diff[0])
	}
}

func TestExecutionFilesHandlerExportsConversationWideDiffEntries(t *testing.T) {
	repoPath := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoPath, "src"), 0o755); err != nil {
		t.Fatalf("mkdir src failed: %v", err)
	}
	writeTestFile(t, filepath.Join(repoPath, "src", "first.txt"), []byte("first\n"))
	writeTestFile(t, filepath.Join(repoPath, "src", "second.txt"), []byte("second\n"))

	state := NewAppState(nil)
	now := "2026-03-01T00:00:00Z"
	projectID := "proj_files_conv_wide"
	conversationID := "conv_files_conv_wide"
	executionOneID := "exec_files_conv_wide_1"
	executionTwoID := "exec_files_conv_wide_2"

	state.mu.Lock()
	state.projects[projectID] = Project{
		ID:                   projectID,
		WorkspaceID:          localWorkspaceID,
		Name:                 "Files Export Project",
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
		Name:          "Files Export Conversation",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_1",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.executions[executionOneID] = Execution{
		ID:             executionOneID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_files_1",
		State:          ExecutionStateCompleted,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     0,
		TraceID:        "tr_files_1",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.executions[executionTwoID] = Execution{
		ID:             executionTwoID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_files_2",
		State:          ExecutionStateCompleted,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     1,
		TraceID:        "tr_files_2",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.executionDiffs[executionOneID] = []DiffItem{{
		ID:         "diff_files_first",
		Path:       "src/first.txt",
		ChangeType: "modified",
		Summary:    "first updated",
	}}
	state.executionDiffs[executionTwoID] = []DiffItem{{
		ID:         "diff_files_second",
		Path:       "src/second.txt",
		ChangeType: "modified",
		Summary:    "second updated",
	}}
	state.conversationExecutionOrder[conversationID] = []string{executionOneID, executionTwoID}
	state.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/v1/executions/"+executionTwoID+"/files", nil)
	req.SetPathValue("execution_id", executionTwoID)
	recorder := httptest.NewRecorder()
	ExecutionFilesHandler(state).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", recorder.Code, recorder.Body.String())
	}
	payload := ExecutionFilesExportResponse{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode files export response failed: %v", err)
	}
	archiveBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(payload.ArchiveBase64))
	if err != nil {
		t.Fatalf("decode archive failed: %v", err)
	}
	reader, err := zip.NewReader(bytes.NewReader(archiveBytes), int64(len(archiveBytes)))
	if err != nil {
		t.Fatalf("open archive failed: %v", err)
	}
	contents := map[string]string{}
	for _, file := range reader.File {
		handle, openErr := file.Open()
		if openErr != nil {
			t.Fatalf("open zip entry %s failed: %v", file.Name, openErr)
		}
		body, readErr := io.ReadAll(handle)
		_ = handle.Close()
		if readErr != nil {
			t.Fatalf("read zip entry %s failed: %v", file.Name, readErr)
		}
		contents[file.Name] = string(body)
	}
	if _, exists := contents["src/first.txt"]; !exists {
		t.Fatalf("expected archive to include src/first.txt, got %#v", contents)
	}
	if _, exists := contents["src/second.txt"]; !exists {
		t.Fatalf("expected archive to include src/second.txt, got %#v", contents)
	}
}

func TestExecutionFilesHandlerExportsWhenProjectIsMarkedGitButRepoIsNotGit(t *testing.T) {
	repoPath := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repoPath, "src"), 0o755); err != nil {
		t.Fatalf("mkdir src failed: %v", err)
	}
	writeTestFile(t, filepath.Join(repoPath, "src", "export.txt"), []byte("export-content\n"))

	state := NewAppState(nil)
	now := "2026-03-01T00:00:00Z"
	projectID := "proj_files_non_git_marked_git"
	conversationID := "conv_files_non_git_marked_git"
	executionID := "exec_files_non_git_marked_git"

	state.mu.Lock()
	state.projects[projectID] = Project{
		ID:                   projectID,
		WorkspaceID:          localWorkspaceID,
		Name:                 "Files Export Non Git Marked Git",
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
		Name:          "Files Export Non Git Marked Git Conversation",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_1",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.executions[executionID] = Execution{
		ID:             executionID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_files_non_git_marked_git",
		State:          ExecutionStateCompleted,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     0,
		TraceID:        "tr_files_non_git_marked_git",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.executionDiffs[executionID] = []DiffItem{{
		ID:         "diff_files_non_git_marked_git",
		Path:       "src/export.txt",
		ChangeType: "modified",
		Summary:    "export changed",
	}}
	state.conversationExecutionOrder[conversationID] = []string{executionID}
	state.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/v1/executions/"+executionID+"/files", nil)
	req.SetPathValue("execution_id", executionID)
	recorder := httptest.NewRecorder()
	ExecutionFilesHandler(state).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected files export 200, got %d (%s)", recorder.Code, recorder.Body.String())
	}
	payload := ExecutionFilesExportResponse{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode files export response failed: %v", err)
	}
	archiveBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(payload.ArchiveBase64))
	if err != nil {
		t.Fatalf("decode archive failed: %v", err)
	}
	reader, err := zip.NewReader(bytes.NewReader(archiveBytes), int64(len(archiveBytes)))
	if err != nil {
		t.Fatalf("open archive failed: %v", err)
	}
	fileNames := make([]string, 0, len(reader.File))
	for _, file := range reader.File {
		fileNames = append(fileNames, file.Name)
	}
	if !containsString(fileNames, "src/export.txt") {
		t.Fatalf("expected archive to include src/export.txt, got %#v", fileNames)
	}
}

func TestExecutionActionHandlerDiscardRestoresAbsolutePathDiff(t *testing.T) {
	repoPath := t.TempDir()
	initGitRepoForDiffCountTest(t, repoPath)
	relativePath := filepath.Join("src", "main.ts")
	absolutePath := filepath.Join(repoPath, relativePath)
	originalContent := "line1\nline2\n"
	updatedContent := "line1\nline2 changed\n"

	writeTestFile(t, absolutePath, []byte(originalContent))
	runGitCommand(t, repoPath, "add", relativePath)
	runGitCommand(t, repoPath, "commit", "-m", "init tracked file")
	writeTestFile(t, absolutePath, []byte(updatedContent))

	state := NewAppState(nil)
	now := "2026-03-01T00:00:00Z"
	projectID := "proj_discard_abs"
	conversationID := "conv_discard_abs"
	executionID := "exec_discard_abs"
	state.mu.Lock()
	state.projects[projectID] = Project{
		ID:                   projectID,
		WorkspaceID:          localWorkspaceID,
		Name:                 "Discard Absolute Path Project",
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
		Name:          "Discard Absolute Path Conversation",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_1",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.executions[executionID] = Execution{
		ID:             executionID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_discard_abs",
		State:          ExecutionStateCompleted,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     0,
		TraceID:        "tr_discard_abs",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.executionDiffs[executionID] = []DiffItem{{
		ID:         "diff_discard_abs",
		Path:       absolutePath,
		ChangeType: "modified",
		Summary:    "absolute path update",
	}}
	state.conversationExecutionOrder[conversationID] = []string{executionID}
	state.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/v1/executions/"+executionID+"/discard", nil)
	req.SetPathValue("execution_id", executionID)
	req.SetPathValue("action", "discard")
	recorder := httptest.NewRecorder()
	ExecutionActionHandler(state).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", recorder.Code, recorder.Body.String())
	}
	restoredRaw, err := os.ReadFile(absolutePath)
	if err != nil {
		t.Fatalf("read restored file failed: %v", err)
	}
	if string(restoredRaw) != originalContent {
		t.Fatalf("expected file restored to original content, got %q", string(restoredRaw))
	}
	state.mu.RLock()
	execution := state.executions[executionID]
	state.mu.RUnlock()
	if execution.State != ExecutionStateCancelled {
		t.Fatalf("expected execution state cancelled after discard, got %s", execution.State)
	}
}

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

	req := httptest.NewRequest(http.MethodPost, "/v1/conversations/"+conversationID+"/rollback", strings.NewReader(`{"message_id":"msg_rollback_target"}`))
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

func TestExecutionActionHandlerDiscardSkipsGitRestoreWhenRepoIsNotGit(t *testing.T) {
	repoPath := t.TempDir()
	filePath := filepath.Join(repoPath, "src", "main.ts")
	writeTestFile(t, filePath, []byte("console.log('changed')\n"))

	state := NewAppState(nil)
	now := "2026-03-01T00:00:00Z"
	projectID := "proj_discard_non_git_marked_git"
	conversationID := "conv_discard_non_git_marked_git"
	executionID := "exec_discard_non_git_marked_git"
	state.mu.Lock()
	state.projects[projectID] = Project{
		ID:                   projectID,
		WorkspaceID:          localWorkspaceID,
		Name:                 "Non Git Marked Git",
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
		Name:          "Non Git Marked Git Conversation",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "rc_model_1",
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.executions[executionID] = Execution{
		ID:             executionID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_discard_non_git",
		State:          ExecutionStateCompleted,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot:  ModelSnapshot{ModelID: "gpt-5.3"},
		QueueIndex:     0,
		TraceID:        "tr_discard_non_git",
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	state.executionDiffs[executionID] = []DiffItem{{
		ID:         "diff_discard_non_git",
		Path:       "src/main.ts",
		ChangeType: "modified",
		Summary:    "non git update",
	}}
	state.conversationExecutionOrder[conversationID] = []string{executionID}
	state.mu.Unlock()

	req := httptest.NewRequest(http.MethodPost, "/v1/executions/"+executionID+"/discard", nil)
	req.SetPathValue("execution_id", executionID)
	req.SetPathValue("action", "discard")
	recorder := httptest.NewRecorder()
	ExecutionActionHandler(state).ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", recorder.Code, recorder.Body.String())
	}
	state.mu.RLock()
	execution := state.executions[executionID]
	_, diffExists := state.executionDiffs[executionID]
	state.mu.RUnlock()
	if execution.State != ExecutionStateCancelled {
		t.Fatalf("expected execution state cancelled after discard, got %s", execution.State)
	}
	if diffExists {
		t.Fatalf("expected diff entries removed after discard")
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

	req := httptest.NewRequest(http.MethodPost, "/v1/conversations/"+conversationID+"/rollback", strings.NewReader(`{"message_id":"msg_rollback_non_git_target"}`))
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

func initGitRepoForDiffCountTest(t *testing.T, repoPath string) {
	t.Helper()
	runGitCommand(t, repoPath, "init")
	runGitCommand(t, repoPath, "config", "user.email", "test@goyais.dev")
	runGitCommand(t, repoPath, "config", "user.name", "goyais-test")
	if err := os.MkdirAll(filepath.Join(repoPath, "src"), 0o755); err != nil {
		t.Fatalf("mkdir src failed: %v", err)
	}
}

func runGitCommand(t *testing.T, repoPath string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", repoPath}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v (%s)", args, err, string(output))
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
