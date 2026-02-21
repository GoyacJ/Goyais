ALTER TABLE executions ADD COLUMN trace_id TEXT;

UPDATE executions
SET trace_id = lower(hex(randomblob(16)))
WHERE trace_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_executions_trace_id ON executions(trace_id);

ALTER TABLE execution_events ADD COLUMN trace_id TEXT;

UPDATE execution_events
SET trace_id = COALESCE(json_extract(payload_json, '$.trace_id'), lower(hex(randomblob(16))))
WHERE trace_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_execution_events_trace_id ON execution_events(trace_id);

ALTER TABLE audit_logs ADD COLUMN trace_id TEXT;

UPDATE audit_logs
SET trace_id = COALESCE(
  (SELECT e.trace_id FROM executions e WHERE e.execution_id = audit_logs.execution_id),
  lower(hex(randomblob(16)))
)
WHERE trace_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_audit_trace_id ON audit_logs(trace_id);
