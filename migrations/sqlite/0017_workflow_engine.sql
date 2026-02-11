CREATE TABLE IF NOT EXISTS workflow_step_queue (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  run_id TEXT NOT NULL,
  step_key TEXT NOT NULL,
  attempt INTEGER NOT NULL DEFAULT 1,
  status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'leased', 'done', 'canceled')),
  available_at TEXT NOT NULL,
  leased_at TEXT,
  leased_by TEXT,
  payload TEXT NOT NULL DEFAULT '{}',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(run_id, step_key, attempt),
  FOREIGN KEY (run_id) REFERENCES workflow_runs(id)
);

CREATE INDEX IF NOT EXISTS idx_workflow_step_queue_run_available
  ON workflow_step_queue(run_id, status, available_at ASC, id ASC);

CREATE INDEX IF NOT EXISTS idx_workflow_step_queue_scope_available
  ON workflow_step_queue(tenant_id, workspace_id, status, available_at ASC, id ASC);
