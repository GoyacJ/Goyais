CREATE TABLE IF NOT EXISTS assets (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  visibility TEXT NOT NULL DEFAULT 'PRIVATE',
  acl_json TEXT NOT NULL DEFAULT '[]',
  name TEXT,
  type TEXT,
  mime TEXT,
  size INTEGER,
  uri TEXT,
  hash TEXT,
  metadata_json TEXT NOT NULL DEFAULT '{}',
  status TEXT NOT NULL DEFAULT 'ready',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_assets_tenant_workspace_created
  ON assets(tenant_id, workspace_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_assets_id
  ON assets(id);
CREATE INDEX IF NOT EXISTS idx_assets_owner
  ON assets(owner_id);
