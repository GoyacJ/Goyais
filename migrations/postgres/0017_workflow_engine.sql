CREATE TABLE IF NOT EXISTS workflow_step_queue (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  run_id TEXT NOT NULL REFERENCES workflow_runs(id),
  step_key TEXT NOT NULL,
  attempt INTEGER NOT NULL DEFAULT 1,
  status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'leased', 'done', 'canceled')),
  available_at TIMESTAMPTZ NOT NULL,
  leased_at TIMESTAMPTZ,
  leased_by TEXT,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  UNIQUE(run_id, step_key, attempt)
);

CREATE INDEX IF NOT EXISTS idx_workflow_step_queue_run_available
  ON workflow_step_queue(run_id, status, available_at ASC, id ASC);

CREATE INDEX IF NOT EXISTS idx_workflow_step_queue_scope_available
  ON workflow_step_queue(tenant_id, workspace_id, status, available_at ASC, id ASC);
