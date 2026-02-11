# Goyais PRD Strict Refactor Plan（Truth-Based, 2026-02-11）

## 1. Baseline 与固定原则

- 唯一需求基线（single source of truth）：`docs/prd.md`
- 严格契约文档（strict contracts）：
  - `go_server/docs/api/openapi.yaml`
  - `go_server/docs/arch/overview.md`
  - `go_server/docs/arch/data-model.md`
  - `go_server/docs/arch/state-machines.md`
  - `go_server/docs/acceptance.md`
- 统一回归门禁（fixed quality gate）：
  - `bash go_server/scripts/ci/contract_regression.sh`

本文件只记录“当前代码真值（as-is truth）+ 下一步建议（next focus）”，不保留历史口径。

## 2. 当前实现真值矩阵（As-Is Truth）

### 2.1 已完成（Completed）

| 领域 | 结论 | 代码证据 |
|---|---|---|
| 路由覆盖（Run Center / Algorithm Library / Permission Management / ContextBundle） | 页面路由已独立存在 | `vue_web/src/router/index.ts:48`, `vue_web/src/router/index.ts:72`, `vue_web/src/router/index.ts:84`, `vue_web/src/router/index.ts:90` |
| Stream 前端控制面对齐 | `update-auth/delete` 前后端已打通 | `vue_web/src/api/streams.ts:84`, `vue_web/src/api/streams.ts:96`, `vue_web/src/views/StreamsView.vue:266`, `vue_web/src/views/StreamsView.vue:300` |
| AI planner 多步规划与策略打分 | 规划器已支持 composite decomposition + strategy scores | `go_server/internal/ai/planner/planner.go:154`, `go_server/internal/ai/planner/planner.go:224`, `go_server/internal/ai/planner/planner.go:459`, `go_server/internal/ai/planner/planner_test.go:164` |
| AI execute 多步命令执行闭环 | `ai.command.execute` 已按 steps 串行提交子命令，回写全量 commandIds 与执行摘要 | `go_server/internal/app/command_executors.go:537`, `go_server/internal/app/command_executors.go:710`, `go_server/internal/app/command_executors.go:731`, `go_server/internal/app/command_executors_ai_test.go:19`, `go_server/internal/access/http/router_integration_test.go:563` |
| AI 工作台预览可解释性 | 前端已展示 score/steps/strategyScores，并避免多步链路被 explicit intent 降级为单命令 | `vue_web/src/views/AIWorkbenchView.vue:81`, `vue_web/src/views/AIWorkbenchView.vue:467`, `vue_web/src/views/AIWorkbenchView.vue:535`, `vue_web/src/views/AIWorkbenchView.spec.ts:245` |
| Workflow 执行语义深化 | step/run 输出已切到 capability 语义（executor/contract/input/output/error/recovery/capabilitySummary） | `go_server/internal/workflow/execution_semantics.go:16`, `go_server/internal/workflow/execution_semantics.go:162`, `go_server/internal/workflow/execution_semantics.go:225`, `go_server/internal/workflow/engine_test.go:281` |
| ContextBundle 质量增强 | workspace rebuild 已输出 stats/risk/recommendations/recentFailures/timeline digest；前端已结构化消费 | `go_server/internal/contextbundle/service.go:526`, `go_server/internal/contextbundle/service.go:542`, `go_server/internal/contextbundle/service.go:574`, `go_server/internal/contextbundle/service.go:590`, `vue_web/src/views/ContextBundleView.vue:49`, `vue_web/src/views/ContextBundleView.vue:260` |
| 统一回归健康 | 全量 contract regression 通过 | `go_server/scripts/ci/contract_regression.sh` |

### 2.2 严格口径判定（Strict PRD Gate）

| 严格项 | 判定 | 说明 |
|---|---|---|
| AI planner 的多步规划 + 策略打分 | 通过（Pass） | 已具备分段、评分、策略选择与拒绝语义；执行路径支持 multi-step。 |
| Workflow 执行语义真实化 | 通过（Pass） | 已不再输出 `handled/mode/stepKey` 占位结构，改为 capability 语义结构。 |
| ContextBundle 可消费质量 | 通过（Pass） | 后端聚合深度与前端可消费视图均已补齐。 |

## 3. 本轮变更 DoD（Definition of Done）

### 3.1 AI 多步执行 DoD

- `ai.command.execute` 对可执行 steps 执行顺序稳定（按 order）。
- turn `commandIds` 包含 AI 命令自身 + 全部子命令。
- AI 助手回合文本可解释执行路径（命令类型、commandId、workflowRun 引用）。
- 前端在 multi-step 计划下不强制注入 `intentCommandType/intentPayload`，避免链路退化。

### 3.2 Workflow 语义 DoD

- step output 包含 capability contract、executor、input/output/error/recovery。
- run output 包含 capabilitySummary（step keys + status counts）。
- Run Center 可直接消费 step/run 输出，不依赖占位字段推断。

### 3.3 ContextBundle DoD

- rebuild 输出 facts/summaries/refs/timeline 的结构化统计信息（coverage/stats/risk/recommendations/recent failures）。
- Web 端 detail 面板优先展示结构化摘要，并保留 raw payload 调试入口。

## 4. 验收命令（Acceptance Commands）

1. `go test ./internal/app`（`go_server/` 下）
2. `go test ./internal/ai/planner`（`go_server/` 下）
3. `go test ./internal/access/http -run TestAPIContractRegression -count=1`（`go_server/` 下）
4. `pnpm -C vue_web typecheck`
5. `pnpm -C vue_web test:run src/views/AIWorkbenchView.spec.ts`
6. `bash go_server/scripts/ci/contract_regression.sh`

当前状态：上述命令在 2026-02-11 本轮改动中已通过。

## 5. 兼容性与接口说明（Compatibility）

- 保持不变：
  - `/api/v1` 前缀不变。
  - Command-first 语义不变。
  - 现有 stream 控制面 API 不变。
- 向后兼容扩展：
  - AI plan 响应的 `score/steps/strategyScores` 继续作为 optional fields，旧调用方不受破坏。

## 6. 后续建议（Post-Closure Enhancements）

以下属于 v0.2 增强，不再归类为本轮 strict gap：

1. Planner 从 rule-based 扩展为可插拔策略层（provider/tool 成本、风险、时延建模）。
2. ContextBundle 增加跨 workspace 的检索索引与评分排序。
3. Run Center 增加 step 级 artifacts/logs 的批量导出能力。
