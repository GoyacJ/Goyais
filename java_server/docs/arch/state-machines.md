# Java Server State Machines Draft (v0.1)

## 1. Command

- States: `accepted -> running -> succeeded|failed|canceled`
- Required audits: `command.authorize`, `command.execute`, `command.egress`

## 2. Policy Refresh (Dynamic AuthZ)

- Command type: `policy.refresh`
- Transitions:
  - `accepted -> running`（command gate allow）
  - `running -> succeeded`（发布 `PolicyInvalidationEvent` 成功）
  - `running -> failed`（发布失败）
- Side effects:
  - 本地 snapshot cache evict
  - Redis invalidation topic 广播（若可用）

## 3. WorkflowRun

- States: `pending -> running -> succeeded|failed|canceled`
- Retry creates new run with `retry_of_run_id`

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
