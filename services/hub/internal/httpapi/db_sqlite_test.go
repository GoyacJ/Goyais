package httpapi

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOpenAuthzStoreRebuildsLegacyResourceConfigsSchema(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "hub.sqlite3")
	legacyDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open legacy sqlite db failed: %v", err)
	}

	if _, err := legacyDB.Exec(`CREATE TABLE resource_configs (
		id TEXT PRIMARY KEY,
		workspace_id TEXT NOT NULL,
		type TEXT NOT NULL,
		name TEXT NOT NULL,
		enabled INTEGER NOT NULL DEFAULT 1,
		payload_json TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("create legacy resource_configs failed: %v", err)
	}
	if _, err := legacyDB.Exec(`CREATE INDEX idx_resource_configs_workspace_type ON resource_configs(workspace_id, type)`); err != nil {
		t.Fatalf("create legacy index failed: %v", err)
	}
	if err := legacyDB.Close(); err != nil {
		t.Fatalf("close legacy sqlite db failed: %v", err)
	}

	store, err := openAuthzStore(dbPath)
	if err != nil {
		t.Fatalf("expected open authz store to rebuild legacy schema, got %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	}()
	ok, hasErr := tableHasColumn(store.db, "conversations", "rule_ids_json")
	if hasErr != nil {
		t.Fatalf("check conversations.rule_ids_json failed: %v", hasErr)
	}
	if !ok {
		t.Fatalf("expected conversations.rule_ids_json to exist after rebuild")
	}
}

func TestOpenAuthzStoreRebuildsLegacyProjectsSchema(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "hub.sqlite3")
	legacyDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open legacy sqlite db failed: %v", err)
	}
	if _, err := legacyDB.Exec(`CREATE TABLE projects (
		id TEXT PRIMARY KEY,
		workspace_id TEXT NOT NULL,
		name TEXT NOT NULL,
		repo_path TEXT NOT NULL,
		is_git INTEGER NOT NULL DEFAULT 1,
		default_model_id TEXT,
		default_mode TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("create legacy projects table failed: %v", err)
	}
	if err := legacyDB.Close(); err != nil {
		t.Fatalf("close legacy sqlite db failed: %v", err)
	}

	store, err := openAuthzStore(dbPath)
	if err != nil {
		t.Fatalf("expected open authz store to rebuild legacy projects schema, got %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	}()
	ok, hasErr := tableHasColumn(store.db, "projects", "current_revision")
	if hasErr != nil {
		t.Fatalf("check projects.current_revision failed: %v", hasErr)
	}
	if !ok {
		t.Fatalf("expected projects.current_revision to exist after rebuild")
	}
}

func TestAuthzStoreCreatesProjectSchema(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	}()

	projectColumns := []string{"id", "workspace_id", "name", "repo_path", "default_model_id", "default_mode", "current_revision", "created_at", "updated_at"}
	for _, column := range projectColumns {
		ok, hasErr := tableHasColumn(store.db, "projects", column)
		if hasErr != nil {
			t.Fatalf("check projects column %s failed: %v", column, hasErr)
		}
		if !ok {
			t.Fatalf("expected projects column %s to exist", column)
		}
	}

	projectConfigColumns := []string{"project_id", "workspace_id", "model_ids_json", "rule_ids_json", "skill_ids_json", "mcp_ids_json", "updated_at"}
	for _, column := range projectConfigColumns {
		ok, hasErr := tableHasColumn(store.db, "project_configs", column)
		if hasErr != nil {
			t.Fatalf("check project_configs column %s failed: %v", column, hasErr)
		}
		if !ok {
			t.Fatalf("expected project_configs column %s to exist", column)
		}
	}
}

func TestAuthzStoreCreatesWorkspaceAgentConfigSchema(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	}()

	workspaceConfigColumns := []string{"workspace_id", "config_json", "updated_at"}
	for _, column := range workspaceConfigColumns {
		ok, hasErr := tableHasColumn(store.db, "workspace_agent_configs", column)
		if hasErr != nil {
			t.Fatalf("check workspace_agent_configs column %s failed: %v", column, hasErr)
		}
		if !ok {
			t.Fatalf("expected workspace_agent_configs column %s to exist", column)
		}
	}

	hasExecutionSnapshotColumn, hasErr := tableHasColumn(store.db, "executions", "agent_config_snapshot_json")
	if hasErr != nil {
		t.Fatalf("check executions agent_config_snapshot_json failed: %v", hasErr)
	}
	if !hasExecutionSnapshotColumn {
		t.Fatalf("expected executions.agent_config_snapshot_json to exist")
	}
	hasTokensInColumn, hasTokensInErr := tableHasColumn(store.db, "executions", "tokens_in")
	if hasTokensInErr != nil {
		t.Fatalf("check executions tokens_in failed: %v", hasTokensInErr)
	}
	if !hasTokensInColumn {
		t.Fatalf("expected executions.tokens_in to exist")
	}
	hasTokensOutColumn, hasTokensOutErr := tableHasColumn(store.db, "executions", "tokens_out")
	if hasTokensOutErr != nil {
		t.Fatalf("check executions tokens_out failed: %v", hasTokensOutErr)
	}
	if !hasTokensOutColumn {
		t.Fatalf("expected executions.tokens_out to exist")
	}

	if err := store.ensureWorkspaceSeeds("ws_agent_schema"); err != nil {
		t.Fatalf("ensure workspace seeds failed: %v", err)
	}
	config, exists, err := store.getWorkspaceAgentConfig("ws_agent_schema")
	if err != nil {
		t.Fatalf("get workspace agent config failed: %v", err)
	}
	if !exists {
		t.Fatalf("expected seeded workspace agent config")
	}
	if config.Execution.MaxModelTurns != defaultWorkspaceAgentMaxModelTurns {
		t.Fatalf("expected default max_model_turns=%d, got %d", defaultWorkspaceAgentMaxModelTurns, config.Execution.MaxModelTurns)
	}
	if !config.Display.ShowProcessTrace {
		t.Fatalf("expected show_process_trace default true")
	}
	if config.Display.TraceDetailLevel != WorkspaceAgentTraceDetailLevelVerbose {
		t.Fatalf("expected trace_detail_level verbose, got %q", config.Display.TraceDetailLevel)
	}
}

func TestResolveHubDBPathFromEnvUsesUserConfigDirByDefault(t *testing.T) {
	t.Setenv("HUB_DB_PATH", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "xdg-config"))
	t.Setenv("HOME", filepath.Join(t.TempDir(), "home"))

	resolved := resolveHubDBPathFromEnv()
	if strings.HasSuffix(filepath.Clean(resolved), filepath.Clean(filepath.Join("data", "hub.sqlite3"))) {
		t.Fatalf("expected default db path to be decoupled from legacy data path, got %q", resolved)
	}
	expectedSuffix := filepath.Clean(filepath.Join(defaultHubDBAppName, defaultHubDBFileName))
	if !strings.HasSuffix(filepath.Clean(resolved), expectedSuffix) {
		t.Fatalf("expected default db path suffix %q, got %q", expectedSuffix, resolved)
	}
}

func TestOpenAuthzStoreDoesNotAutoMigrateLegacyDBToDefaultPath(t *testing.T) {
	baseDir := t.TempDir()
	originalCWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get current directory failed: %v", err)
	}
	if err := os.Chdir(baseDir); err != nil {
		t.Fatalf("change directory failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalCWD)
	})

	legacyPath := filepath.Join("data", "hub.sqlite3")
	legacyStore, err := openAuthzStore(legacyPath)
	if err != nil {
		t.Fatalf("open legacy store failed: %v", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	legacyURL := "http://legacy.local"
	if _, err := legacyStore.upsertWorkspace(Workspace{
		ID:             "ws_migrated",
		Name:           "Migrated",
		Mode:           WorkspaceModeRemote,
		HubURL:         &legacyURL,
		IsDefaultLocal: false,
		CreatedAt:      now,
		LoginDisabled:  false,
		AuthMode:       AuthModePasswordOrToken,
	}); err != nil {
		_ = legacyStore.close()
		t.Fatalf("seed legacy workspace failed: %v", err)
	}
	if err := legacyStore.close(); err != nil {
		t.Fatalf("close legacy store failed: %v", err)
	}

	t.Setenv("HUB_DB_PATH", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(baseDir, "config-home"))
	t.Setenv("HOME", filepath.Join(baseDir, "home"))
	targetPath := resolveHubDBPathFromEnv()

	store, err := openAuthzStore("")
	if err != nil {
		t.Fatalf("open default store failed: %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close migrated store failed: %v", closeErr)
		}
	}()

	if _, err := os.Stat(targetPath); err != nil {
		t.Fatalf("expected default db at %q: %v", targetPath, err)
	}
	if _, err := os.Stat(filepath.Join(baseDir, legacyPath)); err != nil {
		t.Fatalf("expected legacy db kept at %q: %v", filepath.Join(baseDir, legacyPath), err)
	}

	workspaces, err := store.listWorkspaces()
	if err != nil {
		t.Fatalf("list workspaces from default store failed: %v", err)
	}
	if len(workspaces) != 0 {
		t.Fatalf("expected default store to remain empty without legacy path migration, got %#v", workspaces)
	}
}
