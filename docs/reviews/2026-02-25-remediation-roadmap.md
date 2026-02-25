# Goyais 整改路线图（2026-02-25，更新版）

## 1. 目标

- 目标 A：完成上线阻塞项清零（P0/P1）。
- 目标 B：完成工程治理闭环（质量门禁可复现 + 文档治理可执行）。

## 2. 当前状态快照

- Track A（P0/P1）状态：**Done**。
- Track B（P2/P3）状态：**Done（技术项）**。
- 剩余问题：无代码级整改项。

## 3. 已完成整改（可归档）

### A1 认证与多租户边界（F-001/F-002/F-003/F-008）
- 状态：Done
- 结果：无 token 默认放行链路已收紧，workspace 授权绑定修复，工作区与列表接口已补鉴权/收敛。

### A2 SSE 远程鉴权闭环（F-004）
- 状态：Done
- 结果：Hub 支持 SSE `access_token` 查询参数，远程会话流鉴权闭环打通。

### A3 Patch 导出语义修复（F-005）
- 状态：Done
- 结果：Patch 导出收敛到 execution 关联路径，避免仓库全量脏改动混入。

### A4 内部 token 强制化（F-006）
- 状态：Done
- 结果：Hub/Worker 默认不再依赖固定 token；缺配置 fail fast（可通过显式不安全开关用于本地开发）。

### A5 Worker 命令执行防护（F-007）
- 状态：Done
- 结果：移除 `shell=True`，引入 tokenized allowlist 与危险模式阻断。

### B1-部分 质量门禁一致性（F-009/F-012）
- 状态：Done
- 结果：Worker lint 问题已修复且 CI 已纳入 `ruff`。

### B2 SSE 回放鲁棒性（F-010）
- 状态：Done
- 结果：`last_event_id` 未命中时返回重同步信号，Desktop 已实现 resync 拉全量状态。

### B3 文档治理（F-013）
- 状态：Done（Draft）
- 结果：`docs/PRD.md` 已补齐最小模板并形成 v0.1 Draft，待联合评审。

## 4. 剩余整改任务

当前无代码级整改剩余任务。已完成项摘要：

1. `@vitest/coverage-v8` 依赖固化。
2. coverage 统计口径收敛到业务可单测代码。
3. 覆盖率阈值改为分指标/分层级门禁（overall + core）。
4. `cd apps/desktop && pnpm coverage:gate` 已通过。

## 5. 建议执行顺序（剩余）

1. 已完成首版 RC 发布演练记录：`docs/reviews/2026-02-25-rc-rehearsal-record.md`。
2. 使用 `docs/reviews/2026-02-25-release-signoff-record.md` 完成正式发布签字流程。

## 6. 发布准入门槛（更新）

### Go
1. P0/P1 全部关闭（已达成）。
2. Desktop/Hub/Worker 的 test + lint 全部通过（已达成）。
3. Desktop coverage gate 通过（已达成）。
4. PRD 已有 Draft（已达成），release checklist 到位（已达成）。

### No-Go
1. 未按 release checklist 完成发布演练与记录留存。
