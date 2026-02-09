# Goyais 工具系统设计

> 本文档定义 Goyais 工具系统（Tool System）的契约模型、注册管理、执行模式、运行时分发与治理策略。该文档是 `Tool` 一等对象的标准设计。

最后更新：2026-02-09

---

## 1. 目标与边界

### 1.1 目标

工具系统的核心目标：

1. 统一承载所有可执行能力（HTTP/CLI/MCP/模型调用/本地函数）
2. 用声明式 `ToolSpec` 替代散落在代码中的执行配置
3. 在执行前引入可治理能力（权限、风险、预算、数据访问）

### 1.2 非目标

工具系统不负责：

- 算法意图建模（见 `05-algorithm-library.md`）
- DAG 编排（见 `06-workflow-engine.md`）
- Agent 决策（见 `07-agent-runtime.md`）

---

## 2. 统一 Tool 抽象

### 2.1 ToolSpec 结构

`ToolSpec` 为唯一执行契约，字段定义以 `02-domain-model.md` 为权威，补充约束如下：

- `id`：UUID 主键（数据库内部标识）
- `code`：全局唯一且稳定的人类可读标识（建议 `namespace.tool_name`）
- `version`：语义化版本，建议 `MAJOR.MINOR.PATCH`
- `input_schema`：必须可被 JSON Schema 验证器执行
- `output_schema`：必须覆盖成功返回结构
- `side_effects`：必须完整声明外部可观察副作用
- `risk_level`：默认 `low`，高风险必须审批

### 2.2 生命周期

| 状态 | 说明 |
|------|------|
| `active` | 可被工作流/Agent 使用 |
| `disabled` | 禁用，不接受新调用 |
| `deprecated` | 弃用，仅兼容旧流程 |

约束：

- `deprecated` 仍可被白名单 workflow 调用
- 已被 `AlgorithmBinding` 引用的版本不可直接删除

---

## 3. Tool Registry

### 3.1 核心职责

Tool Registry 提供：

- 注册与版本发布
- 查询与过滤（按分类、风险、执行模式、租户）
- 启停控制（`active/disabled`）
- 兼容性检查（Schema 变更、依赖项变更）

### 3.2 接口定义

```go
type ToolRegistry interface {
    Register(ctx context.Context, spec *ToolSpec) error
    Get(ctx context.Context, id uuid.UUID) (*ToolSpec, error)
    GetByCode(ctx context.Context, code string) (*ToolSpec, error)
    List(ctx context.Context, filter ToolFilter) ([]*ToolSpec, int64, error)
    Update(ctx context.Context, spec *ToolSpec) error
    Deprecate(ctx context.Context, id uuid.UUID) error
    Disable(ctx context.Context, id uuid.UUID) error
}
```

### 3.3 版本解析策略

- 指定版本：精确命中 `code + version`
- 未指定版本：取租户可见范围内最新 `active` 版本
- 指定 `^2.0`：按 semver 范围匹配（可选能力）

---

## 4. 执行模式与运行时分发

### 4.1 支持模式

| execution_mode | 说明 | 典型场景 |
|---------------|------|---------|
| `in_process` | 进程内直接调用 | 轻量纯函数、低延迟转换 |
| `subprocess` | 本地子进程调用 | FFmpeg、OCR CLI、脚本工具 |
| `container` | 容器隔离执行 | 高风险依赖、复杂运行环境 |
| `remote` | 远程 HTTP/gRPC 调用 | 外部服务、云 API |

### 4.2 Dispatcher

```go
type Dispatcher interface {
    Execute(ctx context.Context, spec ToolSpec, env ExecutionEnvelope) (ToolResult, error)
}

func (d *DefaultDispatcher) Execute(ctx context.Context, spec ToolSpec, env ExecutionEnvelope) (ToolResult, error) {
    switch spec.ExecutionMode {
    case "in_process":
        return d.inproc.Execute(ctx, spec, env)
    case "subprocess":
        return d.subproc.Execute(ctx, spec, env)
    case "container":
        return d.container.Execute(ctx, spec, env)
    case "remote":
        return d.remote.Execute(ctx, spec, env)
    default:
        return ToolResult{}, ErrUnsupportedMode
    }
}
```

### 4.3 隔离级别建议

| 风险等级 | 推荐模式 |
|---------|---------|
| `low` | `in_process` / `subprocess` |
| `medium` | `subprocess` / `remote` |
| `high` | `container` / `remote` |
| `critical` | `container` + 人工审批 |

---

## 5. 执行信封与返回协议

### 5.1 ExecutionEnvelope

```go
type ExecutionEnvelope struct {
    TraceID      string
    RunID        string
    NodeID       string
    ToolCallID   string
    TenantID     string
    CallerType   string            // workflow|agent|api
    Input        map[string]any
    ContextRefs  []ContextRef
    Constraints  ExecutionConstraints
    AuthContext  AuthContext
}

type ExecutionConstraints struct {
    Timeout       time.Duration
    MaxRetries    int
    BudgetLimit   float64
    NetworkPolicy []string
}
```

### 5.2 ToolResult

```go
type ToolResult struct {
    Status      string            // succeeded|failed|timeout|cancelled
    Output      map[string]any
    Artifacts   []ArtifactOutput
    Diagnostics Diagnostics
    Error       *ToolError
}
```

#### ToolError（标准错误结构）

```go
type ToolError struct {
    Code      string `json:"code"`                 // 领域错误码，如 TOOL_TIMEOUT / TOOL_EXECUTION_ERROR
    Message   string `json:"message"`              // 人类可读错误信息
    Category  string `json:"category"`             // transient|dependency|validation|policy|fatal
    Retryable bool   `json:"retryable"`            // 是否允许自动重试
}
```

约束：

- `ToolResult.Status=failed|timeout|cancelled` 时 `error` 不可为空。
- `retryable=true` 仅表示技术可重试；是否执行仍受 `RetryPolicy` 与 `Policy Engine` 限制。

### 5.3 返回约束

- 成功返回必须符合 `output_schema`
- 错误返回必须标准化为 `ToolError`
- 任何非幂等操作都必须记录 `idempotency_key` 与副作用日志

---

## 6. 执行前治理链路

在 Dispatcher 前必须经过 Policy Engine：

```text
Resolve ToolSpec
 -> RBAC 权限校验
 -> DataAccess scope 校验
 -> RiskLevel + 审批状态校验
 -> Budget 预算校验
 -> bucket_prefixes / db_scopes / domain_whitelist 校验
 -> 进入运行时执行
```

### 6.1 权限模型

最低权限建议：

- `tool:invoke:{tool_id}`
- `asset:read:{scope}`
- `asset:write:{scope}`

### 6.2 预算策略

`CostHint` 与实时计量联动：

```go
type CostHint struct {
    Unit        string  // call|second|token|frame
    Estimated   float64
    Currency    string  // USD
    Confidence  float64 // 0~1
}
```

若预计成本 + 已用成本超过预算，返回 `policy_violation`。

---

## 7. 重试、超时与幂等

### 7.1 重试策略

```go
type RetryPolicy struct {
    MaxAttempts int
    BackoffType string        // fixed|linear|exponential
    InitialDelay time.Duration
    MaxDelay    time.Duration
    Jitter      bool
}
```

### 7.2 幂等规则

- `idempotent=true` 的 Tool 允许自动重试
- `idempotent=false` 仅在显式允许时重试
- 所有重试行为需产出 `tool_retry_scheduled` 事件

### 7.3 超时策略

超时优先级：

1. 调用方节点超时（Workflow/Agent）
2. ToolSpec.Timeout
3. 系统全局默认超时

取最小值生效。

---

## 8. Schema 校验与发布守门

### 8.1 发布前检查清单

1. `input_schema` / `output_schema` 可编译
2. `required_permissions` 完整声明
3. `risk_level` 与执行模式匹配
4. 健康检查通过（remote/container）
5. 文档示例输入输出存在

### 8.2 兼容性判定

| 变更 | 兼容性 |
|------|--------|
| 新增可选输入字段 | 向后兼容 |
| 删除输入字段 | 不兼容 |
| 修改输出字段类型 | 不兼容 |
| 新增输出字段 | 向后兼容 |

不兼容变更必须升 `MAJOR`。

---

## 9. 数据库设计

### 9.1 tools 表

```sql
CREATE TABLE tools (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID         NOT NULL,
    code                 VARCHAR(200) NOT NULL UNIQUE,
    name                 VARCHAR(200) NOT NULL,
    description          TEXT         NOT NULL DEFAULT '',
    category             VARCHAR(50)  NOT NULL,
    input_schema         JSONB        NOT NULL DEFAULT '{}',
    output_schema        JSONB        NOT NULL DEFAULT '{}',
    side_effects         TEXT[]       NOT NULL DEFAULT '{}',
    risk_level           VARCHAR(20)  NOT NULL DEFAULT 'low',
    required_permissions TEXT[]       NOT NULL DEFAULT '{}',
    execution_mode       VARCHAR(50)  NOT NULL DEFAULT 'remote',
    timeout_ms           BIGINT       NOT NULL DEFAULT 30000,
    retry_policy         JSONB        NOT NULL DEFAULT '{}',
    idempotent           BOOLEAN      NOT NULL DEFAULT false,
    cost_hint            JSONB        NOT NULL DEFAULT '{}',
    determinism          VARCHAR(50)  NOT NULL DEFAULT 'deterministic',
    data_access          JSONB        NOT NULL DEFAULT '{}',
    version              VARCHAR(50)  NOT NULL DEFAULT '',
    status               VARCHAR(20)  NOT NULL DEFAULT 'active',
    config               JSONB        NOT NULL DEFAULT '{}',
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tools_tenant_status ON tools(tenant_id, status);
CREATE INDEX idx_tools_category ON tools(category);
CREATE INDEX idx_tools_mode ON tools(execution_mode);
CREATE INDEX idx_tools_risk ON tools(risk_level);
```

### 9.2 tool_health 表（可选）

用于记录 runtime 心跳与失败率：

```sql
CREATE TABLE tool_health (
    id             BIGSERIAL PRIMARY KEY,
    tool_id        UUID         NOT NULL REFERENCES tools(id) ON DELETE CASCADE,
    tool_version   VARCHAR(50)  NOT NULL,
    healthy        BOOLEAN      NOT NULL,
    error_rate_1m  DOUBLE PRECISION NOT NULL DEFAULT 0,
    latency_p95_ms INTEGER      NOT NULL DEFAULT 0,
    checked_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
```

### 9.3 ToolHealthChecker 接口与自动修复

```go
type ToolHealthChecker interface {
    Check(ctx context.Context, toolID uuid.UUID, version string) (ToolHealthReport, error)
    CheckBatch(ctx context.Context, refs []uuid.UUID) ([]ToolHealthReport, error)
    AutoRemediate(ctx context.Context, report ToolHealthReport) (RemediationResult, error)
}

type ToolHealthReport struct {
    ToolID       uuid.UUID
    Version      string
    Healthy      bool
    ErrorRate1m  float64
    LatencyP95Ms int
    LastError    *ToolError
    CheckedAt    time.Time
}

type RemediationResult struct {
    Action  string // restart_worker|switch_region|disable_version|none
    Success bool
    Reason  string
}
```

自动修复策略（建议）：

- 连续 3 次健康检查失败：触发 `restart_worker`。
- 错误率 > 20% 且存在同版本多实例：触发 `switch_region`。
- 连续 10 分钟不健康：自动 `disable_version` 并触发告警与审批。
- 所有修复动作都必须产出 `tool_health_degraded` / `tool_auto_remediated` 事件并写审计日志。

---

## 10. 与算法与工作流的关系

### 10.1 Algorithm 绑定

- `ImplementationBinding.tool_id` 引用 `ToolSpec.id`
- 算法只描述意图，最终执行落在 Tool

### 10.2 Workflow 节点引用

两种方式：

1. 直接 `tool_ref`
2. 间接 `algorithm_ref`（由 resolver 解析成具体 tool）

### 10.3 Agent 调用

Agent 在 `Plan` 阶段从可用工具集中选取 Tool，并在 `Act` 阶段执行。

---

## 11. 事件模型

工具执行关键事件：

| 事件 | 说明 |
|------|------|
| `tool_called` | 发起调用 |
| `tool_retry_scheduled` | 重试排程 |
| `tool_succeeded` | 调用成功 |
| `tool_failed` | 调用失败 |
| `tool_timed_out` | 调用超时 |
| `tool_policy_blocked` | 被策略引擎拦截 |

事件字段定义详见 `08-observability.md`。

---

## 12. 实施建议

### 12.1 分阶段

| 阶段 | 内容 |
|------|------|
| P1 | ToolSpec 模型 + Registry CRUD |
| P2 | Dispatcher + in_process/subprocess |
| P3 | container/remote 执行器 + 健康探针 |
| P4 | Policy/Budget 前置治理 |
| P5 | 发布守门与版本兼容审计 |

### 12.2 测试重点

- schema 校验与不兼容变更拦截
- 幂等工具在重试场景下的副作用一致性
- 高风险工具的审批闸门
- 预算超限时的快速失败路径
