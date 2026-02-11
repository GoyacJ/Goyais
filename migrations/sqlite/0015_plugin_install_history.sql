CREATE TABLE IF NOT EXISTS plugin_install_history (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  install_id TEXT NOT NULL,
  from_version TEXT NOT NULL,
  to_version TEXT NOT NULL,
  command_id TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('started', 'succeeded', 'failed', 'rolled_back')),
  error_code TEXT,
  message_key TEXT,
  created_at TEXT NOT NULL,
  FOREIGN KEY (install_id) REFERENCES plugin_installs(id)
);

CREATE INDEX IF NOT EXISTS idx_plugin_install_history_tenant_workspace_created
  ON plugin_install_history(tenant_id, workspace_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_plugin_install_history_install
  ON plugin_install_history(install_id, created_at DESC, id DESC);
