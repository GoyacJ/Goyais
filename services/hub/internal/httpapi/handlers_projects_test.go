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

func runGitCommandForProjectImportTest(t *testing.T, repoPath string, args ...string) {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", filepath.Clean(repoPath)}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v (%s)", args, err, string(output))
	}
}
