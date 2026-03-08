package httpapi

import (
	"database/sql"
	"testing"
)

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
		DefaultMode:          PermissionModeDefault,
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
		TokenThreshold:       intPointer(1000),
		ModelTokenThresholds: map[string]int{"rc_model_1": 600, "rc_model_unknown": 900},
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
	if savedConfig.TokenThreshold == nil || *savedConfig.TokenThreshold != 1000 {
		t.Fatalf("unexpected token threshold: %#v", savedConfig.TokenThreshold)
	}
	if len(savedConfig.ModelTokenThresholds) != 1 || savedConfig.ModelTokenThresholds["rc_model_1"] != 600 {
		t.Fatalf("unexpected model token thresholds: %#v", savedConfig.ModelTokenThresholds)
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
	if config.TokenThreshold == nil || *config.TokenThreshold != 1000 {
		t.Fatalf("unexpected token threshold from get: %#v", config.TokenThreshold)
	}
	if len(config.ModelTokenThresholds) != 1 || config.ModelTokenThresholds["rc_model_1"] != 600 {
		t.Fatalf("unexpected model token thresholds from get: %#v", config.ModelTokenThresholds)
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
	if workspaceItems[0].Config.TokenThreshold == nil || *workspaceItems[0].Config.TokenThreshold != 1000 {
		t.Fatalf("unexpected workspace token threshold: %#v", workspaceItems[0].Config.TokenThreshold)
	}
	if len(workspaceItems[0].Config.ModelTokenThresholds) != 1 || workspaceItems[0].Config.ModelTokenThresholds["rc_model_1"] != 600 {
		t.Fatalf("unexpected workspace model token thresholds: %#v", workspaceItems[0].Config.ModelTokenThresholds)
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
		DefaultMode:          PermissionModeDefault,
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

func TestAuthzStoreProjectConfigPersistsProjectResourceBindings(t *testing.T) {
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
		ID:                   "proj_bindings",
		WorkspaceID:          "ws_local",
		Name:                 "Bindings",
		RepoPath:             "/tmp/bindings",
		IsGit:                true,
		DefaultModelConfigID: "rc_model_1",
		DefaultMode:          PermissionModeDefault,
		CreatedAt:            nowUTC(),
		UpdatedAt:            nowUTC(),
	}); err != nil {
		t.Fatalf("upsert project failed: %v", err)
	}

	if _, err := store.upsertProjectConfig("ws_local", ProjectConfig{
		ProjectID:            "proj_bindings",
		ModelConfigIDs:       []string{"rc_model_1", "rc_model_2"},
		DefaultModelConfigID: toStringPtr("rc_model_1"),
		RuleIDs:              []string{"rc_rule_1"},
		SkillIDs:             []string{"rc_skill_1"},
		MCPIDs:               []string{"rc_mcp_1"},
		UpdatedAt:            nowUTC(),
	}); err != nil {
		t.Fatalf("upsert project config failed: %v", err)
	}

	bindings, err := listProjectResourceBindingsForTest(store.db, "proj_bindings")
	if err != nil {
		t.Fatalf("list project resource bindings failed: %v", err)
	}
	if len(bindings) != 5 {
		t.Fatalf("expected 5 project resource bindings, got %#v", bindings)
	}

	expected := map[string]projectResourceBindingRow{
		"rc_model_1": {ProjectID: "proj_bindings", ResourceConfigID: "rc_model_1", ResourceType: ResourceTypeModel, BindingIndex: 0, IsDefault: true},
		"rc_model_2": {ProjectID: "proj_bindings", ResourceConfigID: "rc_model_2", ResourceType: ResourceTypeModel, BindingIndex: 1, IsDefault: false},
		"rc_rule_1":  {ProjectID: "proj_bindings", ResourceConfigID: "rc_rule_1", ResourceType: ResourceTypeRule, BindingIndex: 0, IsDefault: false},
		"rc_skill_1": {ProjectID: "proj_bindings", ResourceConfigID: "rc_skill_1", ResourceType: ResourceTypeSkill, BindingIndex: 0, IsDefault: false},
		"rc_mcp_1":   {ProjectID: "proj_bindings", ResourceConfigID: "rc_mcp_1", ResourceType: ResourceTypeMCP, BindingIndex: 0, IsDefault: false},
	}
	for _, binding := range bindings {
		expectedBinding, ok := expected[binding.ResourceConfigID]
		if !ok {
			t.Fatalf("unexpected binding row %#v", binding)
		}
		if binding != expectedBinding {
			t.Fatalf("unexpected binding row %#v", binding)
		}
	}
}

func TestAuthzStoreProjectConfigReadsBindingsAsSourceOfTruth(t *testing.T) {
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
		ID:                   "proj_binding_source",
		WorkspaceID:          "ws_local",
		Name:                 "Binding Source",
		RepoPath:             "/tmp/binding-source",
		IsGit:                true,
		DefaultModelConfigID: "rc_model_1",
		DefaultMode:          PermissionModeDefault,
		CreatedAt:            nowUTC(),
		UpdatedAt:            nowUTC(),
	}); err != nil {
		t.Fatalf("upsert project failed: %v", err)
	}

	if _, err := store.upsertProjectConfig("ws_local", ProjectConfig{
		ProjectID:            "proj_binding_source",
		ModelConfigIDs:       []string{"rc_model_1", "rc_model_2"},
		DefaultModelConfigID: toStringPtr("rc_model_1"),
		TokenThreshold:       intPointer(1200),
		ModelTokenThresholds: map[string]int{"rc_model_1": 700},
		RuleIDs:              []string{"rc_rule_1", "rc_rule_2"},
		SkillIDs:             []string{"rc_skill_1"},
		MCPIDs:               []string{"rc_mcp_1"},
		UpdatedAt:            nowUTC(),
	}); err != nil {
		t.Fatalf("upsert project config failed: %v", err)
	}

	_, err = store.db.Exec(
		`UPDATE project_configs
		 SET model_config_ids_json=?, default_model_config_id=?, rule_ids_json=?, skill_ids_json=?, mcp_ids_json=?
		 WHERE project_id=?`,
		`["rc_model_stale"]`,
		"rc_model_stale",
		`["rc_rule_stale"]`,
		`["rc_skill_stale"]`,
		`["rc_mcp_stale"]`,
		"proj_binding_source",
	)
	if err != nil {
		t.Fatalf("corrupt project_configs json failed: %v", err)
	}

	config, exists, err := store.getProjectConfig("proj_binding_source")
	if err != nil {
		t.Fatalf("get project config failed: %v", err)
	}
	if !exists {
		t.Fatalf("expected config exists")
	}
	assertProjectConfigBindings(t, config, "proj_binding_source")
	if config.TokenThreshold == nil || *config.TokenThreshold != 1200 {
		t.Fatalf("expected token threshold to remain in project_configs row, got %#v", config.TokenThreshold)
	}
	if len(config.ModelTokenThresholds) != 1 || config.ModelTokenThresholds["rc_model_1"] != 700 {
		t.Fatalf("expected model token thresholds preserved, got %#v", config.ModelTokenThresholds)
	}

	items, err := store.listWorkspaceProjectConfigItems("ws_local")
	if err != nil {
		t.Fatalf("list workspace project config items failed: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 workspace project config item, got %#v", items)
	}
	assertProjectConfigBindings(t, items[0].Config, "proj_binding_source")
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
		DefaultMode:          PermissionModeDefault,
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
		DefaultMode:          PermissionModeDefault,
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

func intPointer(input int) *int {
	return &input
}

type projectResourceBindingRow struct {
	ProjectID        string
	ResourceConfigID string
	ResourceType     ResourceType
	BindingIndex     int
	IsDefault        bool
}

func listProjectResourceBindingsForTest(db *sql.DB, projectID string) ([]projectResourceBindingRow, error) {
	rows, err := db.Query(
		`SELECT project_id, resource_config_id, resource_type, binding_index, is_default
		 FROM project_resource_bindings
		 WHERE project_id=?
		 ORDER BY resource_type ASC, binding_index ASC, resource_config_id ASC`,
		projectID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]projectResourceBindingRow, 0)
	for rows.Next() {
		var (
			item         projectResourceBindingRow
			isDefaultInt int
		)
		if err := rows.Scan(
			&item.ProjectID,
			&item.ResourceConfigID,
			&item.ResourceType,
			&item.BindingIndex,
			&isDefaultInt,
		); err != nil {
			return nil, err
		}
		item.IsDefault = parseBoolInt(isDefaultInt)
		items = append(items, item)
	}
	return items, rows.Err()
}

func assertProjectConfigBindings(t *testing.T, config ProjectConfig, projectID string) {
	t.Helper()
	if config.ProjectID != projectID {
		t.Fatalf("expected project id %s, got %s", projectID, config.ProjectID)
	}
	if len(config.ModelConfigIDs) != 2 || config.ModelConfigIDs[0] != "rc_model_1" || config.ModelConfigIDs[1] != "rc_model_2" {
		t.Fatalf("expected model bindings from project_resource_bindings, got %#v", config.ModelConfigIDs)
	}
	if gotDefault := derefString(config.DefaultModelConfigID); gotDefault != "rc_model_1" {
		t.Fatalf("expected default model binding rc_model_1, got %q", gotDefault)
	}
	if len(config.RuleIDs) != 2 || config.RuleIDs[0] != "rc_rule_1" || config.RuleIDs[1] != "rc_rule_2" {
		t.Fatalf("expected rule bindings from project_resource_bindings, got %#v", config.RuleIDs)
	}
	if len(config.SkillIDs) != 1 || config.SkillIDs[0] != "rc_skill_1" {
		t.Fatalf("expected skill bindings from project_resource_bindings, got %#v", config.SkillIDs)
	}
	if len(config.MCPIDs) != 1 || config.MCPIDs[0] != "rc_mcp_1" {
		t.Fatalf("expected mcp bindings from project_resource_bindings, got %#v", config.MCPIDs)
	}
}
