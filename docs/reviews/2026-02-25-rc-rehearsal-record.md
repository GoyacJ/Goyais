# Goyais RC 发布演练记录（2026-02-25）

- 演练时间：2026-02-25 11:34 CST
- 代码基线：`develop` 当前工作区快照
- 演练类型：发布前 Gate 预演（自动化项 + 关键回归项）
- 记录人：Codex（首席架构师/项目经理代理）

## 1. 演练结论

- 结论：`GO (技术项)`
- 说明：代码级质量门禁与关键安全/一致性回归均通过。
- 保留项：需由发布责任人完成最终人工签字（Tech Lead / Release Owner / PM）。

## 2. Checklist 执行结果

| 类别 | 检查项 | 命令/证据 | 结果 |
|---|---|---|---|
| 代码质量 | Hub test + vet | `cd services/hub && go test ./... && go vet ./...` | Pass |
| 代码质量 | Worker lint + test | `cd services/worker && uv run ruff check . && uv run pytest -q` | Pass（41 passed） |
| 代码质量 | Desktop lint + test + coverage gate | `cd apps/desktop && pnpm lint && pnpm test && pnpm coverage:gate` | Pass（108 passed；coverage gate OK） |
| 安全隔离 | Hub workspace 授权边界 | `go test ./internal/httpapi -run 'TestRemoteSessionWorkspaceListIsScoped|TestRemoteSessionCannotManageWorkspaceCatalog|TestLocalAnonymousListEndpointsAreScopedToLocalWorkspace|TestWorkspaceStatusHandlerRemoteRequiresAuth'` | Pass |
| 安全隔离 | Worker 内部 token/命令执行防护 | `uv run pytest -q tests/test_internal_tokens.py tests/test_command_guard.py tests/test_tool_runtime_run_command.py` | Pass（9 passed） |
| 一致性 | Desktop SSE/合并/重同步关键路径 | `pnpm vitest run src/modules/conversation/tests/conversation-stream.spec.ts src/modules/conversation/tests/conversation-hydration.spec.ts src/modules/conversation/tests/execution-merge.spec.ts` | Pass（8 passed） |

## 3. 覆盖率门禁快照

- 命令：`cd apps/desktop && pnpm coverage:gate`
- 结果：`[check-coverage-thresholds] OK`
- 当前 coverage 输出（All files）：
  1. Statements: `78.35%`
  2. Functions: `60.49%`
  3. Lines: `78.35%`
  4. Branches: `71.83%`

## 4. 与 release checklist 的映射

- 参考文档：[release-checklist.md](/Users/goya/Repo/Git/Goyais/docs/release-checklist.md)
- 已完成：第 2 节（代码质量门禁）与第 3/5 节关键自动化验证。
- 待人工确认：
  1. 生产环境变量与不安全开关核查（第 4 节）。
  2. 发布值班与回滚负责人签字（第 6/7/8 节）。

## 5. 建议发布动作

1. 由 Release Owner 发起一次正式 RC Checklist 打勾流程。
2. 绑定本记录到发布单，补齐人工签字后进入最终 `GO/NO-GO` 决策。
