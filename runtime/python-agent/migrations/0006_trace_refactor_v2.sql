ALTER TABLE runs ADD COLUMN trace_id TEXT;

UPDATE runs
SET trace_id = lower(hex(randomblob(16)))
WHERE trace_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_runs_trace_id ON runs(trace_id);

ALTER TABLE events ADD COLUMN trace_id TEXT;

UPDATE events
SET trace_id = COALESCE(json_extract(payload_json, '$.trace_id'), lower(hex(randomblob(16))))
WHERE trace_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_events_trace_id ON events(trace_id);

ALTER TABLE audit_logs ADD COLUMN trace_id TEXT;

UPDATE audit_logs
SET trace_id = COALESCE(
  (SELECT r.trace_id FROM runs r WHERE r.run_id = audit_logs.run_id),
  lower(hex(randomblob(16)))
)
WHERE trace_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_audit_trace_id ON audit_logs(trace_id);
