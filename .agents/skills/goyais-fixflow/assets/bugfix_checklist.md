# goyais-fixflow Bugfix Checklist

## A. 启动前

- [ ] 已读取 `AGENTS.md` 与并行 thread 约束。
- [ ] 已通过脚本创建独立 worktree 与 `codex/<thread-id>-<topic>` 分支。
- [ ] 已确认修复范围（in scope / out of scope）。

## B. 修复执行

- [ ] 给出可复现步骤（输入、预期、实际）。
- [ ] 完成根因定位，写明问题链路。
- [ ] 实施最小修复，避免顺带重构。
- [ ] 补齐必要测试（单测/集成/手测至少一项）。

## C. 验证证据

- [ ] 执行并记录关键命令及结果。
- [ ] 校验主要路径 + 边界路径 + 回归路径。
- [ ] 记录风险点与未覆盖项。

## D. 合并门禁

- [ ] 输出修复摘要，等待用户明确确认“可合并到 master”。
- [ ] 在确认前，不执行任何 master 合并动作。

## E. 合并后

- [ ] 在 master 工作树执行 no-ff merge。
- [ ] 执行默认测试 `go test ./...`（或用户指定命令）。
- [ ] 回收 thread worktree / 分支并 `git worktree prune`。
- [ ] 输出最终交付摘要与回滚说明。
