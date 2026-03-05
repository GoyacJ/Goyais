# Session/Run 语义统一重构任务排期（6 周）

- 日期：2026-03-05
- 周期：6 周
- 关联主计划：`./2026-03-05-session-run-unification-master-plan.md`
- 关联风险台账：`./2026-03-05-session-run-unification-risk-register.md`

---

## 1. 排期总览

| Week | 主题 | 关键目标 | 依赖 |
|---|---|---|---|
| Week 1 | 契约冻结与类型先行 | OpenAPI + shared-core 收敛为 session/run | 无 |
| Week 2 | Hub 入口与路由语义收口 | 路由/handler 参数与 hooks 路径完成切换 | Week 1 |
| Week 3 | Hub 存储与权限语义收口 | 去版本后缀 + DB 命名收口 + 权限键切换 | Week 2 |
| Week 4 | Desktop 全量重命名 | 目录/符号/i18n 全量切换为 session/run | Week 1-3 |
| Week 5 | 兼容清零与门禁升级 | legacy/compat/fallback 清零 + 新门禁启用 | Week 2-4 |
| Week 6 | 全链路回归与发布收口 | 跨栈验证、风险清零、closure 报告 | Week 1-5 |

---

## 2. 周任务明细（编号、依赖、交付物、验收、阻塞条件）

### Week 1：契约冻结与类型先行

| 任务 ID | 任务 | 依赖 | 交付物 | 验收命令 | 阻塞条件 |
|---|---|---|---|---|---|
| W1-T1 | OpenAPI 切换为 Session/Run 主语义 | 无 | `packages/contracts/openapi.yaml`（移除 Conversation/Execution alias schema） | `pnpm contracts:generate && pnpm contracts:check` | 旧路径与旧 schema 仍被依赖 |
| W1-T2 | 字段名统一为 `session_id/run_id/active_run_id` | W1-T1 | OpenAPI schema 字段、示例与注释 | 同上 | Hub/Desktop 尚未准备字段切换 |
| W1-T3 | shared-core 移除 deprecated/transitional alias | W1-T1,W1-T2 | `packages/shared-core/src/api-*` + regenerated types | 同上 | 仍有上游代码依赖旧类型别名 |
| W1-T4 | 合约冻结基线发布 | W1-T1~T3 | 契约冻结记录（本文件周更） | `pnpm contracts:check` | 合约未形成唯一版本 |

Week 1 收口标准：
1. OpenAPI 与 generated types 不再以 Conversation/Execution 作为主模型。
2. shared-core 不再保留旧语义兼容别名。
3. 契约字段命名统一，无双字段并存。

### Week 2：Hub 入口与路由语义收口

| 任务 ID | 任务 | 依赖 | 交付物 | 验收命令 | 阻塞条件 |
|---|---|---|---|---|---|
| W2-T1 | runtime/controlplane/integration 路由参数统一 session/run | Week 1 | `services/hub/internal/*/routes/*.go` | `cd services/hub && go test ./... && go vet ./...` | 仍存在旧 path/query 参数 |
| W2-T2 | 删除 `conversation_id/execution_id` 入参回退逻辑 | W2-T1 | handler/path/query 解析实现 | 同上 | 旧客户端仍强依赖旧参数 |
| W2-T3 | hooks 路径语义切换到 run | W2-T1 | hooks 路径、handler、测试 | 同上 | hooks 调用链未同步 |
| W2-T4 | Hub 入口语义收口回归 | W2-T1~T3 | 路由/handler 回归测试通过 | 同上 | 非 runtime handler 行为漂移 |

Week 2 收口标准：
1. Hub 入参不再接受旧语义字段回退。
2. hooks 接口与 payload 全部 run 语义。
3. Hub 编译、测试、vet 全通过。

### Week 3：Hub 存储层与权限语义收口

| 任务 ID | 任务 | 依赖 | 交付物 | 验收命令 | 阻塞条件 |
|---|---|---|---|---|---|
| W3-T1 | 去除内部 `*_v1` 文件名与符号命名 | Week 2 | `services/hub/internal/httpapi/*` 重命名与引用更新 | `cd services/hub && go test ./... && go vet ./...` | 大规模重命名引发引用断裂 |
| W3-T2 | DB 表/索引去版本后缀（破坏式重建） | W3-T1 | sqlite schema 与仓储层命名更新 | 同上 | 本地旧库文件干扰初始化 |
| W3-T3 | 权限键切换到 `session.*`/`run.control` | W3-T1 | authz defaults/engine/audit 键名统一 | 同上 | 上游 UI 权限显示未同步 |
| W3-T4 | 语义 grep 清零（Hub 范围） | W3-T1~T3 | 审计脚本与结果记录 | `rg -n` 审计命令（见 Week 5） | 白名单边界不清导致误判 |

Week 3 收口标准：
1. Hub 主链路无内部版本后缀命名。
2. 权限与审计键名无旧语义残留。
3. DB schema 仅新命名，且可稳定初始化。

### Week 4：Desktop 全量重命名（目录 + 符号 + i18n）

| 任务 ID | 任务 | 依赖 | 交付物 | 验收命令 | 阻塞条件 |
|---|---|---|---|---|---|
| W4-T1 | `modules/conversation` 迁移到 `modules/session` | Week 1-3 | 目录重命名与引用修复 | `pnpm lint && pnpm test` | 导入路径大面积失效 |
| W4-T2 | `execution*` 文件与符号迁移到 `run*` | W4-T1 | store/service/view/test 命名统一 | `pnpm test:strict` | 事件映射与状态机不一致 |
| W4-T3 | i18n key `conversation.*` -> `session.*` | W4-T1 | `messages.*` 与调用点同步 | `pnpm test` | 翻译 key 漏改导致运行时缺词 |
| W4-T4 | 前端 E2E 与主链路回归 | W4-T1~T3 | 会话核心交互回归通过 | `pnpm e2e:smoke` | SSE 与界面状态不一致 |

Week 4 收口标准：
1. Desktop 无旧目录与旧主符号命名。
2. i18n key 与代码引用全部对齐。
3. lint/test/strict/e2e 全通过。

### Week 5：兼容实现清零与门禁脚本升级

| 任务 ID | 任务 | 依赖 | 交付物 | 验收命令 | 阻塞条件 |
|---|---|---|---|---|---|
| W5-T1 | 清理 legacy/compat/fallback 运行时路径 | Week 2-4 | 首方主链路兼容代码删除 | `cd services/hub && go test ./...` + `pnpm test` | 误删必要分支引发行为回归 |
| W5-T2 | 重写并启用语义门禁脚本 | W5-T1 | 新门禁规则与 CI 接入 | 门禁脚本执行通过 | 白名单策略缺失导致误拦截 |
| W5-T3 | 更新 docs/slides/README 术语 | W5-T1 | 文档统一到新语义 | `pnpm docs:build && pnpm slides:build` | 文档构建链断裂 |
| W5-T4 | 全仓 grep 审计收口 | W5-T1~T3 | 审计报告（本文件周更） | 见下方“统一审计命令” | 审计词表不完整 |

Week 5 收口标准：
1. 兼容路径清零，无双轨行为。
2. 门禁脚本具备防回流能力并在 CI 生效。
3. 文档与演示材料术语一致。

### Week 6：全链路回归与发布收口

| 任务 ID | 任务 | 依赖 | 交付物 | 验收命令 | 阻塞条件 |
|---|---|---|---|---|---|
| W6-T1 | 跨栈验证矩阵执行 | Week 1-5 | 全命令结果记录 | 全矩阵命令（见主计划） | 某栈仍存在语义残留 |
| W6-T2 | 风险复盘与遗留项清零 | W6-T1 | `risk-register` 状态归档 | 周更检查通过 | 高风险未闭环 |
| W6-T3 | 最终 closure 报告（新文档体系） | W6-T1,W6-T2 | `docs/refactor` closure 文档（新命名） | `make health` | 发布敏感检查未通过 |

Week 6 收口标准：
1. 全命令矩阵通过。
2. 风险台账高风险项全部关闭或有可执行回滚预案。
3. 形成可复审 closure 报告并归档。

---

## 3. 测试场景（强制覆盖）

1. 会话创建、提交、控制、事件流、回滚、changeset 全流程。
2. OpenAPI 与 generated types 一致性。
3. 权限与审计键名全量切换验证。
4. 前端核心交互回归（发送、停止、审批、trace、inspector、导出）。
5. 语义清零审计：旧词汇、兼容关键词、内部版本后缀。

---

## 4. 统一审计命令（Week 5+）

1. `rg -n "\\b(conversation|execution|Conversation|Execution)\\b" services/hub apps/desktop/src packages/shared-core/src packages/contracts`
2. `rg -n "\\b(v1|v2|v3|v4|V1|V2|V3|V4)\\b" services/hub apps/desktop/src packages/shared-core/src packages/contracts`
3. `rg -n "\\blegacy\\w*|\\bcompat\\w*|fallback|alias" services/hub apps/desktop/src packages/shared-core/src packages/contracts`

说明：
1. 审计结果必须结合白名单（第三方/平台强制命名）解释。
2. 审计输出与处理结论需写入本文件“每周状态”。

---

## 5. 每周状态维护模板（固定使用）

### Week N 状态

- 完成任务：`Wn-Tx`
- 未完成任务：`Wn-Tx`
- 阻塞项与责任人：
- 本周风险变化：
- 下周计划：
- 验收命令与结果摘要：

---

## 6. 当前实施进度（2026-03-05）

### 6.1 任务状态总览

| 任务 ID | 当前状态 | 进展说明 | 关键证据 |
|---|---|---|---|
| W1-T1 | In Progress | OpenAPI 已将主 schema 收敛为 `Session/Run` 主定义，并移除 `Conversation/Execution` 及相关 alias schema（含 change-set/export/event-batch 兼容壳） | `contracts:generate` / `contracts:check` 持续通过，schema 命中仅剩业务字段级旧语义 |
| W1-T2 | In Progress | 已完成 `hooks` 与 `workspace-status` 的 `session_id/run_id` 及 `session_status` 子域切换；并完成 Hub 对应结构体字段名收敛（`SessionID/SessionStatus`）；核心会话/执行链路尚未全量切换 | hooks payload/模型字段、workspace status payload/模型字段均已切到 session 语义 |
| W1-T3 | In Progress | shared-core/desktop 类型去别名持续推进：`SessionDetailResponse` 去除 `conversation/executions` 兼容镜像，组件/SSE/快照链路改用 `Session*` / `SessionStreamEvent` 主类型 | `api-project.ts`、`modules/conversation/services/index.ts`、`store/state.ts`、`shared/services/sseClient.ts`、`conversationSnapshots.ts` |
| W1-T4 | In Progress | 契约冻结进入收口阶段，当前以 `Session/Run` 主契约验证为准 | `pnpm contracts:generate && pnpm contracts:check` 连续通过 |
| W2-T1 | In Progress | controlplane hooks 路由已切换为 run 语义，其他入口仍需继续收口 | `/v1/hooks/runs/{run_id}` 已注册 |
| W2-T2 | In Progress | 已移除 runtime session path/query 对 `conversation_id` 的回退解析，并同步清理 hooks/workspace-status 子域内部字段名混用 | `runtime_session_path.go` 仅接受 `session_id`；hooks/workspace-status Hub 模型字段已收敛到 `SessionID/SessionStatus` |
| W2-T3 | Done | hooks 路径由 `hooks/executions` 完成切换到 `hooks/runs`，并联动 OpenAPI/Hub/shared-core/tests | 全仓 grep `hooks/executions` 命中 0 |
| W2-T4 | In Progress | 本轮改动已完成 Hub hooks/workspace-status 子域回归，且全 Hub 回归门禁持续通过；全域收口仍未完成 | `go test ./internal/httpapi -run 'Hooks|WorkspaceStatus'`、`go test ./...`、`go vet ./...` 通过 |

### 6.2 本轮已完成增量（2026-03-05）

1. hooks 路径收敛：`/v1/hooks/executions/{run_id}` -> `/v1/hooks/runs/{run_id}`（contracts + hub + shared-core + tests）。
2. hooks 字段收敛：`conversation_id` -> `session_id`（policy/upsert/error-details/records 对外契约）。
3. workspace-status 字段收敛：响应字段与 desktop 消费方统一使用 `session_id`。
4. Hub 入参回退收敛：runtime session 解析已移除 `conversation_id` fallback，仅保留 `session_id`。
5. workspace-status 状态字段收敛：`conversation_status` -> `session_status`（contracts + hub + shared-core + desktop）。
6. OpenAPI 主 schema 收敛：`Session/Run` 成为主定义，并删除 `Conversation/Execution` 及相关 alias schema（并联动 contract types regenerate）。
7. Hub 内部命名收敛：hooks/workspace-status 结构体字段由旧语义映射名进一步收敛为 `SessionID/SessionStatus`，并联动快照持久化与测试夹具同步。
8. W1-T3 启动：shared-core `api-project.ts` 将 `SessionDetailResponse` 设为主结构并将旧字段降级为可选兼容，`ConversationDetailResponse` 收敛为别名；Desktop conversation 核心类型注解迁移到 `Session/Run` 主类型并保留旧 payload 兼容归一化。
9. W1-T3 持续推进：Desktop `modules/project` 子域服务层与 store 层类型注解收敛到 `Session` 主类型，减少 `Conversation` 类型别名扩散。
10. W1-T3 持续推进：Desktop `modules/project` 子域内部调用链切换到 `listSessions/createSession/patchSession/removeSession/exportSessionMarkdown`，并保留 conversation 命名服务函数作为兼容壳。
11. W1-T3 持续推进：Desktop `modules/conversation` 视图与事件去重子域完成类型注解收敛（`Conversation/Execution/ExecutionEvent` -> `Session/Run/RunLifecycleEvent`），保持运行时字段与行为不变。
12. W1-T3 持续推进：Desktop `modules/conversation/store` 核心执行编排子域完成类型注解收敛（`executionActions/executionEventHandlers/events` 切换到 `Session/SessionMessage/RunLifecycleEvent`），保留函数命名与字段语义兼容。
13. W1-T3 持续推进：Desktop `modules/conversation/trace + store/stream` 子域完成类型注解收敛（`Execution/ExecutionEvent` -> `Run/RunLifecycleEvent`），保留 normalize/build 系列函数命名以避免调用面震荡。
14. W1-T3 持续推进：Desktop `modules/conversation/store/state` 与 `views/useMainScreenController` 补齐类型收敛（`Conversation` 参数类型改 `Session`，`ExecutionEvent` 注解改 `RunLifecycleEvent`），保持函数命名与数据字段兼容。
15. W1-T3 持续推进：Desktop `modules/conversation/components` 组件层类型注解收敛（`MainInspectorPanel/MainSidebarPanel` 由 `Conversation/Execution/ExecutionEvent` 切换至 `Session/Run/RunLifecycleEvent`），仅调整类型签名不改交互行为。
16. W1-T3 持续推进：Desktop 测试与辅助视图类型注解继续收敛（`useRunTraceState` 与 `main-screen-actions/execution-merge/run-trace-state` 测试统一为 `Session/Run/SessionMessage`），断言逻辑不变。
17. W1-T3 持续推进：Desktop `modules/conversation` 内部 runtime 类型别名使用收敛（`ConversationRuntime` 注解统一替换为 `SessionRuntime`），保持 store 对外兼容导出不变。
18. W1-T3 持续推进：Desktop 核心会话测试类型注解继续收敛（`conversation/conversation-hydration/conversation-race/conversation-run-tasks-actions` 测试中的 `Conversation` 切换至 `Session`），仅类型替换不改断言。
19. W1-T3 持续推进：Desktop trace 展示层命名收敛（统一使用 `RunTraceViewModel/RunTraceStep/buildRunTraceViewModels` 主符号，组件与 controller 全量切换到 run 命名）。
20. W1-T3 持续推进：Desktop `modules/project/services` 清理会话旧别名导出（移除 `listConversations/createConversation/patchConversation/renameConversation/removeConversation/exportConversationMarkdown`），`session` 命名成为该子域唯一服务接口，并同步更新 project store 测试 mock。
21. W1-T3 持续推进：Desktop trace 状态与展示链路进一步收敛到 run 命名（`buildRunTraceViewModelData/buildRunTraceViewModels/useRunTraceState` 成为主符号，移除 `processTrace` 子域 `ExecutionTrace*` 兼容导出，联动 controller 与测试更新）。
22. W1-T3 持续推进：Desktop trace 状态文件完成命名收口（统一使用 `useRunTraceState.ts`），controller 返回对象统一对外为 `selectedRunTrace`。
23. W1-T3 持续推进：Desktop trace 组件文件级命名收口（统一使用 `RunTraceBlock.{vue,css}`），并同步 `MainConversationPanel` 内部组件引用与局部类型命名。
24. W1-T3 持续推进：Desktop trace 测试文件级命名收口，统一使用 `run-trace-state.spec.ts`，并将测试描述文案同步到 run 语义。
25. W1-T3 持续推进：Desktop trace 测试文件继续收口，统一使用 `run-trace.spec.ts`，并将测试套件标题同步为 run 语义。
26. W1-T3 持续推进：Desktop trace 子目录测试命名收口，统一使用 `tests/trace/run-present.spec.ts`，并将测试套件标题同步为 run 语义。
27. W1-T3 持续推进：Desktop trace 子目录测试继续收口，统一使用 `tests/trace/run-normalize.spec.ts`，并将测试套件标题同步为 run 语义。
28. W1-T3 持续推进：Desktop trace 子目录测试完成收口，统一使用 `tests/trace/run-summarize.spec.ts`，并将测试套件标题同步为 run 语义。
29. W1-T1/W1-T3 推进：OpenAPI 删除 `Conversation/Execution/CreateConversationRequest/UpdateConversationRequest/ConversationChangeSet/ExecutionFilesExportResponse/ExecutionEventBatchRequest` alias schema，并将事件 schema 主名切换到 `RunLifecycleEvent`。
30. W1-T3 推进：shared-core `api-project.ts` 移除 `Conversation*/Execution*` 类型别名与 `SessionDetailResponse` 兼容镜像字段，`SessionSnapshot.messages` 收敛为 `SessionMessage[]`。
31. W1-T3 推进：Desktop 契约消费链路去别名（`sseClient` 切换到 `SessionStreamEvent`，`MainConversationPanel` 与 `conversationSnapshots` 改用 `SessionMessage/SessionSnapshot`，`conversation services/state` 仅消费 `session/runs` 详情结构）。
32. W1-T3 推进：Desktop hydration 相关测试夹具改用 `session/runs` 字段，消除 `conversation/executions` 兼容输入依赖。
33. W1-T2/W2-T4 推进：Hub `internal/httpapi` 对外 JSON 字段继续收敛（`active_execution_id`/`conversation_id`/`execution_id`/`execution_ids` -> `active_run_id`/`session_id`/`run_id`/`run_ids`），并同步标准错误详情键。
34. W1-T1/W1-T3 推进：OpenAPI 与 shared-core 字段级语义继续收敛（`Session`/`Run`/`RunLifecycleEvent`/`SessionChangeSet` 等模型字段统一使用 `session_id/run_id` 族）。
35. W1-T3/W4-T2 推进：Desktop `modules/conversation` 消费链路字段统一切换到 `session_id/run_id/active_run_id/run_ids`，SSE、trace、hydration 与测试夹具同步收口。
36. W4-T3 推进：Desktop i18n 用户可见文案从 “execution” 向 “run” 收敛（trace/inspector/settings 关键文案）。

### 6.3 最新审计快照（2026-03-05）

1. `rg -n "\\b(conversation|execution|Conversation|Execution)\\b" services/hub apps/desktop/src packages/shared-core/src packages/contracts` -> `2403`（较上一轮 `2422` 下降 `19`）。
2. `rg -n "\\b(v1|v2|v3|v4|V1|V2|V3|V4)\\b" services/hub apps/desktop/src packages/shared-core/src packages/contracts` -> `812`（含 `/v1` 路径与允许白名单）。
3. `rg -n "\\blegacy\\w*|\\bcompat\\w*|fallback|alias" services/hub apps/desktop/src packages/shared-core/src packages/contracts` -> `370`。
4. `rg -n "hooks/executions" services/hub apps/desktop/src packages/shared-core/src packages/contracts` -> `0`（代码子域已清零，文档中保留历史描述不计入）。

### 6.4 验收命令与结果摘要（本轮）

1. `pnpm contracts:generate` ✅
2. `pnpm contracts:check` ✅
3. `cd services/hub && go test ./...` ✅
4. `cd services/hub && go vet ./...` ✅
5. `pnpm lint` ✅
6. `pnpm test` ✅
7. `cd services/hub && go test ./internal/httpapi -run 'Hooks|WorkspaceStatus' -count=1` ✅
8. `pnpm --filter @goyais/desktop exec vitest run src/shared/stores/workspaceStatusStore.spec.ts` ✅
9. `pnpm --filter @goyais/desktop exec vitest run src/modules/conversation/tests/execution-merge.spec.ts src/modules/conversation/tests/conversation-hydration.spec.ts src/modules/conversation/tests/conversation-run-tasks-service.spec.ts` ✅
10. `pnpm --filter @goyais/shared-core build` ✅
11. `pnpm --filter @goyais/desktop exec vitest run src/modules/project/store/project-store.spec.ts` ✅
12. `pnpm --filter @goyais/desktop exec vitest run src/modules/conversation/tests/run-trace-state.spec.ts src/modules/conversation/tests/conversation.spec.ts` ✅
13. `pnpm --filter @goyais/desktop exec vitest run src/modules/conversation/tests/running-actions.spec.ts src/modules/conversation/tests/run-trace.spec.ts src/modules/conversation/tests/conversation-token-usage.spec.ts src/modules/conversation/tests/use-queue-messages-view.spec.ts src/modules/conversation/tests/main-screen-actions.spec.ts` ✅
14. `pnpm --filter @goyais/desktop exec vitest run src/modules/conversation/tests/conversation.spec.ts src/modules/conversation/tests/conversation-stream.spec.ts src/modules/conversation/tests/conversation-race.spec.ts src/modules/conversation/tests/main-screen-actions.spec.ts src/modules/conversation/tests/conversation-run-tasks-actions.spec.ts` ✅
15. `pnpm --filter @goyais/desktop exec vitest run src/modules/conversation/tests/trace/run-normalize.spec.ts src/modules/conversation/tests/trace/run-present.spec.ts src/modules/conversation/tests/run-trace.spec.ts src/modules/conversation/tests/running-actions.spec.ts src/modules/conversation/tests/conversation-token-usage.spec.ts src/modules/conversation/tests/conversation-stream.spec.ts` ✅
16. `pnpm --filter @goyais/desktop exec vitest run src/modules/conversation/tests/main-screen-controller.spec.ts src/modules/conversation/tests/conversation.spec.ts` ✅
17. `pnpm --filter @goyais/desktop exec vitest run src/modules/conversation/components/main-sidebar-panel.spec.ts src/modules/conversation/tests/main-inspector-run-tasks.spec.ts src/modules/conversation/tests/conversation.spec.ts` ✅
18. `pnpm --filter @goyais/desktop exec vitest run src/modules/conversation/tests/main-screen-actions.spec.ts src/modules/conversation/tests/execution-merge.spec.ts src/modules/conversation/tests/run-trace-state.spec.ts` ✅
19. `pnpm --filter @goyais/desktop exec vitest run src/modules/conversation/tests/main-screen-actions.spec.ts src/modules/conversation/tests/use-queue-messages-view.spec.ts src/modules/conversation/tests/conversation-token-usage.spec.ts src/modules/conversation/tests/conversation.spec.ts` ✅
20. `pnpm --filter @goyais/desktop exec vitest run src/modules/conversation/tests/conversation.spec.ts src/modules/conversation/tests/conversation-hydration.spec.ts src/modules/conversation/tests/conversation-race.spec.ts src/modules/conversation/tests/conversation-run-tasks-actions.spec.ts` ✅
21. `pnpm --filter @goyais/desktop exec vitest run src/modules/conversation/tests/run-trace-state.spec.ts src/modules/conversation/tests/run-trace.spec.ts src/modules/conversation/tests/main-inspector-run-tasks.spec.ts src/modules/conversation/tests/conversation.spec.ts` ✅
22. `pnpm --filter @goyais/desktop exec vitest run src/modules/project/store/project-store.spec.ts` ✅
23. `pnpm --filter @goyais/desktop build` ✅
24. `pnpm lint` ✅
25. `pnpm test` ✅
26. `pnpm --filter @goyais/desktop exec vitest run src/modules/conversation/tests/run-trace.spec.ts src/modules/conversation/tests/trace/run-present.spec.ts src/modules/conversation/tests/run-trace-state.spec.ts src/modules/conversation/tests/main-screen-controller.spec.ts` ✅
27. `pnpm --filter @goyais/desktop build` ✅
28. `pnpm lint` ✅
29. `pnpm test` ✅
30. `pnpm --filter @goyais/desktop exec vitest run src/modules/conversation/tests/run-trace-state.spec.ts src/modules/conversation/tests/main-screen-controller.spec.ts src/modules/conversation/tests/run-trace.spec.ts src/modules/conversation/tests/trace/run-present.spec.ts` ✅
31. `pnpm --filter @goyais/desktop build` ✅
32. `pnpm lint` ✅
33. `pnpm test` ✅
34. `pnpm --filter @goyais/desktop exec vitest run src/modules/conversation/components/main-sidebar-panel.spec.ts src/modules/conversation/tests/main-screen-actions.spec.ts src/modules/conversation/tests/conversation.spec.ts src/modules/conversation/tests/run-trace.spec.ts` ✅
35. `pnpm --filter @goyais/desktop build` ✅
36. `pnpm lint` ✅
37. `pnpm test` ✅
38. `pnpm --filter @goyais/desktop exec vitest run src/modules/conversation/tests/run-trace-state.spec.ts src/modules/conversation/tests/main-screen-controller.spec.ts src/modules/conversation/tests/run-trace.spec.ts src/modules/conversation/tests/trace/run-present.spec.ts` ✅
39. `pnpm --filter @goyais/desktop build` ✅
40. `pnpm lint` ✅
41. `pnpm test` ✅
42. `pnpm --filter @goyais/desktop exec vitest run src/modules/conversation/tests/run-trace.spec.ts src/modules/conversation/tests/run-trace-state.spec.ts` ✅
43. `pnpm lint` ✅
44. `pnpm test` ✅
45. `pnpm --filter @goyais/desktop exec vitest run src/modules/conversation/tests/trace/run-present.spec.ts src/modules/conversation/tests/run-trace.spec.ts src/modules/conversation/tests/run-trace-state.spec.ts` ✅
46. `pnpm lint` ✅
47. `pnpm test` ✅
48. `pnpm --filter @goyais/desktop exec vitest run src/modules/conversation/tests/trace/run-normalize.spec.ts src/modules/conversation/tests/trace/run-present.spec.ts src/modules/conversation/tests/run-trace.spec.ts src/modules/conversation/tests/run-trace-state.spec.ts` ✅
49. `pnpm lint` ✅
50. `pnpm test` ✅
51. `pnpm --filter @goyais/desktop exec vitest run src/modules/conversation/tests/trace/run-summarize.spec.ts src/modules/conversation/tests/trace/run-normalize.spec.ts src/modules/conversation/tests/trace/run-present.spec.ts src/modules/conversation/tests/run-trace.spec.ts src/modules/conversation/tests/run-trace-state.spec.ts` ✅
52. `pnpm lint` ✅
53. `pnpm test` ✅
54. `pnpm contracts:generate` ✅
55. `pnpm contracts:check` ✅
56. `pnpm --filter @goyais/shared-core build` ✅
57. `pnpm lint` ✅
58. `pnpm test` ✅
59. `cd services/hub && go test ./... && go vet ./...` ✅
60. `scripts/refactor/gate-check.sh` ✅
61. `pnpm contracts:generate && pnpm contracts:check` ✅
62. `cd services/hub && go test ./... && go vet ./...` ✅
63. `pnpm lint && pnpm test` ✅
64. `pnpm --filter @goyais/shared-core build` ✅

### 6.5 Week 3 Hub 先行批次增量（2026-03-05）

1. W3-T1 推进：Hub runtime 仓储命名收口，runtime 仓储装配命名收敛为 `RuntimeRepositorySet/NewSQLiteRuntimeRepositorySet`，并联动查询服务与状态同步链路。
2. W3-T1 推进：Hub runtime 日志文案去版本后缀，runtime fallback 日志文案去版本后缀并统一为 runtime 语义。
3. W3-T2/W3-T3 推进：Hub 权限键完成切换，`conversation.read/conversation.write/execution.control` 收敛为 `session.read/session.write/run.control`，并同步 `authorization_engine`、`authz_defaults`、角色默认权限、权限字典与 handler 审计键。
4. W3-T2/W3-T3 推进：Desktop 权限消费侧最小联动完成，角色权限展示与 workspace 测试断言同步改为 `session.*`。
5. W3-T2 推进：Hub runtime 存储命名收口，sqlite schema 表/索引统一收敛为无版本后缀命名，重建路径同步删除 `_v1` 表清理逻辑。
6. W3-T4 推进：Week 3 审计快照更新，旧语义命中 `2403 -> 2370`（下降 `33`），版本词命中 `812 -> 770`（下降 `42`），兼容词命中保持 `370`。

### 6.6 本批次新增验收命令与结果摘要（2026-03-05）

1. `cd services/hub && go test ./internal/httpapi -count=1` ✅
2. `pnpm contracts:generate` ✅
3. `pnpm contracts:check` ✅
4. `cd services/hub && go test ./...` ✅
5. `cd services/hub && go vet ./...` ✅
6. `pnpm lint` ✅
7. `pnpm test` ✅
8. `scripts/refactor/gate-check.sh` ✅
9. `rg -n "\\b(conversation|execution|Conversation|Execution)\\b" ... | wc -l` -> `2370`
10. `rg -n "\\b(v1|v2|v3|v4|V1|V2|V3|V4)\\b" ... | wc -l` -> `770`
11. `rg -n "\\blegacy\\w*|\\bcompat\\w*|fallback|alias" ... | wc -l` -> `370`

### 6.7 Week 3 收口补充批次（2026-03-05 夜间）

1. W3-C1：补齐 DB 双冷启动自动化证据，新增 `TestOpenAuthzStoreSupportsRuntimeSchemaAfterTwoColdStarts`，每轮均执行“删除 sqlite 文件 -> 重建 -> 校验 session/run/events/changeset/hooks runtime 仓储链路”。
2. W3-C1：执行重启集成用例，覆盖 `project-config` 与 `workspace-agent-config` 在 router 重启后的持久化可用性，作为“冷启动后核心链路可用”补充证据。
3. W3-C2：完成残余命名收口，runtime 相关测试文件级命名全部去版本后缀。
4. W3-C2：修正残余 runtime 文案并完成代码子域去版本语义清零（仅文档历史记录保留）。
5. W3-C4：Week4 preflight 边界冻结完成，更新 `week3-preflight-checklist` 新增“路由/store/views/tests/i18n”批次拆分与回滚粒度。
6. W3-T4 推进：审计快照刷新为 `conversation/execution=2370`、`v* token=769`、`legacy/compat/fallback/alias=370`。

### 6.8 本批次新增验收命令与结果摘要（2026-03-05 夜间）

1. `cd services/hub && go test ./internal/httpapi -run 'TestOpenAuthzStoreSupportsRuntimeSchemaAfterTwoColdStarts|TestProjectConfigPersistsAcrossRouterRestart|TestWorkspaceAgentConfigPersistsAndExecutionSnapshotIsFrozen|TestConversationChangeSetEndpointForNonGitProject|TestConversationInputSubmit_EmitsUserPromptSubmitHookRecord|TestHookExecutionsHandlerListsRecordsForRunConversation' -count=1` ✅
2. `cd services/hub && go test ./...` ✅
3. `cd services/hub && go vet ./...` ✅
4. `pnpm contracts:generate` ✅
5. `pnpm contracts:check` ✅
6. `pnpm lint` ✅
7. `pnpm test` ✅
8. `scripts/refactor/gate-check.sh` ✅
9. `rg -n "\\b(conversation|execution|Conversation|Execution)\\b" ... | wc -l` -> `2370`
10. `rg -n "\\b(v1|v2|v3|v4|V1|V2|V3|V4)\\b" ... | wc -l` -> `769`
11. `rg -n "\\blegacy\\w*|\\bcompat\\w*|fallback|alias" ... | wc -l` -> `370`

### 6.9 Week 4 Batch A/B/C 实施结果（2026-03-05）

1. W4-T1 达成：目录完成迁移 `apps/desktop/src/modules/conversation` -> `apps/desktop/src/modules/session`，并同步修复 router 与 vitest 覆盖入口。
2. W4-T2 推进：Desktop 会话主链路 import 全量切换到 `@/modules/session/*`，移除 `@/modules/conversation/*` 导入残留。
3. W4-T3 推进：i18n key 全量收敛为 `session.*`，`messages.en-US.ts` 与 `messages.zh-CN.ts` 及调用点完成联动。
4. Phase 1（Hub 小步收口）推进：`internal/httpapi` 去 `_v1` 文件级命名（`run_query_service`、`hook_run_query_service`、`run_task_query_service`、`repository/repository_sqlite`）并同步符号重命名。
5. 过程故障已闭环：Batch C 首轮发生变量误替换（`conversation.id` 被替换为 `session.id`）导致 `pnpm lint`/`pnpm test` 失败；已在同批修复并重新验证通过。

### 6.10 本批次验收命令与结果摘要（2026-03-05，Batch A/B/C/D）

1. `pnpm contracts:check` ✅
2. `cd services/hub && go test ./internal/httpapi/...` ✅
3. `pnpm lint` ❌（首轮：变量误替换导致 `session is not defined`）
4. `pnpm test` ❌（首轮：同上）
5. 变量引用修复后，`pnpm lint` ✅
6. 变量引用修复后，`pnpm test` ✅
7. `pnpm contracts:generate && pnpm contracts:check` ✅
8. `cd services/hub && go test ./... && go vet ./...` ✅
9. `pnpm test:strict` ✅
10. `pnpm e2e:smoke` ✅
11. `scripts/refactor/gate-check.sh` ✅
12. `rg -n "\\b(conversation|execution|Conversation|Execution)\\b" ... | wc -l` -> `1620`（较上一快照 `2370` 下降 `750`）
13. `rg -n "\\b(v1|v2|v3|v4|V1|V2|V3|V4)\\b" ... | wc -l` -> `769`（持平）
14. `rg -n "\\blegacy\\w*|\\bcompat\\w*|fallback|alias" ... | wc -l` -> `370`（持平）

### 6.11 Week 4 准入结论（2026-03-05）

1. Week 4 Batch A/B/C 已完成并通过高风险门禁（`test:strict` + `e2e:smoke`）。
2. 目录迁移导致的路径断裂风险（R-003）已从“潜在”转为“可控并已验证”。
3. 当前可准入 Week 5（兼容清零与门禁升级），前提是继续按“分批门禁 + 日终全量门禁”执行。

### 6.12 Week 5-1 兼容路径清零与门禁升级增量（2026-03-05）

1. Phase A 冻结基线：以提交 `bc6ea79` 固化 Week 4 目录迁移与命名收口结果，作为独立可回滚检查点。
2. Phase B（Hub+Desktop+Contracts）推进：移除 `execution_enqueued` 兼容响应分支，统一为 `run_enqueued`，并将 payload 字段从 `execution` 收敛为 `run`（contracts/shared-core/httpapi/desktop 同步）。
3. Phase B（SSE）推进：移除 `legacy_event_type` 写回逻辑，事件 payload 不再输出该兼容字段。
4. Phase B（Hub fallback 收口）推进：`ConversationsHandler`/`ExecutionsHandler` 在 repository 可用但查询失败时不再回落到内存 map，改为显式 `RUNTIME_QUERY_FAILED`。
5. Phase C 推进：`scripts/refactor/gate-check.sh` 新增增量阻断词 `execution_enqueued`、`legacy_event_type`、`fallback to in-memory map`（仅针对新增代码）。
6. 审计快照刷新：
   - `rg -n "\\b(conversation|execution|Conversation|Execution)\\b" ... | wc -l` -> `1592`（较 Week 4 快照 `1620` 下降 `28`）
   - `rg -n "\\b(v1|v2|v3|v4|V1|V2|V3|V4)\\b" ... | wc -l` -> `769`（持平）
   - `rg -n "\\blegacy\\w*|\\bcompat\\w*|fallback|alias" ... | wc -l` -> `358`（较 Week 4 快照 `370` 下降 `12`）

### 6.13 本批次验收命令与结果摘要（2026-03-05，Week 5-1）

1. `pnpm contracts:generate && pnpm contracts:check` ✅
2. `cd services/hub && go test ./internal/httpapi/...` ✅
3. `cd services/hub && go test ./... && go vet ./...` ✅
4. `pnpm lint && pnpm test` ✅
5. `pnpm test:strict && pnpm e2e:smoke` ✅
6. `scripts/refactor/gate-check.sh` ✅

### 6.14 Week 5-2A/B/C 实施结果（2026-03-05）

1. W5-2A（Hub fallback 清零）完成：`internal/httpapi` 中 8 处 `fallback to in-memory map` 运行时分支清理完成，统一为“repository 不可用才走内存；repository 可用但查询失败返回 `RUNTIME_QUERY_FAILED`”。
2. W5-2A 覆盖模块：`change_set_service`、`conversation_by_id`、`token_usage_aggregate`、`project_conversations_config`、`run_tasks`、`hooks`。
3. W5-2A 链路联动：`computeTokenUsageAggregate` 改为显式错误返回，`projects/workspace_project_configs/resource_configs` 调用侧统一透传 `RUNTIME_QUERY_FAILED`。
4. W5-2B（门禁升级）完成：`scripts/refactor/gate-check.sh` 新增“总量不回升”校验，并接入白名单与基线文件。
5. W5-2B 新增文件：`scripts/refactor/gate-whitelist.txt`、`scripts/refactor/gate-baseline.env`。
6. W5-2B 本地演示完成：临时注入 `execution_enqueued` 新增行后，`gate-check.sh` 按预期阻断；还原注入后门禁恢复通过。
7. W5-2C（文档收口）完成：本文件与 `risk-register` 写入 Week 5-2 delta 与失败样例证据。

### 6.15 Week 5-2 审计快照与门禁证据（2026-03-05）

1. 审计 before（Week 5-1 基线）：
   - `conversation/execution`：`1592`
   - `v1/v2/v3/v4`：`769`
   - `legacy/compat/fallback/alias`：`358`
   - `fallback to in-memory map`（httpapi）：`8`
2. 审计 after（Week 5-2）：
   - `rg -n "\\b(conversation|execution|Conversation|Execution)\\b" ... | wc -l` -> `1586`（下降 `6`）
   - `rg -n "\\b(v1|v2|v3|v4|V1|V2|V3|V4)\\b" ... | wc -l` -> `769`（持平）
   - `rg -n "\\blegacy\\w*|\\bcompat\\w*|fallback|alias" ... | wc -l` -> `350`（下降 `8`）
   - `rg -n "fallback to in-memory map" services/hub/internal/httpapi` -> `0`（下降 `8`）
3. 失败样例（本地注入验证）：
   - 门禁输出：`FAIL: detected forbidden addition pattern: execution_enqueued in services/hub/internal/httpapi/handlers_hooks.go`
   - 处置：移除临时注入后复跑 `scripts/refactor/gate-check.sh` 通过。
4. 本批次验收命令：
   - `pnpm contracts:generate && pnpm contracts:check` ✅
   - `cd services/hub && go test ./internal/httpapi/...` ✅
   - `cd services/hub && go test ./... && go vet ./...` ✅
   - `pnpm lint && pnpm test` ✅
   - `pnpm test:strict && pnpm e2e:smoke` ✅
   - `scripts/refactor/gate-check.sh` ✅
