# Java Server Data Model Draft (v0.1)

## 1. Common Resource Fields

- `id`
- `tenant_id`
- `workspace_id`
- `owner_id`
- `visibility` (`PRIVATE|WORKSPACE|TENANT|PUBLIC`)
- `acl` (jsonb)
- `status`
- `created_at`
- `updated_at`

## 2. Identity and Policy

- `tenants`
- `workspaces`
- `users`
- `roles`
- `user_roles`
- `policies`

## 3. ACL

- `acl_entries(id, tenant_id, workspace_id, resource_type, resource_id, subject_type, subject_id, permissions, expires_at, created_by, created_at)`
- `permissions` 使用 jsonb 存储。

## 4. Command and Audit

- `commands`
- `audit_events`

### 4.1 `commands`（已落地）

- `id` (PK)
- `tenant_id/workspace_id/owner_id`
- `visibility/status/command_type`
- `payload_json/result_json` (jsonb)
- `trace_id`
- `error_code/error_message_key`
- `accepted_at/created_at/updated_at`

### 4.2 `audit_events`（已落地）

- `id` (PK)
- `tenant_id/workspace_id/user_id`
- `trace_id/event_type/command_type`
- `decision/reason`
- `payload_json` (jsonb)
- `occurred_at`

## 5. Dynamic AuthZ Cache Model

- `PolicySnapshot`（内存/Redis 缓存模型）：
  - `tenantId/workspaceId/userId`
  - `policyVersion`
  - `roles`
  - `deniedCommandTypes`
  - `updatedAt`
- `policies`（持久化表，已落地）：
  - `tenant_id/workspace_id/user_id`（唯一键）
  - `policy_version`
  - `roles_json`
  - `denied_command_types_json`
  - `updated_at`
- `PolicyInvalidationEvent`（Redis topic payload）：
  - `tenantId/workspaceId/userId/policyVersion/traceId/changedAt`

## 6. Asset and Lineage

- `assets`（已落地）
  - `id` (PK)
  - `tenant_id/workspace_id/owner_id`
  - `visibility/acl_json/status`
  - `name/type/mime/size/hash/uri`
  - `metadata_json`
  - `created_at/updated_at`
- `asset_lineage`（已落地）
  - `id` (PK)
  - `tenant_id/workspace_id`
  - `source_asset_id/target_asset_id`
  - `run_id/step_id/relation`
  - `created_at`

## 7. Workflow

- `workflow_templates`（已落地）
- `workflow_template_versions`（已落地）
- `workflow_runs`（已落地）
- `step_runs`（已落地）
- `workflow_run_events`（已落地）

## 8. Registry/Plugin/Stream/Context

- `capability_providers`
- `capabilities`
- `algorithms`
- `plugin_packages`
- `plugin_installs`
- `streaming_assets`
- `stream_recordings`
- `context_bundles`

## 9. Data Permission Strategy

- 行级过滤统一由 SQL 层注入：tenant/workspace/visibility/acl。
- 过滤规则必须保留 `policyVersion` 上下文用于审计关联。
- 业务层禁止手写散落式数据权限 if-else。
