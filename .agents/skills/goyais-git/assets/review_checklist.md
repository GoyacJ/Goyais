# Review Checklist

## P0 阻断项

- [ ] 是否违反 `AGENTS.md` 冻结决策
- [ ] 是否破坏 `/api/v1`、错误结构、鉴权链路
- [ ] 是否遗漏契约文档同步
- [ ] 是否缺少可执行验收证据
- [ ] 并行 thread 是否遵守“一线程一 worktree”，且未在同一 worktree 切换多个 thread 分支
- [ ] 提交前是否执行禁止路径检查并拦截 `data/objects/**`、`*.db`、`build/`、`web/dist/`、`web/node_modules/`、`.agents/**`

## 功能与行为

- [ ] 行为与需求一致，边界条件明确
- [ ] 异常路径有可定位错误（含 `messageKey`）
- [ ] 兼容性与回滚路径明确
- [ ] 涉及分支指针移动时是否有 `backup tag + backup branch`

## 可维护性

- [ ] 命名与结构可读
- [ ] 说明文档与模板同步更新
- [ ] 无引入与冻结约束冲突的新规则
- [ ] 若需覆盖远端历史，是否仅使用 `--force-with-lease` 且附提交清单与风险说明
