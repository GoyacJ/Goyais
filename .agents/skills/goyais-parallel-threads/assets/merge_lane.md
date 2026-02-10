# Merge Lane（仅 master 集成工作树）

## 原则

- 仅在主仓库工作树（`master`）执行 thread 合并。
- thread 工作树只做开发，不做主线集成。

## 集成流程

```bash
cd /Users/goya/Repo/Git/Goyais
git switch master
git fetch origin --prune
git pull --ff-only

git merge --no-ff codex/<thread-id>-<topic>
go test ./...
make build
```

## 分支指针安全

发生分支修复或指针移动前：

```bash
TS=$(date +%Y%m%d-%H%M%S)
git tag -a "backup/<branch>-${TS}" <old_sha> -m "backup before moving branch pointer"
git branch "backup/<branch>-${TS}" <old_sha>
```

## 推送策略

- 默认不推送，先输出待推送 commit 列表。
- 若必须覆盖远端历史，仅允许：

```bash
git push --force-with-lease origin <branch>
```
