# Goyais 总览

> 本文档为 Goyais 的顶层设计概览，定义产品定位、核心概念、数据流与外部依赖，作为所有后续设计文档的共识基准。

最后更新：2026-02-09

---

## 1. 产品定位

### 1.1 一句话定义

**Goyais = 全意图驱动的智能 Agent + 多模态 AI 原生平台（Go 实现）**

### 1.2 核心描述

Goyais 是一个以 Go 语言构建的多模态 AI 编排与执行平台。它接收多种模态的输入（视频、图片、音频、文档、Excel、流媒体），通过统一的工具/模型/算法注册与 DAG 工作流编排进行处理，输出结构化结果、新资产或诊断报告。

平台默认以 AI 交互为主入口，且所有功能点均需支持 AI 触发：用户可通过对话式或语音式输入完成平台级操作（例如创建用户/角色、调整权限与设置、上传与处理资产、触发运行）。同时也支持页面操作，且与 AI 入口保持能力语义一致。

平台提供：

- **统一上下文管理**：工作流与 Agent 会话共享可版本化、可回放的运行上下文
- **任务全生命周期管理**：从触发、调度、执行到产物沉淀的完整链路
- **全 AI 操作入口**：自然语言/语音输入统一编译为可执行动作计划与 DAG
- **全功能 AI 映射契约**：身份、权限、设置、资产、编排、运行控制、系统管理均需具备 Intent Action 映射
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
6. **可最小化部署**：单二进制在默认配置下可通过 SQLite + memory cache + local storage 运行

### 1.4 核心设计哲学

秉承 **"业务 = 配置，能力 = 插件，执行 = 引擎"** 理念：

| 维度 | 设计方式 |
|------|----------|
| 配置 | IntentPlan + 工作流 + ContextSpec + ToolSpec + 算法绑定 |
| 插件 | 统一 Tool 抽象（合并所有执行模式） |
| 引擎 | Intent 编译器 + DAG 引擎 + Agent Runtime + 策略引擎 |

### 1.5 业务边界与核心作业目标

为确保“平台能力”可落为“业务结果”，定义四类业务角色与职责边界：

| 角色 | 核心职责 | 责任边界 |
|------|----------|----------|
| 平台管理员 | 维护租户、角色、权限、策略与集成配置 | 负责规则与治理，不直接承担业务结果审核 |
| 业务操作员 | 发起资产处理、工作流执行与日常运行操作 | 对输入质量与执行触发负责，不负责策略配置 |
| 审核员 | 处理高风险审批、结果复核与异常裁决 | 对业务结果有效性与放行负责 |
| 集成开发者 | 维护 Tool/Algorithm 集成与接口连通 | 对能力可用性与兼容性负责，不直接审批业务动作 |

首批 10 个核心作业目标（必须支持 AI/API/UI 并行触发）：

1. 创建用户并绑定角色（身份治理）
2. 调整角色权限并生效校验（权限治理）
3. 上传并注册多模态资产（资产入库）
4. 检索并引用历史资产（资产复用）
5. 启动指定工作流（执行触发）
6. 启动 Agent 会话并执行工具链（智能执行）
7. 暂停/恢复/取消运行（运行控制）
8. 失败运行重试并保留追踪链路（稳定性恢复）
9. 高风险动作发起审批并完成复核（安全放行）
10. 查询审计记录并关联产物（追溯闭环）

### 1.6 前后端技术基线与发布产物契约

前后端技术栈采用 **latest stable** 策略（仅跟踪最新稳定版，不在文档锁定具体 patch 号）：

- 前端：Vue + TypeScript + Vite + Tailwind CSS + pnpm（latest stable）
- 后端：Go 与核心后端依赖（latest stable）

发布产物硬约束：

- 最终可运行交付物必须是单一 Go 二进制
- 前端构建产物 `web/dist` 必须通过 `go:embed` 内嵌到该二进制
- 运行时不得依赖额外前端静态文件目录

前端体验硬约束：

- 响应式设计必须覆盖桌面/平板/移动端主流分辨率
- 必须支持深色/浅色模式切换
- 必须支持国际化（至少 `zh-CN` 与 `en`）

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

**Asset 元数据结构**（概念示意）：

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

> 下面为概念示意，字段与类型以当前仓库文档为准。

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

**全功能映射要求**：
- 身份管理（User/Role/Permission）必须有对应 Intent Action
- 平台设置（Security Policy / Integration / Runtime Settings）必须有对应 Intent Action
- 资产链路（上传、检索、派生、删除、归档）必须有对应 Intent Action
- 编排执行（Workflow/Run/Agent Session/ToolCall）必须有对应 Intent Action
- 运行控制（pause/resume/cancel/retry）必须有对应 Intent Action
- 系统管理（租户级配置、审计查询、策略管理）必须有对应 Intent Action
- 所有写操作必须纳入 Policy Engine 校验；中高风险动作必须进入确认/审批流程

### 3.6 业务闭环模板

统一业务闭环模板：

`触发 -> 意图解析 -> 执行 -> 复核 -> 产物确认 -> 归档`

每个业务闭环必须声明以下字段，缺一不可：

| 字段 | 说明 |
|------|------|
| 输入 | 触发来源、输入资产、参数约束、前置条件 |
| 输出 | 产物类型、结构化结果、可追溯标识（run_id/trace_id） |
| 责任人 | 发起人、审批人、复核人、归档责任人 |
| 完成条件 | 满足业务完成定义（DoD）且证据字段齐全 |
| 失败回退 | 失败后降级路径、人工接管路径、重试边界 |

业务闭环执行规则：

1. 任意写操作进入策略校验与风险分级
2. 命中高风险条件必须进入审批与人工复核
3. 归档阶段必须绑定运行事件链与最终产物

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

### 4.4 业务完成定义（DoD）

任务是否“业务完成”按以下维度联合判定：

| 维度 | 判定标准 |
|------|----------|
| 结果正确性 | 输出与预期语义一致，关键字段通过校验 |
| 时效 | 在场景约定时限内完成，超时需标记异常 |
| 可解释性 | 提供关键决策依据、失败原因或审批理由 |
| 可追溯性 | 可回放完整事件链并关联责任人 |
| 可复现性 | 相同输入和策略下可得到一致结果或可解释差异 |

任务状态标准：

| 状态 | 业务判定 |
|------|----------|
| 成功（Success） | 五项维度全部达标，产物可用且已归档 |
| 部分成功（Partial Success） | 产物可用但至少一项非核心维度未达标，需后续补偿 |
| 失败（Failed） | 核心维度（结果正确性/可追溯性）不达标，必须进入回退或人工接管 |

禁止将“仅技术执行成功”视为“业务成功”。

---

## 5. 外部依赖

### 5.1 依赖总览

| 依赖 | 默认角色 | 说明 |
|------|----------|------|
| **SQLite** | 默认必备 | 元数据持久化、Event Store 持久化、上下文状态存储（单机最小部署） |
| **本地文件存储（local）** | 默认必备 | Asset Store 默认实现，保存所有 Asset 原始内容 |
| **本地内存缓存（memory）** | 默认必备 | 缓存与进程内队列默认实现（单实例模式） |
| **PostgreSQL / MySQL** | 可选 | 可替换 SQLite 作为 SQL 后端，三者对外功能语义一致 |
| **MinIO / S3** | 可选 | 可替换 local 作为对象存储后端 |
| **Docker Engine** | 可选 | 仅当 Tool 需要容器级隔离（execution_mode=container）时 |
| **Redis** | 可选 | 分布式缓存、锁、队列增强实现（需显式配置启用） |
| **MediaMTX** | 可选 | 流媒体能力（RTSP/RTMP/HLS/WebRTC）接入与录制（需显式配置启用） |
| **外部 AI API** | 可选 | 云模型调用（OpenAI/Anthropic/Ollama 等），通过 Model Runtime 适配 |
| **FFmpeg** | 可选 | 帧提取、转码等本地媒体处理；作为 subprocess 类型 Tool 调用 |

### 5.2 依赖策略

**最小化必须依赖原则**：

- 默认最小化部署为：单二进制 + SQLite + local storage + memory cache
- PostgreSQL/MySQL、Redis、MinIO/S3、MediaMTX 均通过配置显式启用
- 事件通知统一走应用层 EventBus，不依赖特定数据库能力
- 显式启用的外部依赖若不可达，启动失败（fail-fast），不做静默回退
- Docker/外部 AI API 仅在对应能力被启用时生效
- 构建产物必须包含内嵌前端资源（`web/dist`）；未内嵌则构建产物不合格，不可发布

**依赖替代方案**：

| 组件 | 未显式启用时 | 显式启用但不可达时 |
|------|---------------|-----------------------|
| PostgreSQL / MySQL | 使用 SQLite 默认配置 | 启动失败（fail-fast） |
| Redis | 使用 memory cache + channel | 启动失败（fail-fast） |
| MinIO / S3 | 使用 local storage | 启动失败（fail-fast） |
| MediaMTX | 关闭流媒体相关能力 | 启动失败（fail-fast） |
| Docker Engine | 关闭 container execution_mode | 启动失败（若已显式启用 container runtime） |
| 外部 AI API | 仅使用本地模型或非 AI 类 Tool | 启动失败（若已显式绑定必需 provider） |

### 5.3 业务连续性与人工介入策略

必须人工复核的触发条件：

1. 高风险写操作（权限变更、策略变更、批量删除、跨租户敏感动作）
2. 低置信度或结果冲突（模型置信不足、规则冲突、上下文冲突）
3. 关键链路异常（外部依赖失败、连续重试失败、审批链卡滞）

复核超时策略：

| 场景 | 超时动作 |
|------|----------|
| 普通复核任务 | 升级到指定审核员或管理员 |
| 高风险放行任务 | 默认拒绝并回退到安全状态 |
| 运行中断决策 | 暂停运行并等待人工裁决 |

异常降级路径：

| 异常类型 | 降级策略 |
|----------|----------|
| 依赖不可达 | 切换本地可用路径或进入人工接管 |
| 工具执行失败 | 按重试策略执行，超限后转人工 |
| 审批超时 | 自动升级或按默认安全策略拒绝 |
| 上下文冲突 | 冻结冲突节点并触发人工合并 |

人工接管流程：

`发现异常 -> 冻结自动执行 -> 指派责任人 -> 人工裁决 -> 重新执行或终止 -> 归档`

### 5.4 构建时依赖与运行时依赖边界

| 类别 | 组件 | 规则 |
|------|------|------|
| 构建时依赖 | Node.js / pnpm / Vite / Tailwind CSS / TypeScript | 仅用于前端构建阶段，不作为运行时依赖 |
| 运行时依赖 | Go 二进制 + SQLite/local/memory（默认） | 目标机器可无 Node/pnpm 环境直接运行 |
| 发布门禁 | `web/dist` 内嵌校验 | 若二进制未包含前端资源，发布流程必须失败 |
| 运行门禁 | 前端入口可用性校验 | 服务启动后必须可提供 `index.html` 与静态资源入口 |

---

## 6. 文档索引

`docs/` 目录下的完整设计文档集：

| 编号 | 文件名 | 标题 | 简述 |
|------|--------|------|------|
| 00 | `00-overview.md` | 总览 | 产品定位、核心概念、数据流与外部依赖（本文档） |
| 01 | `01-architecture.md` | 架构设计 | 6 层逻辑架构、Go 包结构、依赖规则、单二进制部署、执行隔离与跨层通信 |

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
