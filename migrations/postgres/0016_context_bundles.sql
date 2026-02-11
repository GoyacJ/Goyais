CREATE TABLE IF NOT EXISTS context_bundles (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  visibility TEXT NOT NULL DEFAULT 'PRIVATE',
  acl_json JSONB NOT NULL DEFAULT '[]'::jsonb,
  scope_type TEXT NOT NULL CHECK (scope_type IN ('run', 'session', 'workspace')),
  scope_id TEXT NOT NULL,
  facts JSONB NOT NULL DEFAULT '{}'::jsonb,
  summaries JSONB NOT NULL DEFAULT '{}'::jsonb,
  refs JSONB NOT NULL DEFAULT '{}'::jsonb,
  embeddings_index_refs JSONB NOT NULL DEFAULT '[]'::jsonb,
  timeline JSONB NOT NULL DEFAULT '[]'::jsonb,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_context_bundles_scope_unique
  ON context_bundles(tenant_id, workspace_id, owner_id, scope_type, scope_id);

CREATE INDEX IF NOT EXISTS idx_context_bundles_tenant_workspace_created
  ON context_bundles(tenant_id, workspace_id, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS context_bundle_items (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  bundle_id TEXT NOT NULL REFERENCES context_bundles(id),
  item_type TEXT NOT NULL,
  item_id TEXT NOT NULL,
  digest TEXT NOT NULL DEFAULT '',
  weight DOUBLE PRECISION NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_context_bundle_items_bundle_created
  ON context_bundle_items(bundle_id, created_at DESC, id DESC);
