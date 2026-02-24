# Goyais v0.4.0 实施计划（与新版 PRD 对齐）

> 本计划严格对齐 `docs/v_0_4_0/PRD.md` 与 `docs/v_0_4_0/TECH_ARCH.md`。
> 核心交付主线：工作区隔离 -> 资源体系 -> 对话执行队列 -> 共享审批 -> 安全审计。

---

## 0. 交付边界

### 0.1 P0 目标

1. 本地默认工作区可直接使用（免登录、全能力）。
2. 远程工作区可连接与登录，具备管理员基础后台。
3. 支持 Workspace 资源池与 Project 资源绑定。
4. Conversation 并行、单 Conversation FIFO、Stop 可用。
5. 资源导入、共享申请、审批、撤销全链路可用。
6. 模型密钥可共享，且满足高风险审批和审计要求。
7. Git 路径与非 Git 降级路径都可工作。

### 0.2 P1 不阻塞项

1. LangGraph ReAct。
2. 扩展 ABAC 条件表达式。
3. 高级命令面板与复杂工作流。

---

## Phase 1：基础骨架与统一术语

**目标**：建立统一命名与工程骨架，消除历史术语混乱。

### 工作内容

1. 目录与模块命名统一为 `conversation` 主名。
2. Hub、Worker、Desktop 建立最小可运行链路。
3. 统一错误响应与 trace_id 透传中间件。
4. 补齐本地默认工作区初始化逻辑。

### 验收标准

1. `GET /health`、基础链路可通。
2. 新建 Conversation 并写入 DB 成功。
3. 错误响应格式统一。
4. trace_id 在三层日志可串联。

### 依赖

- 无

---

## Phase 2：Workspace 隔离与权限底座

**目标**：落地 Local/Remote 双模式与 RBAC + 核心 ABAC。

### 工作内容

1. Workspace 模式字段落库（local/remote，is_default_local，tenant_id）。
2. Remote 登录流程与 token 刷新。
3. RBAC 角色基线（viewer/developer/approver/admin）。
4. ABAC 四维策略引擎（subject/resource/action/context）最小实现。
5. 菜单权限、路由权限、操作权限联动。

### 验收标准

1. Local 模式无需登录且全能力可用。
2. Remote 模式无权限用户不能访问受限路由和操作。
3. 相同 API 在不同角色返回符合预期（200/403）。
4. A/B Workspace 数据互不可见。

### 依赖

- Phase 1

---

## Phase 3：管理员后台（P0）

**目标**：交付远程工作区管理员 API + 基础 UI。

### 工作内容

1. 成员管理 API：新增、编辑、删除、停用、分配角色。
2. 角色管理 API：新增、编辑、删除、停用、分配权限。
3. 菜单/动作权限治理：编辑、删除、停用与可见性映射（hidden/disabled/readonly/enabled）。
4. 账号信息页实现：账号信息卡 + 工作区信息卡 + 连接状态卡。
5. 默认管理员账号引导与首次登录流程。

### 验收标准

1. 管理员可完成成员与角色全流程 CRUD+停用+分配。
2. 权限绑定可即时影响菜单可见性与操作可用性。
3. 非管理员不可访问管理后台，ABAC 拒绝返回 403。
4. 账号信息页显示当前账号与当前工作区关键字段。
5. 管理行为全部写审计日志。

### 依赖

- Phase 2

---

## Phase 4：资源体系与项目规范绑定

**目标**：建立 Workspace 资源池与 Project 绑定链路。

### 工作内容

1. 统一 Resource 数据模型（model/rule/skill/mcp）与共享状态机。
2. Workspace 资源池 CRUD（含 Agent 配置、Rules、Skills、MCP）。
3. 模型配置两级结构：Vendor -> Models（7 个 P0 厂商）。
4. 模型目录手工 JSON 目录加载能力（严格新格式 + 旧格式静默补齐写回 + embedded 回退）。
5. 模型目录重载触发链路（manual/page_open/scheduled）与失败审计落地。
6. ProjectConfig 落地：模型/规则/技能/MCP 四类绑定。
7. Conversation 创建时继承 ProjectConfig，并支持会话级覆盖且不反写项目。

### 验收标准

1. 可在工作区创建并管理四类资源及厂商模型目录。
2. ProjectConfig 可配置并在 Conversation 自动生效。
3. Conversation 覆盖不反写 ProjectConfig。
4. 模型目录 manual/page_open/scheduled 重载均可用，并有 requested/apply/fallback_or_failed 审计。
5. 旧目录自动补齐写回成功；补齐失败可回退 embedded 且不阻断读取。
6. 资源查询严格按 workspace_id 隔离。

### 依赖

- Phase 2

---

## Phase 5：Conversation 执行与队列闭环

**目标**：实现并行与串行队列模型，打通执行链路。

### 工作内容

1. 消息入口改为 `POST /conversations/{id}/messages`，统一创建/排队 Execution。
2. Conversation 级 FIFO 队列与 active_execution 锁。
3. Stop 能力改为 `POST /conversations/{id}/stop`。
4. 快照回滚能力 `POST /conversations/{id}/rollback` 与 ConversationSnapshot 存储。
5. SSE 事件流扩展：rollback_requested/snapshot_applied/rollback_completed。
6. Plan/Agent 与模型切换仅影响后续 Execution。
7. Conversation 导出 Markdown 落地。

### 验收标准

1. 同项目两个 Conversation 可并行执行。
2. 单 Conversation 连续发送三条消息按顺序执行。
3. Stop 只终止当前任务且后续自动继续。
4. 回滚后恢复目标消息时点的队列/worktree/Inspector 状态。
5. Markdown 导出成功且包含完整消息轨迹。
6. 事件链路完整可观测。

### 依赖

- Phase 1
- Phase 4

---

## Phase 6：Worktree/Git 与非 Git 降级

**目标**：补全代码变更闭环并支持非 Git 项目。

### 工作内容

1. Git 项目默认 worktree 隔离。
2. Diff 展示、Patch 导出、Commit/Discard。
3. Commit 后 merge-back 策略与冲突处理。
4. 非 Git 项目降级路径（无 commit/worktree，仅 diff/patch）。

### 验收标准

1. Git 项目完成“执行 -> diff -> commit/discard”闭环。
2. merge 冲突可被正确标记与处理。
3. 非 Git 项目界面与能力降级准确。
4. 相关操作均有审计。

### 依赖

- Phase 5

---

## Phase 7：资源导入、共享审批、密钥治理

**目标**：完成本地来源资源远程私有导入与共享审批全流程。

### 工作内容

1. `resource-imports` 导入 API 与 UI（模型/规则/技能/MCP 四类）。
2. `share-requests` 申请、审批、驳回、撤销 API 与 UI。
3. 私有/共享资源可见性规则（含权限可见性状态）。
4. 模型密钥共享策略（审批必需、掩码展示、可撤销、全审计）。
5. 审批权限校验（approver/admin + ABAC）。
6. 设置页与账号信息页的共享模块入口一致性实现。

### 验收标准

1. 导入后资源默认仅本人可见可用。
2. 审批通过后转共享，其他成员可用。
3. 审批拒绝后资源保持私有。
4. 撤销后共享立即失效。
5. 密钥共享相关动作均有高风险审计。
6. 四类资源共享行为在 UI 与 API 语义一致。

### 依赖

- Phase 3
- Phase 4

---

## Phase 8：安全、可靠性与可观测性强化

**目标**：将 P0 安全与稳定性要求做成发布门槛。

### 工作内容

1. Path Guard 与 Command Guard 完整覆盖。
2. Capability Prompt 全风险类别接入。
3. 审计分类标准化与查询能力。
4. Watchdog（执行超时清理）与断线恢复策略。
5. 错误码分层与前端错误映射。

### 验收标准

1. 写文件/命令/网络/删除触发确认率 100%。
2. 执行超时后锁可自动释放。
3. 关键故障场景可恢复且状态一致。
4. 审计日志可追溯执行与审批全链路。

### 依赖

- Phase 5
- Phase 7

---

## Phase 9：前端体验打磨与发布准备

**目标**：完成可发布体验与 QA 验收。

### 工作内容

1. 核心页面空状态、错误状态、加载状态完善。
2. 主屏幕信息架构对齐：左侧导航/中部对话/右侧 Inspector/底部 Hub 状态。
3. 输入区动作顺序校验：`+ -> Agent/Plan -> 模型 -> 发送`。
4. 账号信息动态菜单与设置固定菜单体验校验。
5. i18n 双语补齐。
6. 通用设置能力化：行式策略配置（启动/目录/通知/遥测/更新策略/诊断）+ 本地持久化 + 平台未接入能力显式禁用提示。
7. 更新与诊断页读取通用设置策略摘要，保留即时操作入口。
8. 无障碍（键盘导航、focus trap、对比度）检查。
9. 侧边进程管理与打包流程验证。
10. 设计一致性优化（推荐按 Pencil MCP 设计方法落地）。
11. 模型页进入自动触发目录重载；无手动刷新按钮；无写权限降级 GET。

### 验收标准

1. 12 条以上 PRD 验收场景全部通过（含回滚、导出、模型目录加载、项目配置）。
2. Hub/Desktop/Worker 自动化测试全绿。
3. 发布 checklist 满足 P0 Go 条件。
4. 关键页面视觉与交互一致性可接受。
5. 通用设置在 1440x900 首屏可见分组不少于 4 个，且策略变更即时持久化。
6. 模型页自动重载与目录扩展字段可视化验收通过（docs/homepage/auth/notes/base_urls）。

### 依赖

- Phase 6
- Phase 8

---

## Phase 10：Go/No-Go 评审

**目标**：基于证据做上线决策，不以主观判断替代验证。

### 必须满足

1. P0 功能完整可用。
2. 权限与隔离无高危漏洞。
3. 资源共享与密钥共享链路可审计可撤销。
4. 并发与队列行为符合定义。
5. 关键自动化测试通过。

### 产出物

1. 验收报告（按 PRD 场景逐条打勾）。
2. 风险清单与残留问题列表。
3. 上线决策记录（Go/No-Go + 原因）。

---

## 里程碑与并行建议

### 并行流 A（后端）

1. Phase 2 -> 3 -> 4 -> 7 -> 8

### 并行流 B（执行链路）

1. Phase 1 -> 5 -> 6

### 并行流 C（前端）

1. Phase 1 -> 3 -> 4 -> 5 -> 9

### 汇合点

1. Phase 8 与 Phase 9 汇合后进入 Phase 10。

---

## 测试计划映射

| 测试类型 | 覆盖阶段 | 核心关注 |
|----------|----------|----------|
| 单元测试 | 全阶段 | 队列状态机、回滚快照、策略评估、工具防护 |
| 集成测试 | 2,4,5,7,8 | API + DB + 队列 + 回滚 + 审批 + 模型目录加载 |
| E2E | 5,6,7,9 | 主屏幕流程、设置/账号信息菜单语义、主题模式+字体样式+字号+预设即时生效、通用设置策略即时持久化与平台降级提示、异常恢复 |
| 安全测试 | 2,7,8 | 越权、密钥泄露、注入、路径逃逸、高风险确认 |

---

## 2026-02-23 基础框架补齐门禁（增量）

1. 契约门禁：`packages/contracts/openapi.yaml` 作为唯一 API 权威源，Hub 增加契约漂移测试。
2. 联调门禁：Desktop 新增 strict 通道（`VITE_API_MODE=strict` + 禁用 fallback）。
3. 分页门禁：项目/Conversation/资源/审计列表统一 `cursor + limit`，UI 必须支持前进与回退游标栈。
4. 主题与 i18n 门禁：设置页提供真实切换控件；主题模式、字体样式、字体大小、预设主题、语言切换均即时生效并持久化。
5. 通用设置门禁：设置页 `general` 必须提供 6 组策略行式配置并即时持久化；未接入平台能力必须显式禁用并展示原因。
6. Worker 门禁：`/internal/executions/claim`、`/internal/executions/{execution_id}/events/batch`、`/internal/executions/{execution_id}/control`、`/internal/workers/register`、`/internal/workers/{worker_id}/heartbeat` 必须可用。

---

## 2026-02-24 工作区语义收口门禁（增量）

1. 工作区下拉门禁：仅显示 `本地工作区` + `用户真实新增工作区` + `新增工作区`。
2. 排序门禁：本地固定首位；远程工作区按最近使用排序并持久化；新增入口固定末位。
3. 切换门禁：切换工作区必须同步切换 Hub 地址、项目/会话数据、账号信息、权限菜单、工作区配置数据上下文。
4. 认证门禁：远程工作区 token 缺失或失效时进入 `auth_required`，不自动回退本地。
5. 存储门禁：Hub `workspaces/workspace_connections` 走 SQLite 权威路径；内存仅作为缓存。
6. 验证门禁：Hub `go test ./...`、Desktop `vitest` 与 `test:strict` 必须全绿。

---

## 2026-02-23 资源配置体系完善门禁（增量）

1. 目录门禁：模型目录来源固定为手工 `models.json`，不再依赖厂商自动同步。
2. 路径门禁：本地工作区 `catalog_root` 必须跟随 `defaultProjectDirectory` 同步；远程工作区仅管理员可写。
3. 契约门禁：`model-catalog/catalog-root/resource-configs/project-configs` 路由与 OpenAPI 必须一致。
4. 安全门禁：API Key 必须加密落库、返回掩码、测试调用全量审计。
5. 验收门禁：模型测试、MCP 连接、项目配置继承三类场景必须覆盖自动化测试。

---

## 2026-02-24 模型目录全量对齐门禁（增量）

1. 契约门禁：Hub/Desktop/OpenAPI 必须同时支持 `auth/base_urls/homepage/docs/notes` 与 `base_url_key`。
2. 兼容门禁：旧目录仅允许“静默自动补齐并写回”；补齐失败必须回退 embedded 并记录失败审计。
3. 交互门禁：模型页进入自动触发 `source=page_open` 重载；无手动刷新按钮。
4. 默认门禁：移除 `gpt-4.1` 硬编码兜底，默认模型走目录 `(Default)` 优先 + enabled 首个回退。
5. 审计门禁：`model_catalog.reload` 必须覆盖 requested/apply/fallback_or_failed，含 `workspace_id/source/reason/error/trace_id`。

---

## 2026-02-24 Worker + AI 编程闭环门禁（P0 Phase 5+6 增量）

1. 核心链路门禁：`Desktop -> Hub -> Worker` 必须走真实执行链路，`messages/stop/rollback/events/confirm` 禁 mock fallback。
2. 事件门禁：新增 `GET /v1/conversations/{conversation_id}/events`（SSE）与 `POST /internal/executions/{execution_id}/events/batch` 回传，支持 `last_event_id` 续传。
3. 审批门禁：`POST /v1/executions/{execution_id}/confirm` 与 `POST /v1/conversations/{conversation_id}/stop` 必须转换为 `execution_control_commands`，由 Worker 通过 `GET /internal/executions/{execution_id}/control` 拉取。
4. 快照门禁：Execution 必须固化 `mode_snapshot/model_snapshot/project_revision_snapshot`。
5. 多 Conversation 门禁：同项目下多 Conversation 可并行执行，单 Conversation 仍保持 FIFO + 单活执行。
6. 项目文件只读门禁：新增 `GET /v1/projects/{project_id}/files` 与 `GET /v1/projects/{project_id}/files/content`，强制路径保护。
7. 子代理门禁：P0 仅允许受控子代理并发，最大并发数 `<= 3`，且受父执行风险门禁约束。
8. 测试门禁：Hub `go test ./...`、Worker `uv run pytest`、Desktop `pnpm test` 与 `pnpm test:strict` 必须全绿。

---

## 2026-02-24 Desktop 前端治理门禁落地（增量）

1. 回滚门禁：Conversation 快照恢复不得仅依赖 `execution_ids`，必须恢复 `execution_snapshots(id/state/queue_index/message_id)`。
2. Token 门禁：`check:tokens` 必须同时覆盖“token 引用必须已定义”与“组件内禁止硬编码颜色/字体/间距/圆角”。
3. CI 门禁：Desktop job 必须执行 `lint -> test -> test:strict -> check:tokens -> check:size -> check:complexity -> coverage:gate`。
4. 规模门禁：TS/Vue 生产代码文件必须满足 `<=300` 行；通过 feature-first 子模块拆分落地（controller/store/actions/view-model）。
5. 覆盖率门禁：保留 `coverage:gate` 阻断策略，若 provider 缺失或报告缺失必须明确失败并给出修复提示。

---

## 关键风险与缓解

| 风险 | 阶段 | 缓解 |
|------|------|------|
| 权限复杂导致误授权 | 2/3/8 | 角色基线先行 + ABAC 最小闭环 + 越权回归测试 |
| 队列并发竞态 | 5/8 | 状态机原子化 + 幂等调度 + 压测 |
| 密钥共享安全风险 | 7/8 | 强制审批 + 掩码 + 轮换/撤销 + 审计告警 |
| Git 合并冲突体验差 | 6/9 | 冲突可视化与人工接管路径 |
| 文档与实现漂移 | 全阶段 | 每阶段结束做 PRD/ARCH/PLAN 一致性检查 |

---

## 文档维护规则

1. 任一阶段范围变化，必须同步更新本文件与 `PRD.md`。
2. 任一 API 变化，必须同步更新 `TECH_ARCH.md` 的接口章节。
3. 任一权限变化，必须同步更新权限模型与验收场景。
4. 任一发布条件变化，必须同步更新 Go/No-Go 条款。

---

## 2026-02-24 Worker Pull-Claim 与内部 API 硬切换同步矩阵

| change_type | required_docs_to_update | required_sections | status |
|---|---|---|---|
| 内部调度由 Hub push 改为 Worker pull claim | PRD.md, TECH_ARCH.md, IMPLEMENTATION_PLAN.md, DEVELOPMENT_STANDARDS.md | PRD 7.1/15.1, TECH_ARCH 7.2/9.2, PLAN Worker 门禁增量, STANDARDS 10.4 | done |
| 内部 API v1 硬切换 | TECH_ARCH.md, IMPLEMENTATION_PLAN.md | TECH_ARCH 9.2, PLAN Worker 门禁增量 | done |
| Hub 持久化执行全状态（替代内存主导） | TECH_ARCH.md, DEVELOPMENT_STANDARDS.md | TECH_ARCH 11.x 执行表与恢复语义, STANDARDS 10.4/11 | done |
| P0 增加受控子代理并行（<=3） | PRD.md, TECH_ARCH.md | PRD 7.1/20.2, TECH_ARCH 12.4 | done |
