# Goyais v0.4.0 开发规范（Engineering Executable Standards）

> 本文档是 v0.4.0 的工程执行规范，目标是“可落地、可检查、可阻断”。  
> 业务语义以 `PRD.md` 为准；架构事实以 `TECH_ARCH.md` 为准；实施顺序以 `IMPLEMENTATION_PLAN.md` 为准。

---

## 1. 文档定位与优先级

### 1.1 目标

1. 将开发规范从“原则描述”升级为“工程门禁规范”。
2. 用统一阈值和流程约束代码质量，避免过度设计与无边界扩张。
3. 保证跨 Hub/Worker/Desktop 的一致性与可维护性。

### 1.2 适用范围

1. 适用于 v0.4.0 全部代码与文档变更。
2. 适用于 Go、Python、TypeScript/Vue、Rust（Tauri）代码。
3. 适用于单元测试、集成测试、E2E、CI 配置和 PR 审查流程。

### 1.3 文档优先级

1. 安全红线 > 本文档 > 团队个人偏好。
2. 当本文档与其他文档冲突时，以 `PRD.md` 业务定义为最高裁决。

---

## 2. 规范关键字与适用方式（RFC2119）

### 2.1 关键字定义

1. `MUST`：强制要求，不满足即视为违规。
2. `MUST NOT`：明确禁止，出现即视为违规。
3. `SHOULD`：强烈建议，偏离需在 PR 说明理由。
4. `MAY`：可选项，由模块 owner 判断。

### 2.2 执行方式

1. 所有 `MUST/MUST NOT` 条款默认接入 CI 阻断。
2. `SHOULD` 条款默认作为 Review 告警项。
3. 唯一可豁免路径是 `StandardsExceptionADR`（见第 12 章）。

---

## 3. 软件工程理论落地（实用工程版）

### 3.1 SOLID/KISS/YAGNI/DRY 落地要求

1. `MUST` 保持单一职责：模块变更原因不超过一个业务轴。
2. `MUST` 优先简单实现（KISS）：先可运行再抽象。
3. `MUST NOT` 为“未来可能用到”提前抽象（YAGNI）。
4. `MUST` 消除重复逻辑（DRY），但 `MUST NOT` 通过过度抽象制造复杂性。

### 3.2 高内聚低耦合可执行标准

1. `MUST` 让模块围绕单一领域聚合（权限、共享、执行调度、UI 模块）。
2. `MUST` 避免横向依赖环；出现循环依赖即阻断。
3. `SHOULD` 降低跨域调用深度，单请求链路跨域跳转不超过 3 层。
4. `MUST` 使用清晰边界：handler -> service -> repository（后端），module view -> module service/store（前端）。

### 3.3 抽象引入触发规则

1. `MUST` 满足“第二实现触发”原则：至少存在两个有效实现或一个稳定外部边界，才允许提接口抽象。
2. `MUST NOT` 因“写起来更优雅”单独引入新层。
3. `SHOULD` 在大改架构前新增 ADR 说明收益、风险和回滚策略。

---

## 4. 设计模式白名单与触发条件

### 4.1 白名单（仅允许以下模式）

| Pattern | 适用场景 | 触发条件（MUST） | 禁止条件（MUST NOT） |
|---|---|---|---|
| `Strategy` | 多策略切换（如多 Provider） | 至少 2 种真实策略已存在 | 只有 1 种策略时提前抽象 |
| `Adapter` | 外部协议差异统一 | 存在稳定外部接口差异 | 仅为包装内部函数而创建 |
| `Factory` | 创建流程复杂或有变体 | 构造逻辑包含分支和依赖注入 | 仅替代 `new`/构造函数 |
| `Repository` | 数据访问隔离 | 需要统一数据源访问边界 | 对简单查询硬分层导致冗余 |
| `State` | 显式状态迁移 | 状态>=3 且迁移规则复杂 | 用于简单 if-else 状态切换 |

### 4.2 模式治理契约

```text
PatternWhitelistRule {
  pattern: "Strategy|Adapter|Factory|Repository|State"
  trigger: string[]
  anti_trigger: string[]
  examples: string[]
}
```

### 4.3 评审要求

1. `MUST` 在 PR 描述中说明使用模式、触发条件、替代方案评估。
2. `MUST` 附上测试证明模式引入后收益真实存在。

---

## 5. 反模式与过度设计禁止项

### 5.1 明确禁止

1. `MUST NOT` 引入全局 Service Locator。
2. `MUST NOT` 为单实现场景引入多态层。
3. `MUST NOT` 用多层 DTO/VO 映射掩盖简单业务。
4. `MUST NOT` 在前端全局平铺 views/components 导致领域边界丢失。
5. `MUST NOT` 为“可扩展性想象”增加无使用路径代码。

### 5.2 识别信号（出现任意项即需重构评估）

1. 单模块持续承担多个领域职责。
2. 关键路径调用链过深且难追踪。
3. 代码评审中无法清晰解释抽象收益。

---

## 6. 文件规模与拆分规则（硬阈值）

### 6.1 单文件行数上限（MUST）

| 语言 | 最大行数 |
|---|---:|
| Go | `<= 400` |
| Python | `<= 350` |
| TypeScript / TSX / Vue | `<= 300` |
| Rust | `<= 350` |

### 6.2 函数与组件复杂度提示阈值

1. `SHOULD` 函数长度控制在 80 行以内。
2. `SHOULD` Vue 单组件业务逻辑控制在 200 行以内（模板+脚本+样式总和）。

### 6.3 拆分规则

1. 超限文件 `MUST` 拆分为“领域逻辑 + 基础设施 + 辅助工具”。
2. 拆分后 `MUST` 保持语义内聚，不允许机械切片。
3. 生成文件、迁移文件、测试 fixture 可标记例外，但 `MUST` 在 PR 注明。

### 6.4 工程门禁契约

```text
FileSizePolicy {
  lang: "go|py|ts|tsx|vue|rs"
  max_lines: number
  blocking: true
}
```

---

## 7. 公共设计与复用规范

### 7.1 复用原则

1. `MUST` 将跨模块稳定能力沉淀到共享层。
2. `MUST NOT` 将领域私有逻辑上提到共享层。
3. `SHOULD` 共享前先评估被至少两个模块真实复用。

### 7.2 共享对象范围

1. 共享 UI 组件：统一放 `src/shared/ui`。
2. 共享布局：统一放 `src/shared/layouts`。
3. 共享工具：统一放 `src/shared/utils`。
4. 共享类型契约：按领域定义，避免全局巨型 types 文件。

### 7.3 后端共享规则

1. `MUST` 共享通用中间件、错误模型、审计接口。
2. `MUST NOT` 跨领域直接调用彼此私有 service。

---

## 8. 前端模块化规范（Feature-First）

### 8.1 目录规范（MUST）

```text
src/
  modules/
    <domain>/
      views/
      components/
      store/
      services/
      schemas/
      tests/
  shared/
    ui/
    layouts/
    utils/
```

### 8.2 强制规则

1. `MUST` 采用按业务模块分片（feature-first）。
2. `MUST NOT` 使用全局平铺 `src/views/*` 作为主组织方式。
3. `MUST` 让模块内 view 与模块 service/store 同域维护。
4. `SHOULD` 每个模块提供 `index.ts` 作为对外边界。

### 8.3 模块契约

```text
FrontendModuleLayoutContract {
  required_dirs: ["views","components","store","services","schemas","tests"]
  shared_dirs: ["src/shared/ui","src/shared/layouts","src/shared/utils"]
  flat_views_forbidden: true
}
```

---

## 9. 统一样式与 Token 三层规范

### 9.1 三层 Token 模型（MUST）

1. `global token`：原子值（色板、字号、间距、圆角）。
2. `semantic token`：语义映射（primary-text、danger-bg）。
3. `component token`：组件级语义变量（button-bg、card-border）。

### 9.2 禁止项（MUST NOT）

1. 在组件内硬编码颜色值（如 `#123456`）。
2. 在组件内硬编码字体族、字号、间距、圆角。
3. 在主题切换逻辑中绕过 semantic token 直接读 global token。

### 9.3 Token 契约

```text
TokenLayerContract {
  layers: ["global","semantic","component"]
  direct_hardcode_forbidden: true
  semantic_as_required_bridge: true
}
```

### 9.4 执行建议

1. `SHOULD` 在 CI 增加 token 硬编码扫描。
2. `SHOULD` 核心页面统一组件库实现，避免局部样式漂移。
3. `MUST` 主题增强采用属性驱动覆盖层（如 `theme-profiles.css`），禁止直接改写 `tokens.css` 的设计源定义。

---

## 10. 质量门禁（复杂度、覆盖率、依赖关系）

### 10.1 复杂度门禁（MUST）

1. 圈复杂度 `<= 10`。
2. 认知复杂度 `<= 15`。
3. 超阈值即阻断，除非有 ADR 豁免。

### 10.2 覆盖率门禁（MUST）

1. 核心模块覆盖率 `>= 80%`。
2. 总体覆盖率 `>= 70%`。

核心模块定义：权限、资源共享、密钥治理、执行调度、Conversation 回滚快照。

### 10.3 依赖关系门禁（MUST）

1. 禁止循环依赖。
2. 禁止跨层反向依赖（UI 依赖基础设施内部实现）。
3. 禁止模块私有实现被外部模块直接引用。

### 10.4 语义一致性门禁（MUST）

1. 队列门禁：单 Conversation 必须严格 FIFO，且同一时刻最多一个活动执行。
2. 回滚门禁：`rollback` 必须恢复消息游标、队列状态、worktree_ref、Inspector 状态。
3. 项目配置门禁：Conversation 覆盖不得反写 ProjectConfig。
4. 菜单权限门禁：动态菜单可见性必须与后端权限一致（hidden/disabled/readonly/enabled）。
5. 模型目录门禁：`manual/page_open/scheduled` 重载路径都需可用，严格新格式校验失败必须回退 embedded 并写审计日志。
6. 目录迁移门禁：旧目录仅允许静默自动补齐写回；写回失败必须产出 `fallback_or_failed` 审计。
7. 主题门禁：`theme mode/font style/font scale/preset` 必须全局即时生效并持久化，且具备可回归自动化测试。
8. 通用设置门禁：`general settings` 必须提供 6 组策略型行式配置并即时持久化；系统能力未接入平台时必须显式禁用并展示原因文案。
9. Worker 调度门禁：内部执行链路必须使用 pull-claim（`claim + lease + heartbeat + control poll + events batch`）；禁止恢复 Hub push Worker 语义。
10. 会话恢复门禁：Desktop 进入 Conversation 必须先调用详情接口回填；仅后端无历史消息时允许欢迎语兜底。
11. 流路由门禁：事件应用必须以 `event.conversation_id` 路由，禁止将多会话事件写入同一 runtime。
12. 风险分级门禁：`run_command` 的 `pwd/ls/rg --files/git status/cat` 归类 `low`，其他命令维持 `high/critical`。
13. 执行策略门禁：Agent 模式不得进入 `confirming/wait_confirmation`，高风险调用直接执行并审计；Plan 模式高风险调用必须返回拒绝。
14. Agent 配置门禁：`max_model_turns/show_process_trace/trace_detail_level` 必须由工作区配置中心统一管理，并通过后端 API 权威读写。
15. 快照门禁：Execution 创建时必须固化 `agent_config_snapshot`，运行中 execution 不得被设置页变更重配。
16. 回合门禁：触达 `max_turns` 时优先软收敛到 `execution_done(truncated=true, reason=MAX_TURNS_REACHED)`；仅总结失败时允许 `MAX_TURNS_EXCEEDED`。
17. 过程流门禁：对话区必须可见 `thinking_delta/tool_call/tool_result/execution_started`，且执行终态后不得残留“正在思考/运行中可停止”。

### 10.5 门禁契约

```text
ComplexityPolicy {
  cyclomatic_max: 10
  cognitive_max: 15
  blocking: true
}

CoveragePolicy {
  core_modules_min: 0.80
  overall_min: 0.70
  blocking: true
}

SemanticGatePolicy {
  queue_fifo_required: true
  snapshot_rollback_required: true
  project_config_override_non_persistent: true
  permission_visibility_consistency: true
  model_catalog_reload_auditable: true
  general_settings_strategy_persistent: true
  blocking: true
}

QualityGateResult {
  check_id: string
  threshold: string
  actual: string
  blocking: boolean
}
```

---

## 11. CI 强制策略与失败处理

### 11.1 CI 门禁策略

1. `MUST` 默认 `blocking=true`。
2. `MUST` 覆盖以下检查：行数、复杂度、覆盖率、token 硬编码扫描、目录结构规则、语义一致性门禁。
3. `MUST` 输出可读失败原因和修复建议。
4. `MUST` 在 CI 产物中输出以下专项报告：
   - queue/rollback 语义测试报告
   - 权限可见性一致性报告
   - 项目配置继承与覆盖报告
   - 模型目录加载稳定性报告
   - 通用设置策略即时持久化与平台降级提示报告
5. `MUST` 执行 Desktop strict 联调通道（`VITE_API_MODE=strict` + `VITE_ENABLE_MOCK_FALLBACK=false`）。

### 11.2 失败处理流程

1. 开发者修复后重新触发 CI。
2. 若需临时豁免，`MUST` 提交 `StandardsExceptionADR`。
3. 无 ADR 的门禁绕过请求 `MUST NOT` 被批准。
4. 语义门禁失败（10.4）不得通过“仅补文档”绕过，必须有代码或测试修复。

---

## 12. 例外流程（ADR + 到期治理）

### 12.1 例外适用场景

1. 线上紧急修复且存在时间窗口压力。
2. 迁移过渡阶段必须短期共存旧实现。

### 12.2 例外申请契约

```text
StandardsExceptionADR {
  adr_id: string
  owner: string
  scope: string
  reason: string
  risk: string
  mitigation: string
  expiry_date: string
  rollback_plan: string
}
```

### 12.3 例外治理规则

1. `MUST` 设置到期日（默认不超过 30 天）。
2. `MUST` 指定 owner 和回收计划。
3. 到期未清理 `MUST` 升级为阻断问题。

---

## 13. PR 审查清单与完成定义（DoD）

### 13.1 PR 审查清单（MUST）

1. 是否符合文件行数和复杂度阈值。
2. 是否引入非白名单设计模式或过度抽象。
3. 是否保持工作区隔离与权限校验边界。
4. 是否覆盖拒绝路径测试（权限拒绝、共享审批拒绝、Plan 高风险拒绝、Stop、回滚失败、异常恢复）。
5. 是否满足 token 三层规则且无硬编码样式。
6. 是否补充必要审计日志与 trace_id 传播。
7. 是否完成动态菜单与固定菜单语义校验（账号信息/设置）。
8. 是否验证 ProjectConfig 继承与 Conversation 覆盖语义。
9. 是否验证模型目录加载（手动 + 定时）及 JSON 异常路径。
10. 是否验证模型页进入自动重载且无手动刷新按钮。
11. 是否验证禁用模型不可新建、历史模型配置可读可测。
12. 是否与 PRD/TECH_ARCH/PLAN 语义一致。

### 13.2 完成定义（DoD）

一项变更只有在以下全部满足时可标记完成：

1. 业务语义与 `PRD.md` 对齐。
2. 架构边界与 `TECH_ARCH.md` 对齐。
3. 实施阶段目标与 `IMPLEMENTATION_PLAN.md` 对齐。
4. CI 门禁全部通过，或有有效 ADR 例外。
5. 测试证据完整且可追溯。
6. 同步矩阵状态准确（done/missing），不得遗漏必改文档项。

---

## 14. 与 PRD/TECH_ARCH/PLAN 的一致性维护规则

1. 修改业务规则时，`MUST` 同步更新 `PRD.md`。
2. 修改接口/状态机/模型时，`MUST` 同步更新 `TECH_ARCH.md`。
3. 修改阶段目标或门禁策略时，`MUST` 同步更新 `IMPLEMENTATION_PLAN.md`。
4. 四份文档任意冲突时，先修复冲突再合并代码，`MUST NOT` 带冲突进入主分支。
5. 文档同步 `MUST` 附带变化矩阵，至少包含：
   - `change_type`
   - `required_docs_to_update`
   - `required_sections`
   - `status`
6. 若 `status=missing` 存在，`MUST NOT` 标记任务完成。

---

## 15. 文档级验收场景（用于检查规范可执行性）

1. 出现超限文件时，能按第 6 章直接判定是否阻断或需 ADR。
2. 出现复杂度超阈值函数时，能按第 10 章直接阻断并给出整改路径。
3. 覆盖率不足时，能按第 10/11 章明确阻断与例外流程。
4. 出现非白名单模式时，能按第 4/5 章明确违规判定。
5. 前端出现全局 views 平铺时，能按第 8 章明确违规。
6. 组件硬编码色值/间距时，能按第 9 章明确违规。
7. 紧急修复场景下，能按第 12 章执行 ADR 豁免并追踪到期。
8. 回滚语义变更时，能按第 10.4 阻断“只回滚文本不回滚快照”的错误实现。
9. 项目配置语义变更时，能验证“Conversation 覆盖不反写 ProjectConfig”。
10. 菜单权限变更时，能验证动态菜单/固定菜单行为与权限一致。
11. 模型目录策略变更时，能验证手动/定时重载与失败审计。
12. 模型目录全量对齐变更时，能验证 `auth/base_urls/base_url_key` 契约与禁用模型门禁。
13. PR 审查可按第 13 章逐项打勾并形成可追溯证据。
14. 会话稳定性回归时，能验证“重启恢复 + 执行占位状态 + 多会话不串流”。
15. Agent 配置变更后，能验证“仅新 Execution 生效，运行中 Execution 不切换”。
16. 大任务达到回合上限时，能验证“优先输出截断总结，不直接抛错”。
17. 执行完成后，能验证过程流收敛与运行占位清理，不残留错误状态。

---

## 16. 2026-02-24 工作区语义收口同步矩阵

| change_type | required_docs_to_update | required_sections | status |
|---|---|---|---|
| Workspace 持久化与列表语义 | PRD.md, TECH_ARCH.md | PRD 5.x/9.x/14.x, TECH_ARCH 11.1/9.x | done |
| 工作区切换上下文行为 | PRD.md, TECH_ARCH.md | PRD 5.2/9.2/16.x, TECH_ARCH 14.x/20.x | done |
| 测试门禁与验收项 | IMPLEMENTATION_PLAN.md, DEVELOPMENT_STANDARDS.md | Phase 2/3 验收、DoD/门禁 | done |

---

## 17. 2026-02-23 资源配置体系完善同步矩阵

| change_type | required_docs_to_update | required_sections | status |
|---|---|---|---|
| 模型目录改为手工 JSON 目录加载 | PRD.md, TECH_ARCH.md | PRD 6.3/14.1, TECH_ARCH 3.2/6.5/20.4 | done |
| API 与数据表扩展（catalog-root/resource-configs/project-configs） | TECH_ARCH.md, IMPLEMENTATION_PLAN.md | TECH_ARCH 9.1/11.2/20.5/20.6, PLAN Phase 4 | done |
| 工程门禁更新（JSON 校验与重载审计） | DEVELOPMENT_STANDARDS.md | 10.4, 10.5, 11.1, 13.1, 15 | done |

---

## 18. 2026-02-24 Worker + AI 编程闭环同步矩阵

| change_type | required_docs_to_update | required_sections | status |
|---|---|---|---|
| 实时事件流与执行控制协议（SSE + stop/internal events，无 confirm） | PRD.md, TECH_ARCH.md, IMPLEMENTATION_PLAN.md | PRD 14.x/24, TECH_ARCH 9.x/10.x/11.x, PLAN Phase 5/6 增量门禁 | done |
| Execution 快照字段扩展（mode/model/project revision） | PRD.md, TECH_ARCH.md | PRD 14.2, TECH_ARCH 7.5/11.3/20.x | done |
| 同项目多 Conversation + 项目文件只读能力 | PRD.md, TECH_ARCH.md, IMPLEMENTATION_PLAN.md | PRD 7/14, TECH_ARCH 7/9/11, PLAN Phase 5/6 | done |
| 核心链路 strict 化（禁 fallback） | IMPLEMENTATION_PLAN.md, DEVELOPMENT_STANDARDS.md | PLAN 门禁增量, STANDARDS 11/13/14 | done |

---

## 19. 2026-02-24 模型目录全量对齐同步矩阵

| change_type | required_docs_to_update | required_sections | status |
|---|---|---|---|
| 模型目录 Vendor 扩展字段（auth/base_urls/docs/notes） | PRD.md, TECH_ARCH.md | PRD 6.3/14.2, TECH_ARCH 6.5/20.4 | done |
| `ModelSpec` 新增 `base_url_key` | TECH_ARCH.md | TECH_ARCH 11.2/20.6 | done |
| 目录严格格式 + 静默补齐 + 回退策略 | PRD.md, TECH_ARCH.md, DEVELOPMENT_STANDARDS.md | PRD 6.3/19.1, TECH_ARCH 6.5/20.4, STANDARDS 10.4/15 | done |
| 模型页进入自动重载（无手动按钮） | PRD.md, IMPLEMENTATION_PLAN.md | PRD 19.1, PLAN Phase 4/9 验收 | done |
| 重载失败审计细化（manual/page_open/scheduled） | TECH_ARCH.md, DEVELOPMENT_STANDARDS.md | TECH_ARCH 15.3/20.4, STANDARDS 10.4/13/15 | done |

---

## 20. 2026-02-24 Worker Pull-Claim 与内部 API 硬切换同步矩阵

| change_type | required_docs_to_update | required_sections | status |
|---|---|---|---|
| 内部调度由 Hub push 改为 Worker pull claim | PRD.md, TECH_ARCH.md, IMPLEMENTATION_PLAN.md, DEVELOPMENT_STANDARDS.md | PRD 7.1/15.1, TECH_ARCH 7.2/9.2, PLAN 2026-02-24 Worker 门禁增量, STANDARDS 10.4 | done |
| 内部 API v1 硬切换 | TECH_ARCH.md, IMPLEMENTATION_PLAN.md | TECH_ARCH 9.2, PLAN 2026-02-24 Worker 门禁增量 | done |
| Hub 持久化执行全状态（替代内存主导） | TECH_ARCH.md, DEVELOPMENT_STANDARDS.md | TECH_ARCH 11.x 执行表与恢复语义, STANDARDS 10.4/11 | done |
| P0 增加受控子代理并行（<=3） | PRD.md, TECH_ARCH.md | PRD 7.1/20.2, TECH_ARCH 12.4 | done |

---

## 21. 2026-02-24 Desktop 前端治理同步矩阵

| change_type | required_docs_to_update | required_sections | status |
|---|---|---|---|
| Conversation 回滚快照恢复执行状态（`execution_snapshots`） | TECH_ARCH.md, IMPLEMENTATION_PLAN.md | TECH_ARCH 3.2/7.4, PLAN Desktop 前端治理门禁增量 | done |
| Token 引用完整性与硬编码样式阻断脚本 | DEVELOPMENT_STANDARDS.md, IMPLEMENTATION_PLAN.md | STANDARDS 9.x/11.1, PLAN Desktop 前端治理门禁增量 | done |
| CI 增量门禁（strict/tokens/size/complexity/coverage） | DEVELOPMENT_STANDARDS.md, IMPLEMENTATION_PLAN.md | STANDARDS 6.4/10.1/10.2/11.1, PLAN Desktop 前端治理门禁增量 | done |
| TS/Vue 超行数文件拆分（feature-first 子模块化） | DEVELOPMENT_STANDARDS.md | STANDARDS 6.1/6.3/13.1 | done |

---

## 22. 2026-02-24 状态聚合接口补齐同步矩阵

| change_type | required_docs_to_update | required_sections | status |
|---|---|---|---|
| 状态聚合接口补齐 | TECH_ARCH.md | API 摘要/状态接口 | done |

## 23. 2026-02-24 会话稳定性与并发显示同步矩阵

| change_type | required_docs_to_update | required_sections | status |
|---|---|---|---|
| Conversation 详情回填与重启恢复门禁 | PRD.md, TECH_ARCH.md, IMPLEMENTATION_PLAN.md | PRD 14.1/17, TECH_ARCH 20.9, PLAN Phase 5 | done |
| 会话订阅策略 `active + running/queued` 与防串流路由 | PRD.md, TECH_ARCH.md, IMPLEMENTATION_PLAN.md | PRD 7.1/16.3, TECH_ARCH 10.3/20.9, PLAN Phase 5 | done |
| Worker 默认并发 3 与项目上下文注入 | PRD.md, TECH_ARCH.md, IMPLEMENTATION_PLAN.md | PRD 7.1/17, TECH_ARCH 12.4/16, PLAN Worker 门禁增量 | done |
| `run_command` 只读命令低风险分类 | PRD.md, TECH_ARCH.md, DEVELOPMENT_STANDARDS.md | PRD 15.3, TECH_ARCH 13.2, STANDARDS 10.4/13.1 | done |
| Agent 模式移除风险确认链路（删除 confirm API / confirming 状态） | PRD.md, TECH_ARCH.md, IMPLEMENTATION_PLAN.md, DEVELOPMENT_STANDARDS.md | PRD 14.1/15.3/24, TECH_ARCH 3.3/9.1/9.2/10.1/12.1, PLAN Phase 5/8, STANDARDS 10.4/13 | done |

## 24. 2026-02-24 Agent 配置中心化与执行过程可视化同步矩阵

| change_type | required_docs_to_update | required_sections | status |
|---|---|---|---|
| Workspace Agent 配置中心化（`/workspace/agent` + `agent-config` API） | PRD.md, TECH_ARCH.md, DEVELOPMENT_STANDARDS.md | PRD 12.1/16.2, TECH_ARCH 9.1/20.10, STANDARDS 10.4/13 | done |
| Execution 快照固化与仅新 execution 生效 | PRD.md, TECH_ARCH.md, DEVELOPMENT_STANDARDS.md | PRD 14.2/16.3, TECH_ARCH 11.x/20.10, STANDARDS 10.4/15 | done |
| `max turns` 软收敛与错误路径收口 | PRD.md, TECH_ARCH.md, IMPLEMENTATION_PLAN.md, DEVELOPMENT_STANDARDS.md | PRD 16.3/19, TECH_ARCH 12/20.10, PLAN Phase 5, STANDARDS 10.4/11 | done |
| 对话区过程流展示与终态收敛 | PRD.md, TECH_ARCH.md, DEVELOPMENT_STANDARDS.md | PRD 16.3/19, TECH_ARCH 14.2/20.10, STANDARDS 10.4/11/15 | done |
