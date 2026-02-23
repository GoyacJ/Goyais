# Goyais v0.4.0 产品需求文档（PRD）

## 1. 文档信息

- 文档版本：v0.4.0
- 日期：2026-02-23
- 文档状态：重写版（Design + Requirement Aligned）
- 性质：Clean-Slate Rewrite（以 v0.4.0 设计与需求为权威）
- 协议：Apache-2.0
- 产品形态：桌面端 AI 智能平台（AI Coding + 通用 Agent）
- 目标平台：macOS（P0），Windows/Linux（P1）
- 目标读者：产品、架构、研发、测试、运维、项目管理

---

## 2. 产品定位与价值主张

### 2.1 产品定位

Goyais 是一个工作区隔离的 AI 智能平台，覆盖两类能力：

1. AI Coding：提供类编码助手的对话执行能力，支持队列、停止、回滚、Diff、审计。
2. 通用 Agent：通过 Models/Rules/Skills/MCP 资源体系扩展到非编码场景。

### 2.2 价值主张

1. 多工作区统一治理：本地开箱即用，远程支持企业级权限与审计。
2. 资源闭环：工作区资源池 -> 项目配置 -> Conversation 执行，支持审批共享。
3. 执行可控：多 Conversation 并行、单 Conversation 严格 FIFO、可停止、可回滚。
4. 管理可运营：成员角色、权限审计、模型与 MCP 管理、设置与诊断统一落地。

### 2.3 v0.4.0 重构目标

1. 建立并固化 `Workspace -> Project -> Conversation -> Execution` 对象模型。
2. 建立并固化 `Desktop -> Hub -> Worker` 权威信任边界。
3. 完成主屏幕、账号信息、设置三大信息架构的一致行为定义。
4. 交付发布可验收的 P0 基线（权限、安全、回滚、共享、审计、测试门槛）。

---

## 3. 不可变业务决策（Authority Decisions）

以下条目是 v0.4.0 固定业务决策，不作为待定项：

1. 本地工作区唯一且默认存在，免登录，具备完整能力。
2. 远程工作区必须通过 `hub_url + username + password` 创建连接并登录。
3. 主屏幕左侧支持折叠；顶部为工作区切换按钮，下拉中包含本地工作区、远程工作区及“新增工作区”。
4. 新增工作区通过弹窗完成，必填字段为 `hub_url`、`username`、`password`。
5. 项目由文件管理器导入，v0.4.0 仅支持“目录导入”。
6. 项目列表为树形结构并支持折叠；项目级动作包含“新增 Conversation”“移除项目”。
7. 一个项目支持多个 Conversation；Conversation 级动作至少包含“导出 Markdown”“删除”。
8. 主屏幕右上展示“运行状态 + Hub 连接状态”；标题区显示“项目名称 / Conversation 名称”，Conversation 名称可动态修改。
9. 对话区采用“AI 在左、用户在右”的流式结构；执行中再次发送消息不打断当前执行，而进入队列。
10. 单 Conversation 同一时刻仅允许一个活动 Execution；队列严格 FIFO。
11. Stop 仅终止当前 Execution，并释放 Conversation 锁；若队列非空，自动拉起下一条。
12. 用户消息支持“回滚到此处”；回滚语义为快照回滚，恢复该消息时点的消息游标、队列状态、工作树状态、Inspector 状态，并保留审计。
13. 输入区动作顺序固定：`+ 功能菜单 -> Agent/Plan -> 模型切换 -> 发送`。
14. 右侧 Inspector 为主屏幕固定能力区，至少包含变更记录、执行状态、文件、风险。
15. 账号信息与设置均采用左右布局；账号信息左侧菜单按权限动态渲染，设置左侧菜单固定；本地工作区不显示账号信息卡。
16. 远程 Hub 支持默认管理员账户，管理员具备全权限并强制审计。
17. 本地资源（模型/规则/技能/MCP）可共享到远程工作区，但必须经远程工作区管理员审核。
18. 模型配置采用“厂商 -> 模型”两级结构；P0 必须支持厂商：OpenAI、Google、Qwen、豆包、智谱、MiniMax、本地。
19. 模型目录优先读取手工 JSON 文件（`<catalog_root>/.goyais/model.json`）；文件缺失时回退 `models.default.json` 模板，并支持手动刷新与定时重载及失败审计。
20. v0.4.0 主术语统一使用 `Conversation`，`Session` 不作为主名。

---

## 4. 核心对象模型与关系

### 4.1 主体对象

```text
Workspace
  ├── WorkspaceConnection (remote only)
  ├── Resource Pools: Models / Rules / Skills / MCPs
  ├── Projects
  │    ├── ProjectConfig (models/rules/skills/mcps)
  │    └── Conversations
  │         ├── ConversationSnapshots
  │         └── Executions
  ├── ShareRequests
  └── PermissionPolicies (RBAC + ABAC)
```

### 4.2 对象关系定义

1. Workspace：最高隔离边界，承载菜单、权限、数据、配置与业务策略。
2. WorkspaceConnection：远程连接实体，描述 Hub 地址与认证上下文。
3. Project：工作单元，绑定项目路径与项目级配置。
4. ProjectConfig：项目默认资源规范（模型/规则/技能/MCP）。
5. Conversation：用户交互线程，持有消息、队列与执行状态。
6. ConversationSnapshot：回滚锚点快照，恢复到指定消息时点的执行上下文。
7. Execution：单次消息触发的内部执行过程。
8. Resource：可被创建、绑定、共享、审批、撤销的能力实体。
9. ShareRequest：资源共享审批单据。
10. PermissionPolicy：RBAC 与 ABAC 组合策略。

### 4.3 命名约定

- 产品与接口主名统一使用 `Conversation`。
- 历史兼容映射可在实现层处理，不在产品文档中使用 `Session` 作为主语义。

---

## 5. 工作区体系与隔离模型

### 5.1 工作区类型

#### Local Workspace（本地工作区）

1. 默认自动创建且唯一。
2. 免登录使用。
3. 可访问项目、执行、资源、设置等完整能力。
4. 不展示远程账号信息卡。

#### Remote Workspace（远程工作区）

1. 通过 `hub_url + username + password` 创建连接并登录。
2. 菜单、权限、数据、审计由 Hub 下发并控制。
3. 权限执行采用 RBAC + ABAC。

### 5.2 左侧工作区交互约束

1. 左侧支持折叠。
2. 顶部工作区触发器打开下拉列表，支持切换工作区。
3. 下拉中固定存在“新增工作区”入口。
4. 新增成功后可在同一触发器中切换。

### 5.3 隔离维度

1. 菜单隔离：不同工作区菜单可见性不同。
2. 权限隔离：角色与策略按工作区独立。
3. 数据隔离：项目、Conversation、执行、资源、审计按工作区隔离。
4. 业务隔离：审批流、共享流、执行流在工作区内闭环。

### 5.4 访问默认规则

1. 默认不跨工作区直接读写。
2. 跨工作区复用必须走导入/共享流程。
3. Desktop 不得绕过 Hub 对远程执行与权限做权威控制动作。

---

## 6. 资源体系（Models / Rules / Skills / MCPs）

### 6.1 工作区资源池

每个 Workspace 维护独立资源池：

1. Models：厂商、模型、参数、密钥与启停状态。
2. Rules：执行规则模板与策略文本。
3. Skills：可复用技能定义。
4. MCPs：连接器配置、连接状态、启停与测试结果。

### 6.2 项目配置（ProjectConfig）

1. 项目支持绑定模型、规则、技能、MCP 四类默认配置。
2. 新建 Conversation 自动继承 ProjectConfig。
3. Conversation 可覆盖项目默认值，仅影响当前 Conversation。
4. Conversation 覆盖不反写 ProjectConfig。

### 6.3 模型配置结构与目录（手工 JSON）

1. 模型配置采用两级结构：`Vendor(base_url) -> Models`。
2. P0 厂商清单：OpenAI、Google、Qwen、豆包、智谱、MiniMax、本地。
3. `model.json`（或回退模板）中每个 Vendor 必须携带 `base_url`，并由 Hub 校验 URL 合法性。
4. 模型资源配置以 `vendor + model_id` 为主标识，不再要求单独 `name` 字段。
5. 非本地厂商 `base_url` 固定来自目录文件；本地厂商允许在配置时覆盖。
6. 支持模型启用、停用、编辑、删除与默认模型指定。
7. 模型目录优先来源为手工维护的 `.goyais/model.json`（缺失时回退 `models.default.json`）：
   - 本地工作区：目录根由通用设置 `defaultProjectDirectory` 同步到 Hub 的 `catalog-root`。
   - 远程工作区：由 Hub 独立维护目录根与文件。
8. 提供目录刷新能力：
   - 手动触发目录重载（P0）。
   - 定时文件重载（P0）。
9. 重载失败需提供可视化错误与审计记录。

### 6.4 资源生命周期

1. 创建：在工作区资源池创建。
2. 绑定：绑定到项目配置。
3. 使用：Conversation 执行按继承 + 覆盖装载。
4. 共享：私有资源发起共享申请后由管理员审批。
5. 撤销：共享资源可撤销，撤销后新执行不可引用。

---

## 7. Conversation 执行体系

### 7.1 并发与队列模型

1. Conversation 是并发单位：同一项目多个 Conversation 可并行。
2. Execution 是串行单位：同一 Conversation 仅一个活动执行。
3. 新消息在执行中进入 `queued`，按 FIFO 依次执行。

### 7.2 用户行为与系统响应

1. 发送消息：
   - 若 Conversation 空闲：立即创建并运行 Execution。
   - 若 Conversation 忙：消息入队并返回排队位次。
2. Stop：
   - 终止当前 Execution；
   - 释放 Conversation 锁；
   - 若队列非空，自动拉起下一条。
3. 追加发送：不打断当前运行，仅入队。

### 7.3 回滚语义（回滚到此处）

1. 回滚目标：指定用户消息 `message_id`。
2. 回滚粒度：快照回滚（ConversationSnapshot）。
3. 回滚恢复范围：
   - 消息游标恢复到目标消息；
   - 目标消息之后的队列与执行状态重算；
   - 工作树状态恢复到快照引用点；
   - 右侧 Inspector 状态恢复到快照时点。
4. 回滚后保留完整审计链路，不删除历史审计事实。

### 7.4 模式、模型、导出

1. Conversation 支持 `agent` 与 `plan` 模式；新建默认 `agent`。
2. 模式切换仅影响后续 Execution。
3. Conversation 级支持模型切换，仅影响后续 Execution。
4. Conversation 导出格式在 v0.4.0 固化为 Markdown。

---

## 8. Worktree 与 Git 流程

### 8.1 Worktree 默认策略

1. Git 项目中的 Conversation 默认开启 worktree。
2. Execution 在隔离 worktree 路径运行。
3. worktree 隔离主工作目录，避免未确认修改污染。

### 8.2 Diff / Commit / Discard

1. Execution 结束展示文件级 Diff 与统计。
2. 用户可执行：Commit / Export Patch / Discard。
3. Commit 仅提交 Agent 修改文件，不使用 `git add -A`。

### 8.3 合并回目标分支

1. Commit 后执行安全合并回目标分支。
2. 无冲突：自动完成合并并清理 worktree。
3. 有冲突：保留 worktree，标记冲突状态，引导手动处理。
4. 任意失败路径必须审计。

### 8.4 非 Git 项目降级

1. 非 Git 项目可导入并执行。
2. 仅支持读写、Diff、导出补丁。
3. 不支持 worktree、Commit、分支合并。

---

## 9. 远程管理与权限体系

### 9.1 账号信息页（Remote）

1. 展示当前登录账号信息。
2. 展示当前工作区信息（workspace_id、workspace_name、mode、hub 地址等）。
3. 展示连接与会话统计状态。

### 9.2 菜单与入口规则

1. 账号信息左侧菜单：按权限动态渲染。
2. 设置左侧菜单：固定菜单结构。
3. 账号信息和设置左上角复用工作区切换设计。
4. 左下角复用用户头像触发器 + 上拉菜单。

### 9.3 成员与角色

成员能力：新增、编辑、删除、停用、分配角色。  
角色能力：新增、编辑、删除、停用、分配权限。

### 9.4 权限与审计

1. 菜单树支持编辑、删除、停用。
2. 权限项支持编辑、删除、停用。
3. 无权限时可见性遵循 `hidden/disabled/readonly/enabled`。
4. ABAC 拒绝返回 403，并写入审计日志。

### 9.5 默认管理员

1. Hub 启动可配置默认管理员账户。
2. 默认管理员拥有全权限。
3. 管理员高风险动作必须强制审计。

---

## 10. 本地资源共享到远程工作区

### 10.1 规则

1. 本地资源可申请共享到远程工作区。
2. 共享对象覆盖模型、规则、技能、MCP。
3. 共享前远程不可直接消费本地资源实体。

### 10.2 审批流程

1. 资源所有者发起共享申请。
2. 远程工作区管理员审核。
3. 审核通过后进入远程共享资源池。
4. 审核拒绝或撤销后，资源对他人不可用。

### 10.3 审计要求

1. 申请、审批、拒绝、撤销、实际使用全部可追踪。
2. 审计记录至少包含：actor、action、resource、result、time、trace_id。

---

## 11. 模型密钥共享策略（高风险）

### 11.1 允许范围

1. 允许共享模型配置及其密钥。
2. 密钥共享属于高风险操作。

### 11.2 强制控制

1. 审批必需：需 approver/admin 才可通过。
2. 加密存储：Hub 端密钥加密存储。
3. 展示掩码：前端不回显明文。
4. 可撤销：撤销后立即失效。
5. 全审计：变更、调用、失败路径均审计。

### 11.3 密钥传递语义

1. 密钥明文不落盘、不写日志、不回传前端。
2. Worker 仅在受控短时上下文获取密钥。
3. 异常路径保证密钥不泄露。

---

## 12. 功能范围（P0 / P1）

### 12.1 P0（必须交付）

1. 主屏幕三栏信息架构：左侧导航、右侧工作区、右侧 Inspector。
2. 工作区切换与新增远程工作区（`hub_url/username/password`）。
3. 项目目录导入、项目树折叠与项目级动作。
4. Conversation 创建、重命名、删除、Markdown 导出。
5. 对话执行：并发 Conversation + 单 Conversation FIFO 队列 + Stop。
6. 回滚到此处：快照回滚与审计。
7. 输入区固定动作顺序与 Agent/Plan、模型切换。
8. 账号信息页与设置页双入口架构。
9. 成员与角色管理（新增、编辑、删除、停用、分配）。
10. 权限与审计管理（菜单树/权限项编辑、删除、停用）。
11. 工作区共享配置页：Agent/模型/Rules/Skills/MCP。
12. 设置固定菜单：主题、国际化、更新与诊断、通用设置（主题模式 `system/dark/light`、字体样式、字体大小、预设主题、语言切换均需即时生效并持久化；通用设置需提供策略型行式配置：启动与窗口、默认目录、通知、隐私与遥测、更新策略、诊断与日志）。
13. 项目配置入口（账号信息 + 设置），支持模型/规则/技能/MCP 绑定。
14. 模型配置两级结构（厂商 -> 模型）与厂商清单支持。
15. 模型目录手工 JSON 维护与重载（手动 + 定时）。
16. 本地资源共享到远程并需管理员审核。
17. 模型密钥共享高风险治理。
18. 底部状态栏统一展示 Hub 地址与连接状态。
19. 审计覆盖执行、审批、权限、共享、连接、启停、回滚。
20. zh-CN 与 en-US 国际化能力（切换后菜单与关键页面文案即时更新）。

### 12.2 P1（增强项）

1. 更细粒度 ABAC 表达式（部门、时间窗、网络域）。
2. 更丰富的模型目录自动发现与推荐策略。
3. 更高级回滚策略（局部文件级回滚选择器）。
4. 多 Hub 聚合视图与跨 Hub 管理面板。

### 12.3 Out of Scope（v0.4.0 不做）

1. 移动端（iOS/Android）。
2. 自动 Push / 自动 PR。
3. 多人实时协同编辑（CRDT/OT）。
4. 官方托管 SaaS 服务。

---

## 13. 关键流程

### 13.1 本地首启流程

```text
安装应用 -> 进入本地工作区 -> 导入项目目录 -> 创建 Conversation
-> 发送消息 -> 执行 -> Diff/Inspector -> Commit/Discard
```

### 13.2 远程工作区创建与切换流程

```text
点击工作区切换 -> 新增工作区 -> 填写 hub_url/username/password
-> 连接并登录 -> 拉取菜单/权限/数据 -> 切换生效
```

### 13.3 队列与停止流程

```text
发送消息A(running) -> 发送消息B/C(queued)
-> Stop(A) -> 自动执行B -> 自动执行C
```

### 13.4 回滚流程

```text
用户在某条消息点击“回滚到此处” -> 创建回滚任务
-> 恢复 ConversationSnapshot -> 重算后续队列
-> 更新工作树与Inspector状态 -> 记录审计
```

### 13.5 资源共享流程

```text
本地资源 -> 发起共享申请 -> 远程管理员审批
-> 进入远程共享池 -> 其他成员可用 -> 可撤销
```

---

## 14. 公共接口与类型定义（PRD 级）

### 14.1 新增/修订接口

1. `POST /v1/workspaces/remote-connections`
   - 创建远程工作区连接。
   - 请求体：`hub_url`、`username`、`password`。
2. `POST /v1/projects/import`
   - 导入项目目录（仅目录）。
3. `POST /v1/projects/{project_id}/conversations`
   - 创建 Conversation。
4. `POST /v1/conversations/{conversation_id}/messages`
   - 发送消息；busy 时返回 queued 结果。
5. `POST /v1/conversations/{conversation_id}/stop`
   - 停止当前 Execution。
6. `POST /v1/conversations/{conversation_id}/rollback`
   - 参数：`message_id`；执行快照回滚。
7. `GET /v1/conversations/{conversation_id}/export?format=markdown`
   - 导出 Conversation 为 Markdown。
8. `PUT /v1/projects/{project_id}/config`
   - 更新项目配置（模型/规则/技能/MCP）。
9. `GET /v1/workspaces/{workspace_id}/model-catalog`
   - 查询 `.goyais/model.json`（缺失时回退 `models.default.json`）解析后的厂商/模型目录（`revision/source/updated_at`）。
10. `POST /v1/workspaces/{workspace_id}/model-catalog`
   - 手动触发目录重载（不做厂商自动同步）。
11. `GET|PUT /v1/workspaces/{workspace_id}/catalog-root`
   - 查询/更新目录根路径（远程工作区仅管理员可写）。
12. `GET|POST /v1/workspaces/{workspace_id}/resource-configs`
   - 统一管理 `model|rule|skill|mcp` 配置，支持分页与搜索。
13. `PATCH|DELETE /v1/workspaces/{workspace_id}/resource-configs/{config_id}`
   - 编辑、启停与硬删除资源配置。
14. `POST /v1/workspaces/{workspace_id}/resource-configs/{config_id}/test`
   - 模型最小推理测试。
15. `POST /v1/workspaces/{workspace_id}/resource-configs/{config_id}/connect`
   - MCP 握手与工具列表拉取。
16. `GET /v1/workspaces/{workspace_id}/mcps/export`
   - 导出脱敏后的 MCP 聚合 JSON。
17. `GET /v1/workspaces/{workspace_id}/project-configs`
   - 批量返回工作区项目配置。
18. `GET|PUT /v1/projects/{project_id}/config`
   - 查询/更新项目配置（多模型绑定 + 默认模型）。
19. 共享与审批接口（沿用并强化）：
   - `POST /v1/workspaces/{workspace_id}/share-requests`
   - `POST /v1/share-requests/{request_id}/approve`
   - `POST /v1/share-requests/{request_id}/reject`
   - `POST /v1/share-requests/{request_id}/revoke`
20. 管理员接口组（成员/角色/审计）：
   - `GET|POST /v1/admin/users`
   - `PATCH|DELETE /v1/admin/users/{user_id}`
   - `GET|POST /v1/admin/roles`
   - `PATCH|DELETE /v1/admin/roles/{role_key}`
   - `GET /v1/admin/audit`

### 14.2 关键类型（新增字段）

```text
Workspace {
  workspace_id: string
  name: string
  mode: "local" | "remote"
  is_default_local: boolean
}

WorkspaceConnection {
  workspace_id: string
  hub_url: string
  username: string
  connection_status: "connected" | "reconnecting" | "disconnected"
  connected_at: string
}

RemoteConnectionResponse {
  workspace: Workspace
  connection: WorkspaceConnection
  access_token?: string
}

Project {
  project_id: string
  workspace_id: string
  name: string
  root_path: string
  is_git_repo: boolean
}

ProjectConfig {
  project_id: string
  model_ids: string[]
  default_model_id: string | null
  rule_ids: string[]
  skill_ids: string[]
  mcp_ids: string[]
  updated_at: string
}

Conversation {
  conversation_id: string
  project_id: string
  name: string
  mode: "agent" | "plan"
  queue_state: "idle" | "running" | "queued"
  active_execution_id?: string
}

ConversationSnapshot {
  snapshot_id: string
  conversation_id: string
  rollback_point_message_id: string
  queue_state: "idle" | "running" | "queued"
  worktree_ref?: string
  inspector_state: {
    active_tab: "diff" | "run" | "files" | "risk"
    selected_item_id?: string
  }
  created_at: string
}

ModelVendor {
  vendor_id: string
  workspace_id: string
  name: "OpenAI" | "Google" | "Qwen" | "Doubao" | "Zhipu" | "MiniMax" | "Local"
  base_url: string
  status: "enabled" | "disabled"
}

ModelCatalogItem {
  model_id: string
  vendor_id: string
  display_name: string
  status: "enabled" | "disabled"
  last_synced_at?: string
}

PermissionVisibility {
  menu_key: string
  visibility: "hidden" | "disabled" | "readonly" | "enabled"
}
```

### 14.3 权限校验原则

1. 写操作默认拒绝，显式授权放行。
2. 审批接口需同时通过 RBAC 与 ABAC。
3. 菜单与动作可见性由 `PermissionVisibility` 与动作权限共同决定。
4. 所有回滚、审批、共享、密钥相关动作强制审计。

---

## 15. 安全与合规

### 15.1 Trust Boundary

```text
Desktop -> Hub -> Worker
```

1. Desktop 不直接对 Worker 发权威控制指令。
2. Hub 是权限、密钥、审计、策略唯一权威面。
3. Worker 在受控上下文内执行。
4. Hub 调用 Worker internal 接口必须携带 internal token；无 token 或 token 错误必须返回 401 并记录 trace_id。

### 15.2 安全策略

1. Path Guard：文件访问限制在项目根目录/worktree。
2. Command Guard：命令白名单 + 高危黑名单。
3. Capability Prompt：高风险动作必须显式确认。
4. 审计覆盖：执行、回滚、共享、审批、权限、密钥、MCP。
5. 诊断脱敏：导出日志默认掩码敏感字段。

### 15.3 风险分级

| 能力 | 风险 | 默认行为 |
|------|------|----------|
| read/search/list | low | 自动放行 |
| write/apply_patch | high | 阻断确认 |
| run_command | high | 阻断确认 |
| network/mcp_call | high | 阻断确认 |
| delete/revoke_key | critical | 阻断确认 + 审计增强 |

---

## 16. UI/UX 与设计约束

### 16.1 主屏幕信息架构

1. 左侧导航区：
   - 顶部：工作区切换 + 新增工作区。
   - 中部：项目树（可折叠）与项目/Conversation 图标动作。
   - 底部：头像+名称触发器，上拉菜单含账号信息与设置入口。
2. 右侧工作区：
   - 顶部：`项目名称 / Conversation 名称`（Conversation 名称可编辑）+ 运行/连接状态。
   - 中部：Conversation 区（AI 左/用户右）+ 输入区。
   - 右侧：Inspector（变更记录/执行状态/文件/风险）。
   - 底部：Hub 地址与连接状态。

### 16.2 账号信息与设置

1. 两者均为左右结构。
2. 账号信息左侧菜单为动态权限菜单。
3. 设置左侧菜单为固定菜单。
4. 两者左上角与主屏幕一致使用工作区切换设计。
5. 两者左下角与主屏幕一致使用用户触发器设计。
6. 设置页主题模块包含：主题模式、字体样式、字体大小、预设主题，以及恢复默认动作。
7. 设置页通用模块采用紧凑行式布局，策略项即时生效并自动持久化；系统能力未接入平台必须显式展示“暂不可用”而非静默降级。

### 16.3 输入区与状态规范

1. 输入区动作顺序固定：`+ -> Agent/Plan -> 模型 -> 发送`。
2. 运行状态标准：`running/queued/stopped/done/error`。
3. 连接状态标准：`connected/reconnecting/disconnected`。
4. 审批状态标准：`pending/approved/denied/revoked`。

### 16.4 设计实践约束

1. 使用 token 三层：global -> semantic -> component。
2. 组件内禁止硬编码颜色/字号/间距/圆角。
3. 前端结构遵循 feature-first，不采用全局平铺 `src/views/*`。

---

## 17. 非功能需求（NFR）

| 维度 | 要求 |
|------|------|
| 启动 | Desktop 冷启动至可交互 < 5s |
| 事件延迟 | Hub/Worker 事件到前端渲染 < 200ms（本地网络） |
| 并发 | 支持多 Conversation 并行；并发上限可配置 |
| 队列 | 单 Conversation 严格 FIFO，不允许并发执行 |
| 回滚 | 快照回滚成功率 >= 99%，失败可恢复 |
| 导出 | Markdown 导出成功率 >= 99.5% |
| 观测 | trace_id 贯穿 Hub -> Worker -> Events -> Audit |
| 国际化 | zh-CN + en-US 文案一致性 |
| 无障碍 | WCAG 2.1 AA（对比度、键盘导航、焦点管理） |

---

## 18. 成功指标（产品 + 技术）

### 18.1 产品指标

1. 新用户首日激活率：完成“导入项目 + 创建 Conversation + 首次执行”比例 >= 70%。
2. 队列顺序正确率：同 Conversation 队列顺序正确执行比例 >= 99.5%。
3. 回滚成功率：触发“回滚到此处”后快照恢复成功比例 >= 99%。
4. 对话导出成功率：Markdown 导出成功比例 >= 99.5%。
5. 权限菜单正确率：动态菜单与权限结果一致率 >= 99.9%。
6. 模型目录加载成功率：手动重载/定时重载成功率 >= 98%。

### 18.2 技术指标

1. 简单编码任务完成率 >= 70%。
2. 高风险操作拦截率 = 100%。
3. 审计记录完整率（关键动作）= 100%。
4. 自动化测试通过率：Hub/Desktop/Worker 主干测试全绿。

---

## 19. 验收场景与测试用例

### 19.1 必测业务场景

1. 主屏幕流程：工作区切换、项目树折叠、Conversation 新增/删除/导出、发送与队列、Stop、Inspector 切换。
2. 回滚流程：在 queued/running 场景回滚到指定消息，验证工作树与队列恢复到快照点。
3. 设置与账号信息：账号菜单动态、设置菜单固定、本地工作区不显示账号信息卡；主题模块需验证主题模式/字体样式/字体大小/预设主题全局即时生效与持久化。
4. 成员与角色：成员/角色新增、编辑、删除、停用、分配权限全链路。
5. 权限与审计：菜单/权限编辑停用生效；403 拒绝与审计一致。
6. 项目配置：项目绑定生效，Conversation 覆盖不反写项目。
7. 模型配置：厂商-模型两级管理、启停、删除、默认模型切换、目录加载。
8. 资源共享：本地资源共享到远程需管理员审核；通过后可用，撤销后失效。
9. 底部状态栏：Hub 地址与连接状态在主屏幕、账号信息、设置页一致。
10. 连接异常：`reconnecting/disconnected` 下显示只读与重试提示。
11. 非 Git 降级：非 Git 项目进入降级模式并限制 Commit/worktree。
12. 高风险能力：写入/命令/网络/删除触发确认并可审计。

### 19.2 测试门槛

1. Hub：`go test ./...` 通过。
2. Desktop：`pnpm test` 通过。
3. Worker：`pytest` 通过。
4. 关键 E2E 场景（上述 12 条）通过。

---

## 20. Release Criteria（Go / No-Go）

### 20.1 P0 Go 条件（必须）

1. P0 功能清单全部可用，无阻断级缺陷。
2. 多 Conversation 并行 + 单 Conversation FIFO 行为符合预期。
3. 回滚能力可用且审计完整。
4. 账号信息/设置双入口及动态/固定菜单语义正确。
5. 项目配置与 Conversation 覆盖语义正确。
6. 厂商-模型两级配置与手工目录加载可用。
7. 本地资源共享审批闭环可用。
8. 模型密钥共享满足审批、掩码、审计、撤销。
9. 测试门槛全部通过。

### 20.2 P1 状态（不阻塞 v0.4.0 发布）

1. P1 可延期，不阻塞 v0.4.0 上线。
2. 未完成 P1 项必须进入后续版本并有明确 owner。

---

## 21. 主要风险与缓解

| 风险 | 影响 | 缓解策略 |
|------|------|----------|
| 回滚快照与工作树状态不一致 | 造成错误恢复或数据错判 | 快照原子写入 + 回滚事务 + 失败恢复指引 |
| 队列竞争导致顺序错乱 | 执行语义破坏 | 单 Conversation 强互斥 + 原子状态机 + Watchdog |
| 权限动态菜单错配 | 越权或误拦截 | 后端权限权威化 + 前端可见性枚举 + 回归用例 |
| 模型目录 JSON 异常或路径失效 | 模型可用性下降 | 手动修复 JSON + 定时重载补偿 + 错误审计 |
| 密钥共享泄露风险 | 安全合规风险 | 强制审批、掩码、加密存储、撤销与全审计 |

---

## 22. 术语表

| 术语 | 定义 |
|------|------|
| Workspace | 菜单/权限/数据/业务隔离的最高边界 |
| Local Workspace | 默认唯一、免登录、全能力的本地工作区 |
| Remote Workspace | 需连接 Hub 并登录的远程工作区 |
| WorkspaceConnection | 远程 Hub 连接实体 |
| Project | 工作区下的工作单元，绑定目录与项目配置 |
| ProjectConfig | 项目级资源默认配置（模型/规则/技能/MCP） |
| Conversation | 用户交互线程与队列边界 |
| ConversationSnapshot | 回滚锚点快照 |
| Execution | 一次消息触发的执行过程 |
| Resource | model/rule/skill/mcp 的统称 |
| ShareRequest | 资源共享审批单据 |
| PermissionVisibility | 菜单与操作可见性枚举 |
| Capability Prompt | 高风险操作的人机确认机制 |

---

## 23. 附录 A：统一错误响应

```json
{
  "code": "CONVERSATION_BUSY",
  "message": "Conversation is currently executing another task",
  "details": {
    "active_execution_id": "exec_...",
    "queue_state": "running"
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
- execution_queued
- execution_started
- thinking_delta
- tool_call
- tool_result
- confirmation_required
- confirmation_resolved
- diff_generated
- execution_stopped
- snapshot_created
- rollback_requested
- rollback_completed
- execution_done
- execution_error
```

所有事件必须带 `trace_id`、`conversation_id`、`workspace_id`。

---

## 25. 文档一致性约束与同步矩阵

### 25.1 一致性约束

1. 业务规则变化必须同步更新本 PRD。
2. 接口、状态机、对象模型变化必须同步更新 TECH_ARCH。
3. 阶段与门禁变化必须同步更新 IMPLEMENTATION_PLAN。
4. 工程门禁与 DoD 变化必须同步更新 DEVELOPMENT_STANDARDS。
5. 若任一必需同步项缺失，不得宣称文档链路完成。

### 25.2 v0.4.0 跨文档同步矩阵

| change_type | code_or_doc_change | required_docs_to_update | required_sections | status |
|---|---|---|---|---|
| 业务规则变化（主屏幕/账号/设置/项目配置） | PRD 重写 | `/Users/goya/Repo/Git/Goyais/docs/v_0_4_0/PRD.md` | 3, 5, 6, 7, 9, 12, 16, 19, 20 | done |
| 接口与类型变化（rollback/export/project config/model sync） | 接口与类型定义更新 | `/Users/goya/Repo/Git/Goyais/docs/v_0_4_0/TECH_ARCH.md` | 3, 7, 9, 11, 14, 20 | done |
| 阶段与门禁变化（新增回滚/导出/模型目录加载/项目配置验收） | 实施阶段映射更新 | `/Users/goya/Repo/Git/Goyais/docs/v_0_4_0/IMPLEMENTATION_PLAN.md` | Phase 3, 4, 5, 7, 9, 测试计划映射 | done |
| 工程规范补充（动态菜单/快照回滚测试/审计覆盖） | 规范与 DoD 更新 | `/Users/goya/Repo/Git/Goyais/docs/v_0_4_0/DEVELOPMENT_STANDARDS.md` | 10, 11, 13, 14, 15 | done |

### 25.3 2026-02-23 基础框架补齐矩阵

| change_type | required_docs_to_update | required_sections | status |
|---|---|---|---|
| API/类型变更 | PRD.md, TECH_ARCH.md | PRD 14.x, TECH_ARCH 9.x | done |
| 权限语义变更 | PRD.md, TECH_ARCH.md, DEVELOPMENT_STANDARDS.md | PRD 9.x/14.3, TECH_ARCH 5.x, 标准 10.x | done |
| 阶段与门禁变更 | IMPLEMENTATION_PLAN.md, DEVELOPMENT_STANDARDS.md | Phase 映射, DoD/门禁 | done |
| i18n/主题行为变更 | PRD.md, TECH_ARCH.md | PRD 12.1/体验项, TECH_ARCH 14.x | done |
| 主题配置增强（字体样式/字号/预设） | PRD.md, TECH_ARCH.md, IMPLEMENTATION_PLAN.md, DEVELOPMENT_STANDARDS.md | PRD 12.1/16.2/19.1, TECH_ARCH 14.2/14.3, PLAN 增量门禁/测试映射, 标准 9.x/10.x | done |
| 通用设置能力化（策略型行式配置 + 本地持久化 + 平台降级显式化） | PRD.md, TECH_ARCH.md, IMPLEMENTATION_PLAN.md, DEVELOPMENT_STANDARDS.md | PRD 12.1/16.2/19, TECH_ARCH 3.2/4.1/14, PLAN Phase 9/测试映射, 标准 10.4/11.1 | done |

### 25.4 2026-02-24 工作区语义收口矩阵

| change_type | required_docs_to_update | required_sections | status |
|---|---|---|---|
| Workspace 持久化与列表语义（真实下拉） | PRD.md, TECH_ARCH.md | PRD 5.x/9.x/14.x, TECH_ARCH 11.1/9.x | done |
| 工作区切换全上下文行为（Hub/项目/账号/权限） | PRD.md, TECH_ARCH.md | PRD 5.2/9.2/16.x, TECH_ARCH 14.x/20.x | done |
| 测试门禁与验收项（strict 通道 + 工作区场景） | IMPLEMENTATION_PLAN.md, DEVELOPMENT_STANDARDS.md | Phase 2/3 验收、DoD/门禁 | done |

### 25.5 2026-02-23 资源配置体系完善矩阵

| change_type | required_docs_to_update | required_sections | status |
|---|---|---|---|
| 模型目录语义由“同步”改为“纯手工 JSON 目录” | PRD.md | 6.3, 12.1, 18.1, 20.1, 21 | done |
| 目录来源与本地/远程存储路径规则 | TECH_ARCH.md | 3, 6, 9, 11, 20 | done |
| Phase 4 验收口径调整（移除厂商自动同步） | IMPLEMENTATION_PLAN.md | Phase 4 工作内容与验收标准 | done |
| 安全与工程门禁（密钥加密、目录重载、JSON 校验） | DEVELOPMENT_STANDARDS.md | 10, 11, 13, 15 | done |

### 25.6 2026-02-24 模型配置收口矩阵

| change_type | required_docs_to_update | required_sections | status |
|---|---|---|---|
| 模型目录 Vendor 增加 base_url 并模板化 | PRD.md, TECH_ARCH.md | PRD 6.3, TECH_ARCH 6.5/20.4 | done |
| model 资源配置去 name（接口/存储） | TECH_ARCH.md | TECH_ARCH 11.2/20.6 | done |
| 模型配置页收口（仅列表、无手输） | PRD.md | PRD 6.3/19.1 | done |
