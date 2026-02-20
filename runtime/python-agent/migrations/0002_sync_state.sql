CREATE TABLE sync_state (
  singleton_id INTEGER PRIMARY KEY CHECK(singleton_id = 1),
  device_id TEXT NOT NULL,
  last_pushed_global_seq INTEGER NOT NULL DEFAULT 0,
  last_pulled_server_seq INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL
);

INSERT OR IGNORE INTO sync_state (
  singleton_id, device_id, last_pushed_global_seq, last_pulled_server_seq, updated_at
) VALUES (
  1, lower(hex(randomblob(16))), 0, 0, strftime('%Y-%m-%dT%H:%M:%fZ','now')
);

CREATE TABLE synced_event_map (
  event_id TEXT PRIMARY KEY,
  server_seq INTEGER NOT NULL,
  synced_at TEXT NOT NULL
);

CREATE INDEX idx_synced_event_map_server_seq ON synced_event_map(server_seq);
