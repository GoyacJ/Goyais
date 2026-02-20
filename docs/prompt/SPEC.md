你需要选择一个合适的角色,与我一起共创Goyais这个新的开源级项目.
你需要遵守的点: 
1. 对于我提出的一些技术或者框架,需要搜索对应官网以及文档,确保自己深入理解其核心理念和功能. 
2. 确保自己真的理解业务,如果遇到不明确的可以询问我.
3. 需要对我的想法和需求进行深入思考分析.结合大量实际案例或者设计.给出合理化的建议.
4. 分析我对当前项目定义的功能点和技术点,先理解业务.然后给出你的建议以及方案.
当前项目主要功能: 
1. 一款类似claude code、codex、open code、openclaw 的 AI辅助工具.以及开源协议和通用技巧,例如: MCP、Skills等
2. 基于LangChain、LangGraph、Deep Agents的优秀特性,深度集成与创新. 
3. 不局限于智能代码开发,集成以上所有优秀特性.(你需要查询对应官网以及文档,评估有何优秀的设计和思路)
4. 拥有桌面端(mac、windows、linux)+移动端(ios、android),支持跨平台,基于最新版 tauri 开发.不同平台统一设计、统一UI/UX.
5. 支持本地模式(本地模式下.如需数据库则使用sqlLite,文件管理则使用本地文件系统)和远程服务端模式.(可以连接远程服务器,共享同一数据源,互相协作)
6. 支持AI模型管理.厂商、模型配置.支持切换.支持api-key方式和oauth2登录方式
7. 支持菜单权限+数据权限隔离.同一服务器下.
8. 支持worktree、subagent等优秀设计思想和特性


1) 我对你当前业务定义的理解（先对齐“是什么”）

你要做的不是“又一个聊天 UI”，而是一个 Agentic Dev + 通用 Agent 平台：
	•	形态：类似 Claude Code、Codex CLI、OpenCode 的 AI 编程/工程助手，同时具备 OpenClaw 这类“可扩展技能/插件”的通用能力。 ￼
	•	Agent 栈：深度采用/融合 LangChain + LangGraph + Deep Agents 的优势：
	•	LangGraph：面向“长运行、可状态化”的工作流/Agent 编排（图/状态机思维）。 ￼
	•	Deep Agents：强调“agent harness”——内建 planning、filesystem context、subagent spawning、长期记忆等。 ￼
	•	跨端：桌面 + 移动（mac/win/linux/ios/android），用 Tauri v2 统一工程与 UI/UX。 ￼
	•	双模式：本地模式（SQLite、本地文件系统）+ 远程服务端协作模式（共享数据源、多人协作）。
	•	模型管理：多厂商、多模型配置，支持 API Key 与 OAuth2 登录切换。
	•	多租户隔离：菜单权限 + 数据权限隔离（同一服务器）。
	•	工程化体验：worktree、subagent 等优秀“并行开发/隔离会话”设计思想。Claude Code 的 /resume 会话也明确覆盖 worktrees。 ￼

⸻

2) 关键启发：你列的“优秀特性”背后的共同设计哲学

我从你提到的几个对象里提炼一个共同点：
“把 Agent 的能力拆成可审计的状态 + 受控工具调用 + 可复用技能包 + 可回放事件流”。
	•	LangGraph：用“节点/边/共享状态”把 agent 拆成可控步骤，天然适合可视化、可回放、可插拔。 ￼
	•	Deep Agents：把规划、文件系统上下文、subagent、长期记忆这些“高频刚需”做成 harness，避免每个项目重复造轮子。 ￼
	•	MCP：定义 AI 应用连接外部数据源/工具的标准协议（“像 USB-C”一样的连接口），解决工具生态与可移植性。 ￼
	•	Skills：可复用的“指令+脚本+资产”包，通过 manifest（如 SKILL.md）封装专家流程，强调可迁移的专业方法。 ￼
	•	Codex CLI / Claude Code / OpenCode：都在强化一个方向——本地/仓库内执行、读写文件、跑命令、可迭代协作。 ￼

⸻

3) 我给 Goyais 的总体方案（建议架构）

3.1 分层架构（建议你把“平台能力”拆干净）

A. Client Layer（Tauri v2）
	•	统一 UI/UX：Chat + Task Board + Diff Viewer + Timeline（事件流）+ Tool Permission Center
	•	插件 UI：支持 MCP Apps 这类“工具声明 UI 资源并由 Host 渲染”的模式（可做后续路线）。 ￼
	•	跨端系统能力：文件系统、进程/命令、通知、快捷键、托盘、深度链接等（Rust sidecar/commands）

B. Agent Runtime Layer（核心差异化）
	•	LangGraph 作为“编排内核”：
	•	每个 Agent = 一个 Graph（或多 Graph 组合），节点包括：Plan、Retrieve、ToolCall、CodeEdit、Test、Review、Commit 等
	•	Deep Agents 作为“默认 harness”：
	•	开箱即用：planning + filesystem context + subagent + long-term memory
	•	你可以把“coding agent”作为一个 Deep Agent profile（工具集 + policy） ￼

C. Tooling & Protocol Layer（生态与扩展）
	•	MCP：作为标准工具接入协议（本地/远程 MCP server 都可） ￼
	•	“Skills”：作为可移植的专家流程包（对齐你的“开源协议和通用技巧”诉求） ￼
	•	内置工具：Repo FS、Shell（沙箱/白名单）、Git、Issue Tracker、HTTP、DB、Search、Artifacts（如生成 PR、changelog）

D. Data & Collaboration Layer（本地/远程双模式统一抽象）
	•	统一数据模型：Project / Workspace / Session / Run / Artifact / Permission / AuditLog
	•	本地：SQLite + 本地目录（作为 workspace storage）
	•	远程：PostgreSQL（建议）+ Object Storage（artifact/log）+ Redis（队列/锁）
	•	同步策略：
	•	本地优先：可离线工作，后续同步（CRDT/OT 可选，初期用“会话/工单粒度”同步更稳）
	•	远程协作：共享 session、共享工具配置、共享技能仓库

E. Security & Observability Layer（必须做成卖点）
	•	Prompt Injection/供应链风险已经在“OpenClaw 类事件”中实锤：Agent 能跑命令=必须强隔离。 ￼
	•	建议默认：
	•	工具按 capability 分级授权（读/写/执行/网络/凭证）
	•	Shell 默认“只读 + 显式确认写/执行”
	•	每次工具调用出 audit event（用户可回放/导出）
	•	支持“Lockdown mode”（只允许特定工具/路径/命令白名单）

⸻

4) 你提出的每个功能点：我的建议与落地拆解

4.1 “类似 Claude Code/Codex/OpenCode/OpenClaw”

建议你把产品拆为两条主线（共享底座）：
	1.	Coding Agent（主入口）：repo-aware、可跑测试、生成 PR、支持 worktree 并行会话
	2.	General Agent（扩展入口）：接入 MCP/Skills，面向更广的自动化任务（票务、文档、数据处理等）

这样不会让“通用能力”拖慢“编码体验”的极致打磨。

4.2 LangChain + LangGraph + Deep Agents 深度集成
	•	LangGraph 做编排：保证“可控、可视化、可恢复” ￼
	•	Deep Agents 做默认 harness：你不用自己发明“plan/todo/文件上下文/subagent/记忆”的格式与循环 ￼
	•	LangChain 做生态与集成：模型/向量库/检索/工具适配层（减少供应商锁定） ￼

4.3 Tauri v2 跨平台统一
	•	Tauri v2 已明确覆盖 desktop + mobile（iOS/Android）单代码库方向，适配你的“统一 UI/UX + 多端”目标。 ￼
	•	建议 UI 技术栈：React/Vue/Svelte 任一都可；关键是事件流 UI与diff/patch 体验（coding agent 的核心）

4.4 本地/远程双模式

建议你从 Day1 就做“同一套领域模型 + 两种存储实现”，避免后期大迁移：
	•	Storage 接口（Repository pattern）：SessionStore/RunStore/ArtifactStore/PermissionStore
	•	本地实现：SQLite + FS
	•	远程实现：Postgres + S3 + SSE/WebSocket

4.5 模型管理（API Key + OAuth2）

建议把“模型连接”抽象成：
	•	Provider（OpenAI/Anthropic/Gemini/…）
	•	Credential（apiKey/oauthToken/…）
	•	ModelConfig（modelName、context、tool calling、rate limit、成本上限、策略）
	•	Routing Policy（按任务类型/成本/延迟/安全等级自动选模）

4.6 菜单权限 + 数据权限隔离（多租户）
	•	RBAC（菜单/功能）+ ABAC（数据范围：workspace/project/session）
	•	所有 agent/tool 调用必须带 tenant/workspace 上下文，落 audit log
	•	远程模式建议：每个 workspace 独立 encryption key（便于合规与导出）

4.7 worktree + subagent
	•	worktree：把“并行任务”映射成“并行工作目录 + 分支”，降低上下文污染；Claude Code 的会话列表也会覆盖 worktree 维度，说明这条路是对的。 ￼
	•	subagent：把复杂任务拆成子任务（例如：TestFixer、Refactorer、DocWriter、SecurityReviewer），每个子 agent 有独立工具权限与预算（token/cost/time）

你是一个资深全栈工程师 + Agent 平台架构师。请实现开源项目 “Goyais” 的 MVP-1：一款本地优先的 AI 辅助编码工具（桌面端），支持事件流、diff、权限确认、SQLite 存储，并预埋“单人远程备份/同步（P0）”能力。项目协议 Apache-2.0。

========================
0. 关键决策（必须遵守）
========================
- UI：Tauri v2 + React（P0）
- 平台优先级：macOS（P0）；Windows/Linux（P1），但工程结构与 CI 从 Day1 保证可移植
- 架构：双栈
  - Host：Tauri(v2) + TypeScript/React
  - Runtime：Python（LangGraph + Deep Agents），通过本地 HTTP + SSE 与 Host 通信
- 远程：自建协作服务器，但 MVP-1 只做 “单人远程备份/同步（P0）”；多人共享 session/数据源 为 P1
- 模型：Provider 抽象（P0） + 内置 OpenAI/Anthropic 适配器（P0）
- 安全：所有敏感工具（写文件、apply_patch、run_command、网络访问）默认需要用户确认 + 写 audit log
- 数据：本地 SQLite（P0）；workspace 使用本地文件系统

========================
1. MVP-1 目标与验收标准
========================
A) macOS 上可运行 Demo：
- 启动 Python runtime 服务
- 启动 Tauri app
- 在 UI 输入任务（例如“把 README 的标题改成 XXX”）
- UI 能实时看到 SSE events（plan/tool_call/tool_result/patch/done）
- UI 展示 unified diff，用户确认后才 apply_patch

B) 审计与权限确认：
- 任意 tool_call 必须产生 event + audit_log（包括参数、结果、是否确认、用户决策）
- 默认拒绝危险命令与越界路径写入（workspace 外禁止写；危险命令黑名单/白名单）
- UI 必须有 Permission Center：对每次敏感调用弹确认（approve/deny）

C) SQLite 落地：
- projects, sessions, runs, events, artifacts, model_configs, audit_logs
- events 采用统一 envelope（event_id/run_id/seq/ts/type/payload）
- 提供 migration/init 脚本

D) 单人远程备份/同步（P0，最小实现）：
- 提供一个可自建的 server（Node/TS 或 Python 均可，但需文档与 docker compose）
- 只考虑单用户：token auth
- 同步对象：events + artifacts 元数据（可先不传大文件，先传文本/patch；大文件 P1）
- API：push/pull（按 seq 或 timestamp 增量）
- 冲突策略：清晰说明（建议 last-write-wins 或 server 为准）

========================
2. 项目目录结构（建议，允许微调但需说明）
========================
/apps/desktop-tauri/            # Tauri Host（React UI、权限确认、事件流展示、diff）
/runtime/python-agent/          # Python Runtime（LangGraph/Deep Agents + 工具 + SSE）
/packages/protocol/             # TS/Python 共用 schema（JSON Schema 或 zod + 生成）与版本号
/server/sync-server/            # 单人同步服务器（MVP-1）
/docs/                          # ADR、威胁模型、安全策略、开发说明
/LICENSE                        # Apache-2.0

========================
3. Host <-> Runtime 协议（必须实现）
========================
- POST /v1/runs
  req:
    {
      "project_id": "string",
      "session_id": "string",
      "input": "string",
      "model_config_id": "string",
      "workspace_path": "string",
      "options": { "use_worktree": boolean }
    }
  resp: { "run_id": "string" }

- GET /v1/runs/{run_id}/events  (SSE)
  以统一 event envelope 推送：
  {
    "event_id": "...",
    "run_id": "...",
    "seq": 1,
    "ts": "...",
    "type": "plan|tool_call|tool_result|patch|error|done",
    "payload": { ... }
  }

- POST /v1/tool-confirmations
  req: { "run_id":"...", "call_id":"...", "approved": true|false }
  resp: { "ok": true }

约定：
- tool_call.payload 必须含：
  { "call_id": "...", "tool_name":"...", "args":{}, "requires_confirmation": true|false }
- tool_result.payload 必须含：
  { "call_id":"...", "ok": true|false, "output": "string|json" }
- patch.payload 必须含：
  { "unified_diff": "string" }

========================
4. 工具集（MVP-1 必须）
========================
文件工具：
- list_dir(path)
- read_file(path)
- search_in_files(query, glob?)
写入工具（必须 requires_confirmation）：
- write_file(path, content)
- apply_patch(unified_diff)
命令工具（必须 requires_confirmation + allowlist）：
- run_command(cmd, cwd)
Git/worktree（可选开关，但建议实现）：
- git_worktree_create(task_id) -> 返回新 worktree 路径
- git_worktree_cleanup(task_id)

安全策略：
- workspace 外路径禁止写
- run_command：默认只允许如 "git status"、"npm test"、"pytest" 等白名单；危险命令黑名单（rm、curl|sh 等）
- 所有拒绝也必须记录 audit

========================
5. Agent Runtime（LangGraph + Deep Agents）
========================
目标：实现最小 coding agent（可用 > 完美）
- 输入任务 -> 输出 plan（plan event）
- 根据任务读取必要文件（tool_call + tool_result）
- 生成 unified diff（patch event）
- 等待 Host 通过 /tool-confirmations 批准后 apply_patch（tool_call/apply_patch -> tool_result）
- 可选：在批准后运行测试命令（run_command）
要求：
- 事件必须边执行边推送（SSE）
- 同步写入 SQLite：runs/events/audit/artifacts
- Provider 可切换：OpenAI/Anthropic 适配器，统一接口（completion/chat + tool calling 若可）

========================
6. Host（Tauri + React UI）
========================
必须有：
- 项目选择/创建（workspace_path）
- 模型配置页面：选择 provider（openai/anthropic），设置 apiKey（存本地加密或 OS keychain；若复杂可先明文+警告，后续增强）
- Run 页面：
  - 输入任务
  - 实时显示 events（按 seq）
  - 遇到 tool_call.requires_confirmation = true 时弹出确认 modal
  - patch 事件渲染 diff（可用现成 diff viewer 组件）
  - 用户确认后调用 /v1/tool-confirmations
- Timeline 与 Diff 需要“可回放”：从 SQLite 读取历史 runs/events

========================
7. 单人同步服务器（P0：备份/同步）
========================
实现一个自建 server（建议 Node/TS + sqlite/postgres 均可；简单优先）：
- Token auth（单用户）
- API：
  - POST /v1/sync/push  { device_id, since_global_seq, events:[...], artifacts_meta:[...] }
  - GET  /v1/sync/pull?since_server_seq=...
- 存储：至少保存 events（append-only）
- 文档：docker compose 一键启动；说明如何在客户端配置 server 地址与 token
客户端侧：
- 提供 “Sync now” 按钮：push 本地新增 events -> pull 远端新增 events
- 冲突策略：单人场景，server 为准或 last-write-wins；需要写在 docs/ADR

========================
8. 文档与测试（必须）
========================
- README：启动步骤（runtime、tauri）、配置 apiKey、运行 demo、同步 server 启动与配置
- docs/ADR：
  - ADR-0001：Host/Runtime 分层与通信协议
  - ADR-0002：事件模型（event envelope）与 SQLite schema
  - ADR-0003：安全策略（工具确认、路径限制、命令白名单）
  - ADR-0004：单人同步语义与冲突策略
- Tests：
  - Python：路径越界禁止写、命令白名单、确认流转逻辑（approve/deny）
  - TS：协议 schema 校验（最少）

========================
9. 实施顺序（严格按此执行，避免跑偏）
========================
Step 1：生成骨架 + protocol schema（TS/Python 共用）+ SQLite schema/migrations
Step 2：实现 Python runtime 的 /runs 与 SSE events 通道（先用 mock agent 发事件）
Step 3：实现 React UI：连接 SSE、渲染 timeline、确认弹窗、展示 diff（先用 mock）
Step 4：实现真实 tools（read/list/search/write/apply_patch/run_command）+ 安全策略 + audit
Step 5：接入 LangGraph/Deep Agents，完成最小 coding agent：plan->read->patch->confirm->apply
Step 6：实现单人同步 server + 客户端 Sync now
Step 7：补齐 README + docs/ADR + tests

请开始产出：先给出完整目录树、关键依赖、协议 schema、SQLite migrations、以及 Step 1-3 的最小可运行 demo（mock agent + UI）。
然后再继续 Step 4-7，确保每一步都可运行并有清晰 commit 粒度。
