# 06 - 工作流引擎设计

> 本文档定义 Goyais 工作流引擎的完整设计，包括 WorkflowDefinition 结构、节点类型、DAG 执行引擎、触发机制、版本管理、错误处理与核心接口。

最后更新：2026-02-09

---

## 1. 概述

### 1.1 定位

Workflow Engine 位于 **Control Layer**（`internal/control/workflow/`），是 Goyais 的核心编排引擎，负责：

- 解析 WorkflowDefinition 中定义的 DAG 结构
- 按拓扑顺序（支持并行）驱动节点执行
- 管理执行上下文（ContextState）的初始化、读写与快照
- 协调 Tool 执行、Algorithm 解析、Agent 会话与子 Workflow 嵌套
- 产出结构化 RunEvent 事件流，支持全链路追踪与回放

### 1.2 设计原则

| 原则 | 说明 |
|------|------|
| **声明式定义** | WorkflowDefinition 是纯数据结构，不含执行逻辑；引擎负责解释执行 |
| **上下文驱动** | 节点间数据传递统一通过 ContextState，不允许节点间直接引用 |
| **执行可观测** | 每个关键操作均产出 RunEvent，支持事后回放与审计 |
| **失败可恢复** | 支持节点级重试、部分重试（从指定节点恢复）、暂停/恢复 |
| **嵌套有界** | Workflow 可包含 Agent Node 与 Sub-Workflow Node，但嵌套深度最大 2 级 |

### 1.3 在六层架构中的位置

```
Access Layer (API/SSE/MCP)
    │
    ▼
Control Layer ← ★ Workflow Engine 在此层
    │
    ├── workflow/   ← 本文档范围
    ├── agent/      ← 见 07-agent-runtime.md
    └── scheduler/  ← 调度与触发
    │
    ▼
Registry Layer (Tool/Algorithm/WorkflowDef 注册管理)
    │
    ▼
Runtime Layer (Tool 执行器)
    │
    ▼
Data Layer (PostgreSQL/Asset Store/Event Store)
    │
    ▼
Observe Layer (RunEvent/Trace/Metrics)
```

---

## 2. WorkflowDefinition 结构

WorkflowDefinition 是工作流的完整声明式定义，存储于 Registry Layer 的 `internal/registry/workflowdef/` 中。

### 2.1 顶层结构

```go
type WorkflowDefinition struct {
    ID          uuid.UUID          `json:"id"`
    Code        string             `json:"code"`           // 全局唯一编码
    Name        string             `json:"name"`
    Description string             `json:"description"`
    Version     int                `json:"version"`        // 递增版本号
    Status      DefStatus          `json:"status"`         // draft|active|archived

    Inputs      map[string]InputDef   `json:"inputs"`      // 工作流输入参数定义
    ContextSpec ContextSpec           `json:"context_spec"` // 上下文规格定义
    Nodes       []NodeDefinition      `json:"nodes"`        // 节点列表
    Edges       []EdgeDefinition      `json:"edges"`        // 边列表
    Outputs     map[string]OutputDef  `json:"outputs"`      // 工作流输出映射
    Policy      WorkflowPolicy        `json:"policy"`       // 策略配置
    Triggers    []TriggerDef          `json:"triggers"`     // 触发器列表

    TenantID    uuid.UUID          `json:"tenant_id"`
    CreatedBy   uuid.UUID          `json:"created_by"`
    CreatedAt   time.Time          `json:"created_at"`
    UpdatedAt   time.Time          `json:"updated_at"`
}
```

### 2.2 输入定义（Inputs）

工作流的外部输入参数，使用 JSON Schema 描述类型与约束：

```go
type InputDef struct {
    Type        string      `json:"type"`         // JSON Schema 类型: string|number|boolean|object|array
    Description string      `json:"description"`
    Required    bool        `json:"required"`
    Default     interface{} `json:"default,omitempty"`
    Schema      JSONSchema  `json:"schema,omitempty"` // 完整 JSON Schema（用于复杂类型）
}
```

示例：

```json
{
  "inputs": {
    "video_asset_id": {
      "type": "string",
      "description": "待分析的视频资产 ID",
      "required": true,
      "schema": { "type": "string", "format": "uuid" }
    },
    "detection_threshold": {
      "type": "number",
      "description": "检测置信度阈值",
      "required": false,
      "default": 0.7,
      "schema": { "type": "number", "minimum": 0, "maximum": 1 }
    }
  }
}
```

### 2.3 上下文规格（ContextSpec）

定义工作流运行时上下文的结构与共享规则。详细设计参见领域模型文档，此处仅列出关键字段：

```go
type ContextSpec struct {
    Vars       map[string]VarDef       `json:"vars"`        // 全局变量定义
    SharedKeys map[string]SharedKeyDef `json:"shared_keys"` // 共享键定义（需 CAS）
}

type VarDef struct {
    Type     string      `json:"type"`
    Required bool        `json:"required"`
    Default  interface{} `json:"default,omitempty"`
    ReadOnly bool        `json:"readonly"`
}

type SharedKeyDef struct {
    Type           string `json:"type"`
    ConflictPolicy string `json:"conflict_policy"` // reject|last_write_wins|merge
    CAS            bool   `json:"cas"`             // 是否启用 Compare-And-Swap
}
```

**关键约束**：
- 节点默认只能写入 `nodes.<node_key>.*` 命名空间
- 写入全局共享区（`shared_keys`）的键必须在 ContextSpec 中显式声明
- 启用 CAS 的共享键写入时必须携带 `before_version`，版本不匹配则拒绝

### 2.4 节点定义（Nodes）

```go
type NodeDefinition struct {
    ID          string          `json:"id"`           // 节点唯一标识（DAG 内唯一）
    Name        string          `json:"name"`
    Type        NodeType        `json:"type"`         // tool|algorithm|agent|sub_workflow
    Config      NodeConfig      `json:"config"`       // 节点配置（按类型不同）
    RetryPolicy *RetryPolicy    `json:"retry_policy,omitempty"`
    Timeout     *Duration       `json:"timeout,omitempty"`
    Condition   string          `json:"condition,omitempty"` // 执行条件表达式
}

type NodeType string

const (
    NodeTypeTool        NodeType = "tool"
    NodeTypeAlgorithm   NodeType = "algorithm"
    NodeTypeAgent       NodeType = "agent"
    NodeTypeSubWorkflow NodeType = "sub_workflow"
)
```

### 2.5 边定义（Edges）

```go
type EdgeDefinition struct {
    ID        string `json:"id"`
    FromNode  string `json:"from_node"`  // 源节点 ID
    ToNode    string `json:"to_node"`    // 目标节点 ID
    Condition string `json:"condition,omitempty"` // 条件表达式（可选）
}
```

### 2.6 输出定义（Outputs）

工作流级输出将 Artifact 或 ContextState 中的值映射为工作流对外输出：

```go
type OutputDef struct {
    Description string `json:"description"`
    Mapping     string `json:"mapping"` // JSONPath 表达式，引用 context/artifacts
}
```

示例：

```json
{
  "outputs": {
    "analysis_report": {
      "description": "视频分析结果报告",
      "mapping": "$.context.nodes.summarize.result"
    },
    "detected_frames": {
      "description": "检测到的关键帧列表",
      "mapping": "$.context.artifacts.detect.frames"
    }
  }
}
```

### 2.7 策略配置（Policy）

```go
type WorkflowPolicy struct {
    ToolWhitelist    []string `json:"tool_whitelist,omitempty"`    // 允许使用的 Tool ID 列表（空=全部）
    ToolBlacklist    []string `json:"tool_blacklist,omitempty"`    // 禁止使用的 Tool ID 列表
    BudgetLimit      *Budget  `json:"budget_limit,omitempty"`     // 预算上限
    NetworkWhitelist []string `json:"network_whitelist,omitempty"`// 网络访问白名单
    DataScope        []string `json:"data_scope,omitempty"`       // 数据访问范围
    MaxDuration      Duration `json:"max_duration"`               // 最大执行时长
    MaxNestingLevel  int      `json:"max_nesting_level"`          // 最大嵌套级别（默认 2）
}

type Budget struct {
    MaxCostUSD   float64 `json:"max_cost_usd,omitempty"`
    MaxToolCalls int     `json:"max_tool_calls,omitempty"`
    MaxTokens    int64   `json:"max_tokens,omitempty"`
}
```

### 2.8 触发器定义（Triggers）

```go
type TriggerDef struct {
    Type     TriggerType            `json:"type"`   // manual|schedule|event
    Config   map[string]interface{} `json:"config"` // 按类型不同的配置
    Enabled  bool                   `json:"enabled"`
}

type TriggerType string

const (
    TriggerManual   TriggerType = "manual"
    TriggerSchedule TriggerType = "schedule"
    TriggerEvent    TriggerType = "event"
)
```

触发器配置示例：

```json
[
  {
    "type": "manual",
    "config": {},
    "enabled": true
  },
  {
    "type": "schedule",
    "config": {
      "cron": "0 */6 * * *",
      "timezone": "Asia/Shanghai"
    },
    "enabled": true
  },
  {
    "type": "event",
    "config": {
      "event_type": "new_asset",
      "filter": { "asset_type": "video", "tags": ["surveillance"] }
    },
    "enabled": true
  }
]
```

---

## 3. 节点类型详解

### 3.1 Tool Node（工具节点）

Tool Node 直接引用一个已注册的 Tool，是最基础的执行单元。

#### 配置结构

```go
type ToolNodeConfig struct {
    ToolID        string                 `json:"tool_id"`         // 直接引用 Tool 注册 ID
    InputMapping  map[string]string      `json:"input_mapping"`   // 输入映射规则
    OutputMapping map[string]string      `json:"output_mapping"`  // 输出映射规则
    Params        map[string]interface{} `json:"params,omitempty"`// 静态参数（直接传递）
}
```

#### 执行流程

```
1. 从 ContextState + Inputs + Artifacts 按 InputMapping 解析输入参数
2. 将静态 Params 与动态映射结果合并，构建完整 Tool 输入
3. 构建 ExecutionEnvelope:
   - trace_id, run_id, node_id, tool_call_id
   - 解析后的输入参数
   - 来自 WorkflowPolicy 的策略约束
4. 通过 Runtime Layer 获取对应 ToolExecutor
5. 调用 ToolExecutor.Execute(ctx, envelope) 获取 ToolResult
6. 按 OutputMapping 将 ToolResult 写回 ContextState
7. 产出 RunEvent: tool_called → tool_succeeded / tool_failed
```

#### InputMapping 示例

```json
{
  "input_mapping": {
    "image_url": "$.context.assets.input_image.uri",
    "threshold": "$.context.vars.detection_threshold",
    "prev_result": "$.context.nodes.node_a.output.detections",
    "raw_input": "$.inputs.user_query"
  }
}
```

映射表达式支持的引用来源：

| 前缀 | 含义 | 示例 |
|------|------|------|
| `$.context.vars.*` | ContextState 中的全局变量 | `$.context.vars.language` |
| `$.context.nodes.*` | 其他节点的输出结果 | `$.context.nodes.detect.output.boxes` |
| `$.context.assets.*` | 上下文中的资产引用 | `$.context.assets.input_video.uri` |
| `$.context.artifacts.*` | 节点产出的 Artifact | `$.context.artifacts.extract.frames` |
| `$.context.shared.*` | 共享键数据 | `$.context.shared.alerts` |
| `$.inputs.*` | 工作流输入参数 | `$.inputs.video_asset_id` |

#### OutputMapping 示例

```json
{
  "output_mapping": {
    "$.context.nodes.detect.result": "$.output",
    "$.context.vars.detection_count": "$.output.count",
    "$.context.artifacts.detect.main": "$.output.artifacts[0]"
  }
}
```

映射规则说明：
- 左侧为 ContextState 中的目标路径
- 右侧为 ToolResult 中的源路径
- `$.output` 代表 ToolResult 的完整输出
- `$.output.xxx` 可引用输出的具体字段

### 3.2 Algorithm Node（算法节点）

Algorithm Node 通过算法引用（algorithm_ref）间接关联 Tool，在运行时经过解析流程选择最终实现。

#### 配置结构

```go
type AlgorithmNodeConfig struct {
    AlgorithmRef  AlgorithmRef           `json:"algorithm_ref"`   // 算法引用
    InputMapping  map[string]string      `json:"input_mapping"`
    OutputMapping map[string]string      `json:"output_mapping"`
    Params        map[string]interface{} `json:"params,omitempty"`
    StrategyOverride *string             `json:"strategy_override,omitempty"` // 节点级策略覆盖
}

type AlgorithmRef struct {
    AlgorithmCode     string `json:"algorithm_code"`      // 算法编码
    VersionConstraint string `json:"version_constraint"`  // 版本约束: ">=1.0", "~2.0", "latest"
    Strategy          string `json:"strategy"`            // 默认选择策略: default|high_accuracy|low_cost|low_latency|balanced
}
```

#### 运行时解析流程

Algorithm Node 的核心特色是"引用意图而非实现"，运行时通过多步解析确定最终执行的 Tool：

```
步骤 1: Registry 查找
    │  根据 algorithm_ref.algorithm_code 查找 Algorithm 实体
    │  根据 algorithm_ref.version_constraint 匹配 AlgorithmVersion
    │  获取该版本下所有 ImplementationBinding 列表
    ▼
步骤 2: Registry 推荐
    │  Registry 根据 AlgorithmRef.strategy 对候选列表排序
    │  返回：排序后的候选 ImplementationBinding[] + 推荐默认项
    ▼
步骤 3: Control 覆盖检查
    │  检查 NodeConfig 是否设置了 strategy_override
    │  如果有覆盖 → 使用覆盖策略重新排序候选列表
    │  如果无覆盖 → 使用 Registry 推荐的默认项
    ▼
步骤 4: 选择最终实现
    │  取排序后的第一个 ImplementationBinding
    │  获取绑定的 ToolID
    ▼
步骤 5: 展开为 ToolCall
    │  以选中的 ToolID 构建 ExecutionEnvelope
    │  后续流程与 Tool Node 一致
    ▼
执行完毕 → OutputMapping 写回 ContextState
```

#### AlgorithmRef 使用示例

```json
{
  "algorithm_ref": {
    "algorithm_code": "face_detection",
    "version_constraint": ">=2.0",
    "strategy": "high_accuracy"
  }
}
```

此引用表示：使用"人脸检测"算法的 2.0 或更高版本，优先选择高精度实现。

#### 策略覆盖场景

工作流定义时指定默认策略，但在特定场景下可通过 `strategy_override` 在节点级覆盖：

```json
{
  "type": "algorithm",
  "config": {
    "algorithm_ref": {
      "algorithm_code": "object_detect",
      "version_constraint": "latest",
      "strategy": "default"
    },
    "strategy_override": "low_latency"
  }
}
```

### 3.3 Agent Node（Agent 节点）

Agent Node 在工作流中嵌入一个 Agent 会话，使 DAG 中的特定节点具备自主决策与多步执行能力。

#### 配置结构

```go
type AgentNodeConfig struct {
    AgentProfileID string            `json:"agent_profile_id"` // 引用 AgentProfile
    InputMapping   map[string]string `json:"input_mapping"`
    OutputMapping  map[string]string `json:"output_mapping"`
    Goal           string            `json:"goal,omitempty"`   // 节点级目标描述（注入 Agent 上下文）
    BudgetOverride *AgentBudget      `json:"budget_override,omitempty"` // 节点级预算覆盖
}
```

#### 执行流程

```
1. 加载 AgentProfile（从 Registry Layer）
2. 创建 AgentSession（Run.Type = agent）
   - parent_run_id = 当前 WorkflowRun 的 run_id
   - nesting_level = 当前 Run.NestingLevel + 1
3. 初始化 Agent 上下文：
   - 共享父 Workflow 的 ContextState（同一引用）
   - 通过 InputMapping 向 Agent 注入当前节点的上下文数据
   - 如果设置了 goal，注入到 Agent 的当前目标
4. 进入 Agent Run Loop（详见 07-agent-runtime.md）
   - Plan → Act → Observe → Reflect → Finish
   - Agent 可发起 ToolCall
   - Agent 可发起子 WorkflowRun（见嵌套限制）
5. Agent Finish 后：
   - 按 OutputMapping 将 Agent 的 output 写回父 Workflow 的 ContextState
   - 产出 RunEvent: agent_session_started → ... → agent_session_finished
```

#### 嵌套限制规则

为防止无限递归，系统对嵌套深度有严格限制：

```
级别 0: 顶层 WorkflowRun
    └── 级别 1: Agent Node → AgentSession
         └── 级别 2: Agent 发起的子 WorkflowRun
              └── ✗ 级别 3: 不允许 —— 子 Workflow 中不可再包含 Agent Node
```

**检查逻辑**：

```go
func validateNesting(currentRun *Run, nodeType NodeType) error {
    if nodeType == NodeTypeAgent || nodeType == NodeTypeSubWorkflow {
        if currentRun.NestingLevel + 1 > MaxNestingLevel { // MaxNestingLevel = 2
            return ErrNestingLevelExceeded
        }
    }
    // Agent 发起的子 Workflow 中不允许包含 Agent Node
    if currentRun.Type == RunTypeAgent && nodeType == NodeTypeAgent {
        return ErrAgentNodeInSubWorkflow
    }
    return nil
}
```

**更具体的嵌套规则**：

| 当前 Run 类型 | 当前 NestingLevel | 允许的子节点类型 |
|--------------|-------------------|-----------------|
| WorkflowRun | 0 | tool, algorithm, agent, sub_workflow |
| AgentSession | 1 | tool（通过 ToolCall）, sub_workflow（Agent 发起） |
| WorkflowRun（子） | 2 | tool, algorithm（不允许 agent 和 sub_workflow） |

#### 上下文共享机制

Agent Node 与父 Workflow 共享同一个 ContextState：

```
WorkflowRun.ContextState ←───── 共享引用 ─────→ AgentSession.ContextState
                                                        │
                          Agent 通过 OutputMapping ──────┘
                          写入 $.context.nodes.<agent_node>.result
```

- Agent 可读取 ContextState 中的任意数据（受 Policy 限制）
- Agent 写入遵循同样的 CAS 规则
- Agent 的每次 ContextPatch 都记录 `writer = agent:<session_id>`

### 3.4 Sub-Workflow Node（子工作流节点）

Sub-Workflow Node 引用另一个 WorkflowDefinition，以子 Run 的形式执行。

#### 配置结构

```go
type SubWorkflowNodeConfig struct {
    WorkflowID         uuid.UUID         `json:"workflow_id"`                    // 引用的子工作流 ID
    WorkflowCode       string            `json:"workflow_code"`                  // 或通过 code 引用
    InputMapping       map[string]string `json:"input_mapping"`
    OutputMapping      map[string]string `json:"output_mapping"`
    ContextMode        string            `json:"context_mode,omitempty"`         // shared|isolated|readonly（默认 isolated）
    ErrorPropagation   string            `json:"error_propagation,omitempty"`    // bubble_up|swallow
    CancelPropagation  string            `json:"cancel_propagation,omitempty"`   // parent_to_child|bidirectional|none
}
```

#### 执行流程

```
1. 从 Registry 加载子 WorkflowDefinition（按 ID 或 code，取最新 active 版本）
2. 创建子 WorkflowRun:
   - parent_run_id = 当前 Run 的 run_id
   - nesting_level = 当前 Run.NestingLevel + 1
3. 初始化子工作流上下文边界（按 `context_mode`）：
   - `isolated`（默认）：基于父 ContextState 创建分支快照；子流程内部独立写入
   - `shared`：直接共享父 ContextState（兼容模式，仅建议低冲突场景）
   - `readonly`：可读父上下文，仅允许写入子节点私有命名空间
4. 递归调用 WorkflowEngine.Execute(ctx, childRun, childDef)
5. 子 Workflow 执行完毕后:
   - `isolated`/`readonly`：仅通过 OutputMapping 回写父 ContextState（单次 CAS Patch）
   - `shared`：子流程写入已即时生效，父流程仅收集输出摘要
   - 产出 RunEvent: sub_workflow_started → ... → sub_workflow_finished
```

#### 嵌套级别递增

每创建一层子 Run，NestingLevel 加 1：

```go
childRun := &Run{
    ID:            uuid.New(),
    ParentRunID:   &currentRun.ID,
    NestingLevel:  currentRun.NestingLevel + 1,
    Type:          RunTypeWorkflow,
    // ...
}
```

#### 子工作流隔离语义（新增）

| 模式 | 读边界 | 写边界 | 推荐场景 |
|------|--------|--------|---------|
| `isolated`（默认） | 可读父上下文快照 | 子上下文分支，结束后显式回写 | 大多数生产场景 |
| `shared` | 与父完全共享 | 与父完全共享 | 低并发、低冲突的快速编排 |
| `readonly` | 可读父上下文 | 仅子节点私有命名空间 | 诊断、评估、试运行 |

取消与错误传播规则：

- 父 Run 取消时，默认向子 Run 传播（`cancel_propagation=parent_to_child`）。
- 子 Run 失败时，默认冒泡到父节点失败（`error_propagation=bubble_up`）；可配置 `swallow` 将子失败转为节点告警并继续。
- `bidirectional` 取消传播仅用于强一致链路：子 run 被人工取消时会回传取消信号给父 run。

---

## 4. DAG 执行引擎

### 4.1 执行主流程

```
┌──────────────────────────────────────────────────────────────────────┐
│ 1. 创建 Run                                                          │
│    └── 初始化 ContextState（从 ContextSpec 构建默认值）                  │
│                                                                      │
│ 2. 验证 DAG                                                          │
│    └── 环检测（DFS / 拓扑排序前置检查）                                  │
│    └── 嵌套级别检查（Agent/SubWorkflow 节点）                           │
│    └── 引用有效性检查（Tool/Algorithm/AgentProfile 是否存在）            │
│                                                                      │
│ 3. 拓扑排序（Kahn 算法）                                              │
│    └── 计算入度 → 分层 → 同层节点可并行                                 │
│                                                                      │
│ 4. 分层执行                                                          │
│    ┌─ 对于每一层（layer）:                                             │
│    │  ┌─ 对于该层中的每个节点（并行 goroutine pool）:                    │
│    │  │  a. 评估条件表达式（Condition）                                  │
│    │  │     └── false → 跳过该节点，标记 skipped                       │
│    │  │  b. 解析 InputMapping → 构建节点输入                            │
│    │  │  c. 构建 ExecutionEnvelope                                    │
│    │  │  d. 按节点类型分派执行:                                         │
│    │  │     ├── tool      → ToolExecutor.Execute()                   │
│    │  │     ├── algorithm → AlgorithmResolver → ToolExecutor          │
│    │  │     ├── agent     → AgentRuntime.RunSession()                │
│    │  │     └── sub_workflow → WorkflowEngine.Execute()（递归）        │
│    │  │  e. 应用 OutputMapping → 生成 ContextPatch                    │
│    │  │  f. 提交 Patch 到 ContextState（CAS 校验）                     │
│    │  │  g. 产出 RunEvent (node_started / node_finished)             │
│    │  └──                                                            │
│    └──                                                               │
│                                                                      │
│ 5. 所有节点完成                                                       │
│    └── 收集 Outputs（按 WorkflowDefinition.Outputs 映射）              │
│    └── Run 状态 → completed                                          │
│    └── 产出 RunEvent: run_finished                                    │
│                                                                      │
│ 6. 任意节点失败                                                       │
│    └── 检查 RetryPolicy → 重试 / 跳过 / 终止                          │
│    └── 超过重试上限 → Run 状态 → failed                                │
│    └── 产出 RunEvent: node_failed / run_finished(failed)              │
└──────────────────────────────────────────────────────────────────────┘
```

### 4.2 拓扑排序与并行执行

引擎使用 **Kahn 算法** 进行拓扑排序，天然产出分层结构，同层节点可并行执行：

```go
func topologicalSort(nodes []NodeDefinition, edges []EdgeDefinition) ([][]NodeDefinition, error) {
    // 1. 构建邻接表与入度表
    inDegree := make(map[string]int)
    adjacency := make(map[string][]string)
    for _, node := range nodes {
        inDegree[node.ID] = 0
    }
    for _, edge := range edges {
        adjacency[edge.FromNode] = append(adjacency[edge.FromNode], edge.ToNode)
        inDegree[edge.ToNode]++
    }

    // 2. 收集入度为 0 的节点作为第一层
    var layers [][]NodeDefinition
    queue := collectZeroInDegree(inDegree, nodes)

    // 3. 逐层处理
    for len(queue) > 0 {
        layers = append(layers, queue)
        var nextQueue []NodeDefinition
        for _, node := range queue {
            for _, neighbor := range adjacency[node.ID] {
                inDegree[neighbor]--
                if inDegree[neighbor] == 0 {
                    nextQueue = append(nextQueue, findNode(nodes, neighbor))
                }
            }
        }
        queue = nextQueue
    }

    // 4. 环检测: 如果处理的节点数 < 总节点数，则存在环
    processedCount := 0
    for _, layer := range layers {
        processedCount += len(layer)
    }
    if processedCount < len(nodes) {
        return nil, ErrCycleDetected
    }

    return layers, nil
}
```

并行执行使用限制大小的 goroutine pool，防止资源耗尽：

```go
func (e *Engine) executeLayer(ctx context.Context, run *Run, layer []NodeDefinition) error {
    g, ctx := errgroup.WithContext(ctx)
    g.SetLimit(e.config.MaxParallelNodes) // 默认 10

    for _, node := range layer {
        node := node // 闭包捕获
        g.Go(func() error {
            return e.executeNode(ctx, run, node)
        })
    }

    return g.Wait()
}
```

### 4.3 InputMapping 规则

InputMapping 使用类 JSONPath 表达式，从多个数据源中解析节点输入参数。

#### 表达式语法

```
$.context.vars.<key>                    → ContextState 全局变量
$.context.nodes.<node_id>.output.<path> → 其他节点的输出
$.context.nodes.<node_id>.result        → 其他节点的完整结果
$.context.assets.<key>.<field>          → 上下文资产引用
$.context.artifacts.<node_id>.<key>     → 节点产出的 Artifact
$.context.shared.<key>                  → 共享键
$.inputs.<key>                          → 工作流输入参数
```

#### 完整映射示例

```json
{
  "input_mapping": {
    "image_url": "$.context.assets.input_image.uri",
    "threshold": "$.context.vars.detection_threshold",
    "prev_boxes": "$.context.nodes.preprocess.output.regions",
    "user_prompt": "$.inputs.user_query",
    "model_name": "$.context.vars.preferred_model"
  }
}
```

#### 解析实现

```go
type MappingResolver struct {
    contextState *ContextState
    inputs       map[string]interface{}
    artifacts    map[string]interface{}
}

func (r *MappingResolver) Resolve(mapping map[string]string) (map[string]interface{}, error) {
    result := make(map[string]interface{})
    for targetKey, expr := range mapping {
        value, err := r.evaluateExpression(expr)
        if err != nil {
            return nil, fmt.Errorf("mapping %q → %q: %w", targetKey, expr, err)
        }
        result[targetKey] = value
    }
    return result, nil
}

func (r *MappingResolver) evaluateExpression(expr string) (interface{}, error) {
    switch {
    case strings.HasPrefix(expr, "$.context."):
        return r.resolveContextPath(strings.TrimPrefix(expr, "$.context."))
    case strings.HasPrefix(expr, "$.inputs."):
        return r.resolveInputPath(strings.TrimPrefix(expr, "$.inputs."))
    case strings.HasPrefix(expr, "$.artifacts."):
        return r.resolveArtifactPath(strings.TrimPrefix(expr, "$.artifacts."))
    default:
        return nil, fmt.Errorf("unsupported expression prefix: %s", expr)
    }
}
```

### 4.4 OutputMapping 规则

OutputMapping 将节点执行结果写入 ContextState。

#### 表达式语法

```
左侧（目标路径）:
  $.context.nodes.<current_node>.result  → 当前节点结果区
  $.context.nodes.<current_node>.<key>   → 当前节点自定义键
  $.context.vars.<key>                   → 全局变量（需在 ContextSpec 声明）
  $.context.shared.<key>                 → 共享键（CAS）
  $.context.artifacts.<current_node>.*   → 当前节点 Artifact 区

右侧（源路径）:
  $.output                               → ToolResult 的完整输出
  $.output.<path>                        → ToolResult 的指定字段
  $.output.artifacts[<index>]            → ToolResult 中的 Artifact
```

#### 示例

```json
{
  "output_mapping": {
    "$.context.nodes.detect.result": "$.output",
    "$.context.vars.detection_count": "$.output.summary.total_count",
    "$.context.artifacts.detect.main": "$.output.artifacts[0]",
    "$.context.shared.alerts": "$.output.alerts"
  }
}
```

#### 写入约束

| 目标路径类型 | 写入条件 | CAS 要求 |
|-------------|---------|----------|
| `$.context.nodes.<current_node>.*` | 默认允许，无需声明 | 否 |
| `$.context.vars.*` | 目标 key 必须在 ContextSpec.Vars 中声明且非 readonly | 否（除非配置） |
| `$.context.shared.*` | 目标 key 必须在 ContextSpec.SharedKeys 中声明 | 是（cas=true 时） |
| `$.context.artifacts.*` | 自动创建 Artifact 引用 | 否 |

### 4.5 条件分支

Edge 和 Node 均可配置条件表达式。引擎在执行前评估条件，决定是否执行该节点或是否沿某条边传播。

#### 表达式语法

采用受限表达式语法（兼容 JSONPath 引用），支持比较、逻辑、函数调用与 `NULL` 判定。

BNF（简化）：

```bnf
expr        ::= or_expr
or_expr     ::= and_expr ("||" and_expr)*
and_expr    ::= unary_expr ("&&" unary_expr)*
unary_expr  ::= "!" unary_expr | compare_expr
compare_expr::= primary (comp_op primary)?
primary     ::= literal | path_ref | func_call | "(" expr ")"
func_call   ::= ident "(" arg_list? ")"
arg_list    ::= expr ("," expr)*
comp_op     ::= "==" | "!=" | ">" | ">=" | "<" | "<=" | "in"
literal     ::= string | number | "true" | "false" | "null"
path_ref    ::= "$.context." path | "$.inputs." path
```

运算优先级（高到低）：

1. `!`
2. 比较运算（`== != > >= < <= in`）
3. `&&`
4. `||`

内置函数：

- `exists(path_ref) bool`
- `len(path_ref) int`
- `contains(path_ref, value) bool`

未定义路径与 NULL 处理：

- 未定义路径解析结果为 `null`（非报错）。
- 比较运算中，`null` 与数值/字符串比较返回 `false`。
- `== null` / `!= null` 允许直接判定存在性。
- 函数 `len(null)` 返回 `0`，`contains(null, x)` 返回 `false`。

#### 评估实现

```go
type ConditionEvaluator struct {
    resolver *MappingResolver
}

func (e *ConditionEvaluator) Evaluate(condition string) (bool, error) {
    if condition == "" {
        return true, nil // 无条件则默认通过
    }
    // 解析表达式 → AST → 求值
    ast, err := parseCondition(condition)
    if err != nil {
        return false, fmt.Errorf("invalid condition expression: %w", err)
    }
    return ast.Evaluate(e.resolver)
}
```

#### 条件分支在 Edge 上的应用

```json
{
  "edges": [
    {
      "from_node": "detect",
      "to_node": "alert_handler",
      "condition": "$.context.nodes.detect.result.count > 0"
    },
    {
      "from_node": "detect",
      "to_node": "summary_only",
      "condition": "$.context.nodes.detect.result.count == 0"
    }
  ]
}
```

当 detect 节点完成后，引擎根据条件决定后续激活哪些节点。如果所有出边条件均为 false，则下游节点不会被激活。

### 4.6 ContextPatch 与 CAS 机制

所有对 ContextState 的修改均通过 **JSON Patch（RFC 6902）** 格式提交，支持 CAS 并发控制。

#### Patch 结构

```go
type ContextPatch struct {
    ID            uuid.UUID        `json:"id"`
    RunID         uuid.UUID        `json:"run_id"`
    WriterNodeKey string           `json:"writer_node_key"` // 写入者节点标识
    BeforeVersion int64            `json:"before_version"`  // 期望的当前版本
    AfterVersion  int64            `json:"after_version"`   // 写入后的版本
    Operations    []PatchOperation `json:"operations"`      // RFC 6902 操作列表
    CreatedAt     time.Time        `json:"created_at"`
}

type PatchOperation struct {
    Op    string      `json:"op"`    // add|remove|replace|move|copy|test
    Path  string      `json:"path"`  // JSON Pointer (RFC 6901)
    Value interface{} `json:"value,omitempty"`
    From  string      `json:"from,omitempty"` // move/copy 操作使用
}
```

#### CAS 提交流程

```go
func ApplyPatchWithCAS(runID uuid.UUID, patch ContextPatch, maxRetries int) error {
    for attempt := 1; attempt <= maxRetries; attempt++ {
        // 1) 读取当前状态（state, version）
        state, version := loadContextState(runID)
        if version != patch.BeforeVersion {
            emitEvent("context_conflict", map[string]any{
                "expected_version": patch.BeforeVersion,
                "current_version":  version,
                "attempt":          attempt,
            })

            // 2) 冲突解析：仅 shared key 可走 overwrite/merge
            resolvedPatch, ok := resolveConflict(state, patch)
            if !ok {
                return ErrContextConflict
            }
            patch = resolvedPatch
            continue
        }

        // 3) 事务内原子应用（全成功或全失败）
        // 任一 op 失败必须 rollback，禁止“部分成功”落库。
        err := runTx(func(tx Tx) error {
            newState, err := applyJSONPatchAtomically(state, patch.Operations)
            if err != nil {
                return err
            }
            if err := updateContextState(tx, runID, version, newState); err != nil {
                return err
            }
            return appendContextPatch(tx, patch)
        })
        if err == nil {
            emitEvent("context_patch_applied", map[string]any{
                "before_version": patch.BeforeVersion,
                "after_version":  patch.AfterVersion,
                "attempt":        attempt,
            })
            return nil
        }
    }
    return ErrContextConflict
}
```

冲突策略补充：

- `reject`：直接失败，交由上层重试/恢复。
- `last_write_wins`：基于仲裁键 `(logical_ts, writer_priority, patch_id)` 进行确定性比较；同版本竞争时按该顺序决策。
- `merge`：
  - `array`：按元素追加并去重（保序）。
  - `object`：按 key 深度合并（冲突 key 递归走同策略）。
  - 标量（string/number/bool/null）：默认退化为 `reject`；若显式配置 `allow_scalar_lww=true`，退化为 `last_write_wins`。

多节点并发写同 key 场景：

1. NodeA/NodeB 同时读取 `version=10`。
2. NodeA 成功提交 `10 -> 11`。
3. NodeB 提交失败触发 `context_conflict`，重读为 `11`。
4. 根据 key 的冲突策略执行 `reject/merge/lww`，重建 patch 后重试。
5. 成功则提交 `11 -> 12`，失败则节点进入 Recover。

部分成功回滚策略：

- JSON Patch 操作必须在同一事务中执行，任一操作失败即全部回滚。
- 对外部副作用（如已发送通知）采用补偿事务（Saga）并产出 `compensation_scheduled` 事件。
- 回滚后不得递增 Context 版本号。

#### 回放机制

任意时间点的 ContextState 可通过快照 + 增量 Patch 还原：

```go
func replayContextState(runID uuid.UUID, targetVersion int64) (*ContextState, error) {
    // 1. 找到最近的快照（version <= targetVersion）
    snapshot := findLatestSnapshot(runID, targetVersion)

    // 2. 取快照之后到目标版本的所有 Patch
    patches := listPatches(runID, snapshot.Version, targetVersion)

    // 3. 按顺序应用 Patch
    state := snapshot.Data
    for _, patch := range patches {
        state = applyPatch(state, patch.Operations)
    }

    return &ContextState{Version: targetVersion, Data: state}, nil
}
```

---

## 5. 触发机制

### 5.1 手动触发（manual）

通过 API 调用直接创建 Run：

```
POST /api/workflows/{id}/runs
Content-Type: application/json

{
  "inputs": {
    "video_asset_id": "550e8400-e29b-41d4-a716-446655440000",
    "detection_threshold": 0.8
  }
}
```

处理流程：
1. Access Layer 接收请求
2. 验证 WorkflowDefinition 状态为 active
3. 验证输入参数（按 Inputs 定义的 JSON Schema）
4. 创建 Run 实例（type=workflow, nesting_level=0）
5. 初始化 ContextState
6. 调用 WorkflowEngine.Execute()

Intent 驱动触发补充：

- 当入口来自 `/intents/{id}/execute` 时，Intent Orchestrator 可在执行前动态生成/选择 WorkflowDefinition。
- 生成后的执行流程与手动触发一致，仍由 WorkflowEngine 统一调度与产出 RunEvent。

### 5.2 定时触发（schedule）

使用 gocron/v2 调度库，由 Scheduler 模块管理：

```go
type WorkflowScheduler struct {
    scheduler gocron.Scheduler
    engine    WorkflowEngine
    registry  WorkflowRepository
}

func (s *WorkflowScheduler) LoadSchedules(ctx context.Context) error {
    // 1. 查询所有 active 且包含 schedule trigger 的 WorkflowDefinition
    defs, err := s.registry.ListByTriggerType(ctx, TriggerSchedule)

    for _, def := range defs {
        for _, trigger := range def.Triggers {
            if trigger.Type == TriggerSchedule && trigger.Enabled {
                cronExpr := trigger.Config["cron"].(string)
                // 2. 注册 cron job
                s.scheduler.NewJob(
                    gocron.CronJob(cronExpr, false),
                    gocron.NewTask(s.triggerWorkflow, def.ID),
                )
            }
        }
    }
    return nil
}
```

### 5.3 事件触发（event）

通过内部事件总线监听系统事件，匹配后自动触发工作流：

```go
type EventTriggerHandler struct {
    eventBus EventBus
    registry WorkflowRepository
    engine   WorkflowEngine
}

func (h *EventTriggerHandler) Start(ctx context.Context) {
    // 监听所有触发类事件
    h.eventBus.Subscribe("asset.created", h.handleEvent)
    h.eventBus.Subscribe("asset.recording_complete", h.handleEvent)
    h.eventBus.Subscribe("custom.*", h.handleEvent)
}

func (h *EventTriggerHandler) handleEvent(ctx context.Context, event Event) {
    // 1. 查询所有匹配该事件类型的 WorkflowDefinition
    defs, _ := h.registry.ListByEventType(ctx, event.Type)

    for _, def := range defs {
        // 2. 检查事件过滤条件
        if !matchEventFilter(def.Trigger.Config["filter"], event.Payload) {
            continue
        }
        // 3. 从事件 payload 构建工作流输入
        inputs := buildInputsFromEvent(def, event)
        // 4. 创建并执行 Run
        h.triggerWorkflow(ctx, def.ID, inputs)
    }
}
```

支持的内置事件类型：

| 事件类型 | 说明 | Payload |
|---------|------|---------|
| `asset.created` | 新资产创建 | `{ asset_id, asset_type, tags }` |
| `asset.recording_complete` | 流媒体录制完成 | `{ asset_id, source_id, duration }` |
| `run.finished` | Run 执行完成 | `{ run_id, workflow_id, status }` |
| `custom.<name>` | 自定义事件 | 用户定义 |

---

## 6. Workflow 版本管理

### 6.1 版本化策略

WorkflowDefinition 采用递增版本号进行版本管理：

```go
type WorkflowDefinition struct {
    // ...
    Version int       `json:"version"` // 1, 2, 3, ...
    Status  DefStatus `json:"status"`  // draft|active|archived
}

type DefStatus string

const (
    DefStatusDraft    DefStatus = "draft"
    DefStatusActive   DefStatus = "active"
    DefStatusArchived DefStatus = "archived"
)
```

### 6.2 版本生命周期

```
draft → active → archived
  │        │
  │        └── 新版本创建时，旧 active 自动 archived
  │
  └── 直接激活或继续编辑
```

- 每次修改 WorkflowDefinition 内容时创建新版本（version + 1）
- 同一 workflow_code 下最多只有一个 active 版本
- 激活新版本时，旧 active 版本自动转为 archived

### 6.3 Run 与版本绑定

```go
type Run struct {
    // ...
    WorkflowDefID      uuid.UUID `json:"workflow_def_id"`
    WorkflowDefVersion int       `json:"workflow_def_version"` // 创建时绑定的版本号
}
```

**关键规则**：
- Run 创建时快照绑定当前 active 的 WorkflowDefinition 版本
- 修改 WorkflowDefinition 不影响已创建的 Run
- Run 的执行、回放均按绑定的版本解释
- 部分重试（RetryFromNode）也使用原始绑定版本

### 6.4 版本对比

支持对两个版本的 WorkflowDefinition 进行差异对比，用于审计与变更审查：

```go
type VersionDiff struct {
    FromVersion int                `json:"from_version"`
    ToVersion   int                `json:"to_version"`
    NodesAdded  []NodeDefinition   `json:"nodes_added"`
    NodesRemoved []NodeDefinition  `json:"nodes_removed"`
    NodesChanged []NodeChangeDiff  `json:"nodes_changed"`
    EdgesAdded  []EdgeDefinition   `json:"edges_added"`
    EdgesRemoved []EdgeDefinition  `json:"edges_removed"`
    PolicyChanged *PolicyDiff      `json:"policy_changed,omitempty"`
}
```

---

## 7. Workflow Registry

### 7.1 定位

Workflow Registry 位于 **Registry Layer**（`internal/registry/workflowdef/`），负责 WorkflowDefinition 的全生命周期管理。

### 7.2 核心职责

| 职责 | 说明 |
|------|------|
| CRUD | WorkflowDefinition 的创建、查询、更新、删除 |
| 版本管理 | 版本创建、激活、归档、历史查询 |
| 模板支持 | 预定义工作流模板的管理与实例化 |
| 校验 | DAG 环检测、Schema 校验、嵌套级别检查、引用有效性验证 |

### 7.3 校验规则

在保存或激活 WorkflowDefinition 时执行以下校验：

```go
type WorkflowValidator struct {
    toolRegistry      ToolRegistry
    algorithmRegistry AlgorithmRegistry
    agentRegistry     AgentProfileRegistry
    workflowRegistry  WorkflowRepository
}

func (v *WorkflowValidator) Validate(def *WorkflowDefinition) []ValidationError {
    var errs []ValidationError

    // 1. DAG 环检测
    if hasCycle(def.Nodes, def.Edges) {
        errs = append(errs, ValidationError{Code: "CYCLE_DETECTED", Message: "工作流定义包含环"})
    }

    // 2. 节点引用有效性
    for _, node := range def.Nodes {
        switch node.Type {
        case NodeTypeTool:
            if !v.toolRegistry.Exists(node.Config.ToolID) {
                errs = append(errs, ValidationError{
                    Code:    "TOOL_NOT_FOUND",
                    NodeID:  node.ID,
                    Message: fmt.Sprintf("工具 %s 不存在", node.Config.ToolID),
                })
            }
        case NodeTypeAlgorithm:
            if !v.algorithmRegistry.CanResolve(node.Config.AlgorithmRef) {
                errs = append(errs, ValidationError{
                    Code:    "ALGORITHM_UNRESOLVABLE",
                    NodeID:  node.ID,
                    Message: fmt.Sprintf("算法引用 %s 无法解析", node.Config.AlgorithmRef.AlgorithmCode),
                })
            }
        case NodeTypeAgent:
            if !v.agentRegistry.Exists(node.Config.AgentProfileID) {
                errs = append(errs, ValidationError{
                    Code:    "AGENT_PROFILE_NOT_FOUND",
                    NodeID:  node.ID,
                    Message: fmt.Sprintf("AgentProfile %s 不存在", node.Config.AgentProfileID),
                })
            }
        case NodeTypeSubWorkflow:
            if !v.workflowRegistry.Exists(node.Config.WorkflowID) {
                errs = append(errs, ValidationError{
                    Code:    "SUB_WORKFLOW_NOT_FOUND",
                    NodeID:  node.ID,
                    Message: fmt.Sprintf("子工作流 %s 不存在", node.Config.WorkflowID),
                })
            }
        }
    }

    // 3. 嵌套级别检查
    if def.Policy.MaxNestingLevel > 2 {
        errs = append(errs, ValidationError{
            Code:    "NESTING_LEVEL_EXCEEDED",
            Message: "最大嵌套级别不能超过 2",
        })
    }

    // 4. ContextSpec 校验
    errs = append(errs, v.validateContextSpec(def)...)

    // 5. Mapping 表达式校验
    errs = append(errs, v.validateMappings(def)...)

    // 6. Edge 引用校验（from_node/to_node 必须存在于 nodes 中）
    errs = append(errs, v.validateEdgeReferences(def)...)

    return errs
}
```

### 7.4 模板支持

```go
type WorkflowTemplate struct {
    ID          uuid.UUID `json:"id"`
    Code        string    `json:"code"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    Category    string    `json:"category"`    // 分类: media_analysis|data_processing|content_generation
    Definition  WorkflowDefinition `json:"definition"` // 模板定义
    Tags        []string  `json:"tags"`
    IsBuiltin   bool      `json:"is_builtin"` // 是否系统内置
    CreatedAt   time.Time `json:"created_at"`
}
```

从模板实例化工作流：

```go
func (r *WorkflowRegistry) CreateFromTemplate(ctx context.Context, templateID uuid.UUID, params TemplateParams) (*WorkflowDefinition, error) {
    template, err := r.templateRepo.GetByID(ctx, templateID)
    if err != nil {
        return nil, err
    }

    def := template.Definition.Clone()
    def.ID = uuid.New()
    def.Code = params.Code
    def.Name = params.Name
    def.Version = 1
    def.Status = DefStatusDraft

    // 应用模板参数覆盖
    applyTemplateParams(def, params.Overrides)

    return r.Create(ctx, def)
}
```

---

## 8. 错误处理与恢复

### 8.1 节点级重试（RetryPolicy）

每个节点可配置独立的重试策略：

```go
type RetryPolicy struct {
    MaxAttempts   int      `json:"max_attempts"`    // 最大尝试次数（含首次）
    BackoffType   string   `json:"backoff_type"`    // fixed|exponential|linear
    InitialDelay  Duration `json:"initial_delay"`   // 首次重试延迟
    MaxDelay      Duration `json:"max_delay"`       // 最大重试延迟
    RetryableErrors []string `json:"retryable_errors,omitempty"` // 可重试的错误类别
}
```

重试逻辑：

```go
func (e *Engine) executeNodeWithRetry(ctx context.Context, run *Run, node NodeDefinition) error {
    policy := node.RetryPolicy
    if policy == nil {
        policy = e.config.DefaultRetryPolicy
    }

    var lastErr error
    for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
        err := e.executeNode(ctx, run, node)
        if err == nil {
            return nil
        }
        lastErr = err

        // 检查是否可重试
        if !isRetryable(err, policy.RetryableErrors) {
            return err
        }

        // 最后一次尝试不等待
        if attempt < policy.MaxAttempts {
            delay := calculateBackoff(policy, attempt)
            e.emitEvent(run, RunEvent{
                Type:    "node_retry",
                NodeID:  node.ID,
                Payload: map[string]interface{}{"attempt": attempt, "delay": delay.String(), "error": err.Error()},
            })
            select {
            case <-ctx.Done():
                return ctx.Err()
            case <-time.After(delay):
            }
        }
    }

    return fmt.Errorf("node %s failed after %d attempts: %w", node.ID, policy.MaxAttempts, lastErr)
}
```

### 8.2 部分重试（Partial Retry）

支持从指定节点重新执行，适用于中间节点失败但前序结果仍有效的场景：

```
POST /api/runs/{id}/retry
Content-Type: application/json

{
  "from_node": "detect"
}
```

处理流程：
1. 验证 Run 状态为 failed 或 paused
2. 验证 from_node 存在于 DAG 中
3. 回滚 ContextState 到 from_node 执行前的版本（通过 Patch 回放）
4. 从 from_node 开始重新执行后续 DAG
5. Run 状态恢复为 running

```go
func (e *Engine) RetryFromNode(ctx context.Context, runID uuid.UUID, nodeID string) error {
    run, err := e.runRepo.GetByID(ctx, runID)
    if err != nil {
        return err
    }

    if run.Status != RunStatusFailed && run.Status != RunStatusPaused {
        return ErrInvalidRunStatus
    }

    def, err := e.registry.GetByID(ctx, run.WorkflowDefID)
    if err != nil {
        return err
    }

    // 回滚上下文到指定节点执行前
    err = e.rollbackContextToNode(ctx, run, nodeID)
    if err != nil {
        return err
    }

    // 重新进行拓扑排序，只执行 nodeID 及其下游
    subDAG := extractSubDAG(def, nodeID)

    run.Status = RunStatusRunning
    e.runRepo.Update(ctx, run)
    e.emitEvent(run, RunEvent{Type: "run_retried", Payload: map[string]interface{}{"from_node": nodeID}})

    return e.executeDAG(ctx, run, subDAG)
}
```

### 8.3 取消（Cancel）

通过 `context.Context` 取消机制，立即终止正在执行的节点：

```go
func (e *Engine) Cancel(ctx context.Context, runID uuid.UUID) error {
    run, err := e.runRepo.GetByID(ctx, runID)
    if err != nil {
        return err
    }

    // 触发 context 取消
    e.cancelFuncs[runID]()

    run.Status = RunStatusCancelled
    run.FinishedAt = timePtr(time.Now())
    e.runRepo.Update(ctx, run)
    e.emitEvent(run, RunEvent{Type: "run_cancelled"})

    return nil
}
```

### 8.4 暂停与恢复（Pause / Resume）

```go
func (e *Engine) Pause(ctx context.Context, runID uuid.UUID) error {
    run, err := e.runRepo.GetByID(ctx, runID)
    if err != nil {
        return err
    }

    if run.Status != RunStatusRunning {
        return ErrInvalidRunStatus
    }

    // 设置暂停标志，执行引擎在每个节点开始前检查
    e.pauseSignals[runID] = true

    run.Status = RunStatusPaused
    run.PausedAt = timePtr(time.Now())
    e.runRepo.Update(ctx, run)

    // 创建快照以便恢复
    e.createContextSnapshot(ctx, run, "pause")
    e.emitEvent(run, RunEvent{Type: "run_paused"})

    return nil
}

func (e *Engine) Resume(ctx context.Context, runID uuid.UUID) error {
    run, err := e.runRepo.GetByID(ctx, runID)
    if err != nil {
        return err
    }

    if run.Status != RunStatusPaused {
        return ErrInvalidRunStatus
    }

    delete(e.pauseSignals, runID)

    run.Status = RunStatusRunning
    run.PausedAt = nil
    e.runRepo.Update(ctx, run)
    e.emitEvent(run, RunEvent{Type: "run_resumed"})

    // 从暂停节点继续执行
    return e.resumeExecution(ctx, run)
}
```

暂停检查点：引擎在每个节点执行前检查暂停信号：

```go
func (e *Engine) executeNode(ctx context.Context, run *Run, node NodeDefinition) error {
    // 暂停检查
    if e.pauseSignals[run.ID] {
        return ErrRunPaused
    }
    // ... 正常执行逻辑
}
```

---

## 9. 核心接口定义

### 9.1 WorkflowEngine 接口

```go
// WorkflowEngine 定义工作流执行引擎的核心接口。
// 位于 domain 包中作为 Port，由 Control Layer 实现。
type WorkflowEngine interface {
    // Execute 执行一个完整的工作流 Run。
    // 引擎负责 DAG 解析、拓扑排序、节点调度与上下文管理。
    Execute(ctx context.Context, run *Run, def *WorkflowDefinition) error

    // Cancel 取消正在执行的 Run。
    // 通过 context 取消信号通知所有正在执行的节点。
    Cancel(ctx context.Context, runID uuid.UUID) error

    // Pause 暂停正在执行的 Run。
    // 当前正在执行的节点会完成后暂停，不会中断节点执行。
    Pause(ctx context.Context, runID uuid.UUID) error

    // Resume 恢复已暂停的 Run。
    // 从暂停时的节点继续执行。
    Resume(ctx context.Context, runID uuid.UUID) error

    // RetryFromNode 从指定节点开始重试。
    // 回滚上下文到该节点执行前的状态，重新执行该节点及其下游。
    RetryFromNode(ctx context.Context, runID uuid.UUID, nodeID string) error
}
```

### 9.2 WorkflowRepository 接口

```go
// WorkflowRepository 定义工作流定义的持久化接口。
// 位于 domain 包中，由 Data Layer 实现。
type WorkflowRepository interface {
    // Create 创建一个新的工作流定义。
    Create(ctx context.Context, def *WorkflowDefinition) error

    // GetByID 根据 ID 获取工作流定义。
    GetByID(ctx context.Context, id uuid.UUID) (*WorkflowDefinition, error)

    // GetByCode 根据编码获取最新 active 版本的工作流定义。
    GetByCode(ctx context.Context, code string) (*WorkflowDefinition, error)

    // GetByIDAndVersion 获取指定 ID 和版本号的工作流定义。
    GetByIDAndVersion(ctx context.Context, id uuid.UUID, version int) (*WorkflowDefinition, error)

    // List 按条件查询工作流定义列表。
    List(ctx context.Context, filter WorkflowFilter) ([]*WorkflowDefinition, int64, error)

    // ListByTriggerType 查询包含指定触发类型的工作流定义。
    ListByTriggerType(ctx context.Context, triggerType TriggerType) ([]*WorkflowDefinition, error)

    // ListByEventType 查询匹配指定事件类型的工作流定义。
    ListByEventType(ctx context.Context, eventType string) ([]*WorkflowDefinition, error)

    // Update 更新工作流定义（自动递增版本号）。
    Update(ctx context.Context, def *WorkflowDefinition) error

    // Delete 删除工作流定义（软删除，仅 draft 状态可删除）。
    Delete(ctx context.Context, id uuid.UUID) error

    // ListVersions 列出指定工作流的所有版本。
    ListVersions(ctx context.Context, workflowCode string) ([]*WorkflowDefinition, error)
}
```

### 9.3 RunRepository 接口

```go
// RunRepository 定义 Run 的持久化接口。
type RunRepository interface {
    Create(ctx context.Context, run *Run) error
    GetByID(ctx context.Context, id uuid.UUID) (*Run, error)
    Update(ctx context.Context, run *Run) error
    List(ctx context.Context, filter RunFilter) ([]*Run, int64, error)
    ListByWorkflowID(ctx context.Context, workflowID uuid.UUID) ([]*Run, error)
    ListByParentRunID(ctx context.Context, parentRunID uuid.UUID) ([]*Run, error)
}
```

### 9.4 ContextStateRepository 接口

```go
// ContextStateRepository 定义上下文状态的持久化接口。
type ContextStateRepository interface {
    // Initialize 初始化 Run 的 ContextState。
    Initialize(ctx context.Context, runID uuid.UUID, spec *ContextSpec, inputs map[string]interface{}) error

    // Get 获取当前 ContextState。
    Get(ctx context.Context, runID uuid.UUID) (*ContextState, error)

    // ApplyPatch 以 CAS 方式应用 Patch。
    // 如果 beforeVersion 与当前版本不匹配，返回 ErrVersionConflict。
    ApplyPatch(ctx context.Context, patch *ContextPatch) error

    // CreateSnapshot 创建 ContextState 快照。
    CreateSnapshot(ctx context.Context, runID uuid.UUID, trigger string) error

    // GetSnapshot 获取指定版本之前最近的快照。
    GetSnapshot(ctx context.Context, runID uuid.UUID, beforeVersion int64) (*ContextSnapshot, error)

    // ListPatches 列出指定版本范围内的 Patch。
    ListPatches(ctx context.Context, runID uuid.UUID, fromVersion, toVersion int64) ([]*ContextPatch, error)
}
```

### 9.5 EventEmitter 接口

```go
// EventEmitter 定义 RunEvent 的发射接口。
type EventEmitter interface {
    // Emit 发射一个 RunEvent。
    Emit(ctx context.Context, event RunEvent) error

    // EmitBatch 批量发射 RunEvent。
    EmitBatch(ctx context.Context, events []RunEvent) error
}
```

### 9.6 AlgorithmResolver 接口

```go
// AlgorithmResolver 定义算法引用解析接口，由 Registry Layer 实现。
type AlgorithmResolver interface {
    // Resolve 根据 AlgorithmRef 解析出最终的 ToolID。
    // 返回选中的 ImplementationBinding 和解析过程的元数据。
    Resolve(ctx context.Context, ref AlgorithmRef, strategyOverride *string) (*ResolvedAlgorithm, error)
}

type ResolvedAlgorithm struct {
    ToolID           string                `json:"tool_id"`
    AlgorithmCode    string                `json:"algorithm_code"`
    AlgorithmVersion string                `json:"algorithm_version"`
    ImplementationID uuid.UUID             `json:"implementation_id"`
    Strategy         string                `json:"strategy"`
    Metadata         map[string]interface{} `json:"metadata"` // 性能画像等附加信息
}
```

---

## 10. 数据模型

### 10.1 Run（运行实例）

```go
type Run struct {
    ID                 uuid.UUID  `json:"id"`
    WorkflowDefID      uuid.UUID  `json:"workflow_def_id"`
    WorkflowDefVersion int        `json:"workflow_def_version"`
    ParentRunID        *uuid.UUID `json:"parent_run_id,omitempty"`
    Type               RunType    `json:"type"`           // workflow|agent|tool_call
    NestingLevel       int        `json:"nesting_level"`  // 0=顶层
    Status             RunStatus  `json:"status"`         // pending|running|paused|completed|failed|cancelled

    Inputs             JSONMap    `json:"inputs"`
    Outputs            JSONMap    `json:"outputs,omitempty"`
    Error              *RunError `json:"error,omitempty"`

    TenantID           uuid.UUID  `json:"tenant_id"`
    CreatedBy          uuid.UUID  `json:"created_by"`
    CreatedAt          time.Time  `json:"created_at"`
    StartedAt          *time.Time `json:"started_at,omitempty"`
    PausedAt           *time.Time `json:"paused_at,omitempty"`
    FinishedAt         *time.Time `json:"finished_at,omitempty"`

    // 预算使用
    BudgetUsed         BudgetUsed `json:"budget_used"`
}

type RunType string
const (
    RunTypeWorkflow RunType = "workflow"
    RunTypeAgent    RunType = "agent"
    RunTypeToolCall RunType = "tool_call"
)

type RunStatus string
const (
    RunStatusPending   RunStatus = "pending"
    RunStatusRunning   RunStatus = "running"
    RunStatusPaused    RunStatus = "paused"
    RunStatusCompleted RunStatus = "completed"
    RunStatusFailed    RunStatus = "failed"
    RunStatusCancelled RunStatus = "cancelled"
)

type BudgetUsed struct {
    Steps     int     `json:"steps"`
    ToolCalls int     `json:"tool_calls"`
    CostUSD   float64 `json:"cost_usd"`
    Tokens    int64   `json:"tokens"`
    Duration  Duration `json:"duration"`
}
```

### 10.2 RunEvent（运行事件）

```go
type RunEvent struct {
    ID         uuid.UUID              `json:"id"`
    TraceID    string                 `json:"trace_id"`
    RunID      uuid.UUID              `json:"run_id"`
    Seq        int64                  `json:"seq"` // 单 Run 内单调递增序号
    NodeID     string                 `json:"node_id,omitempty"`
    ToolCallID string                 `json:"tool_call_id,omitempty"`
    StepID     string                 `json:"step_id,omitempty"`

    Type       string                 `json:"type"`      // 事件类型
    Source     string                 `json:"source"`    // 产出者标识
    Payload    map[string]interface{} `json:"payload"`
    Timestamp  time.Time              `json:"timestamp"`
}
```

Workflow Engine 直接产出的核心事件子集（全局完整枚举以 `02-domain-model.md` / `08-observability.md` 为准）：

| 事件类型 | 产出阶段 | 说明 |
|---------|---------|------|
| `run_started` | Run 开始 | 包含 inputs、workflow_def_version |
| `run_finished` | Run 结束 | 包含 status、outputs、duration |
| `run_paused` | Run 暂停 | 包含暂停原因 |
| `run_resumed` | Run 恢复 | - |
| `run_cancelled` | Run 取消 | 包含取消原因 |
| `run_retried` | 部分重试 | 包含 from_node |
| `node_started` | 节点开始执行 | 包含 node_id、inputs |
| `node_finished` | 节点执行完成 | 包含 node_id、outputs、duration |
| `node_failed` | 节点执行失败 | 包含 node_id、error |
| `node_skipped` | 节点被跳过 | 包含 node_id、condition 评估结果 |
| `node_retry` | 节点重试 | 包含 attempt、delay |
| `tool_called` | 工具调用开始 | 包含 tool_id、inputs |
| `tool_succeeded` | 工具调用成功 | 包含 tool_id、outputs、duration |
| `tool_failed` | 工具调用失败 | 包含 tool_id、error |
| `tool_timed_out` | 工具调用超时 | 包含 tool_id、timeout_ms |
| `tool_retry_scheduled` | 工具调用重试计划 | 包含 attempt、delay_ms |
| `context_patch_applied` | 上下文 Patch 应用 | 包含 patch 详情 |
| `context_conflict` | 上下文版本冲突 | 包含冲突键、处理策略 |
| `context_snapshot_created` | 上下文快照创建 | 包含 snapshot_version |
| `agent_session_started` | Agent 会话开始 | 包含 profile_id |
| `agent_session_finished` | Agent 会话结束 | 包含 output |
| `sub_workflow_started` | 子工作流开始 | 包含 child_run_id |
| `sub_workflow_finished` | 子工作流结束 | 包含 child_run_id、status |
| `policy_evaluated` | 策略评估 | 包含 decision、reason_code |
| `policy_blocked` | 策略拦截 | 包含 violations |
| `approval_requested` | 审批请求创建 | 包含 ticket_id |
| `approval_resolved` | 审批结果回写 | 包含 approval_status |
| `asset_created` | 新资产创建 | 包含 asset_id、asset_type |
| `asset_derived` | 派生资产创建 | 包含 parent_id、asset_id |
| `stream_slice_created` | 流切片创建 | 包含 slice_id、start_at、end_at |
| `budget_warning` | 预算告警 | 80% 阈值触发 |
| `budget_exceeded` | 预算超限 | 包含超限维度 |

### 10.3 ContextState（上下文状态）

```go
type ContextState struct {
    RunID   uuid.UUID              `json:"run_id"`
    Version int64                  `json:"version"`
    Data    map[string]interface{} `json:"data"`
    UpdatedAt time.Time            `json:"updated_at"`
}
```

ContextState.Data 的顶层结构：

```json
{
  "vars": {
    "detection_threshold": 0.7,
    "language": "zh"
  },
  "nodes": {
    "detect": {
      "result": { "count": 3, "boxes": [...] },
      "status": "completed"
    },
    "classify": {
      "result": null,
      "status": "pending"
    }
  },
  "assets": {
    "input_video": { "id": "...", "uri": "s3://...", "type": "video" }
  },
  "artifacts": {
    "detect": {
      "main": { "id": "...", "type": "structured", "uri": "..." }
    }
  },
  "shared": {
    "alerts": []
  }
}
```

### 10.4 Run 事件顺序保证（新增）

顺序与一致性约定：

- 单 `run_id` 内事件具备**因果顺序保证**，通过单调递增 `seq` 字段表达。
- `seq` 由事件写入层统一分配；同一 `seq` 不可重复。
- 跨 `run_id` 不保证全局顺序，仅保证 `trace_id + parent_run_id` 的部分有序关系。

乱序与去重处理：

- SSE 客户端按 `seq` 处理；若检测到缺口（如收到 `seq=45` 前缺少 `44`），需回退调用 `GET /runs/{id}/events?from_seq=44` 补拉。
- 客户端与服务端均以 `event.id` 做幂等去重。
- Event Store 查询按 `(timestamp, id)` 排序，回放时按 `seq` 为准。

---

## 11. 完整执行示例

以一个视频分析工作流为例，展示完整的定义与执行流程。

### 11.1 WorkflowDefinition

```json
{
  "code": "video_analysis_pipeline",
  "name": "视频分析流水线",
  "inputs": {
    "video_asset_id": { "type": "string", "required": true, "schema": {"format": "uuid"} },
    "detection_threshold": { "type": "number", "required": false, "default": 0.7 }
  },
  "context_spec": {
    "vars": {
      "detection_threshold": { "type": "number", "required": false, "default": 0.7, "readonly": false }
    },
    "shared_keys": {
      "alerts": { "type": "array", "conflict_policy": "merge", "cas": true }
    }
  },
  "nodes": [
    {
      "id": "load_asset",
      "name": "加载视频资产",
      "type": "tool",
      "config": {
        "tool_id": "asset_loader",
        "input_mapping": { "asset_id": "$.inputs.video_asset_id" },
        "output_mapping": {
          "$.context.assets.input_video": "$.output.asset_info",
          "$.context.nodes.load_asset.result": "$.output"
        }
      }
    },
    {
      "id": "extract_frames",
      "name": "提取关键帧",
      "type": "tool",
      "config": {
        "tool_id": "frame_extractor",
        "input_mapping": { "video_url": "$.context.assets.input_video.uri" },
        "output_mapping": { "$.context.nodes.extract_frames.result": "$.output" }
      },
      "retry_policy": { "max_attempts": 2, "backoff_type": "exponential", "initial_delay": "5s" }
    },
    {
      "id": "detect_objects",
      "name": "目标检测",
      "type": "algorithm",
      "config": {
        "algorithm_ref": {
          "algorithm_code": "object_detection",
          "version_constraint": ">=2.0",
          "strategy": "high_accuracy"
        },
        "input_mapping": {
          "frames": "$.context.nodes.extract_frames.result.frames",
          "threshold": "$.context.vars.detection_threshold"
        },
        "output_mapping": {
          "$.context.nodes.detect_objects.result": "$.output",
          "$.context.shared.alerts": "$.output.critical_alerts"
        }
      }
    },
    {
      "id": "analyze_results",
      "name": "智能分析",
      "type": "agent",
      "config": {
        "agent_profile_id": "video-analyst",
        "goal": "基于检测结果生成视频分析报告",
        "input_mapping": {
          "detections": "$.context.nodes.detect_objects.result",
          "video_info": "$.context.assets.input_video"
        },
        "output_mapping": {
          "$.context.nodes.analyze_results.result": "$.output"
        }
      }
    },
    {
      "id": "generate_report",
      "name": "生成报告",
      "type": "tool",
      "config": {
        "tool_id": "report_generator",
        "input_mapping": {
          "analysis": "$.context.nodes.analyze_results.result",
          "detections": "$.context.nodes.detect_objects.result"
        },
        "output_mapping": {
          "$.context.artifacts.generate_report.main": "$.output.report_asset"
        }
      },
      "condition": "$.context.nodes.detect_objects.result.count > 0"
    }
  ],
  "edges": [
    { "from_node": "load_asset", "to_node": "extract_frames" },
    { "from_node": "extract_frames", "to_node": "detect_objects" },
    { "from_node": "detect_objects", "to_node": "analyze_results" },
    { "from_node": "analyze_results", "to_node": "generate_report" }
  ],
  "outputs": {
    "report": { "description": "分析报告", "mapping": "$.context.artifacts.generate_report.main" },
    "alert_count": { "description": "告警数量", "mapping": "$.context.nodes.detect_objects.result.count" }
  },
  "policy": {
    "max_duration": "30m",
    "budget_limit": { "max_cost_usd": 10.0, "max_tool_calls": 100 },
    "max_nesting_level": 2
  },
  "triggers": [
    { "type": "manual", "config": {}, "enabled": true },
    {
      "type": "event",
      "config": { "event_type": "asset.created", "filter": { "asset_type": "video" } },
      "enabled": true
    }
  ]
}
```

### 11.2 执行流程跟踪

```
时间轴:
T0  → [run_started] Run 创建，ContextState 初始化 (version=1)
T1  → [node_started: load_asset] 开始加载资产
T2  → [tool_called: asset_loader] 调用 asset_loader 工具
T3  → [tool_succeeded: asset_loader] 返回资产信息
T4  → [context_patch_applied] version=1→2, 写入 assets.input_video
T5  → [node_finished: load_asset]

T6  → [node_started: extract_frames]
T7  → [tool_called: frame_extractor]
T8  → [tool_succeeded: frame_extractor] 返回 15 个关键帧
T9  → [context_patch_applied] version=2→3
T10 → [node_finished: extract_frames]

T11 → [node_started: detect_objects]
      → AlgorithmResolver: object_detection >=2.0, strategy=high_accuracy
      → 选择 ImplementationBinding: yolov8-gpu (accuracy=0.95)
T12 → [tool_called: yolov8-gpu-detector]
T13 → [tool_succeeded: yolov8-gpu-detector] 检测到 3 个目标
T14 → [context_patch_applied] version=3→4, 写入 nodes.detect_objects.result
T15 → [context_patch_applied] version=4→5, 写入 shared.alerts (CAS)
T16 → [node_finished: detect_objects]

T17 → [node_started: analyze_results]
T18 → [agent_session_started: video-analyst] AgentSession 创建 (nesting_level=1)
      → Agent Plan: "分析 3 个检测目标，生成分类与风险评估报告"
T19 → [agent_plan] Plan 创建
T20 → [tool_called: summarize] Agent 调用摘要工具
T21 → [tool_succeeded: summarize]
T22 → [agent_observe] 观察结果，更新记忆
T23 → [agent_reflect] 评估：目标达成，决策 finish
T24 → [agent_session_finished] Agent 输出结构化报告
T25 → [context_patch_applied] version=5→6, 写入 nodes.analyze_results.result
T26 → [node_finished: analyze_results]

T27 → 评估条件: $.context.nodes.detect_objects.result.count > 0 → true
T28 → [node_started: generate_report]
T29 → [tool_called: report_generator]
T30 → [tool_succeeded: report_generator]
T31 → [context_patch_applied] version=6→7, 写入 artifacts.generate_report.main
T32 → [node_finished: generate_report]

T33 → [run_finished: completed] 输出: { report: ..., alert_count: 3 }
```

---

## 12. 配置参数

引擎的运行时行为通过配置控制：

```yaml
workflow_engine:
  max_parallel_nodes: 10          # 同层最大并行节点数
  default_node_timeout: "5m"      # 默认节点超时
  default_retry_policy:
    max_attempts: 1               # 默认不重试
    backoff_type: "exponential"
    initial_delay: "2s"
    max_delay: "30s"
  context:
    max_patch_retries: 3          # CAS 冲突最大重试次数
    snapshot_interval: 10         # 每 N 个 Patch 后自动创建快照
    max_state_size_bytes: 10485760  # ContextState 最大 10MB
  budget:
    warning_threshold: 0.8        # 预算告警阈值（80%）
  nesting:
    max_level: 2                  # 最大嵌套级别
```

---

## 附录 A：术语对照

| 术语 | 英文 | 说明 |
|------|------|------|
| 工作流定义 | WorkflowDefinition | DAG 工作流的声明式描述 |
| 运行 | Run | 工作流定义的一次执行实例 |
| 上下文规格 | ContextSpec | 工作流上下文的定义态（属于 WorkflowDefinition） |
| 上下文状态 | ContextState | 工作流上下文的运行态（属于 Run） |
| 上下文补丁 | ContextPatch | 对 ContextState 的一次 JSON Patch 变更 |
| 输入映射 | InputMapping | 节点输入参数的解析规则 |
| 输出映射 | OutputMapping | 节点输出结果的写回规则 |
| 拓扑排序 | Topological Sort | DAG 节点执行顺序的确定算法 |
| 算法引用 | AlgorithmRef | 通过意图描述间接引用 Tool 的机制 |
| 嵌套级别 | NestingLevel | 当前 Run 在 Workflow/Agent 嵌套链中的深度 |
| 执行信封 | ExecutionEnvelope | 封装工具调用上下文的载体 |
