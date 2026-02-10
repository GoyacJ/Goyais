# docs/arch/overview.md 约束引用（链接+关键摘录）

- source_path: `docs/arch/overview.md`
- authority_scope: 架构层边界、运行拓扑、发布形态、执行链路

## key_constraints

- v0.1 目标：AI/UI 双入口一致、统一权限与可见性、最小化运行可闭环。
- 最小化运行拓扑：`sqlite + memory + local object store + mediamtx`。
- 单二进制发布冻结：生产必须 Go embed 前端 dist。
- 路由优先级：`/api/v1/*` -> 静态文件 -> `favicon/robots` 策略 -> SPA fallback。
- Header 策略：`index.html` 命中场景返回 `Cache-Control: no-store`；静态文件类型必须正确。
- 执行管道：Validate -> Authorize -> Execute -> Audit -> Event。

## sync_note

本文件用于快速核对架构冻结点，详细语义以 `docs/arch/overview.md` 原文为准。
