# Hub v1 全量重构方案（无兼容切换，执行基线）

- 日期：2026-03-04
- 范围：`services/hub`、`apps/desktop`、`packages/shared-core`（Mobile 不在本轮）
- 目标：将 Hub 从 `conversation/execution + v4 bridge` 迁移到 `session/run` 单内核架构；CLI/ACP/HTTP 全面切换到 v1 语义
- 文档角色：本文件是后续实施与验收的唯一架构基线；任务拆解见 `docs/refactor/hub/2026-03-04-hub-v1-refactor-task-plan-table.md`

---

## 0. 决策冻结（本轮不可变）

1. 交付范围：`Hub + Desktop` 同步重构。
2. 资源模型：采用纯 `Session/Run`，不再暴露 `Execution`。
3. HTTP 流式：以 `SSE` 为主，统一 `run_*` 事件词汇。
4. ACP：保留 JSON-RPC over stdio 传输，但协议方法集升级为 ACP v1。
5. CLI：采用新的 CLI v1 命令面，不保留旧命令语义兼容。
6. 数据层：采用统一仓储层（repository + sqlite），不再以 `AppState` map 作为主存储。
7. 数据迁移：允许破坏式重建 schema，不提供旧 schema 迁移脚本。
8. API 版本：OpenAPI `info.version` 升级为 `1.0.0`。
9. 术语：前后端与 UI 统一为 `Session`，不再使用 `Conversation` 作为主术语。

---

## 1. 目标与非目标

### 1.1 目标

1. Hub 运行链路单内核：CLI/ACP/HTTP 三端全部直接接 `core.Engine`。
2. 对外契约单词汇：`Session/Run/RunEvent`。
3. 去除过渡层：删除 `execution_runtime_router` / `execution_runtime_v4_bridge` / `runtimebridge`。
4. 占位目录补齐：`core/events`、`core/statemachine`、`policy/hookscope`、`policy/sandbox` 达到生产可用深度。
5. 运行态可持久化：Hub 重启后可查询 Session/Run/Event 历史。
6. Desktop 与共享类型同步切换，不依赖 execution 映射适配。

### 1.2 非目标

1. 不处理 Mobile 端同步改造。
2. 不保留旧 API 别名、旧字段回填、旧协议透传。
3. 不做旧数据迁移或兼容读取。

---

## 2. As-Is 核心问题

1. HTTP 层仍依赖 legacy execution domain（`executions`、`executionEvents`、`queue_state` 等）。
2. HTTP 通过 `v4 bridge` 进行 execution->run 映射与 shadow event 回灌，存在双语义。
3. `internal/agent` 中存在目录级占位模块，未接入主运行链。
4. `runtimebridge` 在 CLI/ACP 中保留旧 runtime domain 投影路径。
5. Desktop 仍以 `Conversation/Execution` 为主数据模型，依赖 run-to-execution 适配。

---

## 3. To-Be 架构

```text
Hub v1
  internal/agent
    core
      events/           # 事件词汇、typed payload、编码约束
      statemachine/     # Run 生命周期状态机
    runtime
      loop/             # 单会话单活跃 + FIFO 调度
      session/          # resume/fork/rewind/clear/handoff
      model/            # provider loop
      compaction/       # 上下文压缩
    tools
    policy
      hookscope/        # 多作用域 hook 策略解析
      sandbox/          # 沙箱决策与约束
    transport
      events/
      subscribers/
    adapters
      httpapi/
      cli/
      acp/

  internal/httpapi
    handlers -> services -> repositories

  sqlite
    sessions/runs/run_events/run_tasks/...（v1 schema）

Desktop
  SessionStore + RunStore + RunEventStream
```

关键原则：

1. 业务语义在 `agent/core + runtime` 定义一次。
2. 适配层只做协议映射，不持有业务状态机。
3. `AppState` 从“主业务存储”降级为“依赖容器 + 轻量运行缓存”。

---

## 4. 领域模型与状态机

### 4.1 领域对象

1. `Session`
- 字段：`id/project_id/workspace_id/name/default_mode/model_config_id/rule_ids/skill_ids/mcp_ids/active_run_id/created_at/updated_at`
- 含义：用户交互上下文容器与运行策略容器。

2. `Run`
- 字段：`id/session_id/message_id/state/mode/model_id/tokens_in/tokens_out/trace_id/created_at/updated_at`
- 含义：一次执行实体。

3. `RunEvent`
- 字段：`event_id/run_id/session_id/sequence/type/timestamp/payload_json`
- 词汇：`run_queued|run_started|run_output_delta|run_approval_needed|run_completed|run_failed|run_cancelled`

### 4.2 状态机

1. 状态：`queued/running/waiting_approval/waiting_user_input/completed/failed/cancelled`
2. 控制动作：`stop/approve/deny/resume/answer`
3. 约束：单 Session 同时只允许 1 个 `running or waiting_*` Run，其余排队 FIFO。

---

## 5. HTTP v1 契约重构

### 5.1 运行链路路径重构

| 旧路径 | 新路径 | 说明 |
|---|---|---|
| `GET/POST /v1/projects/{project_id}/conversations` | `GET/POST /v1/projects/{project_id}/sessions` | 主资源改名 |
| `GET/PATCH/DELETE /v1/conversations/{conversation_id}` | `GET/PATCH/DELETE /v1/sessions/{session_id}` | 主资源改名 |
| `POST /v1/conversations/{id}/input/submit` | `POST /v1/sessions/{id}/runs` | 提交执行 |
| `GET /v1/conversations/{id}/events` | `GET /v1/sessions/{id}/events` | SSE 事件流 |
| `POST /v1/conversations/{id}/stop` | `POST /v1/sessions/{id}/stop` | 停止活跃 Run |
| `POST /v1/conversations/{id}/rollback` | `POST /v1/sessions/{id}/rollback` | 变更回滚 |
| `GET/POST /v1/conversations/{id}/changeset*` | `GET/POST /v1/sessions/{id}/changeset*` | 变更集 |
| `GET /v1/executions` | `GET /v1/runs` | 列表资源改名 |
| 无 | `GET /v1/runs/{run_id}` | 新增 Run 详情 |

保留路径：

1. `/v1/runs/{run_id}/control`
2. `/v1/runs/{run_id}/graph`
3. `/v1/runs/{run_id}/tasks`
4. `/v1/runs/{run_id}/tasks/{task_id}`
5. `/v1/runs/{run_id}/tasks/{task_id}/control`
6. `/v1/hooks/executions/{run_id}`（后续语义可再评估是否重命名为 `/v1/hooks/runs/{run_id}`，本轮先保持）

### 5.2 SSE 事件协议（统一 run_*）

1. 只输出 `run_*` 词汇，不再输出 execution 词汇。
2. 事件 payload 使用 typed schema，对应 OpenAPI `RunEvent` 组件。
3. 支持 `Last-Event-ID` 与 cursor 补偿，保证序列单调。

---

## 6. ACP v1 协议重构（JSON-RPC over stdio）

### 6.1 方法集

1. `session.start`
2. `session.get`
3. `session.list`
4. `session.fork`
5. `session.rewind`
6. `session.clear`
7. `session.handoff`
8. `run.submit`
9. `run.control`
10. `stream.subscribe`
11. `stream.unsubscribe`

### 6.2 删除方法

1. `session/new`
2. `session/load`
3. `session/prompt`
4. `session/set_mode`
5. `session/cancel`

### 6.3 ACP 事件

1. `run_event`
2. `approval_needed`
3. `command_result`（仅在 CLI 命令桥接场景）

---

## 7. CLI v1 命令面重构

### 7.1 命令集合

1. `goyais-cli session start --cwd <path>`
2. `goyais-cli session list`
3. `goyais-cli session get <session_id>`
4. `goyais-cli run submit --session <id> --prompt "..."`
5. `goyais-cli run control --run <id> --action <stop|approve|deny|resume|answer>`
6. `goyais-cli run stream --session <id> --cursor <n> --output-format <text|json|stream-json>`

### 7.2 删除对象

1. `cmd/goyais-cli/adapters/v4_runner.go`
2. 与 `V4Runner` 命名相关的入口与测试。

---

## 8. 数据层重构（Repository First）

### 8.1 策略

1. `AppState` 不再维护 Session/Run/Event 主 map。
2. 全部领域读写经 `services` -> `repositories`。
3. sqlite 为主存储，内存仅做短生命周期订阅缓存。

### 8.2 v1 schema（破坏式重建）

核心表建议：

1. `sessions`
2. `session_messages`
3. `runs`
4. `run_events`
5. `run_tasks`
6. `run_diffs`
7. `session_snapshots`
8. `hook_policies`
9. `hook_execution_records`

说明：

1. 保留 workspace/auth/project/resource/admin 相关表，但统一经 repository 访问。
2. 旧 `conversations/executions/execution_events` 不再作为主写路径。

---

## 9. 占位模块补齐（生产可用）

### 9.1 `core/events`

1. 下沉事件 spec 与 payload 绑定注册。
2. 提供事件编码/解码与 schema 校验工具。
3. 统一供 runtime/transport/adapters 引用。

### 9.2 `core/statemachine`

1. 下沉状态迁移矩阵与动作映射。
2. 暴露可测试 transition API。
3. runtime/loop 只消费该状态机，不自定义状态跳转。

### 9.3 `policy/hookscope`

1. 解析 global/workspace/project/session/plugin 多作用域规则。
2. 冲突优先级：`deny > ask > allow`，同层按作用域与匹配精度排序。
3. 输出命中轨迹用于审计。

### 9.4 `policy/sandbox`

1. 定义文件边界、命令执行、网络访问三类沙箱策略。
2. 对接 `tools/executor` 前置决策，输出 allow/ask/deny。
3. 写入统一审计字段（tool、path、reason、matched_rule）。

---

## 10. 传输与适配重构

### 10.1 HTTP Adapter

1. 删除 runtime bridge 与 shadow event。
2. HTTP handler 直接通过 runtime service 调用 Engine + repository。
3. SSE 直接订阅 Session run stream。

### 10.2 ACP Adapter

1. server 仅负责 ACP v1 方法映射。
2. bridge 层不再复用 CLI runner 的 execution 语义。
3. stream 采用统一 subscriber manager。

### 10.3 CLI Adapter

1. 仅保留 v1 命令转换与输出 writer。
2. 删除 runtime domain projector 与 legacy event bridge。

---

## 11. 删除清单

必须删除：

1. `services/hub/internal/httpapi/execution_runtime_router.go`
2. `services/hub/internal/httpapi/execution_runtime_v4_bridge.go`
3. `services/hub/internal/agent/adapters/runtimebridge/*`
4. `services/hub/cmd/goyais-cli/adapters/v4_runner.go`
5. 所有依赖 execution shadow/v4 runtime mode 的测试与脚本。

必须清零的代码关键词：

1. `executionRuntime`
2. `v4Service`
3. `route_v4`
4. `legacy_execution_id`
5. `runEventAdapter` 中 execution 映射逻辑

---

## 12. 验证与门禁

### 12.1 Hub

1. `cd services/hub && go test ./... && go vet ./...`
2. `scripts/refactor/gate-check.sh --strict`（需扩展到 Hub v1 词汇门禁）

### 12.2 合约

1. `pnpm contracts:generate`
2. `pnpm contracts:check`

### 12.3 Desktop

1. `pnpm lint`
2. `pnpm test`
3. `pnpm test:strict`
4. `pnpm e2e:smoke`

### 12.4 发布健康检查

1. `make health`

---

## 13. 风险与缓解

1. 风险：全链路术语切换导致前后端错配。
- 缓解：OpenAPI 先行锁定 + 共享类型先切换 + Desktop 同步提交。

2. 风险：破坏式 schema 重建影响开发数据。
- 缓解：在重构期间明确 DB 版本边界，启动脚本自动重建并提示。

3. 风险：大规模移除 legacy 导致回归范围大。
- 缓解：按 phase 主干提交，每阶段保证可运行且测试全绿。

4. 风险：SSE 事件词汇切换引发 UI 状态机抖动。
- 缓解：Desktop store 先完成 run-only 状态机，再切服务端事件输出。

---

## 14. 交付物清单

1. Hub v1 架构实现（session/run 单内核链路）。
2. OpenAPI 1.0.0 与 `packages/shared-core/src/generated/openapi.ts` 同步。
3. Desktop Session/Run 新模型与 UI 术语切换。
4. ACP v1 协议实现与测试。
5. CLI v1 命令面实现与测试。
6. 清理 legacy/v4 过渡代码并通过全量门禁。
