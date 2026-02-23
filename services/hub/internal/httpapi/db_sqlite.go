package httpapi

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	defaultHubDBPath       = "data/hub.sqlite3"
	defaultAccessTokenTTL  = time.Hour
	defaultRefreshTokenTTL = 24 * time.Hour
)

type authzStore struct {
	db *sql.DB
}

func openAuthzStore(path string) (*authzStore, error) {
	if strings.TrimSpace(path) == "" {
		path = defaultHubDBPath
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
	}

	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			return fmt.Errorf("apply migration: %w", err)
		}
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
		{Key: RoleDeveloper, Name: "Developer", Permissions: []string{"project.read", "project.write", "conversation.read", "conversation.write", "execution.control", "resource.read", "resource.write", "share.request", "share.revoke", "model_catalog.sync"}, Enabled: true},
		{Key: RoleApprover, Name: "Approver", Permissions: []string{"project.read", "project.write", "conversation.read", "conversation.write", "execution.control", "resource.read", "resource.write", "share.request", "share.approve", "share.reject", "share.revoke", "model_catalog.sync", "admin.audit.read"}, Enabled: true},
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
		{Key: "share.request", Label: "发起共享", Enabled: true},
		{Key: "share.approve", Label: "审批共享", Enabled: true},
		{Key: "share.reject", Label: "拒绝共享", Enabled: true},
		{Key: "share.revoke", Label: "撤销共享", Enabled: true},
		{Key: "model_catalog.sync", Label: "同步模型目录", Enabled: true},
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
		return defaultHubDBPath
	}
	return path
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
