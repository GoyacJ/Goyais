---
name: goyais-parallel-threads
description: 当任务需要在 Codex 中并行启动多个 thread 开发同一仓库，且要求 Git 互不干扰时触发；当任务只在单分支单工作树完成时不触发。
---

# goyais-parallel-threads

将 Goyais 的并行 thread 开发流程固化为“一线程一 worktree”的可执行 SOP，避免分支污染与误提交。

## 适用场景

- 需要同时推进多个 thread（例如 Thread A/B/C）并行开发。
- 需要隔离不同 thread 的工作区、依赖安装、构建产物与未跟踪文件。
- 需要明确 master 集成通道与 thread 开发通道。

## 非适用场景

- 只在单个分支上做短期改动，不并行。
- 仅本地一次性实验，不形成可审计提交。
- 与仓库 Git 流程无关的纯业务实现任务。

## 输入（需要哪些仓库文件）

- `AGENTS.md`
- `../goyais-git/SKILL.md`
- `assets/worktree_sop.md`
- `assets/precommit_guard.md`
- `assets/merge_lane.md`

## 输出（会改哪些文件/会生成哪些文件）

- 输出 thread-worktree 映射表。
- 输出每个 thread 的标准分支命名与创建命令。
- 输出提交前防呆检查结果。
- 输出 master 集成合并顺序与回归记录。

## 严格步骤

1. 按 `assets/worktree_sop.md` 建立并维护“一线程一 worktree”。
2. 每个 thread 分支必须从本地 `master` 创建，命名 `codex/<thread-id>-<topic>`。
3. 提交前必须执行 `assets/precommit_guard.md` 中的 staged 防呆检查。
4. 严禁在同一 worktree 切换多个 thread 分支。
5. 仅在 master 集成工作树执行合并，流程遵循 `assets/merge_lane.md`。
6. 任何分支指针移动前必须先创建 `backup tag + backup branch`。
7. 若必须覆盖远端历史，仅允许 `--force-with-lease`，并先给出待推送 commit 清单与风险说明。
8. thread 完成后回收 worktree 并执行 `git worktree prune`。

## 验收方式

- `git worktree list` 显示每个 thread 独立路径与分支绑定。
- 任意 thread 提交前都能通过防呆检查且不包含禁止路径。
- master 集成记录可回放（merge 顺序、回归命令、结果）。
- 分支指针修复动作具备 backup tag/branch 审计证据。
