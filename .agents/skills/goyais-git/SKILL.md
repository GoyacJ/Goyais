---
name: goyais-git
description: 当任务涉及分支管理、提交规范、PR 协作、版本发布或回滚流程时触发；当任务只需本地一次性试验且不进入协作交付流程时不触发。
---

# goyais-git

定义 Goyais 仓库的 GitHub Flow 协作规范，统一提交质量与可回滚性。

## 适用场景

- 准备开始开发分支并计划提 PR。
- 需要统一 commit message、PR 描述与评审清单。
- 需要制定版本发布与故障回滚步骤。
- 需要在多 thread 并行场景下保证 Git 隔离。

## 非适用场景

- 临时本地调试，不形成提交与 PR。
- 与仓库无关的通用 Git 教学。
- 仅做草稿讨论，无需形成可审查变更。

## 输入（需要哪些仓库文件）

- `AGENTS.md`
- `docs/spec/v0.1.md`
- `docs/acceptance.md`
- `assets/pr_template.md`
- `assets/commit_examples.md`
- `assets/review_checklist.md`
- `assets/version_and_rollback.md`

## 输出（会改哪些文件/会生成哪些文件）

- 生成规范化 commit message。
- 生成标准 PR 描述与 review checklist。
- 输出版本发布与回滚执行记录（如 tag 说明、revert 记录）。
- 并行 thread 场景下输出 worktree 隔离执行清单。

## 严格步骤

1. 采用 GitHub Flow：从 `master` 拉短生命周期分支开发，完成后通过 PR 合并回 `master`。
2. 提交信息必须使用 Conventional Commits，优先在 scope 中标注模块。
3. 若任务涉及多 thread 并行，必须调用 `goyais-parallel-threads` 并执行“一线程一 worktree”隔离流程。
4. PR 描述必须使用 `assets/pr_template.md`，强制填写范围、风险、验收证据与契约同步项。
5. 评审时使用 `assets/review_checklist.md`，先看阻断风险，再看实现细节。
6. 若变更触及契约，必须在同一 PR 同步 `docs/api/openapi.yaml`、`docs/arch/data-model.md`、`docs/arch/state-machines.md`、`docs/arch/overview.md`、`docs/acceptance.md`。
7. 版本与回滚按 `assets/version_and_rollback.md` 执行，禁止不可审计的回滚方式。

## 验收方式

- PR 可直接审阅并覆盖变更背景、风险、验收证据与文档同步。
- commit message 可机读且符合 Conventional Commits。
- 回滚方案可在不重写历史的前提下执行并追踪。
