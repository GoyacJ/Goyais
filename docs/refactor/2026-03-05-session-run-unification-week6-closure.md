# Session/Run 语义统一重构 Week 6 Closure 报告

- 日期：2026-03-05
- 基线提交：`345edd2`
- 关联主计划：`./2026-03-05-session-run-unification-master-plan.md`
- 关联排期：`./2026-03-05-session-run-unification-task-schedule.md`
- 关联风险：`./2026-03-05-session-run-unification-risk-register.md`

---

## 1. 结论

1. 本轮 `session/run` 语义统一重构按 Week 6 收口计划完成。
2. 对外 API 路径保持 `/v1/*`，无新增协议破坏性变更。
3. Week 6 全量验证矩阵执行完成，7/7 命令退出码均为 `0`。
4. 风险台账高优先风险项已收敛至 `Mitigated`，具备可审计证据。

---

## 2. 本轮消除的设计债

1. 可观测层命名遗留：`catalog_files.go` 审计阶段 `fallback_or_failed` 与 `fallback_*` details 键已收敛为中性命名 `recovery_or_failed` 与 `recovery_*`。
2. Week 5 遗留决策项闭环：将“是否在 Week 6 改造 `fallback_*`”从待决策转为已实施。
3. 文档治理一致性：`docs/refactor` 完成“基线三件套 + closure 归档”结构，避免基线漂移。

---

## 3. 新结构为何更优

1. 命名一致性更强：可观测事件与 details 不再传播 `fallback` 历史语义，降低后续审计规则复杂度。
2. 风险归档可执行：风险台账状态与周更记录一致，避免“顶部状态 Open、底部记录 Mitigated”的不一致噪音。
3. 收口证据完整：矩阵执行时间窗、命令、退出码、审计计数可追溯，便于后续发布复核。

---

## 4. 影响面与兼容性

### 4.1 代码影响面

1. `services/hub/internal/httpapi/catalog_files.go`
2. `services/hub/internal/httpapi/catalog_files_test.go`

### 4.2 文档影响面

1. `docs/refactor/2026-03-05-session-run-unification-task-schedule.md`
2. `docs/refactor/2026-03-05-session-run-unification-risk-register.md`
3. `docs/refactor/README.md`
4. 本文件

### 4.3 兼容性说明

1. 不变更对外协议路径与字段（`/v1/*` 保持不变）。
2. 本轮代码语义变更仅发生在内部审计可观测层。
3. Release Note 建议：记录“`model_catalog.reload` 审计阶段/详情键从 `fallback_*` 更名为 `recovery_*`，不影响业务行为”。

---

## 5. 验证证据（Week 6 全量矩阵）

执行窗口：`2026-03-05 21:02:22 +0800` -> `2026-03-05 21:02:59 +0800`

| # | 命令 | 结果 | 耗时 |
|---|---|---|---|
| 1 | `pnpm contracts:generate && pnpm contracts:check` | ✅ | 1s |
| 2 | `cd services/hub && go test ./... && go vet ./...` | ✅ | 11s |
| 3 | `pnpm lint && pnpm test && pnpm test:strict && pnpm e2e:smoke` | ✅ | 6s |
| 4 | `pnpm lint:mobile && pnpm test:mobile && pnpm build:mobile && pnpm --filter @goyais/mobile e2e:smoke` | ✅ | 10s |
| 5 | `pnpm docs:build && pnpm slides:build` | ✅ | 3s |
| 6 | `make health` | ✅ | 4s |
| 7 | `scripts/refactor/gate-check.sh --strict` | ✅ | 2s |

strict gate 审计计数：
1. `conversation/execution`: `1588`（baseline `1592`）
2. `version token`: `763`（baseline `769`）
3. `legacy/compat/fallback/alias`: `208`（baseline `358`）

---

## 6. 未完成项与后续责任

1. 当前无阻塞发布的未完成项。
2. 后续维护责任：继续按周更新 `task-schedule` 与 `risk-register`，并以 `gate-check --strict` 作为回归门禁基线。
