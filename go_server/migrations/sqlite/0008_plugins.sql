CREATE TABLE IF NOT EXISTS plugin_packages (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  visibility TEXT NOT NULL DEFAULT 'PRIVATE',
  acl_json TEXT NOT NULL DEFAULT '[]',
  name TEXT NOT NULL,
  version TEXT NOT NULL,
  package_type TEXT NOT NULL,
  manifest_json TEXT NOT NULL DEFAULT '{}',
  artifact_uri TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL CHECK (status IN ('uploaded', 'validating', 'installing', 'enabled', 'disabled', 'failed', 'rolled_back')),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_plugin_packages_tenant_workspace_created
  ON plugin_packages(tenant_id, workspace_id, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS plugin_installs (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  visibility TEXT NOT NULL DEFAULT 'PRIVATE',
  acl_json TEXT NOT NULL DEFAULT '[]',
  package_id TEXT NOT NULL,
  scope TEXT NOT NULL CHECK (scope IN ('workspace', 'tenant')),
  status TEXT NOT NULL CHECK (status IN ('uploaded', 'validating', 'installing', 'enabled', 'disabled', 'failed', 'rolled_back')),
  error_code TEXT,
  message_key TEXT,
  installed_at TEXT,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (package_id) REFERENCES plugin_packages(id)
);

CREATE INDEX IF NOT EXISTS idx_plugin_installs_tenant_workspace_created
  ON plugin_installs(tenant_id, workspace_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_plugin_installs_package
  ON plugin_installs(package_id);
