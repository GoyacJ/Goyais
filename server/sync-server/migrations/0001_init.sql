CREATE TABLE events (
  server_seq INTEGER PRIMARY KEY AUTOINCREMENT,
  protocol_version TEXT NOT NULL,
  event_id TEXT NOT NULL UNIQUE,
  run_id TEXT NOT NULL,
  run_seq INTEGER NOT NULL,
  ts TEXT NOT NULL,
  type TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  source_device TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE INDEX idx_events_server_seq ON events(server_seq);
CREATE INDEX idx_events_run ON events(run_id, run_seq);
