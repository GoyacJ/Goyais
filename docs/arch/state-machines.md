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

## 0.3 幂等约束
- 若存在 `Idempotency-Key`，必须在同一事务内执行：查有效映射 -> 同 hash 复用/异 hash 冲突 -> 无有效映射则创建并 upsert。
- 有效映射判定：`expires_at >= now`；过期映射视为不存在。
- `GET /api/v1/commands` 固定排序 `created_at DESC, id DESC`，cursor 基于 `(created_at,id)` 生成不透明 token。
- SQLite / PostgreSQL 在上述语义上保持等价（冲突码、排序、cursor 语义一致）。

## 0.3.1 Provider 就绪态（health gate）
- provider 认证失败（如 Redis NOAUTH）不改变 command 状态机，但会将 healthz 状态标记为 `degraded`。
- 当 provider 就绪恢复后，healthz 状态回到 `ok`，不需要额外迁移。

## 0.4 Share Domain Sugar + 授权闸门（A3）
- `POST /api/v1/shares` 必须转换为 `share.create` command 执行（Command-first）。
- `DELETE /api/v1/shares/{shareId}` 必须转换为 `share.delete` command 执行（Command-first）。
- `POST /api/v1/shares` 执行顺序固定：`Tenant -> Visibility -> ACL -> RBAC -> Egress`。
- v0.1 支持 `resource_type=command|asset`，且 `subject_type=user`。
- 分享前必须校验“同资源 SHARE 权限”：`owner` 或 `acl_entries(resource_type=<目标资源类型>, resource_id=<目标资源ID>, permission=SHARE)` 命中。
- 校验失败返回 `403 FORBIDDEN`，`messageKey=error.authz.forbidden`。
- `share.delete` v0.1 最小语义：仅允许创建者删除同租户/工作区下的 share 记录；不存在时返回 `404 SHARE_NOT_FOUND`。

## 0.5 Asset Domain Sugar（A/B 过渡）
- `POST /api/v1/assets` 必须转换为 `asset.upload` command 执行（Command-first）。
- command 执行器失败时，状态转移固定为 `running -> failed`，并回填：
  - `error_code`
  - `message_key`
  - `command.failed` 事件
- 当前 `assets` 读接口可用；`PATCH/DELETE/lineage` 为占位，统一 `501 NOT_IMPLEMENTED`。

## 0.6 Workflow Domain Sugar（M1 最小闭环）
- `POST /api/v1/workflow-templates`、`POST /api/v1/workflow-templates/{templateId}:patch`、`POST /api/v1/workflow-templates/{templateId}:publish`、`POST /api/v1/workflow-runs`、`POST /api/v1/workflow-runs/{runId}:cancel` 必须转换为 `workflow.*` command 执行（Command-first）。
- run 执行模式（v0.1 最小实现）：
  - `mode=sync`：`pending -> succeeded`，并创建 1 条 `step_run(succeeded)`。
  - `mode=running`：`pending -> running`，并创建 1 条 `step_run(running)`。
  - `mode=fail`：`pending -> failed`，并创建 1 条 `step_run(failed)`，且回填 `error_code/message_key`。
- cancel 语义：`pending/running -> canceled`，并将同 run 下 `pending/running` step 收敛到 `canceled`。

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
| enabled/disabled | command.rollback | 目标版本可用 | rolled_back | `plugin.install.rolled_back` |

## 2.3 约束
- `failed` 需包含 `error_code/message_key`。
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
| offline | mediamtx.onPublish | path 授权通过 | online | `stream.online` |
| online | command.record_start | 用户有 EXECUTE 权限 | recording | `stream.record.start` |
| recording | command.record_stop | 录制任务存在 | online | `stream.record.stop` |
| online/recording | mediamtx.disconnect | 连接断开 | offline | `stream.offline` |
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
| starting | provider.ack | MediaMTX 返回成功 | recording | `stream.recording` |
| starting | provider.error | 启动失败 | failed | `stream.record.failed` |
| recording | command.record_stop | 调用者有权限 | stopping | `stream.record.stopping` |
| stopping | provider.file_ready | 录制文件落盘并入 asset | succeeded | `stream.record.succeeded` |
| stopping | provider.error | 停止失败或文件缺失 | failed | `stream.record.failed` |
| starting/recording | command.cancel | 调用者有权限 | canceled | `stream.record.canceled` |

## 3.3 约束
- `succeeded` 时必须回填 `asset_id` 并写入 `asset_lineage`。
- 无占位静态文件时 `/favicon.ico`、`/robots.txt` 404 不影响流状态。
- 事件触发 `workflow.run` 必须经过 Command Gate。

---

## 4. 与 API/数据模型一致性要求

- 状态枚举必须与 `docs/api/openapi.yaml`、`docs/arch/data-model.md` 完全一致。
- 任意状态拒绝时，返回错误结构：`error { code, messageKey, details }`。
- 所有转换需关联 `tenantId/workspaceId/ownerId` 上下文并可审计。

---

## 5. Share 授权判定点（A3/B1）

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
