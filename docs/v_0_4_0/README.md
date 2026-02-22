# Goyais v0.4.0 文档总览（Rewrite Authority）

> 本目录是 v0.4.0 的权威文档集。  
> v0.4.0 为 Clean-Slate Rewrite：完全重写，不兼容旧版本实现。

## 文档索引

| 文档 | 角色 | 使用时机 |
|------|------|---------|
| [PRD.md](./PRD.md) | 产品权威源 | 定义业务边界、不可变决策、验收标准 |
| [TECH_ARCH.md](./TECH_ARCH.md) | 架构权威源 | 设计对象模型、接口、状态机、安全与部署 |
| [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md) | 实施权威源 | 规划阶段、依赖、里程碑、上线门槛 |
| [DEVELOPMENT_STANDARDS.md](./DEVELOPMENT_STANDARDS.md) | 开发权威源 | 约束编码、测试、安全、评审、DoD |

## 推荐阅读顺序

1. `PRD.md`：先确认业务口径和 P0/P1 边界。
2. `TECH_ARCH.md`：再看如何技术落地。
3. `IMPLEMENTATION_PLAN.md`：按阶段拆分实现路径。
4. `DEVELOPMENT_STANDARDS.md`：最后执行开发规范与验收规范。

## v0.4.0 核心口径（摘要）

1. 产品定位：AI 智能平台（AI Coding + 通用 Agent）。
2. 主对象模型：Workspace -> Project -> Conversation -> Execution。
3. 工作区模式：本地唯一工作区（免登录、全能力）+ 远程工作区（RBAC + 核心 ABAC）。
4. 执行模型：多 Conversation 并行；单 Conversation 严格 FIFO；Stop 只终止当前执行。
5. 资源体系：models/rules/skills/mcps 工作区资源池 + 项目绑定 + 会话覆盖。
6. 共享机制：本地来源资源先导入远程私有副本，再经管理员审批共享。
7. 模型密钥：允许共享，但必须高风险审批、审计、掩码展示、可撤销。
8. UI 设计：推荐使用 Pencil MCP 设计方法，参考 [Pencil Docs](https://docs.pencil.dev)。

## P0 / P1 规则

1. P0 必须形成完整业务闭环并满足 Go/No-Go 条件。
2. P1 可以延期，不阻塞 v0.4.0 发布。
3. 任何改动不得出现 P0/P1 语义混写或冲突叙述。

## 文档一致性规则

1. 修改业务规则时，必须同步更新 `PRD.md` 与 `TECH_ARCH.md`。
2. 修改实现阶段时，必须同步更新 `IMPLEMENTATION_PLAN.md`。
3. 修改开发与验收要求时，必须同步更新 `DEVELOPMENT_STANDARDS.md`。
4. 若文档之间冲突，以 `PRD.md` 为最终业务裁决，再回写其余文档。

## 历史版本说明

| 版本 | 状态 | 说明 |
|------|------|------|
| v0.1.x | 废弃 | MVP 阶段，不再作为约束来源 |
| v0.2.x | 废弃 | Hub-First 探索阶段，仅可参考经验 |
| v0.3.x | 废弃 | Agent 重构未收敛，不作为实现依据 |
| v0.4.0 | 当前 | 统一权威文档集，按本目录执行 |
