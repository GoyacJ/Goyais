CREATE TABLE IF NOT EXISTS capability_providers (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  visibility TEXT NOT NULL DEFAULT 'PRIVATE',
  acl_json TEXT NOT NULL DEFAULT '[]',
  name TEXT NOT NULL,
  provider_type TEXT NOT NULL,
  endpoint TEXT NOT NULL DEFAULT '',
  metadata_json TEXT NOT NULL DEFAULT '{}',
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_capability_providers_tenant_workspace_created
  ON capability_providers(tenant_id, workspace_id, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS capabilities (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  visibility TEXT NOT NULL DEFAULT 'PRIVATE',
  acl_json TEXT NOT NULL DEFAULT '[]',
  provider_id TEXT,
  name TEXT NOT NULL,
  kind TEXT NOT NULL,
  version TEXT NOT NULL,
  input_schema TEXT NOT NULL DEFAULT '{}',
  output_schema TEXT NOT NULL DEFAULT '{}',
  required_permissions TEXT NOT NULL DEFAULT '[]',
  egress_policy TEXT NOT NULL DEFAULT '{}',
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  FOREIGN KEY (provider_id) REFERENCES capability_providers(id)
);

CREATE INDEX IF NOT EXISTS idx_capabilities_tenant_workspace_created
  ON capabilities(tenant_id, workspace_id, created_at DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_capabilities_provider
  ON capabilities(provider_id);

CREATE TABLE IF NOT EXISTS algorithms (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  visibility TEXT NOT NULL DEFAULT 'PRIVATE',
  acl_json TEXT NOT NULL DEFAULT '[]',
  name TEXT NOT NULL,
  version TEXT NOT NULL,
  template_ref TEXT NOT NULL DEFAULT '',
  defaults_json TEXT NOT NULL DEFAULT '{}',
  constraints_json TEXT NOT NULL DEFAULT '{}',
  dependencies_json TEXT NOT NULL DEFAULT '{}',
  status TEXT NOT NULL DEFAULT 'active',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_algorithms_tenant_workspace_created
  ON algorithms(tenant_id, workspace_id, created_at DESC, id DESC);
