# Goyais API 设计

> 本文档定义 Goyais 的外部 API 契约，包括 REST API、SSE 事件推送、MCP Server 协议映射、统一错误模型与版本策略。

最后更新：2026-02-09

---

## 1. 总体约定

### 1.1 基础信息

- Base URL：`/api`
- 数据格式：`application/json; charset=utf-8`
- 时间格式：ISO-8601（UTC）
- 认证：`Authorization: Bearer <token>`

### 1.2 通用请求头

| Header | 说明 |
|--------|------|
| `Authorization` | JWT 或 API Key |
| `X-Tenant-ID` | 租户标识（服务端也可由 token 推导） |
| `X-Trace-ID` | 可选链路追踪 ID |
| `Idempotency-Key` | 可选幂等键（写操作推荐） |
| `Accept-Language` | 语言协商头（如 `zh-CN,zh;q=0.9,en;q=0.8`） |
| `X-Locale` | 可选语言覆盖头（如 `zh-CN`、`en`，优先级高于 `Accept-Language`） |

---

## 2. 统一响应封装

### 2.1 成功响应

```json
{
  "code": "OK",
  "message": "success",
  "data": {},
  "meta": {
    "trace_id": "trc_...",
    "request_id": "req_...",
    "locale": "en",
    "fallback": false
  }
}
```

### 2.2 分页响应

```json
{
  "code": "OK",
  "message": "success",
  "data": [ ... ],
  "meta": {
    "page": 1,
    "page_size": 20,
    "total": 135,
    "has_next": true,
    "trace_id": "trc_...",
    "locale": "en",
    "fallback": false
  }
}
```

### 2.3 错误响应

```json
{
  "code": "POLICY_BLOCKED",
  "message": "tool invoke denied by policy",
  "error": {
    "type": "policy_violation",
    "reason": "missing_permission",
    "message_key": "error.policy.blocked",
    "localized_message": "current role is missing required permission",
    "details": {
      "required": "tool:invoke:face.detect",
      "current": ["tool:invoke:ocr.*"]
    }
  },
  "meta": {
    "trace_id": "trc_...",
    "request_id": "req_...",
    "locale": "en",
    "fallback": false
  }
}
```

---

## 3. 错误码规范

### 3.1 通用错误码

| HTTP | code | 说明 |
|------|------|------|
| 400 | `BAD_REQUEST` | 参数校验失败 |
| 401 | `UNAUTHORIZED` | 未认证 |
| 403 | `FORBIDDEN` | 无权限 |
| 404 | `NOT_FOUND` | 资源不存在 |
| 409 | `CONFLICT` | 状态冲突 / CAS 冲突 |
| 422 | `UNPROCESSABLE` | schema 校验失败 |
| 429 | `RATE_LIMITED` | 限流 |
| 500 | `INTERNAL_ERROR` | 服务内部错误 |
| 503 | `SERVICE_UNAVAILABLE` | 依赖不可用 |

### 3.2 领域错误码

| code | 场景 |
|------|------|
| `POLICY_BLOCKED` | 策略校验拒绝 |
| `BUDGET_EXCEEDED` | 预算超限 |
| `APPROVAL_REQUIRED` | 需要人工审批 |
| `TOOL_TIMEOUT` | 工具调用超时 |
| `TOOL_EXECUTION_ERROR` | 工具执行失败（含 provider 错误） |
| `ALGORITHM_RESOLUTION_FAILED` | 算法解析无可用实现 |
| `STREAM_UNAVAILABLE` | 流媒体源不可用或断连 |
| `CONTEXT_CONFLICT` | CAS 冲突 |
| `CONTEXT_PATCH_INVALID` | Patch 语义无效 |
| `MODEL_UNAVAILABLE` | 模型不可用 |
| `RBAC_ROLE_NOT_FOUND` | 角色不存在 |
| `RBAC_PERMISSION_DENIED` | 角色权限不足 |
| `APPROVAL_QUORUM_PENDING` | 双人审批尚未达到法定通过人数 |
| `INTENT_PARSE_FAILED` | 意图解析失败（无法生成可执行计划） |
| `INTENT_CONFIRMATION_REQUIRED` | 意图计划需用户确认后才能执行 |
| `INTENT_ACTION_UNSUPPORTED` | 当前版本不支持该意图动作类型 |

`INTENT_PARSE_FAILED` 建议返回：

```json
{
  "code": "INTENT_PARSE_FAILED",
  "message": "intent requires clarification",
  "error": {
    "type": "intent_parse_failed",
    "details": {
      "clarification_questions": [
        "你希望创建的是系统级角色还是租户级角色？",
        "该角色是否需要用户管理权限？"
      ]
    }
  }
}
```

---

## 4. 资源端点（REST）

### 4.1 Asset API

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/assets/upload` | 上传资产 |
| `POST` | `/assets/import` | 外部 URL 导入 |
| `GET` | `/assets/{id}` | 查询资产详情 |
| `GET` | `/assets` | 资产列表 |
| `POST` | `/assets/{id}/archive` | 归档资产 |
| `DELETE` | `/assets/{id}` | 软删除资产 |
| `GET` | `/assets/{id}/lineage` | 查询血缘 |

上传响应示例：

```json
{
  "code": "OK",
  "data": {
    "id": "8f3b...",
    "type": "video",
    "uri": "s3://bucket/...",
    "digest": "sha256:..."
  }
}
```

### 4.2 Tool API

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/tools` | 注册 ToolSpec |
| `GET` | `/tools` | 列表查询 |
| `GET` | `/tools/{id}/versions/{version}` | 查询特定版本 |
| `POST` | `/tools/{id}/versions/{version}/publish` | 发布 |
| `POST` | `/tools/{id}/versions/{version}/disable` | 禁用 |
| `POST` | `/tools/invoke` | 独立调用工具 |

### 4.3 Algorithm API

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/algorithms` | 创建算法意图 |
| `GET` | `/algorithms` | 列表查询 |
| `POST` | `/algorithms/{id}/versions` | 创建算法版本 |
| `POST` | `/algorithms/{id}/versions/{version}/bindings` | 新增实现绑定 |
| `POST` | `/algorithms/resolve` | 按策略解析实现 |
| `POST` | `/algorithms/{id}/evaluate` | 发起评测 |

### 4.4 Workflow API

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/workflows` | 创建工作流定义 |
| `GET` | `/workflows` | 列表查询 |
| `GET` | `/workflows/{id}` | 获取详情 |
| `POST` | `/workflows/{id}/revisions` | 创建修订 |
| `POST` | `/workflows/{id}/publish` | 发布修订 |
| `POST` | `/workflows/{id}/runs` | 触发执行（创建 Run） |

### 4.5 Run API

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/runs/{id}` | 运行详情 |
| `GET` | `/runs` | 运行列表 |
| `POST` | `/runs/{id}/cancel` | 取消运行 |
| `POST` | `/runs/{id}/pause` | 暂停运行 |
| `POST` | `/runs/{id}/resume` | 恢复运行 |
| `POST` | `/runs/{id}/retry` | 从失败节点重试 |
| `GET` | `/runs/{id}/events` | 查询事件 |
| `GET` | `/runs/{id}/artifacts` | 查询产物 |

### 4.6 Agent API

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/agent/sessions` | 启动会话 |
| `GET` | `/agent/sessions/{id}` | 查询会话状态 |
| `POST` | `/agent/sessions/{id}/input` | 追加用户输入 |
| `POST` | `/agent/sessions/{id}/pause` | 暂停会话 |
| `POST` | `/agent/sessions/{id}/resume` | 恢复会话 |
| `POST` | `/agent/sessions/{id}/cancel` | 取消会话 |

### 4.7 Policy / Approval API

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/policy/evaluate` | 手动策略评估 |
| `GET` | `/approvals` | 审批单列表 |
| `GET` | `/approvals/{id}` | 审批单详情（含投票进度） |
| `POST` | `/approvals/{id}/approve` | 审批通过 |
| `POST` | `/approvals/{id}/reject` | 审批拒绝 |
| `POST` | `/approvals/{id}/rewrite` | 审批改写参数并回退重规划 |

审批语义补充：

- `high` 风险默认 `single` 审批；首个有效 `approve` 即可结束。
- `critical` 风险默认 `dual` 审批；需两位不同审批人通过后才结束为 `approved`。
- `POST /approvals/{id}/approve` 在双人审批首票时返回 `status=pending, quorum_reached=false`，不产出最终放行。
- 审批人身份以认证 token 解析；同一审批人重复投票必须幂等且不重复计数。

`GET /approvals/{id}` 响应字段建议：

```json
{
  "id": "apv_...",
  "risk_level": "critical",
  "mode": "dual",
  "required_approvers": 2,
  "approved_by": ["user_a"],
  "rejected_by": [],
  "quorum_reached": false,
  "status": "pending",
  "timeout_at": "2026-02-09T10:30:00Z"
}
```

### 4.8 User / Role API

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/users` | 用户列表 |
| `GET` | `/users/{id}` | 用户详情 |
| `POST` | `/users` | 创建用户 |
| `PATCH` | `/users/{id}` | 更新用户基础信息 |
| `GET` | `/roles` | 角色列表 |
| `POST` | `/roles` | 创建角色 |
| `PATCH` | `/roles/{id}` | 更新角色（含权限集） |
| `GET` | `/permissions` | 权限模板列表 |
| `POST` | `/permissions` | 创建权限模板（系统/租户级） |
| `POST` | `/roles/{id}/bindings` | 绑定角色到用户 |
| `DELETE` | `/roles/{id}/bindings/{binding_id}` | 解绑角色 |

### 4.9 Context API

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/runs/{id}/context/state` | 查询 ContextState 当前快照 |
| `POST` | `/runs/{id}/context/patches` | 提交 CAS Patch |
| `GET` | `/runs/{id}/context/patches` | 查询 Patch 历史 |

CAS Patch 请求示例：

```json
{
  "before_version": 42,
  "operations": [
    {"op": "replace", "path": "/vars/threshold", "value": 0.8}
  ]
}
```

### 4.10 Model API

| 方法 | 路径 | 说明 |
|------|------|------|
| `GET` | `/models` | 列出模型注册项 |
| `GET` | `/models/{ref}` | 查询模型配置与状态 |
| `POST` | `/models/{ref}/test` | 触发连通性测试 |

### 4.11 Intent API（全 AI 交互）

| 方法 | 路径 | 说明 |
|------|------|------|
| `POST` | `/intents` | 提交文本意图（对话输入） |
| `POST` | `/intents/voice` | 提交语音意图（上传音频或引用音频资产） |
| `GET` | `/intents/{id}` | 查询意图详情（状态/计划/执行结果） |
| `GET` | `/intents` | 查询意图列表 |
| `POST` | `/intents/{id}/plan` | 生成或重生成 IntentPlan |
| `POST` | `/intents/{id}/confirm` | 用户确认计划（高风险前置） |
| `POST` | `/intents/{id}/reject` | 用户拒绝计划 |
| `POST` | `/intents/{id}/execute` | 执行计划（自动或确认后） |
| `GET` | `/intents/{id}/actions` | 查询动作执行明细 |

`POST /intents` 请求示例：

```json
{
  "source_type": "text",
  "input": "为租户创建一个只读审计员角色，并把张三绑定到该角色",
  "execution_mode": "confirm_then_execute",
  "input_assets": [],
  "constraints": {
    "allow_cross_tenant": false
  }
}
```

---

## 5. 查询参数规范

### 5.1 分页

- `page`：默认 `1`
- `page_size`：默认 `20`，最大 `200`

### 5.2 排序

- `sort_by`：字段名
- `sort_order`：`asc` / `desc`

### 5.3 通用过滤

统一使用扁平 query 参数，例如：

```text
GET /api/runs?status=failed&type=workflow&created_from=2026-02-01T00:00:00Z
```

### 5.4 各端点过滤/排序字段

| 资源 | 常用过滤字段 | `sort_by` 可选值 |
|------|--------------|------------------|
| `GET /assets` | `type`,`status`,`owner_id`,`parent_id`,`tag`,`created_from`,`created_to`,`q` | `created_at`,`updated_at`,`size`,`name` |
| `GET /tools` | `category`,`status`,`risk_level`,`execution_mode`,`code`,`q` | `created_at`,`updated_at`,`name`,`risk_level` |
| `GET /algorithms` | `category`,`status`,`code`,`tag`,`q` | `created_at`,`updated_at`,`name`,`code` |
| `GET /workflows` | `status`,`owner_id`,`tag`,`q` | `created_at`,`updated_at`,`name`,`code` |
| `GET /runs` | `type`,`status`,`workflow_id`,`parent_run_id`,`trace_id`,`created_from`,`created_to` | `created_at`,`updated_at`,`started_at`,`finished_at` |
| `GET /runs/{id}/events` | `type`,`node_id`,`tool_call_id`,`from`,`to`,`from_seq` | `seq`,`timestamp`,`id` |
| `GET /approvals` | `status`,`risk_level`,`requester_id`,`approver_id`,`created_from`,`created_to` | `created_at`,`updated_at`,`resolved_at` |
| `GET /runs/{id}/context/patches` | `from_version`,`to_version`,`writer`,`path_prefix` | `version`,`timestamp` |
| `GET /models` | `provider`,`status`,`capability`,`q` | `ref`,`provider`,`updated_at` |
| `GET /users` | `status`,`role_id`,`q` | `created_at`,`updated_at`,`name`,`email` |
| `GET /roles` | `scope`,`q` | `created_at`,`updated_at`,`name` |
| `GET /permissions` | `scope`,`q` | `created_at`,`updated_at`,`code`,`name` |
| `GET /intents` | `status`,`source_type`,`actor_id`,`created_from`,`created_to`,`q` | `created_at`,`updated_at`,`status` |

---

## 6. SSE 事件推送

### 6.1 端点

| 端点 | 说明 |
|------|------|
| `/runs/{id}/events/stream` | 单 Run 事件流 |
| `/traces/{trace_id}/events/stream` | 跨 Run 事件流 |
| `/agent/sessions/{id}/stream` | Agent 会话流 |
| `/intents/{id}/stream` | 意图任务流（计划、确认、执行状态） |

### 6.2 统一 SSE 帧格式（规范）

Goyais 统一采用 `event: run_event`，事件类型放在 `data.type`（snake_case），避免客户端注册大量 event name。

```text
event: run_event
id: evt_123
data: {"type":"node_started","seq":41,"run_id":"...","timestamp":"...","payload":{}}
```

约定：

- `event` 固定为 `run_event`
- `data.type` 必须是 `08-observability.md` 定义的 snake_case 事件类型
- `data.seq` 为单 run 单调递增序号（用于重放与去重）
- 心跳使用 `event: ping`

### 6.3 重连

- 支持 `Last-Event-ID`
- 服务端按事件 ID 增量补发
- 默认心跳 `event: ping` 每 15s
- 客户端应按 `id` 去重，按 `seq` 检测乱序并触发补拉

---

## 7. MCP Server 协议映射

### 7.1 暴露能力

通过 MCP Server 向外提供：

- Tool 列表与 schema
- Tool 调用
- Run 查询与事件查询
- Asset 只读查询（受策略限制）
- Model 注册表只读查询

### 7.2 JSON-RPC 方法映射

| MCP 方法 | REST 映射 | 说明 |
|----------|-----------|------|
| `tools/list` | `GET /tools` | 列出可调用工具 |
| `tools/get` | `GET /tools/{id}/versions/{version}` | 查询工具详情 |
| `tools/call` | `POST /tools/invoke` | 发起工具调用 |
| `runs/get` | `GET /runs/{id}` | 查询 run |
| `runs/events/list` | `GET /runs/{id}/events` | 查询历史事件 |
| `assets/get` | `GET /assets/{id}` | 查询资产详情 |
| `assets/list` | `GET /assets` | 资产检索 |
| `models/list` | `GET /models` | 列出模型 |
| `models/get_status` | `GET /models/{ref}` | 查询模型状态 |
| `intents/submit` | `POST /intents` | 提交文本意图 |
| `intents/get` | `GET /intents/{id}` | 查询意图详情 |
| `intents/execute` | `POST /intents/{id}/execute` | 执行意图计划 |

### 7.3 JSON-RPC 示例

```json
{
  "jsonrpc": "2.0",
  "id": "1",
  "method": "tools/call",
  "params": {
    "name": "face.detect",
    "arguments": {
      "image_asset_id": "..."
    }
  }
}
```

### 7.4 安全模型

- MCP 鉴权 token 与 REST token 隔离，最小权限授权
- 每次 MCP 调用都必须经过 Policy Engine（RBAC、预算、数据访问、审批）
- MCP 会话绑定 `tenant_id`，禁止跨租户透传
- MCP 调用产出的 RunEvent/Audit 必须与 REST 调用同等落库
- 高风险方法（如 `tools/call` 针对高风险 Tool）必须支持审批中断与恢复

---

## 8. 幂等与并发控制

### 8.1 幂等建议

以下接口要求支持 `Idempotency-Key`：

- `POST /tools/invoke`
- `POST /workflows/{id}/runs`
- `POST /agent/sessions`
- `POST /runs/{id}/context/patches`
- `POST /approvals/{id}/approve`
- `POST /approvals/{id}/reject`
- `POST /approvals/{id}/rewrite`
- `POST /intents`
- `POST /intents/{id}/execute`

### 8.2 CAS 接口

更新 ContextState 场景必须带版本号：

```json
{
  "before_version": 42,
  "operations": [ ... ]
}
```

若版本不匹配返回 `409 CONTEXT_CONFLICT`，并在 `error.details` 返回 `current_version`。

---

## 9. 版本策略

### 9.1 API 版本

- 当前主版本基础路径：`/api`
- 资源端点均相对于 Base URL 定义（例如表内 `/runs/{id}` 实际路径为 `/api/runs/{id}`）
- 非兼容升级时提升为新主路径（如 `/api/v2`）
- 非兼容变更才升级主版本
- 向后兼容变更通过字段扩展完成

### 9.2 废弃策略

- 接口标记 `deprecated` 后至少保留 90 天
- 响应 Header 增加：`Deprecation: true`
- 提供替代接口链接：`Link: <...>; rel="successor-version"`

---

## 10. OpenAPI 与 SDK

### 10.1 OpenAPI 规范

- 每个端点必须有请求/响应 schema
- 错误响应统一引用 `ErrorResponse`
- 枚举字段必须穷举
- SSE 的 `data.type` 枚举必须与 `08-observability.md` 同步

### 10.2 SDK 生成

优先提供：

- TypeScript SDK（前端）
- Go SDK（内部服务）

SDK 版本与 API 主版本同步。

---

## 11. 与其他模块关系

| 模块 | 关系 |
|------|------|
| `02-domain-model.md` | API 资源与 Intent/Run/Identity 领域对象一一对应 |
| `03-asset-system.md` | Asset 端点对应其领域模型 |
| `04-tool-system.md` | Tool 注册与调用 API |
| `06-workflow-engine.md` | Workflow 执行触发与 Run/Context 查询 |
| `07-agent-runtime.md` | Agent Session API 与审批恢复 |
| `08-observability.md` | SSE 事件类型与 payload 约定 |
| `09-security-policy.md` | 鉴权、策略、审批与 RBAC 接口 |
