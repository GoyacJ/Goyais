-- Runtime registry (one runtime endpoint per workspace)
CREATE TABLE IF NOT EXISTS workspace_runtimes (
  workspace_id TEXT PRIMARY KEY REFERENCES workspaces(workspace_id) ON DELETE CASCADE,
  runtime_base_url TEXT NOT NULL,
  runtime_status TEXT NOT NULL CHECK(runtime_status IN ('online','offline')),
  last_heartbeat_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

-- Optional run index (authoritative events remain in runtime)
CREATE TABLE IF NOT EXISTS run_index (
  run_id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(workspace_id) ON DELETE CASCADE,
  created_by TEXT NOT NULL REFERENCES users(user_id),
  status TEXT NOT NULL,
  trace_id TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_run_index_ws ON run_index(workspace_id, created_at DESC);

-- Optional audit summary index for workspace scoped queries
CREATE TABLE IF NOT EXISTS audit_index (
  audit_id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL REFERENCES workspaces(workspace_id) ON DELETE CASCADE,
  run_id TEXT,
  user_id TEXT NOT NULL REFERENCES users(user_id),
  action TEXT NOT NULL,
  tool_name TEXT,
  outcome TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_audit_ws ON audit_index(workspace_id, created_at DESC);

-- Phase 3 permissions
INSERT OR IGNORE INTO permissions(perm_key) VALUES
('run:read'),
('confirm:write');

-- Backfill existing system roles with Phase 3 permissions
INSERT OR IGNORE INTO role_permissions(role_id, perm_key)
SELECT role_id, 'run:read'
FROM roles
WHERE name = 'Owner';

INSERT OR IGNORE INTO role_permissions(role_id, perm_key)
SELECT role_id, 'confirm:write'
FROM roles
WHERE name = 'Owner';

INSERT OR IGNORE INTO role_permissions(role_id, perm_key)
SELECT role_id, 'run:read'
FROM roles
WHERE name = 'Member';

INSERT OR IGNORE INTO role_permissions(role_id, perm_key)
SELECT role_id, 'confirm:write'
FROM roles
WHERE name = 'Member';

-- Member can access replay navigation in Phase 3
INSERT OR IGNORE INTO role_menus(role_id, menu_id)
SELECT role_id, 'nav_replay'
FROM roles
WHERE name = 'Member';
