# ADR-0007: Hub Control Plane Phase 1（Bootstrap/Auth/Workspace/Navigation）

## Status
Accepted

## Context
Desktop 需要引入 Remote Workspace 作为菜单与权限的权威来源，同时必须保留现有 Local Workspace 模式并保持 runtime/python-agent 协议与 API 行为不变。Phase 1 仅实现控制面能力，不引入 runs/SSE/tools/audit。

## Decision
- 引入独立 `hub-server`（Fastify + SQLite），与 runtime/sync 解耦。
- 认证方案采用 `opaque bearer token`：
  - 服务端只存 `token_hash`（`auth_tokens` 表），不存 token 明文。
  - 默认 TTL 为 7 天（`GOYAIS_TOKEN_TTL_SECONDS` 可调）。
- Bootstrap 安全默认：
  - `setup_mode = (users_count == 0) OR (system_state.setup_completed == 0)`。
  - 仅 `setup_mode=true` 时允许创建首个 admin。
  - 必须提供 `GOYAIS_BOOTSTRAP_TOKEN` 且请求体 `bootstrap_token` 匹配。
  - 完成 bootstrap 后自动创建 `Default` workspace、`Owner/Member` 系统角色、权限与菜单绑定、admin membership，并写入 `setup_completed=1`。
- Navigation 下发策略：
  - 根据 `workspace_members.role_id` 聚合 `role_permissions` 与 `role_menus`。
  - 菜单按 `sort_order ASC, menu_id ASC` 构树；若父节点不可见，子节点提升至根。
- 统一可观测性与错误模型：
  - 所有 HTTP 响应回传 `X-Trace-Id`，缺失时服务端生成。
  - 错误统一 `{ error: { code, message, trace_id, retryable, details? } }`。

## Consequences
- Remote Workspace 能在不改 runtime 协议的前提下先落地“身份/工作区/导航/权限”基础能力。
- Opaque token 便于服务端吊销与过期控制，Phase 2 若引入 JWT 需要兼容迁移策略。
- Desktop 可按工作区切换远端菜单与权限；本地模式保持原有行为，支持渐进迁移。
