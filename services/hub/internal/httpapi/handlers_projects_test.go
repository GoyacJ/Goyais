package httpapi

import (
	"net/http"
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
