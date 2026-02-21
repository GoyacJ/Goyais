ALTER TABLE events ADD COLUMN trace_id TEXT;

UPDATE events
SET trace_id = COALESCE(json_extract(payload_json, '$.trace_id'), lower(hex(randomblob(16))))
WHERE trace_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_events_trace_id ON events(trace_id);
