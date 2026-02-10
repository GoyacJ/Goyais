CREATE TABLE IF NOT EXISTS workflow_templates (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  visibility TEXT NOT NULL DEFAULT 'PRIVATE',
  acl_json TEXT NOT NULL DEFAULT '[]',
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL CHECK (status IN ('draft', 'published', 'disabled')),
  current_version INTEGER NOT NULL DEFAULT 0,
  graph TEXT NOT NULL DEFAULT '{}',
  schema_inputs TEXT NOT NULL DEFAULT '{}',
  schema_outputs TEXT NOT NULL DEFAULT '{}',
  ui_state TEXT NOT NULL DEFAULT '{}',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_workflow_templates_tenant_workspace_created
  ON workflow_templates(tenant_id, workspace_id, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS workflow_template_versions (
  id TEXT PRIMARY KEY,
  template_id TEXT NOT NULL,
  version INTEGER NOT NULL,
  graph TEXT NOT NULL DEFAULT '{}',
  schema_inputs TEXT NOT NULL DEFAULT '{}',
  schema_outputs TEXT NOT NULL DEFAULT '{}',
  checksum TEXT NOT NULL,
  created_by TEXT NOT NULL,
  created_at TEXT NOT NULL,
  UNIQUE(template_id, version),
  FOREIGN KEY (template_id) REFERENCES workflow_templates(id)
);

CREATE TABLE IF NOT EXISTS workflow_runs (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  visibility TEXT NOT NULL DEFAULT 'PRIVATE',
  acl_json TEXT NOT NULL DEFAULT '[]',
  template_id TEXT NOT NULL,
  template_version INTEGER NOT NULL DEFAULT 0,
  command_id TEXT,
  inputs TEXT NOT NULL DEFAULT '{}',
  outputs TEXT NOT NULL DEFAULT '{}',
  status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'succeeded', 'failed', 'canceled')),
  error_code TEXT,
  message_key TEXT,
  started_at TEXT NOT NULL,
  finished_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (template_id) REFERENCES workflow_templates(id)
);

CREATE INDEX IF NOT EXISTS idx_workflow_runs_tenant_workspace_created
  ON workflow_runs(tenant_id, workspace_id, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS step_runs (
  id TEXT PRIMARY KEY,
  run_id TEXT NOT NULL,
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  visibility TEXT NOT NULL DEFAULT 'PRIVATE',
  step_key TEXT NOT NULL,
  step_type TEXT NOT NULL,
  attempt INTEGER NOT NULL DEFAULT 1,
  input TEXT NOT NULL DEFAULT '{}',
  output TEXT NOT NULL DEFAULT '{}',
  artifacts TEXT NOT NULL DEFAULT '{}',
  log_ref TEXT,
  status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'succeeded', 'failed', 'canceled', 'skipped')),
  error_code TEXT,
  message_key TEXT,
  started_at TEXT NOT NULL,
  finished_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (run_id) REFERENCES workflow_runs(id)
);

CREATE INDEX IF NOT EXISTS idx_step_runs_run_created
  ON step_runs(run_id, created_at DESC, id DESC);
