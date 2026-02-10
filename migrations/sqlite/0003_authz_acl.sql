CREATE TABLE IF NOT EXISTS acl_entries (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  resource_type TEXT NOT NULL,
  resource_id TEXT NOT NULL,
  subject_type TEXT NOT NULL,
  subject_id TEXT NOT NULL,
  permissions TEXT NOT NULL DEFAULT '[]',
  expires_at TEXT,
  created_by TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_acl_entries_resource
  ON acl_entries(resource_type, resource_id);
CREATE INDEX IF NOT EXISTS idx_acl_entries_subject
  ON acl_entries(subject_type, subject_id);
CREATE INDEX IF NOT EXISTS idx_acl_entries_tenant_workspace
  ON acl_entries(tenant_id, workspace_id);
