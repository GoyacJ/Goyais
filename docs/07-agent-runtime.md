# Goyais Agent 运行时设计

> 本文档定义 Goyais Agent Runtime 的执行状态机、上下文模型、工具调用链、恢复机制与会话治理规则，覆盖从一次用户输入到会话结束的完整生命周期。

最后更新：2026-02-09

---

## 1. 目标与定位

Agent Runtime 位于 Control Layer（`internal/control/agent/`），是平台中的“决策执行器”，负责：

- 解释用户目标并生成执行计划
- 选择并调用合适 Tool/Algorithm
- 对执行结果进行观察、反思与下一步决策
- 在失败或冲突时进行恢复或升级（escalation）

Agent Runtime 与 Workflow Engine 的关系：

- Workflow 更适合静态 DAG 场景
- Agent 更适合开放式任务与动态规划
- Workflow 节点可以嵌入 Agent Session（`node_type=agent`）

---

## 2. 核心实体

### 2.1 AgentSession（Run 形态）

Agent 会话统一建模为 `Run(type=agent_session)`。

关键字段：

| 字段 | 说明 |
|------|------|
| `run_id` | 会话唯一标识 |
| `trace_id` | 全链路追踪 ID |
| `status` | pending/running/paused/succeeded/failed/cancelled |
| `objective` | 用户目标或上游节点目标 |
| `max_steps` | 最大决策步数 |
| `budget` | 成本与 token 上限 |

### 2.2 AgentProfile

`AgentProfile` 定义会话默认行为：

```go
type AgentProfile struct {
    ID               string
    Name             string
    SystemPrompt     string
    ToolAllowlist    []string
    ToolDenylist     []string
    MaxSteps         int
    MaxToolCalls     int
    MaxContextTokens int
    RecoveryPolicy   RecoveryPolicy
    EscalationPolicy EscalationPolicy
}
```

### 2.3 AgentState

```go
type AgentState struct {
    StepIndex      int
    LastObservation map[string]any
    WorkingMemory   map[string]any
    Plan            []PlanItem
    PendingAction   *Action
    ErrorCount      int
}
```

---

## 3. Run Loop 状态机

### 3.1 主状态

```text
Init -> Plan -> Act -> Observe -> (Plan|Recover|Finish)
                              \-> Escalate -> Finish
```

### 3.2 完整状态转移表

| 当前状态 | 事件 | 下一状态 | 动作 |
|---------|------|---------|------|
| `Init` | `session_started` | `Plan` | 初始化 AgentState、写入 `agent_session_started` |
| `Plan` | `plan_succeeded` | `Act` | 产出计划与候选工具，写入 `agent_plan` |
| `Plan` | `plan_failed` 且 `retry_count < max_plan_retry` | `Recover` | 记录计划失败原因，进入恢复策略 |
| `Plan` | `plan_failed` 且 `retry_count >= max_plan_retry` | `Escalate` | 进入人工升级 |
| `Act` | `action_succeeded` | `Observe` | 写入工具输出与诊断信息 |
| `Act` | `action_failed` | `Recover` | 分类错误并决定重试/替代 |
| `Observe` | `goal_reached` | `Finish` | 汇总最终输出 |
| `Observe` | `need_more_steps` | `Plan` | 基于观察结果重规划 |
| `Observe` | `policy_blocked` | `Escalate` | 请求审批或人工干预 |
| `Recover` | `recovery_succeeded` | `Act` | 应用恢复动作（重试或替代工具） |
| `Recover` | `recovery_needs_replan` | `Plan` | 放弃当前动作，重建计划 |
| `Recover` | `recovery_failed` 且 `recover_count >= max_recover_retry` | `Escalate` | 达到恢复上限后升级 |
| `Escalate` | `approval_approved` | `Plan` | 恢复会话并重新规划下一步 |
| `Escalate` | `approval_rejected` / `approval_expired` | `Finish` | 失败结束并输出原因 |
| `Finish` | `user_followup_message` | `Plan` | 多轮对话重入：沿用会话上下文继续规划 |
| `Finish` | `session_closed` | `Finish` | 终态保持 |

### 3.3 阶段职责

| 阶段 | 输入 | 输出 |
|------|------|------|
| `Plan` | objective + context + memory | 计划步骤、候选工具 |
| `Act` | 当前计划项 | 工具调用结果或子任务结果 |
| `Observe` | act 输出 | 目标进度评估、置信度、下一步 |
| `Recover` | 错误上下文 | 重试/降级/跳过/终止决策 |
| `Finish` | 最终状态 | 会话输出（text/structured/asset） |
| `Escalate` | 高风险或低置信度 | 人工审批或转交 |

### 3.4 终止条件

满足任一条件结束会话：

1. 目标达成且输出通过校验
2. 超过 `max_steps`
3. 预算耗尽
4. 连续恢复失败超过阈值
5. 用户取消

---

## 4. 上下文与记忆模型

### 4.1 分层记忆

| 层级 | 存储位置 | 生命周期 | 用途 |
|------|---------|---------|------|
| `scratchpad` | 内存 | 单步 | 暂存推理草稿 |
| `working_memory` | ContextState | 会话级 | 多步任务状态 |
| `episodic_memory` | 可选持久化 | 跨会话 | 历史经验摘要 |

### 4.2 ContextState 命名空间

Agent 运行时使用如下命名空间：

- `agent.goal`
- `agent.plan`
- `agent.steps.<index>.action`
- `agent.steps.<index>.observation`
- `agent.memory.*`

所有写入通过 CAS Patch 提交，避免并发污染。

### 4.3 长上下文压缩

当 token 超阈值时执行压缩：

1. 保留系统约束与最近 N 步
2. 历史步骤摘要化
3. 大对象转 AssetRef（不内联）

---

## 5. 工具调用链

### 5.1 工具选择

Plan 阶段候选工具来自：

- `AgentProfile.ToolAllowlist`
- 当前租户可见且 `active` 的工具
- 策略引擎允许的风险等级范围

### 5.1.1 工具选择与排序算法

当存在多个候选 Tool 时，运行时按“规则预过滤 + 评分排序 + 回退重选”执行：

1. 预过滤：
   - 仅保留 `active` 状态工具。
   - 应用 `allowlist/denylist`、租户可见性、Policy 预算与风险限制。
2. 评分：
   - `intent_match`（语义匹配，0-1，来自 LLM/tool descriptor 匹配）
   - `success_rate_1h`（近 1 小时成功率）
   - `latency_score`（延迟归一化）
   - `cost_score`（成本归一化）
   - `policy_risk_penalty`（高风险惩罚项）
3. 综合分：
   - `final = 0.45*intent_match + 0.25*success_rate_1h + 0.15*latency_score + 0.10*cost_score - 0.15*policy_risk_penalty`
4. 同分裁决：
   - 优先 `required_permissions` 更少者；
   - 其次 `tool_code` 字典序（稳定可复现）。
5. 回退策略：
   - 首选工具被 Policy 拒绝时，标记 `policy_rejected=true` 并从候选集中移除，选择下一名；
   - 候选耗尽则进入 `Recover` 或 `Escalate`。

对 Agent 展示的 Tool 描述格式：

```json
{
  "tool_code": "face.detect",
  "purpose": "检测图像中的人脸框",
  "input_schema_summary": ["image_asset_id:string", "threshold:number?"],
  "side_effects": ["read_asset"],
  "risk_level": "medium",
  "estimated_cost_usd": 0.003,
  "estimated_latency_ms": 420
}
```

### 5.2 调用流程

```text
Plan 产出 Action
 -> 参数组装（含上下文映射）
 -> Policy 校验（权限/预算/网络/数据）
 -> Tool Dispatcher 执行
 -> 输出写回 AgentState + ContextState
 -> 产出 tool_* 与 agent_observe 事件
```

### 5.3 工具失败处理

根据错误类型进入 `Recover`：

- 可重试错误：指数退避重试
- 不可重试错误：尝试替代工具
- 策略拒绝：触发 escalation 或改写计划

### 5.4 平台动作执行（全 AI 操作）

为支持“对话/语音驱动全平台行为”，Agent 在 Act 阶段可产出 `IntentAction`（非仅 Tool 调用）：

- `identity.user.create` / `identity.role.create` / `identity.role.bind`
- `identity.role.permissions.update`
- `settings.update`
- `asset.upload` / `workflow.run`

执行约束：

1. 先做结构化动作校验（参数完整性、资源存在性）。
2. 再做 Policy + RBAC + 风险分级校验。
3. `need_confirmation=true` 的动作必须等待用户确认。
4. 高风险动作确认后仍需审批时，进入 `Escalate` 状态。

---

## 6. 错误分类与恢复策略

### 6.1 错误分类

| 类别 | 示例 | 默认策略 |
|------|------|---------|
| `transient` | 网络抖动、429 | 重试 |
| `tool_logic` | schema 不匹配 | 重新规划 |
| `policy_blocked` | 无权限、超预算 | 升级或终止 |
| `context_conflict` | CAS 冲突 | 重新拉取并重放 |
| `fatal` | 关键依赖不可用 | 失败结束 |

### 6.2 恢复矩阵

```text
transient      -> retry(max=3) -> fallback tool -> fail
tool_logic     -> re-plan       -> alternate params -> fail
context_conflict -> reload snapshot -> reapply patch -> fail
policy_blocked -> escalate -> wait approval -> continue/fail
```

### 6.3 Human-in-the-loop

`risk_level >= high` 或低置信关键决策时，运行时触发 `agent_escalation`：

- 进入 `paused`
- 等待人工批准/驳回/改写参数
- 审批完成后恢复运行

### 6.4 升级处理流程

升级审批流规范：

1. 通知渠道：
   - 优先 `WebSocket/SSE` 推送到前端审批面板；
   - 可选 `Webhook`（企业 IM / 工单系统）异步通知。
2. 超时策略：
   - 默认 `high=30m`，`critical=15m`；
   - 到期未处理自动转 `approval_expired`，按拒绝处理。
3. 审批恢复行为：
   - `approve`：会话从 `Escalate` 回到 `Plan`，重新生成下一步动作（不直接复用旧 Action）。
   - `reject`：会话进入 `Finish`，输出 `blocked_by_approval` 原因。
   - `rewrite`（改写参数）：写入新约束后回到 `Plan`。
4. 审批审计：
   - 必须产出 `approval_requested` / `approval_resolved` RunEvent；
   - 同时写入审计日志，记录 `approver_id`、`comment`、`decision`。

---

## 7. 并发、暂停与恢复

### 7.1 并发策略

单会话默认串行决策；允许有限并发调用（可选）：

- 同步要求低的检索类工具可并发
- 并发上限由 `max_parallel_tool_calls` 控制
- 并发结果进入统一 Observe 汇总

### 7.2 暂停与恢复

支持以下暂停来源：

- 人工审批
- 外部依赖不可用
- 用户主动暂停

恢复流程：

1. 读取最新 `ContextSnapshot`
2. 回放未落盘事件
3. 从上一个稳定 step 继续

### 7.3 取消语义

取消时执行两类清理：

- 尝试取消正在执行的 tool call
- 标记 run 为 `cancelled` 并关闭后续调度

---

## 8. 事件模型

Agent Runtime 事件：

| 事件 | 说明 |
|------|------|
| `agent_session_started` | 会话开始 |
| `agent_plan` | 产生或更新计划 |
| `agent_act` | 发起动作 |
| `agent_observe` | 观察到工具输出 |
| `agent_recover` | 启动恢复策略 |
| `agent_escalation` | 升级人工处理 |
| `agent_session_finished` | 会话结束 |

每个事件必须包含 `trace_id`、`run_id`、`step_id`。

---

## 9. 数据模型建议

### 9.1 agent_profiles

```sql
CREATE TABLE agent_profiles (
    id                UUID PRIMARY KEY,
    tenant_id         UUID NOT NULL,
    name              VARCHAR(128) NOT NULL,
    system_prompt     TEXT NOT NULL,
    tool_allowlist    TEXT[] NOT NULL DEFAULT '{}',
    tool_denylist     TEXT[] NOT NULL DEFAULT '{}',
    max_steps         INTEGER NOT NULL DEFAULT 20,
    max_tool_calls    INTEGER NOT NULL DEFAULT 50,
    max_context_tokens INTEGER NOT NULL DEFAULT 32000,
    recovery_policy   JSONB NOT NULL DEFAULT '{}',
    escalation_policy JSONB NOT NULL DEFAULT '{}',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### 9.2 agent_step_logs（可选）

```sql
CREATE TABLE agent_step_logs (
    id            BIGSERIAL PRIMARY KEY,
    run_id        UUID NOT NULL,
    step_index    INTEGER NOT NULL,
    phase         VARCHAR(32) NOT NULL,
    input_payload JSONB NOT NULL DEFAULT '{}',
    output_payload JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_agent_step_logs_run_step ON agent_step_logs(run_id, step_index);
```

---

## 10. 接口契约

```go
type AgentRuntime interface {
    Start(ctx context.Context, req StartSessionRequest) (Run, error)
    Resume(ctx context.Context, runID uuid.UUID) error
    Pause(ctx context.Context, runID uuid.UUID, reason string) error
    Cancel(ctx context.Context, runID uuid.UUID) error
    GetState(ctx context.Context, runID uuid.UUID) (AgentState, error)
}
```

`StartSessionRequest` 建议字段：

- `objective`
- `profile_id`
- `initial_context`
- `budget`
- `attachments`（AssetRef 列表）

---

## 11. 性能与治理指标

### 11.1 核心指标

- 会话成功率
- 平均步数（steps/session）
- 工具调用成功率
- 平均恢复次数
- 人工升级比例
- 每会话平均成本

### 11.2 告警建议

| 告警 | 阈值 |
|------|------|
| `agent_failure_rate` | 5 分钟内 > 10% |
| `agent_escalation_rate` | 15 分钟内 > 20% |
| `avg_step_latency` | P95 > 8s |
| `budget_block_rate` | 15 分钟内 > 15% |

---

## 12. 与其他模块关系

| 模块 | 关系 |
|------|------|
| `04-tool-system.md` | Agent 在 Act 阶段调用 Tool |
| `05-algorithm-library.md` | Agent 可通过算法引用选择实现 |
| `06-workflow-engine.md` | Workflow 可嵌入 Agent 节点 |
| `08-observability.md` | Agent 全链路事件进入统一 RunEvent |
| `09-security-policy.md` | 高风险操作需审批与策略校验 |
| `10-api-design.md` | Agent Session API 与事件流定义 |
