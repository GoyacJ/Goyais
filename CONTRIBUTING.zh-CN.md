# 参与贡献 Goyais

简体中文 | [English](CONTRIBUTING.md)

## 开始前必读

1. 阅读 `docs/12-dev-kickoff-package.md`。
2. 阅读 `docs/13-development-plan.md`。
3. 阅读 `docs/15-development-standards.md`。
4. 阅读 `docs/16-open-source-governance.md`。

## 开发流程

1. 创建分支：`codex/<task-id>-<topic>`。
2. 在 `docs/14-development-progress.md` 把目标任务改为 `IN_PROGRESS`。
3. 实现并完成测试。
4. 更新受影响文档与进度状态。
5. 按模板提交 PR。

## 设计优先规则

1. 实现必须严格遵循设计文档（适用 `docs/00-17`）。
2. 若发现文档错误或冲突，必须先修正文档（或同 PR 修正）再合并实现。
3. 未文档化行为不得作为最终契约。

## PR 描述必须包含

- 任务 ID / issue 链接
- 变更范围与动机
- 测试证据
- 兼容性影响（API/SSE/事件/错误码）
- 风险与回滚说明

## 质量门禁

至少通过：

- lint
- unit tests
- integration tests（受影响时）
- build

## 契约变更规则

若改动影响 API、事件、错误码、策略或领域模型：

1. 若冻结契约变化，先更新 `docs/12-dev-kickoff-package.md`。
2. 更新相关设计文档（`docs/00-11`、`docs/15`、`docs/16`、`docs/17`）。
3. 在 PR 里写明兼容性与迁移影响。
