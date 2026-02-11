-- SPDX-License-Identifier: Apache-2.0
-- Copyright (c) 2026 Goya
-- Author: Goya
-- Created: 2026-02-11
-- Version: v1.0.0
-- Description: Baseline schema for commands, audit events, ACL entries, and policy snapshots.

CREATE TABLE IF NOT EXISTS commands (
    id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL,
    workspace_id VARCHAR(64) NOT NULL,
    owner_id VARCHAR(64) NOT NULL,
    visibility VARCHAR(16) NOT NULL,
    status VARCHAR(32) NOT NULL,
    command_type VARCHAR(128) NOT NULL,
    payload_json JSONB,
    trace_id VARCHAR(128) NOT NULL,
    result_json JSONB,
    error_code VARCHAR(64),
    error_message_key VARCHAR(128),
    accepted_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_commands_scope_order
    ON commands (tenant_id, workspace_id, created_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS acl_entries (
    id BIGSERIAL PRIMARY KEY,
    resource_type VARCHAR(64) NOT NULL,
    resource_id VARCHAR(64) NOT NULL,
    subject_id VARCHAR(64) NOT NULL,
    permissions JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_acl_entries_lookup
    ON acl_entries (resource_type, resource_id, subject_id);

CREATE TABLE IF NOT EXISTS audit_events (
    id BIGSERIAL PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL,
    workspace_id VARCHAR(64) NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    trace_id VARCHAR(128) NOT NULL,
    event_type VARCHAR(64) NOT NULL,
    command_type VARCHAR(128),
    decision VARCHAR(16),
    reason VARCHAR(256),
    payload_json JSONB,
    occurred_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_events_trace
    ON audit_events (trace_id, occurred_at DESC);

CREATE TABLE IF NOT EXISTS policies (
    id BIGSERIAL PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL,
    workspace_id VARCHAR(64) NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    policy_version VARCHAR(64) NOT NULL,
    roles_json JSONB NOT NULL,
    denied_command_types_json JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT uk_policies_scope UNIQUE (tenant_id, workspace_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_policies_scope
    ON policies (tenant_id, workspace_id, user_id);
