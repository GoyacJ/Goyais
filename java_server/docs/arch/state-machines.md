# Java Server State Machines Draft (v0.1)

## 1. Command

- States: `accepted -> running -> succeeded|failed|canceled`
- Required audits: `command.authorize`, `command.execute`, `command.egress`
- Persistence: `commands.status` 与 `audit_events` 同步记录。

## 2. Policy Refresh (Dynamic AuthZ)

- Command type: `policy.refresh`
- Transitions:
  - `accepted -> running`（command gate allow）
  - `running -> succeeded`（发布 `PolicyInvalidationEvent` 成功）
  - `running -> failed`（发布失败）
- Side effects:
  - 本地/Redis snapshot cache evict
  - `policies` 表 upsert 新版本策略快照
  - Redis invalidation topic 广播（若可用）

## 3. WorkflowRun

- States: `pending -> running -> succeeded|failed|canceled`
- Retry creates new run with `retry_of_run_id`
- v0.1 bootstrap 事件流：
  - `workflow.run.started`
  - `workflow.step.started|workflow.step.succeeded|workflow.step.failed|workflow.step.canceled`
  - `workflow.run.succeeded|workflow.run.failed|workflow.run.canceled`
- 受开关控制：`GOYAIS_FEATURE_WORKFLOW_ENABLED`（关闭时 workflow domain sugar 路径返回 `NOT_IMPLEMENTED`）。

## 3.1 Asset

- Asset lifecycle（v0.1）：`ready -> deleted`。
- Write path:
  - `asset.upload` 创建 `ready`
  - `asset.update` 更新元数据/可见性
  - `asset.delete` 进入 `deleted`
- 受开关控制：`GOYAIS_FEATURE_ASSET_LIFECYCLE`（关闭时 `asset.update/delete` 返回 `NOT_IMPLEMENTED`）。

## 3.2 Share

- Share lifecycle（v0.1）：`active -> deleted`（删除为物理删除，API 返回 `status=deleted`）。
- Write path:
  - `share.create` 写入 `acl_entries`
  - `share.delete` 删除当前创建者的 share 行
- 受开关控制：`GOYAIS_FEATURE_ACL_ROLE_SUBJECT`（关闭时拒绝 `subjectType=role`）。

## 4. StepRun

- States: `pending -> running -> succeeded|failed|canceled|skipped`

## 5. PluginInstall

- States: `uploaded -> validating -> installing -> enabled|failed`
- Additional: `disabled`, `rolled_back`

## 6. Stream Recording

- Stream states: `offline|online|recording|error`
- Recording states: `starting|recording|stopping|succeeded|failed|canceled`

## 7. AI Session

- Session states: `active|archived`
- Turn via command types: `ai.intent.plan` / `ai.command.execute`
