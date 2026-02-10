# 版本与回滚策略（GitHub Flow）

## 版本策略

- 主干：`master`
- 发布基线：`master` 上稳定提交打 SemVer tag（如 `v0.1.0`）
- 补丁发布：在 `master` 继续提交并打 `v0.1.x`

## 回滚策略

1. 优先 `git revert <commit>` 生成可审计逆向提交。
2. 若是多提交问题，使用 `git revert <oldest>^..<newest>`。
3. 回滚后补充 PR，说明影响范围、根因与后续修复。
4. 禁止在共享分支重写历史（避免 `push --force` 破坏协作可追踪性）。
5. 任何分支指针移动前必须先做 `backup tag + backup branch`。
6. 若必须覆盖远端历史，只允许 `push --force-with-lease`，且先输出待推送提交列表与风险说明。

## 发布记录建议

- 记录 tag、变更摘要、风险说明、回滚入口。
- 记录对应验收证据（至少包括关键接口/脚本检查结果）。
