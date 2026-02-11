# Goyais Java Server v0.1 Development Plan

## 1. 周期与组织

- 周期：6 Sprint，双周迭代，共 12 周。
- 团队：2 个后端小队。
- Squad-A：Auth/Security（单应用拓扑 + 动态权限）。
- Squad-B：Command/Domain/Data/Capability。

## 2. Sprint 路线图

| Sprint | 周期 | Squad-A | Squad-B | 验收门禁 |
|---|---|---|---|---|
| S1 | W1-W2 | 单应用拓扑骨架、OIDC 元数据端点 | Maven 模块骨架、`/healthz` | 单应用可启动 |
| S2 | W3-W4 | password/oidc 登录链路、JWT claims | command baseline + `/api/v1/commands` + error envelope | command 202 合规 |
| S3 | W5-W6 | `policyVersion + Redis invalidation` | authz gate + row-level data permission | 动态权限即时生效 |
| S4 | W7-W8 | `single/resource-only` 切换治理 | assets/workflow/shares 同构落地 | 契约字段同构 |
| S5 | W9-W10 | 安全策略收敛与审计增强 | cache/event/messaging/storage provider 切换 | minimal/full 均可跑 |
| S6 | W11-W12 | 安全加固与发布策略 | 回归/性能/发布回滚演练 | v0.1 gates 全绿 |

## 3. 固定 DoD

- API/数据模型/状态机文档与实现同变更同步。
- JavaDoc 与文件头门禁通过。
- 单测+集成测试通过。
- Vue 联调通过且无契约破坏。
- 审计可回查 command/authz/egress/policyVersion。

## 4. 分支与 worktree 规则

- 一 story 一 worktree。
- 分支前缀：`goya/<thread-id>-<topic>`。
- 提交前执行：
  - `git diff --cached --name-only`
  - `bash /Users/goya/Repo/Git/Goyais/go_server/scripts/git/precommit_guard.sh`

## 5. 里程碑

- M0（S1-S2）：单应用框架 + command 入口 + 认证骨架。
- M1（S3-S4）：动态权限闭环 + 核心业务域。
- M2（S5）：通用能力封装 + provider 切换。
- M3（S6）：稳定性、发布、灰度就绪。

## 6. 风险与缓解

- 风险：单应用与 resource-only 配置漂移。
  - 缓解：双模式启动测试 + healthz 拓扑标识。
- 风险：动态权限缓存不一致。
  - 缓解：Redis invalidation + 本地 fallback + 演练脚本。
- 风险：注释规范回归失败。
  - 缓解：`java_javadoc_check.sh` 纳入统一回归。
