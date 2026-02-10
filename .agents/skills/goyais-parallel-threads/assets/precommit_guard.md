# Pre-commit Guard（提交前防呆）

每次提交前必须执行以下检查：

```bash
git diff --cached --name-only
git diff --cached --name-only | rg '^(data/objects/|.*\.db$|build/|web/dist/|web/node_modules/|\.agents/)' && exit 1 || true
```

## 规则说明

- 命中 `data/objects/**`：禁止提交（本地对象存储数据）。
- 命中 `*.db`：禁止提交（本地数据库）。
- 命中 `build/`、`web/dist/`、`web/node_modules/`：禁止提交（构建与依赖产物）。
- 命中 `.agents/**`：默认禁止进入业务改动提交（仅规则/skill专项任务可单独提交）。

## 建议补充检查

```bash
git status --porcelain=v1 -uall
git ls-files --others --exclude-standard
```
