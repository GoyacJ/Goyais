# goyais-fixflow Merge and Cleanup

## 合并前置（必须）

- 用户已明确确认“可合并到 master”。
- 主仓库工作树保持 clean。
- 目标 thread 分支存在于本地（`codex/<thread-id>-<topic>`）。

## 推荐入口（脚本）

```bash
cd /Users/goya/Repo/Git/Goyais
bash .agents/skills/goyais-fixflow/scripts/merge_bugfix_to_master.sh \
  --thread-branch codex/thread39-command-status
```

默认行为：

- `git switch master`
- `git fetch origin --prune && git pull --ff-only`
- `git merge --no-ff <thread-branch>`
- 运行 `go test ./...`
- 自动清理 thread worktree + 分支 + `git worktree prune`
- 默认不 push

## 常用参数

```bash
# 推送 master
bash .agents/skills/goyais-fixflow/scripts/merge_bugfix_to_master.sh \
  --thread-branch codex/thread39-command-status \
  --push

# 覆盖测试命令
bash .agents/skills/goyais-fixflow/scripts/merge_bugfix_to_master.sh \
  --thread-branch codex/thread39-command-status \
  --test-cmd "go test ./... && pnpm -C web test:run"

# 仅预览
bash .agents/skills/goyais-fixflow/scripts/merge_bugfix_to_master.sh \
  --thread-branch codex/thread39-command-status \
  --dry-run
```

## 手工兜底命令

```bash
cd /Users/goya/Repo/Git/Goyais
git switch master
git fetch origin --prune
git pull --ff-only

THREAD_BRANCH="codex/thread39-command-status"
git merge --no-ff "${THREAD_BRANCH}"
go test ./...

WT_PATH="$(git worktree list --porcelain | awk -v target="refs/heads/${THREAD_BRANCH}" '
  $1=="worktree" {path=$2}
  $1=="branch" && $2==target {print path}
' | head -n 1)"

if [ -n "${WT_PATH}" ] && [ "${WT_PATH}" != "/Users/goya/Repo/Git/Goyais" ]; then
  git worktree remove "${WT_PATH}"
fi

git branch -d "${THREAD_BRANCH}"
git worktree prune
```
