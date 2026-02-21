# Goyais 产品需求文档（PRD）

## 1. 文档信息
- 文档版本：v0.2.0
- 更新时间：2026-02-21
- 文档范围：Goyais 本地优先 + Hub-First AI 辅助开发桌面应用 (v0.2.0)
- 参考实现：`apps/desktop-tauri`、`runtime/python-agent`、`server/hub-server-go`、`packages/protocol`

## 2. 背景与目标
Goyais 的目标是成为一个安全、可隔离、可扩展的 AI 开发助手：
- **Hub-First 架构**：无论是本地模式还是远程模式，都统一通过 Hub Server (Go) 作为控制面和数据权威。
- **Session-Centric**：从以 Run 为中心升级为以 Session 为中心，引入互斥锁保证同一会话的安全并发。
- **隔离执行**：默认在 Git Worktree 中隔离执行 Agent 任务，防止破坏用户正在开发的主工作区。
- **生态扩展**：支持动态注入 Skills 和 Model Context Protocol (MCP) Connectors。

## 3. 当前实现状态（As-Is v0.2.0）

### 3.1 代码与测试验证结论
- `pnpm test`：全仓通过。
- `go test ./...`：Hub Server Go 端全量通过。
- `v0.2.0_REFACTOR_PLAN.md` 中规划的 8 个 Phase (0-7) 已经**全部完成**。

### 3.2 能力矩阵
- **Session 互斥与调度**：Hub 控制并发，防冲突 (409 SESSION_BUSY)。
- **工作区隔离**：基于 `git worktree` 的沙盒执行，UI 侧支持审阅 Patch、直接 Commit 或 Discard。
- **双执行模式**：
  - Plan 模式：Agent 产出执行计划 -> 等待用户确认 -> 批准后执行。
  - Agent 模式：自主运行，仅对高危操作（文件写入、终端命令等）拦截确认。
- **动态能力注入**：支持 Hub 端配置 Skill Sets 和 MCP Connectors，运行时自动组装。
- **远程代码库同步**：支持通过 Git URL 添加远程 Project，Hub 后台执行 clone 和 pull。
- **可观测与容错**：SSE 断线续传，Hub Watchdog 定期清理死锁 Execution，详尽的 Audit Logs 审计日志。

### 3.3 已识别缺口 (下一阶段重点)
- **多任务排队**：目前遇到会话忙碌直接拒绝 (409)，未来考虑实现执行队列排队机制。
- **并发配额**：Workspace / Project 级别的最大并发执行数量控制。
- **多文件上下文推理**：针对超大型重构任务的上下文组装和缓存优化。

## 4. 产品范围定义

### 4.1 v0.2.0 已交付范围
- 统一 Hub-First 架构：Desktop + Go Hub + Python Worker
- Session 互斥状态机与 SSE 事件分发
- Git Worktree 隔离与 UI 侧 Commit 交互
- Skills 与 MCP 扩展系统
- 远程仓库 (Remote Project) 同步
- 断线重连与看门狗超时恢复机制

### 4.2 v0.3.0（下一阶段）范围
- 任务执行队列排队机制（替代直接拒绝）。
- 工作区 (Workspace) 并发配额与计费限制限制。
- 深度多文件重构 Agent 规划能力提升。

## 5. 用户与角色
- **个人开发者**：本地快速编码，利用沙盒安全尝试 AI 的修改，确认无误后 Commit。
- **小团队技术负责人**：通过远程 Hub 管理团队级 Prompt Skills 和内部 MCP 服务，控制统一的模型 API Key。
- **平台管理员**：部署 Go Hub 和 Worker Pool，分配 Workspace 并监控审计日志。

## 6. 核心使用场景
1. **沙盒式编码重构**：新建 Session，提出重构需求。Agent 在独立 worktree 中修改代码。用户在侧边栏 Review Diff，点击 Commit 自动合并，或者发现问题直接 Discard。
2. **连接内部知识库/工具**：在 Hub 添加公司内部的 MCP Connector。创建 Session 时勾选该 MCP。Agent 自动获取调用该内部 API 的能力。
3. **安全审批**：Agent 在执行过程中试图运行 `npm install` 等高危命令，执行被挂起。Desktop 弹出确认框，用户审查命令后点击 Approve 放行。
4. **远程仓库协作**：输入 Git 仓库地址和凭证，Hub 自动 Clone 代码。开发者可以在不拉取代码到本地的情况下，让 Agent 分析和修改远程仓库代码（配合云端 Worker）。

## 7. 详细需求 (v0.2.0 实现基准)

### 7.1 Session 生命周期
- 必须选择 Mode (Plan/Agent)、Model 以及可选的 Skills/MCP。
- 执行时获取 Session 排他锁，产生 Execution。

### 7.2 安全与隔离
- 操作在 `goyais-exec-<id>` worktree 中进行。
- 危险操作 (`write_fs`, `exec`, `network`, `delete`) 触发阻断式确认。
- Git 操作通过 Hub 代理，记录用户的 Git Name 和 Email。

### 7.3 可观测与诊断
- Hub 负责生成全局 `trace_id`。
- 所有变更和确认操作都会落入 `audit_logs` 表。

## 8. 非功能需求
- **高性能推送**：Hub Server 使用 Go Goroutine 处理 SSE 连接，确保高并发下的事件实时触达。
- **架构极简**：Hub 被编译为单一二进制文件，去除对 Node.js 运行时的依赖（相较于旧版 TS Hub）。
- **容灾机制**：Worker 崩溃不影响 Hub 状态，Watchdog (30s 扫描) 自动释放锁。

## 9. 成功指标（KPI）
- Session 冲突解决率 (用户选择 Cancel Current 或新建的比例)
- Worktree 最终合并 Commit 的转化率 (衡量 Agent 生成代码的采纳率)
- 敏感工具 (MCP, 命令) 被阻断拦截的次数与拒绝率
- SSE 断线重连的成功率


