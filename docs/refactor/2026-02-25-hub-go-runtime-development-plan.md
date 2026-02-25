# Hub+Go Runtime Development Plan (Maintained Per Task)

- Date: 2026-02-25
- Owner: Goyais Refactor Program
- Status: Active
- Linked baseline: `docs/refactor/2026-02-25-hub-go-runtime-refactor-master-plan.md`

## Execution Principles

1. Behavior parity with original Kode-cli is mandatory.
2. Core first, UI second. UI layer must stay logic-thin.
3. Exactly one mergeable small change per task.
4. Every task update must include code/test/runbook/change list evidence.
5. Unknown behavior must be proven from source/docs/tests or probe experiment results.
6. No secrets in code; clear error handling; explicit debug/log switches.
7. Cross-platform distribution and responsive startup/interaction are required.

## Task Board

Status vocabulary is fixed: `todo`, `in_progress`, `done`, `blocked`.

| Task ID | Title | Status | Owner | Depends On | Notes |
| --- | --- | --- | --- | --- | --- |
| T-001 | Build contract/parity checklist and golden baseline assets | done | Codex | - | Delivered wrapper-level executable baseline + golden asset/check tooling |
| T-002 | Implement core protocol/config/state machine skeleton | done | Codex | T-001 | Added agentcore config/protocol/state skeleton with unit+integration tests |
| T-003 | Implement base tools + safety gate in core | done | Codex | T-002 | Added safety gate + tool executor/base tools; U-002 probe resolved |
| T-004 | Integrate core into Hub execution and SSE path | done | Codex | T-003 | Added run-control endpoint + SSE run-event adapter over agentcore |
| T-005 | Migrate shared types and Desktop/Mobile event adapters | done | Codex | T-004 | Added Run* shared types and Desktop stream run-event adapter |
| T-006 | Implement Go CLI/TUI adapters over core | done | Codex | T-005 | Added cmd/goyais-cli thin adapters over agentcore runtime contract |
| T-007 | Remove Worker + Kode-cli + release rollback package | done | Codex | T-006 | Cut over to Hub-only runtime path and shipped rollback scripts |

## Single Task Delivery Template (Mandatory)

Each completed task must append one section using this exact field set:

```markdown
### Task Delivery: T-XXX

- Task ID:
- 目标与范围:
- 修改文件清单:
- 行为基准证据（原仓源码/README/脚本/测试引用）:
- 实现说明:
- 运行说明:
- 测试证据:
- 输出/错误输出/退出码对等结果:
- 风险与回退:
- 变更点清单:
- 下一任务:
```

No field may be omitted.

## Pending Confirmation List

Use ID prefix `U-001`, increment by one per unresolved behavior.

| Unknown ID | Topic | Current Evidence | Why Unresolved | Next Probe |
| --- | --- | --- | --- | --- |
| U-001 | Runtime parity beyond wrapper bootstrap (`--print`/interactive/help matrix) | `Kode-cli` wrapper-level parity evidence captured in `docs/refactor/parity/kode-cli-wrapper-golden.json`; Phase-4 one-shot cutover removed in-repo `Kode-cli` runtime payload source | Waived with approval for T-007 cutover acceptance (user approved continuing acceptance on 2026-02-25) | N/A（waived；record: `docs/refactor/2026-02-25-u001-waiver.md`） |

## Resolved Probe Records

- U-002 (`approve`/`deny`/`resume` semantics): resolved by `docs/refactor/2026-02-25-u002-run-control-probe.md`.
- U-001（runtime parity full matrix）: waived with approval by `docs/refactor/2026-02-25-u001-waiver.md`.

## Minimal Probe Experiment Template

For any unresolved behavior, record probe details with this template:

```markdown
### Probe: U-XXX

- Goal:
- Original project path:
- Command(s) to run:
- Input fixture:
- Expected observable points:
  - stdout:
  - stderr:
  - exit code:
  - side effect:
- Actual result:
- Conclusion:
- Follow-up action:
```

## Merge Gates Checklist

A task cannot be marked `done` unless all are checked:

- [ ] Code change implemented and scoped to one mergeable task.
- [ ] Unit tests added/updated and passing.
- [ ] Integration tests added/updated and passing.
- [ ] Run instructions documented.
- [ ] Output parity verified (stdout/stderr format).
- [ ] Exit code parity verified.
- [ ] Document sync completed (`master-plan` unchanged except change record; this file updated).

## Pre-release Global Acceptance Checklist

- [x] All tasks T-001..T-007 are `done`.
- [x] No `blocked` task remains unresolved.
- [x] Pending unknown list is empty or explicitly waived with approval.
- [x] Hub API/SSE/control/approval end-to-end tests pass.
- [x] Desktop and Mobile compatibility tests pass.
- [x] Cross-platform distributable checks pass.
- [x] Rollback drill executed and documented.

## Daily / Iteration Summary

Append one short summary per iteration:

```markdown
### Iteration YYYY-MM-DD

- Completed:
- In progress:
- Blockers:
- Risks:
- Next focus:
```

### Iteration 2026-02-25

- Completed: Created `docs/refactor` and initialized frozen master plan + development plan governance.
- In progress: None.
- Blockers: None.
- Risks: None introduced by documentation-only change.
- Next focus: Start T-001 contract/parity baseline work.

### Task Delivery: T-001

- Task ID: T-001
- 目标与范围: 建立可执行的 Kode-cli 基线契约（stdout/stderr/exit code），产出 golden 资产、校验脚本、运行手册与自动化测试，作为后续 Go runtime 对等回归基线。
- 修改文件清单:
  - `scripts/refactor/t001-capture-kode-cli-baseline.mjs`
  - `scripts/refactor/t001-kode-cli-baseline.test.mjs`
  - `docs/refactor/2026-02-25-kode-cli-parity-checklist.md`
  - `docs/refactor/parity/kode-cli-wrapper-golden.json`
  - `package.json`
  - `docs/refactor/2026-02-25-hub-go-runtime-development-plan.md`
- 行为基准证据（原仓源码/README/脚本/测试引用）:
  - `Kode-cli/cli.js`（`--help-lite`、`--version`、fallback 退出行为）
  - `Kode-cli/cli-acp.js`（ACP fallback 退出行为）
  - `Kode-cli/tests/e2e/cli-smoke.test.ts`（help/version/print 参数验证期望）
  - `Kode-cli/README.md`（native binary + Node fallback 说明）
  - `Kode-cli/package.json`（版本号基线）
- 实现说明:
  - 新增 T-001 脚本，固定 4 个 wrapper bootstrap 合约用例，采集 stdout/stderr/exit code 到 `docs/refactor/parity/kode-cli-wrapper-golden.json`。
  - 同脚本支持 `--check` 模式，对比当前输出与 golden 并在漂移时失败。
  - 新增 Node 原生测试，覆盖“采集后可通过校验”与“golden 被篡改时校验失败”两条路径。
  - 新增 parity checklist 文档，固化证据来源、用例矩阵与 runbook。
- 运行说明:
  - `pnpm run refactor:t001:capture`
  - `pnpm run refactor:t001:check`
  - `pnpm run refactor:t001:test`
- 测试证据:
  - `node --test scripts/refactor/t001-kode-cli-baseline.test.mjs` 通过（2 tests）。
  - `node scripts/refactor/t001-capture-kode-cli-baseline.mjs --check` 通过。
- 输出/错误输出/退出码对等结果:
  - `cli_help_lite` 与 `cli_version`：stdout 对等、退出码 0。
  - `cli_help_without_dist` 与 `acp_without_dist`：stderr fallback 文案对等、退出码 1。
- 风险与回退:
  - 风险：当前基线仅覆盖 wrapper bootstrap，不含完整 runtime 交互契约。
  - 回退：删除本次新增脚本/测试/文档与 golden 文件，并将 T-001 状态回退为 `todo`。
- 变更点清单:
  - 新增可复现 golden 采集流程。
  - 新增可自动失败的 parity 漂移校验流程。
  - 新增 T-001 合约矩阵文档与 pending unknown 更新。
- 下一任务: T-002（core protocol/config/state machine skeleton，延续“无 UI 业务逻辑”边界）

### Iteration 2026-02-25 (T-001)

- Completed: Delivered T-001 wrapper-level parity checklist, golden asset, capture/check script, and automated tests.
- In progress: None.
- Blockers: Runtime-level parity probes still blocked by missing reproducible `Kode-cli/dist` payload in this workspace.
- Risks: If wrapper contract changes intentionally, golden must be re-captured to avoid false failures.
- Next focus: Start T-002 core skeleton and expand runtime probe prerequisites.

### Task Delivery: T-002

- Task ID: T-002
- 目标与范围: 在 `services/hub/internal/agentcore` 建立 `config/protocol/state` 三个核心模块骨架，确保与 UI/HTTP adapter 解耦，并为后续 T-003/T-004 集成提供可测试边界。
- 修改文件清单:
  - `services/hub/internal/agentcore/config/provider.go`
  - `services/hub/internal/agentcore/config/provider_test.go`
  - `services/hub/internal/agentcore/protocol/run_event.go`
  - `services/hub/internal/agentcore/protocol/run_event_test.go`
  - `services/hub/internal/agentcore/state/machine.go`
  - `services/hub/internal/agentcore/state/machine_test.go`
  - `services/hub/internal/agentcore/skeleton_integration_test.go`
  - `docs/refactor/2026-02-25-hub-go-runtime-development-plan.md`
- 行为基准证据（原仓源码/README/脚本/测试引用）:
  - `Kode-cli/docs/develop/architecture.md`（分层架构与模块分离）
  - `Kode-cli/docs/develop/configuration.md`（配置分层与覆盖顺序）
  - `Kode-cli/docs/develop/modules/repl-interface.md`（REPL 状态与交互循环上下文）
  - `docs/refactor/2026-02-25-hub-go-runtime-refactor-master-plan.md`（T-002 边界与 interface baseline）
- 实现说明:
  - `config`：定义 `ResolvedConfig`、`SessionMode`、`Provider` 与 `StaticProvider`，实现最小校验（`session_mode`、`default_model`）和路径/环境捕获。
  - `protocol`：定义 `RunEventType` 与 `RunEvent` envelope，并提供 `Validate()` 以校验事件基础字段。
  - `state`：定义 `RunState`、`ControlAction`、`Machine` 与最小合法转移图；支持 `approve/deny/stop` 映射，`resume` 暂以 skeleton 限制并记录到 `U-002`。
  - 新增 `agentcore` 级集成测试，串联 config + state + protocol 生命周期最小路径。
- 运行说明:
  - `cd services/hub && go test ./internal/agentcore/...`
  - `cd services/hub && go test ./...`
- 测试证据:
  - RED: `go test ./internal/agentcore/...`（在实现前因缺失类型失败，已记录）。
  - GREEN: `go test ./internal/agentcore/...` 通过。
  - 回归: `go test ./...` 通过（含 `internal/httpapi`）。
- 输出/错误输出/退出码对等结果:
  - 本任务不引入用户可见 CLI/HTTP 输出变更；对等性在该层以“状态/事件/配置结构可验证且测试通过”作为内部契约基线。
  - 已保持 `/v1` 外部行为不变（未接入 adapter 层）。
- 风险与回退:
  - 风险：状态机 `resume` 语义尚未与原行为对齐（`U-002`）。
  - 回退：删除 `internal/agentcore` 新增目录与测试，并将 T-002 状态回退 `todo`。
- 变更点清单:
  - 新增 `agentcore` 三大核心模块骨架代码。
  - 新增对应单测与跨模块集成测试。
  - 更新未知项追踪（`U-002`）以约束后续行为对齐。
- 下一任务: T-003（base tools + safety gate in core，并将状态机控制语义从 skeleton 提升为证据驱动实现）

### Iteration 2026-02-25 (T-002)

- Completed: Implemented `agentcore/config`, `agentcore/protocol`, `agentcore/state` skeleton with unit and integration coverage.
- In progress: None.
- Blockers: Control-action parity truth table with original runtime remains pending (`U-002`).
- Risks: If downstream integration assumes `resume` semantics prematurely, behavior may diverge.
- Next focus: Start T-003 tooling+safety core module and resolve `U-002` with probes.

### Task Delivery: T-003

- Task ID: T-003
- 目标与范围: 实现 `agentcore` 的基础工具执行层与安全门控层（tools + safety gate），并在编码前完成 `U-002` 控制语义探针，确保 `approve/deny/resume` 有证据驱动的状态映射。
- 修改文件清单:
  - `services/hub/internal/httpapi/control_semantics_probe_test.go`
  - `services/hub/internal/agentcore/safety/gate.go`
  - `services/hub/internal/agentcore/safety/gate_test.go`
  - `services/hub/internal/agentcore/tools/types.go`
  - `services/hub/internal/agentcore/tools/registry.go`
  - `services/hub/internal/agentcore/tools/errors.go`
  - `services/hub/internal/agentcore/tools/executor.go`
  - `services/hub/internal/agentcore/tools/echo_tool.go`
  - `services/hub/internal/agentcore/tools/run_command_tool.go`
  - `services/hub/internal/agentcore/tools/base_tools.go`
  - `services/hub/internal/agentcore/tools/executor_test.go`
  - `services/hub/internal/agentcore/tools_safety_integration_test.go`
  - `services/hub/internal/agentcore/state/machine.go`
  - `services/hub/internal/agentcore/state/machine_test.go`
  - `docs/refactor/2026-02-25-u002-run-control-probe.md`
  - `docs/refactor/2026-02-25-hub-go-runtime-development-plan.md`
- 行为基准证据（原仓源码/README/脚本/测试引用）:
  - `services/hub/internal/httpapi/handlers_worker_internal.go`（legacy `confirmation_resolved` 归一化：`deny -> execution_stopped`）
  - `services/hub/internal/httpapi/execution_control_store.go`（控制轮询仅下发 `stop`）
  - `services/hub/internal/httpapi/state_execution_domain_test.go`（legacy `confirm` 指令/状态持久化兼容）
  - `services/hub/internal/httpapi/integration_contract_test.go`（`/v1/executions/{execution_id}/confirm` 已移除）
  - `docs/refactor/2026-02-25-u002-run-control-probe.md`（U-002 探针结论）
- 实现说明:
  - 新增 `safety.Gate`：基于风险等级与会话模式决策 `allow / require_approval / deny`。
  - 新增 `tools` 执行骨架：`Registry`、`Executor`、统一错误类型、`Tool` 接口模型。
  - 新增 base tools：`echo`（低风险）与 `run_command`（高风险）。
  - `Executor` 在工具执行前接入 `safety.Gate`，将高风险审批与 plan 模式拒绝前置到 core。
  - 基于 U-002 探针结果调整状态机控制语义：
    - `deny -> cancelled`
    - `approve/resume -> running`（queued/waiting_approval）
- 运行说明:
  - `cd services/hub && go test -count=1 ./internal/httpapi -run TestProbe`
  - `cd services/hub && go test -count=1 ./internal/agentcore/...`
  - `cd services/hub && go test -count=1 ./...`
- 测试证据:
  - RED（TDD）:
    - `go test ./internal/agentcore/... ./internal/httpapi -run 'TestProbe|TestMachine|TestGate|TestExecutor'` 在实现前失败（缺失 `safety/tools` 代码，且 `state` 语义不匹配）。
  - GREEN:
    - `go test -count=1 ./internal/httpapi -run TestProbe` 通过。
    - `go test -count=1 ./internal/agentcore/...` 通过。
  - 回归:
    - `go test -count=1 ./...` 通过。
- 输出/错误输出/退出码对等结果:
  - `U-002` 对等探针结论：
    - `approve` 对齐到继续执行（running）
    - `deny` 对齐到停止/取消（cancelled）
    - `resume` 在 legacy 无直接路由语义；按证据推断映射为恢复到 running（已在 probe 文档中标注为 inference）。
  - 本任务未改动对外 `/v1` 路由行为，仅完善 core 层语义与可测试执行链。
- 风险与回退:
  - 风险：`resume` 语义基于 legacy 缺失行为推断，后续若补充更强证据需微调状态机。
  - 回退：删除 `agentcore/safety` 与 `agentcore/tools` 新增代码，并回退 `state` 语义调整与文档更新。
- 变更点清单:
  - 新增可复用安全门控策略层。
  - 新增可扩展工具注册/执行骨架与基础工具。
  - 新增 U-002 探针测试与探针报告，关闭 U-002 未知项。
  - 将状态机控制语义从 skeleton 升级为证据驱动实现。
- 下一任务: T-004（将 `agentcore` 集成进 Hub 执行流与 SSE `/v1` run 语义适配）

### Iteration 2026-02-25 (T-003)

- Completed: Implemented `agentcore/tools` + `agentcore/safety`, resolved U-002 probe, and aligned control-state semantics in `agentcore/state`.
- In progress: None.
- Blockers: None.
- Risks: Resume mapping is evidence-based inference due missing legacy direct route; validate again during T-004 end-to-end integration.
- Next focus: Start T-004 hub integration with run-centric events/control endpoints.

### Task Delivery: T-004

- Task ID: T-004
- 目标与范围: 将 `agentcore` 接入 Hub `/v1` 执行流适配层，落地 run-centric SSE 事件语义与统一 control 语义（`stop/approve/deny/resume`），并保持内部存储结构最小侵入。
- 修改文件清单:
  - `services/hub/internal/httpapi/agentcore_adapter.go`
  - `services/hub/internal/httpapi/handlers_run_control.go`
  - `services/hub/internal/httpapi/handlers_conversation_events.go`
  - `services/hub/internal/httpapi/router.go`
  - `services/hub/internal/httpapi/agentcore_adapter_test.go`
  - `services/hub/internal/httpapi/handlers_run_control_test.go`
  - `services/hub/internal/httpapi/handlers_conversation_events_run_test.go`
  - `services/hub/internal/httpapi/openapi_contract_test.go`
  - `packages/contracts/openapi.yaml`
  - `docs/refactor/2026-02-25-hub-go-runtime-development-plan.md`
- 行为基准证据（原仓源码/README/脚本/测试引用）:
  - `docs/refactor/2026-02-25-hub-go-runtime-refactor-master-plan.md`（T-004 `/v1` run-centric SSE/control 目标）
  - `services/hub/internal/httpapi/handlers_conversation_events.go`（legacy SSE 输出 `ExecutionEvent`）
  - `services/hub/internal/httpapi/handlers_execution_flow.go`（legacy execution action/stop 控制路径）
  - `docs/refactor/2026-02-25-u002-run-control-probe.md`（`approve/deny/resume` 语义依据）
- 实现说明:
  - 新增 `agentcore_adapter`，统一 `ExecutionEvent -> RunEvent`、`ExecutionState <-> RunState`、`action string -> ControlAction` 映射。
  - SSE `/v1/conversations/{conversation_id}/events` 输出改为通过 adapter 序列化 `RunEvent`，实现 run-centric 事件语义，同时保留现有 backlog/subscriber 存储机制。
  - 新增 `/v1/runs/{run_id}/control`，在 handler 中使用 `agentcore/state.Machine` 应用 `stop/approve/deny/resume` 转移，并回写 execution/conversation 状态与事件。
  - 更新 OpenAPI 契约：新增 `/v1/runs/{run_id}/control` 路径与 `RunControlRequest/RunControlResponse` schema。
- 运行说明:
  - `cd services/hub && go test -count=1 ./internal/httpapi -run 'TestMapExecutionEventToRunEvent|TestMapExecutionStateToRunState|TestRunControlEndpoint_DenyQueuedRun|TestConversationEventsSSE_EmitsRunSemantics|TestOpenAPIContainsV040CriticalRoutes'`
  - `cd services/hub && go test -count=1 ./internal/httpapi`
  - `cd services/hub && go test -count=1 ./internal/agentcore/...`
  - `cd services/hub && go test -count=1 ./...`
- 测试证据:
  - RED（TDD）:
    - `go test -count=1 ./internal/httpapi -run 'TestMapExecutionEventToRunEvent|TestMapExecutionStateToRunState|TestRunControlEndpoint_DenyQueuedRun|TestConversationEventsSSE_EmitsRunSemantics|TestOpenAPIContainsV040CriticalRoutes'` 首次失败（缺失 adapter 函数）。
  - GREEN:
    - 同一命令在实现后通过。
    - `go test -count=1 ./internal/httpapi` 通过。
    - `go test -count=1 ./internal/agentcore/...` 通过。
  - 回归:
    - `go test -count=1 ./...` 通过。
- 输出/错误输出/退出码对等结果:
  - `/v1/conversations/{conversation_id}/events` 现在输出 run-centric 字段（`type/session_id/run_id/sequence/timestamp/payload`）。
  - 新增 `/v1/runs/{run_id}/control` 返回统一控制结果（`ok/run_id/state/previous_state`）并对非法 action/state 给出标准错误。
  - 现有 `/v1/executions/*` 与 internal worker 路径保持兼容。
- 风险与回退:
  - 风险：Desktop/Mobile 当前仍消费 `ExecutionEvent` 类型（T-005 待迁移），SSE 字段升级期间需保持客户端联动改造。
  - 回退：移除 `/v1/runs/{run_id}/control` 路由与 `agentcore_adapter` 接线，并将 SSE 序列化回退为 `ExecutionEvent`。
- 变更点清单:
  - 增加 Hub run-control 统一入口。
  - 增加 run-centric SSE adapter 层。
  - 将 Hub 控制转移显式接入 `agentcore/state.Machine`。
  - 更新 OpenAPI 契约与对应 contract 测试。
- 下一任务: T-005（迁移 shared-core 类型与 Desktop/Mobile 事件适配到 `Run*`）

### Iteration 2026-02-25 (T-004)

- Completed: Wired agentcore adapter into Hub SSE/control path, added `/v1/runs/{run_id}/control`, and aligned `/v1` event semantics to run-centric output.
- In progress: None.
- Blockers: None.
- Risks: Desktop/Mobile still rely on `ExecutionEvent` contract until T-005 adapter migration completes.
- Next focus: Start T-005 shared types and client event adapter migration.

### Task Delivery: T-005

- Task ID: T-005
- 目标与范围: 迁移 shared-core 与 Desktop 流式事件适配层到 `Run*` 语义，同时保持现有会话 UI 执行流不回退（通过 adapter 兼容 `ExecutionEvent` 处理链）。
- 修改文件清单:
  - `packages/shared-core/src/api-common.ts`
  - `packages/shared-core/src/api-project.ts`
  - `apps/desktop/src/modules/conversation/store/runEventAdapter.ts`
  - `apps/desktop/src/modules/conversation/store/stream.ts`
  - `apps/desktop/src/modules/conversation/services/index.ts`
  - `apps/desktop/src/shared/services/sseClient.ts`
  - `apps/desktop/src/modules/conversation/tests/run-event-adapter.spec.ts`
  - `apps/desktop/src/modules/conversation/tests/conversation-stream.spec.ts`
  - `docs/refactor/2026-02-25-hub-go-runtime-development-plan.md`
- 行为基准证据（原仓源码/README/脚本/测试引用）:
  - `packages/shared-core/src/api-project.ts`（既有 `ExecutionEvent`/`ExecutionState` 类型契约）
  - `apps/desktop/src/modules/conversation/store/stream.ts`（SSE 事件入口与 normalize 路由）
  - `apps/desktop/src/modules/conversation/store/executionActions.ts`（现有执行事件处理链）
  - `docs/refactor/2026-02-25-hub-go-runtime-refactor-master-plan.md`（T-005：Execution* -> Run*）
- 实现说明:
  - shared-core 新增 `RunState`、`RunControlAction`、`RunEventType`、`RunEvent`、`ConversationStreamEvent`、`RunControlRequest/RunControlResponse` 类型，作为 run-centric 契约基线。
  - Desktop 新增 `runEventAdapter`，将 `RunEvent` 映射回现有 `ExecutionEvent`（`run_queued/start/output_delta/completed/failed/cancelled` -> execution 事件族），并保留对 legacy `ExecutionEvent` 的透传兼容。
  - `stream.ts` 的入口 normalize 改为统一走 `toExecutionEventFromStreamPayload`，实现 run/legacy 双语义兼容。
  - `sseClient.ts` 事件类型升级到 `ConversationStreamEvent`，并在 SSE message 层将 `lastEventId` 注入 `event_id`（当 payload 缺失时）以维持断点续传游标语义。
  - `services/index.ts` 的 `streamConversationEvents` 入参类型从 `unknown` 收敛到 `ConversationStreamEvent`。
- 运行说明:
  - `cd apps/desktop && pnpm vitest run src/modules/conversation/tests/run-event-adapter.spec.ts src/modules/conversation/tests/conversation-stream.spec.ts`
  - `cd apps/desktop && pnpm vitest run src/modules/conversation/tests`
  - `cd apps/desktop && pnpm lint`
  - `cd /Users/goya/Repo/Git/Goyais && pnpm --filter @goyais/shared-core build`
  - `cd /Users/goya/Repo/Git/Goyais && pnpm --filter @goyais/desktop test`
  - `cd /Users/goya/Repo/Git/Goyais && pnpm --filter @goyais/mobile lint && pnpm --filter @goyais/mobile test`
- 测试证据:
  - RED（TDD）:
    - `pnpm vitest run src/modules/conversation/tests/run-event-adapter.spec.ts` 首次失败（`runEventAdapter` 缺失，导入无法解析）。
  - GREEN:
    - `pnpm vitest run src/modules/conversation/tests/run-event-adapter.spec.ts src/modules/conversation/tests/conversation-stream.spec.ts` 通过（8 tests）。
    - `pnpm vitest run src/modules/conversation/tests` 通过（30 tests）。
    - `pnpm lint` 通过（TS 无报错）。
    - `pnpm --filter @goyais/shared-core build` 通过。
    - `pnpm --filter @goyais/mobile lint && pnpm --filter @goyais/mobile test` 通过（1 file / 2 tests）。
  - 回归:
    - `pnpm --filter @goyais/desktop test` 通过（37 files / 120 tests）。
- 输出/错误输出/退出码对等结果:
  - Desktop 会话流现可消费 run-centric SSE 事件（`type/session_id/run_id/...`），并在 store 层转换为既有 execution 事件模型，维持 UI 处理逻辑稳定。
  - 对 legacy `ExecutionEvent` SSE payload 保持兼容，避免联调窗口内双栈服务端导致的前端回退。
- 风险与回退:
  - 风险：当前 adapter 对 `run_output_delta` 的子类型判别基于 payload 特征（`diff/call_id/input/output`），若后端 payload 结构变化需同步规则。
  - 回退：移除 `runEventAdapter` 接线并将 `stream.ts` normalize 回退到 legacy `ExecutionEvent` 直通；shared-core 新增 `Run*` 类型可保留为前向兼容声明。
- 变更点清单:
  - 增加 shared-core `Run*` 类型族并对外导出。
  - 增加 Desktop run-event 兼容 adapter。
  - 将 SSE 客户端与会话服务类型升级到 `ConversationStreamEvent`。
  - 新增 run-event adapter 单测与 stream run 路由回归用例。
- 下一任务: T-006（在 Go CLI/TUI adapter 层接入 run-centric core，保持 UI shell 逻辑薄层）

### Iteration 2026-02-25 (T-005)

- Completed: Added shared-core Run* types and Desktop RunEvent adapter; Desktop stream pipeline now supports run-centric SSE semantics with legacy compatibility.
- In progress: None.
- Blockers: None.
- Risks: `run_output_delta` sub-type inference depends on payload shape contract; keep adapter and backend payload evolution synchronized.
- Next focus: Start T-006 Go CLI/TUI adapter integration over agentcore core flow.

### Task Delivery: T-006

- Task ID: T-006
- 目标与范围: 在 Go 端落地 `cmd/goyais-cli` 的 CLI/TUI 适配层（参数路由、交互 shell、事件渲染），并通过 `agentcore/runtime` 契约接入 core；保持 UI shell 逻辑薄层，不引入业务状态机逻辑到 UI。
- 修改文件清单:
  - `services/hub/internal/agentcore/runtime/engine.go`
  - `services/hub/cmd/goyais-cli/main.go`
  - `services/hub/cmd/goyais-cli/adapters/session_runner.go`
  - `services/hub/cmd/goyais-cli/adapters/session_runner_test.go`
  - `services/hub/cmd/goyais-cli/cli/options.go`
  - `services/hub/cmd/goyais-cli/cli/app.go`
  - `services/hub/cmd/goyais-cli/cli/app_test.go`
  - `services/hub/cmd/goyais-cli/tui/shell.go`
  - `services/hub/cmd/goyais-cli/tui/shell_test.go`
  - `services/hub/cmd/goyais-cli/tui/renderer.go`
  - `docs/refactor/2026-02-25-hub-go-runtime-development-plan.md`
- 行为基准证据（原仓源码/README/脚本/测试引用）:
  - `Kode-cli/cli.js`（`--help-lite`/`--version`/`--print`/`--cwd` 参数契约与 usage 文案）
  - `Kode-cli/tests/e2e/cli-smoke.test.ts`（wrapper smoke: help-lite/version/print 行为检查）
  - `Kode-cli/docs/develop/architecture.md`（UI 与核心执行逻辑分层约束）
  - `docs/refactor/2026-02-25-hub-go-runtime-refactor-master-plan.md`（Phase 3：`cmd/goyais-cli` 仅 adapter 层）
- 实现说明:
  - 新增 `agentcore/runtime.Engine` 接口与 `StartSessionRequest/SessionHandle/UserInput` 契约，统一 CLI/TUI 对 core 的调用边界。
  - 新增 `runtime.UnimplementedEngine` 作为接线占位，确保在未注入真实 engine 时给出显式错误而非隐式 panic。
  - 新增 `adapters.Runner`：负责 `config.Load -> StartSession -> Submit -> Subscribe -> Render` 编排，并在 run terminal 事件时停止本次请求流程。
  - 新增 `cli` 薄层：参数解析（`--help-lite/--help/--version/--print/--cwd`）、print 与 interactive 路由、统一 stderr/exit code 映射。
  - 新增 `tui` 薄层：`goyais> ` 交互循环（支持 `exit/quit`）与 run 事件文本渲染器（output delta / approval / failed / cancelled / completed）。
  - 新增 `cmd/goyais-cli/main.go`，将 `cli + tui + adapters + runtime` 进行最小装配。
- 运行说明:
  - `cd services/hub && go test -count=1 ./cmd/goyais-cli/...`
  - `cd services/hub && go test -count=1 ./internal/agentcore/...`
  - `cd services/hub && go test -count=1 ./...`
  - `cd services/hub && go build -o /tmp/goyais-cli ./cmd/goyais-cli`
  - `/tmp/goyais-cli --help-lite`
- 测试证据:
  - RED（TDD）:
    - `go test -count=1 ./cmd/goyais-cli/...` 首次失败（`internal/agentcore/runtime` 与 CLI/TUI/adapters 缺失）。
  - GREEN:
    - `go test -count=1 ./cmd/goyais-cli/...` 通过（adapters/cli/tui 测试全绿）。
    - `go test -count=1 ./internal/agentcore/...` 通过（含新增 runtime 包编译接线）。
  - 回归:
    - `go test -count=1 ./...` 通过（含 `internal/httpapi` 回归）。
- 输出/错误输出/退出码对等结果:
  - `/tmp/goyais-cli --help-lite`：stdout 输出 usage（与 `Kode-cli` wrapper help-lite 结构对齐），stderr 为空，退出码 `0`。
  - `/tmp/goyais-cli --help`：当前输出 runtime 未接线错误（`full help requires a configured runtime engine`），stderr 输出错误，退出码 `1`，与基线“full help 依赖 runtime，不可用时失败”语义对齐。
  - `/tmp/goyais-cli --version`：stdout 输出 `dev`（由 build-time 变量注入），stderr 为空，退出码 `0`。
  - `/tmp/goyais-cli --print "hello"`：当前因 `runtime.UnimplementedEngine` 返回 `error: start session: agentcore engine is not configured`，stderr 输出错误，退出码 `1`（符合“adapter 先行，engine 后续接线”阶段性行为）。
- 风险与回退:
  - 风险：真实 engine 尚未注入时，`--print`/交互 shell 无法完成实际 run，仅能验证 adapter 管线和错误语义。
  - 回退：删除 `services/hub/cmd/goyais-cli` 与 `internal/agentcore/runtime` 新增文件，并将任务状态回退为 `todo`。
- 变更点清单:
  - 增加 Go CLI/TUI adapter 目录与入口。
  - 增加 core runtime interface 契约，明确 UI->core 边界。
  - 增加 CLI/TUI/adapters 单测，固定 shell 薄层行为。
  - 固定 `--help-lite/--help/--version/--print` 的最小输出与 exit code 行为。
- 下一任务: T-007（一次性 cutover：移除 Worker/Kode-cli 并准备 release rollback package）

### Iteration 2026-02-25 (T-006)

- Completed: Implemented `cmd/goyais-cli` thin adapter stack (cli/tui/adapters) over `agentcore/runtime` contract with TDD coverage and full `services/hub` regression pass.
- In progress: None.
- Blockers: None.
- Risks: Runtime execution remains placeholder until a concrete `agentcore` engine implementation is injected into CLI wiring.
- Next focus: Start T-007 one-shot cutover and cleanup (Worker/Kode-cli removal + rollback package).

### Task Delivery: T-007

- Task ID: T-007
- 目标与范围: 执行一次性 cutover，移除仓库内 Worker 运行依赖与 Kode-cli 运行入口依赖，将 Desktop 本地 sidecar 收敛为 Hub-only，并交付可执行的 release rollback package（恢复 pre-T007 关键资产 + 验证脚本）。
- 修改文件清单:
  - `.github/workflows/ci.yml`
  - `.github/workflows/release.yml`
  - `.env.example`
  - `.gitignore`
  - `Makefile`
  - `README.md`
  - `README.zh-CN.md`
  - `apps/desktop/README.md`
  - `apps/desktop/src-tauri/src/sidecar.rs`
  - `apps/desktop/src-tauri/tauri.conf.json`
  - `docs/PRD.md`
  - `docs/release-checklist.md`
  - `docs/slides/slides.md`
  - `docs/refactor/2026-02-25-t007-rollback-package.md`
  - `package.json`
  - `scripts/dev/print_commands.sh`
  - `scripts/release/ensure-local-sidecars.sh`
  - `scripts/smoke/health_check.sh`
  - `scripts/release/rollback/restore-pre-t007.sh`
  - `scripts/release/rollback/verify-rollback.sh`
  - `scripts/release/rollback/rollback-to-stable.sh`
  - `scripts/release/build-worker-sidecar.sh`（删除）
  - `services/worker/`（删除）
- 行为基准证据（原仓源码/README/脚本/测试引用）:
  - `docs/refactor/2026-02-25-hub-go-runtime-refactor-master-plan.md`（Phase 4：移除 Worker/Kode-cli，保留 rollback 包）
  - `Makefile`/`scripts/smoke/health_check.sh`（cutover 前含 `dev-worker` 与 worker health 检查）
  - `apps/desktop/src-tauri/src/sidecar.rs` + `tauri.conf.json`（cutover 前 Desktop sidecar 依赖 `goyais-worker`）
  - `scripts/release/ensure-local-sidecars.sh` + `.github/workflows/release.yml`（cutover 前 release 需构建 worker sidecar）
- 实现说明:
  - CI/release 流程改为 Hub-only：移除 worker lint/test job、worker sidecar 构建步骤、Python/uv 依赖安装。
  - Desktop sidecar 改为仅管理 `goyais-hub`，删除 worker 子进程拉起、worker 健康探测与对应配置。
  - `make`/dev/smoke/release 脚本统一去除 worker 端口与 worker 运行路径，`health` smoke 仅验证 Hub + Desktop Web。
  - 文档与环境模板更新到 Hub-only 架构（README/PRD/release checklist/slides/.env.example）。
  - 开发工作区不再依赖本地 `Kode-cli/` 目录；根级脚本入口已去除对其默认调用。
  - 新增 rollback package（`scripts/release/rollback/*` + runbook）：
    - `restore-pre-t007.sh`：从稳定 ref 恢复 pre-T007 关键路径
    - `verify-rollback.sh`：验证 Hub health/auth/conversation/message 路径，并在可用时验证 worker health/token 测试
    - `rollback-to-stable.sh`：一键 restore+verify
  - 移除 `services/worker/` 与 `scripts/release/build-worker-sidecar.sh`。
- 运行说明:
  - `bash -n scripts/dev/print_commands.sh scripts/release/ensure-local-sidecars.sh scripts/smoke/health_check.sh scripts/release/rollback/restore-pre-t007.sh scripts/release/rollback/verify-rollback.sh scripts/release/rollback/rollback-to-stable.sh`
  - `cd apps/desktop/src-tauri && cargo fmt --check && cargo check`
  - `cd services/hub && go test -count=1 ./... && go vet ./...`
  - `cd /Users/goya/Repo/Git/Goyais && pnpm --filter @goyais/desktop lint && pnpm --filter @goyais/desktop test`
  - `cd /Users/goya/Repo/Git/Goyais && make health`
  - `cd /Users/goya/Repo/Git/Goyais && GOYAIS_FORCE_SIDECAR_REBUILD=1 bash scripts/release/ensure-local-sidecars.sh`
- 测试证据:
  - RED（TDD）:
    - `rg -n 'services/worker|goyais-worker|build-worker-sidecar|dev-worker|WORKER_PORT' ...` 初始扫描命中 CI/release/smoke/desktop sidecar 多处依赖，证明 cutover 前约束未满足。
    - `cargo fmt --check` 首次失败（`sidecar.rs` 需格式化），修复后转绿。
  - GREEN:
    - `bash -n` 覆盖所有改动脚本通过。
    - `cargo check` 通过（desktop tauri sidecar hub-only 接线可编译）。
    - `go test -count=1 ./...` 与 `go vet ./...` 通过（services/hub）。
    - `pnpm --filter @goyais/desktop lint` 与 `pnpm --filter @goyais/desktop test` 通过（37 files / 120 tests）。
    - `make health` 通过（Hub + Desktop smoke）。
    - `GOYAIS_FORCE_SIDECAR_REBUILD=1 bash scripts/release/ensure-local-sidecars.sh` 通过（仅构建 hub sidecar）。
  - 回归:
    - `rg` 复扫运行入口文件（Makefile/CI/release/smoke/README/tauri conf）已无 worker 运行依赖残留；仅 rollback package 保留恢复路径引用。
- 输出/错误输出/退出码对等结果:
  - `make health` 输出端口收敛为 `hub` + `desktop`，无 worker 启动阶段，退出码 `0`。
  - `scripts/release/ensure-local-sidecars.sh` 输出改为 `hub sidecar ready...`，不再要求 worker binary。
  - Desktop sidecar 启动日志语义改为 `sidecar runtime initialized (hub only)`，与 Hub-only cutover 目标一致。
  - rollback 一键入口：`scripts/release/rollback/rollback-to-stable.sh <stable-tag-or-sha>`。
- 风险与回退:
  - 风险：Hub 内仍保留部分 legacy `/internal/*` worker 协议路由用于兼容测试/历史语义；后续可单独清理。
  - 回退：执行 `scripts/release/rollback/rollback-to-stable.sh <stable-tag-or-sha>`，恢复 pre-T007 关键资产并自动跑验证脚本。
- 变更点清单:
  - 完成 Worker sidecar 构建链、CI 门禁、smoke 门禁的 Hub-only cutover。
  - 完成 Desktop sidecar hub-only 化。
  - 完成 `services/worker` 代码移除。
  - 交付 rollback package 与 runbook。
- 下一任务: 进入 Pre-release Global Acceptance Checklist（全项核验 + rollback drill 记录）。

### Iteration 2026-02-25 (T-007)

- Completed: One-shot cutover to Hub-only runtime path, removed worker service artifacts, and delivered executable rollback package with verification scripts.
- In progress: None.
- Blockers: None.
- Risks: Legacy worker protocol branches in Hub remain for compatibility and should be evaluated for follow-up cleanup.
- Next focus: Execute pre-release global acceptance checklist and archive T-007 rollback drill evidence.

### Iteration 2026-02-25 (Global Acceptance)

- Completed: Verified pre-release global acceptance gates (Hub API/SSE/control/approval tests, Desktop+Mobile compatibility tests, cross-target hub sidecar builds, and rollback drill in temporary clone).
- In progress: None.
- Blockers: None.
- Risks: Cross-platform distributable verification in this pass covers Hub sidecar multi-target build; full Tauri installer signing/notarization still depends on release CI matrix.
- Next focus: Prepare release signoff artifacts and execute final release owner Go/No-Go review.
