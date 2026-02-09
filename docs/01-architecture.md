# Goyais 架构设计

> 本文档定义 Goyais 的 6 层逻辑架构、Go 包结构、依赖规则、单二进制部署模型、执行隔离等级与跨层通信模式。所有后续模块设计文档须遵循本文档的架构约束。

最后更新：2026-02-09

---

## 1. 架构总览

### 1.1 设计动机

Goyais 作为多模态 AI 编排与 Agent 平台，对架构提出以下核心诉求：

1. **职责边界清晰**：业务逻辑、调度编排、协议接入、数据持久化需要明确分层，避免职责交叉
2. **统一注册中心**：Tool、Model、Algorithm 的注册与发现需要集中管理，而非散落各处
3. **统一运行时层**：不同执行模式（HTTP/CLI/MCP/AIModel）的执行器需要统一调度入口
4. **可观测性显式建模**：Trace/Audit/Policy 作为横切关注点需要独立层级，可被任何层调用
5. **统一控制层抽象**：工作流引擎、调度器、Agent Runtime 需要聚合在同一控制层中
6. **全 AI 交互入口**：文本/语音输入需统一解析为平台动作计划（不仅限于 workflow 运行）
7. **国际化支撑**：请求语言协商、消息模板与前端翻译需作为基础能力内建

基于上述诉求，Goyais 采用 6 层逻辑架构，每一层有明确的职责边界和依赖约束。

### 1.2 六层逻辑架构

```
┌─────────────────────────────────────────────────────────────────────┐
│ Layer 1: Access Layer（接入层）                                      │
│ HTTP API + AI Interaction Gateway + SSE | CLI 命令 | MCP Server      │
├─────────────────────────────────────────────────────────────────────┤
│ Layer 2: Control Layer（控制层）                                      │
│ Intent Orchestrator | Workflow Engine | Agent Runtime | Scheduler     │
├─────────────────────────────────────────────────────────────────────┤
│ Layer 3: Registry Layer（注册中心层）                                 │
│ Tool Registry | Model Registry | Algorithm Library | Workflow Defs  │
├─────────────────────────────────────────────────────────────────────┤
│ Layer 4: Runtime Layer（运行时层）                                    │
│ Operator Runtime | MCP Runtime | Model Runtime                      │
├─────────────────────────────────────────────────────────────────────┤
│ Layer 5: Data & Asset Layer（数据与资产层）                            │
│ Asset Store | Metadata DB | Event Store | Context Store | Cache     │
├─────────────────────────────────────────────────────────────────────┤
│ Layer 6: Observability & Security（可观测性与安全层）                  │
│ Trace/Logs/Metrics | 审计日志 | Policy Engine | Budget Control      │
└─────────────────────────────────────────────────────────────────────┘

                    domain（跨层共享领域对象）
```

各层职责详述：

#### Layer 1: Access Layer（接入层）

**职责**：外部世界进入系统的所有入口点。

| 子模块 | 说明 |
|--------|------|
| HTTP API | REST/JSON 端点（Echo v4），统一响应封装，DTO 转换 |
| AI Interaction Gateway | 对话/语音输入接入、转写结果归一化、Intent 请求路由 |
| SSE | Server-Sent Events 实时事件推送（Run 进度、状态变更） |
| CLI | 命令行入口，用于运维、调试、批量操作 |
| MCP Server | Model Context Protocol 服务端，供外部 Agent/LLM 调用平台能力 |
| Middleware | JWT 认证、CORS、请求日志、限流、RBAC 鉴权 |
| I18n Resolver | locale 协商（`X-Locale` / `Accept-Language` / 用户偏好）与上下文注入 |

**关键约束**：
- Access Layer 只做协议转换与路由分发，不包含业务逻辑
- 所有业务操作委托给 Control Layer 或直接读取 Registry Layer
- 绝不直接依赖 Data Layer 或 Runtime Layer
- locale 解析结果需写入请求上下文，供 Control 层与响应封装统一使用

#### Layer 2: Control Layer（控制层）

**职责**：系统的"大脑"——编排、调度与决策。

| 子模块 | 说明 |
|--------|------|
| Intent Orchestrator | 意图解析、动作计划生成、确认/审批编排、统一执行入口 |
| Workflow Engine | DAG 解析（Kahn 拓扑排序）、节点编排、上下文管理（CAS Patch）、错误处理 |
| Agent Runtime | Run Loop 状态机（Plan→Act→Observe→Recover→Finish）、工具选择、决策链路 |
| Scheduler | 任务队列管理、并发控制、优先级调度、定时触发（gocron/v2）、事件触发 |

**关键约束**：
- Control Layer 是唯一可以调用 Runtime Layer 的层
- 所有 Intent 动作（创建用户/角色、修改设置、上传资源等）在此层统一落盘与审计
- 工作流执行与 Agent 决策统一产出 RunEvent 事件
- 所有状态变更通过 Data Layer 持久化

#### Layer 3: Registry Layer（注册中心层）

**职责**：所有可复用能力的注册、发现与版本管理。

| 子模块 | 说明 |
|--------|------|
| Tool Registry | ToolSpec 注册与发现、版本管理、启用/禁用控制 |
| Model Registry | AI 模型连接管理、可用性校验、Provider 适配配置 |
| Algorithm Library | 算法意图管理、版本发布、实现绑定与选择策略 |
| Workflow Defs | 工作流定义注册、Revision 版本化管理 |

**关键约束**：
- Registry Layer 提供查询与注册能力，不执行任何 Tool/Model
- 算法到 Tool 的解析在此层完成（实现选择策略）
- 对外暴露只读接口供 Access Layer 查询

#### Layer 4: Runtime Layer（运行时层）

**职责**：实际执行 Tool 调用的基础设施层。

| 子模块 | 说明 |
|--------|------|
| Operator Runtime | HTTP/CLI/容器化执行器，根据 ToolSpec.execution_mode 分派 |
| MCP Runtime | MCP 协议客户端，管理与外部 MCP Server 的连接与工具调用 |
| Model Runtime | 本地推理引擎适配 + 云模型 API 适配（OpenAI/Anthropic/Ollama 等） |

**关键约束**：
- Runtime Layer 只被 Control Layer 调用，绝不被 Access Layer 直接调用
- 每次执行前，必须经过 Policy Engine（Layer 6）的权限校验
- 执行结果通过 Data Layer 持久化

#### Layer 5: Data & Asset Layer（数据与资产层）

**职责**：所有持久化状态的读写。

| 子模块 | 说明 |
|--------|------|
| Asset Store | 原始内容存储（S3/MinIO/本地），仅存引用地址 |
| Metadata DB | GORM + SQL（SQLite/MySQL/PostgreSQL），所有实体元数据的持久化 |
| Event Store | RunEvent 持久化（SQL events 表）+ 应用层 EventBus 发布/订阅 |
| Context Store | ContextState/Patch/Snapshot 持久化 |
| Cache | 缓存抽象层（进程内 sync.Map / 可选 Redis） |

**关键约束**：
- Data Layer 不包含业务逻辑，仅提供 CRUD 与查询
- 所有 Repository 接口定义在 domain 包中，Data Layer 提供实现
- Event Store 同时支持持久化与进程内发布/订阅

#### Layer 6: Observability & Security（可观测性与安全层）

**职责**：横切关注点——追踪、审计、策略与预算。

| 子模块 | 说明 |
|--------|------|
| Trace | 分布式追踪（trace_id/run_id/node_id/tool_call_id） |
| Audit | 审计日志（谁在什么时间对什么做了什么操作） |
| Policy Engine | 执行前校验：工具权限、数据访问范围、风险等级审批 |
| Budget Control | 预算管理：成本累计、额度控制、超限拦截 |

**关键约束**：
- Layer 6 可被任何层调用（横切特性）
- Policy Engine 在 Tool 执行前由 Control Layer 主动调用
- 审计日志由 Middleware（Access Layer）与 Control Layer 双端写入

---

## 2. Go 包结构

```
internal/
├── access/                    # Layer 1: 接入层
│   ├── api/                   # REST/JSON HTTP 服务
│   │   ├── handler/           # 请求处理器（按领域分组）
│   │   │   ├── asset.go       #   资产相关 Handler
│   │   │   ├── tool.go        #   工具相关 Handler
│   │   │   ├── algorithm.go   #   算法相关 Handler
│   │   │   ├── workflow.go    #   工作流相关 Handler
│   │   │   ├── run.go         #   运行相关 Handler
│   │   │   ├── auth.go        #   认证相关 Handler
│   │   │   └── system.go      #   系统管理 Handler
│   │   ├── dto/               # 请求/响应数据传输对象
│   │   │   ├── asset.go
│   │   │   ├── tool.go
│   │   │   ├── algorithm.go
│   │   │   ├── workflow.go
│   │   │   ├── run.go
│   │   │   └── common.go      #   分页、排序、过滤等通用 DTO
│   │   ├── middleware/        # HTTP 中间件
│   │   │   ├── auth.go        #   JWT 认证
│   │   │   ├── cors.go        #   跨域
│   │   │   ├── logging.go     #   请求日志
│   │   │   ├── ratelimit.go   #   限流
│   │   │   └── tenant.go      #   租户上下文注入
│   │   ├── sse/               # Server-Sent Events
│   │   │   └── broker.go      #   SSE 事件分发
│   │   └── router.go          # 路由注册
│   ├── assistant/             # AI 交互入口（文本/语音）
│   │   ├── handler.go         #   Intent 提交、确认、执行
│   │   ├── speech.go          #   语音转写任务编排（接入 ASR Tool）
│   │   └── mapper.go          #   自然语言到 Intent DTO 映射
│   ├── cli/                   # CLI 命令入口
│   │   ├── root.go            #   根命令
│   │   ├── serve.go           #   启动服务
│   │   ├── migrate.go         #   数据库迁移
│   │   └── tool.go            #   工具管理命令
│   └── mcp/                   # MCP Server
│       ├── server.go          #   MCP 协议服务端
│       └── handler.go         #   MCP 请求处理
│
├── control/                   # Layer 2: 控制层
│   ├── workflow/              # Workflow Engine
│   │   ├── engine.go          #   DAG 执行引擎（Kahn 算法）
│   │   ├── executor.go        #   节点执行编排
│   │   ├── context.go         #   上下文管理（CAS Patch）
│   │   └── resolver.go        #   算法/工具解析
│   ├── agent/                 # Agent Runtime
│   │   ├── runtime.go         #   Run Loop 状态机
│   │   ├── planner.go         #   Plan 阶段（工具选择与参数组装）
│   │   ├── observer.go        #   Observe 阶段（结果评估）
│   │   └── recovery.go        #   Recover 阶段（错误恢复策略）
│   ├── intent/                # Intent Orchestrator
│   │   ├── orchestrator.go    #   意图主流程（parse/plan/confirm/execute）
│   │   ├── compiler.go        #   动作计划与 DAG 生成
│   │   ├── confirmer.go       #   高风险动作确认/审批衔接
│   │   └── executor.go        #   动作执行与恢复
│   └── scheduler/             # 调度器
│       ├── scheduler.go       #   任务调度核心
│       ├── queue.go           #   任务队列（优先级）
│       ├── cron.go            #   定时触发（gocron/v2）
│       └── event.go           #   事件触发
│
├── registry/                  # Layer 3: 注册中心层
│   ├── tool/                  # Tool Registry
│   │   ├── registry.go        #   注册/发现/版本管理
│   │   └── validator.go       #   ToolSpec 校验
│   ├── model/                 # Model Registry
│   │   ├── registry.go        #   模型注册与可用性管理
│   │   └── provider.go        #   Provider 配置管理
│   ├── algorithm/             # Algorithm Library
│   │   ├── library.go         #   算法注册与版本管理
│   │   ├── binding.go         #   实现绑定管理
│   │   └── selector.go        #   实现选择策略
│   └── workflowdef/           # Workflow Definition Registry
│       ├── registry.go        #   工作流定义注册
│       └── revision.go        #   Revision 版本管理
│
├── runtime/                   # Layer 4: 运行时层
│   ├── operator/              # Operator Runtime
│   │   ├── dispatcher.go      #   按 execution_mode 分派
│   │   ├── http.go            #   HTTP 远程调用
│   │   ├── cli.go             #   subprocess 本地命令
│   │   ├── container.go       #   Docker 容器执行
│   │   └── inprocess.go       #   进程内直调
│   ├── mcpclient/             # MCP Runtime
│   │   ├── client.go          #   MCP 客户端连接管理
│   │   └── adapter.go         #   MCP Tool → 统一 Tool 适配
│   └── modelrt/               # Model Runtime
│       ├── router.go          #   模型路由（本地/云端）
│       ├── local.go           #   本地推理引擎适配
│       └── cloud.go           #   云模型 API 适配
│
├── data/                      # Layer 5: 数据与资产层
│   ├── asset/                 # Asset Store
│   │   ├── store.go           #   存储抽象接口实现
│   │   ├── s3.go              #   S3/MinIO 适配
│   │   ├── local.go           #   本地文件系统适配
│   │   └── mediamtx.go        #   MediaMTX 流媒体适配
│   ├── metadata/              # Metadata DB（GORM Repositories）
│   │   ├── repository.go      #   通用 Repository 基础
│   │   ├── asset_repo.go      #   Asset 仓储
│   │   ├── tool_repo.go       #   Tool 仓储
│   │   ├── algorithm_repo.go  #   Algorithm 仓储
│   │   ├── workflow_repo.go   #   Workflow 仓储
│   │   ├── run_repo.go        #   Run 仓储
│   │   ├── intent_repo.go     #   Intent/IntentAction 仓储
│   │   ├── user_repo.go       #   User/Role 仓储
│   │   └── scopes.go          #   租户/可见性/软删除 Scopes
│   ├── event/                 # Event Store
│   │   ├── store.go           #   事件持久化
│   │   ├── publisher.go       #   进程内发布（channel）
│   │   └── bridge.go          #   持久化事件到应用层 EventBus 的桥接
│   ├── context/               # Context Store
│   │   ├── state.go           #   ContextState CRUD
│   │   ├── patch.go           #   Patch 记录
│   │   └── snapshot.go        #   Snapshot 管理
│   └── cache/                 # 缓存抽象
│       ├── cache.go           #   缓存接口定义
│       ├── memory.go          #   进程内 sync.Map 实现
│       └── redis.go           #   Redis 实现（可选）
│
├── observe/                   # Layer 6: 可观测性与安全层
│   ├── trace/                 # 追踪
│   │   ├── tracer.go          #   Trace 上下文管理
│   │   └── span.go            #   Span 创建与结束
│   ├── audit/                 # 审计
│   │   ├── logger.go          #   审计日志写入
│   │   └── query.go           #   审计日志查询
│   ├── policy/                # 策略引擎
│   │   ├── engine.go          #   策略评估引擎
│   │   ├── rules.go           #   内置规则定义
│   │   └── loader.go          #   策略加载（DB/配置）
│   └── budget/                # 预算控制
│       ├── tracker.go         #   成本累计追踪
│       └── limiter.go         #   额度控制与拦截
│
└── domain/                    # 跨层共享领域对象
    ├── asset.go               #   Asset 结构体 + AssetRepository 接口
    ├── tool.go                #   ToolSpec/Tool 结构体 + ToolRepository 接口
    ├── algorithm.go           #   Algorithm/Version/Binding 结构体 + AlgorithmRepository 接口
    ├── run.go                 #   Run/RunEvent 结构体 + RunRepository 接口
    ├── workflow.go            #   Workflow/Node/Edge/Revision 结构体 + WorkflowRepository 接口
    ├── agent.go               #   AgentSession 结构体 + 状态机定义
    ├── context.go             #   ContextSpec/ContextState/Patch/Snapshot 结构体
    ├── user.go                #   User/Role/Permission 结构体 + UserRepository 接口
    ├── policy.go              #   ToolPolicy/PolicyDecision 结构体 + PolicyRepository 接口
    ├── event.go               #   EventBus 接口 + 事件类型枚举
    ├── store.go               #   AssetStore/Cache 接口定义
    ├── errors.go              #   统一错误类型定义
    └── enums.go               #   全局枚举（AssetType/RunStatus/RiskLevel/...）
```

---

## 3. 依赖规则

### 3.1 层间依赖矩阵

```
domain     → 无依赖（纯结构体 + 枚举 + 接口定义）
data       → domain
registry   → domain, data
runtime    → domain, data, registry
control    → domain, data, registry, runtime
observe    → domain, data（可被任何层调用）
access     → domain, control, registry（不直接依赖 data/runtime）
```

可视化表示（`→` 表示"允许依赖"，`✗` 表示"禁止依赖"）：

```
             domain  data  registry  runtime  control  observe  access
domain         -      ✗      ✗         ✗        ✗        ✗       ✗
data           →      -      ✗         ✗        ✗        ✗       ✗
registry       →      →      -         ✗        ✗        ✗       ✗
runtime        →      →      →         -        ✗        ✗       ✗
control        →      →      →         →        -        ✗       ✗
observe        →      →      ✗         ✗        ✗        -       ✗
access         →      ✗      →         ✗        →        ✗       -
```

**特殊说明**：
- `observe` 可被任何层**调用**（通过 domain 中定义的接口），但 `observe` 自身只依赖 `domain` 和 `data`
- `access` 不直接依赖 `data` 和 `runtime`——所有数据访问通过 `control` 或 `registry` 间接完成

### 3.2 domain 包的设计哲学

**核心问题**：为什么 domain 同时包含数据结构和接口定义？

经典 Clean Architecture 将数据结构和接口定义分置于不同的包中（如 `domain/` 和 `port/`）。Goyais 选择将二者合并，原因如下：

1. **避免 Port 层定位模糊**：Port 依赖 Domain（因为接口参数和返回值使用 Domain 类型），但它既不是"业务逻辑"也不是"基础设施"，独立成包意义有限
2. **减少跨层引用**：合并后各层只需导入 `domain` 一个包，无需同时导入 `domain` 和 `port`
3. **接口与数据结构天然耦合**：`AssetRepository` 的方法签名必然引用 `Asset` 结构体，放在同一包中更加自然

**设计选择**：将 domain 定义为"合并了 Entity + Port 角色的纯定义包"。

```go
// internal/domain/asset.go

// === 数据结构（Entity 角色）===

type Asset struct {
    ID        uuid.UUID
    Name      string
    Type      AssetType
    URI       string
    Digest    string
    // ...
}

type AssetType string
const (
    AssetTypeVideo      AssetType = "video"
    AssetTypeImage      AssetType = "image"
    // ...
)

// === 接口定义（Port 角色）===

type AssetRepository interface {
    Create(ctx context.Context, asset *Asset) error
    GetByID(ctx context.Context, id uuid.UUID) (*Asset, error)
    List(ctx context.Context, filter AssetFilter) ([]*Asset, int64, error)
    Update(ctx context.Context, asset *Asset) error
    Delete(ctx context.Context, id uuid.UUID) error
}

type AssetStore interface {
    Put(ctx context.Context, key string, reader io.Reader) error
    Get(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
    PresignGet(ctx context.Context, key string, expires time.Duration) (string, error)
}
```

**这样做的优势**：

1. **依赖反转自然实现**：上层（control/registry）依赖 domain 中的接口，下层（data）实现这些接口——不需要额外的 port 包
2. **单一引用源**：任何层只需导入 `domain` 一个包即可获得所有类型定义和接口契约
3. **零逻辑保证**：domain 包严格只包含结构体定义、枚举常量和接口声明，不包含任何实现代码
4. **编译期检查**：Go 的隐式接口满足机制确保 data 层的实现在编译期被校验

**约束规则**：
- domain 包中 **禁止** 包含任何业务逻辑实现
- domain 包中 **禁止** 依赖任何外部库（标准库 + `github.com/google/uuid` 除外）
- domain 包中的接口 **必须** 以领域概念命名（`AssetRepository`），而非技术概念（`PostgresAssetRepo`）

### 3.3 禁止的依赖路径

以下依赖路径在代码审查中**一票否决**：

| 禁止路径 | 原因 |
|---------|------|
| `domain → *`（任何包） | domain 是依赖链的最内层，不可有外部依赖 |
| `access → data` | Access Layer 不应绕过 Control/Registry 直接操作数据 |
| `access → runtime` | Access Layer 不应直接调用执行器 |
| `data → control` | 数据层不应反向依赖控制层（循环依赖风险） |
| `registry → control` | 注册中心不应依赖控制层（循环依赖风险） |
| `runtime → control` | 运行时不应反向依赖控制层（循环依赖风险） |

### 3.4 observe 的横切调用模式

observe 层（Trace/Audit/Policy/Budget）具有特殊的横切调用权限。其他层通过 domain 中定义的接口调用 observe 的能力：

```go
// internal/domain/policy.go

// PolicyEvaluator 在 domain 中定义接口
type PolicyEvaluator interface {
    Evaluate(ctx context.Context, request PolicyRequest) (PolicyDecision, error)
}

// PolicyRequest 包含评估所需信息
type PolicyRequest struct {
    ToolName    string
    CallerID    uuid.UUID
    RunID       uuid.UUID
    DataScopes  []string
    CostHint    CostHint
}

// PolicyDecision 返回评估结果
type PolicyDecision struct {
    Allowed     bool
    Reason      string
    Conditions  []string   // 附加条件（如需审批）
}
```

```go
// internal/observe/policy/engine.go

// 实现 domain.PolicyEvaluator 接口
type Engine struct {
    repo   domain.PolicyRepository
    rules  []Rule
}

func (e *Engine) Evaluate(ctx context.Context, req domain.PolicyRequest) (domain.PolicyDecision, error) {
    // 策略评估逻辑
}
```

```go
// internal/control/workflow/executor.go

// Control Layer 通过 domain 接口调用 Policy Engine
type NodeExecutor struct {
    policy domain.PolicyEvaluator  // 注入 observe/policy 的实现
    // ...
}

func (e *NodeExecutor) executeNode(ctx context.Context, node domain.WorkflowNode) error {
    // 执行前策略校验
    decision, err := e.policy.Evaluate(ctx, domain.PolicyRequest{
        ToolName: node.ToolName,
        // ...
    })
    if !decision.Allowed {
        return domain.ErrPolicyDenied(decision.Reason)
    }
    // 继续执行...
}
```

---

## 4. 单二进制部署架构

### 4.1 设计目标

Goyais 遵循 **"单二进制，最小依赖"** 的部署理念：

- 一个 Go 二进制文件包含所有子系统
- 最小运行依赖：SQLite + 本地文件存储 + 进程内缓存
- SQL 后端可配置：`sqlite`（默认）/`postgres`/`mysql`，对外功能语义一致
- 前端 Vue SPA 通过 `go:embed` 编译进二进制
- 可选组件通过配置开关按需启用，显式启用但不可达时启动失败（fail-fast）

### 4.2 进程内子系统映射

```
┌─────────────────────────────────────────────────────────────────┐
│                    Goyais 单二进制进程                            │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ HTTP Server (Echo v4)                        :8080       │   │
│  │  ├── REST API (/api/*)                               │   │
│  │  ├── SSE Endpoint (/api/runs/*/events/stream 等)      │   │
│  │  ├── Static Files (go:embed web/dist/)                  │   │
│  │  └── SPA Fallback (index.html)                          │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ MCP Server (可选)                            :8090       │   │
│  │  └── Model Context Protocol Listener                    │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ Control Subsystems (goroutines)                          │   │
│  │  ├── Workflow Engine      (按需启动 goroutine)           │   │
│  │  ├── Agent Runtime        (会话级 goroutine)             │   │
│  │  ├── Scheduler            (gocron/v2 定时循环)           │   │
│  │  └── Event Dispatcher     (channel 消费循环)             │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌──────────────────────────────────────────────────────────┐   │
│  │ 进程内基础设施                                            │   │
│  │  ├── Event Bus        = channel (可选 Redis Pub/Sub)     │   │
│  │  ├── Cache            = sync.Map (可选 Redis)            │   │
│  │  ├── Task Queue       = channel (可选 Redis List)        │   │
│  │  └── Policy Engine    = 内嵌 Go 规则引擎                  │   │
│  └──────────────────────────────────────────────────────────┘   │
│                                                                  │
├──────────────────────── 外部依赖 ────────────────────────────────┤
│                                                                  │
│  [SQLite/MySQL/PostgreSQL] 元数据 + Event Store                  │
│  [Local/MinIO/S3] Asset Store                                    │
│  [Docker]         容器隔离执行 (可选)                             │
│  [Redis]          分布式缓存/锁/队列 (可选)                      │
│  [MediaMTX]       流媒体服务 (可选)                               │
│  [外部 AI API]    云模型调用 (可选)                               │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### 4.3 各子系统的进程内实现策略

#### Event Store

- **实现**：SQL `events` 表（SQLite/MySQL/PostgreSQL）+ 应用层 EventBus
- **写入**：所有 RunEvent 先持久化到 `events` 表
- **订阅**：由应用层 EventBus 负责分发（默认进程内 channel，可选 Redis Pub/Sub）
- **优势**：事件通知不依赖数据库特性，三种数据库保持一致行为

```go
// 写入事件（同步持久化 + 应用层发布）
func (s *EventService) Publish(ctx context.Context, event domain.RunEvent) error {
    // 1. 持久化到 SQL events 表
    if err := s.store.Save(ctx, event); err != nil {
        return err
    }
    // 2. 发布到应用层 EventBus（供 Scheduler/SSE/Audit 消费）
    return s.bus.Publish(ctx, domain.EventFromRunEvent(event))
}
```

#### Cache / Queue

- **默认**：进程内 `sync.Map`（缓存）+ buffered `channel`（队列）
- **可选 Redis**：通过配置 `cache.type: redis` 切换为 Redis 实现
- **接口统一**：domain 中定义 `Cache` 和 `Queue` 接口，data 层提供两种实现

#### Container Runtime

- **原理**：主进程通过 Docker SDK 编排外部容器
- **流程**：拉取镜像 → 创建容器（挂载输入卷）→ 启动 → 等待结束 → 读取输出 → 清理
- **限制**：仅用于 `execution_mode=container` 的 Tool，需要宿主机安装 Docker Engine
- **降级**：未安装 Docker 时，container 模式 Tool 注册会失败并提示

#### MCP Server

- **实现**：同进程额外 listener（独立端口 :8090）
- **协议**：实现 Model Context Protocol 服务端规范
- **能力暴露**：将平台注册的 Tool 作为 MCP Tool 暴露给外部 Agent/LLM
- **可选**：通过配置 `mcp.enabled: true` 开启

#### Policy Engine

- **实现**：内嵌 Go 规则引擎，策略定义存储在 Metadata DB（SQLite/MySQL/PostgreSQL）中
- **评估时机**：Tool 执行前由 Control Layer 主动调用
- **规则类型**：权限检查、数据访问范围、风险等级、预算余额
- **热更新**：策略变更后通过 Event Bus 通知引擎重新加载

#### 前端 SPA

- **技术基线**：Vue + TypeScript + Vite + Tailwind CSS + pnpm（均按 latest stable 策略维护）
- **定位**：前端为统一业务入口（AI 工作台 + 管理控制台），不引入独立后端语义
- **强制构建链路**：`pnpm install -> pnpm build -> 产出 web/dist -> go:embed 打包 -> go build 生成单二进制`
- **嵌入约束**：`web/dist/` 必须通过 `//go:embed` 编译入二进制；运行时不得依赖外部静态目录
- **路由约束**：API 路由 (`/api/*`) 优先匹配；其余路由返回 `index.html`（SPA 前端路由）
- **体验约束**：必须支持响应式设计（桌面/平板/移动端）、深色/浅色模式、国际化（至少 `zh-CN` + `en`）
- **发布门禁**：若缺失 `web/dist` 或未成功嵌入，构建产物判定为不合格，禁止发布

### 4.4 架构全景图（ASCII）

```
┌─────────────────────────────────────────────────────────────────────────┐
│                         Goyais Single Binary                            │
│                                                                         │
│  ╔═══════════════════════════════════════════════════════════════════╗   │
│  ║  Layer 1: ACCESS                                                  ║   │
│  ║  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐            ║   │
│  ║  │REST API │  │  SSE    │  │  CLI    │  │MCP Srv  │            ║   │
│  ║  │ :8080   │  │ :8080   │  │ (stdin) │  │ :8090   │            ║   │
│  ║  └────┬────┘  └────┬────┘  └────┬────┘  └────┬────┘            ║   │
│  ╚═══════╪═══════════╪═══════════╪═══════════╪══════════════════════╝   │
│          │           │           │           │                          │
│  ╔═══════╪═══════════╪═══════════╪═══════════╪══════════════════════╗   │
│  ║  Layer 2: CONTROL │           │           │                      ║   │
│  ║  ┌────────────────┴───────────┴───────────┘                      ║   │
│  ║  │                                                                ║   │
│  ║  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        ║   │
│  ║  │  │  Workflow     │  │    Agent     │  │  Scheduler   │        ║   │
│  ║  │  │  Engine       │  │  Runtime     │  │              │        ║   │
│  ║  │  │  (DAG执行)    │  │  (RunLoop)   │  │  (定时/事件)  │        ║   │
│  ║  │  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘        ║   │
│  ╚══╪═════════╪════════════════╪════════════════╪═══════════════════╝   │
│     │         │                │                │                       │
│  ╔══╪═════════╪════════════════╪════════════════╪═══════════════════╗   │
│  ║  │  Layer 3: REGISTRY       │                │                   ║   │
│  ║  │  ┌──────────┐ ┌─────────┴──┐ ┌───────────┴─┐ ┌───────────┐  ║   │
│  ║  │  │   Tool   │ │  Model     │ │  Algorithm  │ │ Workflow   │  ║   │
│  ║  │  │ Registry │ │ Registry   │ │  Library    │ │   Defs     │  ║   │
│  ║  │  └────┬─────┘ └─────┬─────┘ └──────┬──────┘ └───────────┘  ║   │
│  ╚══╪═══════╪═════════════╪══════════════╪══════════════════════════╝   │
│     │       │             │              │                              │
│  ╔══╪═══════╪═════════════╪══════════════╪══════════════════════════╗   │
│  ║  │  Layer 4: RUNTIME   │              │                          ║   │
│  ║  │  ┌──────────────┐ ┌─┴────────────┐ ┌┴─────────────┐          ║   │
│  ║  │  │  Operator    │ │    MCP       │ │   Model      │          ║   │
│  ║  │  │  Runtime     │ │  Runtime     │ │  Runtime     │          ║   │
│  ║  │  │ (HTTP/CLI/   │ │ (MCP Client) │ │(Local/Cloud) │          ║   │
│  ║  │  │  Container)  │ │              │ │              │          ║   │
│  ║  │  └──────────────┘ └──────────────┘ └──────────────┘          ║   │
│  ╚══╪═══════════════════════════════════════════════════════════════╝   │
│     │                                                                   │
│  ╔══╪═══════════════════════════════════════════════════════════════╗   │
│  ║  │  Layer 5: DATA & ASSET                                        ║   │
│  ║  │  ┌───────────┐ ┌───────────┐ ┌───────────┐ ┌───────────┐    ║   │
│  ║  └─►│  Asset    │ │ Metadata  │ │  Event    │ │  Cache    │    ║   │
│  ║     │  Store    │ │    DB     │ │  Store    │ │           │    ║   │
│  ║     └─────┬─────┘ └─────┬─────┘ └─────┬─────┘ └─────┬─────┘    ║   │
│  ╚═══════════╪═════════════╪═════════════╪═════════════╪════════════╝   │
│              │             │             │             │                 │
│          ┌───┴──────┐  ┌───┴──────┐  ┌───┴──────┐ ┌───┴──────┐        │
│          │Local/    │  │SQL DB    │  │App Event │ │Memory/   │        │
│          │MinIO/S3  │  │(3 drivers)│ │Bus       │ │Redis     │        │
│          └──────────┘  └──────────┘  └──────────┘ └──────────┘        │
│                                                                         │
│  ╔══════════════════════════════════════════════════════════════════╗   │
│  ║  Layer 6: OBSERVE & SECURITY （横切，可被任何层调用）              ║   │
│  ║  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐        ║   │
│  ║  │  Trace   │  │  Audit   │  │  Policy  │  │  Budget  │        ║   │
│  ║  │          │  │  Logger  │  │  Engine  │  │  Control │        ║   │
│  ║  └──────────┘  └──────────┘  └──────────┘  └──────────┘        ║   │
│  ╚══════════════════════════════════════════════════════════════════╝   │
│                                                                         │
│  ╔══════════════════════════════════════════════════════════════════╗   │
│  ║  domain（跨层共享：结构体 + 枚举 + 接口定义）                      ║   │
│  ║  Asset | Tool | Algorithm | Run | Workflow | Context | Errors   ║   │
│  ╚══════════════════════════════════════════════════════════════════╝   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 5. 执行隔离等级

Tool 的执行隔离等级由 `ToolSpec.ExecutionMode` 决定，从轻量到重量依次为：

### 5.1 四种隔离等级详述

#### Level 1: in_process（进程内直调）

```
Goyais 进程
┌────────────────────────────┐
│  Control Layer             │
│    │                       │
│    ▼                       │
│  Runtime Layer             │
│    │                       │
│    ▼                       │
│  ┌──────────────────────┐  │
│  │  Go 函数直接调用      │  │
│  │  (同一 goroutine)     │  │
│  └──────────────────────┘  │
└────────────────────────────┘
```

| 特性 | 说明 |
|------|------|
| 隔离级别 | 无隔离（共享进程内存） |
| 开销 | 零额外开销（函数调用） |
| 适用场景 | 内置工具（格式转换、JSON 处理、内部逻辑）、MCP Tool 适配 |
| 错误边界 | Go panic recovery |
| 超时控制 | `context.WithTimeout` |
| 资源限制 | 无（受限于主进程资源） |

**实现要点**：
```go
type InProcessExecutor struct{}

func (e *InProcessExecutor) Execute(ctx context.Context, spec domain.ToolSpec, input json.RawMessage) (json.RawMessage, error) {
    fn, ok := builtinTools[spec.ID]
    if !ok {
        return nil, domain.ErrToolNotFound(spec.ID)
    }
    return fn(ctx, input)
}
```

#### Level 2: subprocess（子进程隔离）

```
Goyais 进程                    子进程
┌─────────────────────┐       ┌──────────────────────┐
│  Runtime Layer      │       │  exec.Command        │
│    │                │       │                      │
│    ├── stdin ──────────────►│  工具程序             │
│    │                │       │  (Python/Node/Go/...)│
│    ◄── stdout ─────────────┤                      │
│    ◄── stderr ─────────────┤                      │
│                     │       └──────────────────────┘
└─────────────────────┘
```

| 特性 | 说明 |
|------|------|
| 隔离级别 | 进程级（独立进程空间） |
| 开销 | 进程创建开销（毫秒级） |
| 适用场景 | CLI 工具（FFmpeg、Python 脚本、Shell 脚本） |
| 错误边界 | 进程退出码 + stderr |
| 超时控制 | `exec.CommandContext` (SIGKILL) |
| 资源限制 | OS 级进程资源限制 |

**实现要点**：
```go
type SubprocessExecutor struct{}

func (e *SubprocessExecutor) Execute(ctx context.Context, spec domain.ToolSpec, input json.RawMessage) (json.RawMessage, error) {
    cmd := exec.CommandContext(ctx, spec.ExecConfig.Command, spec.ExecConfig.Args...)
    cmd.Stdin = bytes.NewReader(input)
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    if err := cmd.Run(); err != nil {
        return nil, domain.NewToolError("TOOL_EXECUTION", err, stderr.String())
    }
    return stdout.Bytes(), nil
}
```

#### Level 3: container（容器隔离）

```
Goyais 进程                    Docker Container
┌─────────────────────┐       ┌──────────────────────┐
│  Runtime Layer      │       │  隔离的文件系统        │
│    │                │       │  隔离的网络            │
│    ├── volume mount ───────►│  隔离的进程空间        │
│    │                │       │                      │
│    ├── Docker API ─────────►│  工具镜像运行         │
│    │                │       │  (GPU 可选)           │
│    ◄── volume read ────────┤                      │
│                     │       └──────────────────────┘
└─────────────────────┘
```

| 特性 | 说明 |
|------|------|
| 隔离级别 | 容器级（完整隔离） |
| 开销 | 容器启动开销（秒级） |
| 适用场景 | 不可信工具、GPU 推理、重量级处理 |
| 错误边界 | 容器退出码 + 日志 |
| 超时控制 | Docker API 强制停止 |
| 资源限制 | CPU/内存/GPU/网络/存储完整隔离 |

**可选性**：
- 宿主机未安装 Docker 时，container 模式 Tool 注册会失败
- 通过配置 `runtime.container.enabled: false` 显式禁用

**实现要点**：
```go
type ContainerExecutor struct {
    docker *client.Client
}

func (e *ContainerExecutor) Execute(ctx context.Context, spec domain.ToolSpec, input json.RawMessage) (json.RawMessage, error) {
    // 1. 准备输入卷
    inputDir := prepareInputVolume(input)
    // 2. 创建并启动容器
    containerID, err := e.createContainer(ctx, spec.ExecConfig.Image, inputDir)
    // 3. 等待容器结束
    statusCh, errCh := e.docker.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
    // 4. 读取输出
    output := readOutputVolume(containerID)
    // 5. 清理容器
    defer e.docker.ContainerRemove(ctx, containerID, ...)
    return output, nil
}
```

#### Level 4: remote（远程调用）

```
Goyais 进程                    外部服务
┌─────────────────────┐       ┌──────────────────────┐
│  Runtime Layer      │       │                      │
│    │                │       │  HTTP/gRPC 服务       │
│    ├── HTTP POST ──────────►│  (AI API / 微服务)   │
│    │                │       │                      │
│    ◄── HTTP Response ──────┤                      │
│                     │       └──────────────────────┘
└─────────────────────┘
              │
              │ (网络)
              │
```

| 特性 | 说明 |
|------|------|
| 隔离级别 | 完全隔离（独立服务） |
| 开销 | 网络延迟（毫秒到秒级） |
| 适用场景 | 外部 AI API（OpenAI/Anthropic）、远程微服务、第三方 SaaS |
| 错误边界 | HTTP 状态码 + 响应体 |
| 超时控制 | HTTP Client Timeout |
| 资源限制 | 由远程服务控制 |

**实现要点**：
```go
type RemoteExecutor struct {
    httpClient *http.Client
}

func (e *RemoteExecutor) Execute(ctx context.Context, spec domain.ToolSpec, input json.RawMessage) (json.RawMessage, error) {
    req, _ := http.NewRequestWithContext(ctx, "POST", spec.ExecConfig.Endpoint, bytes.NewReader(input))
    req.Header.Set("Content-Type", "application/json")

    // 注入认证头（从 spec 或安全凭证管理获取）
    if spec.ExecConfig.AuthHeader != "" {
        req.Header.Set("Authorization", spec.ExecConfig.AuthHeader)
    }

    resp, err := e.httpClient.Do(req)
    if err != nil {
        return nil, domain.NewToolError("TRANSIENT", err, "remote call failed")
    }
    defer resp.Body.Close()

    body, _ := io.ReadAll(resp.Body)
    if resp.StatusCode >= 400 {
        return nil, domain.NewToolError("DEPENDENCY", fmt.Errorf("HTTP %d", resp.StatusCode), string(body))
    }
    return body, nil
}
```

### 5.2 隔离等级选择指南

```
是否需要网络调用外部服务？ ─── 是 ──► remote
         │
         否
         │
是否需要运行不可信代码？ ─── 是 ──► container
         │
         否
         │
是否需要执行外部程序？ ─── 是 ──► subprocess
         │
         否
         │
         └──► in_process
```

### 5.3 分派逻辑

Runtime Layer 的 Dispatcher 根据 `ToolSpec.ExecutionMode` 选择执行器：

```go
// internal/runtime/operator/dispatcher.go

type Dispatcher struct {
    executors map[domain.ExecutionMode]Executor
}

func (d *Dispatcher) Dispatch(ctx context.Context, spec domain.ToolSpec, input json.RawMessage) (json.RawMessage, error) {
    executor, ok := d.executors[spec.ExecutionMode]
    if !ok {
        return nil, domain.ErrUnsupportedExecutionMode(spec.ExecutionMode)
    }
    return executor.Execute(ctx, spec, input)
}
```

---

## 6. 跨层通信模式

### 6.1 三种通信模式总览

| 模式 | 机制 | 使用场景 | 特征 |
|------|------|----------|------|
| 同步调用 | Go 函数调用 | 层间命令与查询 | 阻塞、直接返回结果 |
| 异步事件 | Application EventBus + Event Store | 状态变更通知、解耦层间协作 | 非阻塞、最终一致 |
| SSE 推送 | Server-Sent Events | 客户端实时更新 | 单向、实时、长连接 |

### 6.2 同步调用（函数调用）

**使用场景**：所有层间的命令式操作。

```
Access ──函数调用──► Control ──函数调用──► Runtime
                        │
                        └──函数调用──► Registry
                        │
                        └──函数调用──► Data
```

**特点**：
- 同一进程内，零网络开销
- 通过 `context.Context` 传递超时、取消与 Trace ID
- 错误通过 Go error 返回，由 Access Layer 统一转换为 HTTP 状态码

**示例——工作流触发**：

```go
// Access Layer (handler)
func (h *WorkflowHandler) Trigger(c echo.Context) error {
    req := new(dto.TriggerRequest)
    if err := c.Bind(req); err != nil {
        return err
    }
    // 同步调用 Control Layer
    run, err := h.engine.Trigger(c.Request().Context(), req.WorkflowID, req.Params)
    if err != nil {
        return err
    }
    return c.JSON(http.StatusCreated, dto.FromRun(run))
}

// Control Layer (engine)
func (e *WorkflowEngine) Trigger(ctx context.Context, workflowID uuid.UUID, params map[string]interface{}) (*domain.Run, error) {
    // 同步调用 Registry Layer
    def, err := e.workflowDefs.GetActiveRevision(ctx, workflowID)
    if err != nil {
        return nil, err
    }
    // 同步调用 Data Layer
    run, err := e.runRepo.Create(ctx, &domain.Run{
        WorkflowID: workflowID,
        RevisionID: def.ID,
        Status:     domain.RunStatusPending,
    })
    if err != nil {
        return nil, err
    }
    // 异步启动执行（见 6.3）
    e.eventBus.Publish(ctx, domain.Event{Type: "run_started", RunID: run.ID})
    return run, nil
}
```

### 6.3 异步事件（Application EventBus + Event Store）

**使用场景**：需要解耦的层间协作，如"创建 Run 后异步启动执行"。

**双层模型设计**：

```
                  ┌─────────────────────────┐
                  │     Event Store         │
                  │ (SQL events table)      │◄──── 查询/回放
  持久化写入 ────►│ (sqlite/mysql/postgres) │
                  └───────────┬─────────────┘
                              │
                              ▼
                  ┌─────────────────────────┐
                  │ Application EventBus    │
                  │ (memory channel/redis)  │
                  └───────────┬─────────────┘
                              │
              ┌───────────────┼───────────────┐
              ▼               ▼               ▼
         [Scheduler]    [SSE Broker]    [Event Handler]
```

**为什么需要双层模型**：
1. **Event Store（持久化）**：保证事件不丢失，支持回放与审计
2. **Application EventBus（实时）**：低延迟分发，不依赖数据库通知能力

**事件流转过程**：

```
1. 产生事件
   Control Layer → eventBus.Publish(event)

2. 持久化
   EventBus → Event Store (INSERT INTO events)

3. 通知
   EventBus → 本地 channel 或 Redis Pub/Sub

4. 进程内分发
   subscriber（Scheduler/SSE/Audit）消费事件

5. 消费
   Scheduler: 收到 run_started → 启动执行 goroutine
   SSE Broker: 收到 node_finished → 推送给订阅客户端
   Audit Logger: 收到 tool_failed → 写入审计日志
```

**事件类型定义**（domain/event.go）：

```go
type EventType string

const (
    // Run 生命周期
    EventRunStarted    EventType = "run_started"
    EventRunFinished   EventType = "run_finished"
    EventRunPaused     EventType = "run_paused"
    EventRunResumed    EventType = "run_resumed"
    EventRunCancelled  EventType = "run_cancelled"
    EventRunRetried    EventType = "run_retried"

    // 节点执行
    EventNodeStarted   EventType = "node_started"
    EventNodeFinished  EventType = "node_finished"
    EventNodeFailed    EventType = "node_failed"
    EventNodeSkipped   EventType = "node_skipped"
    EventNodeRetry     EventType = "node_retry"
    EventSubWorkflowStarted  EventType = "sub_workflow_started"
    EventSubWorkflowFinished EventType = "sub_workflow_finished"

    // 工具调用
    EventToolCalled         EventType = "tool_called"
    EventToolSucceeded      EventType = "tool_succeeded"
    EventToolFailed         EventType = "tool_failed"
    EventToolTimedOut       EventType = "tool_timed_out"
    EventToolRetryScheduled EventType = "tool_retry_scheduled"

    // 上下文
    EventContextPatchApplied   EventType = "context_patch_applied"
    EventContextConflict       EventType = "context_conflict"
    EventContextSnapshotCreated EventType = "context_snapshot_created"

    // Agent
    EventAgentPlan            EventType = "agent_plan"
    EventAgentAct             EventType = "agent_act"
    EventAgentObserve         EventType = "agent_observe"
    EventAgentRecover         EventType = "agent_recover"
    EventAgentEscalation      EventType = "agent_escalation"
    EventAgentSessionStarted  EventType = "agent_session_started"
    EventAgentSessionFinished EventType = "agent_session_finished"

    // Intent
    EventIntentReceived         EventType = "intent_received"
    EventIntentParsed           EventType = "intent_parsed"
    EventIntentPlanned          EventType = "intent_planned"
    EventIntentPlanAdjusted     EventType = "intent_plan_adjusted"
    EventIntentConfirmed        EventType = "intent_confirmed"
    EventIntentRejected         EventType = "intent_rejected"
    EventIntentExecutionStarted EventType = "intent_execution_started"
    EventIntentExecutionFinished EventType = "intent_execution_finished"
    EventIntentExecutionFailed  EventType = "intent_execution_failed"

    // 策略与审批
    EventPolicyEvaluated   EventType = "policy_evaluated"
    EventPolicyBlocked     EventType = "policy_blocked"
    EventApprovalRequested EventType = "approval_requested"
    EventApprovalResolved  EventType = "approval_resolved"

    // 资产
    EventAssetCreated       EventType = "asset_created"
    EventAssetDerived       EventType = "asset_derived"
    EventStreamSliceCreated EventType = "stream_slice_created"

    // 预算治理
    EventBudgetWarning  EventType = "budget_warning"
    EventBudgetExceeded EventType = "budget_exceeded"
)
```

**EventBus 接口**（domain/event.go）：

```go
type EventBus interface {
    // Publish 发布事件（持久化 + 进程内通知）
    Publish(ctx context.Context, event Event) error

    // Subscribe 订阅事件（进程内 channel）
    Subscribe(ctx context.Context, filter EventFilter) (<-chan Event, error)

    // Unsubscribe 取消订阅
    Unsubscribe(ctx context.Context, ch <-chan Event) error
}

type EventFilter struct {
    Types   []EventType   // 事件类型过滤
    RunID   *uuid.UUID    // 按 Run 过滤
    Since   *time.Time    // 时间起点
}
```

### 6.4 SSE 推送（客户端实时更新）

**使用场景**：前端实时展示 Run 进度、节点状态变更、Agent 决策链路。

**架构**：

```
[浏览器]                [Goyais 服务端]
   │                        │
   │  GET /api/runs/{id}/events/stream
   │  Accept: text/event-   │
   │  stream                │
   ├───────────────────────►│
   │                        │
   │   event: run_event     │
   │◄───────────────────────┤
   │   data: {"type":"node_started","node_id":"..."} │
   │                        │
   │   event: run_event     │
   │◄───────────────────────┤
   │   data: {"type":"tool_called","tool_call_id":"..."} │
   │                        │
   │   event: run_event     │
   │◄───────────────────────┤
   │   data: {"type":"node_finished","payload":{...}}  │
   │                        │
   │   event: run_event     │
   │◄───────────────────────┤
   │   data: {"type":"run_finished","status":"completed"}  │
   │                        │
```

**SSE Broker 实现**：

```go
// internal/access/api/sse/broker.go

type Broker struct {
    eventBus    domain.EventBus
    clients     sync.Map  // map[string][]chan domain.Event
}

func (b *Broker) ServeSSE(c echo.Context) error {
    runID := c.Param("id")
    w := c.Response()
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Flush()

    // 订阅事件
    filter := domain.EventFilter{RunID: parseUUID(runID)}
    ch, _ := b.eventBus.Subscribe(c.Request().Context(), filter)
    defer b.eventBus.Unsubscribe(c.Request().Context(), ch)

    for {
        select {
        case event := <-ch:
            frame := map[string]any{
                "type":      event.Type,
                "seq":       event.Seq,
                "run_id":    event.RunID,
                "timestamp": event.Timestamp,
                "payload":   event.Payload,
            }
            fmt.Fprintf(w, "event: run_event\n")
            fmt.Fprintf(w, "data: %s\n\n", mustJSON(frame))
            w.Flush()
        case <-c.Request().Context().Done():
            return nil
        }
    }
}
```

### 6.5 通信模式选择指南

```
需要立即获得返回值？ ─── 是 ──► 同步调用（函数调用）
         │
         否
         │
需要推送到客户端？ ─── 是 ──► SSE（Server-Sent Events）
         │
         否
         │
         └──► 异步事件（Application EventBus + Event Store）
```

### 6.6 完整通信流程示例

以"用户手动触发工作流并实时查看进度"为例，展示三种通信模式的协作：

```
[前端]                    [Access]           [Control]          [Data]
  │                         │                   │                 │
  │ POST /api/workflows/{id}/runs               │                 │
  ├────────────────────────►│                   │                 │
  │                         │ ──同步调用──────►  │                 │
  │                         │                   │ ──同步调用──►   │
  │                         │                   │ (创建Run)       │
  │                         │                   │ ◄──返回Run──    │
  │                         │                   │                 │
  │                         │                   │ ──异步事件──►   │
  │                         │                   │ (run_started)   │
  │                         │ ◄──返回Run──────  │                 │
  │ ◄──201 Created─────────│                   │                 │
  │                         │                   │                 │
  │ GET /api/runs/{id}/events/stream            │                 │
  │ (SSE连接)               │                   │                 │
  ├────────────────────────►│                   │                 │
  │                         │ [SSE Broker 订阅] │                 │
  │                         │                   │                 │
  │                         │      [Scheduler 消费 run_started]   │
  │                         │                   │                 │
  │                         │                   │ ──执行DAG──►    │
  │                         │                   │                 │
  │ ◄──SSE: run_event(node_started)────────────│                 │
  │                         │                   │                 │
  │ ◄──SSE: run_event(tool_called)─────────────│                 │
  │                         │                   │                 │
  │ ◄──SSE: run_event(tool_succeeded)──────────│                 │
  │                         │                   │                 │
  │ ◄──SSE: run_event(node_finished)───────────│                 │
  │                         │                   │                 │
  │ ◄──SSE: run_event(run_finished)────────────│                 │
  │                         │                   │                 │
```

---

## 7. 启动流程与依赖注入

### 7.1 应用启动顺序

```go
// cmd/server/main.go

func main() {
    // 1. 加载配置
    cfg := config.Load()

    // 2. 初始化 Layer 5: Data（按 driver/type 选择实现）
    db := initDatabase(cfg.DB.Driver, cfg.DB)        // sqlite | postgres | mysql
    assetStore := initAssetStore(cfg.Storage)        // local | minio | s3
    cache := initCache(cfg.Cache)
    eventBus := initEventBus(cfg.Event)              // memory | redis
    eventStore := initEventStore(db, eventBus)       // 持久化 + 应用层发布

    // 3. 可选组件健康检查（fail-fast）
    mustHealthCheckDatabase(db)
    if cfg.Cache.Type == "redis" {
        mustHealthCheckRedis(cfg.Cache.Redis)
    }
    if cfg.Storage.Type == "minio" || cfg.Storage.Type == "s3" {
        mustHealthCheckObjectStorage(cfg.Storage)
    }
    if cfg.MediaMTX.Enabled {
        mustHealthCheckMediaMTX(cfg.MediaMTX)
    }

    // 4. 初始化 Layer 6: Observe
    tracer := trace.New(cfg.Trace)
    auditor := audit.New(db)
    policyEngine := policy.New(db)
    budgetCtrl := budget.New(db)

    // 5. 初始化 Layer 3: Registry
    toolRegistry := tool.NewRegistry(db)
    modelRegistry := model.NewRegistry(db)
    algorithmLib := algorithm.NewLibrary(db)
    workflowDefs := workflowdef.NewRegistry(db)

    // 6. 初始化 Layer 4: Runtime
    dispatcher := operator.NewDispatcher(cfg.Runtime)
    mcpRuntime := mcpclient.New(cfg.MCP)
    modelRuntime := modelrt.New(modelRegistry, cfg.Model)

    // 7. 初始化 Layer 2: Control
    engine := workflow.NewEngine(workflowDefs, dispatcher, algorithmLib, eventStore, policyEngine)
    agentRT := agent.NewRuntime(toolRegistry, dispatcher, policyEngine, budgetCtrl)
    intentOrch := intent.NewOrchestrator(agentRT, engine, toolRegistry, policyEngine, auditor, eventStore)
    scheduler := scheduler.New(engine, agentRT, eventStore, cfg.Scheduler)

    // 8. 初始化 Layer 1: Access
    router := api.NewRouter(engine, scheduler, agentRT, intentOrch, toolRegistry, algorithmLib, workflowDefs)
    sseBroker := sse.NewBroker(eventBus)
    mustCheckEmbeddedWebAssets(router)               // index.html + 静态资源入口必须可用

    // 9. 启动
    scheduler.Start()
    if cfg.MCP.Enabled {
        go mcp.Serve(cfg.MCP.Port, toolRegistry)
    }
    router.Start(cfg.Server.Port)
}
```

### 7.2 依赖注入原则

- **构造函数注入**：所有依赖通过构造函数参数传入
- **接口依赖**：上层仅依赖 domain 中定义的接口，不依赖具体实现
- **无框架 DI**：不使用 Wire/Dig 等 DI 框架，保持显式依赖关系

### 7.3 可选依赖健康检查与 Fail-Fast

- 默认最小化部署（SQLite + local + memory）不依赖外部服务，可直接启动
- 显式启用的外部依赖（PostgreSQL/MySQL、Redis、MinIO/S3、MediaMTX）必须在启动阶段通过健康检查
- 任何已启用依赖连接失败即中止启动，避免静默回退导致运行语义漂移
- 所有数据库驱动（sqlite/mysql/postgres）对外 API 语义一致，差异仅允许存在于非功能层（性能/运维）
- 启动阶段必须校验二进制内嵌前端资源（`index.html` 与静态资源入口）可用，缺失即启动失败
- 目标环境即使无 Node/pnpm，也必须可由单二进制直接提供前端页面与 API 服务

---

## 8. 全功能 AI 交互契约

### 8.1 能力覆盖原则

- 平台所有功能域必须存在对应 Intent Action（身份、权限、设置、资产、工作流、运行控制、系统管理）
- 页面/API 入口与 AI 入口属于并行入口，能力语义保持一致
- 任一写操作（包括配置变更、权限变更、执行触发）必须进入 Policy Engine 校验链路
- 中高风险写操作必须进入确认/审批流程，不允许绕过

### 8.2 Intent 编排要求

- Intent Orchestrator 负责统一解析、规划、确认、执行，不允许各模块私有化“旁路执行”
- 所有 AI 触发动作必须生成可审计事件（`intent_*` + `policy_*` + 领域事件）
- 失败恢复与重试策略在 Control Layer 统一管理，保持 API/AI 执行行为一致

### 8.3 业务一致性验收契约

同一能力在 AI/API/UI 三入口必须满足以下一致性项：

| 一致性项 | 验收要求 |
|----------|----------|
| 输入约束 | 参数必填、类型、范围、默认值一致 |
| 权限检查 | 相同角色在三入口得到相同授权结果 |
| 审批链路 | 风险分级与审批触发条件一致 |
| 输出结构 | 成功/失败返回结构与关键字段一致 |
| 错误语义 | 错误码、错误分类、可读提示语义一致 |

每个能力发布前必须产出一致性验收记录（Entry Parity Record），至少包含：

1. 能力标识与版本
2. 三入口请求样例与响应样例
3. 权限与审批对照结果
4. 差异项与处理结论

### 8.4 前端入口一致性补充

- 前端 UI 仅作为能力入口层，所有动作必须映射到同一 `capability_id` / `intent_action`
- 前端不得引入绕过 Intent Orchestrator 或 Policy Engine 的私有执行路径
- 前端触发写操作必须遵循与 AI/API 相同的权限校验、审批触发、审计事件要求
- 响应式布局、主题切换、国际化仅影响呈现，不得改变能力语义与权限结果

---

## 9. 业务运行协议（非商业化）

### 9.1 运行协同规则

运行中的业务协同遵循：

1. 任务分派：发起人、执行者、审核者必须可追溯
2. 审批升级：超时或冲突自动升级到上级审核角色
3. 人工接管：命中高风险或异常阈值后自动冻结并转人工
4. 终止与重试：终止需记录原因；重试需携带上一次失败证据
5. 归档：所有任务必须绑定 run_id/trace_id 与最终产物

### 9.2 关键异常场景 SOP

| 异常场景 | SOP |
|----------|-----|
| 工具失败 | 按重试策略执行，超限后转人工裁决 |
| 依赖不可达 | 触发降级路径；无降级路径时终止并告警 |
| 审批超时 | 自动升级；高风险默认拒绝并回退 |
| 上下文冲突 | 冻结冲突节点，人工合并后继续执行 |

### 9.3 文档层公共契约类型

```go
type BusinessScenarioSpec struct {
    ScenarioID        string
    Goal              string
    TriggerCondition  string
    Inputs            []string
    Outputs           []string
    DefinitionOfDone  string
}

type RoleTaskFlowSpec struct {
    Role              string
    AllowedActions    []string
    ApprovalLinks     []string
    HandoverPoints    []string
}

type TaskOutcomeContract struct {
    SuccessCriteria   []string
    PartialCriteria   []string
    FailureCriteria   []string
    EvidenceFields    []string
}

type HumanReviewPolicy struct {
    TriggerRules      []string
    ReviewerRole      string
    TimeoutSeconds    int64
    TimeoutAction     string
    RejectFlow        string
}

type EntryParityContract struct {
    CapabilityID      string
    CheckItems        []string
    Score             float64
    Differences       []string
}

type BusinessSLOSpec struct {
    Metric            string
    Window            string
    Threshold         string
    AlertLevel        string
}

type FailurePlaybookSpec struct {
    FailureType       string
    DegradeStrategy   string
    HumanTakeover     string
    RecoverCriteria   string
}

type BinaryArtifactContract struct {
    ArtifactType             string   // single_go_binary
    EmbeddedWebDistRequired  bool
    ExternalStaticDirAllowed bool
    StartupCheckRequired     bool
    ReleaseGateChecks        []string
}

type FrontendEmbedBuildSpec struct {
    InstallCommand   string
    BuildCommand     string
    DistPath         string
    EmbedPattern     string
    GoBuildCommand   string
    FailureConditions []string
}

type TechStackVersionPolicy struct {
    Component       string
    VersionRule     string   // latest_stable
    SourceOfTruth   string
    CheckCycle      string
    GateRule        string
}

type FrontendUXContract struct {
    ResponsiveBreakpoints []string
    SupportsLightMode     bool
    SupportsDarkMode      bool
    SupportedLocales      []string
    FallbackLocale        string
}
```

### 9.4 P0/P1 排期（仅业务完整性）

| 优先级 | 周期 | 交付 |
|--------|------|------|
| P0 | 2 周 | 完成 `BusinessScenarioSpec`、`RoleTaskFlowSpec`、`TaskOutcomeContract`、`EntryParityContract` 文档化，并定义首批 10 个任务闭环与 DoD |
| P0 | 1 周 | 完成 `BinaryArtifactContract`、`FrontendEmbedBuildSpec`、`TechStackVersionPolicy`、`FrontendUXContract` 文档化，并形成发布门禁清单 |
| P0 | 1 周 | 完成 `HumanReviewPolicy`、`FailurePlaybookSpec`，落地人工复核与异常回退协议 |
| P1 | 2 周 | 完成 `BusinessSLOSpec` 与指标采集口径，补齐附录 D |
| P1 | 1 周 | 完成运行协议检查表，形成发布前业务验收清单 |

### 9.5 业务验收场景

1. 闭环完整性：每个任务都能跑通 `触发 -> 执行 -> 复核 -> 产物确认 -> 归档`
2. 入口一致性：同一任务从 AI/API/UI 触发结果一致
3. 人工介入：高风险或低置信任务必须转人工复核
4. 异常回退：依赖失败、工具失败、审批超时均进入定义回退路径
5. 结果判定：无证据字段的任务不得标记业务完成
6. 可追溯性：每个任务可关联完整事件链和最终产物
7. 构建完整性：标准构建链路产物必须为内嵌前端资源的单一二进制
8. 前端体验一致性：桌面/平板/移动端显示正常，深浅色模式与国际化切换不影响能力语义

---

## 附录 A：包依赖校验方法

使用 Go 标准工具链校验依赖规则：

```bash
# 检查 domain 包是否有外部依赖
go list -deps ./internal/domain/ | grep -v "standard library" | grep -v "github.com/google/uuid"

# 检查 access 是否误依赖 data
go list -deps ./internal/access/... | grep "internal/data"

# 检查 access 是否误依赖 runtime
go list -deps ./internal/access/... | grep "internal/runtime"

# 检查是否存在循环依赖
go vet ./internal/...
```

建议在 CI 中加入依赖规则校验脚本，违反即阻断合并。

## 附录 B：配置文件结构

```yaml
# configs/config.yaml

server:
  port: 8080

db:
  driver: sqlite       # sqlite | postgres | mysql
  sqlite:
    path: ./data/goyais.db
  postgres:
    host: localhost
    port: 5432
    user: goyais
    password: goyais
    dbname: goyais
    sslmode: disable
  mysql:
    host: localhost
    port: 3306
    user: goyais
    password: goyais
    dbname: goyais
    params: "parseTime=true&charset=utf8mb4"

storage:
  type: local          # local | minio | s3
  local:
    base_path: ./data/assets
  minio:
    endpoint: "localhost:9000"
    bucket: "goyais"
    access_key: ""
    secret_key: ""
    use_ssl: false
  s3:
    region: "us-east-1"
    endpoint: ""
    bucket: "goyais"
    access_key: "__FROM_ENV__"
    secret_key: "__FROM_ENV__"

cache:
  type: memory         # memory | redis
  redis:
    addr: "localhost:6379"
    password: ""
    db: 0

event:
  bus: memory          # memory | redis
  redis:
    addr: "localhost:6379"
    password: ""
    db: 0
    channel: "goyais.run_events"

mcp:
  enabled: false
  port: 8090

runtime:
  container:
    enabled: false
    docker_host: "unix:///var/run/docker.sock"

mediamtx:
  enabled: false
  api_address: "http://localhost:9997"
  hls_address: "http://localhost:8888"
  rtsp_address: "rtsp://localhost:8554"
  rtmp_address: "rtmp://localhost:1935"

auth:
  jwt_secret: "change-me-in-production"
  access_expire: "2h"
  refresh_expire: "168h"

scheduler:
  enabled: true
  worker_count: 4
  queue_size: 100

trace:
  enabled: true
  sample_rate: 1.0

audit:
  enabled: true
  retention_days: 90
```

## 附录 C：典型配置场景

### C.1 场景 A（默认本地最小部署）

- `db.driver=sqlite`
- `storage.type=local`
- `cache.type=memory`
- `event.bus=memory`
- `mediamtx.enabled=false`

### C.2 场景 B（MySQL + Redis + MinIO）

- `db.driver=mysql`
- `cache.type=redis`
- `event.bus=redis`
- `storage.type=minio`
- 启动时对 MySQL、Redis、MinIO 进行健康检查，任一失败即中止启动

### C.3 场景 C（PostgreSQL + Redis + S3 + MediaMTX）

- `db.driver=postgres`
- `cache.type=redis`
- `event.bus=redis`
- `storage.type=s3`
- `mediamtx.enabled=true`
- 启动时对 PostgreSQL、Redis、S3、MediaMTX 进行健康检查，任一失败即中止启动

### C.4 场景 D（MediaMTX 已启用但不可达）

- 条件：`mediamtx.enabled=true` 且 `mediamtx.api_address` 不可达
- 结果：启动失败（fail-fast），不自动降级为“关闭流媒体能力”

## 附录 D：业务 SLO 指标（非商业化）

| 指标 | 定义 | 默认阈值（建议） |
|------|------|------------------|
| 业务完成率 | 成功任务数 / 总任务数 | `>= 95%` |
| 人工介入率 | 进入人工复核任务数 / 总任务数 | `<= 20%` |
| 一次通过率 | 无重试且直接完成任务数 / 总任务数 | `>= 85%` |
| 复核时延 | 从复核触发到复核完成的 P95 耗时 | `<= 30 min` |
| 回退成功率 | 触发回退后恢复到可继续状态的比例 | `>= 98%` |
| 可追溯完整率 | 可关联 run_id/trace_id 与产物的任务比例 | `= 100%` |

## 附录 E：前后端技术栈与版本治理（Latest Stable）

### E.1 技术栈基线

| 层 | 技术栈 | 版本策略 |
|----|--------|----------|
| 前端 | Vue + TypeScript + Vite + Tailwind CSS + pnpm | latest stable |
| 后端 | Go + 核心后端依赖 | latest stable |

### E.2 版本治理规则

- 文档不锁定 patch 号，仅声明 `latest stable`
- 每次迭代前执行版本检查并记录结果
- 升级导致契约变化时，必须先更新本文档中的契约与验收条目
- 版本升级不得破坏“单二进制内嵌前端”发布模型

### E.3 单二进制前端内嵌构建链路

1. `pnpm install`
2. `pnpm build` 产出 `web/dist`
3. `go:embed web/dist/*` 将静态资源编译进 Go 程序
4. `go build` 产出单一可执行文件

### E.4 构建验收门禁

1. `web/dist/index.html` 存在且构建成功
2. 二进制启动后可返回前端入口页面
3. 目标机器在无 Node/pnpm 环境下仍可正常访问前端页面
4. 任一门禁失败则构建/发布流程失败

### E.5 前端体验基线

- 响应式：至少覆盖桌面、平板、移动端三档布局
- 主题：支持浅色与深色模式，主题切换不影响功能可用性
- 国际化：至少支持 `zh-CN` 与 `en`，缺失翻译时回退到默认语言
