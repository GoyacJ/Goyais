# Goyais v0.4.0 产品需求文档（PRD）

## 1. 文档信息

- 文档版本：v0.4.0
- 日期：2026-02-22
- 文档状态：基线版（Rewrite Authority）
- 性质：Clean-Slate Rewrite（完全重写，不兼容旧版本）
- 协议：Apache-2.0
- 产品形态：桌面端 AI 智能平台（AI Coding + 通用 Agent）
- 目标平台：macOS（P0），Windows/Linux（P1）
- 目标读者：产品、架构、研发、测试、运维、项目管理

---

## 2. 产品定位与价值主张

### 2.1 产品定位

Goyais 是一个 **AI 智能平台**，不是单一聊天工具。平台同时覆盖两类能力：

1. **AI Coding**：类似 Codex / Claude Code / Cursor 的对话式编程执行体验。
2. **通用 Agent 平台**：通过 Models/Rules/Skills/MCP 资源体系扩展到非编码任务。

### 2.2 价值主张

1. **业务隔离可控**：工作区级菜单、权限、数据、业务完全隔离。
2. **资源复用高效**：工作区资源池可被项目继承并固化为项目规范。
3. **执行闭环完整**：对话并行、会话内消息队列、可停止、可审计、可提交。
4. **本地与远程统一**：本地工作区开箱即用；远程工作区支持企业级权限管理。

### 2.3 v0.4.0 重构目标

1. 建立统一且可扩展的 Workspace -> Project -> Conversation -> Execution 产品模型。
2. 完成本地与远程双工作区闭环，并明确资源跨工作区导入与共享机制。
3. 交付生产可用的 Agent 执行链路：队列、中断、Diff、Commit、审计全流程。
4. 建立可发布、可验收、可追责的 P0 标准（含安全、权限、观测、测试）。

---

## 3. 不可变业务决策（Authority Decisions）

以下条目是 v0.4.0 的固定业务决策，不作为“待定项”：

1. 本地工作区只有一个，默认存在，免登录，无 RBAC 限制，拥有全部能力。
2. 远程工作区必须配置 Hub 地址并登录，采用 **RBAC + 核心 ABAC**。
3. 远程工作区 P0 必须包含管理员能力：用户创建、角色分配、菜单权限、数据权限、功能权限分配（API + 基础 UI）。
4. 一个工作区可包含多个项目；每个项目可从工作区选择 models/rules/skills/mcps 作为项目规范。
5. 一个项目可创建多个对话；对话间支持并行执行；单对话内执行严格串行（FIFO）。
6. 对话执行中继续发送消息不打断当前执行，新消息进入队列；Stop 仅终止当前执行并释放对话锁。
7. 每个对话默认启用 worktree，支持 Plan/Agent 与模型切换；新对话默认 Agent 模式。
8. 远程工作区允许用户先私用“本地来源资源”，但实际执行为“导入到远程的私有副本”，不是远程调用用户本地机器。
9. 资源共享必须管理员审批；审批通过后资源同步为远程工作区共享资源，其他成员可用。
10. 模型共享允许包含密钥，且属于高风险行为：必须审批、审计、可撤销、密钥全程掩码展示。
11. 前端视觉实现推荐使用 Pencil MCP 作为设计参考，不作为 P0 发布阻塞项。

---

## 4. 核心对象模型与关系

### 4.1 主体对象

```text
Workspace
  ├── Resource Pools: Models / Rules / Skills / MCPs
  ├── Projects
  │    ├── Project Resource Bindings
  │    └── Conversations (Session)
  │         └── Executions
  ├── Share Requests
  └── Permission Policies (RBAC + ABAC)
```

### 4.2 对象关系定义

1. **Workspace**：最高隔离边界，承载菜单、权限、数据、业务策略。
2. **Project**：工作单元，继承并固化工作区资源规范。
3. **Conversation(Session)**：用户交互线程，承载消息队列与执行状态。
4. **Execution**：一次消息触发的 Agent 执行过程（内部概念，对用户以事件流呈现）。
5. **Resource**：可被引用、导入、共享和绑定的能力项（model/rule/skill/mcp）。
6. **ShareRequest**：资源共享审批单据。

### 4.3 命名约定

- 产品术语主名统一使用 `Conversation`。
- 为兼容历史语义，文档中首次出现时标注 `Conversation(Session)`，后续以 Conversation 为主。

---

## 5. 工作区体系与隔离模型

### 5.1 工作区类型

#### Local Workspace（本地工作区）

1. 用户安装应用后自动创建，且唯一。
2. 默认免登录、无 RBAC 限制。
3. 拥有全部能力：项目管理、资源管理、Agent 执行、Git 操作。

#### Remote Workspace（远程工作区）

1. 用户手动添加 Hub 地址并登录。
2. 数据、菜单、权限全部来自远程 Hub。
3. 权限由 RBAC + ABAC 控制，操作受审计。

### 5.2 隔离维度

1. **菜单隔离**：不同工作区返回不同可见菜单。
2. **权限隔离**：工作区内角色与策略独立生效，跨工作区不继承。
3. **数据隔离**：项目、对话、资源、审计日志按工作区隔离。
4. **业务隔离**：审批流、共享流、执行流均在工作区内闭环。

### 5.3 跨工作区访问默认规则

1. 默认不互通，不可直接读取其他工作区数据。
2. 本地来源资源进入远程工作区必须走“导入流程”。
3. 共享后形成远程副本，不依赖用户本地环境在线。

---

## 6. 资源体系（Models / Rules / Skills / MCPs）

### 6.1 工作区资源池

每个 Workspace 维护独立资源池：

1. Models：模型配置与参数。
2. Rules：行为/规范规则文档与模板。
3. Skills：可复用能力模板（提示词与工具组合）。
4. MCPs：外部能力连接器定义。

### 6.2 项目规范绑定

1. Project 可从 Workspace 资源池中选择并绑定 models/rules/skills/mcps。
2. 绑定后作为该项目默认规范。
3. 新建 Conversation 自动继承项目绑定。
4. Conversation 级允许覆盖默认值，不影响项目绑定本身。

### 6.3 资源生命周期

1. 创建：在工作区资源池中创建资源。
2. 绑定：项目选择资源作为项目规范。
3. 使用：Conversation 执行时按继承+覆盖结果装载。
4. 共享：私有资源经审批可转共享资源。
5. 撤销：共享资源可撤销并失效（保留审计记录）。

---

## 7. 对话执行体系

### 7.1 执行模型

1. Conversation 是并发单位：同一项目多个 Conversation 可并行执行。
2. Execution 是串行单位：同一 Conversation 同时仅允许一个 Execution 运行。
3. 消息队列采用 FIFO：执行中新增消息进入队列，等待顺序执行。

### 7.2 用户行为与系统响应

1. 用户发送消息 -> 立即创建 Execution（若 busy 则置为 queued）。
2. 当前 Execution 完成后自动拉起下一条 queued 消息。
3. 用户点击 Stop：
   - 终止当前 Execution；
   - 释放 Conversation 锁；
   - 若队列非空，自动开始下一条消息。

### 7.3 Plan / Agent 模式

1. 每个 Conversation 支持 `plan` 与 `agent` 两种模式。
2. 新建 Conversation 默认 `agent`。
3. 用户可在 Conversation 内切换模式。
4. 模式切换仅影响后续 Execution，不回溯已完成 Execution。

### 7.4 模型切换

1. Conversation 级可切换模型。
2. 切换后对后续 Execution 生效。
3. 若当前模型不可用，Execution 返回明确错误并提示切换。

---

## 8. Worktree 与 Git 流程

### 8.1 Worktree 默认策略

1. 每个 Conversation 默认开启 worktree 模式。
2. 每次 Execution 在独立 worktree 路径中运行。
3. worktree 隔离主工作目录，避免未确认变更污染主分支。

### 8.2 Diff / Commit / Discard

1. Execution 结束后展示文件级 Diff（可折叠）。
2. 用户可执行：Commit / Export Patch / Discard。
3. Commit 仅提交 Agent 修改文件，不使用 `git add -A`。

### 8.3 Commit 后回主分支策略（补全闭环）

1. Commit 成功后，系统执行“安全合并回目标分支”流程。
2. 若无冲突：自动 fast-forward 或 cherry-pick（以实现定义为准）并清理 worktree。
3. 若有冲突：
   - 保留 worktree；
   - 标记 `merge_conflict` 状态；
   - 引导用户手动解决后再提交或丢弃。
4. 合并失败必须记录审计并提供可重试入口。

### 8.4 非 Git 项目降级策略

1. 支持添加非 Git 目录。
2. 非 Git 项目仅支持文件读写、Diff、Patch 导出。
3. 非 Git 项目不支持 worktree、Commit、分支合并。
4. UI 必须明确展示“非 Git 降级模式”。

---

## 9. 远程管理与权限体系

### 9.1 管理能力（P0）

Remote Workspace P0 交付管理员 API + 基础管理 UI，覆盖：

1. 用户创建/禁用。
2. 角色分配与回收。
3. 菜单权限分配。
4. 数据权限分配（可见域、归属域）。
5. 功能权限分配（执行、审批、资源管理等）。
6. 共享请求审批与撤销。

### 9.2 默认管理员

1. 每个远程 Hub 可在配置文件中预置默认管理员账户。
2. 默认管理员首次登录拥有全量管理权限。
3. 管理员行为必须完整审计。

### 9.3 RBAC + 核心 ABAC

#### RBAC（角色授权）

- 通过角色快速定义基础能力边界。

#### ABAC（属性控制）

采用核心四维：

1. `subject`：用户属性（user_id、roles）。
2. `resource`：资源属性（owner_id、scope、resource_type、share_status）。
3. `action`：动作（read/write/approve/share/revoke）。
4. `context`：上下文（risk_level、workspace_id、operation_type）。

RBAC 负责“能不能做这类事”，ABAC 负责“能对哪个对象在何条件下做”。

### 9.4 角色建议基线

| 角色 | 能力基线 |
|------|----------|
| viewer | 只读访问：项目/对话/执行结果只读，不可执行写操作 |
| developer | viewer + 发起执行 + 资源私有导入 + 发起共享申请 |
| approver | developer + 审批共享 + 审批高风险操作 |
| admin | 全权限 + 用户/权限/菜单/数据管理 + 审计查看 |

> 说明：审批能力与角色必须一一对应，避免“场景与权限定义冲突”。

---

## 10. 本地资源在远程工作区使用

### 10.1 产品规则

1. 用户在远程工作区可使用“本地来源资源”。
2. 使用方式是 **导入为远程私有副本**，不是远程调用本地资源。
3. 导入后默认仅资源所有者可见可用。

### 10.2 共享流程

1. 用户可对私有资源发起共享申请。
2. 管理员审批通过后，资源转为远程共享资源。
3. 共享后其他成员可按权限使用。
4. 共享可撤销，撤销后新执行不可再引用该共享资源。

### 10.3 同步原则

1. 共享是“同步到远程工作区资源池”。
2. 不依赖用户本地设备在线。
3. 远程执行统一使用远程资源副本。

---

## 11. 模型密钥共享策略（高风险）

### 11.1 允许共享的范围

1. 允许共享模型配置及其密钥。
2. 密钥共享属于高风险能力，默认受最严格审批和审计。

### 11.2 强制安全控制

1. **审批必需**：共享模型密钥必须管理员批准。
2. **加密存储**：密钥在 Hub 端加密（AES-256-GCM）。
3. **显示掩码**：前端仅展示掩码值，不回显明文。
4. **可撤销**：管理员可撤销共享，撤销后立即失效。
5. **全审计**：申请、审批、使用、撤销都需审计。

### 11.3 密钥传递语义（修订）

将“Secrets 不出 Hub”修订为：

1. 密钥明文不落盘、不回传前端。
2. 执行面如需使用密钥，仅允许受控短时下发，且不写日志。
3. 任意失败路径必须保证密钥不泄露。

---

## 12. 功能范围（P0 / P1）

### 12.1 P0（必须交付）

1. 本地工作区：默认创建、免登录、全能力。
2. 远程工作区：Hub 配置、登录、工作区切换。
3. 工作区隔离：菜单/权限/数据/业务隔离。
4. 工作区资源池：models/rules/skills/mcps CRUD（基础能力）。
5. 项目资源绑定：Project 绑定并下发资源规范。
6. Conversation 管理：创建、重命名、归档。
7. 执行体系：对话并行 + 对话内队列串行 + Stop。
8. 模式与模型：Plan/Agent 切换，默认 Agent，模型切换。
9. Worktree 默认开启（Git 项目），Diff、Patch 导出、Commit、Discard。
10. 非 Git 降级模式（无 Commit/Worktree）。
11. Capability Prompt：写文件/执行命令/网络/删除均触发确认。
12. 资源导入：本地来源资源导入远程私有副本。
13. 资源共享：申请 -> 管理员审批 -> 工作区共享。
14. 模型密钥共享能力（审批+审计+可撤销）。
15. 远程管理员能力：用户、角色、菜单权限、数据权限、功能权限（API+基础 UI）。
16. 审计日志：执行、审批、共享、Git、权限变更全覆盖。
17. zh-CN + en-US 双语。

### 12.2 P1（增强能力）

1. LangGraph ReAct 执行引擎（与 Vanilla 并存）。
2. SSE 断线重连优化（断点续传细化策略）。
3. Hub Docker 部署增强与集群化准备。
4. 更细粒度 ABAC 条件表达式（部门、时间窗、网络域）。
5. 命令面板高级能力与工作流模板。

### 12.3 Out of Scope（v0.4.0 不做）

1. 移动端（iOS/Android）。
2. 自动 Push / 自动 PR。
3. 多人实时协同编辑（CRDT/OT）。
4. 官方托管 SaaS 服务。

---

## 13. 关键流程

### 13.1 本地首启流程

```text
安装应用 -> 打开即进入本地工作区 -> 添加项目 -> 创建 Conversation
-> 发送消息 -> Agent 执行 -> Diff -> Commit/Discard
```

### 13.2 远程工作区流程

```text
添加远程 Hub -> 登录 -> 拉取菜单/权限/数据
-> 选择项目 -> 创建 Conversation -> 执行与审批
```

### 13.3 消息队列执行流程

```text
用户发消息A（running）
用户继续发消息B/C（queued）
A完成或被Stop -> 自动执行B -> 自动执行C
```

### 13.4 资源共享流程

```text
私有资源 -> 发起共享申请 -> 管理员审批
-> 转共享资源 -> 其他成员可见可用
-> 可撤销（撤销后失效）
```

---

## 14. 公共接口与类型定义（PRD 级）

### 14.1 新增/修订接口

1. `POST /v1/workspaces/{workspace_id}/resource-imports`
   - 用途：导入本地来源资源到远程私有资源。
2. `POST /v1/workspaces/{workspace_id}/share-requests`
   - 用途：发起资源共享申请（model/rule/skill/mcp）。
3. `POST /v1/share-requests/{request_id}/approve`
   - 用途：审批通过共享申请。
4. `POST /v1/share-requests/{request_id}/reject`
   - 用途：驳回共享申请。
5. `POST /v1/shared-resources/{id}/revoke`
   - 用途：撤销共享资源。
6. 管理员接口组（用户/角色/菜单/数据/功能权限）：
   - `POST /v1/admin/users`
   - `POST /v1/admin/users/{id}/roles`
   - `POST /v1/admin/permissions/menu-bindings`
   - `POST /v1/admin/permissions/data-bindings`
   - `POST /v1/admin/permissions/action-bindings`

### 14.2 关键类型（补充字段）

```text
Workspace {
  workspace_id: string
  name: string
  mode: "local" | "remote"
  tenant_id?: string
  is_default_local: boolean
}

Resource {
  resource_id: string
  workspace_id: string
  resource_type: "model" | "rule" | "skill" | "mcp"
  scope: "private" | "shared"
  source: "workspace_native" | "local_import"
  owner_user_id: string
  share_status: "not_shared" | "pending" | "approved" | "rejected" | "revoked"
}

ShareRequest {
  request_id: string
  workspace_id: string
  resource_id: string
  resource_type: "model" | "rule" | "skill" | "mcp"
  requester_user_id: string
  approver_user_id?: string
  status: "pending" | "approved" | "rejected" | "revoked"
  reason?: string
  audit_ref: string
}

PermissionPolicy {
  role_grants: RoleGrant[]
  abac_conditions: {
    subject: object
    resource: object
    action: object
    context: object
  }
}

Conversation {
  conversation_id: string
  project_id: string
  default_worktree: true
  default_mode: "agent"
  queue_state: "idle" | "running" | "queued"
  active_execution_id?: string
}

ExecutionEvent {
  event_id: string
  execution_id: string
  conversation_id: string
  trace_id: string
  queue_index: number
  risk_level?: "low" | "high" | "critical"
  approval_state?: "not_required" | "pending" | "approved" | "denied"
}
```

### 14.3 权限校验原则

1. 写操作默认拒绝，显式授权才放行。
2. 审批类接口需同时满足角色权限与 ABAC 条件。
3. 资源访问必须校验 `workspace_id` 与归属策略。

---

## 15. 安全与合规

### 15.1 Trust Boundary

```text
Desktop -> Hub -> Worker
```

1. Desktop 不直接访问 Worker。
2. Hub 是权限、审计、密钥、策略的权威面。
3. Worker 仅在授权上下文内执行。

### 15.2 安全策略

1. Path Guard：文件访问限制在项目根目录/worktree。
2. Command Guard：命令白名单 + 高危模式黑名单。
3. Capability Prompt：高风险工具调用必须确认。
4. 审计全覆盖：工具调用、审批决策、共享、权限变更、Git 操作。
5. 诊断脱敏：导出信息自动掩码敏感字段。

### 15.3 风险分级

| 能力 | 风险 | 默认行为 |
|------|------|---------|
| read/search/list | Low | 自动放行 |
| write/apply_patch | High | 阻断确认 |
| run_command | High | 阻断确认 |
| network/mcp_call | High | 阻断确认 |
| delete/revoke_key | Critical | 阻断确认 + 审计增强 |

---

## 16. UI/UX 与设计约束

### 16.1 核心界面

1. Sidebar：Workspace / Project / Conversation 结构。
2. Conversation Area：事件流（思考/工具调用/Diff/确认/结果）。
3. Input Composer：模式切换、模型切换、发送、Stop。
4. Admin Console（Remote）：用户与权限管理、共享审批。

### 16.2 状态规范

1. 空状态：无项目、无对话、无模型配置。
2. 执行状态：running/queued/stopped/done/error。
3. 网络状态：connected/reconnecting/disconnected。
4. 审批状态：pending/approved/denied/revoked。

### 16.3 设计实践约束

1. 前端视觉与交互设计推荐使用 Pencil MCP 方法沉淀统一风格。
2. 参考资料：[Pencil Docs](https://docs.pencil.dev)
3. 该约束为推荐实践，不作为 v0.4.0 发布阻塞项。

---

## 17. 非功能需求（NFR）

| 维度 | 要求 |
|------|------|
| 启动 | Desktop 冷启动至可交互 < 5s；Hub 健康就绪 < 5s |
| 延迟 | Worker 事件到前端渲染 < 200ms（本地网络） |
| 并发 | 本地与远程均支持多 Conversation 并行；并发上限可配置 |
| 队列 | 单 Conversation 严格 FIFO，不允许并发执行 |
| 恢复 | 执行中断后状态一致，支持继续处理后续队列 |
| 观测 | trace_id 贯穿 Hub -> Worker -> Events -> Audit |
| 国际化 | zh-CN + en-US 全量文案同步 |
| 无障碍 | WCAG 2.1 AA（对比度、键盘导航、焦点管理） |

---

## 18. 成功指标（产品 + 技术）

### 18.1 产品指标

1. 新用户首日激活率：完成“创建项目 + 创建 Conversation + 发起首次执行”比例 >= 70%。
2. Worktree 采纳率：Agent 产出 Diff 被 Commit 的比例 >= 50%。
3. 共享转化率：发起共享申请后 7 天内审批通过比例 >= 60%。
4. 队列成功率：Conversation 队列任务按序执行成功率 >= 95%。

### 18.2 技术指标

1. 简单编码任务完成率 >= 70%。
2. 敏感操作拦截率 = 100%。
3. 断线恢复成功率 >= 99%。
4. 自动化测试通过率：Hub/Desktop/Worker 主干测试全绿。

---

## 19. 验收场景与测试用例

### 19.1 必测业务场景

1. 本地首启：免登录完成一次 Agent 改码闭环。
2. 远程登录：Hub 配置与登录后权限菜单正确加载。
3. 工作区隔离：A/B 工作区菜单和数据互不可见。
4. 项目继承：Project 成功继承并生效资源规范。
5. 并发执行：同项目两个 Conversation 并行且互不阻塞。
6. 队列与停止：执行中追加消息，Stop 后自动执行下一条。
7. 私有导入：本地来源资源导入后仅本人可见可用。
8. 共享审批：申请 -> 审批 -> 共享可用全链路可追踪。
9. 模型密钥共享：审批后生效，展示掩码，日志完整。
10. 权限拒绝：无审批权限用户无法批准共享。
11. Capability Prompt：写文件/命令/网络/删除全部触发确认。
12. 异常恢复：Worker 崩溃、SSE 断线后状态一致且可追溯。

### 19.2 测试门槛

1. Hub：`go test ./...` 通过。
2. Desktop：`pnpm test` 通过。
3. Worker：`pytest` 通过。
4. 关键 E2E 场景（上述 12 条）通过。

---

## 20. Release Criteria（Go / No-Go）

### 20.1 P0 Go 条件（必须）

1. P0 功能全部可用，无阻断级缺陷。
2. 多 Conversation 并行 + 单 Conversation 串行队列行为符合预期。
3. 远程管理员 API + 基础 UI 可用。
4. 资源导入、共享审批、共享撤销全流程可用。
5. 模型密钥共享满足审批、审计、掩码与撤销要求。
6. Capability Prompt 对高风险操作拦截率 100%。
7. 测试门槛全部通过。

### 20.2 P1 状态（不阻塞 v0.4.0 发布）

1. P1 功能可延期交付，不影响 v0.4.0 上线。
2. P1 未完成项必须进入后续版本计划并有明确 owner。

---

## 21. 主要风险与缓解

| 风险 | 影响 | 缓解策略 |
|------|------|---------|
| 权限模型复杂导致错配 | 数据越权或审批绕过 | RBAC 基线 + ABAC 四维最小闭环；上线前做越权回归测试 |
| 模型密钥共享泄露风险 | 安全与合规风险 | 强制审批、密钥加密、掩码展示、可撤销、全链路审计 |
| 并发与队列状态竞争 | 执行顺序错误、锁异常 | 单 Conversation 强互斥 + 原子状态机 + Watchdog |
| Agent 长链路不稳定 | 任务失败率上升 | 重试策略、错误码分层、可恢复执行与人工接管 |
| 多端实现风格不一致 | UX 割裂 | 设计规范统一，Pencil MCP 作为推荐设计方法 |

---

## 22. 术语表

| 术语 | 定义 |
|------|------|
| Workspace | 菜单/权限/数据/业务隔离的最高边界 |
| Local Workspace | 默认唯一、免登录、全能力的本地工作区 |
| Remote Workspace | 需配置 Hub 并登录的远程工作区 |
| Project | 工作区下的工作单元，承载代码与资源绑定 |
| Conversation(Session) | 用户交互线程；对外主术语为 Conversation |
| Execution | 一次消息触发的内部执行过程 |
| Resource | model/rule/skill/mcp 的统称 |
| ShareRequest | 资源共享审批单据 |
| RBAC | 基于角色的访问控制 |
| ABAC | 基于属性的访问控制（subject/resource/action/context） |
| Worktree | Git 隔离工作目录 |
| Capability Prompt | 高风险操作的人机确认机制 |

---

## 23. 附录 A：统一错误响应

```json
{
  "code": "SESSION_BUSY",
  "message": "Conversation is currently executing another task",
  "details": {
    "active_execution_id": "exec_..."
  },
  "trace_id": "tr_..."
}
```

错误码建议前缀：`AUTH_`、`WORKSPACE_`、`PROJECT_`、`CONVERSATION_`、`EXEC_`、`RESOURCE_`、`SHARE_`、`PERMISSION_`、`TOOL_`、`INTERNAL_`。

---

## 24. 附录 B：执行事件模型

```text
event types:
- message_received
- execution_started
- thinking_delta
- tool_call
- tool_result
- confirmation_required
- confirmation_resolved
- diff_generated
- execution_stopped
- execution_done
- execution_error
```

所有事件必须带 `trace_id` 与 `conversation_id`。

---

## 25. 附录 C：文档一致性约束（用于后续维护）

1. 若修改 P0/P1 范围，必须同步更新：功能范围、验收场景、Release Criteria。
2. 若修改权限策略，必须同步更新：角色表、ABAC 维度、审批流程与安全章节。
3. 若修改资源共享逻辑，必须同步更新：接口定义、类型定义、测试场景。
4. 若修改执行模型，必须同步更新：对话队列、并发 NFR、事件模型与错误码。

