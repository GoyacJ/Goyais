# goyais-fixflow Worktree Quickstart

## 默认入口（推荐）

```bash
cd /Users/goya/Repo/Git/Goyais
bash .agents/skills/goyais-fixflow/scripts/create_bugfix_worktree.sh --topic ui-status-wrap
```

输出示例（重点字段）：

- `thread_id=thread20260210-231500`
- `branch=codex/thread20260210-231500-ui-status-wrap`
- `worktree_path=/Users/goya/Repo/Git/Goyais-wt-thread20260210-231500`

## 常用参数

```bash
# 指定 thread-id
bash .agents/skills/goyais-fixflow/scripts/create_bugfix_worktree.sh \
  --topic command-detail-status \
  --thread-id thread39

# 仅预览，不执行
bash .agents/skills/goyais-fixflow/scripts/create_bugfix_worktree.sh \
  --topic command-detail-status \
  --dry-run

# 跳过 master 同步（仅紧急场景）
bash .agents/skills/goyais-fixflow/scripts/create_bugfix_worktree.sh \
  --topic command-detail-status \
  --skip-sync
```

## 手工命令兜底

```bash
cd /Users/goya/Repo/Git/Goyais
git switch master
git fetch origin --prune
git pull --ff-only

THREAD_ID="thread$(date +%Y%m%d-%H%M%S)"
TOPIC="command-detail-status"
BRANCH="codex/${THREAD_ID}-${TOPIC}"
WT="/Users/goya/Repo/Git/Goyais-wt-${THREAD_ID}"

git worktree add "${WT}" -b "${BRANCH}" master
```

## 注意事项

- 一个 bugfix thread 对应一个独立 worktree，不复用旧路径。
- 禁止在主仓库工作树直接修复 bug。
- `--topic` 会自动标准化为 `lower-kebab-case`。
