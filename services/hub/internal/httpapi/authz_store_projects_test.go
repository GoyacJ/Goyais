package httpapi

import "testing"

func TestAuthzStoreProjectAndConfigCRUD(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	}()

	project, err := store.upsertProject(Project{
		ID:                   "proj_alpha",
		WorkspaceID:          "ws_local",
		Name:                 "Alpha",
		RepoPath:             "/tmp/alpha",
		IsGit:                true,
		DefaultModelConfigID: "gpt-4.1",
		DefaultMode:          ConversationModeAgent,
		CreatedAt:            nowUTC(),
		UpdatedAt:            nowUTC(),
	})
	if err != nil {
		t.Fatalf("upsert project failed: %v", err)
	}
	if project.ID != "proj_alpha" {
		t.Fatalf("unexpected project id: %q", project.ID)
	}

	items, err := store.listProjects("ws_local")
	if err != nil {
		t.Fatalf("list projects failed: %v", err)
	}
	if len(items) != 1 || items[0].ID != "proj_alpha" {
		t.Fatalf("unexpected projects: %#v", items)
	}

	savedConfig, err := store.upsertProjectConfig("ws_local", ProjectConfig{
		ProjectID:            "proj_alpha",
		ModelConfigIDs:       []string{"rc_model_1", "rc_model_2"},
		DefaultModelConfigID: toStringPtr("rc_model_1"),
		RuleIDs:              []string{"rc_rule_1"},
		SkillIDs:             []string{"rc_skill_1"},
		MCPIDs:               []string{"rc_mcp_1"},
		UpdatedAt:            nowUTC(),
	})
	if err != nil {
		t.Fatalf("upsert project config failed: %v", err)
	}
	if savedConfig.DefaultModelConfigID == nil || *savedConfig.DefaultModelConfigID != "rc_model_1" {
		t.Fatalf("unexpected default model id: %#v", savedConfig.DefaultModelConfigID)
	}

	config, exists, err := store.getProjectConfig("proj_alpha")
	if err != nil {
		t.Fatalf("get project config failed: %v", err)
	}
	if !exists {
		t.Fatalf("expected config exists")
	}
	if len(config.ModelConfigIDs) != 2 || config.ModelConfigIDs[0] != "rc_model_1" {
		t.Fatalf("unexpected config model ids: %#v", config.ModelConfigIDs)
	}

	workspaceItems, err := store.listWorkspaceProjectConfigItems("ws_local")
	if err != nil {
		t.Fatalf("list workspace project config items failed: %v", err)
	}
	if len(workspaceItems) != 1 {
		t.Fatalf("expected 1 workspace project config item, got %d", len(workspaceItems))
	}
	if workspaceItems[0].ProjectName != "Alpha" {
		t.Fatalf("unexpected project name: %q", workspaceItems[0].ProjectName)
	}
	if workspaceItems[0].Config.DefaultModelConfigID == nil || *workspaceItems[0].Config.DefaultModelConfigID != "rc_model_1" {
		t.Fatalf("unexpected workspace default model id: %#v", workspaceItems[0].Config.DefaultModelConfigID)
	}

	if err := store.deleteProject("proj_alpha"); err != nil {
		t.Fatalf("delete project failed: %v", err)
	}
	_, exists, err = store.getProject("proj_alpha")
	if err != nil {
		t.Fatalf("get deleted project failed: %v", err)
	}
	if exists {
		t.Fatalf("expected deleted project missing")
	}
	_, exists, err = store.getProjectConfig("proj_alpha")
	if err != nil {
		t.Fatalf("get deleted project config failed: %v", err)
	}
	if exists {
		t.Fatalf("expected deleted project config missing")
	}
}

func TestAuthzStoreWorkspaceProjectConfigFallbackToProjectDefault(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	}()

	if _, err := store.upsertProject(Project{
		ID:                   "proj_beta",
		WorkspaceID:          "ws_local",
		Name:                 "Beta",
		RepoPath:             "/tmp/beta",
		IsGit:                true,
		DefaultModelConfigID: "gpt-4.1-mini",
		DefaultMode:          ConversationModeAgent,
		CreatedAt:            nowUTC(),
		UpdatedAt:            nowUTC(),
	}); err != nil {
		t.Fatalf("upsert project failed: %v", err)
	}

	items, err := store.listWorkspaceProjectConfigItems("ws_local")
	if err != nil {
		t.Fatalf("list workspace project config items failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if len(items[0].Config.ModelConfigIDs) != 1 || items[0].Config.ModelConfigIDs[0] != "gpt-4.1-mini" {
		t.Fatalf("expected fallback model ids from project default model, got %#v", items[0].Config.ModelConfigIDs)
	}
}

func TestAuthzStoreListProjectsOrdersByNewestFirst(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	}()

	if _, err := store.upsertProject(Project{
		ID:                   "proj_old",
		WorkspaceID:          "ws_local",
		Name:                 "Old",
		RepoPath:             "/tmp/old",
		IsGit:                true,
		DefaultModelConfigID: "gpt-4.1",
		DefaultMode:          ConversationModeAgent,
		CreatedAt:            "2026-02-23T00:00:00Z",
		UpdatedAt:            "2026-02-23T00:00:00Z",
	}); err != nil {
		t.Fatalf("upsert old project failed: %v", err)
	}
	if _, err := store.upsertProject(Project{
		ID:                   "proj_new",
		WorkspaceID:          "ws_local",
		Name:                 "New",
		RepoPath:             "/tmp/new",
		IsGit:                true,
		DefaultModelConfigID: "gpt-4.1",
		DefaultMode:          ConversationModeAgent,
		CreatedAt:            "2026-02-23T00:00:01Z",
		UpdatedAt:            "2026-02-23T00:00:01Z",
	}); err != nil {
		t.Fatalf("upsert new project failed: %v", err)
	}

	items, err := store.listProjects("ws_local")
	if err != nil {
		t.Fatalf("list projects failed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].ID != "proj_new" || items[1].ID != "proj_old" {
		t.Fatalf("expected newest first, got %#v", items)
	}
}
