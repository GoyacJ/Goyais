# Goyais 开发启动包（接口冻结 + 实施优先级 + 联调验收）

> 本文档作为开发启动前的执行基线，冻结关键契约并给出分 Sprint 的落地路径与联调验收标准。

最后更新：2026-02-09

---

## 1. 启动基线

### 1.1 目标

在不牺牲可治理性的前提下，把平台从“手工编排优先”推进到“全意图驱动优先”，并保证以下能力可上线：

1. 用户可通过文本/语音驱动平台核心行为。
2. 已创建资产可被后续意图与工作流复用。
3. 身份、权限、设置等管理面能力纳入同一意图执行链。
4. 全链路可审计、可回放、可审批、可恢复。

### 1.2 权威文档（Source of Truth）

- 领域对象与枚举：`02-domain-model.md`
- API 契约：`10-api-design.md`
- 事件协议与存储：`08-observability.md`
- 策略与审批：`09-security-policy.md`
- 工具契约：`04-tool-system.md`
- 前端交互基线：`11-frontend-design.md`

### 1.3 冻结策略

| 冻结级别 | 规则 |
|---|---|
| `A` | 禁止破坏性变更（仅允许文档修辞修正） |
| `B` | 允许向后兼容扩展（加字段/加端点/加事件，不改既有语义） |

---

## 2. 接口冻结清单

### 2.1 全局协议冻结（级别 A）

| 项 | 冻结值 |
|---|---|
| Base URL | `/api` |
| SSE event name | 固定 `run_event` |
| SSE 事件类型位置 | `data.type`（snake_case） |
| 追踪头 | `X-Trace-ID` |
| 幂等头 | `Idempotency-Key` |
| 国际化头 | `Accept-Language`（协商） / `X-Locale`（覆盖） |

### 2.2 RunEvent 契约冻结（级别 A）

以下字段必须在事件落库与 SSE 下行中保持一致：

`id, trace_id, run_id, seq, parent_run_id, node_id, step_id, tool_call_id, type, status, timestamp, source, payload, error, tenant_id, actor_id`

RunEvent 类型枚举以 `02-domain-model.md` 为准，不允许在各子文档定义“私有变体”。

### 2.3 审批模型冻结（级别 A）

| 项 | 冻结值 |
|---|---|
| 高风险审批模式 | `high -> single` |
| 严重风险审批模式 | `critical -> dual` |
| 双人审批通过条件 | 两位不同审批人 + `quorum_reached=true` |
| 双人审批首票 | 保持 `status=pending`，不放行执行 |

### 2.4 DataAccessSpec 冻结（级别 A）

字段固定为：

`bucket_prefixes, db_scopes, domain_whitelist, read_scopes, write_scopes`

禁止再使用 `network_allowlist` 旧命名。

### 2.5 REST 端点冻结（级别 A/B）

| 端点 | 级别 | 说明 |
|---|---|---|
| `POST /workflows/{id}/runs` | A | 工作流触发标准入口 |
| `POST /runs/{id}/retry` | A | 失败节点重试入口 |
| `GET /runs/{id}/events/stream` | A | 单 Run SSE |
| `GET /traces/{trace_id}/events/stream` | A | 跨 Run SSE |
| `POST /intents` | A | 文本意图入口 |
| `POST /intents/voice` | A | 语音意图入口 |
| `POST /intents/{id}/plan` | A | 重规划入口 |
| `POST /intents/{id}/execute` | A | 意图执行入口 |
| `GET /users` / `POST /users` | A | 身份管理 |
| `GET /roles` / `POST /roles` | A | 角色管理 |
| `GET /permissions` / `POST /permissions` | A | 权限模板管理 |
| `GET /approvals/{id}` | A | 审批票进度查询 |
| `POST /approvals/{id}/approve` | A | 审批通过 |
| `POST /approvals/{id}/reject` | A | 审批拒绝 |
| `POST /approvals/{id}/rewrite` | B | 改写参数并回退重规划 |

### 2.6 幂等端点冻结（级别 A）

以下端点必须支持 `Idempotency-Key`：

- `POST /tools/invoke`
- `POST /workflows/{id}/runs`
- `POST /agent/sessions`
- `POST /runs/{id}/context/patches`
- `POST /approvals/{id}/approve`
- `POST /approvals/{id}/reject`
- `POST /approvals/{id}/rewrite`
- `POST /intents`
- `POST /intents/{id}/execute`

### 2.7 意图动作覆盖冻结（级别 B）

`IntentActionType` 首批必须支持：

- `identity.user.create`
- `identity.role.create`
- `identity.permission.create`
- `identity.role.permissions.update`
- `identity.role.bind`
- `settings.update`
- `asset.upload`
- `workflow.run`
- `tool.invoke`

---

## 3. 实现优先级（按 Sprint）

### 3.1 Sprint 切分

| Sprint | 周期 | 目标 | 必交付 |
|---|---|---|---|
| `S0` | 1 周 | 契约落地与骨架 | API/SSE 骨架、错误码与幂等中间件、RunEvent 落库 |
| `S1` | 2 周 | 意图编排 MVP | `POST /intents`→plan→confirm/approve→execute 主链跑通 |
| `S2` | 2 周 | 资源与执行闭环 | 资产复用、工作流触发、Run 重试、Context CAS 联动 |
| `S3` | 2 周 | 全 AI 交互增强 | 语音意图、审批改写回退、前端助手页联调 |
| `S4` | 1-2 周 | 稳定性与灰度 | 压测、审计核验、回放演练、上线灰度 |

### 3.2 每 Sprint 退出标准（DoD）

| Sprint | DoD |
|---|---|
| `S0` | 所有冻结端点可返回稳定契约；SSE 帧格式固定为 `run_event + data.type` |
| `S1` | 高风险动作进入审批，`critical` 双人审批链路通过 |
| `S2` | “上传资产->意图引用资产->执行工作流->产出事件/产物”全链路可回放 |
| `S3` | 文本与语音入口行为一致，澄清式交互可重规划 |
| `S4` | 关键链路在压测阈值内，审计与回滚演练通过 |

### 3.3 并行开发泳道

| 泳道 | S0-S1 | S2-S3 | S4 |
|---|---|---|---|
| Access/API | 契约与鉴权、幂等、错误码 | 语音入口、审批回调 | 灰度与限流 |
| Control/Intent | Planner/Executor、确认与审批状态机 | 资产复用与重规划 | 故障恢复与回放 |
| Data/Observe | RunEvent/EventStore、SSE 补拉 | 事件索引优化 | 压测与归档策略 |
| Frontend | Assistant 页面骨架、SSE 状态 | 审批票、Diff、语音交互 | 体验打磨与监控面板 |

---

## 4. 联调验收用例（按 Sprint）

### 4.1 Sprint S0

| 用例ID | 场景 | 验收标准 |
|---|---|---|
| `S0-API-001` | `POST /workflows/{id}/runs` 创建运行 | 返回 `run_id`，状态初始 `pending/running`，含 `trace_id` |
| `S0-SSE-001` | 订阅 `/runs/{id}/events/stream` | 仅出现 `event: run_event`；`data.type` 为 snake_case |
| `S0-EVT-001` | 事件落库字段完整性 | `run_events` 可查 `trace_id,tenant_id,source,payload,error` |
| `S0-IDEM-001` | 幂等重复提交 | 同一 `Idempotency-Key` 不产生重复写入 |

### 4.2 Sprint S1

| 用例ID | 场景 | 验收标准 |
|---|---|---|
| `S1-INT-001` | 文本意图生成计划 | `POST /intents` 返回 `planned` 或 `waiting_confirmation` |
| `S1-INT-002` | 澄清式重规划 | `INTENT_PARSE_FAILED` 返回 `clarification_questions`，`/intents/{id}/plan` 可成功重规划 |
| `S1-APR-001` | `high` 单人审批 | 单次 `approve` 后 `approval_resolved(approved)` |
| `S1-APR-002` | `critical` 双人审批 | 首票后 `quorum_reached=false`；第二不同审批人通过后放行 |
| `S1-RBAC-001` | 意图创建用户/角色/权限 | 产出对应 `IntentAction`，并记录审计事件 |

### 4.3 Sprint S2

| 用例ID | 场景 | 验收标准 |
|---|---|---|
| `S2-AST-001` | 上传资产并引用执行 | 新资产可在后续 Intent 中作为 `input_assets` 使用 |
| `S2-WF-001` | 意图触发工作流运行 | `workflow.run` 动作映射到 `POST /workflows/{id}/runs` |
| `S2-RUN-001` | 运行重试 | `POST /runs/{id}/retry` 触发 `run_retried` |
| `S2-CAS-001` | Context CAS 冲突 | 返回 `CONTEXT_CONFLICT`，并可按版本重放成功 |

### 4.4 Sprint S3

| 用例ID | 场景 | 验收标准 |
|---|---|---|
| `S3-VOI-001` | 语音意图链路 | 录音上传后可编辑转写，再进入与文本一致流程 |
| `S3-UI-001` | 前端审批恢复 | 审批通过后会话自动从 `Escalate` 回到 `Plan` 并续跑 |
| `S3-OBS-001` | 事件补拉与去重 | 按 `id` 去重、按 `seq` 检测缺口并补拉 |

### 4.5 Sprint S4

| 用例ID | 场景 | 验收标准 |
|---|---|---|
| `S4-SEC-001` | 跨租户敏感操作 | 非 `system_admin` 被拒绝并写审计 |
| `S4-BUD-001` | 预算超限 | 触发 `budget_exceeded` + 策略拒绝 |
| `S4-PERF-001` | 并发运行压测 | 高并发下单 Run `seq` 单调不乱、SSE 延迟达标 |
| `S4-DR-001` | 回放与恢复演练 | 指定 run 可按事件回放并复现关键状态 |

---

## 5. 上线前检查清单（Release Gate）

| 检查项 | 通过标准 |
|---|---|
| 契约一致性 | 00/01/02/04/06/08/09/10/11 无端点与枚举冲突 |
| 安全治理 | RBAC、审批、预算、DataAccess 全部可观测 |
| 审计追踪 | 高风险操作有完整 trace + event + approval 记录 |
| 前后端联调 | Assistant 主链路（文本/语音）端到端通过 |
| 灰度回滚 | 具备开关、限流、回滚方案与演练记录 |

---

## 6. 责任分配建议

| 角色 | 主责 |
|---|---|
| 产品设计 | Intent 行为边界、确认与审批 UX、验收口径 |
| 架构负责人 | 契约冻结执行、跨域一致性、技术风险收敛 |
| 后端负责人 | API/Intent/Policy/EventStore 主链落地 |
| 前端负责人 | Assistant/审批/SSE 交互与错误处理 |
| 测试负责人 | Sprint 验收用例执行与回归基线维护 |
