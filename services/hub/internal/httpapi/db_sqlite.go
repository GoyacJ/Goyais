package httpapi

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	defaultHubDBAppName    = "goyais"
	defaultHubDBFileName   = "hub.sqlite3"
	defaultAccessTokenTTL  = time.Hour
	defaultRefreshTokenTTL = 24 * time.Hour
)

type authzStore struct {
	db     *sql.DB
	dbPath string
}

var legacyDBBackupCopyFile = copyFileContents

func openAuthzStore(path string) (*authzStore, error) {
	if strings.TrimSpace(path) == "" {
		path = resolveHubDBPathFromEnv()
	}

	dsn := path
	if path != ":memory:" {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return nil, fmt.Errorf("resolve db path: %w", err)
		}

		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			return nil, fmt.Errorf("create db dir: %w", err)
		}
		dsn = absPath
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	store := &authzStore{
		db:     db,
		dbPath: dsn,
	}
	if err := store.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *authzStore) close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *authzStore) migrate() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS workspaces (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			mode TEXT NOT NULL,
			hub_url TEXT,
			is_default_local INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			login_disabled INTEGER NOT NULL DEFAULT 0,
			auth_mode TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS workspace_connections (
			workspace_id TEXT PRIMARY KEY,
			hub_url TEXT NOT NULL,
			username TEXT NOT NULL,
			connection_status TEXT NOT NULL,
			connected_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS workspace_agent_configs (
			workspace_id TEXT PRIMARY KEY,
			config_json TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL,
			username TEXT NOT NULL,
			password_hash TEXT NOT NULL,
			display_name TEXT NOT NULL,
			role TEXT NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			UNIQUE(workspace_id, username)
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			access_token TEXT PRIMARY KEY,
			refresh_token TEXT NOT NULL UNIQUE,
			workspace_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			display_name TEXT NOT NULL,
			role TEXT NOT NULL,
			expires_at TEXT NOT NULL,
			refresh_expires_at TEXT NOT NULL,
			revoked INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS roles (
			workspace_id TEXT NOT NULL,
			role_key TEXT NOT NULL,
			name TEXT NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY(workspace_id, role_key)
		)`,
		`CREATE TABLE IF NOT EXISTS role_grants (
			workspace_id TEXT NOT NULL,
			role_key TEXT NOT NULL,
			permission_key TEXT NOT NULL,
			PRIMARY KEY(workspace_id, role_key, permission_key)
		)`,
		`CREATE TABLE IF NOT EXISTS permissions (
			workspace_id TEXT NOT NULL,
			permission_key TEXT NOT NULL,
			label TEXT NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY(workspace_id, permission_key)
		)`,
		`CREATE TABLE IF NOT EXISTS menus (
			workspace_id TEXT NOT NULL,
			menu_key TEXT NOT NULL,
			label TEXT NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY(workspace_id, menu_key)
		)`,
		`CREATE TABLE IF NOT EXISTS permission_visibility (
			workspace_id TEXT NOT NULL,
			role_key TEXT NOT NULL,
			menu_key TEXT NOT NULL,
			visibility TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY(workspace_id, role_key, menu_key)
		)`,
		`CREATE TABLE IF NOT EXISTS abac_policies (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL,
			name TEXT NOT NULL,
			effect TEXT NOT NULL,
			priority INTEGER NOT NULL DEFAULT 100,
			enabled INTEGER NOT NULL DEFAULT 1,
			subject_expr TEXT NOT NULL,
			resource_expr TEXT NOT NULL,
			action_expr TEXT NOT NULL,
			context_expr TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL,
			actor_user_id TEXT,
			action_key TEXT NOT NULL,
			target_type TEXT NOT NULL,
			target_id TEXT NOT NULL,
			result TEXT NOT NULL,
			details_json TEXT NOT NULL,
			trace_id TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS workspace_catalog_roots (
			workspace_id TEXT PRIMARY KEY,
			catalog_root TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS projects (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL,
			name TEXT NOT NULL,
			repo_path TEXT NOT NULL,
			is_git INTEGER NOT NULL DEFAULT 1,
			default_model_config_id TEXT,
			default_mode TEXT NOT NULL,
			current_revision INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_projects_workspace_created ON projects(workspace_id, created_at)`,
		`CREATE TABLE IF NOT EXISTS project_configs (
			project_id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL,
			model_config_ids_json TEXT NOT NULL,
			default_model_config_id TEXT,
			rule_ids_json TEXT NOT NULL,
			skill_ids_json TEXT NOT NULL,
			mcp_ids_json TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_project_configs_workspace_updated ON project_configs(workspace_id, updated_at DESC)`,
		`CREATE TABLE IF NOT EXISTS conversations (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL,
			project_id TEXT NOT NULL,
			name TEXT NOT NULL,
			queue_state TEXT NOT NULL,
			default_mode TEXT NOT NULL,
			model_config_id TEXT NOT NULL,
			rule_ids_json TEXT NOT NULL,
			skill_ids_json TEXT NOT NULL,
			mcp_ids_json TEXT NOT NULL,
			base_revision INTEGER NOT NULL DEFAULT 0,
			active_execution_id TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_conversations_project_created ON conversations(project_id, created_at)`,
		`CREATE TABLE IF NOT EXISTS conversation_messages (
			id TEXT PRIMARY KEY,
			conversation_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			queue_index INTEGER,
			can_rollback INTEGER,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_conversation_messages_conversation_created ON conversation_messages(conversation_id, created_at)`,
		`CREATE TABLE IF NOT EXISTS conversation_snapshots (
			id TEXT PRIMARY KEY,
			conversation_id TEXT NOT NULL,
			rollback_point_message_id TEXT NOT NULL,
			queue_state TEXT NOT NULL,
			worktree_ref TEXT,
			inspector_state_json TEXT NOT NULL,
			messages_json TEXT NOT NULL,
			execution_ids_json TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_conversation_snapshots_conversation_created ON conversation_snapshots(conversation_id, created_at)`,
		`CREATE TABLE IF NOT EXISTS executions (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL,
			conversation_id TEXT NOT NULL,
			message_id TEXT NOT NULL,
			state TEXT NOT NULL,
			mode TEXT NOT NULL,
			model_id TEXT NOT NULL,
			mode_snapshot TEXT NOT NULL,
			model_snapshot_json TEXT NOT NULL,
			resource_profile_snapshot_json TEXT,
			agent_config_snapshot_json TEXT,
			tokens_in INTEGER NOT NULL DEFAULT 0,
			tokens_out INTEGER NOT NULL DEFAULT 0,
			project_revision_snapshot INTEGER NOT NULL DEFAULT 0,
			queue_index INTEGER NOT NULL,
			trace_id TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_executions_conversation_created ON executions(conversation_id, created_at)`,
		`CREATE TABLE IF NOT EXISTS execution_events (
			event_id TEXT PRIMARY KEY,
			execution_id TEXT NOT NULL,
			conversation_id TEXT NOT NULL,
			trace_id TEXT NOT NULL,
			sequence INTEGER NOT NULL,
			queue_index INTEGER NOT NULL,
			type TEXT NOT NULL,
			timestamp TEXT NOT NULL,
			payload_json TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_execution_events_conversation_sequence ON execution_events(conversation_id, sequence)`,
		`CREATE TABLE IF NOT EXISTS resource_configs (
				id TEXT PRIMARY KEY,
				workspace_id TEXT NOT NULL,
				type TEXT NOT NULL,
				enabled INTEGER NOT NULL DEFAULT 1,
				payload_json TEXT NOT NULL,
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_resource_configs_workspace_type ON resource_configs(workspace_id, type)`,
		`CREATE TABLE IF NOT EXISTS resource_test_logs (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL,
			config_id TEXT NOT NULL,
			test_type TEXT NOT NULL,
			result TEXT NOT NULL,
			latency_ms INTEGER NOT NULL DEFAULT 0,
			error_code TEXT,
			details_json TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_resource_test_logs_workspace_created ON resource_test_logs(workspace_id, created_at DESC)`,
	}

	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			return fmt.Errorf("apply migration: %w", err)
		}
	}
	if validationErr := s.validateStrictSchema(); validationErr != nil {
		backupPath := ""
		if shouldBackupLegacySchema(s.dbPath, validationErr) {
			var backupErr error
			backupPath, backupErr = backupLegacyDBFile(s.dbPath)
			if backupErr != nil {
				return fmt.Errorf("backup legacy db before rebuild: %w", backupErr)
			}
			log.Printf("legacy authz db schema detected (%s); backup created at %s", s.dbPath, backupPath)
		}
		if rebuildErr := s.rebuildSchema(statements); rebuildErr != nil {
			return fmt.Errorf("rebuild schema after validation failure: %w (original: %v)", rebuildErr, validationErr)
		}
		if backupPath != "" {
			log.Printf("legacy authz db schema rebuild succeeded (%s) using backup %s", s.dbPath, backupPath)
		}
	}
	return nil
}

func shouldBackupLegacySchema(dbPath string, validationErr error) bool {
	if validationErr == nil {
		return false
	}
	normalized := strings.TrimSpace(dbPath)
	if normalized == "" || normalized == ":memory:" {
		return false
	}
	return strings.Contains(strings.ToLower(validationErr.Error()), "legacy db schema detected")
}

func backupLegacyDBFile(dbPath string) (string, error) {
	normalized := strings.TrimSpace(dbPath)
	if normalized == "" || normalized == ":memory:" {
		return "", nil
	}
	info, statErr := os.Stat(normalized)
	if statErr != nil {
		return "", fmt.Errorf("stat legacy db: %w", statErr)
	}
	if info.IsDir() {
		return "", fmt.Errorf("legacy db path is a directory: %s", normalized)
	}

	now := time.Now().UTC()
	backupPath := fmt.Sprintf(
		"%s.legacy-%s%09d.bak",
		normalized,
		now.Format("20060102150405"),
		now.Nanosecond(),
	)
	if copyErr := legacyDBBackupCopyFile(normalized, backupPath); copyErr != nil {
		return "", copyErr
	}
	return backupPath, nil
}

func (s *authzStore) rebuildSchema(statements []string) error {
	dropStatements := []string{
		`DROP TABLE IF EXISTS workspace_connections`,
		`DROP TABLE IF EXISTS workspace_agent_configs`,
		`DROP TABLE IF EXISTS sessions`,
		`DROP TABLE IF EXISTS role_grants`,
		`DROP TABLE IF EXISTS permission_visibility`,
		`DROP TABLE IF EXISTS menus`,
		`DROP TABLE IF EXISTS permissions`,
		`DROP TABLE IF EXISTS roles`,
		`DROP TABLE IF EXISTS users`,
		`DROP TABLE IF EXISTS abac_policies`,
		`DROP TABLE IF EXISTS audit_logs`,
		`DROP TABLE IF EXISTS workspace_catalog_roots`,
		`DROP TABLE IF EXISTS project_configs`,
		`DROP TABLE IF EXISTS projects`,
		`DROP TABLE IF EXISTS conversation_snapshots`,
		`DROP TABLE IF EXISTS conversation_messages`,
		`DROP TABLE IF EXISTS execution_events`,
		`DROP TABLE IF EXISTS executions`,
		`DROP TABLE IF EXISTS conversations`,
		`DROP TABLE IF EXISTS execution_control_commands`,
		`DROP TABLE IF EXISTS execution_leases`,
		`DROP TABLE IF EXISTS workers`,
		`DROP TABLE IF EXISTS resource_test_logs`,
		`DROP TABLE IF EXISTS resource_configs`,
		`DROP TABLE IF EXISTS workspaces`,
	}
	for _, statement := range dropStatements {
		if _, err := s.db.Exec(statement); err != nil {
			return err
		}
	}
	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			return err
		}
	}
	return s.validateStrictSchema()
}

func (s *authzStore) validateStrictSchema() error {
	requiredColumns := []struct {
		table  string
		column string
	}{
		{table: "projects", column: "current_revision"},
		{table: "projects", column: "default_model_config_id"},
		{table: "project_configs", column: "model_config_ids_json"},
		{table: "project_configs", column: "default_model_config_id"},
		{table: "executions", column: "agent_config_snapshot_json"},
		{table: "executions", column: "resource_profile_snapshot_json"},
		{table: "executions", column: "tokens_in"},
		{table: "executions", column: "tokens_out"},
		{table: "conversations", column: "model_config_id"},
		{table: "conversations", column: "rule_ids_json"},
		{table: "conversations", column: "skill_ids_json"},
		{table: "conversations", column: "mcp_ids_json"},
	}
	for _, field := range requiredColumns {
		ok, err := tableHasColumn(s.db, field.table, field.column)
		if err != nil {
			return fmt.Errorf("validate schema %s.%s: %w", field.table, field.column, err)
		}
		if !ok {
			return fmt.Errorf("legacy db schema detected: missing required column %s.%s; remove existing hub db and restart", field.table, field.column)
		}
	}

	forbiddenLegacyColumns := []struct {
		table  string
		column string
	}{
		{table: "resource_configs", column: "name"},
	}
	for _, field := range forbiddenLegacyColumns {
		ok, err := tableHasColumn(s.db, field.table, field.column)
		if err != nil {
			return fmt.Errorf("validate schema %s.%s: %w", field.table, field.column, err)
		}
		if ok {
			return fmt.Errorf("legacy db schema detected: unexpected legacy column %s.%s", field.table, field.column)
		}
	}
	forbiddenLegacyTables := []string{
		"execution_control_commands",
		"execution_leases",
		"workers",
	}
	for _, table := range forbiddenLegacyTables {
		exists, err := tableExists(s.db, table)
		if err != nil {
			return fmt.Errorf("validate schema table %s: %w", table, err)
		}
		if exists {
			return fmt.Errorf("legacy db schema detected: unexpected table %s", table)
		}
	}
	return nil
}

func tableHasColumn(db *sql.DB, table string, column string) (bool, error) {
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal sql.NullString
			pk         int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &pk); err != nil {
			return false, err
		}
		if strings.EqualFold(strings.TrimSpace(name), strings.TrimSpace(column)) {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	return false, nil
}

func tableExists(db *sql.DB, table string) (bool, error) {
	row := db.QueryRow(`SELECT 1 FROM sqlite_master WHERE type='table' AND name=? LIMIT 1`, strings.TrimSpace(table))
	var exists int
	err := row.Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func copyFileContents(sourcePath string, targetPath string) error {
	sourceFile, sourceErr := os.Open(sourcePath)
	if sourceErr != nil {
		return fmt.Errorf("open source file: %w", sourceErr)
	}
	defer sourceFile.Close()

	sourceInfo, sourceInfoErr := sourceFile.Stat()
	if sourceInfoErr != nil {
		return fmt.Errorf("read source file metadata: %w", sourceInfoErr)
	}

	targetFile, targetErr := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, sourceInfo.Mode().Perm())
	if targetErr != nil {
		return fmt.Errorf("open target file: %w", targetErr)
	}
	defer targetFile.Close()

	if _, copyErr := io.Copy(targetFile, sourceFile); copyErr != nil {
		return fmt.Errorf("copy file content: %w", copyErr)
	}
	if syncErr := targetFile.Sync(); syncErr != nil {
		return fmt.Errorf("sync target file: %w", syncErr)
	}
	return nil
}

func (s *authzStore) ensureWorkspaceSeeds(workspaceID string) error {
	if strings.TrimSpace(workspaceID) == "" {
		return nil
	}

	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	roles := []AdminRole{
		{Key: RoleViewer, Name: "Viewer", Permissions: []string{"project.read", "conversation.read", "resource.read"}, Enabled: true},
		{Key: RoleDeveloper, Name: "Developer", Permissions: []string{"project.read", "project.write", "project_config.read", "conversation.read", "conversation.write", "execution.control", "resource.read", "resource.write", "resource_config.read", "resource_config.write", "model.test", "mcp.connect", "share.request", "share.revoke", "catalog.update_root"}, Enabled: true},
		{Key: RoleApprover, Name: "Approver", Permissions: []string{"project.read", "project.write", "project_config.read", "conversation.read", "conversation.write", "execution.control", "resource.read", "resource.write", "resource_config.read", "resource_config.write", "resource_config.delete", "model.test", "mcp.connect", "share.request", "share.approve", "share.reject", "share.revoke", "catalog.update_root", "admin.audit.read"}, Enabled: true},
		{Key: RoleAdmin, Name: "Admin", Permissions: []string{"*"}, Enabled: true},
	}
	for _, role := range roles {
		if _, err = tx.Exec(
			`INSERT INTO roles(workspace_id, role_key, name, enabled, created_at, updated_at)
			 VALUES(?,?,?,?,?,?)
			 ON CONFLICT(workspace_id, role_key) DO UPDATE SET name=excluded.name, enabled=excluded.enabled, updated_at=excluded.updated_at`,
			workspaceID,
			string(role.Key),
			role.Name,
			boolToInt(role.Enabled),
			now,
			now,
		); err != nil {
			return err
		}
		if _, err = tx.Exec(`DELETE FROM role_grants WHERE workspace_id=? AND role_key=?`, workspaceID, string(role.Key)); err != nil {
			return err
		}
		for _, permissionKey := range role.Permissions {
			if _, err = tx.Exec(`INSERT OR IGNORE INTO role_grants(workspace_id, role_key, permission_key) VALUES(?,?,?)`, workspaceID, string(role.Key), permissionKey); err != nil {
				return err
			}
		}
	}

	defaultPermissions := []adminPermission{
		{Key: "project.read", Label: "读取项目", Enabled: true},
		{Key: "project.write", Label: "写入项目", Enabled: true},
		{Key: "conversation.read", Label: "读取会话", Enabled: true},
		{Key: "conversation.write", Label: "写入会话", Enabled: true},
		{Key: "execution.control", Label: "执行控制", Enabled: true},
		{Key: "resource.read", Label: "读取资源", Enabled: true},
		{Key: "resource.write", Label: "写入资源", Enabled: true},
		{Key: "resource_config.read", Label: "读取资源配置", Enabled: true},
		{Key: "resource_config.write", Label: "写入资源配置", Enabled: true},
		{Key: "resource_config.delete", Label: "删除资源配置", Enabled: true},
		{Key: "project_config.read", Label: "读取项目配置", Enabled: true},
		{Key: "model.test", Label: "测试模型配置", Enabled: true},
		{Key: "mcp.connect", Label: "连接MCP配置", Enabled: true},
		{Key: "catalog.update_root", Label: "更新模型目录根路径", Enabled: true},
		{Key: "share.request", Label: "发起共享", Enabled: true},
		{Key: "share.approve", Label: "审批共享", Enabled: true},
		{Key: "share.reject", Label: "拒绝共享", Enabled: true},
		{Key: "share.revoke", Label: "撤销共享", Enabled: true},
		{Key: "admin.users.manage", Label: "成员管理", Enabled: true},
		{Key: "admin.roles.manage", Label: "角色管理", Enabled: true},
		{Key: "admin.permissions.manage", Label: "权限管理", Enabled: true},
		{Key: "admin.menus.manage", Label: "菜单管理", Enabled: true},
		{Key: "admin.policies.manage", Label: "策略管理", Enabled: true},
		{Key: "admin.audit.read", Label: "审计读取", Enabled: true},
	}
	for _, permission := range defaultPermissions {
		if _, err = tx.Exec(
			`INSERT INTO permissions(workspace_id, permission_key, label, enabled, created_at, updated_at)
			 VALUES(?,?,?,?,?,?)
			 ON CONFLICT(workspace_id, permission_key) DO UPDATE SET label=excluded.label, updated_at=excluded.updated_at`,
			workspaceID,
			permission.Key,
			permission.Label,
			boolToInt(permission.Enabled),
			now,
			now,
		); err != nil {
			return err
		}
	}

	defaultMenus := defaultMenuConfigs()
	for _, menu := range defaultMenus {
		if _, err = tx.Exec(
			`INSERT INTO menus(workspace_id, menu_key, label, enabled, created_at, updated_at)
			 VALUES(?,?,?,?,?,?)
			 ON CONFLICT(workspace_id, menu_key) DO UPDATE SET label=excluded.label, updated_at=excluded.updated_at`,
			workspaceID,
			menu.Key,
			menu.Label,
			boolToInt(menu.Enabled),
			now,
			now,
		); err != nil {
			return err
		}
	}

	for _, role := range []Role{RoleViewer, RoleDeveloper, RoleApprover, RoleAdmin} {
		visibility := defaultMenuVisibility(role)
		for menuKey, item := range visibility {
			if _, err = tx.Exec(
				`INSERT INTO permission_visibility(workspace_id, role_key, menu_key, visibility, created_at, updated_at)
				 VALUES(?,?,?,?,?,?)
				 ON CONFLICT(workspace_id, role_key, menu_key) DO UPDATE SET visibility=excluded.visibility, updated_at=excluded.updated_at`,
				workspaceID,
				string(role),
				menuKey,
				string(item),
				now,
				now,
			); err != nil {
				return err
			}
		}
	}

	defaultPolicies := defaultABACPolicies(workspaceID)
	for _, policy := range defaultPolicies {
		subject, _ := json.Marshal(policy.SubjectExpr)
		resource, _ := json.Marshal(policy.ResourceExpr)
		action, _ := json.Marshal(policy.ActionExpr)
		contextData, _ := json.Marshal(policy.ContextExpr)
		if _, err = tx.Exec(
			`INSERT OR IGNORE INTO abac_policies(id, workspace_id, name, effect, priority, enabled, subject_expr, resource_expr, action_expr, context_expr, created_at, updated_at)
			 VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
			policy.ID,
			workspaceID,
			policy.Name,
			policy.Effect,
			policy.Priority,
			boolToInt(policy.Enabled),
			string(subject),
			string(resource),
			string(action),
			string(contextData),
			now,
			now,
		); err != nil {
			return err
		}
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	_, err = s.ensureWorkspaceAgentConfig(workspaceID)
	return err
}

func resolveHubDBPathFromEnv() string {
	path := strings.TrimSpace(os.Getenv("HUB_DB_PATH"))
	if path == "" {
		return defaultHubDBPath()
	}
	return path
}

func defaultHubDBPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return defaultHubDBFileName
	}
	configDir = strings.TrimSpace(configDir)
	if configDir == "" {
		return defaultHubDBFileName
	}
	return filepath.Join(configDir, defaultHubDBAppName, defaultHubDBFileName)
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
