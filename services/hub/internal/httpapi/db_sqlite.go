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
	legacyHubDBPath        = "data/hub.sqlite3"
	defaultAccessTokenTTL  = time.Hour
	defaultRefreshTokenTTL = 24 * time.Hour
)

type authzStore struct {
	db *sql.DB
}

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
		if err := migrateLegacyHubDBPath(absPath); err != nil {
			return nil, fmt.Errorf("migrate legacy db path: %w", err)
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

	store := &authzStore{db: db}
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
			default_model_id TEXT,
			default_mode TEXT NOT NULL,
			current_revision INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_projects_workspace_created ON projects(workspace_id, created_at)`,
		`CREATE TABLE IF NOT EXISTS project_configs (
			project_id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL,
			model_ids_json TEXT NOT NULL,
			default_model_id TEXT,
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
			model_id TEXT NOT NULL,
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
		`CREATE TABLE IF NOT EXISTS execution_control_commands (
			id TEXT PRIMARY KEY,
			execution_id TEXT NOT NULL,
			type TEXT NOT NULL,
			payload_json TEXT NOT NULL,
			seq INTEGER NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_execution_control_commands_execution_seq ON execution_control_commands(execution_id, seq)`,
		`CREATE TABLE IF NOT EXISTS execution_leases (
			execution_id TEXT PRIMARY KEY,
			worker_id TEXT NOT NULL,
			lease_version INTEGER NOT NULL,
			lease_expires_at TEXT NOT NULL,
			run_attempt INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS workers (
			worker_id TEXT PRIMARY KEY,
			capabilities_json TEXT NOT NULL,
			status TEXT NOT NULL,
			last_heartbeat TEXT NOT NULL
		)`,
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
	if err := s.migrateResourceConfigsDropNameColumn(); err != nil {
		return fmt.Errorf("migrate resource_configs schema: %w", err)
	}
	if err := s.migrateProjectsAddCurrentRevision(); err != nil {
		return fmt.Errorf("migrate projects schema: %w", err)
	}
	return nil
}

func (s *authzStore) migrateProjectsAddCurrentRevision() error {
	hasCurrentRevision, err := tableHasColumn(s.db, "projects", "current_revision")
	if err != nil {
		return err
	}
	if hasCurrentRevision {
		return nil
	}
	_, err = s.db.Exec(`ALTER TABLE projects ADD COLUMN current_revision INTEGER NOT NULL DEFAULT 0`)
	return err
}

func (s *authzStore) migrateResourceConfigsDropNameColumn() error {
	hasNameColumn, err := tableHasColumn(s.db, "resource_configs", "name")
	if err != nil {
		return err
	}
	if !hasNameColumn {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.Exec(`DROP INDEX IF EXISTS idx_resource_configs_workspace_type`); err != nil {
		return err
	}
	if _, err = tx.Exec(`ALTER TABLE resource_configs RENAME TO resource_configs_legacy`); err != nil {
		return err
	}
	if _, err = tx.Exec(
		`CREATE TABLE resource_configs (
			id TEXT PRIMARY KEY,
			workspace_id TEXT NOT NULL,
			type TEXT NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 1,
			payload_json TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
	); err != nil {
		return err
	}
	if _, err = tx.Exec(
		`INSERT INTO resource_configs(id, workspace_id, type, enabled, payload_json, created_at, updated_at)
		 SELECT id, workspace_id, type, enabled, payload_json, created_at, updated_at
		 FROM resource_configs_legacy`,
	); err != nil {
		return err
	}
	if _, err = tx.Exec(`DROP TABLE resource_configs_legacy`); err != nil {
		return err
	}
	if _, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_resource_configs_workspace_type ON resource_configs(workspace_id, type)`); err != nil {
		return err
	}

	return tx.Commit()
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

	err = tx.Commit()
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
		return legacyHubDBPath
	}
	configDir = strings.TrimSpace(configDir)
	if configDir == "" {
		return legacyHubDBPath
	}
	return filepath.Join(configDir, defaultHubDBAppName, defaultHubDBFileName)
}

func migrateLegacyHubDBPath(targetAbsPath string) error {
	normalizedTarget := filepath.Clean(strings.TrimSpace(targetAbsPath))
	if normalizedTarget == "" {
		return nil
	}

	legacyAbsPath, err := filepath.Abs(legacyHubDBPath)
	if err != nil {
		return fmt.Errorf("resolve legacy db path: %w", err)
	}
	if filepath.Clean(legacyAbsPath) == normalizedTarget {
		return nil
	}

	if _, err := os.Stat(normalizedTarget); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("check target db path: %w", err)
	}

	if _, err := os.Stat(legacyAbsPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("check legacy db path: %w", err)
	}

	if err := copyFilePreserveMode(legacyAbsPath, normalizedTarget); err != nil {
		return err
	}
	log.Printf("audit: migrated hub sqlite db from %s to %s", legacyAbsPath, normalizedTarget)
	return nil
}

func copyFilePreserveMode(sourcePath string, targetPath string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("open legacy db file: %w", err)
	}
	defer sourceFile.Close()

	stat, err := sourceFile.Stat()
	if err != nil {
		return fmt.Errorf("stat legacy db file: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("create target db directory: %w", err)
	}

	perm := stat.Mode().Perm()
	if perm == 0 {
		perm = 0o600
	}
	targetFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)
	if err != nil {
		return fmt.Errorf("create target db file: %w", err)
	}

	_, copyErr := io.Copy(targetFile, sourceFile)
	syncErr := targetFile.Sync()
	closeErr := targetFile.Close()

	if copyErr != nil {
		_ = os.Remove(targetPath)
		return fmt.Errorf("copy legacy db file: %w", copyErr)
	}
	if syncErr != nil {
		_ = os.Remove(targetPath)
		return fmt.Errorf("sync target db file: %w", syncErr)
	}
	if closeErr != nil {
		_ = os.Remove(targetPath)
		return fmt.Errorf("close target db file: %w", closeErr)
	}
	return nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
