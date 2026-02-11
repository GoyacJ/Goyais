CREATE TABLE IF NOT EXISTS ai_sessions (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  visibility TEXT NOT NULL DEFAULT 'PRIVATE',
  acl_json TEXT NOT NULL DEFAULT '[]',
  title TEXT NOT NULL,
  goal TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL CHECK (status IN ('active', 'archived')),
  inputs TEXT NOT NULL DEFAULT '{}',
  constraints TEXT NOT NULL DEFAULT '{}',
  preferences TEXT NOT NULL DEFAULT '{}',
  archived_at TEXT,
  last_turn_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ai_sessions_tenant_workspace_created
  ON ai_sessions(tenant_id, workspace_id, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS ai_session_turns (
  id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL,
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  visibility TEXT NOT NULL DEFAULT 'PRIVATE',
  role TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'system')),
  content TEXT NOT NULL,
  command_type TEXT,
  command_ids TEXT NOT NULL DEFAULT '[]',
  created_at TEXT NOT NULL,
  FOREIGN KEY (session_id) REFERENCES ai_sessions(id)
);

CREATE INDEX IF NOT EXISTS idx_ai_session_turns_session_created
  ON ai_session_turns(session_id, created_at ASC, id ASC);
