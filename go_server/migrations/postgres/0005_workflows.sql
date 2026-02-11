CREATE TABLE IF NOT EXISTS workflow_templates (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  visibility TEXT NOT NULL DEFAULT 'PRIVATE',
  acl_json JSONB NOT NULL DEFAULT '[]'::jsonb,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL CHECK (status IN ('draft', 'published', 'disabled')),
  current_version INTEGER NOT NULL DEFAULT 0,
  graph JSONB NOT NULL DEFAULT '{}'::jsonb,
  schema_inputs JSONB NOT NULL DEFAULT '{}'::jsonb,
  schema_outputs JSONB NOT NULL DEFAULT '{}'::jsonb,
  ui_state JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_workflow_templates_tenant_workspace_created
  ON workflow_templates(tenant_id, workspace_id, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS workflow_template_versions (
  id TEXT PRIMARY KEY,
  template_id TEXT NOT NULL REFERENCES workflow_templates(id),
  version INTEGER NOT NULL,
  graph JSONB NOT NULL DEFAULT '{}'::jsonb,
  schema_inputs JSONB NOT NULL DEFAULT '{}'::jsonb,
  schema_outputs JSONB NOT NULL DEFAULT '{}'::jsonb,
  checksum TEXT NOT NULL,
  created_by TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  UNIQUE(template_id, version)
);

CREATE TABLE IF NOT EXISTS workflow_runs (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  visibility TEXT NOT NULL DEFAULT 'PRIVATE',
  acl_json JSONB NOT NULL DEFAULT '[]'::jsonb,
  template_id TEXT NOT NULL REFERENCES workflow_templates(id),
  template_version INTEGER NOT NULL DEFAULT 0,
  command_id TEXT,
  inputs JSONB NOT NULL DEFAULT '{}'::jsonb,
  outputs JSONB NOT NULL DEFAULT '{}'::jsonb,
  status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'succeeded', 'failed', 'canceled')),
  error_code TEXT,
  message_key TEXT,
  started_at TIMESTAMPTZ NOT NULL,
  finished_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_workflow_runs_tenant_workspace_created
  ON workflow_runs(tenant_id, workspace_id, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS step_runs (
  id TEXT PRIMARY KEY,
  run_id TEXT NOT NULL REFERENCES workflow_runs(id),
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  visibility TEXT NOT NULL DEFAULT 'PRIVATE',
  step_key TEXT NOT NULL,
  step_type TEXT NOT NULL,
  attempt INTEGER NOT NULL DEFAULT 1,
  input JSONB NOT NULL DEFAULT '{}'::jsonb,
  output JSONB NOT NULL DEFAULT '{}'::jsonb,
  artifacts JSONB NOT NULL DEFAULT '{}'::jsonb,
  log_ref TEXT,
  status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'succeeded', 'failed', 'canceled', 'skipped')),
  error_code TEXT,
  message_key TEXT,
  started_at TIMESTAMPTZ NOT NULL,
  finished_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_step_runs_run_created
  ON step_runs(run_id, created_at DESC, id DESC);
