CREATE TABLE IF NOT EXISTS context_bundles (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  visibility TEXT NOT NULL DEFAULT 'PRIVATE',
  acl_json TEXT NOT NULL DEFAULT '[]',
  scope_type TEXT NOT NULL CHECK (scope_type IN ('run', 'session', 'workspace')),
  scope_id TEXT NOT NULL,
  facts TEXT NOT NULL DEFAULT '{}',
  summaries TEXT NOT NULL DEFAULT '{}',
  refs TEXT NOT NULL DEFAULT '{}',
  embeddings_index_refs TEXT NOT NULL DEFAULT '[]',
  timeline TEXT NOT NULL DEFAULT '[]',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_context_bundles_scope_unique
  ON context_bundles(tenant_id, workspace_id, owner_id, scope_type, scope_id);

CREATE INDEX IF NOT EXISTS idx_context_bundles_tenant_workspace_created
  ON context_bundles(tenant_id, workspace_id, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS context_bundle_items (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  bundle_id TEXT NOT NULL,
  item_type TEXT NOT NULL,
  item_id TEXT NOT NULL,
  digest TEXT NOT NULL DEFAULT '',
  weight REAL NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  FOREIGN KEY (bundle_id) REFERENCES context_bundles(id)
);

CREATE INDEX IF NOT EXISTS idx_context_bundle_items_bundle_created
  ON context_bundle_items(bundle_id, created_at DESC, id DESC);
