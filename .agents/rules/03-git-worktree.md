# Rule 03: Git Worktree Isolation

## Trigger Conditions

- 任何代码修改、脚本修改、文档契约修改。

## Hard Constraints (MUST)

- 每个 thread 使用独立 worktree。
- 分支命名：`goya/<thread-id>-<topic>`。
- 主工作树仅用于集成与回归。
- 提交前执行 `precommit_guard.sh`。

## Counterexamples

- 在主工作树直接开发多个需求。
- 一个 worktree 来回切换多个线程分支。

## Validation Commands

- `git worktree list`
- `bash go_server/scripts/git/worktree_audit.sh`
- `bash go_server/scripts/git/precommit_guard.sh`
