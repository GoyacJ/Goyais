# Session/Run 语义统一重构风险台账

- 日期：2026-03-05
- 关联主计划：`./2026-03-05-session-run-unification-master-plan.md`
- 关联排期：`./2026-03-05-session-run-unification-task-schedule.md`
- 更新频率：每周至少 1 次；出现阻塞时即时更新。

---

## 1. 风险台账

| 风险 ID | 风险主题 | 触发条件 | 监控信号 | 缓解策略 | 回滚策略 | 当前状态 |
|---|---|---|---|---|---|---|
| R-001 | 契约错配（OpenAPI/HUB/Desktop） | 任一层仍引用旧 schema 或旧字段 | `contracts:check` 失败、Hub/Desktop 类型编译错误 | 契约先行冻结（Week 1），先改 OpenAPI 与 shared-core，再改调用方 | 回滚到上一个契约冻结提交，恢复生成类型 | Open |
| R-002 | 字段改名连锁故障 | `conversation_id/execution_id` 切换后调用链未全量更新 | 运行时 4xx、SSE 解析失败、E2E 用例失败 | 采用“单周单域收口”，每次改名后执行跨层 grep 审计 | 临时回滚到上个通过的跨层提交，重新分批改名 | Open |
| R-003 | 前端全量重命名引发路径断裂 | `modules/conversation` 迁移后 import 大面积失效 | `pnpm lint`/`pnpm test` 大量失败 | 先 `git mv` 目录，再批量修复 import，最后改 i18n key | 回滚目录迁移提交，拆分为更小批次重试 | Open |
| R-004 | DB 破坏式重建导致环境不可用 | 旧库文件与新 schema 命名冲突，启动失败 | Hub 启动报 schema 错误、集成测试异常 | 明确无历史迁移策略，统一重建流程与测试夹具 | 回滚存储层提交并恢复旧 schema，重新设计重建脚本 | Open |
| R-005 | 测试回归窗口过大 | 多周改动叠加，问题定位困难 | Week 5/6 出现大量未知回归 | 周周执行对应门禁，不积压未验证改动 | 回滚最近一周增量，逐批 rebase 修复 | Open |
| R-006 | 兼容代码漏删导致双轨运行 | legacy/compat/fallback 残留触发旧路径 | grep 审计命中、门禁脚本告警 | Week 5 完成兼容清零并启用防回流门禁 | 回滚门禁脚本修改并补齐白名单后重试 | Mitigated |
| R-007 | 权限键改名造成鉴权异常 | `conversation.*` -> `session.*` 切换不完整 | 403 异常、审计事件键名混乱 | 权限键改名与审计改名同一周完成并联测 | 回滚权限模型改名提交，保留最小可用键集 | Open |
| R-008 | 文档基线漂移 | 团队并行新增平行计划文档 | `docs/refactor` 出现重复主计划 | 固定以 `master-plan/task-schedule/risk-register` 为唯一基线 | 回滚新增平行文档并在 README 再次声明 | Open |

---

## 2. 风险分级规则

1. 高风险：影响契约一致性、数据可用性、发布可用性。
2. 中风险：影响单栈可用性或需要手工修复。
3. 低风险：仅影响局部开发效率，不影响基线可用。

说明：`R-001`、`R-003`、`R-004`、`R-005` 为优先跟踪项，周会必须逐条复盘。

---

## 3. 监控与触发机制

1. 每周固定触发：执行排期对应验收命令并记录结果。
2. 变更触发：出现新增 4xx/5xx、测试失败、门禁失败时即时更新状态。
3. 发布触发：Week 6 执行 `make health` 前必须将高风险项更新为 `Mitigated` 或 `Closed`。

---

## 4. 回滚原则

1. 回滚以“最近可用提交”为单位，不跨周回滚多个主题。
2. 回滚后必须补充“失败根因 + 重试策略”，防止重复失败。
3. 回滚仅用于恢复可用性，不作为长期停留状态。

---

## 5. 周更模板（固定使用）

### 更新记录（YYYY-MM-DD）

- 更新人：
- 周次：
- 风险变更：
  - `R-xxx`: `Open -> Mitigated`（原因）
- 新增风险：
- 已关闭风险：
- 需要决策：

### 更新记录（2026-03-05）

- 更新人：Codex
- 周次：Week 1 / Week 2（执行中）
- 风险变更：
  - `R-001`: 保持 `Open`（OpenAPI 已删除 `Conversation/Execution` 及相关 alias schema，contracts/shared-core/desktop/hub 验证通过；仍需继续清理字段级旧语义与跨域调用残留）
  - `R-002`: 保持 `Open`（Desktop 会话详情 hydration 输入已切换为 `session/runs`；尚有运行时字段 `conversation_id/execution_id` 残留待后续分批替换）
  - `R-005`: 保持 `Open`（本轮执行 `pnpm lint` / `pnpm test` / `go test ./...` / `go vet ./...` / `scripts/refactor/gate-check.sh` 全通过，未出现新增回归）
- 新增风险：无
- 已关闭风险：无
- 需要决策：
  - 已决策：下一批次优先处理 Hub `internal/httpapi` 字段级旧语义清零，再推进 Desktop i18n 与视图文案收口。

### 更新记录（2026-03-05，Week 3 Hub 先行批次）

- 更新人：Codex
- 周次：Week 3（执行中）
- 风险变更：
  - `R-001`: 保持 `Open`（`contracts:generate/check` 与 Hub/Desktop 关键验证通过；权限键与 runtime schema 命名已收敛，但全仓旧语义尚未清零）
  - `R-004`: 保持 `Open`（runtime 表/索引去 `_v1` 已落地并通过 `go test ./...`；仍需按 Week 3 清单执行“删除本地 sqlite 后双冷启动”实机验证）
  - `R-007`: `Open -> Mitigated`（Hub 权限模型、默认角色权限、权限字典、handler 审计键与 Desktop 权限消费断言已同步切到 `session.*`/`run.control`，本轮未出现 403 回归）
  - `R-005`: 保持 `Open`（本轮新增执行 `scripts/refactor/gate-check.sh` 并通过，未出现新增回归）
- 新增风险：无
- 已关闭风险：无
- 需要决策：
  - 是否在下一批次将 `R-004` 的 DB 双冷启动验证纳入固定 CI preflight（当前仍为手工门禁）。

### 更新记录（2026-03-05，Week 3 收口补充批次）

- 更新人：Codex
- 周次：Week 3（收口中）
- 风险变更：
  - `R-004`: `Open -> Mitigated`（新增并通过 `TestOpenAuthzStoreSupportsRuntimeSchemaAfterTwoColdStarts`，两轮均执行“删除 sqlite 文件 -> 冷启动重建 -> 校验 session/run/events/changeset/hooks”链路；并补充 router 重启持久化用例验证）
  - `R-001`: 保持 `Open`（契约与跨栈校验持续通过，但全仓旧语义审计仍有存量）
  - `R-005`: 保持 `Open`（本批次继续执行 `go test/go vet + contracts + lint/test + gate-check` 均通过）
- 新增风险：无
- 已关闭风险：无
- 需要决策：
  - Week 4 目录迁移前，是否将双冷启动验证脚本化接入 `scripts/refactor/gate-check.sh`，将 `R-004` 从 `Mitigated` 推进到可持续 `Closed`。

### 更新记录（2026-03-05，Week 4 Batch A/B/C/D）

- 更新人：Codex
- 周次：Week 4（执行中）
- 风险变更：
  - `R-003`: `Open -> Mitigated`（目录迁移 `modules/conversation -> modules/session` 已落地；首轮 lint/test 失败由变量误替换触发，已同批修复并通过 `pnpm lint`、`pnpm test`、`pnpm test:strict`、`pnpm e2e:smoke`）
  - `R-005`: 保持 `Open`（本批次采用“分批门禁 + 日终全量门禁”，完整链路通过；后续仍需 Week 5/6 连续验证）
  - `R-001`: 保持 `Open`（contracts + hub + desktop 验证通过，但全仓旧语义仍有存量，审计命中 `1620`）
- 新增风险：
  - 无（Batch C 的变量误替换已在同批次闭环，不升格为新风险）
- 已关闭风险：
  - 无
- 需要决策：
  - Week 5 是否优先清理 `legacy/compat/fallback/alias` 存量（当前审计 `370` 持平）。

### 更新记录（2026-03-05，Week 5-1）

- 更新人：Codex
- 周次：Week 5（执行中）
- 风险变更：
  - `R-006`: `Open -> Mitigated`（`execution_enqueued` 与 `legacy_event_type` 兼容输出已清理，Hub 部分查询路径移除 repository-error fallback，且 `gate-check.sh` 已新增增量阻断词并本地验证通过）
  - `R-001`: 保持 `Open`（contracts/shared-core/hub/desktop 全链路验证通过，但旧语义命中仍有存量，需继续压降）
  - `R-005`: 保持 `Open`（本批次继续执行 strict/e2e 与 gate-check，均通过；仍需 Week 5 后续批次持续验证）
- 新增风险：无
- 已关闭风险：无
- 需要决策：
  - Week 5-2 是否将增量阻断从“仅新增代码”扩展为“总量阈值 + 白名单”模式（建议在下一批次落地）。

### 更新记录（2026-03-05，Week 5-2）

- 更新人：Codex
- 周次：Week 5（执行中）
- 风险变更：
  - `R-006`: 保持 `Mitigated`（`fallback to in-memory map` 在 `internal/httpapi` 命中已从 `8 -> 0`，并完成门禁“新增阻断 + 总量不回升 + 白名单”落地）
  - `R-001`: 保持 `Open`（Week 5-2 全链路验证通过，但旧语义审计仍有存量：`conversation/execution=1586`）
  - `R-005`: 保持 `Open`（本批次继续执行 contracts/hub/desktop/strict/e2e/gate-check 全绿，仍需持续观察 Week 5 后续切片）
- 新增风险：无
- 已关闭风险：无
- 需要决策：
  - Week 5 后续是否将 `gate-baseline` 由静态数字切换为“按通过提交自动刷新”的受控流程（避免人工漂移）。

### 更新记录（2026-03-05，Week 5-3）

- 更新人：Codex
- 周次：Week 5（收口中）
- 风险变更：
  - `R-006`: 保持 `Mitigated`（`fallback to in-memory map` 维持 `0`；白名单已收敛并移除无效 `Alias` 规则）
  - `R-001`: 保持 `Open`（契约与跨栈门禁通过，但 `conversation/execution` 命中仍为 `1588`，需继续在 Week 6 前压降）
  - `R-005`: 保持 `Open`（E1/E2/E3 分片验证持续全绿，且 Week 5 末跨栈命令矩阵已补齐通过）
- 新增风险：无
- 已关闭风险：无
- 需要决策：
  - Week 6 前是否将 `catalog_files.go` 中 `fallback_*` 审计字段切换到中性命名（仅可观测性层，不影响协议）。
