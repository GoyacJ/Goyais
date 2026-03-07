# Goyais 产品需求文档（PRD）

> 本文面向内部产品与研发团队，描述 Goyais `v0.4.0` 在当前仓库中已实现或可验证的产品能力。对象名称、角色、状态枚举与资源类型以 `packages/contracts/openapi.yaml` 为准。

## 1. 产品定位与版本范围

**Goyais** 是一套面向开发者的 AI 协作开发产品，围绕 `Workspace -> Project -> Session -> Run -> ChangeSet` 组织从需求输入、执行编排到变更交付的完整链路。

- **产品定位**：以会话驱动开发流程的 AI 开发工作台
- **当前版本**：`v0.4.0`
- **协议**：MIT
- **当前文档范围**：Desktop、Mobile、Hub 三类已存在产品面与控制面能力
- **当前文档口径**：仅写入可被路由、调用链、OpenAPI 契约或测试共同验证的能力

当前版本重点不是“多端能力齐平”，而是以 Desktop 为主入口，提供本地工作区与远程工作区两类使用方式；Mobile 聚焦远程工作区访问；Hub 提供统一控制面、认证、资源配置、执行与治理能力。

---

## 2. 产品形态与运行时边界

### 2.1 当前运行时

| 运行时 | 形态 | 当前职责 |
|---|---|---|
| **Desktop** | Tauri + Vue 3 桌面端 | 本地/远程工作区入口，会话执行、资源配置、变更审阅、本地设置 |
| **Mobile** | Tauri Mobile + Vue 复用 | 远程工作区访问、轻量化会话与状态查看 |
| **Hub** | Go 控制面服务 | 工作区、身份认证、项目、Session、Run、ChangeSet、资源配置、管理与审计接口 |

### 2.2 工作区模式  

- **Local Workspace**：仅 Desktop 支持。默认提供本地工作区，本地运行时可通过 sidecar 方式启动 Hub 控制面。
- **Remote Workspace**：Desktop 与 Mobile 均可接入。通过 Hub 登录、获取权限快照并按角色控制可见菜单与可执行动作。

### 2.3 当前客户端能力矩阵

| 能力项 | Desktop | Mobile |
|---|---|---|
| 本地工作区 | ✅ | ❌ |
| 远程工作区 | ✅ | ✅ |
| Hub sidecar | ✅ | ❌ |
| 目录导入 | ✅ | ❌ |
| 窗口控制 | ✅ | ❌ |
| 自启动 | ✅ | ❌ |
| 本地设置（主题/语言/通用） | ✅ | ❌ |
| 远程连接 HTTPS 约束 | 可配置 | 发布版强制 |

说明：代码中存在预留的 Web 运行时目标，但它不属于 `v0.4.0` 当前正式产品面，不进入本 PRD 主体能力承诺。

---

## 3. 核心对象模型

### 3.1 主对象

| 对象 | 作用 | 当前关键字段/状态 |
|---|---|---|
| **Workspace** | 用户进入产品的顶层容器 | `mode=local/remote`、`hub_url`、`auth_mode`、`login_disabled` |
| **Project** | 工作区下的代码项目 | `repo_path`、`is_git`、默认模型、Token 阈值、累计 Token 用量 |
| **Session** | 项目内一次 AI 协作会话 | `queue_state=idle/running/queued`、`default_mode`、`model_config_id`、`rule_ids/skill_ids/mcp_ids` |
| **Run** | Session 内一次执行实例 | `state=queued/pending/executing/confirming/awaiting_input/completed/failed/cancelled`、`queue_index`、`trace_id` |
| **ChangeSet** | Session 当前待交付变更集合 | `entries`、`capability`、`suggested_message`、`project_kind=git/non_git` |

### 3.2 支撑对象

| 对象 | 作用 |
|---|---|
| **ResourceConfig** | 工作区级资源配置，类型包括 `model`、`rule`、`skill`、`mcp` |
| **ProjectConfig** | 项目级资源绑定，决定项目可用模型、规则、技能、MCP 与阈值 |
| **WorkspaceAgentConfig** | 工作区级 Agent 缺省行为，如执行轮次、Trace 细节、默认模式、预算、MCP 搜索与子代理默认值 |
| **PermissionSnapshot** | 远程工作区权限快照，包含角色、权限集合、菜单可见性与动作可见性 |

---

## 4. 核心能力模块

### 4.1 工作区与身份

- Desktop 在支持本地能力的运行时下自动提供默认本地工作区。
- 支持新增远程工作区元数据、远程登录、刷新/注销、工作区切换与最近使用顺序维护。
- Hub 提供 `me`、`permissions`、工作区状态等接口，用于客户端顶部状态栏、连接状态与身份展示。
- 远程工作区基于权限快照控制菜单是否 `hidden / disabled / readonly / enabled`。

### 4.2 项目与 Session

- 支持按目录导入项目、创建项目、删除项目。
- 支持列出项目、项目文件目录与文件内容读取，为 Session 编排与引用提供上下文。
- 支持在项目下创建 Session，并对 Session 执行重命名、删除、详情读取与 Markdown 导出。
- Session 维持项目级默认模型与资源绑定，同时记录自己的默认模式、排队状态与累计 Token 用量。

### 4.3 Session 执行

- Composer 支持普通文本输入、`/` 命令和 `@resource` 引用。
- `@resource` 当前口径覆盖 `model`、`rule`、`skill`、`mcp`、`file`。
- Hub 提供 Composer catalog 与 suggestion 能力；`/` 命令可直接返回 `command_result`，也可提交为排队执行的 Run。
- 单个 Session 支持多条消息排队；后续请求进入队列后通过 `queue_index` 与 `queue_state` 跟踪。
- 执行过程支持停止当前 Run，并继续处理队列中的后续项。
- Run 控制当前至少覆盖 `stop`、`approve`、`deny`、`resume`、`answer` 五类动作，对应审批、恢复和用户补充输入场景。
- Session 事件通过 SSE 持续推送，事件类型覆盖执行开始、思考增量、工具调用、审批请求、变更更新、任务状态变化与执行完成。

### 4.4 执行可观测性

- Desktop 主界面包含 Run Trace 展示，用于按 Run 汇总思考、工具调用、结果与 Token 消耗。
- Trace 细节级别支持 `basic` 与 `verbose`，由 `WorkspaceAgentConfig.display.trace_detail_level` 控制。
- Inspector 支持切换 `diff / run / trace / risk` 四类视图。
- Run Task 能力包含任务图、任务列表、任务详情、筛选、刷新、分页加载与基础控制。
- Workspace 与 Session 层都暴露运行状态、连接状态、执行数量、排队数量与 Token 用量，作为客户端状态栏与汇总信息来源。

### 4.5 ChangeSet 与变更交付

- Session 级 ChangeSet 支持读取待变更列表、文件数、增删行统计与建议提交说明。
- 当前交付动作包括：
  - `commit`
  - `discard`
  - 导出变更文件归档
  - 回滚到指定用户消息锚点
- ChangeSet 自带能力标志，明确当前是否允许提交、丢弃与导出。
- 回滚以消息锚点为中心，恢复消息、Run 与快照到指定位置后的可继续状态。

### 4.6 工作区配置

- **模型配置**：支持工作区模型目录、模型资源配置创建/更新/删除、连通性测试、启停与 Token 统计。
- **规则配置**：支持 Markdown 内容编辑、启停与项目绑定。
- **技能配置**：支持 Markdown 内容编辑、启停与项目绑定。
- **MCP 配置**：支持创建/更新/删除、连通性校验与脱敏导出。
- **项目配置**：支持为每个项目绑定模型、规则、技能、MCP，并配置默认模型与 Token 阈值。
- **Agent 配置**：当前配置面覆盖：
  - `max_model_turns`
  - `show_process_trace`
  - `trace_detail_level`
  - `default_mode`
  - `prompt_budget_chars`
  - `search_threshold_percent`
  - `mcp_search`
  - `output_style`
  - `subagent_defaults`
  - `feature_flags`

### 4.7 远程管理与治理入口

- Desktop 当前提供三类远程管理页面：
  - 远程账号信息
  - 成员与角色
  - 权限审计
- 前台可见重点是账号状态、成员角色分配与审计结果查看。
- Hub 同时提供更底层的治理接口，包括用户、角色、权限定义、菜单定义、菜单可见性、ABAC 策略与审计记录；这些能力属于治理底座，不作为当前桌面端主流程功能展开。

---

## 5. 关键用户流程

### 5.1 Desktop 本地开发流程

1. 启动 Desktop，进入默认本地工作区。
2. 导入项目目录，生成 Project。
3. 创建 Session，选择模型、模式与需要绑定的资源。
4. 通过 Composer 提交需求，创建 Run 或触发命令结果。
5. 在主界面查看队列状态、Run Trace、任务图与 ChangeSet。
6. 审阅变更后执行提交、丢弃、导出或回滚。

### 5.2 远程工作区接入流程

1. 新增远程工作区并填写 Hub 地址。
2. 执行登录，获取访问令牌、身份信息与权限快照。
3. 切换到远程工作区。
4. 客户端按权限快照决定可见菜单、可执行动作与是否可进入管理页。

### 5.3 Session 审批与排队流程

1. 用户在同一 Session 中连续发送多条输入。
2. 第一条输入进入活动 Run，后续输入进入队列。
3. 若 Run 进入 `confirming` 或 `awaiting_input`，用户或审批者通过控制动作处理。
4. Run 完成后刷新 ChangeSet 与 Inspector 数据；用户可继续执行、停止队列或回滚到某个消息锚点。

### 5.4 远程治理流程

1. 管理员进入远程账号与治理页面。
2. 查看成员、角色、连接状态与权限审计结果。
3. 基于 Hub 权限模型限制非管理员用户的菜单与动作可见性。

---

## 6. 权限与治理

### 6.1 角色

| 角色 | 当前职责边界 |
|---|---|
| **viewer** | 只读访问，不发起执行与资源写入 |
| **developer** | 可执行 Session、修改资源配置，不负责治理配置 |
| **approver** | 在 developer 基础上处理审批类动作 |
| **admin** | 完整治理权限，包括远程管理与审计入口 |

### 6.2 能力标志

| 标志 | 含义 |
|---|---|
| `admin_console` | 允许进入远程管理相关页面 |
| `resource_write` | 允许修改工作区资源配置 |
| `execution_control` | 允许发起或控制 Session 执行 |

### 6.3 PermissionMode

| 模式 | 当前语义 |
|---|---|
| `default` | 低风险工具自动允许，非低风险需审批，关键风险拒绝 |
| `acceptEdits` | 低/中风险编辑允许，高风险需审批，关键风险拒绝 |
| `plan` | 只允许低风险读操作，其他风险动作拒绝 |
| `dontAsk` | 允许低风险预批准动作，其他需审批动作直接拒绝 |
| `bypassPermissions` | 跳过权限提示，允许所有动作，属于危险模式 |

### 6.4 治理规则

- 本地工作区默认视为完整权限环境。
- 远程工作区权限由 Hub 下发，客户端只消费快照，不自行推导角色。
- 菜单与动作可见性均以 `PermissionSnapshot` 为准。
- 审计相关能力当前重点覆盖远程工作区。

---

## 7. 非功能要求与质量门禁

### 7.1 工程门禁

当前仓库已定义并可直接执行的质量命令包括：

- `pnpm lint`
- `pnpm test`
- `pnpm test:strict`
- `pnpm coverage:gate`
- `pnpm e2e:smoke`
- `pnpm docs:build`
- `pnpm slides:build`
- `cd services/hub && go test ./...`
- `pnpm contracts:generate`
- `pnpm contracts:check`

### 7.2 安全与运行约束

- 远程工作区以身份认证、权限快照和审计接口构成基础治理链路。
- Mobile 发布场景要求 Hub 使用 HTTPS。
- Desktop 本地能力依赖 sidecar 与本地控制面。
- 高风险执行由权限模式与审批流共同约束。

### 7.3 国际化与契约一致性

- 当前 UI 语言支持 `zh-CN` 与 `en-US`。
- OpenAPI 是 Hub 与客户端共享类型的单一权威源。
- `packages/shared-core` 由契约生成类型，减少产品文档、接口与前端模型漂移。

---

## 8. 已知限制与近期规划

### 8.1 当前版本限制

- Mobile 仅支持远程工作区，不支持本地工作区、sidecar、目录导入与自启动。
- 本地设置页仅存在于 Desktop。
- 远程治理入口当前以 Desktop 为主，Mobile 不承担管理端职责。
- 代码中存在预留的 Web 运行时目标，但 `v0.4.0` 不将其作为正式产品面承诺。
- 部分 Hub 治理接口的覆盖范围高于当前桌面端信息架构，前台入口与后端契约尚未完全对齐。

### 8.2 近期规划（保守口径）

- 继续收敛 Desktop 路由、Hub 契约与测试覆盖之间的能力一致性。
- 补齐远程治理与资源配置能力从 Hub 接口到前台入口的闭环。
- 扩展 Mobile 在远程工作区场景下的验证覆盖与体验完整度。
- 保持预留 Web 运行时的接口边界准备，但不在当前版本承诺发布日期。
