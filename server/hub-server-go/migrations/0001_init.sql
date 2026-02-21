-- +goose Up
-- +goose StatementBegin

-- ============================================================
-- Goyais Hub DB v0.2.0 — 全量初始化
-- ============================================================

-- 系统状态（bootstrap 判断）
CREATE TABLE IF NOT EXISTS system_state (
  singleton_id INTEGER PRIMARY KEY CHECK(singleton_id = 1),
  setup_completed INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
INSERT OR IGNORE INTO system_state(singleton_id, setup_completed, created_at, updated_at)
VALUES (1, 0, datetime('now'), datetime('now'));

-- 用户
CREATE TABLE IF NOT EXISTS users (
  user_id TEXT PRIMARY KEY,
  email TEXT NOT NULL UNIQUE,
  password_hash TEXT NOT NULL,
  display_name TEXT NOT NULL,
  git_name TEXT,
  git_email TEXT,
  status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','disabled')),
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Auth tokens（opaque token → SHA-256 hash 存储）
CREATE TABLE IF NOT EXISTS auth_tokens (
  token_id TEXT PRIMARY KEY,
  token_hash TEXT NOT NULL UNIQUE,
  user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  expires_at TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  last_used_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_auth_tokens_user ON auth_tokens(user_id);

-- 工作区
CREATE TABLE IF NOT EXISTS workspaces (
  workspace_id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  slug TEXT NOT NULL UNIQUE,
  kind TEXT NOT NULL DEFAULT 'local' CHECK(kind IN ('local','remote')),
  created_by TEXT NOT NULL REFERENCES users(user_id),
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 角色
CREATE TABLE IF NOT EXISTS roles (
  role_id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(workspace_id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  is_system INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
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

-- 菜单定义
CREATE TABLE IF NOT EXISTS menus (
  menu_id TEXT PRIMARY KEY,
  parent_id TEXT REFERENCES menus(menu_id) ON DELETE CASCADE,
  sort_order INTEGER NOT NULL DEFAULT 0,
  route TEXT,
  icon_key TEXT,
  i18n_key TEXT NOT NULL,
  feature_flag TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- 角色-菜单可见性
CREATE TABLE IF NOT EXISTS role_menus (
  role_id TEXT NOT NULL REFERENCES roles(role_id) ON DELETE CASCADE,
  menu_id TEXT NOT NULL REFERENCES menus(menu_id) ON DELETE CASCADE,
  PRIMARY KEY(role_id, menu_id)
);

-- 工作区成员
CREATE TABLE IF NOT EXISTS workspace_members (
  workspace_id TEXT NOT NULL REFERENCES workspaces(workspace_id) ON DELETE CASCADE,
  user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  role_id TEXT NOT NULL REFERENCES roles(role_id) ON DELETE RESTRICT,
  status TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','invited','removed')),
  joined_at TEXT NOT NULL DEFAULT (datetime('now')),
  PRIMARY KEY(workspace_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_members_user ON workspace_members(user_id);

-- 项目（v0.2.0：Local = 本地路径，Remote = Git repo URL）
CREATE TABLE IF NOT EXISTS projects (
  project_id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(workspace_id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  root_uri TEXT,                -- 本地 repo 路径（local 模式）
  repo_url TEXT,                -- Git repo URL（remote 模式）
  branch TEXT DEFAULT 'main',
  auth_ref TEXT,                -- secrets 表中的 secret_ref
  repo_cache_path TEXT,         -- 服务端 clone 路径
  sync_status TEXT DEFAULT 'pending' CHECK(sync_status IN ('pending','syncing','ready','error')),
  sync_error TEXT,
  last_synced_at TEXT,
  created_by TEXT NOT NULL REFERENCES users(user_id),
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(workspace_id, name)
);
CREATE INDEX IF NOT EXISTS idx_projects_ws ON projects(workspace_id, created_at DESC);

-- Model Configs（模型配置，不含明文 key）
CREATE TABLE IF NOT EXISTS model_configs (
  config_id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(workspace_id) ON DELETE CASCADE,
  display_name TEXT NOT NULL,
  provider TEXT NOT NULL,
  model TEXT NOT NULL,
  base_url TEXT,
  temperature REAL NOT NULL DEFAULT 0,
  max_tokens INTEGER,
  secret_ref TEXT NOT NULL,
  is_default INTEGER NOT NULL DEFAULT 0,
  created_by TEXT NOT NULL REFERENCES users(user_id),
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_model_configs_ws ON model_configs(workspace_id, created_at DESC);

-- Secrets（AES-256-GCM 加密存储）
CREATE TABLE IF NOT EXISTS secrets (
  secret_ref TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(workspace_id) ON DELETE CASCADE,
  kind TEXT NOT NULL CHECK(kind IN ('api_key','git_credential','generic')),
  value_encrypted TEXT NOT NULL,  -- enc:v1:<base64>
  created_by TEXT NOT NULL REFERENCES users(user_id),
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_secrets_ws ON secrets(workspace_id);

-- ============================================================
-- Sessions（v0.2.0 核心：会话为唯一用户可见概念）
-- ============================================================
CREATE TABLE IF NOT EXISTS sessions (
  session_id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(workspace_id) ON DELETE CASCADE,
  project_id TEXT NOT NULL REFERENCES projects(project_id) ON DELETE CASCADE,
  title TEXT NOT NULL DEFAULT 'New Session',
  mode TEXT NOT NULL DEFAULT 'agent' CHECK(mode IN ('plan','agent')),
  model_config_id TEXT REFERENCES model_configs(config_id),
  skill_set_ids TEXT NOT NULL DEFAULT '[]',      -- JSON array of skill_set_id
  mcp_connector_ids TEXT NOT NULL DEFAULT '[]',  -- JSON array of connector_id
  use_worktree INTEGER NOT NULL DEFAULT 1,
  active_execution_id TEXT,                      -- NULL = idle
  status TEXT NOT NULL DEFAULT 'idle' CHECK(status IN ('idle','executing','waiting_confirmation')),
  created_by TEXT NOT NULL REFERENCES users(user_id),
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  archived_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_sessions_project ON sessions(project_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_sessions_ws ON sessions(workspace_id, created_at DESC);

-- ============================================================
-- Executions（内部概念，不对用户暴露）
-- ============================================================
CREATE TABLE IF NOT EXISTS executions (
  execution_id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL REFERENCES sessions(session_id) ON DELETE CASCADE,
  project_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  created_by TEXT NOT NULL REFERENCES users(user_id),
  state TEXT NOT NULL DEFAULT 'pending'
    CHECK(state IN ('pending','executing','waiting_confirmation','completed','failed','cancelled')),
  trace_id TEXT NOT NULL,
  repo_root TEXT,
  worktree_root TEXT,
  use_worktree INTEGER NOT NULL DEFAULT 1,
  user_message TEXT NOT NULL,
  started_at TEXT,
  ended_at TEXT,
  last_event_ts TEXT,
  token_input INTEGER,
  token_output INTEGER,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_executions_session ON executions(session_id, created_at DESC);

-- Execution Events（Hub 为权威存储）
CREATE TABLE IF NOT EXISTS execution_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  execution_id TEXT NOT NULL REFERENCES executions(execution_id) ON DELETE CASCADE,
  seq INTEGER NOT NULL,
  ts TEXT NOT NULL,
  type TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  UNIQUE(execution_id, seq)
);
CREATE INDEX IF NOT EXISTS idx_events_exec ON execution_events(execution_id, seq);

-- Tool Confirmations（capability prompt 决策）
CREATE TABLE IF NOT EXISTS tool_confirmations (
  confirmation_id TEXT PRIMARY KEY,
  execution_id TEXT NOT NULL REFERENCES executions(execution_id) ON DELETE CASCADE,
  call_id TEXT NOT NULL,
  tool_name TEXT NOT NULL,
  risk_level TEXT NOT NULL DEFAULT 'medium' CHECK(risk_level IN ('low','medium','high','critical')),
  parameters_summary TEXT,
  status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','approved','denied')),
  decided_by TEXT REFERENCES users(user_id),
  decided_at TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(execution_id, call_id)
);

-- Always Allow Scope（用户勾选的自动放行规则）
CREATE TABLE IF NOT EXISTS always_allow_scopes (
  scope_id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(workspace_id) ON DELETE CASCADE,
  user_id TEXT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  tool_name TEXT NOT NULL,
  risk_level TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(workspace_id, user_id, tool_name, risk_level)
);

-- Audit Logs
CREATE TABLE IF NOT EXISTS audit_logs (
  audit_id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  project_id TEXT,
  session_id TEXT,
  execution_id TEXT,
  user_id TEXT NOT NULL,
  action TEXT NOT NULL,
  tool_name TEXT,
  parameters_summary TEXT,
  outcome TEXT NOT NULL CHECK(outcome IN ('success','failure','denied')),
  trace_id TEXT,
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_audit_ws ON audit_logs(workspace_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_exec ON audit_logs(execution_id);

-- ============================================================
-- Skills / MCP
-- ============================================================
CREATE TABLE IF NOT EXISTS skill_sets (
  skill_set_id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(workspace_id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  description TEXT,
  created_by TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(workspace_id, name)
);

CREATE TABLE IF NOT EXISTS skills (
  skill_id TEXT PRIMARY KEY,
  skill_set_id TEXT NOT NULL REFERENCES skill_sets(skill_set_id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  type TEXT NOT NULL CHECK(type IN ('tool_combo','template','custom')),
  config_json TEXT NOT NULL DEFAULT '{}',
  created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS mcp_connectors (
  connector_id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(workspace_id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  transport TEXT NOT NULL CHECK(transport IN ('stdio','sse','streamable_http')),
  endpoint TEXT NOT NULL,
  secret_ref TEXT,
  config_json TEXT NOT NULL DEFAULT '{}',
  enabled INTEGER NOT NULL DEFAULT 1,
  created_by TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(workspace_id, name)
);

-- Worker Registry（Hub 知道哪些 worker 可用）
CREATE TABLE IF NOT EXISTS worker_registry (
  worker_id TEXT PRIMARY KEY,
  workspace_id TEXT,  -- NULL = shared global worker
  base_url TEXT NOT NULL,
  status TEXT NOT NULL DEFAULT 'online' CHECK(status IN ('online','offline')),
  last_heartbeat_at TEXT,
  registered_at TEXT NOT NULL DEFAULT (datetime('now'))
);

-- ============================================================
-- 预置权限点（v0.2.0 全集）
-- ============================================================
INSERT OR IGNORE INTO permissions(perm_key) VALUES
  ('workspace:read'),
  ('workspace:manage'),
  ('user:invite'),
  ('rbac:manage'),
  ('menu:manage'),
  ('project:read'),
  ('project:write'),
  ('session:read'),
  ('session:write'),
  ('execution:read'),
  ('execution:create'),
  ('execution:cancel'),
  ('confirm:write'),
  ('git:read'),
  ('git:write'),
  ('skill:read'),
  ('skill:write'),
  ('mcp:read'),
  ('mcp:write'),
  ('modelconfig:read'),
  ('modelconfig:manage'),
  ('audit:read'),
  ('secret:read'),
  ('secret:write');

-- 预置菜单（v0.2.0）
INSERT OR IGNORE INTO menus(menu_id,parent_id,sort_order,route,icon_key,i18n_key,feature_flag,created_at) VALUES
  ('nav_conversations', NULL, 10, '/conversations', 'message-square', 'nav.conversations', NULL, datetime('now')),
  ('nav_projects',      NULL, 20, '/projects',      'folder',         'nav.projects',      NULL, datetime('now')),
  ('nav_models',        NULL, 40, '/models',        'cpu',            'nav.models',        NULL, datetime('now')),
  ('nav_settings',      NULL, 90, '/settings',      'settings',       'nav.settings',      NULL, datetime('now'));

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS worker_registry;
DROP TABLE IF EXISTS mcp_connectors;
DROP TABLE IF EXISTS skills;
DROP TABLE IF EXISTS skill_sets;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS always_allow_scopes;
DROP TABLE IF EXISTS tool_confirmations;
DROP TABLE IF EXISTS execution_events;
DROP TABLE IF EXISTS executions;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS secrets;
DROP TABLE IF EXISTS model_configs;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS workspace_members;
DROP TABLE IF EXISTS role_menus;
DROP TABLE IF EXISTS menus;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS workspaces;
DROP TABLE IF EXISTS auth_tokens;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS system_state;
-- +goose StatementEnd
