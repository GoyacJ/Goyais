# ADR-0005: 统一错误模型（GoyaisError）与协议 v2

## Status
Accepted

## Context
历史版本中 Runtime、Sync、SSE、tool_result 的错误格式不一致，导致客户端解析分支多、排障慢、trace 关联困难。

## Decision
- 协议升级到 `2.0.0`（breaking）：
  - 不再兼容旧 `1.0.0` 错误结构。
- 全链路统一错误对象 `GoyaisError`：
  - 必填：`code`, `message`, `trace_id`, `retryable`
  - 可选：`details`, `cause`, `ts`
- HTTP 失败响应统一为：
  - `{ "error": GoyaisError }`
- SSE `error` 事件 payload 统一为：
  - `{ "error": GoyaisError, "trace_id": string }`
- `tool_result` payload 统一为：
  - 成功：`{ call_id, ok: true, output, trace_id }`
  - 失败：`{ call_id, ok: false, error: GoyaisError, trace_id }`
- 错误码稳定，不随文案变化（例如：`E_SYNC_AUTH`, `E_PATH_ESCAPE`, `E_TOOL_DENIED`, `E_SCHEMA_INVALID`, `E_INTERNAL`）。

## Consequences
- 客户端错误解析逻辑大幅简化。
- 旧客户端（依赖 `detail`/`payload.message`）需要一次性升级。
- 所有错误都可直接关联 `trace_id`，利于排障和告警聚合。
