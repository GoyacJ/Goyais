# Goyais 前端设计

> 本文档定义 Goyais 前端（Vue 3 + TypeScript）的页面结构、状态管理、组件分层、实时事件接入与关键交互模式，作为前端实现与联调的设计基线。

最后更新：2026-02-09

---

## 1. 技术栈与目标

### 1.1 技术栈

- Vue 3（Composition API）
- TypeScript
- Vue Router
- Pinia
- Vite
- ECharts（监控图表）
- Monaco/CodeMirror（JSON/Schema 编辑）

### 1.2 前端目标

1. 统一展示 Asset/Tool/Workflow/Run/Agent 五大核心域，并补充 AI 控制域
2. 支持实时运行观测（SSE）
3. 提供低门槛工作流编辑能力
4. 让策略审批和故障排查可闭环
5. 提供“对话/语音驱动全平台操作”的统一 AI 控制台
6. 提供产品级国际化能力（首批 `zh-CN` + `en`），支持运行时语言切换

### 1.3 视觉设计方向

设计方向采用 `Control Tower` 风格：强调“可控、可审计、可追踪”，避免通用聊天产品外观。

- 主视觉：浅色基底（默认）+ 青蓝功能色 + 琥珀告警色
- 信息密度：采用中密度（在可读性与信息承载之间平衡）
- 信息层次：运营看板风格（卡片/表格/时间线）优先于气泡聊天风格
- 空间语言：强栅格、清晰分区、固定信息密度梯度（导航 < 列表 < 详情）
- 可解释性：所有 AI 建议必须配套“依据/风险/影响范围”展示区

### 1.4 设计 Token（样式变量）

```css
:root {
  /* Typography */
  --font-ui: "IBM Plex Sans", "Noto Sans SC", "PingFang SC", sans-serif;
  --font-mono: "JetBrains Mono", "SFMono-Regular", monospace;
  --font-display: "Space Grotesk", "Noto Sans SC", sans-serif;

  /* Brand / Semantic */
  --brand-500: #007a8a;
  --brand-600: #006674;
  --success-500: #1f8f5f;
  --warning-500: #b86d00;
  --danger-500: #c23b32;

  /* Radius / Shadow / Spacing */
  --radius-sm: 8px;
  --radius-md: 12px;
  --radius-lg: 18px;
  --shadow-sm: 0 2px 10px rgba(15, 27, 36, 0.06);
  --shadow-md: 0 8px 24px rgba(15, 27, 36, 0.1);
  --space-1: 4px;
  --space-2: 8px;
  --space-3: 12px;
  --space-4: 16px;
  --space-5: 24px;
  --space-6: 32px;

  /* Motion */
  --dur-fast: 120ms;
  --dur-mid: 220ms;
  --dur-slow: 360ms;
  --ease-standard: cubic-bezier(0.2, 0.8, 0.2, 1);

  /* Density: medium */
  --control-height-sm: 30px;
  --control-height-md: 38px;
  --control-height-lg: 46px;
  --table-row-height: 42px;
}

[data-theme="light"] {
  --bg-canvas: #f3f6f8;
  --bg-panel: #ffffff;
  --bg-elevated: #f8fbfc;
  --bg-accent-soft: #e8f7fb;
  --text-primary: #0f1b24;
  --text-secondary: #51606d;
  --text-muted: #738495;
  --border-default: #d8e1e8;
  --focus-ring: rgba(0, 122, 138, 0.35);
}

[data-theme="dark"] {
  --bg-canvas: #0f161c;
  --bg-panel: #16212b;
  --bg-elevated: #1b2a36;
  --bg-accent-soft: #12323a;
  --text-primary: #e5edf3;
  --text-secondary: #b5c2ce;
  --text-muted: #8ea0b1;
  --border-default: #2a3a47;
  --focus-ring: rgba(68, 195, 212, 0.35);
}
```

### 1.5 设计决策（已确认）

- 信息密度：`中密度`
- 主题策略：`浅色 + 深色双主题`
- AI 语气：`助手亲和`（专业但不生硬）

---

## 2. 信息架构

### 2.1 顶级路由

| 路由 | 页面 | 核心功能 |
|------|------|---------|
| `/assets` | 资产中心 | 上传、检索、预览、血缘查询 |
| `/tools` | 工具中心 | ToolSpec 管理、版本发布、测试调用 |
| `/algorithms` | 算法中心 | 意图/版本/绑定管理、评测记录 |
| `/workflows` | 工作流中心 | 定义、修订、发布、运行 |
| `/runs` | 运行中心 | 时间线、事件流、日志与产物 |
| `/agent` | Agent 会话 | 多轮对话、工具轨迹、升级审批 |
| `/assistant` | AI 控制台 | 对话/语音输入、动作计划确认、全平台操作执行 |
| `/observability` | 监控中心 | 指标看板、告警、trace 检索 |
| `/policy` | 安全策略 | RBAC、审批单、策略规则 |
| `/settings` | 系统设置 | 租户、密钥、集成配置 |

### 2.2 跨页全局能力

- 全局搜索（资产、工作流、run、trace）
- 全局 trace 跳转
- 实时通知中心（审批、失败告警）

### 2.3 全局框架交互（App Shell）

- 左侧固定导航：一级模块 + 租户切换 + 快捷入口（审批、告警）
- 顶部控制条：全局搜索、`Command Palette`、SSE 状态点、用户菜单
- 右侧上下文抽屉：最近事件、当前页面相关审批单、关联 run
- `Command Palette`（`Cmd/Ctrl + K`）支持：
  - 资源跳转（asset/workflow/run/user/role）
  - 快捷动作（创建用户、上传资产、触发执行、打开 AI 控制台）
  - 最近访问与收藏命令

---

## 3. 应用分层

### 3.1 目录建议

```text
src/
  app/
    router/
    store/
    providers/
  pages/
    assets/
    tools/
    algorithms/
    workflows/
    runs/
    agent/
    assistant/
    observability/
    policy/
  features/
    workflow-editor/
    run-timeline/
    agent-console/
    approval-panel/
  entities/
    asset/
    tool/
    workflow/
    run/
  shared/
    ui/
    api/
    utils/
    types/
```

### 3.2 分层职责

- `pages`：页面装配与路由容器
- `features`：跨页面复用的复杂业务组件
- `entities`：实体视图与数据适配
- `shared`：通用 UI 与基础设施

---

## 4. 状态管理

### 4.1 Pinia Store 规划

| Store | 作用 |
|------|------|
| `useAuthStore` | 认证与租户上下文 |
| `useLocaleStore` | 当前语言、回退语言、翻译资源状态 |
| `useAssetStore` | 资产列表、详情、上传状态 |
| `useWorkflowStore` | 工作流定义与 revision 编辑态 |
| `useRunStore` | run 列表、详情、事件缓存 |
| `useAgentStore` | 会话状态、消息与步骤 |
| `useIntentStore` | 意图会话、动作计划、确认状态与执行进度 |
| `usePolicyStore` | 审批单与策略配置 |

### 4.2 数据分层缓存

- 短期缓存：页面内响应式状态
- 会话缓存：Pinia（切页保留）
- 持久缓存：localStorage（仅偏好设置）

---

## 5. API 与事件接入

### 5.1 API 客户端

- 基于 `fetch` 或 `axios` 封装统一客户端
- 自动注入 `Authorization`、`X-Tenant-ID`、`X-Trace-ID`
- 自动注入 `Accept-Language`（来自 `useLocaleStore`）
- 若用户显式切换语言，注入 `X-Locale`
- 统一错误映射为前端错误类型

### 5.2 SSE 接入

提供通用 composable：

```ts
export function useEventStream(url: string, onMessage: (evt: MessageEvent) => void) {
  const source = new EventSource(url, { withCredentials: true })
  source.addEventListener('run_event', onMessage)
  source.addEventListener('ping', () => {})
  return () => source.close()
}
```

### 5.3 断线恢复

- 利用 `Last-Event-ID` 自动续传
- 前端本地缓存最近事件 ID
- 重连失败超过阈值时提醒手动刷新

---

## 6. 关键页面设计

### 6.1 工作流编辑器（Workflow Editor）

核心区域：

- 左侧：节点库（Tool/Algorithm/Agent/SubWorkflow）
- 中央：DAG 画布（拖拽连线、条件边）
- 右侧：节点属性（输入映射、输出映射、重试、超时）
- 底部：Schema 校验与发布检查

关键能力：

- 自动拓扑校验（禁止环）
- 输入输出映射提示
- Revision 差异对比
- 一键触发调试 run

### 6.2 Run 时间线（Run Timeline）

展示维度：

- 状态流：pending -> running -> completed/failed
- 节点轨迹：开始/结束/失败
- 工具轨迹：调用参数摘要、耗时、错误
- Context Patch 轨迹：版本变更

支持按 `node_id`、`tool_call_id`、`event_type` 过滤。

### 6.3 Agent Console

区域拆分：

- 会话消息区（用户/Agent）
- 计划步骤区（Plan/Act/Observe）
- 工具调用区（实时）
- 升级审批区（仅高风险出现）

### 6.4 策略审批面板

- 审批队列列表（待处理/已处理）
- 单据详情（请求内容、风险级别、影响范围）
- 审批动作（approve/reject + comment）

### 6.5 Observability 页面（`/observability`）

布局建议：

- 顶部：全局时间范围与租户过滤器
- 左侧：SLO 卡片（run success、tool success、event ingest lag）
- 中央：时序图（run 失败率、tool 超时率、预算告警率）
- 右侧：告警流与最近 Incident
- 底部：trace/run 检索表（支持一键跳转 `/runs/{id}`）

交互约定：

- 指标图表默认 15s 自动刷新，可手动暂停
- 点击异常点可联动打开对应 run/event 明细抽屉
- 告警项支持 ack/silence（受权限控制）

### 6.6 AI 控制台（`/assistant`）

布局建议：

- 左侧：会话历史（按 intent 分组）
- 中央：对话区（文本输入 + 语音输入状态 + AI 回复）
- 右侧：IntentPlan 抽屉（动作列表、风险级别、依赖关系）
- 底部：执行轨迹（action step 状态流 + 关联 run 链接）

关键能力：

- 文本/语音统一输入，提交到 `/intents` 与 `/intents/voice`
- 高风险动作执行前展示变更 diff（如权限集、设置项）
- 支持“确认后执行 / 拒绝 / 修改后重规划”
- 每个动作可跳转对应资源详情（users/roles/assets/runs）

文案语气规范（助手亲和）：

- 使用“建议 + 依据 + 下一步”三段式表达，避免命令式语气
- 错误提示先给可理解解释，再给可操作按钮（重试、补充信息、发起审批）
- 审批场景提示责任边界（谁确认、影响什么、可否回滚）
- 避免夸张承诺词，保持可信与可审计语气

### 6.7 Settings 页面（`/settings`）

分区：

- `Tenant`：租户信息、默认配额、保留策略
- `Security`：API Key 管理、角色模板、审批超时策略
- `Integrations`：Model Provider、对象存储、Webhook、MCP 配置
- `Runtime`：默认并发、默认超时、SSE 心跳参数

交互约定：

- 所有危险配置需二次确认并展示变更 diff
- 保存动作返回 `trace_id`，便于审计追踪
- 高风险变更（如放宽网络白名单）必须触发审批流程

### 6.8 页面级关键交互流

`AI 控制台 - 低风险动作（自动执行）`：

1. 用户输入文本意图
2. 前端调用 `POST /intents` 并展示 `IntentPlan`
3. 动作均为 `low/medium` 且无确认要求时，自动触发 `POST /intents/{id}/execute`
4. 底部轨迹区实时展示 action 状态与关联 run

`AI 控制台 - 高风险动作（确认 + 审批）`：

1. 用户输入意图后返回 `waiting_confirmation`
2. 右侧抽屉展示变更 diff（权限/配置前后）
3. 用户点击确认后执行，若服务端返回 `APPROVAL_REQUIRED`，显示审批单状态卡
4. 审批通过后自动续跑，拒绝后会话保持可重规划

`语音驱动流程`：

1. 用户点击录音按钮，进入 `recording`
2. 录音结束后上传为音频资产并调用 `POST /intents/voice`
3. 展示转写文本 + 可编辑输入框，允许“先改后提交流程”
4. 提交后流程与文本意图一致

`意图不明确（澄清式交互）`：

1. 后端返回 `INTENT_PARSE_FAILED` 并附 `clarification_questions`
2. 对话区以结构化问题卡片展示待补充信息（例如目标租户、角色范围）
3. 用户补充后前端调用 `POST /intents/{id}/plan` 重新生成计划
4. 新计划覆盖右侧抽屉并保留历史版本对比入口

---

## 7. 组件与 Composable 规范

### 7.1 命名规范

- 组件：`PascalCase`（例如 `RunEventTable.vue`）
- Composable：`useXxx`（例如 `useRunEvents.ts`）
- Store：`useXxxStore`

### 7.2 Composable 约束

每个 composable 聚焦单一能力：

- `usePagination`
- `useQueryFilters`
- `useSSEConnection`
- `useApprovalActions`
- `useIntentSession`
- `useVoiceIntentInput`

推荐 TypeScript 接口签名：

```ts
export type RunEvent = {
  id: string
  seq: number
  type: string
  run_id: string
  timestamp: string
  payload: Record<string, unknown>
}

export type SSEConnectionState = 'idle' | 'connecting' | 'connected' | 'reconnecting' | 'closed' | 'error'

export function useSSEConnection(options: {
  url: string
  eventName?: 'run_event'
  heartbeatMs?: number
  onEvent: (event: RunEvent) => void
  onError?: (err: unknown) => void
}): {
  state: Ref<SSEConnectionState>
  lastEventId: Ref<string | null>
  connect: () => void
  disconnect: () => void
}

export function useApprovalActions(): {
  approve: (ticketId: string, comment?: string) => Promise<void>
  reject: (ticketId: string, comment: string) => Promise<void>
  rewrite: (ticketId: string, constraints: Record<string, unknown>, comment?: string) => Promise<void>
  loading: Ref<boolean>
}

export function useRunEvents(runId: string): {
  events: Ref<RunEvent[]>
  append: (event: RunEvent) => void
  backfill: (fromSeq: number) => Promise<void>
  dedupeById: (eventId: string) => boolean
}

export type IntentStatus =
  | 'received'
  | 'parsing'
  | 'planned'
  | 'waiting_confirmation'
  | 'approved'
  | 'executing'
  | 'succeeded'
  | 'failed'
  | 'rejected'
  | 'cancelled'

export type IntentAction = {
  id: string
  type: string
  resource: string
  risk_level: 'low' | 'medium' | 'high' | 'critical'
  need_confirmation: boolean
  status: 'pending' | 'ready' | 'running' | 'succeeded' | 'failed' | 'skipped' | 'rejected'
  params: Record<string, unknown>
  error?: string
}

export function useIntentSession(): {
  currentIntentId: Ref<string | null>
  status: Ref<IntentStatus>
  actions: Ref<IntentAction[]>
  submitTextIntent: (input: string, mode?: 'dry_run' | 'confirm_then_execute' | 'auto_execute') => Promise<string>
  submitVoiceIntent: (audioAssetId: string, mode?: 'dry_run' | 'confirm_then_execute' | 'auto_execute') => Promise<string>
  replan: (intentId: string, feedback?: Record<string, unknown>) => Promise<void>
  confirm: (intentId: string, comment?: string) => Promise<void>
  reject: (intentId: string, reason: string) => Promise<void>
  execute: (intentId: string) => Promise<void>
}

export function useVoiceIntentInput(): {
  recording: Ref<boolean>
  uploading: Ref<boolean>
  startRecord: () => Promise<void>
  stopRecord: () => Promise<{ assetId: string }>
}
```

### 7.3 类型边界

- API DTO 与 UI ViewModel 分离
- 统一在 `shared/types` 管理领域类型

### 7.4 核心组件规范

| 组件 | 用途 | 样式与交互要求 |
|------|------|---------------|
| `IntentComposer` | 文本/语音输入 | 多行输入 + 语音按钮 + 快捷指令建议；`Enter` 提交，`Shift+Enter` 换行 |
| `IntentPlanDrawer` | 展示动作计划 | 按风险分组；支持展开参数、查看依赖、单步重试 |
| `RiskDiffModal` | 高风险确认 | 双栏 diff（before/after）+ 影响范围 + 审计提示 |
| `ActionTimeline` | 动作执行轨迹 | 使用状态色与图标双编码；可按状态过滤 |
| `SSEHealthBadge` | 实时连接状态 | `green/yellow/red` + 文本状态；断线后提供重连按钮 |
| `ApprovalTicketCard` | 审批反馈 | 展示 ticket 状态、审批人、超时时间与倒计时 |

视觉一致性约束：

- 主按钮仅用于“执行/确认”，次按钮用于“取消/返回”
- 危险动作按钮固定右对齐且使用 `danger-500`
- 表格与时间线统一使用 `font-mono` 展示 ID/时间戳/状态码

---

## 8. 权限与路由守卫

### 8.1 路由守卫

在 `beforeEach` 中校验：

- 是否登录
- 是否具备页面权限
- 当前租户是否可访问该资源

### 8.2 按钮级权限

按钮显隐由权限指令控制，例如：

- `v-permission="'workflow:publish:*'"`
- `v-permission="'policy:approve:*'"`

### 8.3 安全显示

- 敏感字段默认掩码
- 审批与删除操作二次确认
- 文件下载前展示访问审计提示

### 8.4 错误状态 UX（新增）

关键异常场景与页面行为：

| 场景 | 触发条件 | UI 反馈 | 恢复动作 |
|------|---------|---------|---------|
| SSE 断连 | 心跳超时或网络中断 | 顶部非阻塞警告条 + 状态点变黄 | 自动重连；失败后显示“手动重连”按钮 |
| Policy 拒绝 | API 返回 `POLICY_BLOCKED` | 节点/按钮级错误提示 + 原因码 | 提供“发起审批”快捷入口 |
| CAS 冲突 | API 返回 `CONTEXT_CONFLICT` | 表单冲突提示 + 本地草稿与远端版本 diff | 一键“重载远端并重放变更” |
| Tool 执行错误 | `TOOL_EXECUTION_ERROR` / `TOOL_TIMEOUT` | 时间线中高亮失败节点 | 支持“重试该节点”或“切换候选工具” |
| Stream 不可用 | `STREAM_UNAVAILABLE` | 预览区降级占位 + 最近心跳时间 | 提供“重连流源”操作（受权限控制） |
| 意图解析失败 | `INTENT_PARSE_FAILED` | 对话区展示“无法执行”原因 + 可编辑建议 | 用户可补充约束后重试 `replan` |
| 意图待确认 | `INTENT_CONFIRMATION_REQUIRED` | 右侧计划抽屉高亮需确认步骤 | 用户点击“确认并执行”后继续 |

---

## 9. 响应式与性能

### 9.1 布局断点

| 断点 | 说明 |
|------|------|
| `>=1280px` | 三栏复杂工作台 |
| `>=768px && <1280px` | 双栏布局 |
| `<768px` | 单栏堆叠，关键操作置顶 |

`/assistant` 适配补充：

- 桌面端：左（会话）/中（对话）/右（计划）三栏
- 平板端：左栏折叠为抽屉，保留中+右双栏
- 手机端：默认仅对话区，IntentPlan 以底部抽屉上拉查看

### 9.2 性能策略

- 路由级懒加载
- 大列表虚拟滚动
- 事件流批量渲染（100ms 合并）
- 图表按需刷新

### 9.3 可访问性

- 键盘可达
- 颜色对比满足 WCAG AA
- 关键状态提供图标+文本双表达

### 9.4 动效与反馈

- 页面切换：主内容淡入上移（`dur-mid`），侧栏保持稳定不做位移
- 列表加载：使用骨架屏，不使用全屏 loading 遮罩
- SSE 新事件：时间线条目采用轻微高亮闪现（`dur-fast`）后回归
- 风险确认弹窗：进入使用缩放 + 淡入，退出使用淡出
- Toast 规则：
  - 成功：右上角 2.5s 自动消失
  - 警告：需用户关闭或 6s 自动消失
  - 错误：不自动关闭，必须可展开详情

### 9.5 主题切换策略（浅色+深色）

- 默认主题：跟随系统（`prefers-color-scheme`）
- 用户可在设置中显式切换：`light` / `dark` / `system`
- 主题切换不触发页面刷新，使用 CSS 变量实时生效
- ECharts 图表主题与站点主题同步切换
- 代码编辑器（Monaco/CodeMirror）主题同步，避免亮暗混搭

---

## 10. 测试策略

### 10.1 单元测试

- Store 行为
- Composable 逻辑
- 纯展示组件

### 10.2 集成测试

- 页面与 API Mock 联调
- SSE 断线重连
- 权限拦截流程

### 10.3 E2E 测试

核心路径：

1. 上传资产 -> 创建工作流 -> 触发运行 -> 查看时间线
2. 启动 Agent 会话 -> 工具调用 -> 审批 -> 继续执行
3. 策略拒绝 -> 前端提示 -> 跳转审批页

---

## 11. 与后端文档映射

| 前端模块 | 后端文档 |
|---------|---------|
| 资产中心 | `03-asset-system.md` |
| 工具中心 | `04-tool-system.md` |
| 算法中心 | `05-algorithm-library.md` |
| 工作流中心 | `06-workflow-engine.md` |
| Agent 控制台 | `07-agent-runtime.md` |
| AI 控制台（Intent） | `02-domain-model.md` / `10-api-design.md` |
| 时间线与监控 | `08-observability.md` |
| 策略审批 | `09-security-policy.md` |
| API 接口层 | `10-api-design.md` |
