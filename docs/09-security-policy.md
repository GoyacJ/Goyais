# Goyais 安全与策略设计

> 本文档定义 Goyais 的安全治理体系，包括认证鉴权、多租户 RBAC、Policy Engine、数据访问限制、风险审批与预算控制。

最后更新：2026-02-09

---

## 1. 设计目标

### 1.1 安全目标

1. 多租户严格隔离
2. 工具调用最小权限原则
3. 高风险操作可审批、可追溯
4. 执行成本可控，防止失控调用

### 1.2 安全边界

覆盖以下边界：

- 接入边界：API/MCP/CLI 身份认证
- 执行边界：Tool 调用前策略校验
- 数据边界：Asset/Context 访问范围限制
- 网络边界：外部域名白名单

---

## 2. 认证与多租户隔离

### 2.1 认证机制

默认使用 JWT（Access + Refresh）：

- Access Token：短期（2h）
- Refresh Token：长期（7d）
- 支持 API Key（服务间调用）

### 2.2 租户上下文注入

每个请求必须绑定：

- `tenant_id`
- `actor_id`
- `roles`
- `scopes`

若租户上下文缺失，直接拒绝。

### 2.3 隔离原则

- 所有表默认按 `tenant_id` 过滤
- 跨租户操作必须显式 `system_admin` 角色且写审计
- 资产访问链接需带租户签名校验

---

## 3. RBAC 与权限模型

### 3.1 角色建议

| 角色 | 说明 |
|------|------|
| `tenant_owner` | 租户所有管理权限 |
| `tenant_admin` | 配置与运维权限 |
| `operator` | 工作流运行与查看权限 |
| `analyst` | 结果查看与导出权限 |
| `viewer` | 只读权限 |
| `system_admin` | 系统级平台管理员；可执行跨租户运维与应急操作（必须审计） |
| `system_observer` | 系统级只读审计角色；可跨租户查看运行与审计数据，不可变更配置 |

系统角色约束：

- `system_admin` 仅用于平台运维路径，常规业务流不得默认授予
- `system_observer` 仅授予 `*:read:system/*` 与审计查询权限，不包含任何写权限

### 3.2 权限命名

权限采用资源动作模型：

```text
{resource}:{action}:{scope}
```

示例：

- `tool:invoke:tenant/*`
- `workflow:publish:tenant/{workflow_id}`
- `asset:read:tenant/{asset_id}`
- `policy:approve:tenant/*`
- `intent:create:tenant/*`
- `intent:execute:tenant/*`
- `settings:update:tenant/*`

### 3.3 工具级细粒度权限

`ToolSpec.required_permissions` 与 RBAC 权限集做交集校验：

- 缺少任一必需权限则 `policy_blocked`
- 高风险工具额外要求审批权限

### 3.4 AI 操作代理权限边界

当用户通过对话/语音驱动平台操作时，系统仍按“用户权限”执行，不授予 AI 额外特权：

- AI 会话仅可发起当前 `actor_id` 已授权的动作。
- 动作执行前必须解析为显式 `IntentAction`（不可执行隐式自由文本指令）。
- 涉及身份、权限、配置变更的动作默认 `need_confirmation=true`。
- `system_admin` 权限不得通过普通对话会话自动继承，需独立运维凭据与审计链。

---

## 4. Policy Engine

### 4.1 定位

Policy Engine 是执行前“守门层”，所有 Tool 调用必须经过：

```text
AuthN/AuthZ -> Scope -> Risk -> Budget -> Network -> Approvals -> Allow/Deny
```

### 4.2 评估输入

```go
type PolicyCheckRequest struct {
    TenantID      string
    ActorID       string
    RunID         string
    ToolSpec      ToolSpec
    InputSummary  map[string]any
    DataRefs      []AssetRef
    BudgetState   BudgetState
    ApprovalState ApprovalState
}
```

### 4.3 评估输出

```go
type PolicyDecision struct {
    Allowed     bool
    ReasonCode  string            // e.g. missing_permission, budget_exceeded
    Violations  []PolicyViolation
    RequireApproval bool
    Constraints map[string]any
}
```

### 4.4 BudgetState / ApprovalState 约定

`PolicyCheckRequest` 中的 `BudgetState`、`ApprovalState` 类型定义以 `02-domain-model.md` 为权威。本节给出策略语义约定：

```go
type BudgetStatus string

const (
    BudgetStatusHealthy  BudgetStatus = "healthy"
    BudgetStatusWarning  BudgetStatus = "warning"
    BudgetStatusExceeded BudgetStatus = "exceeded"
)

type ApprovalStatus string

const (
    ApprovalStatusNotRequired ApprovalStatus = "not_required"
    ApprovalStatusPending     ApprovalStatus = "pending"
    ApprovalStatusApproved    ApprovalStatus = "approved"
    ApprovalStatusRejected    ApprovalStatus = "rejected"
    ApprovalStatusExpired     ApprovalStatus = "expired"
)

type ApprovalMode string

const (
    ApprovalModeSingle ApprovalMode = "single"
    ApprovalModeDual   ApprovalMode = "dual"
)
```

状态转移：

- `BudgetStatus`：`healthy -> warning -> exceeded`（允许在预算补偿后 `warning -> healthy`）
- `ApprovalStatus`：`not_required -> pending -> approved|rejected|expired`
- `ApprovalMode`：`high -> single`，`critical -> dual`（默认策略，可被更严格策略覆盖）
- 当 `BudgetStatus=exceeded` 时，`PolicyDecision.Allowed=false`，`ReasonCode=budget_exceeded`
- 当 `ApprovalStatus=pending` 时，`PolicyDecision.RequireApproval=true`，运行进入 `paused`
- 双人审批下：`approved_by` 计数达到 `required_approvers` 后才可将 `status` 置为 `approved`

---

## 5. 风险分级与审批

### 5.1 风险分级

| 等级 | 典型行为 | 默认策略 |
|------|---------|---------|
| `low` | 只读分析、无外部写入 | 自动放行 |
| `medium` | 生成新资产、外部 API 调用 | 自动放行 + 审计 |
| `high` | 对外通知、批量修改、高敏数据访问 | 需要审批 |
| `critical` | 跨租户操作、删除高敏资产 | 双人审批 |

### 5.2 审批流

```text
policy_check -> approval_requested
            -> approver_action(approve/reject)
            -> [single] approval_resolved
            -> [dual] quorum_reached? (yes -> approval_resolved / no -> pending)
            -> continue/blocked
```

审批事件写入 RunEvent 与审计日志。

双人审批规则（critical）：

- 必须来自两个不同审批人账户。
- 第一位 `approve` 仅更新 `approved_by`，保持 `status=pending`。
- 第二位有效 `approve` 达到法定人数后，写入 `approval_resolved(status=approved)`。
- 任一审批人 `reject` 即可立即结束为 `rejected`（可配置为“需双拒绝”，默认单拒绝生效）。

### 5.3 超时处理

审批超时默认拒绝：

- `high`：30 分钟
- `critical`：15 分钟

### 5.4 全 AI 操作的强制确认规则

以下动作即使来自已登录用户，也应默认“确认后执行”：

- 用户与角色管理（创建用户、创建角色、变更权限、角色绑定）
- 系统设置变更（安全策略、网络白名单、模型与集成配置）
- 高影响资产操作（批量归档、删除、跨租户迁移）

确认语义：

- 确认内容应展示目标资源、变更 diff、影响范围与回滚建议。
- 用户确认后，若风险仍为 `high/critical`，继续进入审批流。

---

## 6. 数据访问控制

### 6.1 DataAccessSpec 联动

`ToolSpec.data_access` 声明：

- `bucket_prefixes`
- `db_scopes`
- `domain_whitelist`
- `read_scopes`
- `write_scopes`

调用时必须满足：

- 目标 Asset 在读范围内
- 目标写入位置在写范围内
- 访问对象存储前缀命中 `bucket_prefixes`
- 访问数据库域命中 `db_scopes`
- 外部域名命中 `domain_whitelist`

### 6.2 Context 访问限制

- Tool 默认只可读当前 run 的上下文
- 跨 run/context 访问需显式权限 `context:read:trace/*`

### 6.3 导出限制

对 `document`/`structured` 导出可配置：

- 行数阈值
- 敏感字段脱敏
- 强制水印

---

## 7. 网络与执行隔离

### 7.1 出网白名单

- 默认拒绝所有外网访问
- 通过 `domain_whitelist` 逐项放行
- 支持通配符但禁止 `*` 全放开

### 7.2 执行隔离建议

| 条件 | 隔离策略 |
|------|---------|
| 普通低风险 Tool | `in_process`/`subprocess` |
| 依赖复杂 + 高风险 | `container` |
| 外部 SaaS 调用 | `remote` + allowlist |

### 7.3 命令执行保护

对 `subprocess` 模式执行如下限制：

- 固定可执行白名单
- 禁止拼接 shell 原始命令
- 限制工作目录与临时目录

---

## 8. 预算与配额

### 8.1 预算维度

- `max_cost_usd`
- `max_tool_calls`
- `max_tokens`
- `max_duration`

### 8.2 配额层级

1. 租户级（月度）
2. 工作流级（每次 run）
3. 会话级（agent session）

### 8.3 超限策略

- 软阈值（80%）：告警
- 硬阈值（100%）：阻断后续调用
- 可配置“仅允许低成本工具继续”

---

## 9. 密钥与凭据管理

### 9.1 原则

- 不在 ToolSpec 存储明文密钥
- 运行时从 Secret Manager 注入
- 日志与事件中严禁输出密钥明文

### 9.2 配置建议

```yaml
secrets:
  backend: env # env | vault | kms
  mask_in_logs: true
  rotate_days: 90
```

### 9.3 轮换策略

- 到期前 7 天预警
- 双密钥窗口切换
- 轮换后自动回归校验

---

## 10. 审计与合规

### 10.1 必审计事件

- 角色权限变更
- 工具审批决策
- 安全策略配置修改
- 高敏资产访问与删除
- API Key 创建/吊销

### 10.2 合规基线建议

- 最小权限
- 默认拒绝
- 全链路可追溯
- 敏感数据最小暴露

### 10.3 留痕要求

审计记录至少保留：

- `who`
- `when`
- `what`
- `why`（reason / policy）
- `result`

---

## 11. 关键接口

```go
type PolicyEngine interface {
    Evaluate(ctx context.Context, req PolicyCheckRequest) (PolicyDecision, error)
}

type ApprovalService interface {
    Request(ctx context.Context, req ApprovalRequest) (ApprovalTicket, error)
    Resolve(ctx context.Context, ticketID string, action ApprovalAction, comment string) error
}
```

---

## 12. 与其他模块关系

| 模块 | 关系 |
|------|------|
| `04-tool-system.md` | Tool 调用前必须通过策略校验 |
| `06-workflow-engine.md` | 节点执行受预算与权限约束 |
| `07-agent-runtime.md` | Agent 高风险动作触发审批 |
| `08-observability.md` | policy 评估/阻断/审批产出事件和审计 |
| `10-api-design.md` | 策略与审批 API 端点 |
