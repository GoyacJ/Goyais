---
name: goyais-worktree-flow
description: 统一执行 goya/* 分支工作流，确保一线程一 worktree、提交边界和安全回收。
---

# goyais-worktree-flow

## 适用场景

- 任何需要编码与提交的任务。

## 输入

- `AGENTS.md`
- `go_server/scripts/git/precommit_guard.sh`
- `go_server/scripts/git/worktree_audit.sh`
- `scripts/create_worktree.sh`
- `scripts/merge_thread.sh`

## 输出

- thread/worktree 映射。
- 提交前防呆检查结果。
- 合并与回收记录。

## 严格步骤

1. 从 `master` 创建 `goya/<thread-id>-<topic>` worktree。
2. 在 thread worktree 开发与验证。
3. 提交前执行 guard 与 staged 范围检查。
4. 在主工作树执行 no-ff 合并并回收 worktree。

## 验收

- `git worktree list` 无重复分支绑定。
