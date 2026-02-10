CREATE TABLE IF NOT EXISTS algorithm_runs (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  visibility TEXT NOT NULL DEFAULT 'PRIVATE',
  acl_json TEXT NOT NULL DEFAULT '[]',
  algorithm_id TEXT NOT NULL,
  workflow_run_id TEXT NOT NULL,
  command_id TEXT,
  outputs TEXT NOT NULL DEFAULT '{}',
  asset_ids TEXT NOT NULL DEFAULT '[]',
  status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'succeeded', 'failed', 'canceled')),
  error_code TEXT,
  message_key TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (algorithm_id) REFERENCES algorithms(id),
  FOREIGN KEY (workflow_run_id) REFERENCES workflow_runs(id)
);

CREATE INDEX IF NOT EXISTS idx_algorithm_runs_tenant_workspace_created
  ON algorithm_runs(tenant_id, workspace_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_algorithm_runs_algorithm
  ON algorithm_runs(algorithm_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_algorithm_runs_workflow
  ON algorithm_runs(workflow_run_id);
