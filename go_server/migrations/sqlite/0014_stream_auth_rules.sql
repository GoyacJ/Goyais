CREATE TABLE IF NOT EXISTS stream_auth_rules (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  stream_id TEXT NOT NULL,
  rule TEXT NOT NULL DEFAULT '{}',
  status TEXT NOT NULL CHECK (status IN ('active', 'disabled')),
  updated_by TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE (tenant_id, workspace_id, stream_id),
  FOREIGN KEY (stream_id) REFERENCES streaming_assets(id)
);

CREATE INDEX IF NOT EXISTS idx_stream_auth_rules_tenant_workspace_updated
  ON stream_auth_rules(tenant_id, workspace_id, updated_at DESC, id DESC);
