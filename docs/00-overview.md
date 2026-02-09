# Goyais 总览

> 本文档为 Goyais 的顶层设计概览，定义产品定位、核心概念、数据流与外部依赖，作为所有后续设计文档的共识基准。

最后更新：2026-02-09

---

## 1. 产品定位

### 1.1 一句话定义

**Goyais = 全意图驱动的智能 Agent + 多模态 AI 原生平台（Go 实现）**

### 1.2 核心描述

Goyais 是一个以 Go 语言构建的多模态 AI 编排与执行平台。它接收多种模态的输入（视频、图片、音频、文档、Excel、流媒体），通过统一的工具/模型/算法注册与 DAG 工作流编排进行处理，输出结构化结果、新资产或诊断报告。

平台默认以 AI 交互为主入口：用户可通过对话式或语音式输入完成平台级操作（例如创建用户/角色、调整权限与设置、上传与处理资产、触发运行）。

平台提供：

- **统一上下文管理**：工作流与 Agent 会话共享可版本化、可回放的运行上下文
- **任务全生命周期管理**：从触发、调度、执行到产物沉淀的完整链路
- **全 AI 操作入口**：自然语言/语音输入统一编译为可执行动作计划与 DAG
- **可观测性**：全链路 trace/run_event 事件化追踪，支持回放与审计
- **审计与安全**：工具级权限控制、数据访问限制、高风险操作审批闸门
- **统一权限治理**：多租户 RBAC + 工具级细粒度授权
- **国际化能力**：按用户/请求语言环境输出 UI 与 API 文案（首批支持 `zh-CN` + `en`）

### 1.3 核心目标

让 **"平台能力调用"** 成为可编排、可追溯、可治理的 AI 原生工程能力。

具体而言：

1. **可编排**：通过 DAG 工作流与 Agent Run Loop，将离散的 AI 能力组合为复杂业务流程
2. **可追溯**：每一次执行、每一个工具调用、每一次上下文变更均有完整事件记录，支持任意时间点回放
3. **可治理**：工具执行受策略引擎管控，具备预算控制、权限限制、风险等级审批与数据访问隔离能力
4. **可对话操作**：用户通过文本/语音即可触发平台全域行为（身份、权限、设置、资产、运行）
5. **可本地化交互**：同一能力链路支持不同 locale 的消息、错误与审批提示

### 1.4 核心设计哲学

秉承 **"业务 = 配置，能力 = 插件，执行 = 引擎"** 理念：

| 维度 | 设计方式 |
|------|----------|
| 配置 | IntentPlan + 工作流 + ContextSpec + ToolSpec + 算法绑定 |
| 插件 | 统一 Tool 抽象（合并所有执行模式） |
| 引擎 | Intent 编译器 + DAG 引擎 + Agent Runtime + 策略引擎 |

---

## 2. 非目标

明确界定 Goyais **不做**的事情，避免范围蔓延：

### 2.1 不做通用标注/数据平台

- Goyais 不内建数据标注工具（如 Labelme、CVAT 等功能）
- 可对接外部标注平台的输出作为 Asset 输入
- 标注结果可作为 structured 类型 Asset 进入平台参与工作流

### 2.2 不做训练平台

- Goyais 不内建模型训练能力（不管理 GPU 训练集群、不实现训练框架）
- 可通过 Tool 接入外部训练服务（如调用训练 API、触发训练流水线）
- 训练产出的模型可注册到 Model Registry 供后续推理使用

### 2.3 不做通用 BI/报表平台

- 运营看板和审计视图仅服务于平台自身运行数据
- 不提供通用数据分析或自定义报表能力

### 2.4 不做容器编排平台

- 容器隔离仅作为 Tool 执行的可选隔离等级
- 不提供通用的容器集群管理、服务发现或负载均衡能力

---

## 3. 核心概念

Goyais 定义 **五个一等对象（First-Class Objects）**，它们构成系统的概念核心。

### 3.1 Asset（资产）

**定义**：一切输入/输出载体的统一抽象。

**核心原则**：
- Asset 只存引用（指针）与摘要（元数据），原始内容进 Asset Store（S3/MinIO/本地文件系统）
- Asset 不可变——处理产生新 Asset，通过 `parent_id` 建立派生关系
- Asset 是工作流的输入起点，也是执行的输出终点

**支持类型**：

| 类型 | 说明 | 典型来源 |
|------|------|----------|
| `video` | 视频文件 | 上传、录制、算子产出 |
| `image` | 图片文件 | 上传、帧提取、算子产出 |
| `audio` | 音频文件 | 上传、音轨分离、算子产出 |
| `document` | 文档（PDF/Word/Excel 等） | 上传、报告生成 |
| `stream` | 实时流媒体 | MediaMTX 接入（RTSP/RTMP/HLS/WebRTC） |
| `structured` | 结构化数据（JSON/CSV） | 算子结果、外部数据导入 |
| `text` | 纯文本 | 用户输入、LLM 输出、转录结果 |

**StreamAsset 特殊性**：
- 关联 MediaMTX path，具备实时预览、状态监控与录制能力
- 带时间轴切片索引（TimeSliceIndex），支持按时间范围检索流片段
- 录制产出的视频段自动注册为 `video` 类型子 Asset

**Asset 元数据结构**（概念示意；完整定义见 `02-domain-model.md`）：

```go
type Asset struct {
    ID        uuid.UUID
    TenantID  uuid.UUID
    OwnerID   uuid.UUID
    Name      string
    Type      AssetType          // video|image|audio|document|stream|structured|text
    URI       string             // 统一资源定位符（s3:// / file:// / rtsp:// / https://）
    Digest    string             // 内容摘要/哈希（推荐 sha256）
    MimeType  string
    Size      int64
    Metadata  JSONMap            // 类型相关扩展元数据
    ParentID  *uuid.UUID         // 派生来源
    Tags      []string
    Status    AssetStatus
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

### 3.2 Tool（工具）

**定义**：可执行能力的统一抽象。

Tool 统一涵盖以下执行模式：

| 执行模式 | 说明 |
|----------|------|
| `remote` | HTTP/gRPC 远程调用（外部服务、AI Model API） |
| `subprocess` | 本地子进程执行（CLI 工具、脚本） |
| `in_process` | 进程内直调（内置工具、MCP Tool 适配） |
| `container` | Docker 容器隔离执行（不可信工具、GPU 推理） |

此外，Tool 可由 Algorithm 绑定引用，作为算法意图的具体实现。

**ToolSpec（工具规格）** 是 Tool 的完整契约定义，包含执行所需的一切声明信息。

> 下面为概念示意，完整字段与类型以 `02-domain-model.md` 为权威定义。

```go
type ToolSpec struct {
    ID                  uuid.UUID
    TenantID            uuid.UUID
    Name                string
    Code                string
    Description         string
    Category            ToolCategory
    InputSchema         JSONSchema
    OutputSchema        JSONSchema
    SideEffects         []SideEffect
    RiskLevel           RiskLevel
    RequiredPermissions []string
    ExecutionMode       ExecutionMode
    TimeoutMs           int64            // 毫秒
    RetryPolicy         RetryPolicy
    Idempotent          bool
    CostHint            CostHint
    Determinism         DeterminismType
    DataAccess          DataAccessSpec
    Version             string
    Status              ToolStatus
    Config              JSONMap
    CreatedAt           time.Time
    UpdatedAt           time.Time
}

type DataAccessSpec struct {
    BucketPrefixes  []string    // 允许访问的存储桶前缀
    DBScopes        []string    // 允许访问的数据库域
    DomainWhitelist []string    // 允许访问的网络域名
    ReadScopes      []string    // 可读数据范围: ["asset:tenant/*", "context:current_run/*"]
    WriteScopes     []string    // 可写数据范围: ["asset:output/*"]
}
```

**设计要点**：
- ToolSpec 是声明式的——描述"是什么"与"能做什么"，而非"怎么执行"
- 执行细节由 Runtime Layer 根据 `execution_mode` 分派到对应运行时
- 安全策略由 Policy Engine 根据 ToolSpec 中的声明进行执行前校验

### 3.3 Algorithm（算法）

**定义**：独立于 Tool 的一等对象，表示"做什么"（意图）而非"怎么做"（实现）。

**为什么算法需要独立建模**：

Tool 回答的是"如何执行一个具体操作"，而 Algorithm 回答的是"用什么方法解决一个特定问题"。同一个算法（如"人脸检测"）可以有多种实现（不同模型、不同服务商、不同精度/成本配比），每种实现绑定到不同的 Tool。

**三层结构**：

```
Algorithm（意图层）
  └── AlgorithmVersion（版本层）
        └── ImplementationBinding（实现层）→ Tool
```

| 层级 | 职责 | 示例 |
|------|------|------|
| Algorithm | 定义算法意图、场景、分类 | "人脸检测" / "语音转文字" / "视频摘要" |
| AlgorithmVersion | 版本化管理、默认实现指定 | v1.0（稳定）/ 2.0-beta（高精度） |
| ImplementationBinding | 绑定具体 Tool + 性能画像 | Tool-A（GPU，高精度，慢）/ Tool-B（CPU，低成本，快） |

**实现选择策略**：
- `default`：使用版本指定的默认实现（稳定优先）
- `high_accuracy`：选择精度最高的实现（质量优先）
- `low_cost`：选择成本最低的实现（预算优先）
- `low_latency`：选择延迟最低的实现（速度优先）
- `balanced`：按精度/成本/延迟综合平衡

**EvaluationProfile（评测证据）**：
- 每个实现可关联评测报告（数据集、指标、置信度）
- 评测结果作为实现选择的参考依据
- 支持 A/B 对比与持续评估

### 3.4 Run（运行）

**定义**：一切执行的统一容器。

所有执行活动统一为 Run，共享追踪标识体系。

**Run 的三种形态**：

| 形态 | 说明 | 典型场景 |
|------|------|----------|
| WorkflowRun | DAG 工作流的一次执行 | 定时触发的视频分析流水线 |
| AgentSession | Agent Run Loop 的一次会话 | 用户发起的智能问答 + 工具调用 |
| ToolCall | 单次工具调用 | 独立调用一个检测 API |

**统一追踪标识**：

```
trace_id     → 全局追踪（跨 Run 关联）
run_id       → 当前 Run 实例
node_id      → DAG 中的节点（WorkflowRun 专用）
tool_call_id → 单次工具调用
step_id      → Agent 决策步（AgentSession 专用）
```

**Run 状态机**：

```
pending → running → completed
                  → failed
                  → cancelled
         → paused → running（恢复）
```

**统一事件模型**：

所有 Run 形态共享统一的事件协议（RunEvent），关键事件类型：

| 事件 | 说明 |
|------|------|
| `run_started` / `run_finished` / `run_paused` / `run_resumed` / `run_cancelled` / `run_retried` | Run 生命周期 |
| `node_started` / `node_finished` / `node_failed` / `node_skipped` / `node_retry` / `sub_workflow_started` / `sub_workflow_finished` | 节点执行（WorkflowRun） |
| `tool_called` / `tool_succeeded` / `tool_failed` / `tool_timed_out` / `tool_retry_scheduled` | 工具调用 |
| `context_patch_applied` / `context_conflict` / `context_snapshot_created` | 上下文变更 |
| `agent_plan` / `agent_act` / `agent_observe` / `agent_escalation` / `agent_recover` / `agent_session_started` / `agent_session_finished` | Agent 决策链（AgentSession） |
| `policy_evaluated` / `policy_blocked` / `approval_requested` / `approval_resolved` | 策略与审批 |
| `asset_created` / `asset_derived` / `stream_slice_created` | 资产链路 |
| `budget_warning` / `budget_exceeded` | 预算治理 |
| `intent_received` / `intent_parsed` / `intent_planned` / `intent_plan_adjusted` / `intent_confirmed` / `intent_rejected` / `intent_execution_started` / `intent_execution_finished` / `intent_execution_failed` | 意图编排链路 |

### 3.5 Intent（意图任务）

**定义**：用户目标的统一执行入口，描述“用户想完成什么平台行为”。

Intent 是平台全 AI 交互的中心对象：用户的文本/语音/视频输入先被标准化为 Intent，再解析为动作计划（IntentPlan）或自动生成 Workflow(DAG)，最后进入统一执行容器（Run）。

**覆盖范围**（示例，不限于此）：
- 身份与权限：创建用户、创建角色、绑定角色、调整权限集
- 平台配置：修改 Settings（安全策略、集成配置、运行参数）
- 资产操作：上传资产、检索资产、引用历史资产参与执行
- 执行触发：启动工作流、发起 Agent 会话、调用工具

**执行语义**：
- 低风险动作可自动执行（auto_execute）
- 中高风险动作默认先确认/审批（confirm_then_execute）
- 全部动作进入审计与 RunEvent 体系，可追溯可回放

---

## 4. 核心数据流

### 4.1 主链路

```
User Input(text/voice/video) → Intent → IntentPlan/Workflow(DAG) → Run → Artifact(new Asset)
```

展开描述：

1. **输入接收**：用户通过文本/语音/视频提交目标，或直接上传/引用 Asset
2. **意图解析**：Intent Compiler 生成 IntentPlan（动作序列）或 Workflow(DAG)
3. **安全闸门**：Policy Engine 对每个动作做权限/风险/预算校验；高风险进入确认或审批
4. **能力绑定**：计划动作绑定到 Tool / Algorithm / Workflow / Identity / Settings 等执行器
5. **编排执行**：Workflow Engine 按 DAG 拓扑执行节点，Agent Runtime 按 Run Loop 驱动决策
6. **运行追踪与沉淀**：Run 持续产出 RunEvent，结果沉淀为新 Asset 或结构化结果

### 4.2 详细流转图

```
┌─────────────────────────────────────────────────────────────────┐
│                        输入层                                    │
│  [上传文件] [流媒体接入] [外部API] [用户输入] [事件触发]            │
└───────────────────────┬─────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Asset Store                                  │
│  video | image | audio | document | stream | structured | text   │
└───────────────────────┬─────────────────────────────────────────┘
                        │
                ┌───────┴────────┐
                ▼                ▼
┌──────────────────┐  ┌──────────────────┐
│  Workflow(DAG)   │  │  Agent Session   │
│  ┌──┐  ┌──┐     │  │  Plan → Act →    │
│  │N1├─►│N2│     │  │  Observe →       │
│  └──┘  └┬─┘     │  │  Recover/Finish  │
│         ▼       │  │                  │
│       ┌──┐      │  │                  │
│       │N3│      │  │                  │
│       └──┘      │  │                  │
└───────┬──────────┘  └────────┬─────────┘
        │                      │
        ▼                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Tool Registry                                │
│  [HTTP] [CLI] [Container] [MCP] [Model] [In-Process]            │
│                         │                                        │
│              Algorithm Library                                   │
│  [意图] → [版本] → [实现绑定] → Tool                              │
└───────────────────────┬─────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────────┐
│                     Run (执行容器)                                │
│  trace_id / run_id / node_id / tool_call_id                     │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │ RunEvents   │  │ ContextState│  │ Artifacts   │             │
│  │ (事件流)     │  │ (上下文)     │  │ (产物)      │             │
│  └─────────────┘  └─────────────┘  └──────┬──────┘             │
└────────────────────────────────────────────┼─────────────────────┘
                                             │
                                             ▼
                                    Asset Store (新 Asset)
```

### 4.3 上下文数据流

工作流执行过程中，上下文（Context）在节点间流转：

```
ContextSpec (定义态，属于 WorkflowRevision)
    │
    ▼ 初始化
ContextState (运行态，属于 Run)
    │
    ├── Node A 读取 → input_mapping 解析 → 执行 → output_mapping → Patch 提交 (CAS)
    │
    ├── Node B 读取 → ... → Patch 提交 (CAS)
    │
    ├── 共享键写入 → conflict_policy 校验 → Patch 提交 (CAS)
    │
    └── Snapshot (周期性/关键节点/手动)
```

---

## 5. 外部依赖

### 5.1 依赖总览

| 依赖 | 必须/可选 | 说明 |
|------|----------|------|
| **PostgreSQL** | 必须 | 元数据持久化、Event Store（LISTEN/NOTIFY）、上下文状态存储 |
| **S3/MinIO 或本地文件系统** | 必须 | Asset Store——所有 Asset 原始内容的持久化存储 |
| **Docker Engine** | 可选 | 仅当 Tool 需要容器级隔离（execution_mode=container）时 |
| **Redis** | 可选 | 分布式缓存与锁；单实例部署可用进程内 channel + sync.Map 替代 |
| **MediaMTX** | 保留 | 流媒体场景必须；提供 RTSP/RTMP/HLS/WebRTC 分发与录制能力 |
| **外部 AI API** | 可选 | 云模型调用（OpenAI/Anthropic/Ollama 等），通过 Model Runtime 适配 |
| **FFmpeg** | 可选 | 帧提取、转码等本地媒体处理；作为 subprocess 类型 Tool 调用 |

### 5.2 依赖策略

**最小化必须依赖原则**：

- 单二进制 + PostgreSQL + 文件存储即可完整运行
- 所有可选依赖通过配置开关控制，缺失时 graceful degrade
- Docker/Redis/外部 AI API 仅在需要对应能力时启用

**依赖替代方案**：

| 可选依赖 | 缺失时的替代 |
|---------|-------------|
| Redis | 进程内 channel + sync.Map（单实例模式） |
| Docker Engine | 仅使用 in_process/subprocess/remote 三种隔离等级 |
| MediaMTX | 流媒体相关功能不可用，其余功能正常 |
| 外部 AI API | 仅使用本地部署模型或非 AI 类 Tool |

---

## 6. 文档索引

`docs/` 目录下的完整设计文档集：

| 编号 | 文件名 | 标题 | 简述 |
|------|--------|------|------|
| 00 | `00-overview.md` | 总览 | 产品定位、核心概念、数据流与外部依赖（本文档） |
| 01 | `01-architecture.md` | 架构设计 | 6 层逻辑架构、Go 包结构、依赖规则、单二进制部署、执行隔离与跨层通信 |
| 02 | `02-domain-model.md` | 领域模型 | Asset/Tool/Algorithm/Intent/Run/Workflow/Context 完整数据结构与关系定义 |
| 03 | `03-asset-system.md` | 资产系统 | Asset 类型体系、Store 抽象、StreamAsset 与 MediaMTX 集成、派生链路 |
| 04 | `04-tool-system.md` | 工具系统 | ToolSpec 契约、Tool Registry、执行模式、Runtime 适配与版本管理 |
| 05 | `05-algorithm-library.md` | 算法库 | Algorithm/Version/Implementation 三层模型、选择策略与评测证据体系 |
| 06 | `06-workflow-engine.md` | 工作流引擎 | DAG 定义、Revision 版本化、ContextSpec/ContextState、CAS 并发控制与回放 |
| 07 | `07-agent-runtime.md` | Agent 运行时 | Run Loop 状态机、决策链路、错误分类与恢复策略、会话管理 |
| 08 | `08-observability.md` | 可观测性 | RunEvent 统一事件协议、Trace 体系、审计日志、运营指标与告警 |
| 09 | `09-security-policy.md` | 安全与策略 | RBAC 扩展、工具级权限、Policy Engine、数据访问限制与高风险审批 |
| 10 | `10-api-design.md` | API 设计 | REST API 完整端点定义、SSE 事件推送、MCP Server 协议、统一响应封装 |
| 11 | `11-frontend-design.md` | 前端设计 | Vue 3 + TypeScript 页面结构、Composables 模式、工作流编辑器与 Agent 会话界面 |

---

## 附录 A：术语表

| 术语 | 英文 | 定义 |
|------|------|------|
| 资产 | Asset | 系统中一切可追踪的输入/输出载体 |
| 工具 | Tool | 可执行能力的统一抽象 |
| 工具规格 | ToolSpec | Tool 的完整契约声明 |
| 算法 | Algorithm | 解决特定问题的意图抽象 |
| 意图任务 | Intent | 用户目标的统一入口与执行任务载体 |
| 意图计划 | IntentPlan | 意图解析后的结构化动作计划（可映射为 DAG） |
| 实现绑定 | ImplementationBinding | 算法版本到具体 Tool 的映射 |
| 运行 | Run | 执行活动的统一容器 |
| 运行事件 | RunEvent | 执行过程中产出的结构化事件 |
| 上下文规格 | ContextSpec | 工作流上下文的定义态 |
| 上下文状态 | ContextState | 工作流上下文的运行态 |
| 产物 | Artifact | 执行的输出结果（可转化为新 Asset） |
| 策略引擎 | Policy Engine | 执行前的权限/风险/预算校验引擎 |
| 工作流修订 | WorkflowRevision | 工作流定义的版本化快照 |

## 附录 B：缩写对照

| 缩写 | 全称 |
|------|------|
| DAG | Directed Acyclic Graph（有向无环图） |
| CAS | Compare-And-Swap（比较并交换） |
| CQRS | Command Query Responsibility Segregation |
| RBAC | Role-Based Access Control |
| MCP | Model Context Protocol |
| SSE | Server-Sent Events |
| DIP | Dependency Inversion Principle |
| SPA | Single Page Application |
