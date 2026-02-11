CREATE TABLE IF NOT EXISTS streaming_assets (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  visibility TEXT NOT NULL DEFAULT 'PRIVATE',
  acl_json JSONB NOT NULL DEFAULT '[]'::jsonb,
  path TEXT NOT NULL,
  protocol TEXT NOT NULL CHECK (protocol IN ('rtsp', 'rtmp', 'srt', 'webrtc', 'hls')),
  source TEXT NOT NULL CHECK (source IN ('push', 'pull')),
  endpoints_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  state_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  status TEXT NOT NULL CHECK (status IN ('offline', 'online', 'recording', 'error')),
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  UNIQUE (tenant_id, workspace_id, path)
);

CREATE INDEX IF NOT EXISTS idx_streaming_assets_tenant_workspace_created
  ON streaming_assets(tenant_id, workspace_id, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS stream_recordings (
  id TEXT PRIMARY KEY,
  stream_id TEXT NOT NULL REFERENCES streaming_assets(id),
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  visibility TEXT NOT NULL DEFAULT 'PRIVATE',
  status TEXT NOT NULL CHECK (status IN ('starting', 'recording', 'stopping', 'succeeded', 'failed', 'canceled')),
  asset_id TEXT REFERENCES assets(id),
  error_code TEXT,
  message_key TEXT,
  started_at TIMESTAMPTZ NOT NULL,
  finished_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_stream_recordings_stream_created
  ON stream_recordings(stream_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_stream_recordings_tenant_workspace_created
  ON stream_recordings(tenant_id, workspace_id, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS asset_lineage (
  id TEXT PRIMARY KEY,
  tenant_id TEXT NOT NULL,
  workspace_id TEXT NOT NULL,
  source_asset_id TEXT REFERENCES assets(id),
  target_asset_id TEXT NOT NULL REFERENCES assets(id),
  run_id TEXT,
  step_id TEXT,
  relation TEXT NOT NULL CHECK (relation IN ('derived_from', 'recorded_from', 'transformed_from')),
  created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_asset_lineage_tenant_workspace_created
  ON asset_lineage(tenant_id, workspace_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_asset_lineage_target
  ON asset_lineage(target_asset_id, created_at DESC, id DESC);
