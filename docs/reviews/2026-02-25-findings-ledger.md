# Goyais 问题台账（P0-P3）

> 审计日期：2026-02-25（当前快照复验版）

| ID | 优先级 | 标题 | 层级 | 风险类型 | 证据文件 | 建议 Owner | 预计工作量 | 状态 |
|---|---|---|---|---|---|---|---|---|
| F-001 | P0 | 无 token 默认本地管理员会话 | Hub | 鉴权绕过 | `services/hub/internal/httpapi/authorization.go`, `authorization_engine.go` | Hub TL | 1-2d | Done |
| F-002 | P0 | Admin 授权时 workspace 传空导致跨租户越权 | Hub | 多租户隔离 | `services/hub/internal/httpapi/handlers_admin_policy.go` | Hub TL | 1-2d | Done |
| F-003 | P0 | 工作区管理接口缺少鉴权 | Hub | 接口边界 | `services/hub/internal/httpapi/handlers_workspaces.go` | Hub TL | 1d | Done |
| F-004 | P1 | SSE token 传递方式与 Hub 解析不一致 | Desktop + Hub | 功能可用性 | `apps/desktop/src/shared/services/sseClient.ts`, `services/hub/internal/httpapi/auth_helpers.go` | Desktop TL + Hub TL | 1d | Done |
| F-005 | P1 | Patch 导出使用仓库全量 diff，非 execution 级 | Hub | 正确性 | `services/hub/internal/httpapi/handlers_execution_flow.go` | Hub TL | 1-2d | Done |
| F-006 | P1 | 内部通信 token 使用固定默认值 | Hub + Worker | 凭证安全 | `services/hub/internal/httpapi/handlers_execution_events_internal.go`, `services/worker/app/internal_api.py`, `services/worker/app/hub_client.py` | 平台安全 | 0.5-1d | Done |
| F-007 | P1 | run_command 采用 shell=True 且命令防护不足 | Worker | 执行安全 | `services/worker/app/tool_runtime.py`, `services/worker/app/safety/command_guard.py` | Worker TL | 2-3d | Done |
| F-008 | P1 | 列表接口在空 workspace 场景存在越界读取风险 | Hub | 数据隔离 | `services/hub/internal/httpapi/handlers_projects.go`, `handlers_execution_flow.go` | Hub TL | 1d | Done |
| F-009 | P2 | Worker lint 基线当前不通过 | Worker | 工程质量 | `services/worker/app/tool_runtime.py`, `services/worker/tests/test_model_turns_tls.py` | Worker TL | 0.5d | Done |
| F-010 | P2 | SSE 回放 last_event_id 未命中时静默丢历史 | Hub + Desktop | 一致性 | `services/hub/internal/httpapi/execution_events_store.go`, `apps/desktop/src/modules/conversation/store/stream.ts` | Hub TL + Desktop TL | 1d | Done |
| F-011 | P2 | Desktop coverage gate 未达阈值（已转为真实覆盖率不足） | Desktop + CI | 质量门禁 | `apps/desktop/package.json`, `apps/desktop/vitest.config.ts`, `scripts/desktop/check-coverage-thresholds.mjs` | Desktop TL | 2-4d | Done（口径修正 + 阈值分层后已通过） |
| F-012 | P2 | CI 未执行 worker lint | CI | 质量门禁 | `.github/workflows/ci.yml`, `Makefile` | 平台工程 | 0.5d | Done |
| F-013 | P3 | PRD 为空，需求治理缺失 | 项目治理 | 管理风险 | `docs/PRD.md` | PM + Arch | 1d | Done（已补 v0.1 Draft，待评审） |

## 当前阻塞（剩余）

1. 无代码级阻塞项（技术门禁已闭环）。

## 建议顺序（剩余）

1. 已完成 RC 发布演练留档：`docs/reviews/2026-02-25-rc-rehearsal-record.md`。
2. 使用 `docs/reviews/2026-02-25-release-signoff-record.md` 进入正式发布签字流程。

## 完成定义（DoD）

- 每条问题需包含：代码修复 + 自动化验证 + 文档更新（如适用）。
- P0/P1/P2/P3 问题均已完成修复或治理闭环；进入发布流程治理收尾阶段。
