CREATE TABLE IF NOT EXISTS commands (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  visibility TEXT NOT NULL DEFAULT 'PRIVATE',
  acl_json JSONB NOT NULL DEFAULT '[]'::jsonb,
  command_type TEXT NOT NULL,
  payload JSONB NOT NULL,
  status TEXT NOT NULL,
  result JSONB,
  error_code TEXT,
  message_key TEXT,
  accepted_at TIMESTAMPTZ NOT NULL,
  finished_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_commands_tenant_workspace_created
  ON commands(tenant_id, workspace_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_commands_owner
  ON commands(owner_id);

CREATE TABLE IF NOT EXISTS command_idempotency (
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  idempotency_key TEXT NOT NULL,
  request_hash TEXT NOT NULL,
  command_id TEXT NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (tenant_id, workspace_id, owner_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS idx_command_idempotency_expires
  ON command_idempotency(expires_at);

CREATE TABLE IF NOT EXISTS command_events (
  id TEXT PRIMARY KEY,
  command_id TEXT NOT NULL,
  event_type TEXT NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_command_events_command
  ON command_events(command_id, created_at DESC);

CREATE TABLE IF NOT EXISTS audit_events (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  user_id TEXT NOT NULL,
  trace_id TEXT NOT NULL,
  command_id TEXT,
  event_type TEXT NOT NULL,
  resource_type TEXT NOT NULL,
  resource_id TEXT NOT NULL,
  decision TEXT NOT NULL,
  reason TEXT,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_events_tenant_workspace_created
  ON audit_events(tenant_id, workspace_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_events_command
  ON audit_events(command_id);
