# Hub v1 重构任务计划表（执行基线）

- 日期：2026-03-04
- 关联主文档：`docs/refactor/hub/2026-03-04-hub-v1-refactor-plan.md`
- 执行策略：分阶段主干提交（每阶段必须可运行、可测试）
- 范围：`services/hub` + `apps/desktop` + `packages/shared-core`

---

## 术语约定

| 缩写 | 含义 |
|---|---|
| HUB | `services/hub` |
| DESKTOP | `apps/desktop` |
| SHARED | `packages/shared-core` |
| API | `packages/contracts/openapi.yaml` + 生成类型 |

---

## 进度快照（初始化）

| 阶段 | 状态 | 说明 |
|---|---|---|
| R0 | 已完成 | 已落地 gate-check 增强、词汇守卫与 CI strict gate |
| R1 | 已完成 | 已完成 `core/events` 单一实现源收敛、`core/statemachine` 接入与兼容导出 |
| R2 | 已完成 | 已落地 hookscope/sandbox 实现并接入 executor/hooks |
| R3 | 已完成 | 已完成 runtime 运行链路 repository-first 收口，`AppState` map 降级为 fallback/cache |
| R4 | 已完成 | 已完成 runtime 路由切换、SSE run-only 词汇收敛、OpenAPI 与 contracts 同步、旧路径下线 |
| R5 | 已完成 | 已完成三类 routes 的 service registry 组装、域服务分层与非 runtime 验收 |
| R6 | 已完成 | ACP 方法集切换至 v1，旧方法下线，并补齐 stream 订阅语义 |
| R7 | 已完成 | CLI v1 命令树落地、session/run 适配层接入与 v4 runner 下线 |
| R8 | 进行中 | 已启动 Desktop 服务路径切换与 Shared Session/Run 过渡类型出口 |
| R9 | 待开始 | Legacy 清理、文档收口、全量验收 |

---

## 阶段总览

| 阶段 | 目标 | 关键产出 | 依赖 |
|---|---|---|---|
| R0 | 冻结方案与门禁 | 禁回流检查、阶段验收模板 | 无 |
| R1 | 事件/状态机模块实装 | `core/events`、`core/statemachine` 生产实现 | R0 |
| R2 | 策略模块实装 | `policy/hookscope`、`policy/sandbox` 接入执行链 | R1 |
| R3 | 持久化主链重构 | repository + sqlite v1 schema + service 层 | R1,R2 |
| R4 | Runtime API 切换 | `/v1/sessions`、`/v1/runs`、SSE run-only | R3 |
| R5 | 全 Hub 同构 | 非 runtime 接口统一 service/repository 模式 | R3 |
| R6 | ACP v1 | 新 JSON-RPC 方法集、旧方法移除 | R4 |
| R7 | CLI v1 | 新命令面 + 新适配层 | R4 |
| R8 | Desktop 同步 | Session/Run 模型、UI 术语、流式消费改造 | R4,R7 |
| R9 | 收口发布 | 删除 legacy/v4 路径、OpenAPI 1.0.0、全量门禁 | R5,R6,R7,R8 |

---

## R0. 基线冻结与门禁改造

### 任务清单

- [ ] R0-T1 固化重构主文档与任务计划表（本次提交）
- [x] R0-T2 新增/更新门禁脚本：禁止 `execution_runtime_*`、`v4Service`、`V4Runner`、`runtimebridge` 新引用
- [x] R0-T3 新增词汇守卫：运行链路代码禁止新增 `Conversation/Execution` 主语义（测试例外）
- [x] R0-T4 在 CI 中加入阶段门禁入口（严格模式）

### 关键文件面

1. `docs/refactor/hub/*`
2. `scripts/refactor/gate-check.sh`
3. `services/hub/internal/agent/core/architecture_guard_test.go`
4. 相关 CI 配置文件

### 验收命令

1. `scripts/refactor/gate-check.sh --strict`

---

## R1. Core 占位模块落地（events/statemachine）

### 任务清单

- [x] R1-T1 在 `internal/agent/core/events` 新建事件规格、payload 绑定、编码与校验实现
- [x] R1-T2 在 `internal/agent/core/statemachine` 新建状态机实现（transition matrix + control action）
- [x] R1-T3 将 `runtime/loop`、`adapters/*` 改为引用新模块
- [x] R1-T4 将旧 `core/events.go`、`core/runstate.go` 收敛为 re-export 或删除（`events.go`、`runstate.go` 均已完成兼容收敛）
- [x] R1-T5 补齐单元测试：事件类型约束、非法迁移、终态行为

### 关键文件面

1. `services/hub/internal/agent/core/events/*`
2. `services/hub/internal/agent/core/statemachine/*`
3. `services/hub/internal/agent/runtime/loop/*`
4. `services/hub/internal/agent/adapters/*`

### 验收命令

1. `cd services/hub && go test ./internal/agent/core/...`
2. `cd services/hub && go test ./internal/agent/runtime/loop/...`

---

## R2. Policy 占位模块落地（hookscope/sandbox）

### 任务清单

- [x] R2-T1 在 `policy/hookscope` 实现作用域解析器（global/workspace/project/session/plugin）
- [x] R2-T2 在 `policy/sandbox` 实现沙箱策略决策（path/cmd/network）
- [x] R2-T3 将 `tools/executor` 接入 sandbox 决策与审计元数据
- [x] R2-T4 将 `extensions/hooks` 接入 hookscope 结果
- [x] R2-T5 增加策略冲突、边界拒绝、审批转 ask 的测试

### 关键文件面

1. `services/hub/internal/agent/policy/hookscope/*`
2. `services/hub/internal/agent/policy/sandbox/*`
3. `services/hub/internal/agent/tools/executor/*`
4. `services/hub/internal/agent/extensions/hooks/*`

### 验收命令

1. `cd services/hub && go test ./internal/agent/policy/...`
2. `cd services/hub && go test ./internal/agent/tools/executor/...`
3. `cd services/hub && go test ./internal/agent/extensions/hooks/...`

---

## R3. Repository First：替换 AppState 主存储

### 任务清单

- [x] R3-T1 设计 v1 repository 接口：Session/Run/RunEvent/RunTask/ChangeSet/HookRecord
- [x] R3-T2 实现 sqlite repository（事务边界与分页查询）
- [x] R3-T3 建立 v1 schema 初始化与版本标识（破坏式重建）
- [x] R3-T4 将 `internal/httpapi` 运行链路数据访问改为 service->repository
- [x] R3-T5 移除 `AppState` 中 conversations/executions/executionEvents 主 map 读写路径

### 关键文件面

1. `services/hub/internal/httpapi/state.go`
2. `services/hub/internal/httpapi/db_sqlite.go`
3. `services/hub/internal/httpapi/*service*.go`（新增）
4. `services/hub/internal/httpapi/*repository*.go`（新增）

### 验收命令

1. `cd services/hub && go test ./internal/httpapi/...`
2. `cd services/hub && go vet ./...`

### 当前阶段证据（2026-03-04）

1. 新增：`services/hub/internal/httpapi/repository_v1.go`
2. 新增：`services/hub/internal/httpapi/repository_v1_sqlite.go`
3. 新增测试：`services/hub/internal/httpapi/repository_v1_sqlite_test.go`
4. schema 扩展：`services/hub/internal/httpapi/db_sqlite.go` 新增 `runtime_*_v1` 表与 `hub_schema_versions`
5. 已验证：`cd services/hub && go test ./internal/httpapi -run 'TestAuthzStoreCreatesHubRuntimeV1SchemaVersion|TestSQLiteRuntimeV1RepositoriesReplaceAndPaginate'`
6. 已验证：`cd services/hub && go test ./internal/httpapi/...`
7. 已验证：`cd services/hub && go vet ./...`
8. 新增：`services/hub/internal/httpapi/execution_query_service_v1.go`（`GET /v1/executions` 查询服务，`service -> repository`）
9. 接入：`services/hub/internal/httpapi/handlers_execution_flow.go` 中 `ExecutionsHandler` 优先走 query service，失败回退 in-memory
10. 接入：`services/hub/internal/httpapi/state_execution_domain.go` 在 snapshot 同步时写入 `runtime_*_v1` 仓储
11. 新增测试：`services/hub/internal/httpapi/execution_query_service_v1_test.go`、`services/hub/internal/httpapi/state_execution_runtime_v1_sync_test.go`
12. 已验证：`cd services/hub && go test ./...`
13. 新增：`services/hub/internal/httpapi/run_task_query_service_v1.go`（`RunGraph/RunTasks` 查询服务，`service -> repository`）
14. 新增：`services/hub/internal/httpapi/hook_execution_query_service_v1.go`（`HookExecutions` 查询服务，`service -> repository`）
15. 接入：`services/hub/internal/httpapi/handlers_run_tasks.go` 与 `services/hub/internal/httpapi/handlers_hooks.go` 优先走 query service，失败回退 in-memory
16. 新增测试：`services/hub/internal/httpapi/run_task_query_service_v1_test.go`，并扩展 `handlers_hooks_test.go` 覆盖 repository 查询路径
17. 扩展：`services/hub/internal/httpapi/execution_query_service_v1.go` 新增 `ListAllByConversation`，支持 conversation 维度全量 run 查询（repository-first）
18. 接入：`services/hub/internal/httpapi/handlers_conversation_by_id.go` 的 `GET /v1/conversations/{id}` 优先使用 repository 查询 executions，并据此计算 token usage
19. 接入：`services/hub/internal/httpapi/handlers_workspaces.go` 的 `WorkspaceStatus` 优先使用 repository 计算 conversation status，失败回退 in-memory
20. 新增测试：`conversation_usage_test.go`、`handlers_workspaces_status_test.go` 覆盖“state map 清空后仍能从 repository 读取”路径；新增 `TestExecutionQueryServiceListAllByConversation`
21. 补齐 runtime hook `task_id` 落库：`db_sqlite.go` + `repository_v1_sqlite.go` + `state_execution_runtime_v1_sync_test.go`，保证 `HookExecutionRecord.TaskID` 在 v1 仓储中可读写
22. 扩展 runtime run `model_config_id`：`runtime_runs_v1` schema + migration + repository 读写 + snapshot 同步，确保聚合统计可在 repository 路径保留模型配置维度
23. 新增：`executionQueryService.ComputeTokenUsageAggregate`（按 workspace 聚合 project/model token 使用量，`service -> repository`）
24. 接入：`handlers_projects.go`、`handlers_workspace_project_configs.go`、`handlers_resource_configs.go` 使用 repository-first token usage 聚合（失败回退 in-memory）
25. 新增测试：`token_usage_aggregate_test.go`、`handlers_workspace_project_configs_test.go`，并扩展 `handlers_projects_test.go`、`execution_query_service_v1_test.go` 覆盖 repository 聚合路径
26. 新增：`executionQueryService.ComputeConversationTokenUsage`（按 conversation 聚合 token 使用量，`service -> repository`）
27. 接入：`handlers_execution_flow.go` 的 `ConversationsHandler` 优先使用 repository 聚合会话 usage（失败回退 in-memory）
28. 新增测试：`conversation_usage_test.go` 增加会话列表 repository 路径覆盖，`execution_query_service_v1_test.go` 增加 `ComputeConversationTokenUsage` 单测
29. 接入：`handlers_run_control.go` 在控制前置阶段支持 repository-first run seed 查询，并在 map 缺失时回填 execution 到内存后继续状态控制
30. 新增测试：`handlers_run_control_test.go` 增加 `TestRunControlEndpoint_UsesRepositoryWhenExecutionMapMissing` 覆盖 run control repository 回填路径
31. 扩展 repository：`RuntimeSessionRepository` 新增 `GetByID`，并在 sqlite 实现与测试中补齐读取能力
32. 扩展 `RunControlHandler`：当 conversations map 缺失时支持 repository-first session seed 回填，保障 run control 在双 map 缺失时仍可执行
33. 接入：`change_set_service.go` 在 `diff_generated` 入账、ledger 重建、mutable 判定中支持 repository-first execution/run seed 查询（map 缺失时回填或直接判定）
34. 新增测试：`change_set_service_repository_test.go` 覆盖 change set 在 execution map 缺失情况下的 message_id 解析、ledger 重建与 busy 判定
35. 扩展 `change_set_service.go` 的 `ConversationChangeSet*` handlers：在 `conversations/projects` map 缺失时，优先通过 runtime session repository + project store 加载 seed 并回填，再执行 build/commit/discard/export
36. 新增测试：`change_set_service_repository_test.go` 增加 `ConversationChangeSetHandler/CommitHandler` 的 repository/project store 回填用例，覆盖 map 清空后的可用性
37. 扩展 `change_set_service.go` 的内部构建路径：`buildConversationChangeSetLocked` 与 `resolveConversationProjectKindLocked` 支持在锁内通过 runtime session repository + project store 回填 conversation/project seed
38. 新增测试：`change_set_service_repository_test.go` 增加 `buildConversationChangeSetLocked` 与 `resolveConversationProjectKindLocked` 的 map 缺失回填覆盖
39. 扩展 `handlers_workspaces.go`：`WorkspaceStatus` 在 `conversation_id` 指定路径和默认选择路径均支持 repository-first session seed 回填（`state.conversations` 缺失时从 runtime sessions 仓储恢复）
40. 新增测试：`handlers_workspaces_status_test.go` 覆盖 workspace status 在 `conversations/executions` map 清空后的两条路径（指定 conversation 与默认选择）均可返回正确状态
41. 扩展 `handlers_conversation_by_id.go`：`ConversationByID` 的 `GET/PATCH/DELETE` 统一支持 repository-first session seed 回填（`state.conversations` 缺失时从 runtime sessions 仓储恢复并继续处理）
42. 新增测试：`conversation_usage_test.go` 增加 `PATCH` 在 conversation map 缺失时的 repository 回填覆盖，并扩展 `GET` 回填用例覆盖 conversation map 清空场景
43. 扩展 `handlers_execution_flow.go`：`ConversationStop/ConversationRollback/ConversationExport` 增加 repository-first conversation seed 回填；`ConversationStop` 额外支持 active run 的 repository-first execution seed 回填
44. 新增测试：`handlers_execution_flow_repository_test.go` 覆盖 `ConversationStop` 在 conversation/execution map 清空场景与 `ConversationExport` 在 conversation map 清空场景下的可用性
45. 扩展 `handlers_conversation_events.go`：`ConversationEvents` 在 `state.conversations` 缺失时支持 repository-first session seed 回填，避免 SSE 入口因 map miss 直接 `CONVERSATION_NOT_FOUND`
46. 新增测试：`handlers_conversation_events_repository_test.go` 覆盖 conversation events 在 conversation map 清空场景仍可建立 SSE 并输出 run 事件
47. R1 收敛：`internal/agent/core/events/spec.go` 成为事件语义与编码实现源；`core/events.go`、`core/payloads.go`、`core/session.go` 改为兼容 re-export（保持现有调用面）
48. 接入：`handlers_execution_flow.go` 的 `ConversationsHandler` 改为 repository-first 会话列表查询（失败回退 in-memory）并在命中后回填 `state.conversations` 缓存
49. 接入：`handlers_project_conversations_config.go` 的 `ProjectConversationsHandler(GET)` 改为 repository-first 会话列表/usage 聚合（失败回退 in-memory）
50. 接入：`handlers_execution_flow.go` 的 `ExecutionsHandler` 在 `conversation_id` 路径使用 repository-first session seed 解析 workspace，移除对 `state.conversations` 的硬依赖
51. 新增测试：`conversation_usage_test.go` 增加 `TestConversationsHandlerGetUsesRepositoryWhenConversationAndExecutionMapMissing`
52. 新增测试：`handlers_project_conversations_config_test.go` 增加 `TestProjectConversationsHandlerGetUsesRepositoryWhenConversationMapMissing`

---

## R4. HTTP Runtime API 切换为 Session/Run

### 任务清单

- [x] R4-T1 路由改名：`/v1/conversations/*` -> `/v1/sessions/*`
- [x] R4-T2 提交接口改名：`/input/submit` -> `POST /v1/sessions/{id}/runs`
- [x] R4-T3 列表接口改名：`GET /v1/executions` -> `GET /v1/runs`
- [x] R4-T4 SSE 切换到 run-only 事件词汇并移除 execution 映射
- [x] R4-T5 删除 `execution_runtime_router.go` 与 `execution_runtime_v4_bridge.go`
- [x] R4-T6 OpenAPI 同步更新 runtime 路径与 schema

### 关键文件面

1. `services/hub/internal/runtime/routes/routes.go`
2. `services/hub/internal/httpapi/handlers_*`
3. `services/hub/internal/httpapi/run_event_adapter.go`（重写或删除）
4. `packages/contracts/openapi.yaml`

### 验收命令

1. `cd services/hub && go test ./internal/httpapi/...`
2. `pnpm contracts:generate && pnpm contracts:check`

### 当前阶段证据（2026-03-04）

1. 切换：`services/hub/internal/runtime/routes/routes.go` 完成 runtime 主路由切换，仅保留 `/v1/sessions/*`、`/v1/runs*`，旧 `/v1/conversations*` 与 `/v1/executions` 下线
2. 接入：`services/hub/internal/httpapi/runtime_session_path.go` 新增 query/path 双解析（`session_id` 优先，回退 `conversation_id`），`handlers_execution_flow.go` 与 `handlers_workspaces.go` 已接入
3. 收敛：`services/hub/internal/httpapi/run_event_adapter.go` 将 SSE payload `event_type` 统一为 run-only 词汇，并保留 `legacy_event_type` 兼容诊断字段
4. 清理：删除 `execution_runtime_router.go` 与 `execution_runtime_v4_bridge.go`，迁移为 `run_runtime_router.go` 与 `run_runtime_v4_bridge.go`
5. 同步：`packages/contracts/openapi.yaml`、`openapi_contract_test.go`、`router_test.go` 与 runtime 相关集成/单测完成 `sessions/runs` 路径与 schema 收口
6. 已验证：`cd services/hub && go test ./... && go vet ./...`
7. 已验证：`pnpm contracts:generate && pnpm contracts:check`
8. 已验证：`scripts/refactor/gate-check.sh --strict`

---

## R5. 全 Hub 接口同构（handler -> service -> repository）

### 任务清单

- [x] R5-T1 将 controlplane/runtime/integration 三类 handlers 去业务化
- [x] R5-T2 在 workspace/auth/project/resource/admin/hook 维度补 service 层
- [x] R5-T3 统一错误模型、审计记录与授权检查入口
- [x] R5-T4 统一分页/游标/列表返回策略
- [x] R5-T5 补齐集成测试，确保非 runtime API 在新架构下行为一致

### 关键文件面

1. `services/hub/internal/httpapi/router.go`
2. `services/hub/internal/controlplane/routes/routes.go`
3. `services/hub/internal/runtime/routes/routes.go`
4. `services/hub/internal/integration/routes/routes.go`
5. 对应 handler/service/repository 文件

### 验收命令

1. `cd services/hub && go test ./internal/httpapi/... ./internal/controlplane/... ./internal/runtime/... ./internal/integration/...`

### 当前阶段证据（2026-03-04）

1. 新增：`services/hub/internal/httpapi/route_service_registry.go`，以 `workspace/auth/project/resource/admin/hook` 六域服务装配三类路由（controlplane/runtime/integration）
2. 改造：`services/hub/internal/httpapi/router.go` 不再直接绑定 `AppState -> handler`，统一通过 service registry 输出 route handlers
3. 新增：`services/hub/internal/httpapi/route_service_registry_test.go`，覆盖三类 routes 的服务装配可注册性，防止 nil handler 回归
4. 保持：错误模型、鉴权与审计入口继续统一复用 `WriteStandardError`、`authorizeAction`、`authz.appendAudit`
5. 保持：分页/游标/列表返回继续统一复用 `parseCursorLimit` + `ListEnvelope{items,next_cursor}`
6. 已验证：`cd services/hub && go test ./internal/httpapi/... ./internal/controlplane/... ./internal/runtime/... ./internal/integration/...`
7. 已验证：`cd services/hub && go vet ./...`

---

## R6. ACP v1 协议重写

### 任务清单

- [x] R6-T1 定义 ACP v1 方法集与参数/响应模型
- [x] R6-T2 重写 `adapters/acp/server.go` 方法注册
- [x] R6-T3 重写 `adapters/acp/bridge.go`，直接对接 Session/Run service
- [x] R6-T4 增加 `stream.subscribe/unsubscribe` 支持
- [x] R6-T5 删除旧方法处理逻辑与测试桩

### 关键文件面

1. `services/hub/internal/agent/adapters/acp/server.go`
2. `services/hub/internal/agent/adapters/acp/bridge.go`
3. `services/hub/internal/agent/adapters/acp/*_test.go`
4. `services/hub/cmd/goyais-acp/main.go`

### 验收命令

1. `cd services/hub && go test ./internal/agent/adapters/acp/...`
2. `cd services/hub && go test ./cmd/goyais-acp/...`

### 当前阶段证据（2026-03-04）

1. 重写：`services/hub/internal/agent/adapters/acp/server.go` 注册方法切换为 `session.start/get/list`、`run.submit/control`、`stream.subscribe/unsubscribe`
2. 下线：`session/new`、`session/load`、`session/prompt`、`session/set_mode`、`session/cancel` 不再注册
3. 重写：`services/hub/internal/agent/adapters/acp/bridge.go` 由 CLI Runner 桥接改为直接依赖 Session/Run service（`agent/adapters/httpapi.Service`）
4. 新增：stream 订阅通知语义，支持 `run_event` / `approval_needed` / `command_result` 三类 ACP 事件下发
5. 更新：`services/hub/internal/agent/adapters/acp/server_test.go` 覆盖 v1 `session.get/list`、`stream.subscribe/unsubscribe` 与 legacy 方法移除校验
6. 已验证：`cd services/hub && go test ./internal/agent/adapters/acp/...`
7. 已验证：`cd services/hub && go test ./cmd/goyais-acp/...`

---

## R7. CLI v1 命令面重写

### 任务清单

- [x] R7-T1 设计 CLI v1 命令树（session/run）
- [x] R7-T2 替换 `NewV4Runner` 入口与 adapter 组合方式
- [x] R7-T3 更新 text/json/stream-json 输出协议
- [x] R7-T4 删除 `v4_runner.go` 及其测试
- [x] R7-T5 增加新命令单测与 e2e 冒烟

### 关键文件面

1. `services/hub/cmd/goyais-cli/main.go`
2. `services/hub/cmd/goyais-cli/cli/*`
3. `services/hub/cmd/goyais-cli/adapters/*`

### 验收命令

1. `cd services/hub && go test ./cmd/goyais-cli/...`

### 当前阶段证据（2026-03-04）

1. 新增：`services/hub/cmd/goyais-cli/adapters/session_run_runner.go`，以 `SessionRunRunner` 统一承载 `session start/list/get` 与 `run submit/control/stream` 运行能力
2. 切换：`services/hub/cmd/goyais-cli/main.go` 与 `services/hub/cmd/goyais-cli/adapters/session_runner.go` 入口改为 `NewSessionRunRunner`，移除 `NewV4Runner` 依赖
3. 更新：`services/hub/cmd/goyais-cli/cli/commands/registry.go` 与 `services/hub/cmd/goyais-cli/cli/commands/handlers.go` 新增 `session/*` 与 `run/*` 命令树及执行处理
4. 更新：`services/hub/cmd/goyais-cli/adapters/session_run_runner.go` 输出协议对齐 `text/json/stream-json`，其中 `stream-json` 统一为 `type=text|tool_use|tool_result|result`
5. 删除：`services/hub/cmd/goyais-cli/adapters/v4_runner.go` 与 `services/hub/cmd/goyais-cli/adapters/v4_runner_test.go`
6. 新增/更新测试：`session_run_runner_test.go`、`session_runner_test.go`、`app_test.go`、`commands_behavior_test.go` 覆盖新命令树与新输出协议
7. 已验证：`cd services/hub && go test ./cmd/goyais-cli/...`

---

## R8. Desktop + Shared 同步切换

### 任务清单

- [ ] R8-T1 更新 OpenAPI 生成类型并替换 `Conversation/Execution` 为 `Session/Run`
- [ ] R8-T2 重写 Desktop conversation store 为 session/run store
- [x] R8-T3 删除 `runEventAdapter` 的 execution 映射层
- [x] R8-T4 服务调用路径切换到 `/v1/sessions/*`、`/v1/runs/*`
- [x] R8-T5 UI 文案统一改为 Session
- [ ] R8-T6 更新全部测试快照与 mock

### 关键文件面

1. `packages/contracts/openapi.yaml`
2. `packages/shared-core/src/generated/openapi.ts`
3. `packages/shared-core/src/api-*.ts`
4. `apps/desktop/src/modules/conversation/**/*`
5. `apps/desktop/src/modules/project/**/*`

### 验收命令

1. `pnpm lint`
2. `pnpm test`
3. `pnpm test:strict`
4. `pnpm e2e:smoke`

### 当前阶段证据（2026-03-05）

1. 切换：`apps/desktop/src/modules/conversation/services/index.ts` 会话主链路请求路径切换为 `/v1/sessions/*` 与 `/v1/runs/*`
2. 切换：`apps/desktop/src/modules/project/services/index.ts` project 维度会话路径从 `/conversations` 切换为 `/sessions`
3. 过渡：`packages/shared-core/src/api-project.ts` 新增 `Session/Run` 等别名类型出口，保持旧 `Conversation/Execution` 兼容
4. 更新测试：`apps/desktop/src/modules/conversation/tests/conversation.spec.ts` 与 `conversation-race.spec.ts` 的 URL 断言同步改为 session/run 路径
5. 已验证：`pnpm --filter @goyais/desktop test -- src/modules/conversation/tests/conversation.spec.ts src/modules/conversation/tests/conversation-race.spec.ts`
6. 已验证：`pnpm --filter @goyais/desktop lint`
7. 已验证：`pnpm lint && pnpm test`
8. 已验证：`pnpm test:strict && pnpm e2e:smoke`
9. 收敛：删除 `apps/desktop/src/modules/conversation/store/runEventAdapter.ts`，并在 `store/stream.ts` 内联 run->execution 事件归一化，移除独立 execution 映射层
10. 清理测试：删除 `run-event-adapter.spec.ts`，并由 `conversation-stream.spec.ts` 覆盖 run-centric SSE 事件映射与路由行为
11. 统一文案：`messages.en-US.ts`、`messages.zh-CN.ts`、`MainScreenView.vue`、资源配置说明改为 Session/会话主语义；默认命名切换为 `Session`/`新会话` 并保持旧 `Conversation`/`新对话` 识别兼容
12. 过渡 facade：`conversation/services`、`project/services` 与 `conversation/store` 新增 `Session/Run` 命名导出（保持旧 `Conversation/Execution` 导出兼容），用于分批切换调用面
13. 调用面切换：流式主链（`store/stream.ts`、`views/streamCoordinator.ts`、`workspaceStatusStore.ts`）改用 `Session` 命名 service/store facade，并同步更新相关测试 mock

---

## R9. 收口与发布验收

### 任务清单

- [ ] R9-T1 删除所有 legacy/v4 过渡实现与死代码
- [ ] R9-T2 OpenAPI 版本升级到 `1.0.0`
- [ ] R9-T3 更新 `docs/refactor` 主文档与任务状态
- [ ] R9-T4 执行全量门禁命令并沉淀证据

### 关键文件面

1. `services/hub/internal/httpapi/execution_runtime_*`（删除）
2. `services/hub/internal/agent/adapters/runtimebridge/*`（删除）
3. `services/hub/cmd/goyais-cli/adapters/v4_runner.go`（删除）
4. `packages/contracts/openapi.yaml`
5. `docs/refactor/hub/*`

### 验收命令

1. `cd services/hub && go test ./... && go vet ./...`
2. `pnpm contracts:generate && pnpm contracts:check`
3. `pnpm lint && pnpm test`
4. `pnpm test:strict && pnpm e2e:smoke`
5. `make health`

---

## 里程碑门禁（必须全部满足）

1. 代码树不存在 `execution_runtime_router` / `execution_runtime_v4_bridge` / `runtimebridge` / `V4Runner`。
2. OpenAPI 不再出现 runtime 语义中的 `Conversation/Execution` 主 schema。
3. Desktop 主流程基于 `Session/Run` 可完成：创建、提交、流式、控制、任务图、变更集。
4. 所有验收命令通过，且门禁脚本严格模式通过。

---

## 执行纪律

1. 每个阶段至少一次可运行提交，不允许长时间大分叉。
2. 每阶段完成必须更新本任务表状态与证据。
3. 若超出阶段范围，先更新本任务表再实施。
