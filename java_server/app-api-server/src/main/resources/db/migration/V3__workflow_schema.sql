-- SPDX-License-Identifier: Apache-2.0
-- Copyright (c) 2026 Goya
-- Author: Goya
-- Created: 2026-02-11
-- Version: v1.0.0
-- Description: Add workflow templates/runs/steps/events tables for workflow vertical slice.

CREATE TABLE IF NOT EXISTS workflow_templates (
    id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL,
    workspace_id VARCHAR(64) NOT NULL,
    owner_id VARCHAR(64) NOT NULL,
    visibility VARCHAR(16) NOT NULL DEFAULT 'PRIVATE',
    acl_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    name VARCHAR(256) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    status VARCHAR(32) NOT NULL CHECK (status IN ('draft', 'published', 'disabled')),
    current_version INTEGER NOT NULL DEFAULT 0,
    graph JSONB NOT NULL DEFAULT '{}'::jsonb,
    schema_inputs JSONB NOT NULL DEFAULT '{}'::jsonb,
    schema_outputs JSONB NOT NULL DEFAULT '{}'::jsonb,
    ui_state JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_workflow_templates_scope_order
    ON workflow_templates (tenant_id, workspace_id, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS workflow_template_versions (
    id VARCHAR(64) PRIMARY KEY,
    template_id VARCHAR(64) NOT NULL REFERENCES workflow_templates (id),
    version INTEGER NOT NULL,
    graph JSONB NOT NULL DEFAULT '{}'::jsonb,
    schema_inputs JSONB NOT NULL DEFAULT '{}'::jsonb,
    schema_outputs JSONB NOT NULL DEFAULT '{}'::jsonb,
    checksum VARCHAR(128) NOT NULL,
    created_by VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT uk_workflow_template_versions UNIQUE (template_id, version)
);

CREATE TABLE IF NOT EXISTS workflow_runs (
    id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL,
    workspace_id VARCHAR(64) NOT NULL,
    owner_id VARCHAR(64) NOT NULL,
    trace_id VARCHAR(128),
    visibility VARCHAR(16) NOT NULL DEFAULT 'PRIVATE',
    acl_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    template_id VARCHAR(64) NOT NULL REFERENCES workflow_templates (id),
    template_version INTEGER NOT NULL DEFAULT 0,
    attempt INTEGER NOT NULL DEFAULT 1,
    retry_of_run_id VARCHAR(64),
    replay_from_step_key VARCHAR(128),
    command_id VARCHAR(64),
    inputs JSONB NOT NULL DEFAULT '{}'::jsonb,
    outputs JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(32) NOT NULL CHECK (status IN ('pending', 'running', 'succeeded', 'failed', 'canceled')),
    error_code VARCHAR(64),
    message_key VARCHAR(128),
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_workflow_runs_scope_order
    ON workflow_runs (tenant_id, workspace_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_workflow_runs_retry_of_run_id
    ON workflow_runs (retry_of_run_id);

CREATE TABLE IF NOT EXISTS step_runs (
    id VARCHAR(64) PRIMARY KEY,
    run_id VARCHAR(64) NOT NULL REFERENCES workflow_runs (id),
    tenant_id VARCHAR(64) NOT NULL,
    workspace_id VARCHAR(64) NOT NULL,
    owner_id VARCHAR(64) NOT NULL,
    trace_id VARCHAR(128),
    visibility VARCHAR(16) NOT NULL DEFAULT 'PRIVATE',
    step_key VARCHAR(128) NOT NULL,
    step_type VARCHAR(64) NOT NULL,
    attempt INTEGER NOT NULL DEFAULT 1,
    input JSONB NOT NULL DEFAULT '{}'::jsonb,
    output JSONB NOT NULL DEFAULT '{}'::jsonb,
    artifacts JSONB NOT NULL DEFAULT '{}'::jsonb,
    log_ref TEXT,
    status VARCHAR(32) NOT NULL CHECK (status IN ('pending', 'running', 'succeeded', 'failed', 'canceled', 'skipped')),
    error_code VARCHAR(64),
    message_key VARCHAR(128),
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_step_runs_run_created
    ON step_runs (run_id, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS workflow_run_events (
    id VARCHAR(64) PRIMARY KEY,
    run_id VARCHAR(64) NOT NULL REFERENCES workflow_runs (id),
    tenant_id VARCHAR(64) NOT NULL,
    workspace_id VARCHAR(64) NOT NULL,
    step_key VARCHAR(128),
    event_type VARCHAR(128) NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_workflow_run_events_run_created
    ON workflow_run_events (run_id, created_at ASC, id ASC);

CREATE INDEX IF NOT EXISTS idx_workflow_run_events_scope_created
    ON workflow_run_events (tenant_id, workspace_id, created_at DESC, id DESC);
