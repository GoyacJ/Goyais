CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS domain_sessions (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    name TEXT NOT NULL,
    default_mode TEXT NOT NULL,
    model_config_id TEXT NOT NULL,
    working_dir TEXT NOT NULL,
    additional_directories_json TEXT NOT NULL DEFAULT '[]',
    rule_ids_json TEXT NOT NULL DEFAULT '[]',
    skill_ids_json TEXT NOT NULL DEFAULT '[]',
    mcp_ids_json TEXT NOT NULL DEFAULT '[]',
    next_sequence INTEGER NOT NULL DEFAULT 0,
    active_run_id TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_domain_sessions_workspace_created
    ON domain_sessions(workspace_id, created_at, id);

CREATE TABLE IF NOT EXISTS domain_runs (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    workspace_id TEXT NOT NULL,
    state TEXT NOT NULL,
    input_text TEXT NOT NULL,
    working_dir TEXT NOT NULL,
    additional_directories_json TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_domain_runs_session_created
    ON domain_runs(session_id, created_at, id);

CREATE TABLE IF NOT EXISTS domain_run_events (
    event_id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL,
    session_id TEXT NOT NULL,
    sequence INTEGER NOT NULL,
    type TEXT NOT NULL,
    payload_json TEXT NOT NULL DEFAULT '{}',
    occurred_at TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_domain_run_events_session_sequence
    ON domain_run_events(session_id, sequence);
