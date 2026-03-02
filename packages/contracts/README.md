# @goyais/contracts

本目录存放 Goyais v0.4.0 的 API 与事件契约定义（单一权威源）。

- `openapi.yaml`：Hub/Worker 对外与内部接口契约（单一权威源）。
- 覆盖范围：认证授权、项目/Conversation/Execution、资源共享、管理后台、Worker 内部执行与事件接口。

## 与 shared-core 类型同步

- 生成类型：`pnpm contracts:generate`
- 校验是否与契约一致：`pnpm contracts:check`

上述命令会基于 `openapi.yaml` 更新或校验 `packages/shared-core/src/generated/openapi.ts`。
