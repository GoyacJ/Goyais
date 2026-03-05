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
| R-006 | 兼容代码漏删导致双轨运行 | legacy/compat/fallback 残留触发旧路径 | grep 审计命中、门禁脚本告警 | Week 5 完成兼容清零并启用防回流门禁 | 回滚门禁脚本修改并补齐白名单后重试 | Open |
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
  - `R-002`: 保持 `Open`（已完成 hooks/workspace-status 的 payload 与 Hub 内部 `SessionID/SessionStatus` 字段收敛；并完成 Desktop conversation 核心层主类型迁移与兼容归一化，`pnpm lint` / `pnpm test` 通过）
  - `R-001`: 保持 `Open`（contracts/hub/desktop 已联动验证通过；shared-core 已启动 `api-project.ts` 去别名收口，但主 schema alias 尚未清零）
  - `R-005`: 保持 `Open`（本轮再次执行 `go test ./...` 与 `go vet ./...`，当前未出现新增回归）
- 新增风险：无
- 已关闭风险：无
- 需要决策：
  - 是否在下一批次直接执行 `api-project.ts` 去 alias（高收益但影响面大），还是继续按业务子域分批切换。
