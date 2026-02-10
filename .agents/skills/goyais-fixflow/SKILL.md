---
name: goyais-fixflow
description: 当任务是修复 bug、排查回归或处理线上故障并需要可审计交付时触发；默认必须先新建独立 worktree，修复验证后等待用户确认，再仅在 master 工作树执行 no-ff 合并并自动回收线程 worktree。
---

# goyais-fixflow

将“bug 修复 -> 用户确认 -> master 合并 -> worktree 回收”固化为可复用流程，避免在主工作树直接开发与误合并。

## 适用场景

- 需要修复功能缺陷、回归问题、线上故障。
- 需要默认启用 worktree 隔离，减少分支污染。
- 需要在用户确认后把修复合并到 `master`，并自动执行收尾清理。

## 非适用场景

- 纯文档改动、无代码行为修复。
- 全新功能开发或架构重构（应使用垂直切片流程）。
- 只做临时本地实验且不需要可审计交付。

## 输入（必须读取）

- `/Users/goya/Repo/Git/Goyais/AGENTS.md`
- `/Users/goya/Repo/Git/Goyais/.agents/skills/goyais-git/SKILL.md`
- `/Users/goya/Repo/Git/Goyais/.agents/skills/goyais-parallel-threads/assets/worktree_sop.md`
- `/Users/goya/Repo/Git/Goyais/.agents/skills/goyais-parallel-threads/assets/precommit_guard.md`
- `/Users/goya/Repo/Git/Goyais/.agents/skills/goyais-parallel-threads/assets/merge_lane.md`
- `/Users/goya/Repo/Git/Goyais/.agents/skills/goyais-fixflow/assets/bugfix_checklist.md`
- `/Users/goya/Repo/Git/Goyais/.agents/skills/goyais-fixflow/assets/merge_and_cleanup.md`
- `/Users/goya/Repo/Git/Goyais/.agents/skills/goyais-fixflow/assets/delivery_template.md`

## 输出（固定）

- thread/worktree 映射（`thread_id`、`branch`、`worktree_path`）。
- bug 根因与修复说明、验证命令与结果。
- 用户确认后的 `master` 合并记录（含 merge commit）。
- 自动清理结果（worktree remove、branch delete、prune）与风险/回滚说明。

## 严格步骤

1. 先创建独立 bugfix worktree，不允许在当前工作树直接修复。
2. 默认调用 `scripts/create_bugfix_worktree.sh`，不复用既有 thread worktree。
3. 在 thread worktree 内完成：复现、定位、最小修复、回归验证。
4. 输出修复摘要与验证证据，等待用户明确回复“可合并到 master”。
5. 未收到明确确认前，禁止执行任何 `master` 合并动作。
6. 收到确认后，仅在主仓库 `master` 工作树执行 `git merge --no-ff codex/<thread-id>-<topic>`。
7. 合并后执行测试，默认 `go test ./...`，可用参数覆盖测试命令。
8. 默认自动清理：移除 thread worktree、删除 thread 分支、执行 `git worktree prune`。
9. 默认不推送；仅在用户明确要求时执行 `git push origin master`。
10. 输出最终交付摘要（merge commit、测试结果、清理结果、风险与回滚）。

## 默认命令入口

- 创建工作树：`scripts/create_bugfix_worktree.sh --topic <topic>`
- 合并并清理：`scripts/merge_bugfix_to_master.sh --thread-branch codex/<thread-id>-<topic>`

## 验收方式

- `create_bugfix_worktree.sh --dry-run` 输出分支与 worktree 路径。
- 已存在同名分支或 worktree 路径时，创建脚本必须失败退出。
- `merge_bugfix_to_master.sh --dry-run` 输出 no-ff 合并、测试、清理步骤。
- 若未收到用户确认，流程必须停在“交付摘要待确认”状态。
