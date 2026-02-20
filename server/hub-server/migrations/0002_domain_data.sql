-- Projects（远端）
CREATE TABLE IF NOT EXISTS projects (
  project_id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(workspace_id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  root_uri TEXT NOT NULL,
  created_by TEXT NOT NULL REFERENCES users(user_id),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(workspace_id, name)
);
CREATE INDEX IF NOT EXISTS idx_projects_ws ON projects(workspace_id, created_at DESC);

-- Model Configs（远端：仅配置，不含真实 key）
CREATE TABLE IF NOT EXISTS model_configs (
  model_config_id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(workspace_id) ON DELETE CASCADE,
  provider TEXT NOT NULL,
  model TEXT NOT NULL,
  base_url TEXT,
  temperature REAL NOT NULL DEFAULT 0,
  max_tokens INTEGER,
  secret_ref TEXT NOT NULL,
  created_by TEXT NOT NULL REFERENCES users(user_id),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_model_configs_ws ON model_configs(workspace_id, created_at DESC);

-- Secrets（远端 secret store）
CREATE TABLE IF NOT EXISTS secrets (
  secret_ref TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(workspace_id) ON DELETE CASCADE,
  kind TEXT NOT NULL CHECK(kind IN ('api_key')),
  value_encrypted TEXT NOT NULL,
  created_by TEXT NOT NULL REFERENCES users(user_id),
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_secrets_ws ON secrets(workspace_id);
