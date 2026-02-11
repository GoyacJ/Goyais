# Goyais v0.1 状态机定义

本文件定义 v0.1 三个核心状态机：
- WorkflowRun / StepRun
- PluginInstall
- Stream 录制

所有状态转换必须产生审计事件（`audit_events`）。

## 0. Command Gate

## 0.1 Command 状态
- `accepted`
- `running`
- `succeeded`
- `failed`
- `canceled`

## 0.2 Command 转换

| From | Trigger | Guard | To | 审计事件 |
|---|---|---|---|---|
| accepted | authorize.allow | Authorize hook 放行 | running | `command.running` |
| accepted/running | execute.success | 执行结果写入成功 | succeeded | `command.succeeded` |
| accepted/running | execute.error | 执行失败 | failed | `command.failed` |
| accepted/running | command.cancel | 具备取消权限 | canceled | `command.canceled` |

### 0.2.1 审计补充约束
- `execute.success` 路径必须追加审计：`command.execute(decision=allow)` 与 `command.egress(policyResult=allow)`。
- `execute.error` 路径必须追加审计：`command.execute(decision=deny)` 与 `command.egress(policyResult=deny)`。
- 审计 payload 必须包含：
  - `initiator`（`userId/tenantId/workspaceId`）
  - `context`（`roles/policyVersion/traceId`）
  - `authzResult`（`eventType/decision/reason`）
  - `resourceImpact`（`resourceType/resourceId/eventType`）
  - `data`（业务数据或摘要）
- `command.egress` 的 `data` 必须最小化为 `destination/policyResult/summary`，其中 `summary` 仅保留 digest 与 bytes，不记录原始敏感内容。
- `GET /api/v1/commands*` 读模型至少包含 `acceptedAt` 与 `traceId`，其中 `traceId` 必须可与审计链路关联。
- 请求上下文解析模式由 `GOYAIS_AUTH_CONTEXT_MODE` 决定：
  - `jwt_or_header`：优先 JWT claims；header 仅可在 claims 允许范围内覆盖。
  - `header_only`：必须提供 `X-Tenant-Id/X-Workspace-Id/X-User-Id`。
- `jwt_or_header` 模式下，header 越权覆盖返回 `403 FORBIDDEN + error.authz.forbidden`；JWT 非法返回 `400 INVALID_TOKEN + error.context.invalid_token`。

## 0.3 幂等约束
- 若存在 `Idempotency-Key`，必须在同一事务内执行：查有效映射 -> 同 hash 复用/异 hash 冲突 -> 无有效映射则创建并 upsert。
- 有效映射判定：`expires_at >= now`；过期映射视为不存在。
- `GET /api/v1/commands` 固定排序 `created_at DESC, id DESC`，cursor 基于 `(created_at,id)` 生成不透明 token。
- SQLite / PostgreSQL 在上述语义上保持等价（冲突码、排序、cursor 语义一致）。

## 0.3.1 Provider 就绪态（health gate）
- provider 认证失败（如 Redis NOAUTH）不改变 command 状态机，但会将 healthz 状态标记为 `degraded`。
- 事件总线 provider（`event_bus`）在 `kafka` 模式下若 broker 不可达，`details.providers.event_bus.status=degraded`，但不阻断已有 command 主交易路径。
- 当 provider 就绪恢复后，healthz 状态回到 `ok`，不需要额外迁移。

## 0.4 Share Domain Sugar + 授权闸门（A3）
- `POST /api/v1/shares` 必须转换为 `share.create` command 执行（Command-first）。
- `DELETE /api/v1/shares/{shareId}` 必须转换为 `share.delete` command 执行（Command-first）。
- `POST /api/v1/shares` 执行顺序固定：`Tenant -> Visibility -> ACL -> RBAC -> Egress`。
- v0.1 支持 `resource_type=command|asset`，`subject_type=user|role`。
- 分享前必须校验“同资源 SHARE 权限”：`owner` 或 `acl_entries(resource_type=<目标资源类型>, resource_id=<目标资源ID>, permission=SHARE)` 命中。
- 校验失败返回 `403 FORBIDDEN`，`messageKey=error.authz.forbidden`。
- `share.delete` v0.1 最小语义：仅允许创建者删除同租户/工作区下的 share 记录；不存在时返回 `404 SHARE_NOT_FOUND`。
- `GOYAIS_FEATURE_ACL_ROLE_SUBJECT=false` 时，`subject_type=role` 的分享请求返回 `400 INVALID_SHARE_REQUEST`。

## 0.5 Asset Domain Sugar（A/B 过渡）
- `POST /api/v1/assets` 必须转换为 `asset.upload` command 执行（Command-first）。
- `PATCH /api/v1/assets/{assetId}` 必须转换为 `asset.update` command 执行（Command-first）。
- `DELETE /api/v1/assets/{assetId}` 必须转换为 `asset.delete` command 执行（Command-first）。
- `GET /api/v1/assets/{assetId}/lineage` 为 read path；写路径仍禁止绕过 command gate。
- command 执行器失败时，状态转移固定为 `running -> failed`，并回填：
  - `error_code`
  - `message_key`
  - `command.failed` 事件
- `GOYAIS_FEATURE_ASSET_LIFECYCLE=true` 时启用 `asset.update/asset.delete/lineage`；关闭时三条路径统一返回 `501 NOT_IMPLEMENTED`。

## 0.6 Workflow Domain Sugar（M1 最小闭环）
- `POST /api/v1/workflow-templates`、`POST /api/v1/workflow-templates/{templateId}:patch`、`POST /api/v1/workflow-templates/{templateId}:publish`、`POST /api/v1/workflow-runs`、`POST /api/v1/workflow-runs/{runId}:cancel` 必须转换为 `workflow.*` command 执行（Command-first）。
- `workflow.retry` 仅通过 `POST /api/v1/commands` 暴露（`commandType=workflow.retry`），不新增 domain retry 路由。
- `POST /api/v1/workflow-runs` 请求支持 `fromStepKey`、`testNode`：
  - `fromStepKey`：从指定节点及其下游子图执行。
  - `testNode=true`：仅执行 `fromStepKey` 指向节点（不继续下游）。
- Workflow Engine 执行口径：
  - run 创建后先写入 `pending` 与 step 初始态，root step 入 `workflow_step_queue`。
  - worker 拉取队列任务后执行 step，按依赖关系推进下游入队。
  - step 失败按退避策略重入队；耗尽后 run 进入 `failed`，并跳过依赖节点。
  - run 事件与 step 事件统一写入 `workflow_run_events`，`/workflow-runs/{runId}/events` 可完整回放。
- cancel 语义：`pending/running -> canceled`，并将同 run 下 `pending/running` step 收敛到 `canceled`。
- retry 语义：对终态 run 执行 `workflow.retry` 时必须新建 run：
  - `attempt = source.attempt + 1`（最小值 2）；
  - `retry_of_run_id = source_run_id`；
  - `replay_from_step_key` 来源于 payload（缺省 `step-1`）。

## 0.7 Plugin Domain Sugar（C2 MVP）
- `POST /api/v1/plugin-market/packages`、`POST /api/v1/plugin-market/installs`、`POST /api/v1/plugin-market/installs/{installId}:enable|:disable|:rollback|:upgrade` 必须转换为 `plugin.*` command 执行（Command-first）。
- `plugin.install` 状态链路固定为 `uploaded -> validating -> installing -> enabled|failed`。
- `plugin.enable|plugin.disable|plugin.rollback|plugin.upgrade` 必须在 install 状态机允许的转换上执行，不允许非法跃迁。
- `plugin.upgrade` 必须记录 `plugin_install_history`，并绑定当前 commandId。

## 0.8 Registry C1 Read Path（M2 启动）
- `GET /api/v1/registry/capabilities`、`GET /api/v1/registry/capabilities/{capabilityId}`、`GET /api/v1/registry/algorithms`、`GET /api/v1/registry/providers` 在 v0.1 作为 read-only 能力落地。
- 读路径授权判定：
  - 同 tenant/workspace；
  - `owner` 直接可读；
  - `visibility=WORKSPACE` 可读；
  - 或命中 `acl_entries(resource_type in capability/capability_provider/algorithm, permission=READ)`。
- 列表固定排序：`created_at DESC, id DESC`；`cursor` 优先于 `page/pageSize`。

## 0.9 Stream Domain Sugar（D1 MVP）
- `POST /api/v1/streams` 必须转换为 `stream.create` command 执行（Command-first）。
- `POST /api/v1/streams/{streamId}:update-auth`、`POST /api/v1/streams/{streamId}:record-start`、`POST /api/v1/streams/{streamId}:record-stop`、`POST /api/v1/streams/{streamId}:kick`、`DELETE /api/v1/streams/{streamId}` 必须转换为 `stream.updateAuth`、`stream.record.start`、`stream.record.stop`、`stream.kick`、`stream.delete` command 执行。
- 当流 `state.onPublishTemplateId` 存在时，`stream.record.start` 必须发布 `stream.on_publish` 事件；事件消费者收到后必须通过 command gate 提交 `workflow.run` command（禁止绕过 command service 直调 workflow 写路径）。
- 事件触发的幂等键固定：`stream-onpublish-<recordingId>`；重复投递不应创建重复 run。
- `GOYAIS_FEATURE_STREAM_CONTROL_PLANE=false` 时，`stream.updateAuth`、`stream.delete` 以及对应 domain sugar 路径返回 `501 NOT_IMPLEMENTED`。

## 0.10 Algorithm Domain Sugar（MVP）
- `POST /api/v1/algorithms/{algorithmId}:run` 必须转换为 `algorithm.run` command 执行（Command-first）。
- v0.1 约束：算法执行应复用 workflow 引擎，`algorithm.run` 结果至少包含：
  - `algorithmId`
  - `workflowRunId`
  - `status`
- 错误语义保持统一：`error { code, messageKey, details }`。
- 当前状态：路由已转正并返回 `202 + resource + commandRef`，`algorithm.run` 结果包含 `workflowRunId` 追踪。

## 0.11 AI Domain Sugar（S3 合同基线）
- `POST /api/v1/ai/sessions` 必须转换为 `ai.session.create` command 执行（Command-first）。
- `POST /api/v1/ai/sessions/{sessionId}:archive` 必须转换为 `ai.session.archive` command 执行。
- `POST /api/v1/ai/sessions/{sessionId}/turns` 必须转换为 `ai.intent.plan` 或 `ai.command.execute` command 执行。
- `GET /api/v1/ai/sessions/{sessionId}/events` 为 SSE 读路径；事件源来自 command 与 workflow 执行链路。

## 0.12 ContextBundle Domain Sugar（S6 合同基线）
- ContextBundle 重建能力必须通过 `context.bundle.rebuild` command 执行（Command-first）。
- `GET /api/v1/context-bundles`、`GET /api/v1/context-bundles/{bundleId}` 为 read path，不得绕过 tenant/workspace + ACL 判定。
- `context.bundle.rebuild` 的输入作用域固定为 `run|session|workspace`。
- `GOYAIS_FEATURE_CONTEXT_BUNDLE=false` 时，以上读路径与 `context.bundle.rebuild` command 返回 `501 NOT_IMPLEMENTED`。

## 1. WorkflowRun / StepRun

## 1.1 WorkflowRun 状态
- `pending`
- `running`
- `succeeded`
- `failed`
- `canceled`

## 1.2 WorkflowRun 转换

| From | Trigger | Guard | To | 审计事件 |
|---|---|---|---|---|
| pending | scheduler.dispatch | 授权通过、模板版本存在 | running | `workflow.run.started` |
| running | all_steps_succeeded | 所有必需 step 成功 | succeeded | `workflow.run.succeeded` |
| running | any_step_failed_and_no_retry | 重试次数耗尽 | failed | `workflow.run.failed` |
| running | command.cancel | 发起者有 cancel 权限 | canceled | `workflow.run.canceled` |
| pending | command.cancel | 发起者有 cancel 权限 | canceled | `workflow.run.canceled` |

## 1.3 StepRun 状态
- `pending`
- `running`
- `succeeded`
- `failed`
- `canceled`
- `skipped`

## 1.4 StepRun 转换

| From | Trigger | Guard | To | 审计事件 |
|---|---|---|---|---|
| pending | executor.start | step 依赖满足且授权通过 | running | `workflow.step.started` |
| running | executor.success | 输出 schema 校验通过 | succeeded | `workflow.step.succeeded` |
| running | executor.error | 可重试次数未耗尽 | pending | `workflow.step.retry_scheduled` |
| running | executor.error | 无重试机会 | failed | `workflow.step.failed` |
| pending/running | run.cancel | run 进入 canceled | canceled | `workflow.step.canceled` |
| pending | dependency.failed | 上游失败且策略不允许继续 | skipped | `workflow.step.skipped` |

## 1.5 约束
- run/step 的 `error_code/message_key` 必须在失败态填写。
- step 进入 `running` 前必须通过 Tool Gate（再次授权）。
- `failed`/`canceled`/`succeeded` 为终态，不可回退。

---

## 2. PluginInstall 状态机

## 2.1 PluginInstall 状态
- `uploaded`
- `validating`
- `installing`
- `enabled`
- `disabled`
- `failed`
- `rolled_back`

## 2.2 转换定义

| From | Trigger | Guard | To | 审计事件 |
|---|---|---|---|---|
| uploaded | command.install | 包文件存在 | validating | `plugin.install.validating` |
| validating | validation.pass | 依赖满足、权限 ceiling 未超界 | installing | `plugin.install.installing` |
| validating | validation.fail | 签名/依赖/权限校验失败 | failed | `plugin.install.failed` |
| installing | install.success | Registry 注册成功 | enabled | `plugin.install.enabled` |
| installing | install.fail | 安装步骤失败 | failed | `plugin.install.failed` |
| enabled | command.disable | 调用者有管理权限 | disabled | `plugin.install.disabled` |
| disabled | command.enable | 依赖仍满足 | enabled | `plugin.install.enabled` |
| enabled/disabled | command.upgrade | 目标版本存在且依赖满足 | validating | `plugin.install.upgrade.validating` |
| validating | upgrade.validation.pass | 升级校验通过 | installing | `plugin.install.upgrade.installing` |
| validating | upgrade.validation.fail | 升级校验失败 | failed | `plugin.install.upgrade.failed` |
| enabled/disabled | command.rollback | 目标版本可用 | rolled_back | `plugin.install.rolled_back` |

## 2.3 约束
- `failed` 需包含 `error_code/message_key`。
- `upgrade` 必须写入 `plugin_install_history` 并绑定 `command_id`。
- rollback 结果必须保留与 `commandId` 的关联。
- `enabled` 前，Capability 必须可在 Registry 查询到。

---

## 3. Stream 录制状态机

包含两层状态：
1. 流在线状态（StreamingAsset）。
2. 录制任务状态（StreamRecording）。

## 3.1 StreamingAsset 状态
- `offline`
- `online`
- `recording`
- `error`

### 转换

| From | Trigger | Guard | To | 审计事件 |
|---|---|---|---|---|
| online | command.stream.record.start | 用户有 EXECUTE 权限 | recording | `stream.record.start` |
| recording | command.stream.record.stop | 录制任务存在 | online | `stream.record.stop` |
| online/recording | command.stream.kick | 用户有 MANAGE 权限 | offline | `stream.kick` |
| any | provider.error | 控制面异常 | error | `stream.error` |

## 3.2 StreamRecording 状态
- `starting`
- `recording`
- `stopping`
- `succeeded`
- `failed`
- `canceled`

### 转换

| From | Trigger | Guard | To | 审计事件 |
|---|---|---|---|---|
| recording | command.stream.record.stop | 调用者有 EXECUTE 权限，资产写入成功 | succeeded | `stream.record.succeeded` |
| recording | provider.error | 停止失败或文件缺失 | failed | `stream.record.failed` |
| starting/recording | command.stream.kick | 调用者有 MANAGE 权限 | canceled | `stream.record.canceled` |

## 3.3 约束
- `succeeded` 时必须回填 `asset_id` 并写入 `asset_lineage`。
- 无占位静态文件时 `/favicon.ico`、`/robots.txt` 404 不影响流状态。
- 事件触发 `workflow.run` 必须经过 Command Gate。

## 3.4 Stream 控制面补齐（S4 合同基线）
- `stream.updateAuth`：仅更新鉴权规则，不改变 `StreamingAsset.status`，但必须记录 `stream.auth.updated` 审计事件。
- `stream.delete`：仅允许在 `offline/error` 终止态执行，成功后资源状态迁移为 `deleted`（或软删除标记），并写入 `stream.deleted` 审计事件。

---

## 4. AI Session 状态机（S3 目标）

## 4.1 AISession 状态
- `active`
- `archived`

## 4.2 转换

| From | Trigger | Guard | To | 审计事件 |
|---|---|---|---|---|
| active | command.ai.session.archive | 发起者具备会话管理权限 | archived | `ai.session.archived` |
| active | command.ai.intent.plan | 会话可读且上下文可用 | active | `ai.turn.planned` |
| active | command.ai.command.execute | 命令授权通过 | active | `ai.turn.executed` |

## 4.3 约束
- AI turn 执行必须绑定 `tenantId/workspaceId/userId/roles/policyVersion/traceId`。
- 当授权拒绝时，返回 `FORBIDDEN + error.authz.forbidden` 并在 details 写入拒绝原因。
- `GET /api/v1/ai/sessions/{sessionId}/events` 仅传递摘要事件，不输出敏感原文。

---

## 5. 与 API/数据模型一致性要求

- 状态枚举必须与 `docs/api/openapi.yaml`、`docs/arch/data-model.md` 完全一致。
- 任意状态拒绝时，返回错误结构：`error { code, messageKey, details }`。
- 所有转换需关联 `tenantId/workspaceId/ownerId` 上下文并可审计。

---

## 6. Share 授权判定点（A3/B1）

`POST /api/v1/shares` 在写入 ACL 前必须按固定顺序执行（通过 `share.create` command）：
1. 请求字段校验（`resourceType`、`subjectType`、`permissions`）。
2. 目标资源存在性与租户/工作区一致性校验。
3. 分享者权限校验：
   - owner 直接允许；
   - 或该资源上已有 `SHARE` 权限（仅同一资源生效，禁止全局 SHARE）。
4. `permissions` 归一化（大写/去重/排序）并落库。

拒绝语义：
- 权限不足返回 `403 FORBIDDEN` + `messageKey=error.authz.forbidden`。
- 非法入参返回 `400 INVALID_SHARE_REQUEST`。

`DELETE /api/v1/shares/{shareId}`（`share.delete` command）：
1. 请求上下文校验（tenant/workspace/user）。
2. share 存在性与作用域校验（同 tenant/workspace）。
3. 删除权限校验（v0.1 最小语义：`created_by == request.userId`）。
4. 删除成功后返回最小资源快照 `{id,status=deleted}`。
