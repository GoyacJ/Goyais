-- SPDX-License-Identifier: Apache-2.0
-- Copyright (c) 2026 Goya
-- Author: Goya
-- Created: 2026-02-11
-- Version: v1.0.0
-- Description: Add assets/lineage tables and extend ACL schema for share APIs.

ALTER TABLE acl_entries
    ALTER COLUMN id TYPE VARCHAR(64) USING id::text;

ALTER TABLE acl_entries
    ALTER COLUMN id DROP DEFAULT;

ALTER TABLE acl_entries
    ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(64),
    ADD COLUMN IF NOT EXISTS workspace_id VARCHAR(64),
    ADD COLUMN IF NOT EXISTS subject_type VARCHAR(16),
    ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS created_by VARCHAR(64);

UPDATE acl_entries
SET subject_type = COALESCE(subject_type, 'user'),
    created_by = COALESCE(created_by, subject_id)
WHERE subject_type IS NULL OR created_by IS NULL;

CREATE TABLE IF NOT EXISTS assets (
    id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL,
    workspace_id VARCHAR(64) NOT NULL,
    owner_id VARCHAR(64) NOT NULL,
    visibility VARCHAR(16) NOT NULL,
    acl_json JSONB NOT NULL,
    name VARCHAR(256) NOT NULL,
    type VARCHAR(128) NOT NULL,
    mime VARCHAR(256) NOT NULL,
    size BIGINT NOT NULL,
    uri VARCHAR(1024) NOT NULL,
    hash VARCHAR(128) NOT NULL,
    metadata_json JSONB NOT NULL,
    status VARCHAR(32) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_assets_scope_order
    ON assets (tenant_id, workspace_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_assets_owner
    ON assets (tenant_id, workspace_id, owner_id);

CREATE TABLE IF NOT EXISTS asset_lineage (
    id VARCHAR(64) PRIMARY KEY,
    tenant_id VARCHAR(64) NOT NULL,
    workspace_id VARCHAR(64) NOT NULL,
    source_asset_id VARCHAR(64),
    target_asset_id VARCHAR(64) NOT NULL,
    run_id VARCHAR(64),
    step_id VARCHAR(64),
    relation VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_asset_lineage_target
    ON asset_lineage (tenant_id, workspace_id, target_asset_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_asset_lineage_source
    ON asset_lineage (tenant_id, workspace_id, source_asset_id, created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_acl_entries_scope_lookup
    ON acl_entries (tenant_id, workspace_id, resource_type, resource_id, subject_type, subject_id);
