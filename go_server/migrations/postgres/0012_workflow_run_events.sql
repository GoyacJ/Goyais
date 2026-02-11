CREATE TABLE IF NOT EXISTS workflow_run_events (
  id TEXT PRIMARY KEY,
  run_id TEXT NOT NULL REFERENCES workflow_runs(id),
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  step_key TEXT,
  event_type TEXT NOT NULL,
  payload JSONB NOT NULL DEFAULT '{}'::jsonb,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_workflow_run_events_run_created
  ON workflow_run_events(run_id, created_at ASC, id ASC);

CREATE INDEX IF NOT EXISTS idx_workflow_run_events_tenant_workspace_created
  ON workflow_run_events(tenant_id, workspace_id, created_at DESC, id DESC);
