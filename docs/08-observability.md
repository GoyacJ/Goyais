# Goyais 可观测性设计

> 本文档定义 Goyais 的可观测性体系，包括统一事件协议（RunEvent）、追踪标识、日志与指标规范、审计链路、回放机制以及告警策略。

最后更新：2026-02-09

---

## 1. 目标与原则

### 1.1 目标

可观测性系统需要支持：

1. 运行中实时可见（进度、状态、成本）
2. 运行后可回放（事件、上下文、决策）
3. 故障可定位（跨层 Trace 对齐）
4. 合规可审计（谁在何时做了什么）

### 1.2 设计原则

- **事件优先**：所有关键动作必须产出结构化事件
- **统一标识**：全链路使用 `trace_id` 对齐
- **低侵入**：业务模块只提交事件，不耦合展示逻辑
- **可采样**：高吞吐场景支持日志/trace 采样，审计不采样

---

## 2. RunEvent 统一协议

### 2.1 顶层结构

```go
type RunEvent struct {
    ID          uuid.UUID
    TraceID     string
    RunID       uuid.UUID
    Seq         int64
    ParentRunID *uuid.UUID
    NodeID      string
    StepID      string
    ToolCallID  string
    Type        string
    Status      string
    Timestamp   time.Time
    Source      string            // access|control|runtime|policy|data
    Payload     map[string]any
    Error       *EventError
    TenantID    uuid.UUID
    ActorID     *uuid.UUID
}
```

### 2.2 事件分类

| 类别 | 示例事件 |
|------|---------|
| Run 生命周期 | `run_started`,`run_finished`,`run_paused`,`run_resumed`,`run_cancelled`,`run_retried` |
| Workflow 节点 | `node_started`,`node_finished`,`node_failed`,`node_skipped`,`node_retry`,`sub_workflow_started`,`sub_workflow_finished` |
| Tool 调用 | `tool_called`,`tool_succeeded`,`tool_failed`,`tool_timed_out`,`tool_retry_scheduled` |
| Agent 决策 | `agent_plan`,`agent_act`,`agent_observe`,`agent_recover`,`agent_escalation`,`agent_session_started`,`agent_session_finished` |
| Intent 编排 | `intent_received`,`intent_parsed`,`intent_planned`,`intent_plan_adjusted`,`intent_confirmed`,`intent_rejected`,`intent_execution_started`,`intent_execution_finished`,`intent_execution_failed` |
| Context 变更 | `context_patch_applied`,`context_conflict`,`context_snapshot_created` |
| Policy 审核 | `policy_evaluated`,`policy_blocked`,`approval_requested`,`approval_resolved` |
| Asset 事件 | `asset_created`,`asset_derived`,`stream_slice_created` |
| Budget 事件 | `budget_warning`,`budget_exceeded` |

### 2.3 最小字段要求

任何事件至少包含：

- `trace_id`
- `run_id`
- `type`
- `timestamp`
- `tenant_id`

建议字段：

- `seq`（单 run 单调递增序号，用于乱序检测与补拉）

### 2.4 事件 Payload Schema

`Payload` 为 `map[string]any`，但必须遵循按事件类别约束的字段规范：

| 事件类别 | 事件类型 | 必需字段 | 可选字段 |
|---------|---------|---------|---------|
| Run 生命周期 | `run_started`,`run_finished`,`run_paused`,`run_resumed`,`run_cancelled`,`run_retried` | `status` | `reason`,`duration_ms`,`from_node`,`workflow_version`,`budget_used` |
| Workflow 节点 | `node_started`,`node_finished`,`node_failed`,`node_skipped`,`node_retry`,`sub_workflow_started`,`sub_workflow_finished` | `node_id`（子工作流事件还需 `child_run_id`） | `inputs_summary`,`outputs_summary`,`error`,`attempt`,`delay_ms`,`condition_result`,`child_run_id` |
| Tool 调用 | `tool_called`,`tool_succeeded`,`tool_failed`,`tool_timed_out`,`tool_retry_scheduled` | `tool_id` 或 `tool_code` | `input_summary`,`output_summary`,`diagnostics`,`error`,`attempt` |
| Agent 决策 | `agent_plan`,`agent_act`,`agent_observe`,`agent_recover`,`agent_escalation`,`agent_session_started`,`agent_session_finished` | `step_id`（session 级事件可空） | `goal`,`action`,`observation`,`recovery_action`,`approval_ticket`,`output_summary` |
| Intent 编排 | `intent_received`,`intent_parsed`,`intent_planned`,`intent_plan_adjusted`,`intent_confirmed`,`intent_rejected`,`intent_execution_started`,`intent_execution_finished`,`intent_execution_failed` | `intent_id`,`source_type`,`goal` | `action_count`,`requires_confirmation`,`linked_run_id`,`error`,`replan_reason` |
| Context 变更 | `context_patch_applied`,`context_conflict`,`context_snapshot_created` | `before_version`,`after_version`（快照事件仅需 `snapshot_version`） | `operations`,`writer`,`conflict_keys`,`strategy`,`retry_count` |
| Policy 审核 | `policy_evaluated`,`policy_blocked`,`approval_requested`,`approval_resolved` | `decision` 或 `approval_status` | `reason_code`,`violations`,`ticket_id`,`comment`,`expires_at` |
| Asset 事件 | `asset_created`,`asset_derived`,`stream_slice_created` | `asset_id` | `parent_id`,`asset_type`,`uri`,`slice_id`,`start_at`,`end_at` |
| Budget 事件 | `budget_warning`,`budget_exceeded` | `metric`,`current`,`limit` | `percentage`,`window`,`action_taken` |

规范要求：

- 字段命名统一使用 snake_case。
- 任意 `*_failed` 事件应附带 `error.code` 与 `error.message`。
- 事件消费者不可依赖未在本表声明的字段；扩展字段应置于 `payload.extra`。

---

## 3. 追踪标识体系

### 3.1 ID 语义

| 标识 | 作用 |
|------|------|
| `trace_id` | 跨 Run 全链路关联 |
| `run_id` | 单次执行容器 |
| `node_id` | Workflow 节点定位 |
| `step_id` | Agent 决策步骤定位 |
| `tool_call_id` | 单次工具调用定位 |

### 3.2 传播规则

- Access Layer 在入口生成或透传 `trace_id`
- 所有下游调用必须透传 `trace_id`
- 跨进程调用通过 HTTP Header `X-Trace-ID` 传递

### 3.3 与 OpenTelemetry 映射

| Goyais 字段 | OTel 字段 |
|------------|----------|
| `trace_id` | `trace_id` |
| `run_id` | `attributes.run_id` |
| `node_id` | `attributes.node_id` |
| `tool_call_id` | `attributes.tool_call_id` |

---

## 4. Event Store 设计

### 4.1 存储策略

- 主存储：PostgreSQL `run_events`（append-only）
- 实时推送：`LISTEN/NOTIFY` + 进程内 broker
- 查询优化：按 `run_id`、`trace_id`、`type` 建索引

### 4.2 表结构

```sql
CREATE TABLE run_events (
    id            UUID PRIMARY KEY,
    trace_id      VARCHAR(64) NOT NULL,
    run_id        UUID NOT NULL,
    seq           BIGINT NOT NULL,
    parent_run_id UUID NULL,
    node_id       VARCHAR(128) NOT NULL DEFAULT '',
    step_id       VARCHAR(128) NOT NULL DEFAULT '',
    tool_call_id  VARCHAR(128) NOT NULL DEFAULT '',
    type          VARCHAR(64) NOT NULL,
    status        VARCHAR(32) NOT NULL DEFAULT '',
    source        VARCHAR(32) NOT NULL,
    payload       JSONB NOT NULL DEFAULT '{}',
    error         JSONB NULL,
    tenant_id     UUID NOT NULL,
    actor_id      UUID NULL,
    timestamp     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_run_events_run_time ON run_events(run_id, timestamp);
CREATE INDEX idx_run_events_trace_time ON run_events(trace_id, timestamp);
CREATE INDEX idx_run_events_type_time ON run_events(type, timestamp);
CREATE UNIQUE INDEX idx_run_events_run_seq ON run_events(run_id, seq);
```

### 4.3 吞吐控制

- 批量写入：`COPY` 或批量 insert
- SSE 推送节流：默认 100ms flush
- 历史查询分页：按 `timestamp + id` 游标

---

## 5. 日志规范

### 5.1 结构化日志字段

最低字段：

- `level`
- `msg`
- `trace_id`
- `run_id`
- `component`
- `tenant_id`
- `ts`

### 5.2 日志等级

| 等级 | 场景 |
|------|------|
| `DEBUG` | 调试细节（默认关闭） |
| `INFO` | 正常流程关键里程碑 |
| `WARN` | 可恢复异常 |
| `ERROR` | 失败且需干预 |

### 5.3 脱敏规则

日志中禁止直接输出：

- Token/密钥
- 原始个人敏感信息
- 完整文件内容

---

## 6. 指标体系

### 6.1 指标分类

| 分类 | 指标 |
|------|------|
| 吞吐 | `run_started_total`,`tool_calls_total` |
| 成功率 | `run_success_ratio`,`tool_success_ratio` |
| 延迟 | `run_duration_ms`,`tool_latency_ms` |
| 资源 | `queue_depth`,`worker_busy_ratio` |
| 成本 | `cost_usd_total`,`cost_usd_per_run` |
| 质量 | `agent_escalation_ratio`,`context_conflict_ratio` |

### 6.2 推荐 SLO

| SLO | 目标 |
|-----|------|
| Workflow 成功率 | >= 99.0% |
| Tool 调用成功率 | >= 99.5% |
| 事件入库延迟 P95 | <= 300ms |
| Run 状态可见延迟 P95 | <= 1s |

---

## 7. 告警策略

### 7.1 告警规则示例

```yaml
alerts:
  - name: run-failure-spike
    expr: rate(run_finished_total{status="failed"}[5m]) > 0.1
    severity: critical

  - name: event-ingest-lag
    expr: histogram_quantile(0.95, sum(rate(event_ingest_latency_bucket[5m])) by (le)) > 0.5
    severity: warning

  - name: tool-timeout-spike
    expr: rate(tool_failed_total{reason="timeout"}[5m]) > 0.05
    severity: warning
```

### 7.2 升级路径

- `warning`：通知当班群
- `critical`：通知 + 电话升级 + 自动创建 incident

---

## 8. 审计日志

### 8.1 审计范围

必须审计的行为：

- 权限变更
- 高风险工具执行
- 资产删除/公开
- 策略审批操作
- API token 管理

### 8.2 审计字段

| 字段 | 说明 |
|------|------|
| `action` | 操作类型 |
| `actor_id` | 执行者 |
| `target_type` / `target_id` | 操作对象 |
| `before` / `after` | 变更快照 |
| `trace_id` | 链路关联 |
| `ip` / `user_agent` | 来源信息 |

审计日志必须 append-only，禁止更新删除。

---

## 9. 回放与诊断

### 9.1 回放输入

回放最小输入：

- `run_id`
- `run_events` 序列
- `context_snapshots` + `context_patches`

### 9.2 回放模式

| 模式 | 说明 |
|------|------|
| `event-only` | 仅重建事件时间线 |
| `stateful` | 重建每步上下文状态 |
| `deterministic` | 对幂等工具进行可重复模拟 |

### 9.3 失败复盘建议

1. 先看 `run_finished` 错误原因
2. 定位最后一个 `tool_failed` / `context_conflict`
3. 对比前后 context patch
4. 检查 policy/budget 阻断记录

---

## 10. SSE 与观测接口

观测数据对外主要通过：

- `GET /api/runs/{id}/events`（历史查询）
- `GET /api/runs/{id}/events/stream`（SSE）
- `GET /api/traces/{trace_id}/events/stream`（跨 run 实时事件流）

SSE 事件格式：

```text
event: run_event
id: evt_123
data: {"type":"node_started","seq":41,"run_id":"...","timestamp":"...","payload":{}}
```

详细 API 见 `10-api-design.md`。

---

## 11. 保留与归档策略

### 11.1 建议保留期

| 数据 | 保留期 |
|------|-------|
| run_events | 180 天 |
| trace spans | 30 天 |
| metrics 明细 | 15 天 |
| metrics 聚合 | 365 天 |
| audit logs | 365 天起（按合规） |

### 11.2 冷热分层

- 热数据：最近 7-30 天，支持高频查询
- 冷数据：归档到低成本存储，仅审计与复盘使用

---

## 12. 与其他模块关系

| 模块 | 关系 |
|------|------|
| `06-workflow-engine.md` | Workflow 事件是主要来源之一 |
| `07-agent-runtime.md` | Agent 决策事件进入同一协议 |
| `09-security-policy.md` | 审批与阻断行为写审计/事件 |
| `10-api-design.md` | 事件查询与 SSE 接口规范 |
| `11-frontend-design.md` | 前端时间线、监控面板依赖该协议 |
