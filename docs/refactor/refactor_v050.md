# Goyais v0.5.0 重构方案

**文档版本**: 3.0
**创建日期**: 2026-03-07
**更新日期**: 2026-03-07
**目标版本**: v0.5.0
**重构类型**: 架构级重构（Breaking Changes）
**架构评审**: Claude Code Agent 首席架构师评估完成（v3.0 技术调整版）
**数据库选型**: SQLite（v0.5.0）→ PostgreSQL（v0.6.0+）

---

## 执行摘要

Goyais v0.4.0 已建立了完整的产品形态，但在架构层面积累了显著的技术债务。本次 v0.5.0 重构将进行**架构级改造**，不保留向后兼容性，目标是建立可扩展、可维护、生产就绪的代码库。

### Claude Code 架构对标分析

作为 Claude Code Agent 首席架构师，我对比了 Goyais 与 Claude.ai 的架构设计（基于官方产品内部工具系统逆向工程研究）：

**Claude Code 核心架构优势**:
1. **工具发现机制**: `tool_search` 元工具实现延迟加载，节省 3000-5000 tokens/会话
2. **分层工具系统**: Always-loaded (21 工具) vs Deferred (11 工具)，按需加载
3. **平台特定优化**: Browser/Desktop/Mobile 三个独立工具集，针对性优化
4. **内存系统**: `memory_user_edits` + `conversation_search` + `recent_chats` 三层记忆栈
5. **交互式工具**: `ask_user_input_v0` (结构化选择器)、`message_compose_v1` (消息草稿)
6. **上下文效率**: 延迟加载策略保持基线上下文精简（节省 30-40% tokens）

**Goyais 当前架构缺陷**:
1. ❌ **无工具发现机制**: 所有工具静态加载，无延迟加载优化
2. ❌ **无内存系统**: 无跨会话记忆能力，每次对话从零开始
3. ❌ **无交互式工具**: 缺少结构化输入组件（选择器、排序器等）
4. ❌ **无平台优化**: 单一工具集，未针对 Desktop/Mobile 差异化
5. ⚠️ **上帝对象**: 399 行 `AppState`，40+ 个 map 字段，单一全局锁
6. ⚠️ **无持久化**: Agent Engine 内存状态，Hub 重启丢失所有会话

### 核心问题（按优先级排序）

#### 🔴 Critical（阻塞生产）
1. **Agent 运行时无持久化**: Engine 内存 map，Hub 重启丢失所有 Session/Run 状态
2. **数据丢失风险**: SSE 重连期间静默丢弃事件，流附加竞态条件
3. **后端可扩展性瓶颈**: 399 行 `AppState` 上帝对象，40+ 个 map 字段，单一全局锁

#### 🟠 High（影响可扩展性）
4. **无工具发现机制**: 缺少 Claude Code 式的延迟加载系统
5. **无内存系统**: 无跨会话记忆能力
6. **前端状态管理混乱**: Session/Conversation 双重命名，镜像投影，无界增长

#### 🟡 Medium（技术债务）
7. **测试覆盖不足**: 仅 4 个 E2E 场景，关键路径排除在覆盖率外
8. **无交互式工具框架**: 缺少结构化输入组件

### 重构目标（对标 Claude Code）

- **工具系统**: 引入工具发现机制 + 延迟加载策略（节省 30-40% 上下文 tokens）
- **内存系统**: 实现三层记忆栈（用户编辑 + 会话搜索 + 最近对话）
- **持久化**: 从内存 map 迁移到 SQLite（v0.5.0）/ PostgreSQL（v0.6.0+）（Agent Engine + AppState）
- **状态管理**: 统一命名（Session），消除镜像投影，引入 Repository 模式
- **并发安全**: 修复竞态条件，实现细粒度锁（per-session）
- **交互式工具**: 实现结构化输入组件框架（选择器、排序器、消息草稿）
- **平台优化**: Desktop/Mobile 差异化工具集（可选，v0.6.0）
- **测试完备**: E2E 覆盖率提升至关键用户旅程 80%+

---

## 当前架构分析

### 架构优势

✅ **清晰的分层设计**: `core` 接口层零外部依赖
✅ **契约优先**: OpenAPI 驱动类型生成
✅ **现代技术栈**: Vue 3 Composition API + Go 1.24
✅ **质量门禁**: 圈复杂度、文件大小、覆盖率自动检查

### 关键缺陷

#### 0. Claude Code 架构缺失（新增）

**问题**: Goyais 缺少 Claude Code 的核心架构模式

**0.1 无工具发现机制** 🔴 Critical

Claude Code 通过 `tool_search` 元工具实现延迟加载：
- Always-loaded 工具（21 个）：`web_search`, `web_fetch`, `memory_user_edits` 等
- Deferred 工具（11 个）：`user_time_v0`, `chart_display_v0`, `calendar_search_v0` 等
- 节省 3000-5000 tokens/会话（约 30-40% 上下文）

Goyais 当前状态：
- 所有工具静态加载到每个会话
- 无工具发现 API
- 无延迟加载策略
- 上下文 token 浪费严重

**影响**:
- 上下文窗口利用率低
- 无法支持大规模工具生态（100+ 工具时不可行）
- 无法按平台差异化工具集

**0.2 无内存系统** 🟠 High

Claude Code 三层记忆栈：
```
Layer 1: memory_user_edits    - 用户主动编辑的记忆（姓名、偏好、上下文）
Layer 2: conversation_search  - 跨会话语义搜索
Layer 3: recent_chats         - 最近对话列表
```

Goyais 当前状态：
- 无跨会话记忆能力
- 每次对话从零开始
- 无用户偏好持久化
- 无会话历史搜索

**影响**:
- 用户体验割裂（每次重复介绍自己）
- 无法构建长期用户画像
- 无法实现个性化响应

**0.3 无交互式工具框架** 🟡 Medium

Claude Code 交互式工具：
- `ask_user_input_v0`: 结构化选择器（单选/多选/排序）
- `message_compose_v1`: 消息草稿（邮件/短信/Slack）
- `chart_display_v0`: 内联图表渲染

Goyais 当前状态：
- 仅支持纯文本交互
- 无结构化输入组件
- 无内联可视化

**影响**:
- 交互效率低（多轮对话收集信息）
- 无法支持复杂决策场景
- 用户体验单一

#### 1. 后端架构缺陷

**问题**: `services/hub/internal/httpapi/state.go` (399 行)

```go
type AppState struct {
    mu sync.RWMutex  // ⚠️ 单一全局锁
    workspaces map[string]Workspace
    sessions map[string]Session
    conversations map[string]Conversation
    executions map[string]Execution
    executionEvents map[string][]ExecutionEvent
    // ... 35+ 个 map 字段
}
```

**影响**:
- 所有操作串行化通过单一锁
- 无界 map 增长导致内存泄漏
- 无领域模型封装
- 违反单一职责原则

#### 2. 前端状态管理缺陷

**问题**: `apps/desktop/src/modules/session/store/state.ts`

```typescript
type SessionState = {
  bySessionId: Record<string, SessionRuntime>;
  byConversationId: Record<string, SessionRuntime>;  // ⚠️ 同一引用
  sessionTimers: Record<string, ReturnType<typeof setTimeout>>;
  timers: Record<string, ReturnType<typeof setTimeout>>;  // ⚠️ 镜像
}

type SessionRuntime = {
  runs: Run[];
  executions: Run[];  // ⚠️ 手动同步
  events: RunLifecycleEvent[];  // ⚠️ 无界增长（最多 1000）
}
```

**影响**:
- 命名混乱（Session vs Conversation）
- 内存重复
- 手动同步易出错
- 长会话内存膨胀

#### 3. 数据丢失风险

**问题**: `apps/desktop/src/modules/session/store/stream.ts:51-76`

```typescript
export function attachSessionStream(session: Session, token?: string): void {
  let resyncInFlight = false;

  sessionStore.sessionStreams[session.id] = streamSessionEvents(session.id, {
    onEvent: (event) => {
      if (isSSEBackfillResyncEvent(incoming)) {
        resyncInFlight = true;
        void getSessionDetail(session.id, { token })
          .finally(() => { resyncInFlight = false; });
        return;
      }
      if (resyncInFlight) return;  // ⚠️ 静默丢弃事件
      applyIncomingExecutionEvent(eventSessionId, incoming);
    }
  });
}
```

**影响**:
- SSE 重连期间丢失事件
- 无事件队列缓冲
- UI 显示过期数据

#### 4. 测试覆盖不足

**现状**:
- E2E 测试: 仅 4 个场景（主屏渲染、路由守卫、设置 UI、Inspector）
- 排除覆盖: `useMainScreen*.ts`, `streamCoordinator.ts`, `sseClient.ts`
- 缺失场景: 工作区切换、ChangeSet 审批、MCP 集成、Hook 调度

#### 5. Agent 运行时架构缺陷

**问题**: `services/hub/internal/agent/runtime/loop/engine.go` (789 行)

Agent v4 虽然采用了清晰的接口驱动设计，但在实现层面存在关键问题：

#### 6. 资源配置继承机制缺失 🔴 Critical

**问题**: 工作区级 ResourceConfig 与项目级 ProjectConfig 之间缺少清晰的继承与覆盖机制

**当前状态**:
```go
// ProjectConfig 定义了项目可用资源
type ProjectConfig struct {
    ProjectID              string
    ModelConfigIDs         []string
    DefaultModelConfigID   string
    TokenThreshold         int
    ModelTokenThresholds   map[string]int
    RuleIDs                []string
    SkillIDs               []string
    MCPIDs                 []string
}

// Session 直接引用资源 ID
type Session struct {
    ModelConfigID string
    RuleIDs       []string
    SkillIDs      []string
    MCPIDs        []string
}
```

**缺陷**:
1. **无继承验证**: Session 创建时不验证 `model_config_id` 是否在 ProjectConfig 允许列表中
2. **无级联更新**: ProjectConfig 移除某个 model 时,已有 Session 不会收到通知或自动降级
3. **无默认值回退**: 当 Session 引用的资源被删除时,无回退到项目默认值的机制
4. **无实时感知**: 业务需求要求"模型、规则、技能、MCP 变更时会话实时感知",但当前无事件推送机制

**影响**:
- Session 可能引用已删除的资源,导致执行失败
- 资源配置变更后,已有会话状态不一致
- 无法实现"工作区管理员禁用某个模型后,所有使用该模型的会话自动切换"的治理需求

#### 7. 回滚点(Checkpoint)实现不完整 🟠 High

**问题**: 业务需求要求"支持回滚点,回滚到发送消息时的状态",但当前实现仅支持 Git 项目的 commit 快照

**当前状态**:
```go
// CheckpointSummary 仅记录 Git commit 信息
type CheckpointSummary struct {
    CheckpointID  string
    Message       string
    GitCommitID   string  // ⚠️ 仅 Git 项目有效
    EntriesDigest string
    CreatedAt     string
}

// 无 Rollback API
// 无 Session 状态快照机制
// 无消息级回滚能力
```

**缺陷**:
1. **非 Git 项目无回滚**: `project_kind=non_git` 的项目无法创建 Checkpoint
2. **无 Session 状态回滚**: 仅能回滚文件变更,无法回滚 Session 的消息历史、Run 队列、资源绑定
3. **无 API 支持**: OpenAPI 规范中无 `/sessions/{id}/rollback` 端点
4. **无时间旅行**: 无法查看"某个 Checkpoint 时刻的 Session 完整状态"

**影响**:
- 用户无法实现"撤销最近 3 条消息,回到之前的对话分支"
- 非 Git 项目用户无回滚能力
- 无法支持"保存对话快照,稍后恢复"的高级场景

#### 8. 变更追踪(ChangeSet)竞态条件 🟠 High

**问题**: `change_set_service.go` 中的 ChangeSet 构建与提交存在 check-then-act 竞态

**当前代码**:
```go
func buildConversationChangeSetLocked(state *AppState, conversationID string) (ConversationChangeSet, error) {
    // 1. 检查是否有运行中的 Run
    busy := hasMutableExecutionsLocked(state, conversationID)
    capability := ChangeSetCapability{
        CanCommit:  !busy,
        CanDiscard: !busy,
    }
    // ...
}

func ConversationChangeSetCommitHandler(state *AppState) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 2. 再次检查(但可能已过期)
        changeSet, err := buildConversationChangeSetLocked(state, conversationID)
        if !changeSet.Capability.CanCommit {
            // 返回错误
        }
        // 3. 执行 commit(期间可能有新 Run 提交)
        checkpoint, err := driver.Commit(project, changeSet.Entries, input.Message)
    }
}
```

**竞态场景**:
1. 用户 A 调用 `GET /sessions/{id}/changeset`,得到 `can_commit=true`
2. 用户 B 提交新消息,创建 Run,Session 进入 `running` 状态
3. 用户 A 调用 `POST /sessions/{id}/changeset/commit`,此时 `busy=true`,但前端已显示"可提交"

**影响**:
- 前端 UI 状态与后端实际状态不一致
- 可能导致 commit 失败,用户体验差
- 无乐观锁或版本号机制防止并发冲突

#### 9. Session 队列状态机不完整 🟡 Medium

**问题**: `QueueState` 状态机缺少关键转换路径与错误恢复机制

**当前状态**:
```typescript
type QueueState = "idle" | "running" | "queued";
```

**缺陷**:
1. **无 paused 状态**: 用户无法"暂停当前 Run,稍后恢复"
2. **无 error 状态**: Run 失败后,Session 直接回到 `idle`,丢失错误上下文
3. **无队列优先级**: 所有 Run 严格 FIFO,无法实现"紧急消息插队"
4. **无并发控制**: 业务需求说"消息都是 FIFO",但代码中无并发提交保护

**影响**:
- 无法支持"暂停长时间运行的 Run"
- Run 失败后,用户不知道 Session 是否可安全重试
- 无法实现"管理员优先级消息"等高级调度

#### 10. 实时资源变更感知机制缺失 🟠 High

**问题**: 业务需求要求"模型、规则、技能、MCP 变更时会话实时感知",但当前无事件推送机制

**当前状态**:
- Hub 修改 ResourceConfig 后,仅更新内存 map
- 已连接的 Desktop/Mobile 客户端不会收到通知
- Session 继续使用旧的资源配置,直到手动刷新

**缺失能力**:
1. **无 WebSocket/SSE 资源变更事件**: 当前 SSE 仅推送 Run 生命周期事件
2. **无 Session 资源版本号**: Session 不知道自己引用的资源是否已过期
3. **无自动降级策略**: 资源被删除后,Session 无法自动切换到备用资源
4. **无前端轮询机制**: 前端无定时检查资源配置变更的逻辑

**影响**:
- 管理员禁用某个模型后,用户仍可能使用该模型(直到 Session 重启)
- 资源配置不一致导致执行失败
- 无法实现"全局紧急禁用某个 MCP 服务器"的治理需求

**5.1 内存状态无持久化** 🔴 Critical

```go
type Engine struct {
    mu sync.Mutex
    sessions map[core.SessionID]*sessionRuntime  // ⚠️ 内存 map，无持久化
    runs     map[core.RunID]*runRuntime          // ⚠️ 重启丢失
}
```

**影响**:
- Hub 进程重启后所有 Session/Run 状态丢失
- 无法支持分布式部署（多 Hub 实例）
- 内存无界增长风险（sessions/runs map 永不清理）

**5.2 全局锁竞争** 🟠 High

```go
func (e *Engine) Submit(_ context.Context, sessionID string, input core.UserInput) (runID string, err error) {
    e.mu.Lock()         // ⚠️ 全局锁
    defer e.mu.Unlock()
    // 所有 Session 的 Run 提交串行化
}
```

**影响**:
- 所有 Session 操作串行化，并发性能受限
- 高负载下锁竞争导致延迟增加
- 无法利用多核并行处理

**5.3 接口实现不完整** 🟡 Medium

`core/interfaces.go` 定义了 10+ 个核心接口，但部分未实现：
- `CommandBus` - Slash 命令分发（当前通过伪造 Run 事件实现）
- `ToolExecutor` - 工具调用流水线（逻辑分散）
- `CheckpointStore` - 文件快照与回滚（存根）

**5.4 权限评估逻辑分散** 🟠 High

权限决策分散在多个层级：
- `policy/gate.go` - 基于规则 DSL 的评估
- `policy/approval/router.go` - 运行时审批等待
- `runtime/loop/default_model_executor.go` - 工具调用前检查
- Hook 系统 - `PreToolUse` 事件可修改决策

**影响**: 权限决策路径不透明，难以审计，Hook 可绕过 PermissionGate

**5.5 Subscriber 背压策略不透明** 🟠 High

```go
if subscriberCfg.BackpressurePolicy == "" {
    subscriberCfg.BackpressurePolicy = subscribers.BackpressureDropNewest
}
```

**问题**:
- 默认策略 `DropNewest` 会丢弃最新事件
- 无监控指标暴露丢弃事件数量
- 调用方无法感知事件丢失

---

## v0.5.0 架构愿景

### 设计原则

1. **领域驱动设计 (DDD)**: 明确领域边界，聚合根，值对象
2. **CQRS 轻量化**: 读写分离，优化查询性能
3. **事件溯源（可选）**: 为审计和回放保留事件日志
4. **依赖注入**: 消除全局单例，提升可测试性
5. **Repository 模式**: 抽象数据访问层

### 目标架构图

```
┌─────────────────────────────────────────────────────────────┐
│                     Frontend (Vue 3 + Pinia)                │
├─────────────────────────────────────────────────────────────┤
│  Presentation Layer                                         │
│  ├─ Views (useController composables)                       │
│  └─ Components (UI primitives)                              │
├─────────────────────────────────────────────────────────────┤
│  Application Layer                                          │
│  ├─ Stores (Pinia, 单一数据源)                              │
│  ├─ Services (业务逻辑编排)                                 │
│  └─ Repositories (API 抽象层)                               │
├─────────────────────────────────────────────────────────────┤
│  Infrastructure Layer                                       │
│  ├─ HTTP Client (axios/fetch)                               │
│  ├─ SSE Client (EventSource wrapper)                        │
│  └─ Local Storage (persistence)                             │
└─────────────────────────────────────────────────────────────┘
                            ↕ REST + SSE
┌─────────────────────────────────────────────────────────────┐
│                     Backend (Go Hub)                        │
├─────────────────────────────────────────────────────────────┤
│  API Layer (httpapi)                                        │
│  ├─ Handlers (HTTP 路由)                                    │
│  ├─ Middleware (Auth, Logging, Tracing)                     │
│  └─ DTOs (Request/Response)                                 │
├─────────────────────────────────────────────────────────────┤
│  Application Layer                                          │
│  ├─ Commands (写操作)                                       │
│  ├─ Queries (读操作)                                        │
│  └─ Event Handlers (异步处理)                               │
├─────────────────────────────────────────────────────────────┤
│  Domain Layer                                               │
│  ├─ Aggregates (Workspace, Session, Run)                    │
│  ├─ Entities (Project, Message, ChangeSet)                  │
│  ├─ Value Objects (RunState, TokenUsage)                    │
│  └─ Domain Services (Agent Runtime)                         │
├─────────────────────────────────────────────────────────────┤
│  Infrastructure Layer                                       │
│  ├─ Repositories (数据访问)                                 │
│  ├─ Database (SQLite/PostgreSQL)                            │
│  ├─ Event Bus (内存/Redis)                                  │
│  └─ External Services (OpenAI, Google AI)                   │
└─────────────────────────────────────────────────────────────┘
```

---

## 重构领域详解

### 领域 0: Claude Code 架构模式引入（新增）

#### 0.1 工具发现与延迟加载系统

**目标**: 实现 Claude Code 式的工具发现机制，节省 30-40% 上下文 tokens

**架构设计**:

```go
// 新增：internal/agent/tools/discovery/
package discovery

// ToolRegistry 工具注册表
type ToolRegistry struct {
    alwaysLoaded map[string]ToolDefinition  // 21 个核心工具
    deferred     map[string]ToolDefinition  // 按需加载工具
    categories   map[string][]string        // 工具分类索引
}

// ToolSearchRequest 工具搜索请求
type ToolSearchRequest struct {
    Query      string   // 关键词查询
    Categories []string // 分类过滤
    MaxResults int      // 最大返回数
}

// ToolSearchResponse 工具搜索响应
type ToolSearchResponse struct {
    Tools []ToolDefinition
}

// ToolDefinition 工具定义
type ToolDefinition struct {
    Name        string
    Description string
    Parameters  json.RawMessage
    Category    string
    Version     string
    Platform    []string // ["desktop", "mobile", "web"]
}

// Search 实现工具搜索
func (r *ToolRegistry) Search(req ToolSearchRequest) ToolSearchResponse {
    // 1. 关键词匹配（语义搜索）
    // 2. 分类过滤
    // 3. 平台过滤
    // 4. 返回 top-k 结果
}
```

**工具分类**:

| 分类 | Always-Loaded | Deferred | 说明 |
|------|---------------|----------|------|
| Core | `web_search`, `web_fetch`, `bash_tool` | - | 核心能力 |
| Memory | `memory_user_edits`, `conversation_search` | - | 记忆系统 |
| Context | - | `user_time_v0`, `user_location_v0` | 上下文感知 |
| Interaction | `ask_user_input_v0` | `message_compose_v1` | 交互组件 |
| Visualization | - | `chart_display_v0` | 可视化 |
| Calendar | - | `calendar_search_v0`, `event_create_v0` 等 | 日历管理 |
| Device | - | `alarm_create_v0`, `timer_create_v0` | 设备集成 |

**延迟加载策略**:

```go
// ContextBuilder 集成工具发现
func (b *Builder) Build(ctx context.Context, req BuildContextRequest) (PromptContext, error) {
    // 1. 加载 always-loaded 工具
    tools := b.registry.GetAlwaysLoaded()

    // 2. 分析用户输入，预测需要的工具
    if needsTimeContext(req.Input) {
        discovered := b.registry.Search(ToolSearchRequest{
            Query: "time date timezone",
            Categories: []string{"context"},
        })
        tools = append(tools, discovered.Tools...)
    }

    // 3. 构建 prompt context
    return PromptContext{
        Tools: tools,
        // ...
    }
}
```

**预期效果**:
- 基线上下文从 ~15k tokens 降至 ~10k tokens
- 支持 100+ 工具生态（按需加载）
- 平台特定工具集（Desktop/Mobile 差异化）

#### 0.2 内存系统实现

**目标**: 实现三层记忆栈，支持跨会话用户画像

**数据库 Schema**（SQLite + sqlite-vec 扩展）:

```sql
-- 用户记忆（Layer 1）
CREATE TABLE user_memories (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    workspace_id TEXT NOT NULL,
    memory_type TEXT NOT NULL CHECK (memory_type IN ('fact', 'preference', 'context')),
    key TEXT NOT NULL,
    value TEXT NOT NULL,
    source TEXT,  -- 'user_edit' | 'inferred' | 'explicit'
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(user_id, workspace_id, key)
);

-- 会话索引（Layer 2）- 使用 sqlite-vec 扩展
CREATE TABLE session_embeddings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    chunk_index INTEGER NOT NULL,
    content TEXT NOT NULL,
    embedding BLOB,  -- 存储为 BLOB（float32 数组序列化）
    created_at TEXT NOT NULL
);

CREATE INDEX idx_session_embeddings_session ON session_embeddings(session_id);

-- 创建 sqlite-vec 虚拟表用于向量检索
CREATE VIRTUAL TABLE session_embeddings_vec USING vec0(
    embedding float[1536]
);

-- 最近会话（Layer 3）
CREATE TABLE recent_sessions (
    session_id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    title TEXT,
    last_message TEXT,
    message_count INTEGER DEFAULT 0,
    updated_at TEXT NOT NULL
);

CREATE INDEX idx_recent_sessions_workspace ON recent_sessions(workspace_id, updated_at DESC);
```

**技术说明**:
- v0.5.0 使用 SQLite + [sqlite-vec](https://github.com/asg017/sqlite-vec) 扩展实现向量检索
- sqlite-vec 是官方支持的 SQLite 向量扩展，性能优秀，零配置
- v0.6.0+ 可选迁移到 PostgreSQL + pgvector（生产环境大规模部署）

**Go 实现**（使用 sqlite-vec）:

```go
// 新增：internal/agent/extensions/memory/
package memory

import (
    "context"
    "database/sql"
    "encoding/json"
    "time"

    "github.com/asg017/sqlite-vec-go-bindings/vec"
)

type MemoryStore interface {
    // Layer 1: 用户记忆
    SetMemory(ctx context.Context, req SetMemoryRequest) error
    GetMemories(ctx context.Context, userID, workspaceID string) ([]Memory, error)
    DeleteMemory(ctx context.Context, id string) error

    // Layer 2: 会话搜索（向量检索）
    SearchSessions(ctx context.Context, req SearchRequest) ([]SessionChunk, error)
    IndexSession(ctx context.Context, sessionID string, messages []Message) error

    // Layer 3: 最近会话
    GetRecentSessions(ctx context.Context, workspaceID string, limit int) ([]SessionSummary, error)
}

type SQLiteMemoryStore struct {
    db              *sql.DB
    embeddingClient EmbeddingClient  // OpenAI Embeddings API 客户端
}

// SearchSessions 使用 sqlite-vec 进行向量检索
func (m *SQLiteMemoryStore) SearchSessions(ctx context.Context, req SearchRequest) ([]SessionChunk, error) {
    // 1. 生成查询向量（调用 OpenAI Embeddings API）
    queryEmbedding, err := m.embeddingClient.GenerateEmbedding(ctx, req.Query)
    if err != nil {
        return nil, err
    }

    // 2. 使用 sqlite-vec 进行余弦相似度搜索
    query := `
        SELECT
            se.session_id,
            se.content,
            se.chunk_index,
            vec_distance_cosine(sev.embedding, ?) as distance
        FROM session_embeddings se
        JOIN session_embeddings_vec sev ON se.id = sev.rowid
        WHERE se.workspace_id = ?
          AND vec_distance_cosine(sev.embedding, ?) < ?
        ORDER BY distance ASC
        LIMIT ?
    `

    // 序列化查询向量为 BLOB
    embeddingBlob := vec.SerializeFloat32(queryEmbedding)
    maxDistance := 1.0 - req.MinScore  // 转换相似度为距离

    rows, err := m.db.QueryContext(ctx, query,
        embeddingBlob,
        req.WorkspaceID,
        embeddingBlob,
        maxDistance,
        req.Limit,
    )
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var results []SessionChunk
    for rows.Next() {
        var chunk SessionChunk
        var distance float64
        if err := rows.Scan(&chunk.SessionID, &chunk.Content, &chunk.ChunkIndex, &distance); err != nil {
            return nil, err
        }
        chunk.Score = 1.0 - distance  // 转换距离为相似度分数
        results = append(results, chunk)
    }

    return results, nil
}

// IndexSession 为会话消息生成向量索引
func (m *SQLiteMemoryStore) IndexSession(ctx context.Context, sessionID string, messages []Message) error {
    tx, err := m.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // 1. 删除旧的索引
    if _, err := tx.ExecContext(ctx, "DELETE FROM session_embeddings WHERE session_id = ?", sessionID); err != nil {
        return err
    }

    // 2. 分块并生成向量
    chunks := m.chunkMessages(messages)
    for i, chunk := range chunks {
        // 生成向量
        embedding, err := m.embeddingClient.GenerateEmbedding(ctx, chunk.Content)
        if err != nil {
            return err
        }

        // 序列化向量为 BLOB
        embeddingBlob := vec.SerializeFloat32(embedding)

        // 插入到 session_embeddings
        result, err := tx.ExecContext(ctx, `
            INSERT INTO session_embeddings (session_id, chunk_index, content, embedding, created_at)
            VALUES (?, ?, ?, ?, ?)
        `, sessionID, i, chunk.Content, embeddingBlob, time.Now().Format(time.RFC3339))
        if err != nil {
            return err
        }

        // 获取插入的 rowid
        rowid, _ := result.LastInsertId()

        // 插入到 sqlite-vec 虚拟表
        if _, err := tx.ExecContext(ctx, `
            INSERT INTO session_embeddings_vec (rowid, embedding)
            VALUES (?, ?)
        `, rowid, embeddingBlob); err != nil {
            return err
        }
    }

    return tx.Commit()
}

type Memory struct {
    ID          string
    Type        string  // "fact" | "preference" | "context"
    Key         string
    Value       string
    Source      string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type SearchRequest struct {
    Query       string
    WorkspaceID string
    Limit       int
    MinScore    float64
}

type SessionChunk struct {
    SessionID  string
    Content    string
    ChunkIndex int
    Score      float64
}

// EmbeddingClient 向量生成客户端接口
type EmbeddingClient interface {
    GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
}
```

**依赖安装**:

```bash
# services/hub/go.mod
go get github.com/asg017/sqlite-vec-go-bindings/vec
```

**sqlite-vec 扩展加载**:

```go
// services/hub/internal/infrastructure/sqlite/connection.go
package sqlite

import (
    "database/sql"
    _ "github.com/mattn/go-sqlite3"
    "github.com/asg017/sqlite-vec-go-bindings/vec"
)

func OpenDatabase(path string) (*sql.DB, error) {
    db, err := sql.Open("sqlite3", path)
    if err != nil {
        return nil, err
    }

    // 加载 sqlite-vec 扩展
    if err := vec.Load(db); err != nil {
        return nil, fmt.Errorf("failed to load sqlite-vec extension: %w", err)
    }

    return db, nil
}
```

**前端集成**:

```typescript
// apps/desktop/src/modules/memory/
export interface MemoryService {
  // Layer 1: 用户记忆管理
  setMemory(key: string, value: string, type: 'fact' | 'preference' | 'context'): Promise<void>;
  getMemories(): Promise<Memory[]>;
  deleteMemory(id: string): Promise<void>;

  // Layer 2: 会话搜索
  searchSessions(query: string): Promise<SessionSearchResult[]>;

  // Layer 3: 最近会话
  getRecentSessions(limit: number): Promise<SessionSummary[]>;
}
```

**预期效果**:
- 用户无需重复介绍自己
- 跨会话上下文连续性
- 个性化响应能力

#### 0.3 交互式工具框架

**目标**: 实现结构化输入组件，提升交互效率

**工具定义**:

```typescript
// packages/shared-core/src/tools/interactive.ts

// 结构化选择器
export interface AskUserInputTool {
  name: 'ask_user_input_v0';
  parameters: {
    questions: Array<{
      question: string;
      type: 'single_select' | 'multi_select' | 'rank_priorities';
      options: string[];
    }>;
  };
  response: {
    answers: Record<string, string | string[]>;
  };
}

// 消息草稿
export interface MessageComposeTool {
  name: 'message_compose_v1';
  parameters: {
    type: 'email' | 'textMessage' | 'other';
    subject?: string;
    body: string;
    variants?: Array<{
      label: string;
      body: string;
    }>;
  };
  response: {
    action: 'sent' | 'copied' | 'cancelled';
  };
}

// 内联图表
export interface ChartDisplayTool {
  name: 'chart_display_v0';
  parameters: {
    series: Array<{
      name: string;
      data: Array<{ x: number | string; y: number }>;
    }>;
    style: 'line' | 'bar' | 'scatter';
    title?: string;
  };
}
```

**Vue 组件实现**:

```vue
<!-- apps/desktop/src/modules/session/components/InteractiveTools/ -->

<!-- AskUserInputWidget.vue -->
<template>
  <div class="interactive-widget">
    <div v-for="(q, idx) in questions" :key="idx">
      <p>{{ q.question }}</p>

      <!-- 单选 -->
      <div v-if="q.type === 'single_select'" class="options">
        <button v-for="opt in q.options" @click="selectOption(idx, opt)">
          {{ opt }}
        </button>
      </div>

      <!-- 多选 -->
      <div v-else-if="q.type === 'multi_select'" class="options">
        <label v-for="opt in q.options">
          <input type="checkbox" :value="opt" v-model="answers[idx]" />
          {{ opt }}
        </label>
      </div>

      <!-- 排序 -->
      <draggable v-else-if="q.type === 'rank_priorities'" v-model="answers[idx]">
        <div v-for="opt in q.options" :key="opt">{{ opt }}</div>
      </draggable>
    </div>

    <button @click="submit">Submit</button>
    <button @click="skip">Skip</button>
  </div>
</template>
```

**预期效果**:
- 信息收集效率提升 3-5x（单次交互 vs 多轮对话）
- 支持复杂决策场景（优先级排序、多维度选择）
- 用户体验现代化

---

### 领域 1: 后端持久化与状态管理

#### 1.1 目标架构

**从内存 Map 迁移到 Repository + Database**

```go
// 新架构：领域聚合根
package domain

type Workspace struct {
    ID        WorkspaceID
    Mode      WorkspaceMode
    HubURL    string
    CreatedAt time.Time
    UpdatedAt time.Time
}

type Session struct {
    ID          SessionID
    WorkspaceID WorkspaceID
    ProjectID   ProjectID
    QueueState  QueueState
    CreatedAt   time.Time
}

// Repository 接口（在 domain 层定义）
type WorkspaceRepository interface {
    Save(ctx context.Context, ws *Workspace) error
    FindByID(ctx context.Context, id WorkspaceID) (*Workspace, error)
    List(ctx context.Context, filter WorkspaceFilter) ([]*Workspace, error)
}

type SessionRepository interface {
    Save(ctx context.Context, session *Session) error
    FindByID(ctx context.Context, id SessionID) (*Session, error)
    ListByWorkspace(ctx context.Context, wsID WorkspaceID) ([]*Session, error)
}
```

#### 1.2 数据库 Schema 设计

```sql
-- 工作区
CREATE TABLE workspaces (
    id TEXT PRIMARY KEY,
    mode TEXT NOT NULL CHECK (mode IN ('local', 'remote')),
    hub_url TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 会话（统一命名为 Session）
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    project_id TEXT NOT NULL,
    queue_state TEXT NOT NULL CHECK (queue_state IN ('idle', 'running', 'queued')),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 运行（Run）
CREATE TABLE runs (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    state TEXT NOT NULL,
    tokens_in INTEGER DEFAULT 0,
    tokens_out INTEGER DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- 事件日志（事件溯源）
CREATE TABLE run_events (
    id BIGSERIAL PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    session_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_run_events_run_id ON run_events(run_id);
CREATE INDEX idx_run_events_session_id ON run_events(session_id);
```

#### 1.3 细粒度锁策略

**问题**: 当前单一 `sync.RWMutex` 锁住所有操作

**解决方案**: 按聚合根分离锁

```go
type SessionRepository struct {
    db *sql.DB
    sessionLocks sync.Map // map[SessionID]*sync.RWMutex
}

func (r *SessionRepository) lockSession(id SessionID) *sync.RWMutex {
    lock, _ := r.sessionLocks.LoadOrStore(id, &sync.RWMutex{})
    return lock.(*sync.RWMutex)
}
```

#### 1.4 事件总线架构

```go
type EventBus interface {
    Publish(ctx context.Context, event Event) error
    Subscribe(ctx context.Context, filter EventFilter) (<-chan Event, error)
}

type Event struct {
    ID        string
    Type      string
    SessionID string
    Payload   json.RawMessage
    Timestamp time.Time
}
```

---

### 领域 2: 前端状态管理重构

#### 2.1 统一命名：Session 替代 Conversation

**当前问题**: 双重命名导致认知负担

**重构策略**:
1. 全局搜索替换 `Conversation` → `Session`
2. 删除所有别名导出（`answerConversationExecutionQuestion as answerSessionRunQuestion`）
3. 更新 API 路径（保持后端兼容，前端统一使用 Session 术语）

**影响文件**:
- `apps/desktop/src/modules/session/store/index.ts` - 删除 14 个别名导出
- `apps/desktop/src/modules/session/services/index.ts` - 统一函数命名
- `apps/desktop/src/modules/session/views/*.ts` - 更新所有引用

#### 2.2 消除镜像投影

**当前问题**: `state.ts` 中的重复字段

```typescript
// ❌ 删除这些镜像字段
type SessionState = {
  byConversationId: Record<string, SessionRuntime>;  // 删除
  timers: Record<string, ReturnType<typeof setTimeout>>;  // 删除
  streams: Record<string, StreamHandle>;  // 删除
}

type SessionRuntime = {
  executions: Run[];  // 删除
}
```

**新架构**:

```typescript
type SessionState = {
  bySessionId: Record<string, SessionRuntime>;
  sessionTimers: Record<string, ReturnType<typeof setTimeout>>;
  sessionStreams: Record<string, StreamHandle>;
}

type SessionRuntime = {
  runs: Run[];  // 唯一数据源
  messages: SessionMessage[];
  events: RunLifecycleEvent[];  // 限制最大 200 条
  changeSet: ChangeSet | null;
}
```

#### 2.3 Repository 模式引入

**目标**: 解耦 Store 与 HTTP 客户端

```typescript
// 新增：repositories/SessionRepository.ts
export interface SessionRepository {
  getSession(id: string): Promise<Session>;
  listSessions(workspaceId: string): Promise<Session[]>;
  createSession(request: CreateSessionRequest): Promise<Session>;
  deleteSession(id: string): Promise<void>;
}

export class HttpSessionRepository implements SessionRepository {
  constructor(private client: ApiClient) {}

  async getSession(id: string): Promise<Session> {
    return this.client.get<Session>(`/v1/sessions/${id}`);
  }
}

// Store 使用 Repository
import { sessionRepository } from '@/shared/repositories';

export async function loadSession(id: string): Promise<void> {
  try {
    const session = await sessionRepository.getSession(id);
    sessionStore.bySessionId[id] = hydrateSessionRuntime(session);
  } catch (error) {
    handleError(error);
  }
}
```

#### 2.4 修复 SSE 事件丢失

**当前问题**: 重连期间静默丢弃事件

**解决方案**: 事件队列缓冲

```typescript
// 新增：stream/eventQueue.ts
class SessionEventQueue {
  private queue: RunLifecycleEvent[] = [];
  private processing = false;

  enqueue(event: RunLifecycleEvent): void {
    this.queue.push(event);
    this.processQueue();
  }

  private async processQueue(): Promise<void> {
    if (this.processing) return;
    this.processing = true;

    while (this.queue.length > 0) {
      const event = this.queue.shift()!;
      await applyIncomingExecutionEvent(event.sessionId, event);
    }

    this.processing = false;
  }
}

// 修改：stream.ts
export function attachSessionStream(session: Session, token?: string): void {
  const eventQueue = new SessionEventQueue();
  let resyncInFlight = false;

  sessionStore.sessionStreams[session.id] = streamSessionEvents(session.id, {
    onEvent: (event) => {
      if (isSSEBackfillResyncEvent(event)) {
        resyncInFlight = true;
        void getSessionDetail(session.id, { token })
          .finally(() => { resyncInFlight = false; });
      }
      // ✅ 所有事件进入队列，不再丢弃
      eventQueue.enqueue(event);
    }
  });
}
```

#### 2.5 防止竞态条件

**问题**: 流附加的 check-then-act 竞态

**解决方案**: 原子操作

```typescript
// 新增：stream/streamRegistry.ts
class StreamRegistry {
  private streams = new Map<string, StreamHandle>();
  private locks = new Map<string, Promise<void>>();

  async attachStream(sessionId: string, factory: () => StreamHandle): Promise<StreamHandle> {
    // 等待已有锁
    const existingLock = this.locks.get(sessionId);
    if (existingLock) {
      await existingLock;
    }

    // 检查是否已存在
    const existing = this.streams.get(sessionId);
    if (existing) return existing;

    // 创建新锁
    const lock = (async () => {
      const stream = factory();
      this.streams.set(sessionId, stream);
    })();

    this.locks.set(sessionId, lock);
    await lock;
    this.locks.delete(sessionId);

    return this.streams.get(sessionId)!;
  }
}
```

---

### 领域 4: Agent 运行时优化

#### 4.1 Engine 持久化层

**目标**: 解决内存状态管理问题，支持 Hub 重启后状态恢复

**方案**: 复用领域 1 的 Repository 架构

```go
// 扩展 domain 层
type RunRepository interface {
    Save(ctx context.Context, run *Run) error
    FindByID(ctx context.Context, id RunID) (*Run, error)
    ListBySession(ctx context.Context, sessionID SessionID) ([]*Run, error)
    UpdateState(ctx context.Context, runID RunID, state RunState) error
}

// Engine 使用 Repository
type Engine struct {
    mu           sync.Mutex
    sessionRepo  domain.SessionRepository
    runRepo      domain.RunRepository
    eventStore   *transportevents.Store

    // 内存缓存（可选，用于热数据）
    sessionCache map[core.SessionID]*sessionRuntime
    runCache     map[core.RunID]*runRuntime
}

func (e *Engine) Submit(ctx context.Context, sessionID string, input core.UserInput) (runID string, err error) {
    // 1. 从 Repository 加载 Session
    session, err := e.sessionRepo.FindByID(ctx, core.SessionID(sessionID))
    if err != nil {
        return "", err
    }

    // 2. 创建 Run 并持久化
    run := domain.NewRun(session.ID, input)
    if err := e.runRepo.Save(ctx, run); err != nil {
        return "", err
    }

    // 3. 更新内存缓存
    e.mu.Lock()
    e.runCache[run.ID] = hydrateRunRuntime(run)
    e.mu.Unlock()

    return string(run.ID), nil
}
```

**数据库 Schema** (复用领域 1):
```sql
CREATE TABLE runs (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES sessions(id),
    state TEXT NOT NULL,
    input_text TEXT NOT NULL,
    output_text TEXT,
    created_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    INDEX idx_runs_session_id (session_id),
    INDEX idx_runs_state (state)
);

CREATE TABLE run_events (
    id BIGSERIAL PRIMARY KEY,
    session_id TEXT NOT NULL,
    run_id TEXT NOT NULL,
    sequence INTEGER NOT NULL,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    UNIQUE(session_id, sequence)
);
```

#### 4.2 细粒度锁优化

**目标**: 消除全局锁竞争，提升并发性能

**方案**: 按 Session 分离锁

```go
type Engine struct {
    sessionsMu sync.RWMutex
    sessions   map[core.SessionID]*sessionRuntime

    runsMu sync.RWMutex
    runs   map[core.RunID]*runRuntime
}

type sessionRuntime struct {
    mu sync.Mutex  // per-session 锁
    id core.SessionID
    queue []core.RunID
    active core.RunID
}

func (e *Engine) Submit(ctx context.Context, sessionID string, input core.UserInput) (runID string, err error) {
    normalizedSessionID := core.SessionID(sessionID)

    // 1. 读锁查找 Session
    e.sessionsMu.RLock()
    session, exists := e.sessions[normalizedSessionID]
    e.sessionsMu.RUnlock()

    if !exists {
        return "", core.ErrSessionNotFound
    }

    // 2. 仅锁定目标 Session
    session.mu.Lock()
    defer session.mu.Unlock()

    // 3. 创建 Run（不需要全局锁）
    newRunID := e.generateRunID()
    run := &runRuntime{...}

    // 4. 写锁添加 Run
    e.runsMu.Lock()
    e.runs[newRunID] = run
    e.runsMu.Unlock()

    session.queue = append(session.queue, newRunID)
    return string(newRunID), nil
}
```

**预期效果**:
- Session 创建/查询并行化
- Run 提交仅锁定目标 Session
- 并发性能提升 5-10x

#### 4.3 完善核心接口实现

**目标**: 实现所有 `core/interfaces.go` 定义的接口

**CommandBus 实现**:
```go
package slash

type Bus struct {
    registry map[string]CommandHandler
}

func (b *Bus) Execute(ctx context.Context, sessionID string, cmd core.SlashCommand) (core.CommandResponse, error) {
    handler, exists := b.registry[cmd.Name]
    if !exists {
        return core.CommandResponse{}, fmt.Errorf("unknown command: %s", cmd.Name)
    }
    return handler.Execute(ctx, sessionID, cmd)
}

// 注册内置命令
func NewBus() *Bus {
    bus := &Bus{registry: make(map[string]CommandHandler)}
    bus.Register("compact", &CompactHandler{})
    bus.Register("help", &HelpHandler{})
    return bus
}
```

**ToolExecutor 实现**:
```go
package tools

type Executor struct {
    hookDispatcher core.HookDispatcher
    permissionGate core.PermissionGate
    approvalRouter *approval.Router
    toolRegistry   map[string]ToolHandler
}

func (e *Executor) Execute(ctx context.Context, call core.ToolCall) (core.ToolResult, error) {
    // 1. PreToolUse hook
    hookDecision, _ := e.hookDispatcher.Dispatch(ctx, core.HookEvent{
        Type: "PreToolUse",
        Payload: map[string]any{"tool": call.ToolName},
    })

    // 2. Permission evaluation
    permDecision, _ := e.permissionGate.Evaluate(ctx, core.PermissionRequest{
        ToolName: call.ToolName,
    })

    // 3. Approval wait (if needed)
    if permDecision.Kind == core.PermissionDecisionAsk {
        action, _ := e.approvalRouter.WaitForApproval(ctx, call.RunID)
        if action != core.ControlActionApprove {
            return core.ToolResult{Error: &core.RunError{Code: "denied"}}, nil
        }
    }

    // 4. Tool execution
    handler := e.toolRegistry[call.ToolName]
    result, err := handler.Execute(ctx, call)

    // 5. PostToolUse hook
    e.hookDispatcher.Dispatch(ctx, core.HookEvent{
        Type: "PostToolUse",
        Payload: map[string]any{"tool": call.ToolName, "result": result},
    })

    return result, err
}
```

#### 4.4 统一权限评估流程

**目标**: 单一权限决策路径，提升可审计性

**方案**: 统一权限门控

```go
type UnifiedPermissionGate struct {
    ruleGate       *policy.Gate
    hookDispatcher core.HookDispatcher
    auditLogger    AuditLogger
}

func (g *UnifiedPermissionGate) Evaluate(ctx context.Context, req core.PermissionRequest) (core.PermissionDecision, error) {
    traceID := generateTraceID()

    // 1. PreToolUse hook (可修改 req)
    hookDecision, _ := g.hookDispatcher.Dispatch(ctx, core.HookEvent{
        Type: "PreToolUse",
        Payload: map[string]any{
            "tool": req.ToolName,
            "trace_id": traceID,
        },
    })

    // 2. Hook decision 优先级最高
    if hookDecision.Decision == "deny" {
        g.auditLogger.Log(AuditEntry{
            TraceID: traceID,
            Decision: "deny",
            Reason: "blocked by hook",
            HookID: hookDecision.MatchedPolicyID,
        })
        return core.PermissionDecision{
            Kind: core.PermissionDecisionDeny,
            Reason: "blocked by hook: " + hookDecision.MatchedPolicyID,
        }, nil
    }

    // 3. Rule-based evaluation
    decision, _ := g.ruleGate.Evaluate(ctx, req)

    // 4. Audit logging
    g.auditLogger.Log(AuditEntry{
        TraceID: traceID,
        Tool: req.ToolName,
        Decision: string(decision.Kind),
        Reason: decision.Reason,
        MatchedRule: decision.MatchedRule,
    })

    return decision, nil
}
```

#### 4.5 Subscriber 背压监控

**目标**: 暴露事件丢失指标，提升可观测性

**方案**: 添加监控指标

```go
type SubscriberManager struct {
    subscribers []*subscriber
    metrics     *Metrics
}

func (m *SubscriberManager) Publish(ctx context.Context, event core.EventEnvelope) error {
    dropped := 0
    for _, sub := range m.subscribers {
        select {
        case sub.ch <- event:
            // 成功发送
        default:
            // 背压触发，丢弃事件
            dropped++
            m.metrics.RecordCounter("subscriber.events.dropped", 1, map[string]string{
                "session_id": string(event.SessionID),
                "event_type": string(event.Type),
            })
        }
    }

    if dropped > 0 {
        m.metrics.RecordCounter("subscriber.backpressure.triggered", 1)
    }

    return nil
}
```

**监控告警**:
```yaml
# Prometheus 告警规则
- alert: SubscriberEventsDropped
  expr: rate(subscriber_events_dropped_total[5m]) > 0
  for: 1m
  annotations:
    summary: "SSE 事件丢失检测"
    description: "Session {{ $labels.session_id }} 在过去 5 分钟内丢失了事件"
```

---

### 领域 5: 资源配置继承与实时感知

#### 5.1 资源配置继承机制

**目标**: 建立清晰的 Workspace -> Project -> Session 三层资源继承与验证机制

**数据库 Schema**（SQLite，v3.0 调整为软删除）:

```sql
-- 资源配置版本控制（添加软删除字段）
CREATE TABLE resource_configs (
    id TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('model', 'rule', 'skill', 'mcp')),
    name TEXT NOT NULL,
    config TEXT NOT NULL,  -- SQLite 使用 TEXT 存储 JSON
    scope TEXT NOT NULL CHECK (scope IN ('private', 'shared')),
    version INTEGER NOT NULL DEFAULT 1,
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_deleted BOOLEAN NOT NULL DEFAULT false,  -- ✅ 软删除标记
    deleted_at TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_resource_configs_workspace ON resource_configs(workspace_id, type, is_active, is_deleted);

-- 项目资源绑定（白名单）
CREATE TABLE project_resource_bindings (
    project_id TEXT NOT NULL,
    resource_config_id TEXT NOT NULL,
    resource_version INTEGER NOT NULL,
    is_default BOOLEAN NOT NULL DEFAULT false,
    bound_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (project_id, resource_config_id),
    FOREIGN KEY (resource_config_id) REFERENCES resource_configs(id) ON DELETE CASCADE
);

-- Session 资源快照（记录创建时的资源版本 + deprecated 标记）
CREATE TABLE session_resource_snapshots (
    session_id TEXT NOT NULL,
    resource_config_id TEXT NOT NULL,
    resource_version INTEGER NOT NULL,
    is_deprecated BOOLEAN NOT NULL DEFAULT false,  -- ✅ 资源已废弃
    fallback_resource_id TEXT,  -- ✅ 降级到的资源 ID
    snapshot_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (session_id, resource_config_id)
);
```

**Go 实现**:

```go
// 新增：internal/domain/resource.go
package domain

type ResourceConfigService struct {
    resourceRepo ResourceConfigRepository
    projectRepo  ProjectRepository
    sessionRepo  SessionRepository
    eventBus     EventBus
}

// ValidateSessionResources 验证 Session 资源是否在项目白名单内
func (s *ResourceConfigService) ValidateSessionResources(ctx context.Context, sessionID SessionID) error {
    session, err := s.sessionRepo.FindByID(ctx, sessionID)
    if err != nil {
        return err
    }

    projectConfig, err := s.projectRepo.GetConfig(ctx, session.ProjectID)
    if err != nil {
        return err
    }

    // 验证 model
    if !contains(projectConfig.ModelConfigIDs, session.ModelConfigID) {
        return fmt.Errorf("model %s not allowed in project %s", session.ModelConfigID, session.ProjectID)
    }

    // 验证 rules/skills/mcps
    for _, ruleID := range session.RuleIDs {
        if !contains(projectConfig.RuleIDs, ruleID) {
            return fmt.Errorf("rule %s not allowed in project %s", ruleID, session.ProjectID)
        }
    }

    return nil
}

// OnResourceDeleted 资源删除时的软删除处理（v3.0 调整）
func (s *ResourceConfigService) OnResourceDeleted(ctx context.Context, resourceID string, resourceType string) error {
    // 1. 软删除资源配置
    resource, err := s.resourceRepo.FindByID(ctx, resourceID)
    if err != nil {
        return err
    }

    resource.IsDeleted = true
    resource.DeletedAt = time.Now()
    if err := s.resourceRepo.Save(ctx, resource); err != nil {
        return err
    }

    // 2. 查找所有使用该资源的 Session
    affectedSessions, err := s.sessionRepo.FindByResourceID(ctx, resourceID)
    if err != nil {
        return err
    }

    // 3. 标记资源为 deprecated，自动降级（不删除 Session）
    for _, session := range affectedSessions {
        project, _ := s.projectRepo.FindByID(ctx, session.ProjectID)

        var fallbackResourceID string
        switch resourceType {
        case "model":
            if session.ModelConfigID == resourceID {
                fallbackResourceID = project.DefaultModelConfigID
                session.ModelConfigID = fallbackResourceID
            }
        case "rule":
            session.RuleIDs = removeID(session.RuleIDs, resourceID)
        case "skill":
            session.SkillIDs = removeID(session.SkillIDs, resourceID)
        case "mcp":
            session.MCPIDs = removeID(session.MCPIDs, resourceID)
        }

        // ✅ 保存 Session（不删除）
        if err := s.sessionRepo.Save(ctx, session); err != nil {
            return err
        }

        // ✅ 更新资源快照
        snapshot := SessionResourceSnapshot{
            SessionID:          session.ID,
            ResourceConfigID:   resourceID,
            IsDeprecated:       true,
            FallbackResourceID: fallbackResourceID,
        }
        s.resourceRepo.SaveSnapshot(ctx, snapshot)

        // ✅ 推送警告事件（不删除 Session）
        s.eventBus.Publish(ctx, Event{
            Type:      "resource.deprecated",
            SessionID: string(session.ID),
            Payload: map[string]any{
                "resource_id":   resourceID,
                "resource_type": resourceType,
                "fallback_to":   fallbackResourceID,
                "action":        "auto_switched",
            },
        })
    }

    return nil
}
```

#### 5.2 实时资源变更推送

**目标**: 通过 SSE 推送资源配置变更事件到已连接客户端

**后端事件定义**:

```go
// 新增事件类型
const (
    RunEventTypeResourceConfigChanged = "resource_config.changed"
    RunEventTypeResourceConfigDeleted = "resource_config.deleted"
    RunEventTypeProjectConfigChanged  = "project_config.changed"
)

// 资源变更事件
type ResourceConfigChangedEvent struct {
    ResourceID   string
    ResourceType string  // "model" | "rule" | "skill" | "mcp"
    Version      int
    IsActive     bool
    ChangedAt    string
}

// 推送到所有相关 Session
func (s *ResourceConfigService) NotifyResourceChange(ctx context.Context, resourceID string) error {
    affectedSessions, _ := s.sessionRepo.FindByResourceID(ctx, resourceID)

    for _, session := range affectedSessions {
        s.eventBus.Publish(ctx, Event{
            Type:      RunEventTypeResourceConfigChanged,
            SessionID: string(session.ID),
            Payload: map[string]any{
                "resource_id": resourceID,
                "version":     resource.Version,
            },
        })
    }

    return nil
}
```

**前端处理**（v3.0 调整为软删除）:

```typescript
// apps/desktop/src/modules/session/store/stream.ts

export function attachSessionStream(session: Session, token?: string): void {
  sessionStore.sessionStreams[session.id] = streamSessionEvents(session.id, {
    onEvent: (event) => {
      // ✅ 处理资源废弃事件（不删除 Session）
      if (event.type === 'resource.deprecated') {
        handleResourceDeprecated(session.id, event.payload);
      }
      // ... 其他事件处理
    }
  });
}

function handleResourceDeprecated(sessionId: string, payload: any): void {
  const runtime = sessionStore.bySessionId[sessionId];
  if (!runtime) return;

  // ✅ 更新本地资源配置
  if (payload.resource_type === 'model') {
    runtime.modelId = payload.fallback_to;
  }

  // ✅ 显示警告通知（不关闭会话）
  notificationStore.add({
    type: 'warning',
    title: '资源配置已更新',
    message: `${payload.resource_type} 资源已被管理员禁用，已自动切换到默认配置`,
    duration: 5000,
    actions: [
      { label: '查看详情', onClick: () => showResourceDetails(payload.resource_id) }
    ]
  });
}
```

**预期效果**:
- 管理员禁用模型后，所有使用该模型的会话自动切换到默认模型
- 用户收到明确的警告通知，但会话继续可用
- 资源配置变更可审计（事件日志）
- 避免用户数据意外丢失

---

### 领域 6: 回滚点(Checkpoint)完整实现

#### 6.1 统一 Checkpoint 抽象

**目标**: 支持 Git 和非 Git 项目的统一回滚机制

**数据库 Schema**（SQLite，v3.0 调整）:

```sql
-- 统一 Checkpoint 表
CREATE TABLE checkpoints (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    checkpoint_type TEXT NOT NULL CHECK (checkpoint_type IN ('git_commit', 'file_snapshot', 'file_snapshot_compacted', 'message_boundary')),
    message TEXT NOT NULL,
    parent_id TEXT,  -- 链式引用，用于增量恢复
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,

    -- Git 项目字段
    git_commit_id TEXT,
    git_branch TEXT,

    -- 非 Git 项目字段
    entries_digest TEXT,

    -- Session 状态快照（JSON 格式）
    session_state TEXT NOT NULL,

    FOREIGN KEY (parent_id) REFERENCES checkpoints(id) ON DELETE SET NULL
);

CREATE INDEX idx_checkpoints_session ON checkpoints(session_id, created_at DESC);

-- Checkpoint 文件快照（增量存储）
CREATE TABLE checkpoint_file_snapshots (
    checkpoint_id TEXT NOT NULL,
    file_path TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    content BLOB,  -- 仅变更文件有内容
    change_type TEXT NOT NULL CHECK (change_type IN ('added', 'modified', 'deleted')),
    PRIMARY KEY (checkpoint_id, file_path),
    FOREIGN KEY (checkpoint_id) REFERENCES checkpoints(id) ON DELETE CASCADE
);
```

**Go 实现**:

```go
// 新增：internal/domain/checkpoint.go
package domain

type CheckpointService struct {
    checkpointRepo CheckpointRepository
    sessionRepo    SessionRepository
    projectRepo    ProjectRepository
}

// CreateCheckpoint 创建 Checkpoint（自动选择策略）
func (s *CheckpointService) CreateCheckpoint(ctx context.Context, sessionID SessionID, message string) (*Checkpoint, error) {
    session, _ := s.sessionRepo.FindByID(ctx, sessionID)
    project, _ := s.projectRepo.FindByID(ctx, session.ProjectID)

    // 快照当前 Session 状态
    sessionState := SessionStateSnapshot{
        ModelConfigID: session.ModelConfigID,
        RuleIDs:       session.RuleIDs,
        SkillIDs:      session.SkillIDs,
        MCPIDs:        session.MCPIDs,
        MessageCount:  len(session.Messages),
        RunCount:      len(session.Runs),
    }

    var checkpoint *Checkpoint
    var err error

    if project.IsGit {
        checkpoint, err = s.createGitCheckpoint(ctx, session, project, message, sessionState)
    } else {
        checkpoint, err = s.createFileSnapshotCheckpoint(ctx, session, project, message, sessionState)
    }

    return checkpoint, err
}

// RollbackToCheckpoint 回滚到指定 Checkpoint
func (s *CheckpointService) RollbackToCheckpoint(ctx context.Context, checkpointID string) error {
    checkpoint, err := s.checkpointRepo.FindByID(ctx, checkpointID)
    if err != nil {
        return err
    }

    session, _ := s.sessionRepo.FindByID(ctx, SessionID(checkpoint.SessionID))
    project, _ := s.projectRepo.FindByID(ctx, session.ProjectID)

    // 1. 回滚文件系统
    if checkpoint.Type == "git_commit" {
        if err := s.rollbackGitCommit(project, checkpoint.GitCommitID); err != nil {
            return err
        }
    } else {
        if err := s.rollbackFileSnapshot(checkpoint); err != nil {
            return err
        }
    }

    // 2. 回滚 Session 状态
    sessionState := checkpoint.SessionState
    session.ModelConfigID = sessionState.ModelConfigID
    session.RuleIDs = sessionState.RuleIDs
    session.SkillIDs = sessionState.SkillIDs
    session.MCPIDs = sessionState.MCPIDs

    // 3. 截断消息和 Run 历史
    session.Messages = session.Messages[:sessionState.MessageCount]
    session.Runs = session.Runs[:sessionState.RunCount]

    if err := s.sessionRepo.Save(ctx, session); err != nil {
        return err
    }

    return nil
}

func (s *CheckpointService) createFileSnapshotCheckpoint(ctx context.Context, session *Session, project *Project, message string, state SessionStateSnapshot) (*Checkpoint, error) {
    // 1. 获取上一个 Checkpoint（用于增量对比）
    lastCheckpoint, err := s.checkpointRepo.GetLatestBySession(ctx, session.ID)
    var lastFiles map[string]FileInfo
    if err == nil && lastCheckpoint != nil {
        lastFiles = s.loadCheckpointFiles(ctx, lastCheckpoint.ID)
    } else {
        lastFiles = make(map[string]FileInfo)
    }

    // 2. 读取当前项目文件
    currentFiles, err := s.readProjectFiles(project.RepoPath)
    if err != nil {
        return nil, err
    }

    // 3. 计算增量变更（仅保存变更的文件）
    changedFiles := []FileInfo{}
    for path, currentFile := range currentFiles {
        lastFile, exists := lastFiles[path]
        if !exists || lastFile.Hash != currentFile.Hash {
            // 文件新增或内容变更
            changedFiles = append(changedFiles, currentFile)
        }
    }

    // 4. 记录删除的文件
    deletedPaths := []string{}
    for path := range lastFiles {
        if _, exists := currentFiles[path]; !exists {
            deletedPaths = append(deletedPaths, path)
        }
    }

    // 5. 计算内容哈希（用于快速对比）
    digest := computeFilesDigest(currentFiles)

    // 6. 保存 Checkpoint 元数据
    checkpoint := &Checkpoint{
        ID:            "ckpt_" + randomHex(8),
        SessionID:     string(session.ID),
        Type:          "file_snapshot",
        Message:       message,
        EntriesDigest: digest,
        SessionState:  state,
        ParentID:      lastCheckpoint.ID,  // 链式引用
        CreatedAt:     time.Now(),
    }

    if err := s.checkpointRepo.Save(ctx, checkpoint); err != nil {
        return nil, err
    }

    // 7. 仅保存变更的文件内容
    for _, file := range changedFiles {
        snapshot := FileSnapshot{
            CheckpointID: checkpoint.ID,
            FilePath:     file.Path,
            ContentHash:  file.Hash,
            Content:      file.Content,
            ChangeType:   "modified",  // "added" | "modified"
        }
        s.checkpointRepo.SaveFileSnapshot(ctx, snapshot)
    }

    // 8. 记录删除的文件路径（不保存内容）
    for _, path := range deletedPaths {
        snapshot := FileSnapshot{
            CheckpointID: checkpoint.ID,
            FilePath:     path,
            ChangeType:   "deleted",
        }
        s.checkpointRepo.SaveFileSnapshot(ctx, snapshot)
    }

    return checkpoint, nil
}

// RollbackToCheckpoint 回滚到指定 Checkpoint（增量恢复）
func (s *CheckpointService) RollbackToCheckpoint(ctx context.Context, checkpointID string) error {
    checkpoint, err := s.checkpointRepo.FindByID(ctx, checkpointID)
    if err != nil {
        return err
    }

    session, _ := s.sessionRepo.FindByID(ctx, SessionID(checkpoint.SessionID))
    project, _ := s.projectRepo.FindByID(ctx, session.ProjectID)

    // 1. 回滚文件系统（增量恢复）
    if checkpoint.Type == "git_commit" {
        if err := s.rollbackGitCommit(project, checkpoint.GitCommitID); err != nil {
            return err
        }
    } else {
        // 增量恢复：从目标 Checkpoint 向前追溯到根 Checkpoint
        if err := s.rollbackFileSnapshotIncremental(ctx, checkpoint, project); err != nil {
            return err
        }
    }

    // 2. 回滚 Session 状态
    sessionState := checkpoint.SessionState
    session.ModelConfigID = sessionState.ModelConfigID
    session.RuleIDs = sessionState.RuleIDs
    session.SkillIDs = sessionState.SkillIDs
    session.MCPIDs = sessionState.MCPIDs

    // 3. 截断消息和 Run 历史
    session.Messages = session.Messages[:sessionState.MessageCount]
    session.Runs = session.Runs[:sessionState.RunCount]

    if err := s.sessionRepo.Save(ctx, session); err != nil {
        return err
    }

    return nil
}

// rollbackFileSnapshotIncremental 增量恢复文件系统
func (s *CheckpointService) rollbackFileSnapshotIncremental(ctx context.Context, targetCheckpoint *Checkpoint, project *Project) error {
    // 1. 构建 Checkpoint 链（从目标向前追溯）
    checkpointChain := []*Checkpoint{targetCheckpoint}
    currentID := targetCheckpoint.ParentID
    for currentID != "" {
        parent, err := s.checkpointRepo.FindByID(ctx, currentID)
        if err != nil {
            break
        }
        checkpointChain = append([]*Checkpoint{parent}, checkpointChain...)
        currentID = parent.ParentID
    }

    // 2. 清空项目目录（保留 .git 等隐藏文件）
    if err := s.cleanProjectFiles(project.RepoPath); err != nil {
        return err
    }

    // 3. 按顺序应用每个 Checkpoint 的变更
    fileStates := make(map[string]FileSnapshot)
    for _, cp := range checkpointChain {
        snapshots, _ := s.checkpointRepo.GetFileSnapshots(ctx, cp.ID)
        for _, snapshot := range snapshots {
            switch snapshot.ChangeType {
            case "added", "modified":
                fileStates[snapshot.FilePath] = snapshot
            case "deleted":
                delete(fileStates, snapshot.FilePath)
            }
        }
    }

    // 4. 写入最终文件状态
    for path, snapshot := range fileStates {
        fullPath := filepath.Join(project.RepoPath, path)
        if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
            return err
        }
        if err := os.WriteFile(fullPath, snapshot.Content, 0644); err != nil {
            return err
        }
    }

    return nil
}
```

**API 端点**:

```yaml
# packages/contracts/openapi.yaml

/v1/sessions/{session_id}/checkpoints:
  post:
    summary: 创建 Checkpoint
    requestBody:
      content:
        application/json:
          schema:
            type: object
            properties:
              message:
                type: string
    responses:
      200:
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/Checkpoint'

/v1/sessions/{session_id}/checkpoints/{checkpoint_id}/rollback:
  post:
    summary: 回滚到 Checkpoint
    responses:
      200:
        content:
          application/json:
            schema:
              type: object
              properties:
                ok:
                  type: boolean
                session:
                  $ref: '#/components/schemas/Session'
```

**预期效果**:
- Git 和非 Git 项目统一回滚体验
- 支持"撤销最近 3 条消息"的时间旅行
- Session 状态完整恢复（资源绑定 + 消息历史）

#### 6.2 Checkpoint 健壮性增强（v3.0 新增）

**目标**: 处理 Checkpoint 链断裂、链过长、文件快照损坏等边界条件

**边界条件处理**:

1. **Checkpoint 链断裂**: 如果中间某个 Checkpoint 被删除，回退到完整快照策略
2. **Checkpoint 链过长**: 超过 100 个 Checkpoint 时，自动触发压缩
3. **文件快照损坏**: 验证内容哈希，损坏时拒绝回滚

**Go 实现**:

```go
// buildCheckpointChain 构建 Checkpoint 链，处理断裂情况
func (s *CheckpointService) buildCheckpointChain(ctx context.Context, target *Checkpoint) ([]*Checkpoint, error) {
    chain := []*Checkpoint{target}
    currentID := target.ParentID

    for currentID != "" {
        parent, err := s.checkpointRepo.FindByID(ctx, currentID)
        if err != nil {
            // ✅ 链断裂，返回错误
            return nil, fmt.Errorf("checkpoint chain broken at %s: %w", currentID, err)
        }
        chain = append([]*Checkpoint{parent}, chain...)
        currentID = parent.ParentID
    }

    return chain, nil
}

// validateCheckpointChain 验证链的完整性
func (s *CheckpointService) validateCheckpointChain(chain []*Checkpoint) error {
    for i := 1; i < len(chain); i++ {
        if chain[i].ParentID != chain[i-1].ID {
            return fmt.Errorf("checkpoint chain inconsistent at index %d", i)
        }
    }
    return nil
}

// compactCheckpoints 压缩 Checkpoint 链
func (s *CheckpointService) compactCheckpoints(ctx context.Context, chain []*Checkpoint) (*Checkpoint, error) {
    // 1. 合并前 50 个 Checkpoint 的文件快照
    mergedSnapshots := make(map[string]FileSnapshot)

    for _, cp := range chain[:50] {
        snapshots, _ := s.checkpointRepo.GetFileSnapshots(ctx, cp.ID)
        for _, snapshot := range snapshots {
            switch snapshot.ChangeType {
            case "added", "modified":
                mergedSnapshots[snapshot.FilePath] = snapshot
            case "deleted":
                delete(mergedSnapshots, snapshot.FilePath)
            }
        }
    }

    // 2. 创建压缩后的 Checkpoint
    compactedCheckpoint := &Checkpoint{
        ID:            "ckpt_compact_" + randomHex(8),
        SessionID:     chain[0].SessionID,
        Type:          "file_snapshot_compacted",
        Message:       "Compacted from " + chain[0].ID + " to " + chain[49].ID,
        SessionState:  chain[49].SessionState,
        CreatedAt:     time.Now(),
    }

    if err := s.checkpointRepo.Save(ctx, compactedCheckpoint); err != nil {
        return nil, err
    }

    // 3. 保存合并后的文件快照
    for _, snapshot := range mergedSnapshots {
        snapshot.CheckpointID = compactedCheckpoint.ID
        s.checkpointRepo.SaveFileSnapshot(ctx, snapshot)
    }

    // 4. 删除旧的 Checkpoint
    for _, cp := range chain[:50] {
        s.checkpointRepo.Delete(ctx, cp.ID)
    }

    return compactedCheckpoint, nil
}

// validateFileSnapshot 验证文件快照完整性
func (s *CheckpointService) validateFileSnapshot(snapshot FileSnapshot) error {
    if snapshot.ChangeType == "deleted" {
        return nil
    }

    actualHash := computeHash(snapshot.Content)
    if actualHash != snapshot.ContentHash {
        return fmt.Errorf("content hash mismatch: expected %s, got %s",
            snapshot.ContentHash, actualHash)
    }

    return nil
}

// rollbackFileSnapshotIncremental 增量恢复文件系统（增强版）
func (s *CheckpointService) rollbackFileSnapshotIncremental(ctx context.Context, targetCheckpoint *Checkpoint, project *Project) error {
    // 1. 构建 Checkpoint 链
    chain, err := s.buildCheckpointChain(ctx, targetCheckpoint)
    if err != nil {
        // ✅ 链断裂，回退到完整快照
        return s.rollbackToFullSnapshot(ctx, targetCheckpoint, project)
    }

    // 2. 验证链的完整性
    if err := s.validateCheckpointChain(chain); err != nil {
        return err
    }

    // 3. 如果链过长，先压缩
    if len(chain) > 100 {
        compactedCheckpoint, err := s.compactCheckpoints(ctx, chain)
        if err != nil {
            return err
        }
        // 重新构建链
        chain, _ = s.buildCheckpointChain(ctx, targetCheckpoint)
    }

    // 4. 清空项目目录（保留 .git 等隐藏文件）
    if err := s.cleanProjectFiles(project.RepoPath); err != nil {
        return err
    }

    // 5. 按顺序应用每个 Checkpoint 的变更
    fileStates := make(map[string]FileSnapshot)
    for _, cp := range chain {
        snapshots, err := s.checkpointRepo.GetFileSnapshots(ctx, cp.ID)
        if err != nil {
            return fmt.Errorf("failed to load snapshots for checkpoint %s: %w", cp.ID, err)
        }

        for _, snapshot := range snapshots {
            // ✅ 验证文件快照完整性
            if err := s.validateFileSnapshot(snapshot); err != nil {
                return fmt.Errorf("corrupted snapshot for file %s: %w", snapshot.FilePath, err)
            }

            switch snapshot.ChangeType {
            case "added", "modified":
                fileStates[snapshot.FilePath] = snapshot
            case "deleted":
                delete(fileStates, snapshot.FilePath)
            }
        }
    }

    // 6. 写入最终文件状态
    for path, snapshot := range fileStates {
        fullPath := filepath.Join(project.RepoPath, path)
        if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
            return err
        }
        if err := os.WriteFile(fullPath, snapshot.Content, 0644); err != nil {
            return err
        }
    }

    return nil
}

// rollbackToFullSnapshot 完整快照回滚（链断裂时的回退策略）
func (s *CheckpointService) rollbackToFullSnapshot(ctx context.Context, checkpoint *Checkpoint, project *Project) error {
    // 1. 清空项目目录
    if err := s.cleanProjectFiles(project.RepoPath); err != nil {
        return err
    }

    // 2. 加载目标 Checkpoint 的所有文件快照
    snapshots, err := s.checkpointRepo.GetFileSnapshots(ctx, checkpoint.ID)
    if err != nil {
        return err
    }

    // 3. 写入所有文件
    for _, snapshot := range snapshots {
        if snapshot.ChangeType == "deleted" {
            continue
        }

        fullPath := filepath.Join(project.RepoPath, snapshot.FilePath)
        if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
            return err
        }
        if err := os.WriteFile(fullPath, snapshot.Content, 0644); err != nil {
            return err
        }
    }

    return nil
}
```

**自动压缩策略**:

```go
// 定期清理旧 Checkpoint（后台任务）
func (s *CheckpointService) CleanupOldCheckpoints(ctx context.Context) error {
    sessions, _ := s.sessionRepo.ListAll(ctx)

    for _, session := range sessions {
        checkpoints, _ := s.checkpointRepo.ListBySession(ctx, session.ID)

        // 按时间排序，保留最近 10 个
        sort.Slice(checkpoints, func(i, j int) bool {
            return checkpoints[i].CreatedAt.After(checkpoints[j].CreatedAt)
        })

        toDelete := checkpoints[10:]
        for _, cp := range toDelete {
            if time.Since(cp.CreatedAt) > 30*24*time.Hour {
                s.checkpointRepo.Delete(ctx, cp.ID)
            }
        }
    }

    return nil
}
```

---

### 领域 7: ChangeSet 并发安全

#### 7.1 乐观锁机制

**目标**: 防止 ChangeSet 提交时的竞态条件

**数据库 Schema**（SQLite，v3.0 调整）:

```sql
-- 为 sessions 表添加版本号
ALTER TABLE sessions ADD COLUMN version INTEGER NOT NULL DEFAULT 1;

-- 为 change_ledgers 添加版本号
CREATE TABLE change_ledgers (
    session_id TEXT PRIMARY KEY,
    version INTEGER NOT NULL DEFAULT 1,
    entries TEXT NOT NULL,  -- SQLite 使用 TEXT 存储 JSON
    pending_change_set_id TEXT,
    last_committed_checkpoint_id TEXT,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

**Go 实现**:

```go
// 修改：change_set_service.go

type ChangeSetCommitRequest struct {
    Message         string `json:"message"`
    ExpectedVersion int    `json:"expected_version"`  // 新增：乐观锁版本号
}

func ConversationChangeSetCommitHandler(state *AppState) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        conversationID := runtimeSessionIDFromPath(r)
        input := ChangeSetCommitRequest{}
        if err := decodeJSONBody(r, &input); err != nil {
            err.write(w, r)
            return
        }

        state.mu.Lock()
        ledger := ensureConversationChangeLedgerLocked(state, conversationID)

        // 乐观锁检查
        if input.ExpectedVersion != 0 && ledger.Version != input.ExpectedVersion {
            state.mu.Unlock()
            WriteStandardError(w, r, http.StatusConflict, "VERSION_CONFLICT",
                "ChangeSet version mismatch, please refresh", map[string]any{
                    "expected": input.ExpectedVersion,
                    "actual":   ledger.Version,
                })
            return
        }

        // 再次检查是否有运行中的 Run
        if hasMutableExecutionsLocked(state, conversationID) {
            state.mu.Unlock()
            WriteStandardError(w, r, http.StatusConflict, "SESSION_BUSY",
                "Cannot commit while session has running executions", nil)
            return
        }

        // 执行 commit
        project := loadChangeSetProjectSeedLocked(state, conversationID)
        driver := resolveProjectChangeDriver(project)
        state.mu.Unlock()

        checkpoint, err := driver.Commit(project, ledger.Entries, input.Message)
        if err != nil {
            WriteStandardError(w, r, http.StatusInternalServerError, "COMMIT_FAILED",
                err.Error(), nil)
            return
        }

        // 更新版本号
        state.mu.Lock()
        ledger.Entries = []ChangeEntry{}
        ledger.Version += 1  // 版本号递增
        ledger.LastCommittedCheckpoint = &checkpoint
        state.mu.Unlock()

        writeJSON(w, http.StatusOK, ChangeSetCommitResponse{
            OK:         true,
            Checkpoint: checkpoint,
            Version:    ledger.Version,  // 返回新版本号
        })
    }
}
```

**前端实现**:

```typescript
// apps/desktop/src/modules/session/services/changeset.ts

export async function commitChangeSet(
  sessionId: string,
  message: string
): Promise<void> {
  const runtime = sessionStore.bySessionId[sessionId];
  if (!runtime?.changeSet) {
    throw new Error('No changeset to commit');
  }

  // 携带当前版本号
  const response = await apiClient.post(
    `/v1/sessions/${sessionId}/changeset/commit`,
    {
      message,
      expected_version: runtime.changeSet.version  // 乐观锁
    }
  );

  if (response.status === 409) {
    // 版本冲突，提示用户刷新
    notificationStore.add({
      type: 'error',
      title: '提交失败',
      message: 'ChangeSet 已被其他操作修改，请刷新后重试',
      actions: [
        { label: '刷新', onClick: () => reloadSession(sessionId) }
      ]
    });
    throw new Error('Version conflict');
  }

  // 更新本地版本号
  if (runtime.changeSet) {
    runtime.changeSet.version = response.data.version;
  }
}
```

**预期效果**:
- 并发提交时自动检测冲突
- 用户收到明确的冲突提示
- 避免"覆盖他人变更"的数据丢失

---

### 领域 8: Session 队列状态机增强（v3.0 调整为仅支持取消）

#### 8.1 简化状态机

**目标**: 支持 Run 取消功能，暂停/恢复延后到 v0.6.0

**新状态定义**:

```typescript
// packages/shared-core/src/api-common.ts

export type QueueState =
  | "idle"           // 空闲
  | "running"        // 运行中
  | "queued"         // 排队中
  | "error"          // 错误状态（新增）
  | "rate_limited";  // 速率限制（新增）

// ❌ v0.5.0 不支持 paused 状态（延后到 v0.6.0）
```

**状态转换规则**:

```go
// 新增：internal/domain/session_state_machine.go

type SessionStateMachine struct {
    allowedTransitions map[QueueState][]QueueState
}

func NewSessionStateMachine() *SessionStateMachine {
    return &SessionStateMachine{
        allowedTransitions: map[QueueState][]QueueState{
            QueueStateIdle:         {QueueStateRunning, QueueStateQueued},
            QueueStateRunning:      {QueueStateIdle, QueueStateError},
            QueueStateQueued:       {QueueStateRunning, QueueStateIdle},
            QueueStateError:        {QueueStateIdle, QueueStateRunning},
            QueueStateRateLimited: {QueueStateQueued, QueueStateIdle},
        },
    }
}

func (sm *SessionStateMachine) CanTransition(from, to QueueState) bool {
    allowed, exists := sm.allowedTransitions[from]
    if !exists {
        return false
    }
    return contains(allowed, to)
}

func (sm *SessionStateMachine) Transition(session *Session, to QueueState, reason string) error {
    if !sm.CanTransition(session.QueueState, to) {
        return fmt.Errorf("invalid transition from %s to %s", session.QueueState, to)
    }

    session.QueueState = to
    session.QueueStateReason = reason
    session.UpdatedAt = time.Now()

    return nil
}
```

**API 端点**（仅支持取消）:

```yaml
# packages/contracts/openapi.yaml

/v1/runs/{run_id}/cancel:
  post:
    summary: 取消 Run
    description: |
      取消正在执行或排队中的 Run：
      - 中断 AI 模型调用（发送取消请求）
      - 标记 Run 状态为 cancelled
      - 触发队列中的下一个 Run
    responses:
      200:
        content:
          application/json:
            schema:
              type: object
              properties:
                ok:
                  type: boolean
                run:
                  $ref: '#/components/schemas/Run'
```

**Go 实现**（仅取消功能）:

```go
// ✅ 仅实现取消功能
func (e *Engine) CancelRun(ctx context.Context, runID core.RunID) error {
    e.mu.Lock()
    run, exists := e.runs[runID]
    e.mu.Unlock()

    if !exists {
        return core.ErrRunNotFound
    }

    // 1. 取消 AI 模型调用
    if cancelFunc, ok := e.cancelFuncs[runID]; ok {
        cancelFunc()
        delete(e.cancelFuncs, runID)
    }

    // 2. 更新 Run 状态
    run.mu.Lock()
    run.state = core.RunStateCancelled
    run.completedAt = time.Now()
    run.mu.Unlock()

    // 3. 持久化
    if err := e.runRepo.UpdateState(ctx, runID, core.RunStateCancelled); err != nil {
        return err
    }

    // 4. 发布事件
    e.eventStore.Append(ctx, core.Event{
        Type:      core.EventTypeRunCancelled,
        SessionID: run.sessionID,
        RunID:     runID,
        Timestamp: time.Now(),
    })

    // 5. 触发下一个 Run（如果队列中有）
    session := e.sessions[run.sessionID]
    session.mu.Lock()
    e.scheduleNextRunLocked(session)
    session.mu.Unlock()

    return nil
}
```

**预期效果**:
- 用户可以取消正在执行的 Run
- AI 模型调用立即中断（HTTP 请求取消）
- 队列中的下一个 Run 自动开始执行
- v0.6.0 再实现暂停/恢复功能（需要 AI 模型支持 resumable streaming）

---
      - 阻止新 Run 提交
      - 保存暂停时的中间状态
    responses:
      200:
        content:
          application/json:
            schema:
              type: object
              properties:
                ok:
                  type: boolean
                queue_state:
                  type: string
                  enum: [paused]
                paused_run_id:
                  type: string
                  description: 被暂停的 Run ID

/v1/sessions/{session_id}/resume:
  post:
    summary: 恢复暂停的 Session
    description: |
      恢复 Session 的执行：
      - 如果有被暂停的 Run，从暂停点继续执行
      - 允许新 Run 提交
    responses:
      200:
        content:
          application/json:
            schema:
              type: object
              properties:
                ok:
                  type: boolean
                queue_state:
                  type: string
                resumed_run_id:
                  type: string
```

**Go 实现（支持中断 AI 模型调用）**:

```go
// 新增：internal/agent/runtime/loop/pause.go

type PauseManager struct {
    mu              sync.Mutex
    pausedSessions  map[SessionID]*PausedState
    cancelFuncs     map[RunID]context.CancelFunc  // 用于取消 AI 调用
}

type PausedState struct {
    SessionID       SessionID
    PausedRunID     RunID
    IntermediateState json.RawMessage  // 保存中间状态
    PausedAt        time.Time
}

func (pm *PauseManager) PauseSession(ctx context.Context, sessionID SessionID) error {
    pm.mu.Lock()
    defer pm.mu.Unlock()

    // 1. 查找当前运行的 Run
    activeRun, exists := pm.findActiveRun(sessionID)
    if !exists {
        return fmt.Errorf("no active run to pause")
    }

    // 2. 取消 AI 模型调用（通过 context.CancelFunc）
    if cancelFunc, ok := pm.cancelFuncs[activeRun.ID]; ok {
        cancelFunc()  // 触发 context.Done()，中断 HTTP 请求
        delete(pm.cancelFuncs, activeRun.ID)
    }

    // 3. 保存中间状态
    intermediateState, _ := json.Marshal(activeRun.IntermediateState)
    pm.pausedSessions[sessionID] = &PausedState{
        SessionID:         sessionID,
        PausedRunID:       activeRun.ID,
        IntermediateState: intermediateState,
        PausedAt:          time.Now(),
    }

    // 4. 更新 Run 状态为 paused
    activeRun.State = RunStatePaused
    pm.saveRun(ctx, activeRun)

    return nil
}

func (pm *PauseManager) ResumeSession(ctx context.Context, sessionID SessionID) error {
    pm.mu.Lock()
    defer pm.mu.Unlock()

    pausedState, exists := pm.pausedSessions[sessionID]
    if !exists {
        return fmt.Errorf("session not paused")
    }

    // 1. 恢复 Run 状态
    run, _ := pm.loadRun(ctx, pausedState.PausedRunID)
    run.State = RunStateExecuting

    // 2. 恢复中间状态
    json.Unmarshal(pausedState.IntermediateState, &run.IntermediateState)

    // 3. 创建新的 cancelable context
    cancelCtx, cancelFunc := context.WithCancel(ctx)
    pm.cancelFuncs[run.ID] = cancelFunc

    // 4. 重新提交到执行队列
    go pm.continueRunExecution(cancelCtx, run)

    delete(pm.pausedSessions, sessionID)
    return nil
}

// Engine 集成
func (e *Engine) Submit(ctx context.Context, sessionID string, input core.UserInput) (runID string, err error) {
    // ... 现有逻辑 ...

    // 创建 cancelable context
    cancelCtx, cancelFunc := context.WithCancel(ctx)
    e.pauseManager.RegisterCancelFunc(newRunID, cancelFunc)

    // 执行 Run（传递 cancelCtx）
    go e.executeRun(cancelCtx, newRunID)

    return string(newRunID), nil
}

func (e *Engine) executeRun(ctx context.Context, runID RunID) {
    // 在 AI 模型调用时检查 context
    select {
    case <-ctx.Done():
        // 收到取消信号，保存中间状态并退出
        e.saveIntermediateState(runID)
        return
    default:
        // 继续执行
        response, err := e.modelClient.Chat(ctx, request)
        // ...
    }
}
```

**预期效果**:
- 用户点击"暂停"后，AI 模型调用立即中断（HTTP 请求取消）
- 保存当前已完成的工具调用结果
- 恢复时从暂停点继续执行，无需重新开始

**预期效果**:
- 支持"暂停长时间运行的 Run"
- 错误状态明确，用户知道是否可重试
- 为未来的优先级调度预留扩展点

---

#### 3.1 E2E 测试场景扩展

**当前**: 4 个场景
**目标**: 20+ 个关键用户旅程

**新增场景**:

```typescript
// e2e/workspace-switching.spec.ts
test('切换工作区保留会话状态', async ({ page }) => {
  await page.goto('/');
  await createWorkspace(page, 'workspace-1');
  await createSession(page, 'session-1');
  await sendMessage(page, 'Hello');

  // 切换到另一个工作区
  await switchWorkspace(page, 'workspace-2');

  // 切换回来，验证状态保留
  await switchWorkspace(page, 'workspace-1');
  await expect(page.locator('[data-testid="message"]')).toContainText('Hello');
});

// e2e/changeset-approval.spec.ts
test('ChangeSet 审批流程', async ({ page }) => {
  await createSessionWithRun(page);
  await waitForChangeSet(page);

  // 审查变更
  await page.click('[data-testid="changeset-review"]');
  await expect(page.locator('[data-testid="diff-viewer"]')).toBeVisible();

  // 批准变更
  await page.click('[data-testid="approve-changeset"]');
  await expect(page.locator('[data-testid="commit-success"]')).toBeVisible();
});

// e2e/mcp-integration.spec.ts
test('MCP 工具调用', async ({ page }) => {
  await enableMCPServer(page, 'filesystem');
  await createSession(page);
  await sendMessage(page, 'Read file test.txt');

  // 验证 MCP 工具调用
  await expect(page.locator('[data-testid="tool-call"]')).toContainText('mcp__filesystem__read_file');
});
```

#### 3.2 移除覆盖率排除

**当前排除**:
- `useMainScreen*.ts`
- `streamCoordinator.ts`
- `sseClient.ts`

**策略**:
1. 拆分 `useMainScreenController.ts` (1136 行) 为多个可测试单元
2. 为 `streamCoordinator.ts` 添加单元测试（模拟 SSE）
3. 为 `sseClient.ts` 添加集成测试

```typescript
// tests/streamCoordinator.spec.ts
describe('StreamCoordinator', () => {
  it('should buffer events during resync', async () => {
    const coordinator = new StreamCoordinator();
    const events: RunLifecycleEvent[] = [];

    coordinator.onEvent((e) => events.push(e));

    // 触发重同步
    coordinator.handleResync();

    // 发送事件（应被缓冲）
    coordinator.handleEvent({ type: 'run_started', sessionId: 'test' });

    // 完成重同步
    await coordinator.completeResync();

    // 验证事件已应用
    expect(events).toHaveLength(1);
  });
});
```

#### 3.3 集成测试增强

**新增**: Hub + Desktop 集成测试

```typescript
// tests/integration/session-lifecycle.spec.ts
describe('Session Lifecycle Integration', () => {
  let hubServer: HubTestServer;

  beforeAll(async () => {
    hubServer = await startTestHub();
  });

  it('should create session and execute run', async () => {
    const client = createTestClient(hubServer.url);

    // 创建会话
    const session = await client.post('/v1/sessions', {
      workspace_id: 'test-ws',
      project_id: 'test-proj'
    });

    // 提交消息
    const run = await client.post(`/v1/sessions/${session.id}/runs`, {
      message: 'Hello'
    });

    // 等待完成
    await waitForRunState(client, run.id, 'completed');

    // 验证事件
    const events = await client.get(`/v1/sessions/${session.id}/events`);
    expect(events).toContainEqual(expect.objectContaining({ type: 'run_completed' }));
  });
});
```

---

## 实施路线图

### 阶段 0: 准备阶段 (1 周)

**目标**: 建立重构基础设施

- [ ] 创建 `v0.5.0-refactor` 分支
- [ ] 设置数据库迁移工具（golang-migrate）
- [ ] 建立 feature flag 系统（支持渐进式迁移）
- [ ] 更新 CI/CD 流水线（增加数据库测试）
- [ ] 编写迁移脚本（内存数据 → SQLite）

**交付物**:
- `services/hub/migrations/` 目录
- `scripts/migrate-to-v050.sh` 迁移脚本
- 更新的 `.github/workflows/ci.yml`

---

### 阶段 1: 后端领域层重构 (2-3 周)

**目标**: 建立 DDD 领域模型

#### Week 1: 领域模型定义

- [ ] 创建 `services/hub/internal/domain/` 包
- [ ] 定义聚合根：`Workspace`, `Session`, `Run`
- [ ] 定义值对象：`WorkspaceID`, `SessionID`, `RunState`
- [ ] 定义 Repository 接口
- [ ] 编写领域模型单元测试

**关键文件**:
```
services/hub/internal/domain/
├── workspace.go
├── session.go
├── run.go
├── repository.go
└── events.go
```

#### Week 2-3: Repository 实现

- [ ] 实现 SQLite Repository（`internal/infrastructure/sqlite/`）
- [ ] **新增**: 实现 `RunRepository` 和 `RunEventRepository`
- [ ] 实现事件总线（内存版本）
- [ ] 数据库迁移脚本（创建表结构）
- [ ] Repository 集成测试
- [ ] **新增**: Agent Engine 持久化集成测试
- [ ] 性能基准测试（对比内存 map）

**关键文件**:
```
services/hub/internal/infrastructure/
├── sqlite/
│   ├── workspace_repo.go
│   ├── session_repo.go
│   ├── run_repo.go           # 新增
│   └── run_event_repo.go     # 新增
└── eventbus/
    └── memory.go
```

**Agent Engine 改造**:
- [ ] 重构 `runtime/loop/engine.go` 使用 Repository
- [ ] 实现细粒度锁（per-session）
- [ ] 添加内存缓存层（可选）
- [ ] Engine 重启恢复测试

---

### 阶段 2: 后端应用层重构 (3 周)

**目标**: CQRS 命令/查询分离 + Agent 接口完善 + 资源配置继承

#### Week 1: 命令层 + Agent CommandBus

- [ ] 创建 `internal/application/commands/` 包
- [ ] 实现命令：`CreateSession`, `SubmitMessage`, `ApproveRun`
- [ ] **新增**: 实现 `CommandBus` 接口（`internal/agent/runtime/slash/`）
- [ ] **新增**: 实现 `ToolExecutor` 接口（`internal/agent/runtime/tools/`）
- [ ] 命令处理器单元测试
- [ ] 集成到现有 HTTP handlers

#### Week 2: 资源配置继承机制（新增）

- [ ] 实现 `ResourceConfigService`（资源继承与验证）
- [ ] 实现资源版本控制（`resource_configs` 表）
- [ ] 实现项目资源绑定白名单（`project_resource_bindings` 表）
- [ ] 实现 Session 资源快照（`session_resource_snapshots` 表）
- [ ] 实现资源删除时的级联处理与自动降级
- [ ] 添加资源变更事件推送（SSE）
- [ ] 单元测试与集成测试

**关键文件**:
```
services/hub/internal/domain/
├── resource_config.go          # 资源配置领域模型
├── resource_config_service.go  # 资源继承与验证逻辑
└── resource_events.go          # 资源变更事件定义

services/hub/internal/infrastructure/sqlite/
├── resource_config_repo.go     # 资源配置 Repository
└── migrations/
    └── 003_resource_inheritance.sql
```

#### Week 2-3: 查询层 + Agent 权限优化 + Checkpoint 实现

- [ ] 创建 `internal/application/queries/` 包
- [ ] 实现查询：`GetSessionDetail`, `ListSessions`, `GetRunEvents`
- [ ] **新增**: 实现统一权限评估流程（`UnifiedPermissionGate`）
- [ ] **新增**: 添加权限审计日志
- [ ] **新增**: 实现 Checkpoint 服务（Git + 非 Git 项目）
- [ ] **新增**: 实现 Rollback API
- [ ] 查询优化（索引、投影）
- [ ] 查询层测试

**Checkpoint 实现**:
```
services/hub/internal/domain/
├── checkpoint.go               # Checkpoint 领域模型
├── checkpoint_service.go       # 创建与回滚逻辑
└── checkpoint_strategy.go      # Git/非Git 策略

services/hub/internal/infrastructure/sqlite/
├── checkpoint_repo.go          # Checkpoint Repository
└── migrations/
    └── 004_checkpoints.sql
```

**Agent 权限优化**:
```
services/hub/internal/agent/policy/
├── unified/
│   ├── gate.go             # UnifiedPermissionGate
│   ├── audit.go            # 审计日志
│   └── gate_test.go
```

---

### 阶段 3: 前端状态管理重构 (2-3 周)

**目标**: 统一命名 + Repository 模式 + 资源变更感知

#### Week 1: 命名统一

- [ ] 全局替换 `Conversation` → `Session`
- [ ] 删除镜像字段（`byConversationId`, `executions`）
- [ ] 更新所有导入路径
- [ ] 运行完整测试套件
- [ ] 更新文档

**脚本辅助**:
```bash
# scripts/refactor/rename-conversation-to-session.sh
find apps/desktop/src -type f \( -name "*.ts" -o -name "*.vue" \) \
  -exec sed -i '' 's/Conversation/Session/g' {} \;
```

#### Week 2: Repository 模式 + 资源变更感知（新增）

- [ ] 创建 `apps/desktop/src/shared/repositories/` 目录
- [ ] 实现 `SessionRepository`, `WorkspaceRepository`, `ResourceConfigRepository`
- [ ] 重构 Store 使用 Repository
- [ ] **新增**: 实现资源变更事件处理（SSE）
- [ ] **新增**: 实现资源配置实时通知 UI
- [ ] 添加 Repository 单元测试（模拟 HTTP）
- [ ] 更新集成测试

**资源变更感知**:
```typescript
// apps/desktop/src/modules/resource/
├── services/
│   └── resourceConfigService.ts    # 资源配置服务
├── store/
│   └── resourceStore.ts            # 资源配置 Store
└── components/
    └── ResourceChangeNotification.vue  # 变更通知组件
```

#### Week 3: ChangeSet 乐观锁 + Checkpoint UI（新增）

- [ ] 实现 ChangeSet 乐观锁机制
- [ ] 实现版本冲突检测与提示
- [ ] **新增**: 实现 Checkpoint 创建 UI
- [ ] **新增**: 实现 Checkpoint 列表与回滚 UI
- [ ] **新增**: 实现 Session 暂停/恢复 UI
- [ ] 添加 E2E 测试

**Checkpoint UI**:
```vue
<!-- apps/desktop/src/modules/session/components/CheckpointPanel.vue -->
<template>
  <div class="checkpoint-panel">
    <button @click="createCheckpoint">创建回滚点</button>
    <div v-for="cp in checkpoints" :key="cp.id">
      <span>{{ cp.message }}</span>
      <button @click="rollback(cp.id)">回滚</button>
    </div>
  </div>
</template>
---

### 阶段 4: SSE 事件处理重构 (1 周)

**目标**: 修复事件丢失问题

- [ ] 实现 `SessionEventQueue` 类
- [ ] 实现 `StreamRegistry` 防止竞态
- [ ] 重构 `stream.ts` 使用新架构
- [ ] 添加 SSE 重连集成测试
- [ ] 压力测试（模拟高频事件）

**验证**:
```typescript
// 测试：重连期间不丢失事件
test('SSE reconnection preserves events', async () => {
  const events: RunLifecycleEvent[] = [];
  const stream = attachSessionStream(session, token);

  // 模拟断连
  stream.close();

  // 发送事件（应被缓冲）
  mockSSE.emit('run_started', { sessionId: 'test' });

  // 重连
  await stream.reconnect();

  // 验证事件已应用
  expect(events).toHaveLength(1);
});
```

---

### 阶段 5: E2E 测试扩展 (2 周)

**目标**: 覆盖关键用户旅程 + 新增业务场景

- [ ] 工作区切换测试（5 个场景）
- [ ] ChangeSet 审批流程（3 个场景）
- [ ] MCP 集成测试（4 个场景）
- [ ] Hook 调度测试（2 个场景）
- [ ] 多会话并发测试（3 个场景）
- [ ] **新增**: 资源配置变更感知测试（3 个场景）
- [ ] **新增**: Checkpoint 创建与回滚测试（4 个场景）
- [ ] **新增**: Session 暂停/恢复测试（2 个场景）
- [ ] **新增**: ChangeSet 乐观锁冲突测试（2 个场景）
- [ ] 性能回归测试（基准对比）

**新增测试场景**:

```typescript
// e2e/resource-config-change.spec.ts
test('资源配置变更实时感知', async ({ page }) => {
  await createSession(page, { modelId: 'model-1' });

  // 管理员禁用模型
  await adminDisableModel(page, 'model-1');

  // 验证会话收到通知
  await expect(page.locator('[data-testid="resource-change-notification"]'))
    .toContainText('模型已切换');

  // 验证自动切换到默认模型
  await expect(page.locator('[data-testid="current-model"]'))
    .toContainText('model-default');
});

// e2e/checkpoint-rollback.spec.ts
test('Checkpoint 回滚恢复会话状态', async ({ page }) => {
  await createSession(page);
  await sendMessage(page, 'Message 1');
  await sendMessage(page, 'Message 2');

  // 创建 Checkpoint
  await page.click('[data-testid="create-checkpoint"]');
  await page.fill('[data-testid="checkpoint-message"]', 'Before message 3');
  await page.click('[data-testid="confirm-checkpoint"]');

  await sendMessage(page, 'Message 3');

  // 回滚到 Checkpoint
  await page.click('[data-testid="checkpoint-list"]');
  await page.click('[data-testid="rollback-checkpoint-0"]');

  // 验证消息历史已回滚
  const messages = await page.locator('[data-testid="message"]').count();
  expect(messages).toBe(2);
});

// e2e/changeset-version-conflict.spec.ts
test('ChangeSet 版本冲突检测', async ({ page, context }) => {
  await createSessionWithChanges(page);

  // 打开第二个标签页（模拟并发）
  const page2 = await context.newPage();
  await page2.goto(page.url());

  // 两个标签页同时提交
  await Promise.all([
    page.click('[data-testid="commit-changeset"]'),
    page2.click('[data-testid="commit-changeset"]')
  ]);

  // 验证其中一个收到冲突提示
  const errorNotification = page.locator('[data-testid="version-conflict-error"]')
    .or(page2.locator('[data-testid="version-conflict-error"]'));
  await expect(errorNotification).toBeVisible();
});
```

**目标指标**:
- E2E 场景数: 4 → 30+
- 关键路径覆盖率: 60% → 90%
- 测试执行时间: < 8 分钟

---

### 阶段 6: 性能优化与监控 (1-2 周)

**目标**: 生产就绪 + Agent 可观测性 + 资源配置性能

- [ ] 添加 OpenTelemetry 追踪
- [ ] 实现请求去重（防止重复提交）
- [ ] 添加 ChangeSet 刷新防抖（300ms）
- [ ] 数据库连接池优化
- [ ] 前端性能监控（Web Vitals）
- [ ] 内存泄漏检测
- [ ] **新增**: Agent Subscriber 背压监控
- [ ] **新增**: Agent Engine 并发性能基准测试
- [ ] **新增**: 权限评估审计日志查询接口
- [ ] **新增**: 资源配置查询性能优化（索引、缓存）
- [ ] **新增**: Checkpoint 存储空间监控与清理策略

**Agent 监控指标**:
```go
// 后端指标
metrics.RecordHistogram("agent.engine.submit.duration", duration)
metrics.RecordCounter("agent.run.state.transition", 1, tags)
metrics.RecordGauge("agent.active.sessions", count)
metrics.RecordCounter("agent.subscriber.events.dropped", dropped)
metrics.RecordHistogram("agent.permission.evaluate.duration", duration)
metrics.RecordCounter("resource.config.changed", 1, tags)
metrics.RecordHistogram("checkpoint.create.duration", duration)
metrics.RecordGauge("checkpoint.storage.bytes", size)

// 前端指标
performance.measure('agent-run-submit-time', 'start', 'end');
performance.measure('resource-change-notification-time', 'start', 'end');
reportWebVitals((metric) => {
  analytics.track(metric.name, metric.value);
});
```

**Checkpoint 清理策略**:
```go
// 自动清理超过 30 天的 Checkpoint（保留最近 10 个）
func (s *CheckpointService) CleanupOldCheckpoints(ctx context.Context) error {
    sessions, _ := s.sessionRepo.ListAll(ctx)

    for _, session := range sessions {
        checkpoints, _ := s.checkpointRepo.ListBySession(ctx, session.ID)

        // 按时间排序，保留最近 10 个
        sort.Slice(checkpoints, func(i, j int) bool {
            return checkpoints[i].CreatedAt.After(checkpoints[j].CreatedAt)
        })

        toDelete := checkpoints[10:]
        for _, cp := range toDelete {
            if time.Since(cp.CreatedAt) > 30*24*time.Hour {
                s.checkpointRepo.Delete(ctx, cp.ID)
            }
        }
    }

    return nil
}
```

---

### 阶段 7: 文档与发布 (1 周)

**目标**: 平滑迁移

- [ ] 编写迁移指南（v0.4.0 → v0.5.0）
- [ ] 更新 API 文档（OpenAPI 规范）
- [ ] **新增**: 编写资源配置继承机制文档
- [ ] **新增**: 编写 Checkpoint 使用指南
- [ ] **新增**: 编写 Session 队列状态机文档
- [ ] 录制演示视频（新功能）
- [ ] 发布 Beta 版本（内部测试）
- [ ] 收集反馈并修复
- [ ] 正式发布 v0.5.0

**交付物**:
- `docs/migration/v040-to-v050.md`
- `docs/features/resource-inheritance.md`（新增）
- `docs/features/checkpoint-rollback.md`（新增）
- `docs/features/session-queue-states.md`（新增）
- 更新的 `packages/contracts/openapi.yaml`
- `CHANGELOG.md` v0.5.0 条目

---
- [ ] 添加 SSE 重连集成测试
- [ ] 压力测试（模拟高频事件）

**验证**:
```typescript
// 测试：重连期间不丢失事件
test('SSE reconnection preserves events', async () => {
  const events: RunLifecycleEvent[] = [];
  const stream = attachSessionStream(session, token);

  // 模拟断连
  stream.close();

  // 发送事件（应被缓冲）
  mockSSE.emit('run_started', { sessionId: 'test' });

  // 重连
  await stream.reconnect();

  // 验证事件已应用
  expect(events).toHaveLength(1);
});
```

---

### 阶段 5: E2E 测试扩展 (1-2 周)

**目标**: 覆盖关键用户旅程

- [ ] 工作区切换测试（5 个场景）
- [ ] ChangeSet 审批流程（3 个场景）
- [ ] MCP 集成测试（4 个场景）
- [ ] Hook 调度测试（2 个场景）
- [ ] 多会话并发测试（3 个场景）
- [ ] 性能回归测试（基准对比）

**目标指标**:
- E2E 场景数: 4 → 20+
- 关键路径覆盖率: 60% → 85%
- 测试执行时间: < 5 分钟

---

### 阶段 6: 性能优化与监控 (1 周)

**目标**: 生产就绪 + Agent 可观测性

- [ ] 添加 OpenTelemetry 追踪
- [ ] 实现请求去重（防止重复提交）
- [ ] 添加 ChangeSet 刷新防抖（300ms）
- [ ] 数据库连接池优化
- [ ] 前端性能监控（Web Vitals）
- [ ] 内存泄漏检测
- [ ] **新增**: Agent Subscriber 背压监控
- [ ] **新增**: Agent Engine 并发性能基准测试
- [ ] **新增**: 权限评估审计日志查询接口

**Agent 监控指标**:
```go
// 后端指标
metrics.RecordHistogram("agent.engine.submit.duration", duration)
metrics.RecordCounter("agent.run.state.transition", 1, tags)
metrics.RecordGauge("agent.active.sessions", count)
metrics.RecordCounter("agent.subscriber.events.dropped", dropped)
metrics.RecordHistogram("agent.permission.evaluate.duration", duration)

// 前端指标
performance.measure('agent-run-submit-time', 'start', 'end');
reportWebVitals((metric) => {
  analytics.track(metric.name, metric.value);
});
```

---

### 阶段 7: 文档与发布 (1 周)

**目标**: 平滑迁移

- [ ] 编写迁移指南（v0.4.0 → v0.5.0）
- [ ] 更新 API 文档（OpenAPI 规范）
- [ ] 录制演示视频（新功能）
- [ ] 发布 Beta 版本（内部测试）
- [ ] 收集反馈并修复
- [ ] 正式发布 v0.5.0

**交付物**:
- `docs/migration/v040-to-v050.md`
- 更新的 `packages/contracts/openapi.yaml`
- `CHANGELOG.md` v0.5.0 条目

---

## 迁移策略

### 数据迁移

**从内存 Map 到 SQLite**

```bash
#!/bin/bash
# scripts/migrate-to-v050.sh

echo "开始迁移 v0.4.0 数据到 v0.5.0..."

# 1. 备份当前数据（如果有持久化）
if [ -f ~/.goyais/data.json ]; then
  cp ~/.goyais/data.json ~/.goyais/data.json.backup
fi

# 2. 运行数据库迁移
cd services/hub
go run cmd/migrate/main.go up

# 3. 导入历史数据（如果需要）
if [ -f ~/.goyais/data.json.backup ]; then
  go run cmd/import/main.go --source ~/.goyais/data.json.backup
fi

echo "迁移完成！"
```

### API 兼容性

**策略**: 保持 REST API 向后兼容

- 保留 `/v1/conversations/*` 路径（内部映射到 Session）
- 响应中同时返回 `conversation_id` 和 `session_id`（相同值）
- 在 v0.6.0 废弃旧端点

```go
// 兼容层
func (h *Handler) GetConversation(w http.ResponseWriter, r *http.Request) {
    // 内部调用 GetSession
    session, err := h.queries.GetSession(r.Context(), conversationID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }

    // 响应包含两个字段
    response := map[string]interface{}{
        "conversation_id": session.ID,  // 兼容旧客户端
        "session_id":      session.ID,  // 新字段
        // ... 其他字段
    }
    json.NewEncoder(w).Encode(response)
}
```

### 渐进式部署

**Feature Flag 控制**

```go
// internal/config/features.go
type FeatureFlags struct {
    UseSQLiteRepository bool
    UseEventBus         bool
    EnableCQRS          bool
}

// 环境变量控制
flags := FeatureFlags{
    UseSQLiteRepository: os.Getenv("FEATURE_SQLITE_REPO") == "true",
    UseEventBus:         os.Getenv("FEATURE_EVENT_BUS") == "true",
    EnableCQRS:          os.Getenv("FEATURE_CQRS") == "true",
}
```

**部署步骤**:
1. 部署 v0.5.0（所有 feature flags 关闭）
2. 逐步开启 `FEATURE_SQLITE_REPO`（监控性能）
3. 开启 `FEATURE_EVENT_BUS`（验证事件推送）
4. 开启 `FEATURE_CQRS`（完整切换）
5. 移除 feature flags 和旧代码

---

## 风险评估

### 高风险项

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 数据库迁移失败 | 用户数据丢失 | 1. 自动备份<br>2. 回滚脚本<br>3. 迁移前验证 |
| 性能回归 | 用户体验下降 | 1. 基准测试<br>2. 性能监控<br>3. 数据库索引优化 |
| SSE 重构引入新 bug | 事件丢失 | 1. 全面集成测试<br>2. 灰度发布<br>3. 快速回滚机制 |
| 命名统一遗漏 | 功能异常 | 1. 自动化脚本<br>2. 全量测试<br>3. 代码审查 |

### 中风险项

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| Repository 抽象过度 | 开发效率降低 | 保持接口简洁，避免过度设计 |
| 测试时间增加 | CI/CD 变慢 | 并行测试，缓存依赖 |
| 学习曲线 | 团队适应慢 | 编写详细文档，代码示例 |

### 回滚计划

**如果 v0.5.0 出现严重问题**:

1. **立即回滚**: 切换到 v0.4.0 分支
2. **数据恢复**: 从备份恢复（如果有数据迁移）
3. **通知用户**: 发布回滚公告
4. **问题修复**: 在 hotfix 分支修复
5. **重新发布**: v0.5.1 修复版本

```bash
# 回滚脚本
#!/bin/bash
# scripts/rollback-v050.sh

echo "回滚到 v0.4.0..."

# 1. 停止服务
systemctl stop goyais-hub

# 2. 恢复代码
git checkout v0.4.0

# 3. 恢复数据
if [ -f ~/.goyais/data.json.backup ]; then
  cp ~/.goyais/data.json.backup ~/.goyais/data.json
fi

# 4. 重启服务
systemctl start goyais-hub

echo "回滚完成！"
```

---

## 成功指标

### 技术指标

| 指标 | v0.4.0 基线 | v0.5.0 目标 |
|------|-------------|-------------|
| 后端并发请求处理 | ~100 req/s | 500+ req/s |
| Agent Engine 并发 Session 创建 | ~50 req/s | 500+ req/s |
| Agent Run 提交延迟 (p99) | ~50ms | < 10ms |
| 内存占用（10 会话） | ~200 MB | < 150 MB |
| SSE 事件丢失率 | ~5% | 0% |
| Subscriber 背压触发率 | 未监控 | < 1% |
| E2E 测试覆盖 | 4 场景 | 20+ 场景 |
| 代码圈复杂度 | 平均 8 | 平均 ≤ 6 |
| 上帝对象数量 | 1 (AppState) | 0 |
| Agent 接口实现完整度 | 70% (7/10) | 100% (10/10) |
| 权限评估审计覆盖率 | 0% | 100% |

### 业务指标

| 指标 | 目标 |
|------|------|
| 用户迁移成功率 | > 95% |
| 迁移时间 | < 5 分钟 |
| 生产事故数 | 0 |
| 用户满意度 | > 4.5/5 |

---

## 总结

v0.5.0 重构是 Goyais 走向生产就绪的关键里程碑。通过引入 DDD、CQRS、Repository 模式和持久化层，我们将建立一个可扩展、可维护、高性能的架构基础。

**核心改进**:
1. ✅ 消除上帝对象，建立清晰的领域边界
2. ✅ 统一命名，消除认知负担
3. ✅ 修复数据丢失风险，提升可靠性
4. ✅ 全面测试覆盖，保障质量
5. ✅ **Agent 运行时持久化，支持 Hub 重启恢复**
6. ✅ **Agent 细粒度锁优化，提升并发性能 5-10x**
7. ✅ **完善 Agent 核心接口，消除技术债务**
8. ✅ **统一权限评估流程，提升安全可审计性**

**Agent 架构改进亮点**:
- **持久化**: Engine 状态持久化到 SQLite，Hub 重启后自动恢复
- **并发优化**: 从全局锁改为 per-session 锁，并发性能提升 5-10x
- **接口完整**: 实现 CommandBus、ToolExecutor、CheckpointStore，接口完整度 70% → 100%
- **权限统一**: 单一权限决策路径，100% 审计覆盖
- **可观测性**: Subscriber 背压监控，事件丢失率降至 0%

**预计工期**: 8-10 周
**团队规模**: 2-3 名全职开发者
**发布时间**: 2026 Q2

---

## 附录

### A. 参考资料

- [Domain-Driven Design](https://martinfowler.com/bliki/DomainDrivenDesign.html)
- [CQRS Pattern](https://martinfowler.com/bliki/CQRS.html)
- [Repository Pattern](https://martinfowler.com/eaaCatalog/repository.html)
- [Event Sourcing](https://martinfowler.com/eaaDev/EventSourcing.html)
- [ClaudeHiddenToolkit.md](ClaudeHiddenToolkit.md)
- [Claude Code Docs](https://code.claude.com/docs)
### B. 相关 Issue

- #TBD: 后端 Repository 模式实现
- #TBD: 前端命名统一重构
- #TBD: SSE 事件队列实现
- #TBD: E2E 测试扩展

### C. 联系方式

如有疑问，请联系架构团队：
- 技术负责人: [待定]
- 架构评审: [待定]

---

**文档结束**


