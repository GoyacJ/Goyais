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

- `acl_entries(resource_type, resource_id, subject_type, subject_id, permissions, expires_at, created_by, created_at)`
- `permissions` 使用 jsonb 存储。

## 4. Command and Audit

- `commands`
- `command_events`
- `audit_events`

## 5. Dynamic AuthZ Cache Model

- `PolicySnapshot`（内存/Redis 缓存模型）：
  - `tenantId/workspaceId/userId`
  - `policyVersion`
  - `roles`
  - `deniedCommandTypes`
  - `updatedAt`
- `PolicyInvalidationEvent`（Redis topic payload）：
  - `tenantId/workspaceId/userId/policyVersion/traceId/changedAt`

## 6. Asset and Lineage

- `assets`
- `asset_lineage`

## 7. Workflow

- `workflow_templates`
- `workflow_template_versions`
- `workflow_runs`
- `step_runs`
- `workflow_step_queue`

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
