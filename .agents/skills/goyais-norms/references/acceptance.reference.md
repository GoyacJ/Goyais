# docs/acceptance.md 约束引用（链接+关键摘录）

- source_path: `docs/acceptance.md`
- authority_scope: v0.1 验收门槛与通过判定

## key_constraints

- P0 条目重点：最小化运行、single binary、Command-first 一致性、Visibility/ACL。
- Single Binary 验收：`make build` 后即使 `web/dist` 不存在，`/`、`/canvas`、`/api/v1/healthz` 仍可用。
- 静态与特殊路径：`/api/v1/*` 不被 fallback 覆盖；无占位时 `favicon/robots` 返回 404。
- 响应头与类型：`/` 与 `/canvas` 的 `Cache-Control` 精确为 `no-store`；`/assets/*.js` 为 JS 类型。
- 错误与本地化：后端错误结构包含 `messageKey`，前端需可映射。

## sync_note

仅保留验收关键点摘录，执行与判定细节以 `docs/acceptance.md` 原文为准。
