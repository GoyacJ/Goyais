# ADR-0006: Trace / Observability / Diagnostics 统一方案

## Status
Accepted

## Context
线上问题定位需要跨 Runtime、Sync、SSE、数据库事件快速串联，且诊断信息必须脱敏可导出。

## Decision
- Trace 统一：
  - 所有 HTTP 请求支持/透传 `X-Trace-Id`，服务端缺失时生成。
  - 所有响应回传 `X-Trace-Id`。
  - event envelope 顶层 `trace_id` 必填，payload 同步包含 `trace_id`。
  - Runtime 持久化 `runs.trace_id` / `events.trace_id` / `audit_logs.trace_id`，并建索引。
- 结构化日志：
  - Runtime 与 Sync 全部输出 JSON 日志。
  - 最小字段：`level`, `ts`, `trace_id`, `run_id?`, `event_id?`, `tool_name?`, `duration_ms?`, `outcome`.
  - 请求日志包含：`route/path`, `status`, `latency`.
- 最小指标：
  - Runtime: `runs_total`, `runs_failed_total`, `tool_calls_total`, `confirmations_pending`, `sync_push_total`, `sync_pull_total`
  - Sync: `events_total`, `push_requests_total`, `pull_requests_total`, `auth_fail_total`
- 诊断导出：
  - Runtime 提供 `GET /v1/diagnostics/run/{run_id}`。
  - 默认要求 `X-Runtime-Token`。
  - 返回 run 元信息、事件样本、审计摘要、关键错误。
  - 所有输出经过递归脱敏（Authorization/token/apiKey/secret_ref/path）。

## Consequences
- 任意错误都可通过 `trace_id` 关联请求日志、事件流与审计数据。
- 运维侧可用 `/v1/metrics` 做 P0 统计，不依赖外部监控系统。
- 诊断导出可直接用于支持流程，降低敏感信息泄露风险。
