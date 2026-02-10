# Goyais v0.1 数据模型

## 1. 设计原则

- 全对象统一多租户字段与可见性字段。
- 先保证“关键字段齐全、可扩展”，复杂业务字段在后续版本细化。
- `metadata/config/params/details` 在 v0.1 使用 JSON 对象占位。
- 与 API 契约一致：关键对象必须可映射到 `openapi.yaml`。

## 2. 通用字段规范（适用于核心对象）

| 字段 | 类型（逻辑） | 说明 |
|---|---|---|
| id | string | 全局唯一 ID（UUID/ULID 均可） |
| tenantId | string | 租户 ID |
| workspaceId | string | 工作区 ID |
| ownerId | string | 拥有者用户 ID |
| visibility | enum | `PRIVATE/WORKSPACE/TENANT/PUBLIC` |
| acl | object/array | ACL 快照或引用（详细在 `acl_entries`） |
| status | string | 对象状态（按对象枚举） |
| createdAt | datetime | 创建时间（UTC） |
| updatedAt | datetime | 更新时间（UTC） |

> 约定：物理表使用 `snake_case`（例如 `tenant_id`），API 使用 camelCase（例如 `tenantId`）。

## 3. 核心实体与表结构

## 3.1 身份与权限域

### tenants
- `id`
- `name`
- `status`
- `created_at`
- `updated_at`

### workspaces
- `id`
- `tenant_id`
- `name`
- `status`
- `created_at`
- `updated_at`

### users
- `id`
- `tenant_id`
- `email`
- `display_name`
- `status`
- `created_at`
- `updated_at`

### roles
- `id`
- `tenant_id`
- `workspace_id`（可空，支持租户级角色）
- `name`
- `status`
- `created_at`
- `updated_at`

### user_roles
- `id`
- `tenant_id`
- `workspace_id`
- `user_id`
- `role_id`
- `created_at`

### policies
- `id`
- `tenant_id`
- `workspace_id`
- `version`
- `policy_json`
- `status`
- `created_at`
- `updated_at`

## 3.2 可见性与共享

### acl_entries
- `id`
- `tenant_id`
- `workspace_id`
- `resource_type`（v0.1 已实现：`command`、`asset`）
- `resource_id`
- `subject_type`（v0.1 API 限制为 `user`）
- `subject_id`
- `permissions`（JSON 数组：READ/WRITE/EXECUTE/MANAGE/SHARE，写入时大写/去重/排序）
- `expires_at`（可空）
- `created_by`
- `created_at`

建议索引：
- `(tenant_id, resource_type, resource_id)`
- `(tenant_id, subject_type, subject_id)`
- `(tenant_id, workspace_id)`

权限包含查询口径（冻结）：
- SQLite：`EXISTS (SELECT 1 FROM json_each(a.permissions) p WHERE p.value = 'READ')`
- PostgreSQL：`a.permissions @> '["READ"]'::jsonb`

## 3.3 资产与血缘

### assets
- `id`（TEXT/UUID）
- `tenant_id`（NOT NULL）
- `workspace_id`（NOT NULL）
- `owner_id`（NOT NULL）
- `visibility`（`PRIVATE/WORKSPACE/TENANT/PUBLIC`，NOT NULL，默认 `PRIVATE`）
- `acl_json`（NOT NULL，默认 `[]`，禁止 NULL）
  - SQLite：`TEXT NOT NULL DEFAULT '[]'`
  - PostgreSQL：`JSONB NOT NULL DEFAULT '[]'::jsonb`
- `name`
- `type`（string，本轮不做硬枚举校验）
- `mime`
- `size`
- `uri`（NOT NULL，本轮固定 `local://<relative_path>`）
- `hash`（sha256，NOT NULL）
- `metadata_json`（NOT NULL，默认 `{}`，禁止 NULL）
  - SQLite：`TEXT NOT NULL DEFAULT '{}'`
  - PostgreSQL：`JSONB NOT NULL DEFAULT '{}'::jsonb`
- `status`（本轮最小 `ready/deleted`，默认 `ready`）
- `created_at`（NOT NULL）
- `updated_at`（NOT NULL）

local object store 路径规范（冻结）：
- 不包含原始文件名；
- 路径结构：`tenant/workspace/YYYY/MM/DD/<sha256>`；
- `uri` 示例：`local://tenant_a/ws_1/2026/02/10/84ba...`

### asset_lineage
- `id`
- `tenant_id`
- `workspace_id`
- `source_asset_id`
- `target_asset_id`
- `run_id`
- `step_id`
- `relation`（derived_from/recorded_from/transformed_from）
- `created_at`

## 3.4 工作流与运行

### workflow_templates
- `id, tenant_id, workspace_id, owner_id, visibility`
- `name, description`
- `status`（draft/published/disabled）
- `current_version`
- `graph`（JSON）
- `schema_inputs`（JSON）
- `schema_outputs`（JSON）
- `ui_state`（JSON）
- `created_at, updated_at`

### workflow_template_versions
- `id`
- `template_id`
- `version`
- `graph`（JSON）
- `schema_inputs`（JSON）
- `schema_outputs`（JSON）
- `checksum`
- `created_by`
- `created_at`

### workflow_runs
- `id, tenant_id, workspace_id, owner_id, visibility`
- `template_id, template_version`
- `command_id`
- `inputs`（JSON）
- `outputs`（JSON）
- `status`（pending/running/succeeded/failed/canceled）
- `error_code`
- `message_key`
- `started_at, finished_at`
- `created_at, updated_at`

### step_runs
- `id`
- `run_id`
- `tenant_id, workspace_id, owner_id, visibility`
- `step_key, step_type`
- `attempt`
- `input`（JSON）
- `output`（JSON）
- `artifacts`（JSON）
- `log_ref`
- `status`（pending/running/succeeded/failed/canceled/skipped）
- `error_code`
- `message_key`
- `started_at, finished_at`
- `created_at, updated_at`

建议索引：
- `workflow_runs(tenant_id, workspace_id, status, created_at desc)`
- `step_runs(run_id, step_key, attempt)`

## 3.5 能力注册与插件

### capability_providers
- `id, tenant_id, workspace_id, owner_id, visibility`
- `name, provider_type`（http/container/mcp）
- `endpoint`
- `metadata`（JSON）
- `status`
- `created_at, updated_at`

### capabilities
- `id, tenant_id, workspace_id, owner_id, visibility`
- `provider_id`
- `name, kind, version`
- `input_schema`（JSON）
- `output_schema`（JSON）
- `required_permissions`（JSON）
- `egress_policy`（JSON）
- `status`
- `created_at, updated_at`

### algorithms
- `id, tenant_id, workspace_id, owner_id, visibility`
- `name, version`
- `template_ref`
- `defaults`（JSON）
- `constraints`（JSON）
- `dependencies`（JSON）
- `status`
- `created_at, updated_at`

### plugin_packages
- `id, tenant_id, workspace_id, owner_id, visibility`
- `name, version, package_type`
- `manifest`（JSON）
- `artifact_uri`
- `status`
- `created_at, updated_at`

### plugin_installs
- `id, tenant_id, workspace_id, owner_id, visibility`
- `package_id`
- `scope`（workspace/tenant）
- `status`（uploaded/validating/installing/enabled/disabled/failed/rolled_back）
- `error_code`
- `message_key`
- `installed_at`
- `updated_at`

## 3.6 流媒体

### streaming_assets
- `id, tenant_id, workspace_id, owner_id, visibility`
- `stream_id`
- `path`
- `protocol`（rtsp/rtmp/srt/webrtc/hls）
- `source`（push/pull）
- `endpoints`（JSON）
- `state`（JSON）
- `status`（offline/online/recording/error）
- `created_at, updated_at`

### stream_recordings
- `id`
- `stream_id`
- `tenant_id, workspace_id, owner_id, visibility`
- `asset_id`（可空，录制结束后回填）
- `status`（starting/recording/stopping/succeeded/failed/canceled）
- `started_at, finished_at`
- `error_code`
- `message_key`
- `created_at, updated_at`

## 3.7 命令、审计与上下文索引

### commands
- `id, tenant_id, workspace_id, owner_id`
- `visibility`（默认 `PRIVATE`）
- `acl_json`（默认 `[]`，禁止 NULL）
- `command_type`
- `payload`（JSON）
- `status`（accepted/running/succeeded/failed/canceled）
- `result`（JSON）
- `error_code`
- `message_key`
- `accepted_at, finished_at`
- `created_at, updated_at`

### command_idempotency
- `tenant_id, workspace_id, owner_id, idempotency_key`（联合主键）
- `request_hash`
- `command_id`
- `expires_at`
- `created_at`

语义：
- 查询时仅认为 `expires_at >= now` 的记录有效；
- 过期记录视为不存在；
- 创建命令时按“查有效 -> 同 hash 复用 / 异 hash 冲突 -> 不存在或仅过期则 upsert”执行。

### audit_events
- `id`
- `tenant_id, workspace_id, user_id`
- `trace_id`
- `command_id`
- `event_type`（authorize/execute/egress/plugin/stream/...）
- `resource_type`
- `resource_id`
- `decision`（allow/deny）
- `reason`
- `payload`（JSON）
- `created_at`

建议索引：
- `audit_events(tenant_id, workspace_id, created_at desc)`
- `audit_events(trace_id)`
- `audit_events(command_id)`

### context_bundles
- `id`
- `tenant_id, workspace_id, owner_id, visibility`
- `scope_type`（run/session/workspace）
- `scope_id`
- `facts`（JSON）
- `summaries`（JSON）
- `refs`（JSON）
- `embeddings_index_refs`（JSON）
- `timeline`（JSON）
- `created_at, updated_at`

### chunk_index
- `id`
- `tenant_id`
- `workspace_id`
- `source_type`（asset/report/run）
- `source_id`
- `chunk_key`
- `content`
- `vector_ref`（Redis key 或 SQLite 向量引用）
- `metadata`（JSON）
- `created_at`

## 4. 状态字段约定

- `workflow_runs.status`：`pending/running/succeeded/failed/canceled`
- `step_runs.status`：`pending/running/succeeded/failed/canceled/skipped`
- `plugin_installs.status`：`uploaded/validating/installing/enabled/disabled/failed/rolled_back`
- `streaming_assets.status`：`offline/online/recording/error`
- `stream_recordings.status`：`starting/recording/stopping/succeeded/failed/canceled`

## 5. PostgreSQL / SQLite 差异与兼容策略

## 5.1 类型差异
- JSON：
  - PostgreSQL 使用 `JSONB`。
  - SQLite 使用 `TEXT` 存 JSON 字符串。
- `assets` 表冻结口径：
  - SQLite：`acl_json TEXT NOT NULL DEFAULT '[]'`，`metadata_json TEXT NOT NULL DEFAULT '{}'`
  - PostgreSQL：`acl_json JSONB NOT NULL DEFAULT '[]'::jsonb`，`metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb`
- 时间：
  - PostgreSQL 使用 `TIMESTAMPTZ`。
  - SQLite 使用 ISO8601 字符串（UTC）。
- 枚举：
  - PostgreSQL 可用 enum/check。
  - SQLite 建议 `TEXT + CHECK`。

## 5.2 并发与锁
- PostgreSQL 支持更细粒度行锁，适合高并发调度。
- SQLite 在写竞争场景需降低并发并缩短事务。

## 5.3 查询与索引
- PostgreSQL 可为 JSONB 字段建 GIN 索引。
- SQLite 对 JSON 查询能力有限，v0.1 关键过滤字段应单独结构化。

## 6. 迁移策略

1. 统一迁移版本号（单向递增）。
2. 方言分层：通用迁移 + 方言补丁迁移（pg/sqlite）。
3. Expand-Contract：
   - 先加新字段（可空/有默认）
   - 回填
   - 切读写
   - 再清理旧字段（跨版本执行）
4. 破坏性变更需要：
   - 提前在 `docs/spec/v0.1.md` 标注
   - 同步更新 `openapi.yaml` 与状态机文档
5. 回滚要求：
   - 保留最近两个版本可逆迁移脚本
   - 回滚后保持 API 契约可用（至少只读）。
