# Goyais PRD 严格口径重构计划（文档化 + 开发启动版）

## 1. 背景与结论

### 1.1 背景
- 本计划基于 `docs/prd.md` 的严格口径审计结论制定。
- 当前实现在 v0.1 冻结范围内已具备基础闭环能力，但与 PRD 的“完整目标”仍存在结构性缺口。

### 1.2 审计结论引用
- 审计结论：当前实现**不满足** PRD 严格口径。
- 核心阻断：
  1. Workflow Engine 仍以最小模式（`step-1/noop + applyRunMode`）为主，未达到 PRD 要求的真实 DAG 执行/调度/重试/回放能力。
  2. 复杂可视化编排画布未满足 PRD 8.9 必须项（typed ports、minimap、undo/redo、run from here、test node）。
  3. AI 工作台（会话/计划/解释/执行反馈）能力缺失。
  4. MediaMTX 控制面、插件生命周期、ContextBundle/ACL 覆盖面与 PRD 要求存在缺口。

## 2. 目标与完成标准（PRD 严格口径）

### 2.1 总体目标
- 先文档化重构方案，再按固定切片顺序实施，优先清零 P0 阻断项。

### 2.2 完成标准
- 满足判定以 `docs/prd.md` 为唯一需求基线。
- 每个切片必须同时满足：
  - DoD
  - 测试证据
  - 回滚策略（feature flag）
  - 契约文档同步（openapi/data-model/state-machines/overview/acceptance）
- 所有副作用写路径保持 Command-first，不允许 AI 绕过 command gate。

## 3. 范围与非范围

### 3.1 In Scope
- S0~S6 全切片（含契约与实现）。
- 新增 API、commandType、实体与迁移定义。
- 前后端、构建、single binary 全量回归。

### 3.2 Out of Scope
- PRD 中明确延后项（多人实时协作、计费结算、深度供应链安全、跨地域容灾）。
- 未在本计划列出的大范围架构重写。

## 4. 公共接口/类型变更总览

### 4.1 新增 API
- AI 工作台
  - `POST /api/v1/ai/sessions`
  - `GET /api/v1/ai/sessions`
  - `GET /api/v1/ai/sessions/{sessionId}`
  - `POST /api/v1/ai/sessions/{sessionId}:archive`
  - `POST /api/v1/ai/sessions/{sessionId}/turns`
  - `GET /api/v1/ai/sessions/{sessionId}/events`
- Workflow 事件流
  - `GET /api/v1/workflow-runs/{runId}/events`
- 插件市场补齐
  - `GET /api/v1/plugin-market/packages/{packageId}:download`
  - `POST /api/v1/plugin-market/installs/{installId}:upgrade`
- Stream 控制面补齐
  - `POST /api/v1/streams/{streamId}:update-auth`
  - `DELETE /api/v1/streams/{streamId}`
- ContextBundle 读接口
  - `GET /api/v1/context-bundles`
  - `GET /api/v1/context-bundles/{bundleId}`

### 4.2 新增 commandType
- `ai.session.create`
- `ai.session.archive`
- `ai.intent.plan`
- `ai.command.execute`
- `plugin.upgrade`
- `stream.updateAuth`
- `stream.delete`
- `context.bundle.rebuild`

### 4.3 新增实体/迁移目标
- `ai_sessions`
- `ai_session_turns`
- `workflow_run_events`
- `context_bundles`
- `context_bundle_items`
- `plugin_install_history`
- `stream_auth_rules`
- `acl_entries.subject_type` 扩展为 `user|role`

### 4.4 兼容性约束
- 保持 `/api/v1` 不变。
- 保持 domain sugar 响应结构 `resource + commandRef`。
- 新能力默认允许 `501 NOT_IMPLEMENTED` 作为渐进落地状态，但路径/契约必须先到位。

## 5. 重构切片总览（S0-S6）

| 切片 | 优先级 | 目标 | 状态门槛 |
|---|---|---|---|
| S0 | P0 | 契约同步与门禁 | 完成后方可进入 S1 |
| S1 | P0 | Workflow Engine | DAG 执行链路转正 |
| S2 | P0 | Canvas 重构 | 满足 PRD 8.9 |
| S3 | P0 | AI 工作台 | 文本 AI 闭环 |
| S4 | P0 | MediaMTX 控制面 | 事件驱动 + 录制资产化 |
| S5 | P1 | 插件市场生命周期 | download/upgrade/状态机完善 |
| S6 | P1 | ContextBundle + ACL 扩面 | run/session/workspace 上下文闭环 |

## 6. 每切片详细实施（目标/范围/受影响路径/DoD/测试/回滚）

### 6.1 S0（P0）契约同步与门禁
- 目标：先完成契约，确保新增 API/状态机/数据模型可审计。
- 范围：文档与路由可达性，不引入破坏性运行时变更。
- 受影响绝对路径：
  - `/Users/goya/Repo/Git/Goyais/docs/api/openapi.yaml`
  - `/Users/goya/Repo/Git/Goyais/docs/arch/data-model.md`
  - `/Users/goya/Repo/Git/Goyais/docs/arch/state-machines.md`
  - `/Users/goya/Repo/Git/Goyais/docs/arch/overview.md`
  - `/Users/goya/Repo/Git/Goyais/docs/acceptance.md`
  - `/Users/goya/Repo/Git/Goyais/internal/access/http/openapi_reachability_test.go`
- DoD：
  - 新增 API 全部进入 OpenAPI。
  - 路由可达性测试覆盖新增 path params。
  - 变更说明写入本计划文档与 acceptance。
- 测试命令：见第 7 章。

### 6.2 S1（P0）Workflow Engine
- 目标：替换最小 run mode，转为真实 DAG 执行。
- 范围：拓扑校验、并发调度、重试退避、Tool Gate、run/step 事件输出。
- 受影响绝对路径：
  - `/Users/goya/Repo/Git/Goyais/internal/workflow`
  - `/Users/goya/Repo/Git/Goyais/internal/command`
  - `/Users/goya/Repo/Git/Goyais/internal/app`
  - `/Users/goya/Repo/Git/Goyais/migrations/sqlite`
  - `/Users/goya/Repo/Git/Goyais/migrations/postgres`
- DoD：DAG 并发与重试行为可验证，events 可回放。
- 测试命令：见第 7 章 + workflow 专项回归。

### 6.3 S2（P0）Canvas 重构
- 目标：落地复杂可视化编排器。
- 范围：typed ports、minimap、undo/redo、run from here、test node。
- 受影响绝对路径：
  - `/Users/goya/Repo/Git/Goyais/web/src/views/CanvasView.vue`
  - `/Users/goya/Repo/Git/Goyais/web/src/components/runtime`
  - `/Users/goya/Repo/Git/Goyais/web/src/api/workflow.ts`
- DoD：满足 PRD 8.9 五条验收。

### 6.4 S3（P0）AI 工作台
- 目标：文本 AI 会话 + Intent 计划 + Command 解释与执行反馈。
- 范围：会话 API、turn 执行、SSE 事件、前端路由与页面。
- 受影响绝对路径：
  - `/Users/goya/Repo/Git/Goyais/internal/access/http`
  - `/Users/goya/Repo/Git/Goyais/internal/command`
  - `/Users/goya/Repo/Git/Goyais/internal/app`
  - `/Users/goya/Repo/Git/Goyais/web/src/router/index.ts`
  - `/Users/goya/Repo/Git/Goyais/web/src/views`
  - `/Users/goya/Repo/Git/Goyais/migrations/sqlite`
  - `/Users/goya/Repo/Git/Goyais/migrations/postgres`
- DoD：AI/UI 同动作 command 同形，权限拒绝原因可解释。

### 6.5 S4（P0）MediaMTX 控制面
- 目标：真实控制面调用与录制资产化。
- 范围：控制面 API、事件消费、workflow 触发、录制回填。
- 受影响绝对路径：
  - `/Users/goya/Repo/Git/Goyais/internal/stream`
  - `/Users/goya/Repo/Git/Goyais/internal/app/eventbus_stream_consumer.go`
  - `/Users/goya/Repo/Git/Goyais/internal/access/http/streams.go`
- DoD：onPublish/onRead/onConnect/onRecordFinish 事件可驱动 workflow。

### 6.6 S5（P1）插件市场完整生命周期
- 目标：补齐 download/upgrade 与状态机中间态。
- 范围：validating/installing、依赖校验、ceiling 校验、回滚链。
- 受影响绝对路径：
  - `/Users/goya/Repo/Git/Goyais/internal/plugin`
  - `/Users/goya/Repo/Git/Goyais/internal/access/http/plugins.go`
  - `/Users/goya/Repo/Git/Goyais/web/src/api/plugins.ts`
  - `/Users/goya/Repo/Git/Goyais/web/src/views/PluginsView.vue`
- DoD：状态机与文档一致，不再直接 `install -> enabled`。

### 6.7 S6（P1）ContextBundle + ACL 扩面
- 目标：补齐 ContextBundle 与 ACL role 主体。
- 范围：读写接口、聚合策略、role 维度授权。
- 受影响绝对路径：
  - `/Users/goya/Repo/Git/Goyais/migrations/sqlite`
  - `/Users/goya/Repo/Git/Goyais/migrations/postgres`
  - `/Users/goya/Repo/Git/Goyais/internal/command`
  - `/Users/goya/Repo/Git/Goyais/internal/access/http`
- DoD：RUN/SESSION/WORKSPACE 三层上下文可写可查。

## 7. 验收与回归命令（固定清单）

### 7.1 场景矩阵
1. OpenAPI 路由可达与实现一致。
2. Command-first 与 domain sugar 一致。
3. Workflow DAG 校验/调度/重试/回放。
4. Canvas 类型连线与差异 patch。
5. AI 会话计划解释与授权拒绝原因。
6. MediaMTX 事件触发与录制资产化。
7. 插件安装/升级/回滚与依赖/ceiling。
8. ContextBundle 读写与 ACL role 覆盖。

### 7.2 固定命令
- `go test ./...`
- `pnpm -C /Users/goya/Repo/Git/Goyais/web typecheck`
- `pnpm -C /Users/goya/Repo/Git/Goyais/web test:run`
- `make -C /Users/goya/Repo/Git/Goyais build`
- `GOYAIS_VERIFY_BASE_URL=http://127.0.0.1:18080 GOYAIS_START_CMD='GOYAIS_SERVER_ADDR=:18080 ./build/goyais' bash /Users/goya/Repo/Git/Goyais/.agents/skills/goyais-single-binary-acceptance/scripts/verify_single_binary.sh`

## 8. 风险与缓解（P0/P1/P2）

### 8.1 P0 阻断
1. Workflow 引擎仍为最小模式。
2. Canvas 未满足 PRD 8.9。
3. AI 工作台主链路缺失。
4. MediaMTX 控制面仅 MVP。

缓解：按 S0->S4 顺序推进，任一切片未达 DoD 不进入下一切片。

### 8.2 P1 重大
1. 插件状态机与市场生命周期不完整。
2. ContextBundle/ACL role 维度不完整。

缓解：S5/S6 前置契约 + 迁移回滚脚本双活。

### 8.3 P2 一般
1. 文档与实现存在阶段性漂移风险。
2. feature flag 组合导致回归矩阵膨胀。

缓解：固定回归命令 + feature flag 显式覆盖 + 契约文档同变更更新。

## 9. 里程碑与交付节奏

- M0：S0（契约同步与门禁）
- M1：S1 + S2（Workflow + Canvas）
- M2：S3 + S4（AI + Stream 控制面）
- M3：S5 + S6（插件生命周期 + ContextBundle/ACL）

节奏规则：
- 每切片单独交付、单独回归、单独回滚开关。
- 每切片完成后执行第 7 章全量回归。

## 10. 执行规范（thread/worktree、契约同步）

### 10.1 Git 线程隔离
- 每个切片必须使用独立 worktree 与分支：`codex/<thread-id>-<topic>`。
- 主工作树仅用于 `master` 集成与回归。
- 合并后必须移除 thread worktree 并 `git worktree prune`。

### 10.2 契约同步规则
- 若改动 API/实体/状态机/ACL/静态路由，必须同变更更新：
  - `/Users/goya/Repo/Git/Goyais/docs/api/openapi.yaml`
  - `/Users/goya/Repo/Git/Goyais/docs/arch/data-model.md`
  - `/Users/goya/Repo/Git/Goyais/docs/arch/state-machines.md`
  - `/Users/goya/Repo/Git/Goyais/docs/arch/overview.md`
  - `/Users/goya/Repo/Git/Goyais/docs/acceptance.md`

### 10.3 执行顺序与默认值
1. 固定顺序：`S0 -> S1 -> S2 -> S3 -> S4 -> S5 -> S6`。
2. feature gate 不作为“满足 PRD”的豁免，仅作为回滚措施。
3. 需求判定仅以 `/Users/goya/Repo/Git/Goyais/docs/prd.md` 为准。
4. 本轮语音输入保持 optional，文本 AI 入口必须完成。
