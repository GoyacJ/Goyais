-- 基础迁移表
CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY,
  applied_at TEXT NOT NULL
);

-- 系统状态：用于 bootstrap 模式判断
CREATE TABLE IF NOT EXISTS system_state (
  singleton_id INTEGER PRIMARY KEY CHECK(singleton_id = 1),
  setup_completed INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
INSERT OR IGNORE INTO system_state(singleton_id, setup_completed, created_at, updated_at)
VALUES (1, 0, strftime('%Y-%m-%dT%H:%M:%fZ','now'), strftime('%Y-%m-%dT%H:%M:%fZ','now'));

-- 用户
CREATE TABLE IF NOT EXISTS users (
  user_id TEXT PRIMARY KEY,
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  display_name TEXT NOT NULL,
  status TEXT NOT NULL CHECK(status IN ('active','disabled')),
  created_at TEXT NOT NULL
);

-- 会话 token（opaque token）
CREATE TABLE IF NOT EXISTS auth_tokens (
  token_id TEXT PRIMARY KEY,
  token_hash TEXT NOT NULL UNIQUE,
  user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  expires_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  last_used_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_auth_tokens_user ON auth_tokens(user_id);

-- 工作区
CREATE TABLE IF NOT EXISTS workspaces (
  workspace_id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  slug TEXT NOT NULL UNIQUE,
  created_by TEXT NOT NULL REFERENCES users(user_id),
  created_at TEXT NOT NULL
);

-- 角色
CREATE TABLE IF NOT EXISTS roles (
  role_id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(workspace_id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  is_system INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  UNIQUE(workspace_id, name)
);

-- 权限点
CREATE TABLE IF NOT EXISTS permissions (
  perm_key TEXT PRIMARY KEY
);

-- 角色-权限映射
CREATE TABLE IF NOT EXISTS role_permissions (
  role_id TEXT NOT NULL REFERENCES roles(role_id) ON DELETE CASCADE,
  perm_key TEXT NOT NULL REFERENCES permissions(perm_key) ON DELETE CASCADE,
  PRIMARY KEY(role_id, perm_key)
);

-- 菜单定义（系统菜单项）
CREATE TABLE IF NOT EXISTS menus (
  menu_id TEXT PRIMARY KEY,
  parent_id TEXT REFERENCES menus(menu_id) ON DELETE CASCADE,
  sort_order INTEGER NOT NULL DEFAULT 0,
  route TEXT,
  icon_key TEXT,
  i18n_key TEXT NOT NULL,
  feature_flag TEXT,
  created_at TEXT NOT NULL
);

-- 角色-菜单映射（可见性）
CREATE TABLE IF NOT EXISTS role_menus (
  role_id TEXT NOT NULL REFERENCES roles(role_id) ON DELETE CASCADE,
  menu_id TEXT NOT NULL REFERENCES menus(menu_id) ON DELETE CASCADE,
  PRIMARY KEY(role_id, menu_id)
);

-- 工作区成员关系
CREATE TABLE IF NOT EXISTS workspace_members (
  workspace_id TEXT NOT NULL REFERENCES workspaces(workspace_id) ON DELETE CASCADE,
  user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  role_id TEXT NOT NULL REFERENCES roles(role_id) ON DELETE RESTRICT,
  status TEXT NOT NULL CHECK(status IN ('active','invited','removed')),
  joined_at TEXT NOT NULL,
  PRIMARY KEY(workspace_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_members_user ON workspace_members(user_id);

-- 预置权限点（P0 最小集）
INSERT OR IGNORE INTO permissions(perm_key) VALUES
('workspace:read'),
('workspace:manage'),
('user:invite'),
('rbac:manage'),
('menu:manage'),
('modelconfig:read'),
('modelconfig:manage'),
('project:read'),
('project:write'),
('run:create'),
('audit:read');

-- 预置菜单（P0 最小集）
INSERT OR IGNORE INTO menus(menu_id,parent_id,sort_order,route,icon_key,i18n_key,feature_flag,created_at) VALUES
('nav_projects',NULL,10,'/projects','folder','nav.projects',NULL,strftime('%Y-%m-%dT%H:%M:%fZ','now')),
('nav_run',NULL,20,'/run','terminal','nav.run',NULL,strftime('%Y-%m-%dT%H:%M:%fZ','now')),
('nav_replay',NULL,30,'/replay','clock','nav.replay',NULL,strftime('%Y-%m-%dT%H:%M:%fZ','now')),
('nav_models',NULL,40,'/models','cpu','nav.models',NULL,strftime('%Y-%m-%dT%H:%M:%fZ','now')),
('nav_settings',NULL,90,'/settings','settings','nav.settings',NULL,strftime('%Y-%m-%dT%H:%M:%fZ','now'));
