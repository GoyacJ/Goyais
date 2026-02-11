CREATE TABLE IF NOT EXISTS ai_sessions (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  visibility TEXT NOT NULL DEFAULT 'PRIVATE',
  acl_json JSONB NOT NULL DEFAULT '[]'::jsonb,
  title TEXT NOT NULL,
  goal TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL CHECK (status IN ('active', 'archived')),
  inputs JSONB NOT NULL DEFAULT '{}'::jsonb,
  constraints JSONB NOT NULL DEFAULT '{}'::jsonb,
  preferences JSONB NOT NULL DEFAULT '{}'::jsonb,
  archived_at TIMESTAMPTZ,
  last_turn_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ai_sessions_tenant_workspace_created
  ON ai_sessions(tenant_id, workspace_id, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS ai_session_turns (
  id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL REFERENCES ai_sessions(id),
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  visibility TEXT NOT NULL DEFAULT 'PRIVATE',
  role TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'system')),
  content TEXT NOT NULL,
  command_type TEXT,
  command_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_ai_session_turns_session_created
  ON ai_session_turns(session_id, created_at ASC, id ASC);
