# Conventional Commits 示例

## 推荐格式

`type(scope): summary`

常用 `type`：`feat`、`fix`、`refactor`、`docs`、`test`、`chore`、`build`。

## 正例

- `feat(command): add commandRef to domain write response`
- `fix(webstatic): enforce no-store for index fallback`
- `docs(acceptance): align single binary verification steps`
- `refactor(authz): split resource gate and tool gate checks`

## 反例

- `update code`
- `fix bug`
- `WIP`
- `misc changes`

## 建议

- summary 使用祈使句，聚焦“做了什么”。
- 单个 commit 只表达一个主意图，避免巨型提交。
- 涉及契约变更时在 commit body 中写清同步文档。
