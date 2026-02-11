# Rule 03: Git Worktree Lifecycle

## Trigger Conditions

- 任何代码修改、脚本修改、文档契约修改。

## Hard Constraints (MUST)

- 每个 thread 使用独立 worktree，且通过 `bash .agents/skills/goyais-worktree-flow/scripts/create_worktree.sh --topic <topic>` 创建。
- worktree 默认目录为 `<repo>/.worktrees/`。
- 分支命名：`goya/<thread-id>-<topic>`。
- thread worktree 仅用于开发与验证；主工作树仅用于集成与回归。
- 提交前执行 `git diff --cached --name-only` 与 `bash go_server/scripts/git/precommit_guard.sh`。
- 线程收口必须通过 `bash .agents/skills/goyais-worktree-flow/scripts/merge_thread.sh --thread-branch <goya/...>` 执行 no-ff 合并与本地 cleanup（worktree + branch）。
- 禁止使用手工 `git merge` / `git branch -d` / `git worktree remove` 绕过标准收口流程。

## Counterexamples

- 在主工作树直接开发多个需求。
- 一个 worktree 来回切换多个线程分支。
- 已合并 thread 分支但未回收本地 worktree 与本地分支。

## Validation Commands

- `bash .agents/skills/goyais-worktree-flow/scripts/create_worktree.sh --help`
- `bash .agents/skills/goyais-worktree-flow/scripts/merge_thread.sh --help`
- `git worktree list`
- `bash go_server/scripts/git/worktree_audit.sh`
- `bash go_server/scripts/git/merged_thread_cleanup_audit.sh`
- `bash go_server/scripts/git/precommit_guard.sh`
