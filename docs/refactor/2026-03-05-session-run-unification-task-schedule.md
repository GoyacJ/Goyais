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
| W1-T1 | In Progress | OpenAPI 已将主 schema 反转为 `Session/Run` 主定义，`Conversation/Execution` 退为 alias；字段级旧语义仍有存量 | `Session/Run/SessionDetailResponse/SessionChangeSet/RunFilesExportResponse` 已为主定义 |
| W1-T2 | In Progress | 已完成 `hooks` 与 `workspace-status` 的 `session_id/run_id` 及 `session_status` 子域切换；并完成 Hub 对应结构体字段名收敛（`SessionID/SessionStatus`）；核心会话/执行链路尚未全量切换 | hooks payload/模型字段、workspace status payload/模型字段均已切到 session 语义 |
| W1-T3 | In Progress | 已启动 shared-core/desktop 类型去别名收口：`SessionDetailResponse` 改为主结构 + 旧字段可选兼容，Desktop conversation 核心服务/状态层改用 `Session/Run` 主类型并保留入参兼容归一化 | `api-project.ts`、`modules/conversation/services/index.ts`、`store/{state,executionMerge,executionRuntime}.ts` 已落地主类型改造 |
| W1-T4 | Not Started | 契约未冻结，仍处于持续切换阶段 | 本文件状态为执行中 |
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
6. OpenAPI 主 schema 收敛：`Session/Run` 成为主定义，`Conversation/Execution` 改为 alias（并联动 contract tests 与 generated types）。
7. Hub 内部命名收敛：hooks/workspace-status 结构体字段由旧语义映射名进一步收敛为 `SessionID/SessionStatus`，并联动快照持久化与测试夹具同步。
8. W1-T3 启动：shared-core `api-project.ts` 将 `SessionDetailResponse` 设为主结构并将旧字段降级为可选兼容，`ConversationDetailResponse` 收敛为别名；Desktop conversation 核心类型注解迁移到 `Session/Run` 主类型并保留旧 payload 兼容归一化。
9. W1-T3 持续推进：Desktop `modules/project` 子域服务层与 store 层类型注解收敛到 `Session` 主类型，减少 `Conversation` 类型别名扩散。
10. W1-T3 持续推进：Desktop `modules/project` 子域内部调用链切换到 `listSessions/createSession/patchSession/removeSession/exportSessionMarkdown`，并保留 conversation 命名服务函数作为兼容壳。
11. W1-T3 持续推进：Desktop `modules/conversation` 视图与事件去重子域完成类型注解收敛（`Conversation/Execution/ExecutionEvent` -> `Session/Run/RunLifecycleEvent`），保持运行时字段与行为不变。
12. W1-T3 持续推进：Desktop `modules/conversation/store` 核心执行编排子域完成类型注解收敛（`executionActions/executionEventHandlers/events` 切换到 `Session/SessionMessage/RunLifecycleEvent`），保留函数命名与字段语义兼容。
13. W1-T3 持续推进：Desktop `modules/conversation/trace + store/stream` 子域完成类型注解收敛（`Execution/ExecutionEvent` -> `Run/RunLifecycleEvent`），保留 normalize/build 系列函数命名以避免调用面震荡。
14. W1-T3 持续推进：Desktop `modules/conversation/store/state` 与 `views/useMainScreenController` 补齐类型收敛（`Conversation` 参数类型改 `Session`，`ExecutionEvent` 注解改 `RunLifecycleEvent`），保持函数命名与数据字段兼容。

### 6.3 最新审计快照（2026-03-05）

1. `rg -n "\\b(conversation|execution|Conversation|Execution)\\b" services/hub apps/desktop/src packages/shared-core/src packages/contracts` -> `2498`（较上一轮 `2517` 下降 `19`）。
2. `rg -n "\\b(v1|v2|v3|v4|V1|V2|V3|V4)\\b" services/hub apps/desktop/src packages/shared-core/src packages/contracts` -> `821`（含 `/v1` 路径与允许白名单）。
3. `rg -n "\\blegacy\\w*|\\bcompat\\w*|fallback|alias" services/hub apps/desktop/src packages/shared-core/src packages/contracts` -> `371`。
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
12. `pnpm --filter @goyais/desktop exec vitest run src/modules/conversation/tests/execution-trace-state.spec.ts src/modules/conversation/tests/conversation.spec.ts` ✅
13. `pnpm --filter @goyais/desktop exec vitest run src/modules/conversation/tests/running-actions.spec.ts src/modules/conversation/tests/process-trace.spec.ts src/modules/conversation/tests/conversation-token-usage.spec.ts src/modules/conversation/tests/use-queue-messages-view.spec.ts src/modules/conversation/tests/main-screen-actions.spec.ts` ✅
14. `pnpm --filter @goyais/desktop exec vitest run src/modules/conversation/tests/conversation.spec.ts src/modules/conversation/tests/conversation-stream.spec.ts src/modules/conversation/tests/conversation-race.spec.ts src/modules/conversation/tests/main-screen-actions.spec.ts src/modules/conversation/tests/conversation-run-tasks-actions.spec.ts` ✅
15. `pnpm --filter @goyais/desktop exec vitest run src/modules/conversation/tests/trace/normalize.spec.ts src/modules/conversation/tests/trace/present.spec.ts src/modules/conversation/tests/process-trace.spec.ts src/modules/conversation/tests/running-actions.spec.ts src/modules/conversation/tests/conversation-token-usage.spec.ts src/modules/conversation/tests/conversation-stream.spec.ts` ✅
16. `pnpm --filter @goyais/desktop exec vitest run src/modules/conversation/tests/main-screen-controller.spec.ts src/modules/conversation/tests/conversation.spec.ts` ✅
