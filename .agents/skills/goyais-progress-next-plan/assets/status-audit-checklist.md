# Status Audit Checklist

用于计划前的证据扫描，不做实现写入。

## A. Baseline Snapshot

- [ ] 记录 `repo_root`、`branch`、`HEAD`。
- [ ] 记录 `git status --short --branch`。
- [ ] 记录 `git worktree list`。
- [ ] 标注 `clean/dirty` 状态。

## B. Acceptance Progress

- [ ] 统计 `docs/acceptance.md` 总项数。
- [ ] 统计完成项与未完成项。
- [ ] 输出完成比例。
- [ ] 列出未完成条目（带行号或证据）。

## C. Implementation Scan

- [ ] 扫描路由挂载：`router.go`。
- [ ] 扫描 handler/service/repo 存在性。
- [ ] 扫描 migration 覆盖痕迹。
- [ ] 扫描 integration/regression 测试覆盖痕迹。
- [ ] 标记域状态：`implemented|partial|placeholder|unknown`。

## D. Contract Drift

- [ ] 对比 `openapi.yaml` 与 `router.go` 路由路径。
- [ ] 对比关键路径与 `router_integration_test.go`。
- [ ] 输出 drift 结论（confirmed/partial/unknown）。

## E. Plan Readiness

- [ ] 每个结论都包含绝对路径证据。
- [ ] 每个结论都包含可复现命令。
- [ ] 深度模式未跑回归时已写明风险。
