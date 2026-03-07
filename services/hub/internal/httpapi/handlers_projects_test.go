package httpapi

import (
	"net/http"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestProjectsHandlerPostDoesNotAssignDefaultModelConfigID(t *testing.T) {
	state := NewAppState(nil)
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/projects", ProjectsHandler(state))

	res := performJSONRequest(t, mux, http.MethodPost, "/v1/projects", map[string]any{
		"workspace_id": localWorkspaceID,
		"name":         "No Default Model",
		"repo_path":    "/tmp/no-default-model",
		"is_git":       false,
	}, nil)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected create project 201, got %d (%s)", res.Code, res.Body.String())
	}

	project := Project{}
	mustDecodeJSON(t, res.Body.Bytes(), &project)
	if project.DefaultModelConfigID != "" {
		t.Fatalf("expected empty default_model_config_id, got %q", project.DefaultModelConfigID)
	}
	config, exists := state.projectConfigs[project.ID]
	if !exists {
		t.Fatalf("expected project config initialized")
	}
	if len(config.ModelConfigIDs) != 0 {
		t.Fatalf("expected no model bindings on project init, got %#v", config.ModelConfigIDs)
	}
	if config.DefaultModelConfigID != nil {
		t.Fatalf("expected nil default_model_config_id on project init, got %#v", *config.DefaultModelConfigID)
	}
}

func TestProjectsImportHandlerDetectsNonGitDirectory(t *testing.T) {
	state := NewAppState(nil)
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/projects/import", ProjectsImportHandler(state))

	projectDir := t.TempDir()
	res := performJSONRequest(t, mux, http.MethodPost, "/v1/projects/import", map[string]any{
		"workspace_id":   localWorkspaceID,
		"directory_path": projectDir,
	}, nil)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected import project 201, got %d (%s)", res.Code, res.Body.String())
	}

	project := Project{}
	mustDecodeJSON(t, res.Body.Bytes(), &project)
	if project.IsGit {
		t.Fatalf("expected non-git directory to be imported with is_git=false, got %#v", project)
	}
}

func TestProjectsImportHandlerDetectsGitDirectory(t *testing.T) {
	state := NewAppState(nil)
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/projects/import", ProjectsImportHandler(state))

	projectDir := t.TempDir()
	runGitCommandForProjectImportTest(t, projectDir, "init")
	runGitCommandForProjectImportTest(t, projectDir, "config", "user.email", "test@goyais.dev")
	runGitCommandForProjectImportTest(t, projectDir, "config", "user.name", "goyais-test")
	runGitCommandForProjectImportTest(t, projectDir, "commit", "--allow-empty", "-m", "init")
	res := performJSONRequest(t, mux, http.MethodPost, "/v1/projects/import", map[string]any{
		"workspace_id":   localWorkspaceID,
		"directory_path": projectDir,
	}, nil)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected import project 201, got %d (%s)", res.Code, res.Body.String())
	}

	project := Project{}
	mustDecodeJSON(t, res.Body.Bytes(), &project)
	if !project.IsGit {
		t.Fatalf("expected git directory to be imported with is_git=true, got %#v", project)
	}
}

func TestProjectsHandlerGetUsesRepositoryTokenUsageWhenExecutionMapMissing(t *testing.T) {
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
	projectID := "proj_list_repo_" + randomHex(4)
	conversationID := "conv_list_repo_" + randomHex(4)
	runID := "run_list_repo_" + randomHex(4)

	project := Project{
		ID:                   projectID,
		WorkspaceID:          localWorkspaceID,
		Name:                 "List Repository Project",
		RepoPath:             "/tmp/list-repository-project",
		IsGit:                true,
		DefaultModelConfigID: "mcfg_list_repo",
		DefaultMode:          PermissionModeDefault,
		CurrentRevision:      0,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if _, err := saveProjectToStore(state, project); err != nil {
		t.Fatalf("save project failed: %v", err)
	}
	if _, err := saveProjectConfigToStore(state, localWorkspaceID, ProjectConfig{
		ProjectID:      projectID,
		ModelConfigIDs: []string{"mcfg_list_repo"},
		UpdatedAt:      now,
	}); err != nil {
		t.Fatalf("save project config failed: %v", err)
	}

	state.mu.Lock()
	state.conversations[conversationID] = Conversation{
		ID:            conversationID,
		WorkspaceID:   localWorkspaceID,
		ProjectID:     projectID,
		Name:          "List Usage Conversation",
		QueueState:    QueueStateIdle,
		DefaultMode:   PermissionModeDefault,
		ModelConfigID: "mcfg_list_repo",
		BaseRevision:  0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	state.conversationExecutionOrder[conversationID] = []string{runID}
	state.executions[runID] = Execution{
		ID:             runID,
		WorkspaceID:    localWorkspaceID,
		ConversationID: conversationID,
		MessageID:      "msg_list_repo",
		State:          RunStateCompleted,
		Mode:           PermissionModeDefault,
		ModelID:        "gpt-5.3",
		ModeSnapshot:   PermissionModeDefault,
		ModelSnapshot: ModelSnapshot{
			ModelID:  "gpt-5.3",
			ConfigID: "mcfg_list_repo",
		},
		TokensIn:  5,
		TokensOut: 8,
		TraceID:   "trace_list_repo",
		CreatedAt: now,
		UpdatedAt: now,
	}
	state.mu.Unlock()

	syncExecutionDomainBestEffort(state)

	state.mu.Lock()
	state.executions = map[string]Execution{}
	state.conversationExecutionOrder = map[string][]string{}
	state.mu.Unlock()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/projects", ProjectsHandler(state))

	res := performJSONRequest(t, mux, http.MethodGet, "/v1/projects?workspace_id="+localWorkspaceID, nil, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("expected list projects 200, got %d (%s)", res.Code, res.Body.String())
	}

	payload := map[string]any{}
	mustDecodeJSON(t, res.Body.Bytes(), &payload)
	items, ok := payload["items"].([]any)
	if !ok || len(items) == 0 {
		t.Fatalf("expected project list items, got %#v", payload["items"])
	}
	projectItem, ok := items[0].(map[string]any)
	if !ok {
		t.Fatalf("expected first project item object, got %#v", items[0])
	}
	if got := int(projectItem["tokens_in_total"].(float64)); got != 5 {
		t.Fatalf("expected tokens_in_total 5, got %d", got)
	}
	if got := int(projectItem["tokens_out_total"].(float64)); got != 8 {
		t.Fatalf("expected tokens_out_total 8, got %d", got)
	}
	if got := int(projectItem["tokens_total"].(float64)); got != 13 {
		t.Fatalf("expected tokens_total 13, got %d", got)
	}
}

func runGitCommandForProjectImportTest(t *testing.T, repoPath string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", filepath.Clean(repoPath)}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v (%s)", args, err, string(output))
	}
}
