package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	agenthttpapi "goyais/services/hub/internal/agent/adapters/httpapi"
)

func TestOpenAuthzStoreRebuildsLegacyResourceConfigsSchema(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "hub.sqlite3")
	previousDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open previous sqlite db failed: %v", err)
	}

	if _, err := previousDB.Exec(`CREATE TABLE resource_configs (
		id TEXT PRIMARY KEY,
		workspace_id TEXT NOT NULL,
		type TEXT NOT NULL,
		name TEXT NOT NULL,
		enabled INTEGER NOT NULL DEFAULT 1,
		payload_json TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("create previous resource_configs failed: %v", err)
	}
	if _, err := previousDB.Exec(`CREATE INDEX idx_resource_configs_workspace_type ON resource_configs(workspace_id, type)`); err != nil {
		t.Fatalf("create previous index failed: %v", err)
	}
	if err := previousDB.Close(); err != nil {
		t.Fatalf("close previous sqlite db failed: %v", err)
	}

	store, err := openAuthzStore(dbPath)
	if err != nil {
		t.Fatalf("expected open authz store to rebuild previous schema, got %v", err)
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
	previousDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open previous sqlite db failed: %v", err)
	}
	if _, err := previousDB.Exec(`CREATE TABLE projects (
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
		t.Fatalf("create previous projects table failed: %v", err)
	}
	if err := previousDB.Close(); err != nil {
		t.Fatalf("close previous sqlite db failed: %v", err)
	}

	store, err := openAuthzStore(dbPath)
	if err != nil {
		t.Fatalf("expected open authz store to rebuild previous projects schema, got %v", err)
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

func TestOpenAuthzStoreBacksUpLegacyDBBeforeRebuild(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "hub.sqlite3")
	previousDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open previous sqlite db failed: %v", err)
	}
	if _, err := previousDB.Exec(`CREATE TABLE projects (
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
		t.Fatalf("create previous projects table failed: %v", err)
	}
	if err := previousDB.Close(); err != nil {
		t.Fatalf("close previous sqlite db failed: %v", err)
	}

	store, err := openAuthzStore(dbPath)
	if err != nil {
		t.Fatalf("expected open authz store to rebuild previous schema with backup, got %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	}()

	backupPaths, globErr := filepath.Glob(dbPath + ".previous-*.bak")
	if globErr != nil {
		t.Fatalf("glob previous backup files failed: %v", globErr)
	}
	if len(backupPaths) != 1 {
		t.Fatalf("expected exactly one previous backup file, got %d (%v)", len(backupPaths), backupPaths)
	}

	backupDB, backupErr := sql.Open("sqlite", backupPaths[0])
	if backupErr != nil {
		t.Fatalf("open previous backup db failed: %v", backupErr)
	}
	defer backupDB.Close()

	hasRevisionColumnInBackup, hasRevisionColumnErr := tableHasColumn(backupDB, "projects", "current_revision")
	if hasRevisionColumnErr != nil {
		t.Fatalf("check backup projects.current_revision failed: %v", hasRevisionColumnErr)
	}
	if hasRevisionColumnInBackup {
		t.Fatalf("expected backup db to preserve previous schema without projects.current_revision")
	}

	hasRevisionColumnInCurrentDB, currentDBErr := tableHasColumn(store.db, "projects", "current_revision")
	if currentDBErr != nil {
		t.Fatalf("check current projects.current_revision failed: %v", currentDBErr)
	}
	if !hasRevisionColumnInCurrentDB {
		t.Fatalf("expected rebuilt db to contain projects.current_revision")
	}
}

func TestOpenAuthzStoreFailsWhenLegacyBackupFails(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "hub.sqlite3")
	previousDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open previous sqlite db failed: %v", err)
	}
	if _, err := previousDB.Exec(`CREATE TABLE projects (
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
		t.Fatalf("create previous projects table failed: %v", err)
	}
	if err := previousDB.Close(); err != nil {
		t.Fatalf("close previous sqlite db failed: %v", err)
	}

	originalCopyFn := schemaBackupCopyFile
	schemaBackupCopyFile = func(_ string, _ string) error {
		return errors.New("forced backup failure")
	}
	t.Cleanup(func() {
		schemaBackupCopyFile = originalCopyFn
	})

	store, openErr := openAuthzStore(dbPath)
	if openErr == nil {
		if store != nil {
			_ = store.close()
		}
		t.Fatalf("expected open authz store to fail when previous backup fails")
	}
	if !strings.Contains(openErr.Error(), "backup previous-schema db before rebuild") {
		t.Fatalf("expected backup failure context in error, got %v", openErr)
	}
	if !strings.Contains(openErr.Error(), "forced backup failure") {
		t.Fatalf("expected original backup error in message, got %v", openErr)
	}

	backupPaths, globErr := filepath.Glob(dbPath + ".previous-*.bak")
	if globErr != nil {
		t.Fatalf("glob previous backup files failed: %v", globErr)
	}
	if len(backupPaths) != 0 {
		t.Fatalf("expected no previous backup file created on forced failure, got %v", backupPaths)
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

	projectColumns := []string{"id", "workspace_id", "name", "repo_path", "default_model_config_id", "default_mode", "current_revision", "created_at", "updated_at"}
	for _, column := range projectColumns {
		ok, hasErr := tableHasColumn(store.db, "projects", column)
		if hasErr != nil {
			t.Fatalf("check projects column %s failed: %v", column, hasErr)
		}
		if !ok {
			t.Fatalf("expected projects column %s to exist", column)
		}
	}

	projectConfigColumns := []string{
		"project_id",
		"workspace_id",
		"model_config_ids_json",
		"default_model_config_id",
		"token_threshold",
		"model_token_thresholds_json",
		"rule_ids_json",
		"skill_ids_json",
		"mcp_ids_json",
		"updated_at",
	}
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

func TestOpenAuthzStoreAppliesStageZeroMigrations(t *testing.T) {
	store, err := openAuthzStore(":memory:")
	if err != nil {
		t.Fatalf("open authz store failed: %v", err)
	}
	defer func() {
		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed: %v", closeErr)
		}
	}()

	requiredTables := []string{
		"schema_migrations",
		"domain_sessions",
		"domain_runs",
		"domain_run_events",
	}
	for _, table := range requiredTables {
		exists, existsErr := tableExists(store.db, table)
		if existsErr != nil {
			t.Fatalf("check table %s failed: %v", table, existsErr)
		}
		if !exists {
			t.Fatalf("expected %s to exist after openAuthzStore migration", table)
		}
	}
}

func TestNewAppStatePersistsRuntimeSessionsWhenSQLiteRepositoryFlagEnabled(t *testing.T) {
	t.Setenv("FEATURE_SQLITE_REPO", "true")
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
	started, err := state.runtimeService.StartSession(context.Background(), agenthttpapi.StartSessionRequest{
		WorkingDir: "/tmp/persistent-runtime",
	})
	if err != nil {
		t.Fatalf("start runtime session failed: %v", err)
	}
	if _, err := state.runtimeService.Submit(context.Background(), agenthttpapi.SubmitRequest{
		SessionID: started.SessionID,
		Input:     "hello",
	}); err != nil {
		t.Fatalf("submit runtime run failed: %v", err)
	}

	var sessionCount int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM domain_sessions`).Scan(&sessionCount); err != nil {
		t.Fatalf("count domain sessions failed: %v", err)
	}
	if sessionCount != 1 {
		t.Fatalf("expected 1 persisted domain session, got %d", sessionCount)
	}

	var runCount int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM domain_runs`).Scan(&runCount); err != nil {
		t.Fatalf("count domain runs failed: %v", err)
	}
	if runCount != 1 {
		t.Fatalf("expected 1 persisted domain run, got %d", runCount)
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
		t.Fatalf("expected default db path to be decoupled from previous data path, got %q", resolved)
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

	previousPath := filepath.Join("data", "hub.sqlite3")
	previousStore, err := openAuthzStore(previousPath)
	if err != nil {
		t.Fatalf("open previous store failed: %v", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	previousURL := "http://previous.local"
	if _, err := previousStore.upsertWorkspace(Workspace{
		ID:             "ws_migrated",
		Name:           "Migrated",
		Mode:           WorkspaceModeRemote,
		HubURL:         &previousURL,
		IsDefaultLocal: false,
		CreatedAt:      now,
		LoginDisabled:  false,
		AuthMode:       AuthModePasswordOrToken,
	}); err != nil {
		_ = previousStore.close()
		t.Fatalf("seed previous workspace failed: %v", err)
	}
	if err := previousStore.close(); err != nil {
		t.Fatalf("close previous store failed: %v", err)
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
	if _, err := os.Stat(filepath.Join(baseDir, previousPath)); err != nil {
		t.Fatalf("expected previous db kept at %q: %v", filepath.Join(baseDir, previousPath), err)
	}

	workspaces, err := store.listWorkspaces()
	if err != nil {
		t.Fatalf("list workspaces from default store failed: %v", err)
	}
	if len(workspaces) != 0 {
		t.Fatalf("expected default store to remain empty without previous path migration, got %#v", workspaces)
	}
}

func TestOpenAuthzStoreSupportsRuntimeSchemaAfterTwoColdStarts(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "hub-coldstart.sqlite3")
	ctx := context.Background()

	for round := 1; round <= 2; round++ {
		_ = os.Remove(dbPath)
		_ = os.Remove(dbPath + "-wal")
		_ = os.Remove(dbPath + "-shm")

		store, err := openAuthzStore(dbPath)
		if err != nil {
			t.Fatalf("open authz store failed in cold start round %d: %v", round, err)
		}

		repositories := NewSQLiteRuntimeRepositorySet(store.db)
		now := time.Now().UTC().Format(time.RFC3339)
		sessionID := fmt.Sprintf("sess_cold_%d", round)
		runID := fmt.Sprintf("run_cold_%d", round)

		if err := repositories.Sessions.ReplaceAll(ctx, []RuntimeSessionRecord{
			{
				ID:            sessionID,
				WorkspaceID:   localWorkspaceID,
				ProjectID:     fmt.Sprintf("proj_cold_%d", round),
				Name:          fmt.Sprintf("Cold Start Session %d", round),
				DefaultMode:   string(PermissionModeDefault),
				ModelConfigID: "mcfg_cold",
				RuleIDs:       []string{"rule_cold"},
				SkillIDs:      []string{"skill_cold"},
				MCPIDs:        []string{"mcp_cold"},
				ActiveRunID:   &runID,
				CreatedAt:     now,
				UpdatedAt:     now,
			},
		}); err != nil {
			_ = store.close()
			t.Fatalf("replace sessions failed in round %d: %v", round, err)
		}

		if err := repositories.Runs.ReplaceAll(ctx, []RuntimeRunRecord{
			{
				ID:            runID,
				SessionID:     sessionID,
				WorkspaceID:   localWorkspaceID,
				MessageID:     fmt.Sprintf("msg_cold_%d", round),
				State:         string(RunStateExecuting),
				Mode:          string(PermissionModeDefault),
				ModelID:       "gpt-5.3",
				ModelConfigID: "mcfg_cold",
				TokensIn:      13,
				TokensOut:     21,
				TraceID:       fmt.Sprintf("trace_cold_%d", round),
				CreatedAt:     now,
				UpdatedAt:     now,
			},
		}); err != nil {
			_ = store.close()
			t.Fatalf("replace runs failed in round %d: %v", round, err)
		}

		if err := repositories.RunEvents.ReplaceAll(ctx, []RuntimeRunEventRecord{
			{
				EventID:    fmt.Sprintf("evt_cold_%d", round),
				RunID:      runID,
				SessionID:  sessionID,
				Sequence:   1,
				Type:       string(RunEventTypeExecutionStarted),
				Timestamp:  now,
				Payload:    map[string]any{"status": "started"},
				OccurredAt: now,
			},
		}); err != nil {
			_ = store.close()
			t.Fatalf("replace run events failed in round %d: %v", round, err)
		}

		if err := repositories.ChangeSets.ReplaceAll(ctx, []RuntimeChangeSetRecord{
			{
				ChangeSetID: fmt.Sprintf("cs_cold_%d", round),
				SessionID:   sessionID,
				RunID:       &runID,
				Payload:     map[string]any{"files": []any{}},
				CreatedAt:   now,
				UpdatedAt:   now,
			},
		}); err != nil {
			_ = store.close()
			t.Fatalf("replace change sets failed in round %d: %v", round, err)
		}

		if err := repositories.HookRecords.ReplaceAll(ctx, []RuntimeHookRecord{
			{
				ID:        fmt.Sprintf("hook_cold_%d", round),
				RunID:     runID,
				SessionID: sessionID,
				TaskID:    stringPtrOrNil(fmt.Sprintf("task_cold_%d", round)),
				Event:     string(HookEventTypePreToolUse),
				ToolName:  stringPtrOrNil("bash"),
				PolicyID:  stringPtrOrNil(fmt.Sprintf("policy_cold_%d", round)),
				Decision: HookDecision{
					Action: HookDecisionActionAllow,
					Reason: "cold-start-check",
				},
				Timestamp: now,
			},
		}); err != nil {
			_ = store.close()
			t.Fatalf("replace hook records failed in round %d: %v", round, err)
		}

		sessions, err := repositories.Sessions.ListByWorkspace(ctx, localWorkspaceID, RepositoryPage{Limit: 10, Offset: 0})
		if err != nil || len(sessions) != 1 || sessions[0].ID != sessionID {
			_ = store.close()
			t.Fatalf("verify sessions failed in round %d: err=%v payload=%#v", round, err, sessions)
		}
		runs, err := repositories.Runs.ListBySession(ctx, sessionID, RepositoryPage{Limit: 10, Offset: 0})
		if err != nil || len(runs) != 1 || runs[0].ID != runID {
			_ = store.close()
			t.Fatalf("verify runs failed in round %d: err=%v payload=%#v", round, err, runs)
		}
		events, err := repositories.RunEvents.ListBySession(ctx, sessionID, 0, 10)
		if err != nil || len(events) != 1 {
			_ = store.close()
			t.Fatalf("verify events failed in round %d: err=%v payload=%#v", round, err, events)
		}
		changeSets, err := repositories.ChangeSets.ListBySession(ctx, sessionID, RepositoryPage{Limit: 10, Offset: 0})
		if err != nil || len(changeSets) != 1 {
			_ = store.close()
			t.Fatalf("verify change sets failed in round %d: err=%v payload=%#v", round, err, changeSets)
		}
		hookRecords, err := repositories.HookRecords.ListByRun(ctx, runID, RepositoryPage{Limit: 10, Offset: 0})
		if err != nil || len(hookRecords) != 1 {
			_ = store.close()
			t.Fatalf("verify hook records failed in round %d: err=%v payload=%#v", round, err, hookRecords)
		}

		if closeErr := store.close(); closeErr != nil {
			t.Fatalf("close authz store failed in round %d: %v", round, closeErr)
		}
	}
}
