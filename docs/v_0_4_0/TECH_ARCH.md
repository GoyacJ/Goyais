# Goyais v0.4.0 技术架构文档

## 1. 文档定位

- 版本：v0.4.0（2026-02-23 同步版）
- 对齐源：`/Users/goya/Repo/Git/Goyais/docs/v_0_4_0/PRD.md`
- 目的：将 PRD 的业务决策落为可实现的技术架构与边界
- 适用对象：架构师、后端、前端、AI/Agent 工程师、测试、运维

---

## 2. 架构总览

### 2.1 三层系统

```text
Desktop (Tauri + Vue)
  -> Hub (Go, 权威控制面)
    -> Worker (Python, 执行面)
```

### 2.2 核心原则

1. **Hub-Authoritative**：权限、调度、审计、资源状态以 Hub 为唯一权威。
2. **Workspace Isolation First**：菜单、权限、数据、业务按工作区隔离。
3. **Conversation Concurrency Model**：跨 Conversation 可并行；单 Conversation 串行队列。
4. **Resource as First-Class Object**：models/rules/skills/mcps 统一建模、统一治理。
5. **Secure by Default**：高风险操作默认阻断确认，密钥操作全审计。

### 2.3 Local / Remote 运行模式

| 维度 | Local Workspace | Remote Workspace |
|------|-----------------|------------------|
| 登录 | 免登录 | 需登录 |
| 权限 | 无 RBAC（全能力） | RBAC + 核心 ABAC |
| Hub | 本机 sidecar | 远程部署 |
| Worker | 本机 sidecar | 远程执行面 |
| 数据归属 | 本地 | 租户/工作区隔离 |
| 管理台 | 不需要 | 必须（P0） |

---

## 3. 领域模型

### 3.1 主要对象

```text
Workspace
  -> WorkspaceConnections(remote)
  -> Projects
    -> ProjectConfigs
    -> Conversations
      -> ConversationSnapshots
      -> Executions
  -> ResourceConfigs(model/rule/skill/mcp)
  -> WorkspaceCatalogRoots
  -> ResourceTestLogs
  -> ShareRequests
  -> PermissionPolicies
  -> PermissionVisibility
  -> AuditLogs
```

### 3.2 关键语义

1. `Conversation` 为主术语，`Session` 不作为主语义。
2. `WorkspaceConnection` 仅用于 remote 工作区，承载 `hub_url/username` 与连接状态。
3. `ProjectConfig` 是项目级资源默认绑定（models/rules/skills/mcps），Conversation 仅可覆盖不反写。
4. `ConversationSnapshot` 是回滚锚点，最小恢复集合包含：消息游标、`execution_snapshots(id/state/queue_index/message_id)`、worktree 引用、Inspector 视图状态。
5. `Execution` 是消息触发的内部执行单元；Conversation 是队列与锁边界。
6. `Resource` 支持 `private/shared` 与 `workspace_native/local_import` 双维度。
7. `ShareRequest` 是共享审批权威记录，支持 `pending/approved/rejected/revoked` 全状态流转。
8. `GeneralSettings` 属于 Desktop 本地策略模型，包含 `launch/default_directory/notifications/telemetry/update_policy/diagnostics`，并要求即时持久化与平台能力显式降级。
9. 模型目录采用手工 JSON 文件：本地工作区目录根来源于 Desktop `defaultProjectDirectory` 同步，远程工作区目录根由 Hub 管理。

### 3.3 状态机（Conversation + Execution + Snapshot）

```text
Conversation queue_state:
  idle -> running -> queued -> running -> idle
      \-> rolling_back -> idle

Execution state:
  queued -> pending -> executing -> completed
                              \-> failed / cancelled

ConversationSnapshot state:
  created -> applied -> stale
```

不变量：

1. 同一 Conversation 仅允许一个活动执行（`pending(leased)|executing`）。
2. 队列必须 FIFO。
3. Stop 只影响当前执行，不清空后续队列。
4. 回滚只允许指向当前 Conversation 内已有消息，且必须写审计。
5. 回滚后目标消息后的队列与执行状态必须重算。

---

## 4. 组件架构

### 4.1 Desktop（Tauri + Vue）

职责：

1. 渲染工作区/项目/对话/事件流 UI。
2. 管理输入、模式切换、模型切换、Stop。
3. 承接权限反馈（403、无菜单、按钮禁用）。
4. 呈现执行状态提示、共享审批页面、管理员页面。

关键模块：

1. `stores/workspace`：当前工作区、菜单、权限快照。
2. `stores/conversation`：消息、队列状态、当前执行。
3. `stores/resource`：资源池、导入与共享视图。
4. `stores/admin`：用户、角色、权限绑定与审批任务。
5. `stores/execution`：事件流、风险审计、Diff 状态。
6. `stores/general_settings`：本地通用设置策略、能力探测、即时持久化状态。

### 4.2 Hub（Go）

职责：

1. 认证鉴权（Remote）。
2. RBAC + ABAC 策略评估。
3. Execution 调度与队列推进。
4. 事件持久化、SSE 分发、重连补发。
5. 资源导入/共享审批/撤销。
6. 密钥治理与审计治理。

关键服务：

1. `workspace_service`
2. `permission_service`
3. `resource_service`
4. `share_request_service`
5. `execution_scheduler`
6. `audit_service`
7. `secret_service`

### 4.3 Worker（Python）

职责：

1. 执行 Agent Loop（P0: Vanilla；P1: LangGraph）。
2. 工具执行（文件/命令/git/patch/skills/mcp）。
3. 高风险调用在 Agent 模式直接执行并审计；Plan 模式拒绝执行。
4. 产出标准化事件流。

执行约束：

1. 工具执行前必须做 path/command 风险检查。
2. 所有 tool_call/tool_result 都必须带 `trace_id`。
3. 无法落地记录任何密钥明文。

---

## 5. 权限与隔离架构

### 5.1 权限模型

Remote Workspace 采用 `RBAC + 核心 ABAC`。

#### RBAC

基础角色：`viewer`、`developer`、`approver`、`admin`。

#### ABAC 四维

1. `subject`：user_id、roles
2. `resource`：workspace_id、owner_id、scope、resource_type、share_status
3. `action`：read/write/execute/share/approve/revoke/admin_manage
4. `context`：risk_level、operation_type、request_source

### 5.2 鉴权执行顺序

1. 身份校验（token/session）。
2. Workspace 边界校验。
3. RBAC 粗粒度动作校验。
4. ABAC 资源条件校验。
5. 通过后执行业务。
6. 写审计日志。

### 5.3 隔离层落地

1. 菜单层：Hub 返回可见菜单集。
2. 路由层：前端路由守卫按权限拦截。
3. 数据层：查询默认加 `workspace_id` 条件。
4. 操作层：写操作再做 ABAC 校验。

---

## 6. 资源导入与共享架构

### 6.1 资源来源模型

`resource.source`：

1. `workspace_native`：本工作区原生创建。
2. `local_import`：从本地来源导入到远程私有副本。

### 6.2 共享流程

```text
private resource
  -> create share request
  -> admin approve/reject
  -> if approved: scope=shared
  -> revoke allowed
```

### 6.3 共享语义

1. 共享后是“远程副本可用”，不依赖用户本地在线。
2. 撤销后新执行不可使用该共享资源。
3. 历史执行日志保留，不做审计删除。

### 6.4 模型密钥共享

1. 允许共享密钥。
2. `share model key` 标记为 `critical` 风险动作。
3. 审批通过前不可被他人使用。
4. 密钥展示全程掩码。

### 6.5 模型目录加载

1. 模型目录优先读取 `<catalog_root>/.goyais/model.json`。
2. 当 `.goyais/model.json` 不存在时，回退使用内置 `models.default.json` 模板。
3. 目录采用严格新格式：`vendors[*].auth` 与 `vendors[*].base_url` 为必填，`homepage/docs/base_urls/notes` 为可选扩展。
4. Hub 对旧格式执行静默自动补齐并写回（本地/远程一致），写回时清理未知字段；补齐失败或校验失败则回退 embedded 模板。
5. 重载触发来源统一为 `manual/page_open/scheduled`，并记录 requested/apply/fallback_or_failed 阶段审计。
6. 定时重载周期固定 3 秒；scheduled 命中缓存可跳过重复审计。
7. 本地工作区目录根通过 `PUT /v1/workspaces/{workspace_id}/catalog-root` 从 Desktop 同步。
8. 远程工作区目录根仅允许管理员变更。

---

## 7. 执行与队列架构

### 7.1 请求入口

`POST /v1/conversations/{conversation_id}/messages`

处理流程：

1. 写入用户消息。
2. 生成 Execution 记录（空闲进入 `pending`，忙碌进入 `queued`）。
3. 返回 `execution_id`、`queue_state`、`queue_index`。
4. 触发队列调度器异步推进。

### 7.2 Queue Dispatcher

队列推进条件：

1. 当前执行 `completed`。
2. 当前执行 `failed/cancelled`。
3. 当前执行被 Stop。

推进动作：

1. 从队列取最早一条 queued 进入 pending。
2. Hub 写入/刷新 `active_execution_id` 与队列锁，不主动 push 到 Worker。
3. Worker 通过 claim 接口主动认领 pending 执行并获得 lease。
4. Worker 以 heartbeat 续租；lease 过期自动回收并保持 FIFO 重排。
5. Worker 上报 `execution_started` 等事件后刷新 Inspector 执行态。

### 7.3 Stop 语义

`POST /v1/conversations/{conversation_id}/stop`

1. 只取消当前 `active_execution_id`。
2. 不删除队列中后续消息。
3. 触发 dispatcher 拉起下一条 queued。
4. 产出 `execution_stopped` 审计事件。

### 7.4 回滚语义（ConversationSnapshot）

`POST /v1/conversations/{conversation_id}/rollback`

1. 入参 `message_id` 必须属于当前 Conversation 且可追溯到有效快照。
2. Hub 锁定 Conversation，切换状态为 `rolling_back`。
3. 应用快照：恢复消息游标、`execution_snapshots` 对应的执行状态（含 `state/queue_index/message_id`）、worktree_ref、Inspector 状态。
4. 重算目标消息之后的 queued 列表，释放锁并回到 `idle|running`。
5. 回滚全过程强制写审计与事件：`rollback_requested`、`snapshot_applied`、`rollback_completed`。

### 7.5 Plan / Agent 与模型切换

1. Conversation 字段 `mode` 默认 `agent`。
2. Execution 使用提交时 `mode_snapshot` 与 `model_snapshot`，防止执行中配置飘移。
3. 模式与模型切换仅影响后续 Execution。

---

## 8. Worktree 与 Git 架构

### 8.1 Git 项目路径

1. Execution 启动创建独立 worktree。
2. 变更完成后产出 Diff。
3. 用户可 Commit / Discard / Export Patch。

### 8.2 合并回主分支

1. Commit 后执行 merge-back 流程。
2. 无冲突则自动完成并清理 worktree。
3. 有冲突则标记 `merge_conflict`，保留 worktree。
4. 用户可继续处理或 discard。

### 8.3 非 Git 降级

1. 非 Git 项目不创建 worktree。
2. 仍可执行文件工具并展示 Diff。
3. 禁用 Commit/merge UI。

---

## 9. API 设计

### 9.1 公共 API（摘要）

#### Workspace / Auth

1. `GET /v1/workspaces`
2. `POST /v1/workspaces/remote-connections`
3. `GET /v1/workspaces/{workspace_id}/status?conversation_id=...`
4. `POST /v1/auth/login`
5. `GET /v1/me`

#### Project / Conversation / Execution

1. `GET|POST /v1/projects`
2. `POST /v1/projects/import`（仅目录）
3. `DELETE /v1/projects/{project_id}`
4. `PUT /v1/projects/{project_id}/config`
5. `GET|POST /v1/projects/{project_id}/conversations`
6. `GET /v1/projects/{project_id}/files`
7. `GET /v1/projects/{project_id}/files/content?path=...`
8. `PATCH|DELETE /v1/conversations/{conversation_id}`
9. `GET /v1/conversations/{conversation_id}`
10. `POST /v1/conversations/{conversation_id}/messages`
11. `GET /v1/conversations/{conversation_id}/events`
12. `POST /v1/conversations/{conversation_id}/stop`
13. `POST /v1/conversations/{conversation_id}/rollback`
14. `GET /v1/conversations/{conversation_id}/export?format=markdown`
15. `GET /v1/executions`
16. `GET /v1/executions/{execution_id}/diff`
17. `POST /v1/executions/{execution_id}/commit`
18. `POST /v1/executions/{execution_id}/discard`

#### Resource / Share

1. `GET /v1/resources`
2. `POST /v1/workspaces/{workspace_id}/resource-imports`
3. `POST /v1/workspaces/{workspace_id}/share-requests`
4. `POST /v1/share-requests/{request_id}/approve`
5. `POST /v1/share-requests/{request_id}/reject`
6. `POST /v1/share-requests/{request_id}/revoke`
7. `GET /v1/workspaces/{workspace_id}/model-catalog`
8. `POST /v1/workspaces/{workspace_id}/model-catalog`
9. `GET|PUT /v1/workspaces/{workspace_id}/catalog-root`
10. `GET|POST /v1/workspaces/{workspace_id}/resource-configs`
11. `PATCH|DELETE /v1/workspaces/{workspace_id}/resource-configs/{config_id}`
12. `POST /v1/workspaces/{workspace_id}/resource-configs/{config_id}/test`
13. `POST /v1/workspaces/{workspace_id}/resource-configs/{config_id}/connect`
14. `GET /v1/workspaces/{workspace_id}/mcps/export`
15. `GET /v1/workspaces/{workspace_id}/project-configs`

#### Admin（P0）

1. `GET|POST /v1/admin/users`
2. `PATCH|DELETE /v1/admin/users/{user_id}`
3. `GET|POST /v1/admin/roles`
4. `PATCH|DELETE /v1/admin/roles/{role_key}`
5. `GET /v1/admin/audit`

### 9.2 内部 API（Hub <-> Worker）

1. `POST /internal/workers/register`
2. `POST /internal/workers/{worker_id}/heartbeat`
3. `POST /internal/executions/claim`
4. `POST /internal/executions/{execution_id}/events/batch`
5. `GET /internal/executions/{execution_id}/control?after_seq=&wait_ms=`
6. `POST /v1/conversations/{conversation_id}/stop` 在 Hub 内部转换为 `execution_control_commands`（`stop`），由 Worker 通过 control poll 拉取。
7. 内部接口必须携带共享 internal token（`X-Internal-Token` 或 `Authorization: Bearer <token>`），无效或缺失返回 `401`。
8. `trace_id` 必须贯穿 Worker claim/control/events 全链路并进入审计。

### 9.3 错误响应

统一格式：

```json
{
  "code": "CONVERSATION_BUSY",
  "message": "Conversation is currently executing another task",
  "details": {
    "active_execution_id": "exec_...",
    "queue_state": "running",
    "queue_index": 2
  },
  "trace_id": "tr_..."
}
```

---

## 10. 事件协议

### 10.1 事件类型

1. `message_received`
2. `execution_started`
3. `thinking_delta`
4. `tool_call`
5. `tool_result`
6. `diff_generated`
7. `execution_stopped`
8. `rollback_requested`
9. `snapshot_applied`
10. `rollback_completed`
11. `execution_done`
12. `execution_error`

### 10.2 事件基础字段

```json
{
  "event_id": "evt_...",
  "execution_id": "exec_...",
  "conversation_id": "conv_...",
  "trace_id": "tr_...",
  "sequence": 42,
  "queue_index": 1,
  "snapshot_id": "snap_...",
  "timestamp": "2026-02-22T10:30:00Z",
  "payload": {}
}
```

### 10.3 SSE 可靠性

1. Hub 先落库再推送。
2. 支持 `Last-Event-ID` 补发。
3. 客户端断线后自动重连。
4. Desktop 订阅策略固定为 `active conversation + all running/queued conversations`。
5. 事件应用路由必须以 `event.conversation_id` 为最终键；与订阅会话不一致时按事件会话分发并记录告警。

---

## 11. 数据库设计（核心表）

### 11.1 Workspace 与权限

```sql
CREATE TABLE workspaces (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  mode TEXT NOT NULL CHECK(mode IN ('local','remote')),
  tenant_id TEXT,
  is_default_local BOOLEAN NOT NULL DEFAULT FALSE,
  hub_url TEXT,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);

CREATE TABLE workspace_connections (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  hub_url TEXT NOT NULL,
  username TEXT NOT NULL,
  auth_state TEXT NOT NULL CHECK(auth_state IN ('connected','reconnecting','disconnected')),
  last_connected_at DATETIME,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);

CREATE TABLE users (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  username TEXT NOT NULL,
  password_hash TEXT,
  status TEXT NOT NULL,
  created_at DATETIME NOT NULL
);

CREATE TABLE roles (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  role_key TEXT NOT NULL,
  UNIQUE(workspace_id, role_key)
);

CREATE TABLE role_grants (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  role_key TEXT NOT NULL,
  action_key TEXT NOT NULL
);

CREATE TABLE abac_policies (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  subject_expr TEXT NOT NULL,
  resource_expr TEXT NOT NULL,
  action_expr TEXT NOT NULL,
  context_expr TEXT NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE permission_visibility (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  role_key TEXT NOT NULL,
  menu_key TEXT NOT NULL,
  visibility TEXT NOT NULL CHECK(visibility IN ('hidden','disabled','readonly','enabled')),
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);
```

### 11.2 资源与共享

```sql
CREATE TABLE resources (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  resource_type TEXT NOT NULL CHECK(resource_type IN ('model','rule','skill','mcp')),
  name TEXT NOT NULL,
  scope TEXT NOT NULL CHECK(scope IN ('private','shared')),
  source TEXT NOT NULL CHECK(source IN ('workspace_native','local_import')),
  owner_user_id TEXT NOT NULL,
  share_status TEXT NOT NULL CHECK(share_status IN ('not_shared','pending','approved','rejected','revoked')),
  spec_json TEXT NOT NULL,
  secret_ref TEXT,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);

CREATE TABLE workspace_catalog_roots (
  workspace_id TEXT PRIMARY KEY,
  catalog_root TEXT NOT NULL,
  updated_at DATETIME NOT NULL
);

CREATE TABLE resource_configs (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  type TEXT NOT NULL CHECK(type IN ('model','rule','skill','mcp')),
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  payload_json TEXT NOT NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);

CREATE TABLE resource_test_logs (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  config_id TEXT NOT NULL,
  test_type TEXT NOT NULL CHECK(test_type IN ('model_test','mcp_connect')),
  result TEXT NOT NULL CHECK(result IN ('success','failed')),
  latency_ms INTEGER NOT NULL DEFAULT 0,
  error_code TEXT,
  details_json TEXT,
  created_at DATETIME NOT NULL
);

CREATE TABLE share_requests (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  resource_id TEXT NOT NULL,
  requester_user_id TEXT NOT NULL,
  approver_user_id TEXT,
  status TEXT NOT NULL CHECK(status IN ('pending','approved','rejected','revoked')),
  reason TEXT,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);
```

`resource_configs.payload_json` 在 `type=model` 场景的关键字段：

1. `vendor` + `model_id`：模型主标识。
2. `base_url_key`：可选 endpoint key，命中 `vendors[*].base_urls`。
3. `base_url`：仅 `Local` 厂商允许覆盖。
4. `api_key`：密钥字段，存储与返回遵循掩码/审计策略。

### 11.3 项目与对话

```sql
CREATE TABLE projects (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  name TEXT NOT NULL,
  repo_path TEXT,
  repo_url TEXT,
  supports_git BOOLEAN NOT NULL DEFAULT TRUE,
  current_revision INTEGER NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);

CREATE TABLE project_configs (
  project_id TEXT PRIMARY KEY,
  default_model_id TEXT,
  model_ids_json TEXT NOT NULL,
  rule_ids_json TEXT NOT NULL,
  skill_ids_json TEXT NOT NULL,
  mcp_ids_json TEXT NOT NULL,
  updated_at DATETIME NOT NULL
);

CREATE TABLE conversations (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  project_id TEXT NOT NULL,
  name TEXT NOT NULL,
  default_mode TEXT NOT NULL DEFAULT 'agent',
  model_id TEXT NOT NULL,
  base_revision INTEGER NOT NULL DEFAULT 0,
  default_worktree BOOLEAN NOT NULL DEFAULT TRUE,
  active_execution_id TEXT,
  queue_state TEXT NOT NULL DEFAULT 'idle',
  archived BOOLEAN NOT NULL DEFAULT FALSE,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);

CREATE TABLE conversation_snapshots (
  id TEXT PRIMARY KEY,
  conversation_id TEXT NOT NULL,
  rollback_point_message_id TEXT NOT NULL,
  queue_state TEXT NOT NULL CHECK(queue_state IN ('idle','running','queued')),
  worktree_ref TEXT,
  inspector_state_json TEXT NOT NULL,
  snapshot_state TEXT NOT NULL CHECK(snapshot_state IN ('created','applied','stale')),
  created_at DATETIME NOT NULL
);

CREATE TABLE executions (
  id TEXT PRIMARY KEY,
  conversation_id TEXT NOT NULL,
  state TEXT NOT NULL,
  queue_index INTEGER NOT NULL,
  mode_snapshot TEXT NOT NULL,
  model_snapshot TEXT NOT NULL,
  project_revision_snapshot INTEGER NOT NULL DEFAULT 0,
  user_message TEXT NOT NULL,
  worktree_path TEXT,
  tokens_in INTEGER NOT NULL DEFAULT 0,
  tokens_out INTEGER NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL,
  completed_at DATETIME
);
```

### 11.4 事件与审计

```sql
CREATE TABLE execution_events (
  id TEXT PRIMARY KEY,
  execution_id TEXT NOT NULL,
  conversation_id TEXT NOT NULL,
  trace_id TEXT NOT NULL,
  event_type TEXT NOT NULL,
  sequence INTEGER NOT NULL,
  payload_json TEXT NOT NULL,
  created_at DATETIME NOT NULL
);

CREATE TABLE execution_control_commands (
  id TEXT PRIMARY KEY,
  execution_id TEXT NOT NULL,
  command_type TEXT NOT NULL CHECK(command_type IN ('stop')),
  payload_json TEXT NOT NULL,
  seq INTEGER NOT NULL,
  created_at DATETIME NOT NULL
);

CREATE TABLE execution_leases (
  execution_id TEXT PRIMARY KEY,
  worker_id TEXT NOT NULL,
  lease_version INTEGER NOT NULL,
  lease_expires_at DATETIME NOT NULL,
  run_attempt INTEGER NOT NULL DEFAULT 1,
  updated_at DATETIME NOT NULL
);

CREATE TABLE workers (
  worker_id TEXT PRIMARY KEY,
  capabilities_json TEXT NOT NULL,
  status TEXT NOT NULL,
  last_heartbeat DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);

CREATE TABLE audit_logs (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  actor_user_id TEXT,
  action_key TEXT NOT NULL,
  target_type TEXT,
  target_id TEXT,
  details_json TEXT,
  trace_id TEXT NOT NULL,
  created_at DATETIME NOT NULL
);

CREATE TABLE secrets (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  encrypted_blob BLOB NOT NULL,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);
```

---

## 12. Agent 执行架构

### 12.1 P0：Vanilla Loop

```python
while True:
    if cancelled(): break
    response = provider.chat(messages, tools, stream=True)
    if response.stop_reason == "tool_use":
        for call in response.tool_calls:
            risk = assess_risk(call)
            if mode == "plan" and risk.level in {"high", "critical"}:
                emit_plan_mode_rejected(call); continue
            result = execute_tool(call)
            emit_tool_result(result)
        continue
    if response.stop_reason == "end_turn":
        emit_done(); break
```

### 12.2 P1：LangGraph ReAct

1. 在 P0 不阻塞上线。
2. 通过 `agent_mode=langgraph` 独立开关。

### 12.3 上下文压缩

1. 保留 system prompt。
2. 保留最近 N 轮对话。
3. 早期轮次摘要化。
4. 超长工具输出截断。

### 12.4 P0 子代理边界

1. P0 仅支持受控子代理并发，最大并发数 `<= 3`（Worker 默认并发 `= 3`，可由环境变量覆盖）。
2. 子代理为短生命周期上下文，不持有独立长期状态。
3. 子代理工具集必须显式降权并继承父执行的风险门禁。
4. 长期自治团队编排不属于 v0.4.0 P0 范围。

---

## 13. 安全架构

### 13.1 密钥治理

1. 密钥明文不落盘。
2. 前端不回显明文。
3. Worker 仅受控短时取用。
4. 使用与审批行为全审计。

### 13.2 工具防护

1. Path Guard：仅允许 repo/worktree 内路径。
2. Command Guard：白名单命令 + 黑名单模式。
3. `run_command` 风险分级细化：`pwd/ls/rg --files/git status/cat` 归类为 `low`，写入/联网/变更命令维持 `high/critical`。
4. Agent 模式高风险调用直接执行并记录审计，Plan 模式高风险调用返回拒绝。

### 13.3 高风险动作

`critical` 动作示例：

1. 删除文件/目录。
2. 外部网络调用可能导致数据外流。
3. 模型密钥共享审批通过。

---

## 14. 前端架构与设计约束

### 14.1 页面架构

1. 主屏幕：左侧导航 + 中部 Conversation 工作区 + 右侧 Inspector + 底部状态栏。
2. 左侧导航：工作区切换（含新增远程工作区）、项目树、用户触发器上拉菜单。
3. 账号信息页：动态权限菜单 + 账号/工作区信息 + 成员角色/权限审计。
4. 设置页：固定菜单（主题、国际化、更新与诊断、通用设置）。
5. 工作区共享配置页：Agent/模型/Rules/Skills/MCP，按入口与权限展示能力差异。
6. 项目配置入口：同时出现在账号信息与设置中。

### 14.2 关键交互约束

1. 输入区固定顺序：`+功能菜单 -> Agent/Plan -> 模型切换 -> 发送按钮`。
2. Conversation 区域消息方向：AI 在左、用户在右。
3. 执行中发送新消息必须入队，不能打断当前执行。
4. “回滚到此处”必须走快照回滚并更新 Inspector 状态。
5. 设置页 `theme` 必须支持 `system/dark/light`，并提供字体样式、字体大小、预设主题；以上配置需即时生效并持久化到本地存储。
6. 设置页 `i18n` 必须支持 `zh-CN/en-US` 即时切换，并持久化到本地存储。
7. 设置页 `general` 必须采用紧凑行式配置，覆盖启动与窗口、默认目录、通知、隐私与遥测、更新策略、诊断与日志；策略项即时生效并持久化，未接入平台能力必须显示不可用原因。
8. 列表页统一 `cursor + limit` 分页，前端必须提供前进/回退游标栈交互。

### 14.3 状态管理建议

1. 全局：workspace/auth/navigation/connection/theme_settings(mode,font_style,font_scale,preset,resolved)/general_settings(launch,default_directory,notifications,telemetry,update_policy,diagnostics)。
2. 领域：conversation/execution/snapshot/resource/admin/project_config。
3. 视图：ui transient state（tab、drawer、dialog、selection）。

### 14.4 UI 实施建议

1. 推荐使用 Pencil MCP 进行样式与交互基线沉淀。
2. 参考：[Pencil Docs](https://docs.pencil.dev)
3. 核心页面必须遵守 token 三层，不得硬编码样式值。

---

## 15. 可观测性

### 15.1 Trace

1. 每个请求生成或透传 `trace_id`。
2. `Desktop -> Hub -> Worker(claim/control/events) -> Audit` 全链路传播。

### 15.2 Metrics（建议）

1. execution_duration_ms
2. queue_wait_ms
3. approval_latency_ms
4. share_request_cycle_ms
5. tool_call_error_rate

### 15.3 审计事件分类

1. execution.create/cancel/done/error
2. execution.stop
3. resource.import/share_request/share_approve/share_reject/share_revoke
4. admin.user_create/role_bind/policy_update
5. git.commit/discard/merge_conflict
6. model_catalog.reload.requested/apply/fallback_or_failed（含 `source=manual|page_open|scheduled`）

---

## 16. 非功能架构约束

1. 多 Conversation 并发能力可配置，默认开启并发。
2. 单 Conversation FIFO 必须严格保证。
3. 事件推送链路本地延迟目标 < 200ms。
4. 执行态聚合必须暴露 `pending/executing/queued` 分态，顶部状态优先级为 `running > queued > idle`。
5. 异常恢复必须以状态一致性为第一目标。
6. Hub/Desktop 重启后需通过 `GET /v1/conversations/{conversation_id}` 恢复历史消息、执行态与快照。
7. i18n 强制双语齐套。

---

## 17. 部署架构

### 17.1 Local

1. Tauri sidecar 启动 Hub + Worker。
2. 本地数据库默认 SQLite，路径优先 `HUB_DB_PATH`，否则 `os.UserConfigDir()/goyais/hub.sqlite3`。
3. 若新默认路径不存在且旧相对路径 `data/hub.sqlite3` 存在，Hub 启动时执行一次性迁移复制并记录审计日志。
4. 启停受 Desktop 生命周期管理。

### 17.2 Remote

1. Hub 可二进制或 Docker 部署。
2. 推荐 Postgres。
3. Worker 可独立横向扩展。

### 17.3 配置优先级

`环境变量 > 配置文件 > 默认值`

---

## 18. P0 / P1 技术边界

### 18.1 P0 必须完成

1. RBAC + 核心 ABAC。
2. 管理员 API + 基础 UI。
3. 资源导入/共享审批/撤销。
4. 模型密钥共享安全控制。
5. Conversation 并发与队列执行。
6. Git 项目与非 Git 降级双路径。

### 18.2 P1 增强项

1. LangGraph ReAct。
2. 更细粒度 ABAC 条件。
3. 更完整命令面板工作流。
4. 高级部署与集群能力。

---

## 19. 与 PRD 的一致性检查点

实施过程中必须保证以下一致：

1. 术语：Conversation 主名，不回退为 Session 主名。
2. 共享语义：远程副本，不远程调用本地。
3. 权限语义：审批能力不得绕过角色与 ABAC。
4. 密钥语义：可共享但高风险，必须审计可撤销。
5. 发布语义：P1 不阻塞 v0.4.0 P0 上线。

---

## 20. 附录：关键接口示例

### 20.1 新增远程工作区连接

`POST /v1/workspaces/remote-connections`

```json
{
  "hub_url": "https://hub-prod.goyais.io",
  "username": "admin@example.com",
  "password": "******"
}
```

### 20.2 Conversation 快照回滚

`POST /v1/conversations/{conversation_id}/rollback`

```json
{
  "message_id": "msg_42",
  "reason": "rollback_to_user_selected_point"
}
```

### 20.3 项目配置绑定

`PUT /v1/projects/{project_id}/config`

```json
{
  "default_model_id": "model_openai_gpt_4_1",
  "model_ids": ["model_openai_gpt_4_1"],
  "rule_ids": ["rule_secure_defaults"],
  "skill_ids": ["skill_review_pr"],
  "mcp_ids": ["mcp_github"]
}
```

### 20.4 模型目录（手工 JSON）重载

`GET /v1/workspaces/{workspace_id}/model-catalog`

```json
{
  "workspace_id": "ws_local",
  "revision": 12,
  "updated_at": "2026-02-23T15:04:05Z",
  "source": "/Users/goya/.goyais/model.json",
  "vendors": [
    {
      "name": "OpenAI",
      "homepage": "https://openai.com/api/",
      "docs": "https://developers.openai.com/api/docs/models",
      "base_url": "https://api.openai.com/v1",
      "base_urls": {
        "global": "https://api.openai.com/v1"
      },
      "auth": {
        "type": "http_bearer",
        "header": "Authorization",
        "scheme": "Bearer",
        "api_key_env": "OPENAI_API_KEY"
      },
      "models": [
        { "id": "gpt-5.3", "label": "GPT-5.3 (Default)", "enabled": true }
      ],
      "notes": ["OpenAI default model"]
    }
  ]
}
```

`POST /v1/workspaces/{workspace_id}/model-catalog`

```json
{
  "source": "page_open"
}
```

### 20.5 目录根路径同步

`PUT /v1/workspaces/{workspace_id}/catalog-root`

```json
{
  "catalog_root": "/Users/goya/.goyais"
}
```

### 20.6 统一资源配置与运行态接口

1. `GET|POST /v1/workspaces/{workspace_id}/resource-configs`
2. `PATCH|DELETE /v1/workspaces/{workspace_id}/resource-configs/{config_id}`
3. `POST /v1/workspaces/{workspace_id}/resource-configs/{config_id}/test`
4. `POST /v1/workspaces/{workspace_id}/resource-configs/{config_id}/connect`
5. `type=model` 时 `name` 为非必填字段；主标识为 `vendor + model_id`。
6. `ModelSpec` 增加 `base_url_key`，由 Hub 校验是否命中 Vendor `base_urls`。
7. `enabled=false` 模型禁止新建配置；历史已存在配置允许继续读取与测试。
8. `GET /v1/workspaces/{workspace_id}/mcps/export`
9. `GET /v1/workspaces/{workspace_id}/project-configs`

### 20.7 2026-02-24 工作区语义收口（实现约束）

1. 工作区数据源以 Hub SQLite 为权威：`workspaces`、`workspace_connections`。
2. `GET /v1/workspaces` 仅返回：本地工作区 + 用户真实新增远程工作区，不允许预置固定 remote demo。
3. 远程连接通过 `POST /v1/workspaces/remote-connections` 落库连接状态，审计记录至少包含 `workspace.create_remote`、`workspace.connect`、`workspace.switch_context`。
4. Desktop 工作区切换必须触发上下文失效与重载，最少覆盖：`auth/permission/project/admin/resource`。
5. 切换到远程工作区若 token 缺失或 401/403，状态置为 `auth_required`，不得自动回退到本地工作区。

### 20.8 工作区运行态聚合接口（状态位权威）

1. `GET /v1/workspaces/{workspace_id}/status` 为主屏幕/账号信息/设置页状态位唯一权威接口。
2. 响应字段最小集合：`workspace_id`、`conversation_id`、`conversation_status`、`hub_url`、`connection_status`、`user_display_name`、`updated_at`。
3. `conversation_status` 标准：`running/queued/stopped/done/error`。
4. `conversation_status` 计算映射：
   - `pending|executing -> running`
   - `queued -> queued`
   - `completed -> done`
   - `failed -> error`
   - `cancelled|无会话|无执行 -> stopped`
5. `connection_status` 权威来源为 `workspace_connections`；本地工作区固定 `connected`。
6. 远程工作区请求该接口必须通过 workspace 级 `conversation.read` 权限校验；无 token 不得降级为本地语义。

### 20.9 Conversation 详情读取与重启恢复

1. `GET /v1/conversations/{conversation_id}` 返回 `conversation/messages/executions/snapshots`，作为 Desktop 重启回填权威来源。
2. Desktop 进入会话时必须先拉取详情并回填 runtime；仅当后端无历史消息时允许使用欢迎语兜底。
3. 会话流应用层必须以 `event.conversation_id` 作为最终路由键，禁止仅按订阅会话 ID 写入。
4. stream detach 条件：会话“非 active 且无未完成执行（queued/pending/executing）”时才允许断开。

### 20.10 Workspace Agent 配置中心化与执行过程可视化

1. 新增 `GET|PUT /v1/workspaces/{workspace_id}/agent-config` 作为工作区 Agent 配置权威接口；鉴权沿用 `resource_config.read/write`。
2. 配置结构固定为：
   - `execution.max_model_turns`（范围 `4..64`）
   - `display.show_process_trace`
   - `display.trace_detail_level`（`basic|verbose`）
3. Hub 持久化层新增 `workspace_agent_configs`（`workspace_id` 主键），并在 `executions` 增加 `agent_config_snapshot_json`。
4. `POST /v1/conversations/{conversation_id}/messages` 创建 Execution 时，必须读取当前工作区 Agent 配置并固化到 `execution.agent_config_snapshot`。
5. Worker `max turns` 解析优先级必须为：`execution.agent_config_snapshot.max_model_turns` -> `WORKER_MAX_MODEL_TURNS` -> 默认值 `24`，并裁剪到 `4..64`。
6. 命中回合上限时，Worker 必须优先软收敛：先发 `thinking_delta(stage=turn_limit_reached)`，再执行一次禁用工具的总结回合；成功则发 `execution_done(truncated=true, reason=MAX_TURNS_REACHED)`，仅总结失败时发 `execution_error(MAX_TURNS_EXCEEDED)`。
7. Desktop 对话主面板执行过程流必须渲染 `execution_started/thinking_delta/tool_call/tool_result`，数据源为 `runtime.events`，并按 `execution_id + sequence/timestamp` 有序聚合。
8. 过程展示粒度由 `execution.agent_config_snapshot.trace_detail_level` 控制；执行结束后不再显示“正在思考/运行中可停止”占位。
9. 配置生效范围必须为“仅新 Execution”；运行中的 Execution 不得被重配。

## 21. 2026-02-24 会话稳定性与并发显示同步矩阵

| change_type | required_docs_to_update | required_sections | status |
|---|---|---|---|
| 新增 Conversation 详情读取接口与回填链路 | PRD.md, TECH_ARCH.md | PRD 14.1/17, TECH_ARCH 9.1/20.9 | done |
| Hub 默认 DB 路径改为用户配置目录并加入一次性迁移 | TECH_ARCH.md, IMPLEMENTATION_PLAN.md | TECH_ARCH 17.1, PLAN Phase 5 门禁增量 | done |
| 会话订阅策略改为 `active + running/queued`，并增加防串流路由 | PRD.md, TECH_ARCH.md, IMPLEMENTATION_PLAN.md | PRD 7.1/16.3, TECH_ARCH 10.3/20.9, PLAN Phase 5 门禁增量 | done |
| 风险分级细化（`run_command` 只读命令 low） | PRD.md, TECH_ARCH.md, DEVELOPMENT_STANDARDS.md | PRD 15.3, TECH_ARCH 13.2, STANDARDS 10.4/13.1 | done |
| Agent 模式移除风险确认链路（删除 confirm API 与 confirming 状态） | PRD.md, TECH_ARCH.md, IMPLEMENTATION_PLAN.md, DEVELOPMENT_STANDARDS.md | PRD 14.1/15.3/24, TECH_ARCH 3.3/9.1/9.2/10.1/12.1/20.8/20.9, PLAN Phase 5/8, STANDARDS 10.4/13 | done |

## 22. 2026-02-24 Agent 配置中心化与执行过程可视化同步矩阵

| change_type | required_docs_to_update | required_sections | status |
|---|---|---|---|
| 新增 Workspace Agent Config API 与 Execution 快照语义 | PRD.md, TECH_ARCH.md | PRD 14/16/17, TECH_ARCH 9.1/20.10 | done |
| `max turns` 改为 Agent 配置驱动 + 软收敛 done(truncated) | PRD.md, TECH_ARCH.md, IMPLEMENTATION_PLAN.md | PRD 16.3/19, TECH_ARCH 12/20.10, PLAN Phase 5 门禁增量 | done |
| 对话区执行过程流（thinking/tool/command）与结束收敛规则 | PRD.md, TECH_ARCH.md, DEVELOPMENT_STANDARDS.md | PRD 16.3/19, TECH_ARCH 14.2/20.10, STANDARDS 11/13 | done |
| 设置 `/workspace/agent` 从占位改为可编辑并动态保存 | PRD.md, IMPLEMENTATION_PLAN.md | PRD 12.1/16.2, PLAN Phase 4/9 验收 | done |
