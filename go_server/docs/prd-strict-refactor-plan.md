# Goyais PRD Strict Refactor Plan（Truth-Based, 2026-02-11）

## 1. 基线与原则（Baseline）

- 唯一需求基线（single source of truth）：`docs/prd.md`
- 严格口径契约（strict contracts）：
  - `go_server/docs/api/openapi.yaml`
  - `go_server/docs/arch/overview.md`
  - `go_server/docs/arch/data-model.md`
  - `go_server/docs/arch/state-machines.md`
  - `go_server/docs/acceptance.md`
- 统一回归命令（fixed gate）：
  - `bash go_server/scripts/ci/contract_regression.sh`

本文件只记录“代码真值（code truth）+ 下一步缺口（next gaps）”，不复述历史结论。

## 2. 当前实现真值矩阵（As-Is Truth）

### 2.1 已完成（Completed）

| 领域 | 结论 | 证据 |
|---|---|---|
| 页面路由覆盖（Run Center / Algorithm Library / Permission Management / ContextBundle） | 已存在独立路由 | `vue_web/src/router/index.ts:48`, `vue_web/src/router/index.ts:72`, `vue_web/src/router/index.ts:84`, `vue_web/src/router/index.ts:90` |
| Stream 前端控制面对齐（update-auth / delete） | 已有 API + UI 入口 | `vue_web/src/api/streams.ts:84`, `vue_web/src/api/streams.ts:96`, `vue_web/src/views/StreamsView.vue:266`, `vue_web/src/views/StreamsView.vue:300` |
| AI 计划预览链路（preview-only） | 已新增后端路由、前端 API、工作台接入 | `go_server/internal/access/http/router.go:66`, `go_server/internal/access/http/ai_context.go:68`, `vue_web/src/api/ai.ts:87`, `vue_web/src/views/AIWorkbenchView.vue:395` |
| Canvas AI patch 闭环（server-validated） | 已支持“preview -> workflow.patch(operations) -> 服务端校验应用 -> 前端差异/失败反馈” | `go_server/internal/workflow/service.go:81`, `vue_web/src/views/CanvasView.vue:726`, `go_server/internal/access/http/router_integration_test.go:898` |
| 算法库页面运行闭环 | 已支持输入 JSON、触发 `algorithm.run`、展示 run 结果与 commandId | `vue_web/src/views/AlgorithmLibraryView.vue:73`, `vue_web/src/api/algorithms.ts:13`, `vue_web/src/views/AlgorithmLibraryView.spec.ts:122` |
| 统一回归健康 | 本轮复核通过 | `go_server/scripts/ci/contract_regression.sh` |

### 2.2 部分完成（Partially Completed）

| 领域 | 现状 | 证据 | 严格缺口 |
|---|---|---|---|
| AI planner | 已从单点解析抽离为可扩展 parser chain，并返回 explainability | `go_server/internal/ai/planner/planner.go:43`, `go_server/internal/ai/planner/planner.go:93` | 仍属 deterministic rule-based，缺少更高层 intent strategy（reject/alternative 语义仍偏模板化） |
| Workflow 执行语义 | DAG/调度/重试骨架在位 | `go_server/internal/workflow/engine.go:291` | step 输出仍以 `handled/mode/stepKey` 规则化结果为主，真实 capability 语义不足 |
| ContextBundle rebuild | 已有 run/session/workspace 聚合 | `go_server/internal/contextbundle/service.go:376`, `go_server/internal/contextbundle/service.go:497` | workspace 大规模场景下摘要质量仍偏浅层统计 |

### 2.3 未闭环（Open Gaps）

| 领域 | 现状 | 证据 | 目标 |
|---|---|---|---|
| 权限管理页面语义 | 当前以 share command 审计视图为主 | `vue_web/src/views/PermissionManagementView.vue:146` | 补齐用户/角色/策略最小管理闭环 |
| Run Center 操作深度 | 当前以 events/steps 基础浏览为主 | `vue_web/src/views/RunCenterView.vue:129` | 补日志与产物可操作视图（引用/跳转/下载） |

## 3. 下一步未完成项（Next Steps）

### 3.1 P0（必须优先）

#### P0-1 文档真值持续同步（Doc Truth Sync）
- DoD：
  - 每个严格缺口必须给出代码证据、验收命令、风险与回滚开关。
  - 本文、`acceptance.md`、`openapi.yaml` 保持同一口径。
- 验收命令：
  - `bash go_server/scripts/ci/contract_regression.sh`

#### P0-2 AI 规划能力深化（Command-first 保持不变）
- 范围：
  - 在现有 planner chain 上扩展更强 intent strategy。
  - 强化 reject reason 与 alternatives 的可解释性（explainability）。
  - 执行仍必须经 `ai.command.execute -> command gate -> tool gate`。
- DoD：
  - 同输入稳定输出可解释 plan。
  - 不支持输入返回明确拒绝原因与替代建议。
- 风险与回滚：
  - 风险：planner 误判导致 payload 过宽。
  - 回滚：`GOYAIS_FEATURE_AI_WORKBENCH=false`。

#### P0-4 Workflow 语义深化（Execution Semantics）
- 范围：
  - step 输出从占位结构升级为真实执行上下文（input/output/artifacts/error metadata）。
- DoD：
  - step 详情可用于 Run Center 直接消费，不再仅靠 `handled/mode/stepKey`。

### 3.2 P1（在 P0 后）

#### P1-1 ContextBundle 质量增强
- 目标：提升 facts/summaries/refs/timeline 的跨 run/session/workspace 可读性与可检索性。

#### P1-2 页面能力补齐
- 权限管理：用户/角色/策略最小闭环。
- Run Center：日志与产物操作入口。

## 4. 接口与兼容性（Interfaces）

- 保持不变：
  - `/api/v1` 前缀不变。
  - Command-first 语义不变。
  - 既有 stream 控制面 API 不变。
- 本轮新增：
  - `POST /api/v1/ai/plans:preview`（preview-only，无副作用）。
- 兼容策略：
  - 未来 AI planner 字段扩展采用向后兼容（optional fields），不破坏既有调用方。

## 5. 固定验收命令（Quality Gates）

- `go test ./...`（在 `go_server/`）
- `pnpm -C vue_web typecheck`
- `pnpm -C vue_web test:run`
- `bash go_server/scripts/ci/contract_regression.sh`

## 6. 风险控制（Risk Control）

- 风险：`acceptance.md` 与严格口径再次漂移。
  处理：每个切片 PR 同步更新契约文档矩阵。
- 风险：AI 规划能力增强引入越权路径。
  处理：AI 仅产生命令草案，执行仍经 command/tool gate 与审计链路。
- 风险：前端页面能力补齐导致交互复杂度上升。
  处理：按最小闭环分阶段发布，并保持 feature-gate 回滚点。
