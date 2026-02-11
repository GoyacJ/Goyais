# Contributing to Goyais

感谢你参与 Goyais。本文档定义了本仓库的贡献流程与质量门槛。  
Thank you for contributing to Goyais. This guide defines contribution workflow and quality gates.

## 1. 贡献前必读 | Read Before You Start

- `docs/prd.md`（产品需求基线 / PRD baseline）
- `AGENTS.md`（根级工程强约束 / root hard constraints）
- `go_server/AGENTS.md` 与 `vue_web/AGENTS.md`（模块约束 / module constraints）
- `go_server/docs/acceptance.md`（验收与证据命令 / acceptance and evidence commands）

## 2. 分支与 Worktree 规范 | Branch and Worktree Policy

本仓库采用“一线程一 worktree”策略。  
This repository enforces one thread per worktree.

- 分支命名：`goya/<thread-id>-<topic>`
- 必须从 `master` 创建 thread 分支
- 主工作树仅用于集成与回归，不用于日常开发

示例 | Example:

```bash
git worktree add .worktrees/goya-thread20260211-my-topic -b goya/thread20260211-my-topic master
```

## 3. 开发过程要求 | Development Requirements

### 3.1 契约同步（强制） | Contract Sync (MUST)

当以下内容变化时，必须在同一变更中同步更新文档：
- API 路径/请求响应/错误结构/分页语义
- 核心实体字段、状态机、生命周期
- 可见性/ACL 判定规则
- provider 配置键名/默认值/优先级

至少同步：
- `go_server/docs/api/openapi.yaml`
- `go_server/docs/arch/overview.md`
- `go_server/docs/arch/data-model.md`
- `go_server/docs/arch/state-machines.md`
- `go_server/docs/acceptance.md`

### 3.2 源码头规范（强制） | Source Header Policy (MUST)

源码（含测试）必须包含标准头（SPDX/版权/Author/Created/Version/Description）。

校验命令 | Validation:

```bash
bash go_server/scripts/ci/source_header_check.sh
```

自动回填（幂等） | Auto backfill (idempotent):

```bash
bash go_server/scripts/ci/source_header_backfill.sh
```

### 3.3 代码注释要求 | Commenting Rule

只注释“为什么”（边界、兼容、权限、安全、性能取舍），避免逐行翻译式注释。

## 4. 提交前检查 | Pre-Commit Checklist

最小必跑命令：

```bash
git diff --cached --name-only
bash go_server/scripts/git/precommit_guard.sh
bash go_server/scripts/ci/contract_regression.sh
```

## 5. Commit 与 PR 规范 | Commit and PR Rules

- 推荐使用 Conventional Commits（如 `feat:`, `fix:`, `chore:`）
- 一个 PR 聚焦一个主题，避免混入无关改动
- 机械改动（如批量格式化、批量头注释回填）应与功能改动分开提交
- PR 描述需包含：变更范围、风险、回滚路径、验证证据

## 6. PR 审核重点 | PR Review Focus

- 行为回归风险（behavior regression risk）
- 权限与隔离边界（authz/isolation boundary）
- 契约漂移（contract drift）
- 测试与验收证据（tests and acceptance evidence）

## 7. 行为准则与安全报告 | Conduct and Security

- 社区行为规范：`CODE_OF_CONDUCT.md`
- 漏洞报告流程：`SECURITY.md`
- 日常支持渠道：`SUPPORT.md`
