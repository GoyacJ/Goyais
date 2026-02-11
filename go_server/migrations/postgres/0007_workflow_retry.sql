ALTER TABLE workflow_runs ADD COLUMN IF NOT EXISTS attempt INTEGER NOT NULL DEFAULT 1;
ALTER TABLE workflow_runs ADD COLUMN IF NOT EXISTS retry_of_run_id TEXT;
ALTER TABLE workflow_runs ADD COLUMN IF NOT EXISTS replay_from_step_key TEXT;

CREATE INDEX IF NOT EXISTS idx_workflow_runs_retry_of_run_id
  ON workflow_runs(retry_of_run_id);
