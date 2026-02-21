PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY,
  applied_at TEXT NOT NULL
);

CREATE TABLE projects (
  project_id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  workspace_path TEXT NOT NULL UNIQUE,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE sessions (
  session_id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(project_id) ON DELETE CASCADE,
  title TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE model_configs (
  model_config_id TEXT PRIMARY KEY,
  provider TEXT NOT NULL CHECK(provider IN ('openai', 'anthropic')),
  model TEXT NOT NULL,
  base_url TEXT,
  temperature REAL NOT NULL DEFAULT 0,
  max_tokens INTEGER,
  secret_ref TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE executions (
  execution_id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(project_id) ON DELETE CASCADE,
  session_id TEXT NOT NULL REFERENCES sessions(session_id) ON DELETE CASCADE,
  model_config_id TEXT REFERENCES model_configs(model_config_id),
  input TEXT NOT NULL,
  workspace_path TEXT NOT NULL,
  status TEXT NOT NULL CHECK(status IN ('queued', 'executing', 'waiting_confirmation', 'completed', 'failed', 'cancelled')),
  created_at TEXT NOT NULL,
  started_at TEXT,
  completed_at TEXT
);

CREATE TABLE execution_events (
  global_seq INTEGER PRIMARY KEY AUTOINCREMENT,
  event_id TEXT NOT NULL UNIQUE,
  execution_id TEXT NOT NULL REFERENCES executions(execution_id) ON DELETE CASCADE,
  seq INTEGER NOT NULL,
  ts TEXT NOT NULL,
  type TEXT NOT NULL CHECK(type IN ('plan', 'tool_call', 'tool_result', 'patch', 'error', 'done', 'text_delta', 'heartbeat', 'confirmation_request', 'confirmation_decision', 'cancelled')),
  payload_json TEXT NOT NULL,
  created_at TEXT NOT NULL,
  UNIQUE(execution_id, seq)
);

CREATE TABLE execution_artifacts (
  artifact_id TEXT PRIMARY KEY,
  execution_id TEXT NOT NULL REFERENCES executions(execution_id) ON DELETE CASCADE,
  kind TEXT NOT NULL CHECK(kind IN ('patch', 'text', 'log', 'meta')),
  title TEXT,
  path TEXT,
  content_text TEXT,
  metadata_json TEXT NOT NULL DEFAULT '{}',
  created_at TEXT NOT NULL
);

CREATE TABLE execution_confirmations (
  execution_id TEXT NOT NULL REFERENCES executions(execution_id) ON DELETE CASCADE,
  call_id TEXT NOT NULL,
  status TEXT NOT NULL CHECK(status IN ('pending', 'approved', 'denied')),
  decided_at TEXT,
  decided_by TEXT NOT NULL DEFAULT 'user',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (execution_id, call_id)
);

CREATE TABLE audit_logs (
  audit_id TEXT PRIMARY KEY,
  execution_id TEXT REFERENCES executions(execution_id) ON DELETE SET NULL,
  event_id TEXT REFERENCES execution_events(event_id) ON DELETE SET NULL,
  call_id TEXT,
  action TEXT NOT NULL,
  tool_name TEXT,
  args_json TEXT,
  result_json TEXT,
  requires_confirmation INTEGER NOT NULL DEFAULT 0,
  user_decision TEXT CHECK(user_decision IN ('approve', 'deny', 'n/a')),
  decision_ts TEXT,
  outcome TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE INDEX idx_executions_project_session ON executions(project_id, session_id, created_at DESC);
CREATE INDEX idx_execution_events_execution_seq ON execution_events(execution_id, seq);
CREATE INDEX idx_execution_events_global_seq ON execution_events(global_seq);
CREATE INDEX idx_audit_execution_ts ON audit_logs(execution_id, created_at DESC);
