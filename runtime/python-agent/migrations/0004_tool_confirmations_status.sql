PRAGMA foreign_keys = OFF;

ALTER TABLE tool_confirmations RENAME TO tool_confirmations_legacy;

CREATE TABLE tool_confirmations (
  run_id TEXT NOT NULL REFERENCES runs(run_id) ON DELETE CASCADE,
  call_id TEXT NOT NULL,
  status TEXT NOT NULL CHECK(status IN ('pending', 'approved', 'denied')),
  decided_at TEXT,
  decided_by TEXT NOT NULL DEFAULT 'user',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (run_id, call_id)
);

INSERT INTO tool_confirmations(run_id, call_id, status, decided_at, decided_by, created_at, updated_at)
SELECT
  run_id,
  call_id,
  CASE
    WHEN approved = 1 THEN 'approved'
    WHEN approved = 0 THEN 'denied'
    ELSE 'pending'
  END,
  decided_at,
  decided_by,
  COALESCE(decided_at, strftime('%Y-%m-%dT%H:%M:%fZ','now')),
  strftime('%Y-%m-%dT%H:%M:%fZ','now')
FROM tool_confirmations_legacy;

DROP TABLE tool_confirmations_legacy;

PRAGMA foreign_keys = ON;
