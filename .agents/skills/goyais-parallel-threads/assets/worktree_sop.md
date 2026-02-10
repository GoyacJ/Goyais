# Worktree SOP（并行 Thread）

## 目标

- 一个 thread 对应一个独立 worktree。
- 主仓库工作树只用于 `master` 集成与回归。

## 标准命名

- 分支：`codex/<thread-id>-<topic>`
- 工作树目录：`/Users/goya/Repo/Git/Goyais-wt-<thread-id>`

## 创建流程

```bash
cd /Users/goya/Repo/Git/Goyais
git switch master
git fetch origin --prune
git pull --ff-only

THREAD="thread6-xxx"
git worktree add "/Users/goya/Repo/Git/Goyais-wt-${THREAD}" -b "codex/${THREAD}" master
```

## 日常同步（在线程工作树）

```bash
cd /Users/goya/Repo/Git/Goyais-wt-<thread-id>
git fetch origin --prune
git merge --no-ff master
```

## 可视化盘点

```bash
cd /Users/goya/Repo/Git/Goyais
git worktree list
git branch -vv
```

## 回收流程

```bash
cd /Users/goya/Repo/Git/Goyais
git worktree remove "/Users/goya/Repo/Git/Goyais-wt-<thread-id>"
git worktree prune
```
