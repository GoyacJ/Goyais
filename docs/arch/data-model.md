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
- `resource_type`（v0.1 share API 支持 `command/asset`；查询判定可覆盖 `workflow_template/workflow_run/capability/capability_provider/algorithm`）
- `resource_id`
- `subject_type`（v0.1 固定 `user`）
- `subject_id`
- `permissions`（JSON 数组：READ/WRITE/EXECUTE/MANAGE/SHARE）
- `expires_at`（可空）
- `created_by`
- `created_at`

建议索引：
- `(resource_type, resource_id)`
- `(subject_type, subject_id)`
- `(tenant_id, workspace_id)`

存储与查询口径（冻结）：
- SQLite：`permissions` 使用 `TEXT` 存 JSON 数组；权限包含判断使用 `json_each`（示例：`EXISTS (SELECT 1 FROM json_each(a.permissions) p WHERE p.value='READ')`）。
- PostgreSQL：`permissions` 使用 `JSONB`；权限包含判断使用 `@>`（示例：`a.permissions @> '["READ"]'::jsonb`）。
- 过期判定统一：`expires_at < now` 视为无效 ACL。
- 写入路径固定为 Command-first：
  - `share.create`：新增 `acl_entries` 记录（subjectType 固定 `user`）。
  - `share.delete`：按 `(id, tenant_id, workspace_id, created_by)` 删除记录。

## 3.3 资产与血缘

### assets
- `id, tenant_id, workspace_id, owner_id, visibility`
- `name, asset_type, mime_type, size_bytes`
- `uri`（对象存储路径）
- `hash`（sha256）
- `metadata`（JSON）
- `status`（active/archived/deleted）
- `created_at, updated_at`

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
- `trace_id`（用于跨 command/run/step 串联）
- `template_id, template_version`
- `attempt`
- `retry_of_run_id`（可空，引用被重试 run）
- `replay_from_step_key`（可空，记录从哪个 step 重放）
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
- `trace_id`（与所属 run 对齐）
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
- `workflow_runs(retry_of_run_id)`
- `step_runs(run_id, step_key, attempt)`

## 3.5 能力注册与插件

v0.1 当前实现进度：
- C1 read-only 已落地：`capability_providers/capabilities/algorithms` 支持租户+工作区隔离查询与 ACL.READ 判定。
- C2 plugin market MVP 已落地：`plugin_packages/plugin_installs` 支持上传、安装、启停、回滚与状态回查。

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

### algorithm_runs
- `id, tenant_id, workspace_id, owner_id, visibility`
- `algorithm_id`
- `workflow_run_id`（映射到 workflow 实际执行）
- `command_id`（`algorithm.run`）
- `status`（pending/running/succeeded/failed/canceled）
- `inputs`（JSON）
- `outputs`（JSON）
- `asset_ids`（JSON，运行产生的资产引用）
- `error_code`
- `message_key`
- `created_at, updated_at`

建议索引：
- `algorithm_runs(tenant_id, workspace_id, created_at desc, id desc)`
- `algorithm_runs(algorithm_id, created_at desc, id desc)`
- `algorithm_runs(workflow_run_id)`

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
- `acl_json`（JSON）
- `path`
- `protocol`（rtsp/rtmp/srt/webrtc/hls）
- `source`（push/pull）
- `endpoints_json`（JSON）
- `state_json`（JSON）
- `status`（offline/online/recording/error）
- `created_at, updated_at`

约束与索引：
- `UNIQUE(tenant_id, workspace_id, path)`
- `streaming_assets(tenant_id, workspace_id, created_at desc, id desc)`

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

建议索引：
- `stream_recordings(stream_id, created_at desc, id desc)`
- `stream_recordings(tenant_id, workspace_id, created_at desc, id desc)`

## 3.7 命令、审计与上下文索引

### commands
- `id, tenant_id, workspace_id, owner_id`
- `command_type`（示例：`asset.upload`、`workflow.run`、`workflow.retry`、`algorithm.run`、`share.create`、`share.delete`、`plugin.upload`、`plugin.install`、`plugin.enable`、`plugin.disable`、`plugin.rollback`、`stream.create`、`stream.record.start`、`stream.record.stop`、`stream.kick`）
- `payload`（JSON）
- `status`（accepted/running/succeeded/failed/canceled）
- `visibility`（默认 `PRIVATE`，NOT NULL）
- `acl_json`（默认 `[]`，NOT NULL）
- `result`（JSON）
- `error_code`
- `message_key`
- `accepted_at, finished_at`
- `created_at, updated_at`
- `trace_id`（API 读模型字段，来源 `audit_events` 中同 `command_id` 的聚合投影；`commands` 主表可不落该列）

建议索引：
- `commands(tenant_id, workspace_id, created_at desc, id desc)`（list 固定排序）
- `commands(id)`

### command_idempotency
- `tenant_id, workspace_id, owner_id`
- `idempotency_key`
- `request_hash`
- `command_id`
- `expires_at`
- `created_at`

约束与语义：
- 唯一键：`(tenant_id, workspace_id, owner_id, idempotency_key)`。
- 查询幂等映射时，仅 `expires_at >= now` 视为有效。
- 过期记录视为不存在，可在事务内 upsert 覆盖。

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

payload 约定（v0.1）：
- `payload.context.roles`：请求角色列表（默认 `member`）。
- `payload.context.policyVersion`：策略版本（默认 `v0.1`）。
- `payload.context.traceId`：跨 command/run/step 关联追踪 ID。
- `payload.data`：事件原始业务数据（脱敏/摘要后落库）。

建议索引：
- `audit_events(tenant_id, workspace_id, created_at desc)`
- `audit_events(trace_id)`
- `audit_events(command_id)`

### command_events
- `id`
- `command_id`
- `event_type`
- `payload`（JSON）
- `created_at`

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

补充（v0.1 provider 配置语义）：
- `cache.redis_password` 与 `vector.redis_password` 仅作为运行时连接配置，不入库。
- `event_bus.kafka.*` 仅作为运行时连接配置，不入库（broker/topic/group 由配置管理）。
- 认证失败等 provider 连接错误通过 healthz `details.providers.*.error` 暴露，不写入业务表。

## 5.1 类型差异
- JSON：
  - PostgreSQL 使用 `JSONB`。
  - SQLite 使用 `TEXT` 存 JSON 字符串。
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
