# Goyais Java Server v0.1 Development Plan

## 1. 周期与组织

- 周期：6 Sprint，双周迭代，共 12 周。
- 团队：2 个后端小队。
- Squad-A：Auth/Security。
- Squad-B：Command/Domain/Data/Capability。

## 2. Sprint 路线图

| Sprint | 周期 | Squad-A | Squad-B | 验收门禁 |
|---|---|---|---|---|
| S1 | W1-W2 | auth server skeleton + OIDC base + password login | maven modules + api server boot + healthz | 双应用可启动 |
| S2 | W3-W4 | sms/oidc/social login + JWT claims | command baseline + `/api/v1/commands` + error envelope | command 202 合规 |
| S3 | W5-W6 | RBAC + policyVersion refresh | authz gate + acl/visibility + row-level data permission | SQL 过滤语义通过 |
| S4 | W7-W8 | SSO session governance | assets/workflow/shares domain sugar -> command | 契约字段同构 |
| S5 | W9-W10 | high-risk policy hardening | cache/event/messaging/storage capability wrappers | minimal/full 均可跑 |
| S6 | W11-W12 | security hardening and remediation | regression/perf/release/rollback drill | v0.1 gates 全绿 |

## 3. 固定 DoD

- API/数据模型/状态机文档与实现同变更同步。
- 单测+集成测试通过。
- Vue 联调通过且无契约破坏。
- 审计可回查 command/authz/egress。
- 回滚路径经过演练并有记录。

## 4. 分支与 worktree 规则

- 一 story 一 worktree。
- 分支前缀：`goya/<thread-id>-<topic>`。
- 提交前执行：
  - `git diff --cached --name-only`
  - `bash /Users/goya/Repo/Git/Goyais/go_server/scripts/git/precommit_guard.sh`

## 5. 里程碑

- M0（S1-S2）：基础框架 + 认证骨架 + command 入口。
- M1（S3-S4）：权限闭环 + 核心业务域。
- M2（S5）：通用能力封装 + provider 切换。
- M3（S6）：稳定性、发布、灰度就绪。

## 6. 风险与缓解

- 风险：Auth 与 API 契约漂移。
  - 缓解：OpenAPI diff + contract tests 每 Sprint 执行。
- 风险：数据权限规则复杂度上升。
  - 缓解：统一 SQL 策略器 + 回归矩阵。
- 风险：外部 provider 不稳定。
  - 缓解：memory/local fallback + feature gate 回滚。
