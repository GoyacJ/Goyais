ALTER TABLE workflow_runs ADD COLUMN IF NOT EXISTS trace_id TEXT;
ALTER TABLE step_runs ADD COLUMN IF NOT EXISTS trace_id TEXT;

CREATE INDEX IF NOT EXISTS idx_workflow_runs_trace_id
  ON workflow_runs(trace_id);

CREATE INDEX IF NOT EXISTS idx_step_runs_trace_id
  ON step_runs(trace_id);

CREATE INDEX IF NOT EXISTS idx_audit_events_trace_id
  ON audit_events(trace_id);
