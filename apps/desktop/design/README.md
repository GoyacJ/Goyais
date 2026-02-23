# Desktop Design Ownership (v0.4.0)

## Token 文件所有权

- `src/styles/tokens.css` 由 Pencil 同步管理。
- Antigravity / Goyais Desktop 作为消费方，仅使用 token，不直接修改 token 定义。
- 本轮设计基线文档：`/Users/goya/Repo/Git/Goyais/apps/desktop/design/goyais-v0.4.0-design-system.pen`（Pencil 编辑器内）。

## 设计范围（本次）

- 三层 Token：`global -> semantic -> component`
- 双主题：`dark`（主） + `light`（辅）
- UI Kit：Button / Input / Select / Badge / Card / SidebarItem / Topbar / Modal(RiskConfirm) / Toast(Inline Alert)
- Screens（严格按 `.pen` 13 面）：
  - `/main`
  - `/remote/account`
  - `/remote/members-roles`
  - `/remote/permissions-audit`
  - `/workspace/agent`
  - `/workspace/model`
  - `/workspace/rules`
  - `/workspace/skills`
  - `/workspace/mcp`
  - `/settings/theme`
  - `/settings/i18n`
  - `/settings/updates-diagnostics`
  - `/settings/general`

## 同步流程（v0.4.0）

1. 设计侧在 Pencil 更新 token。
2. 同步到 `src/styles/tokens.css`，确保 light/dark 均可用。
3. 前端验证核心状态：空态/错误态/加载态/权限拒绝态（403）。
4. 运行 `pnpm --dir apps/desktop check:tokens` 校验 token 与设计基线未漂移。
5. 业务组件仅消费 token，不在组件内硬编码颜色/间距/字号/圆角。

## 状态命名映射（UI 文案层）

- `done` / `completed` -> `success`
- `error` / `failed` -> `failed`
- `cancelled` / `stopped` -> `cancelled`
- `queued` -> `queued`
- `running` / `executing` -> `running`
- `confirming` -> `confirming`
- `connected` / `reconnecting` / `disconnected` -> `connected` / `degraded` / `disconnected`

## 冲突处理原则

- 业务样式冲突优先回到 token 层协商。
- 禁止在业务组件里直接改 token 源值来绕过同步流程。
