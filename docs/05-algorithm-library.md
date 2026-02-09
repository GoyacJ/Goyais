# 算法库设计

> 本文档定义 Goyais 算法库（Algorithm Library）的完整设计，涵盖领域模型、生命周期管理、策略选择机制、工作流集成、评测流程与数据库 Schema。

最后更新：2026-02-09

---

## 1. 概述

算法库位于 Registry Layer（`internal/registry/algorithm/`），是 Goyais 的**算法资产管理中心**。

在 Goyais 中，Algorithm 是独立的一等对象。它表示**"做什么"**——即场景定义、问题描述与能力边界，与 Tool（表示"怎么做"——即执行能力）严格分离。这种分离使得同一个业务意图（如"目标检测"）可以拥有多种实现（不同模型、不同服务商、不同精度/成本配比），每种实现通过 ImplementationBinding 绑定到具体的 Tool。

**核心职责**：

- 管理算法意图（Algorithm）的定义与分类
- 管理算法版本（AlgorithmVersion）的 schema 契约与生命周期
- 维护实现绑定（ImplementationBinding）关系，将算法版本映射到具体 Tool
- 提供策略选择（AlgorithmResolver）机制，为 Workflow 节点和 Agent 决策提供最优实现推荐
- 管理评测档案（EvaluationProfile），通过评测 Workflow 产出评测证据

**在整体架构中的位置**：

```
Access Layer (API/MCP)
    │
    ▼
Control Layer (Workflow Engine / Agent Runtime)
    │
    ├── 引用 algorithm_ref
    │
    ▼
Registry Layer ← 本文档范围
    ├── Algorithm Library ← 算法意图/版本/绑定/评测
    ├── Tool Registry     ← 工具注册与发现（见 04-tool-system.md）
    └── Model Registry    ← 模型配置与路由（见 04-tool-system.md）
    │
    ▼
Runtime Layer (Tool Executors)
    │
    ▼
Data Layer (Asset Store / DB)
```

---

## 2. 领域模型

### 2.1 Algorithm（算法）

Algorithm 是算法库的顶层实体，代表一个**场景/问题定义/能力边界**的抽象。它不包含任何实现细节，只描述"这个算法解决什么问题"。

**核心定位**：

- "目标检测"是一个 Algorithm——描述"在图像或视频中定位并标注物体"的意图
- "语音识别"是一个 Algorithm——描述"将音频转换为文本"的意图
- "文档摘要"是一个 Algorithm——描述"提取文档核心信息并生成摘要"的意图

**数据结构**：

```go
type Algorithm struct {
    ID          uuid.UUID          `json:"id"`
    Name        string             `json:"name"`          // 人类可读名称，如"目标检测"
    Code        string             `json:"code"`          // 全局唯一编码，如"object_detection"
    Description string             `json:"description"`   // 详细描述
    Scene       string             `json:"scene"`         // 应用场景描述
    Problem     string             `json:"problem"`       // 解决的核心问题
    Boundary    string             `json:"boundary"`      // 能力边界与限制说明
    Category    AlgorithmCategory  `json:"category"`      // 算法分类
    Tags        []string           `json:"tags"`          // 标签（用于检索与分组）
    TenantID    uuid.UUID          `json:"tenant_id"`     // 租户隔离
    CreatedBy   uuid.UUID          `json:"created_by"`    // 创建者
    CreatedAt   time.Time          `json:"created_at"`
    UpdatedAt   time.Time          `json:"updated_at"`
}
```

**category 枚举**：

```go
type AlgorithmCategory string

const (
    CategoryDetection      AlgorithmCategory = "detection"       // 检测类（目标检测、异常检测、变化检测）
    CategoryClassification AlgorithmCategory = "classification"  // 分类类（图像分类、文本分类、情感分析）
    CategorySegmentation   AlgorithmCategory = "segmentation"    // 分割类（语义分割、实例分割、全景分割）
    CategoryGeneration     AlgorithmCategory = "generation"      // 生成类（文本生成、图像生成、视频生成）
    CategoryTransform      AlgorithmCategory = "transform"       // 转换类（格式转换、风格迁移、超分辨率）
    CategoryAnalysis       AlgorithmCategory = "analysis"        // 分析类（统计分析、趋势分析、关联分析）
    CategoryRecognition    AlgorithmCategory = "recognition"     // 识别类（人脸识别、语音识别、OCR）
    CategoryExtraction     AlgorithmCategory = "extraction"      // 提取类（关键信息提取、特征提取、实体抽取）
)
```

**设计要点**：

- `code` 为全局唯一标识，用于工作流节点引用，一旦发布后不可修改
- `scene`、`problem`、`boundary` 三个字段共同构成算法的完整语义描述，供 Agent 理解和选择
- `tags` 支持灵活的多维分类，弥补 `category` 单维分类的不足

### 2.2 AlgorithmVersion（算法版本）

AlgorithmVersion 是 Algorithm 的版本化实例，承载**契约定义**（输入/输出 schema）和**运行时默认配置**。每个版本是一个独立的、可发布的算法快照。

**数据结构**：

```go
type AlgorithmVersion struct {
    ID              uuid.UUID              `json:"id"`
    AlgorithmID     uuid.UUID              `json:"algorithm_id"`     // 所属算法
    Version         string                 `json:"version"`          // 语义化版本号（semver），如 "1.0.0"、"2.1.0-beta"
    InputSchema     map[string]interface{} `json:"input_schema"`     // JSON Schema，定义输入契约
    OutputSchema    map[string]interface{} `json:"output_schema"`    // JSON Schema，定义输出契约
    DefaultParams   map[string]interface{} `json:"default_params"`   // 默认参数（可被 Binding 和 NodeConfig 覆盖）
    ResourceProfile ResourceProfile        `json:"resource_profile"` // 资源需求估算
    Status          AlgorithmLifecycle     `json:"status"`           // 生命周期状态
    Changelog       string                 `json:"changelog"`        // 版本变更说明
    CreatedAt       time.Time              `json:"created_at"`
    UpdatedAt       time.Time              `json:"updated_at"`
}

type ResourceProfile struct {
    CPU       string `json:"cpu"`        // CPU 需求估算，如 "500m"、"2"
    Memory    string `json:"memory"`     // 内存需求估算，如 "512Mi"、"4Gi"
    GPU       string `json:"gpu"`        // GPU 需求，如 "1"、"0"（不需要）
    GPUMemory string `json:"gpu_memory"` // 显存需求估算，如 "4Gi"
}
```

**版本号规范**：

- 遵循 [Semantic Versioning 2.0.0](https://semver.org/) 规范
- 主版本号（MAJOR）：不兼容的 schema 变更（input_schema 或 output_schema 的 breaking change）
- 次版本号（MINOR）：向后兼容的功能新增（新增可选输入字段、新增输出字段）
- 修订号（PATCH）：向后兼容的缺陷修复（default_params 调整、文档修正）
- 预发布标签：`-alpha`、`-beta`、`-rc.1` 等，用于测试阶段

**InputSchema / OutputSchema 示例**（以目标检测为例）：

```json
// InputSchema
{
  "type": "object",
  "required": ["image_ref"],
  "properties": {
    "image_ref": {
      "type": "string",
      "description": "输入图像的 Asset Store 引用路径"
    },
    "confidence_threshold": {
      "type": "number",
      "minimum": 0,
      "maximum": 1,
      "default": 0.5,
      "description": "检测置信度阈值"
    },
    "target_classes": {
      "type": "array",
      "items": { "type": "string" },
      "description": "目标类别过滤列表，为空则检测所有类别"
    }
  }
}

// OutputSchema
{
  "type": "object",
  "required": ["detections"],
  "properties": {
    "detections": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["class", "confidence", "bbox"],
        "properties": {
          "class": { "type": "string" },
          "confidence": { "type": "number" },
          "bbox": {
            "type": "object",
            "properties": {
              "x": { "type": "number" },
              "y": { "type": "number" },
              "width": { "type": "number" },
              "height": { "type": "number" }
            }
          }
        }
      }
    },
    "processing_time_ms": { "type": "integer" }
  }
}
```

### 2.3 ImplementationBinding（实现绑定）

ImplementationBinding 将 AlgorithmVersion 绑定到具体的 Tool。一个 AlgorithmVersion 可以拥有多个 Binding，每个 Binding 对应不同的实现策略（默认/高精度/低成本/低延迟/均衡）。

**数据结构**：

```go
type ImplementationBinding struct {
    ID                 uuid.UUID              `json:"id"`
    AlgorithmVersionID uuid.UUID              `json:"algorithm_version_id"` // 所属算法版本
    ToolID             string                 `json:"tool_id"`              // 绑定的 Tool ID（引用 Tool Registry）
    Priority           int                    `json:"priority"`             // 优先级（同策略下的排序依据，数值越小优先级越高）
    Strategy           BindingStrategy        `json:"strategy"`             // 实现策略
    ParamOverrides     map[string]interface{} `json:"param_overrides"`      // 参数覆盖（覆盖 Tool 的默认参数）
    PerformanceProfile PerformanceProfile     `json:"performance_profile"`  // 性能画像
    Status             BindingStatus          `json:"status"`               // 绑定状态
    CreatedAt          time.Time              `json:"created_at"`
    UpdatedAt          time.Time              `json:"updated_at"`
}

type BindingStrategy string

const (
    StrategyDefault      BindingStrategy = "default"       // 默认策略
    StrategyHighAccuracy BindingStrategy = "high_accuracy" // 高精度优先
    StrategyLowCost      BindingStrategy = "low_cost"      // 低成本优先
    StrategyLowLatency   BindingStrategy = "low_latency"   // 低延迟优先
    StrategyBalanced     BindingStrategy = "balanced"      // 均衡策略
)

type PerformanceProfile struct {
    LatencyP50MS  int     `json:"latency_p50_ms"`   // P50 延迟（毫秒）
    LatencyP99MS  int     `json:"latency_p99_ms"`   // P99 延迟（毫秒）
    CostPerCall   float64 `json:"cost_per_call"`     // 每次调用成本（USD）
    AccuracyScore float64 `json:"accuracy_score"`    // 精度评分（0-1）
    Throughput    int     `json:"throughput"`         // 每秒处理量（QPS）
}

type BindingStatus string

const (
    BindingActive   BindingStatus = "active"   // 活跃可用
    BindingDisabled BindingStatus = "disabled"  // 已禁用（手动或因故障自动禁用）
)
```

**设计要点**：

- `tool_id` 引用 Tool Registry 中的工具，不直接持有 Tool 定义
- `priority` 在同一 strategy 下决定选择顺序，数值越小优先级越高
- `param_overrides` 用于针对特定 Tool 覆盖参数（如某个 Tool 需要特定的 `model_name` 参数）
- `performance_profile` 存储实际或预估的性能指标，用于策略选择决策
- 已发布（published）的 AlgorithmVersion 的 Binding 不可删除，只能 disable

### 2.4 EvaluationProfile（评测档案）

评测档案记录算法版本的评测结果。**评测本身作为一个特殊的 Workflow Run 执行**，评测产出的 Artifacts 链接到 EvaluationProfile。

**数据结构**：

```go
type EvaluationProfile struct {
    ID                 uuid.UUID              `json:"id"`
    AlgorithmVersionID uuid.UUID              `json:"algorithm_version_id"` // 评测对象
    RunID              uuid.UUID              `json:"run_id"`               // 评测 Workflow 的 Run ID
    Metrics            EvaluationMetrics      `json:"metrics"`              // 评测指标
    InputDatasetRef    string                 `json:"input_dataset_ref"`    // 输入数据集的 Asset Store 引用
    DatasetInfo        DatasetInfo            `json:"dataset_info"`         // 数据集摘要信息
    Status             EvaluationStatus       `json:"status"`               // 评测状态
    Summary            string                 `json:"summary"`              // 评测总结（人工或自动生成）
    CompletedAt        *time.Time             `json:"completed_at"`         // 评测完成时间
    CreatedAt          time.Time              `json:"created_at"`
    UpdatedAt          time.Time              `json:"updated_at"`
}

type EvaluationMetrics struct {
    Accuracy     *float64 `json:"accuracy,omitempty"`      // 准确率
    Precision    *float64 `json:"precision,omitempty"`      // 精确率
    Recall       *float64 `json:"recall,omitempty"`         // 召回率
    F1           *float64 `json:"f1,omitempty"`             // F1 分数
    LatencyP50MS *int     `json:"latency_p50_ms,omitempty"` // P50 延迟（毫秒）
    LatencyP99MS *int     `json:"latency_p99_ms,omitempty"` // P99 延迟（毫秒）
    CostPerCall  *float64 `json:"cost_per_call,omitempty"`  // 每次调用成本（USD）
    CustomMetrics map[string]interface{} `json:"custom_metrics,omitempty"` // 自定义指标
}

type DatasetInfo struct {
    Name        string `json:"name"`         // 数据集名称
    SampleCount int    `json:"sample_count"` // 样本数量
    Description string `json:"description"`  // 数据集描述
}

type EvaluationStatus string

const (
    EvalPending   EvaluationStatus = "pending"   // 等待执行
    EvalRunning   EvaluationStatus = "running"   // 评测执行中
    EvalCompleted EvaluationStatus = "completed"  // 评测完成
    EvalFailed    EvaluationStatus = "failed"     // 评测失败
)
```

**评测与 Run 的关系**：

- 评测创建时同步创建一个 Run（WorkflowRun），`run_id` 指向该 Run
- 评测 Workflow 的执行过程遵循标准的 Run 生命周期
- 评测产出的 Artifacts 通过 Run 关联，可溯源每个样本的处理结果
- 评测指标（metrics）在 Workflow 完成后由汇总节点计算并回填

---

## 3. 生命周期管理

### 3.1 AlgorithmVersion 生命周期状态机

```go
type AlgorithmLifecycle string

const (
    LifecycleDraft      AlgorithmLifecycle = "draft"      // 草稿
    LifecycleTested     AlgorithmLifecycle = "tested"     // 已测试
    LifecyclePublished  AlgorithmLifecycle = "published"  // 已发布
    LifecycleDeprecated AlgorithmLifecycle = "deprecated" // 已弃用
)
```

**状态转换规则**：

```
draft → tested → published → deprecated
  │        │         │            │
  │        │         │            └── 不可用于新 Workflow 创建
  │        │         │                已绑定的 Workflow 继续正常运行
  │        │         │                不可回退到 published
  │        │         │
  │        │         └── 冻结状态（CRITICAL）
  │        │              input_schema 不可修改
  │        │              output_schema 不可修改
  │        │              default_params 不可修改
  │        │              已有 ImplementationBinding 不可删除（只能 disable 或新增）
  │        │              如需变更 → 必须创建新版本
  │        │
  │        └── 至少有一个 status=completed 的 EvaluationProfile
  │             至少有一个 status=active 的 ImplementationBinding
  │             可回退到 draft（如评测不达标需要修改）
  │
  └── 自由修改一切字段
       可添加/删除/修改 ImplementationBinding
       可创建和执行 EvaluationProfile
```

**转换前置条件汇总**：

| 转换 | 前置条件 |
|------|---------|
| draft → tested | 至少有 1 个 completed 的 EvaluationProfile；至少有 1 个 active 的 ImplementationBinding |
| tested → published | 由管理员确认发布；可配置自动发布规则（如指标达标自动发布） |
| tested → draft | 无限制（允许回退修改） |
| published → deprecated | 无限制（但会触发告警通知引用该版本的 Workflow 负责人） |
| 其他任意转换 | 不允许 |

### 3.2 发布冻结规则（CRITICAL）

published 状态是算法版本生命周期中的关键节点，一旦进入 published 状态，以下字段/操作被冻结：

**AlgorithmVersion 冻结字段**：

| 字段 | 冻结行为 | 原因 |
|------|---------|------|
| `input_schema` | 不可修改 | 下游 Workflow 节点依赖输入契约 |
| `output_schema` | 不可修改 | 下游节点依赖输出格式 |
| `default_params` | 不可修改 | 已有 Workflow 可能依赖默认值 |
| `version` | 不可修改 | 版本号是引用标识 |

**ImplementationBinding 冻结操作**：

| 操作 | 冻结行为 | 替代方案 |
|------|---------|---------|
| 删除已有 Binding | 禁止 | 将 Binding status 设为 disabled |
| 新增 Binding | 允许 | 可随时添加新的实现 |
| 修改 Binding 的 param_overrides | 允许 | 不影响已有行为，只影响后续调用 |
| 修改 Binding 的 priority | 允许 | 调整选择顺序 |
| 修改 Binding 的 status | 允许 | 可在 active/disabled 间切换 |

**如需 schema 或 default_params 变更**：必须创建新的 AlgorithmVersion（MAJOR 或 MINOR 版本递增），迁移 Binding，通知引用方升级。

---

## 4. 策略选择机制（两层协作）

策略选择是算法库的核心能力之一，通过 Registry 层和 Control 层的两层协作，将算法意图解析为具体的 Tool 调用。

### 4.1 Registry 层（默认策略）

Registry 层负责根据算法引用（AlgorithmRef）返回候选实现列表和默认推荐。

```go
// AlgorithmResolver 算法解析器接口
type AlgorithmResolver interface {
    // Resolve 返回候选实现列表，按 priority 排序
    // 调用方（Control 层）可在此基础上进一步过滤和排序
    Resolve(ctx context.Context, ref AlgorithmRef) ([]ResolvedBinding, error)

    // ResolveDefault 返回默认策略推荐的实现
    // 快捷方法，等价于 Resolve() 后取第一个结果
    ResolveDefault(ctx context.Context, ref AlgorithmRef) (*ResolvedBinding, error)
}
```

**AlgorithmRef（算法引用）**：

```go
// AlgorithmRef 工作流节点中对算法的引用
type AlgorithmRef struct {
    AlgorithmCode     string `json:"algorithm_code"`     // 算法编码，如 "object_detection"
    VersionConstraint string `json:"version_constraint"` // semver 版本约束，如 ">=1.0.0,<2.0.0"、"~1.2.0"、"^2.0"
    Strategy          string `json:"strategy"`           // 策略偏好："default"|"high_accuracy"|"low_cost"|"low_latency"|"balanced"|""（空表示使用默认策略）
}
```

**ResolvedBinding（解析结果）**：

```go
// ResolvedBinding 算法解析后的绑定结果
type ResolvedBinding struct {
    AlgorithmVersion      *AlgorithmVersion      `json:"algorithm_version"`
    ImplementationBinding *ImplementationBinding  `json:"implementation_binding"`
    Tool                  *ToolSpec              `json:"tool"`
    Score                 float64                `json:"score"`  // 综合匹配分数（基于 strategy + priority + performance_profile 计算）
}
```

**解析算法**：

```
输入: AlgorithmRef { code="object_detection", version=">=2.0", strategy="high_accuracy" }

1. 按 code 查找 Algorithm
2. 按 version_constraint 过滤 AlgorithmVersion（仅 status=published）
   - 匹配 semver range，如 >=2.0.0 匹配 2.0.0, 2.1.0, 2.5.3 等
   - 如有多个版本匹配，取最新版本（highest semver）
3. 获取该版本的所有 ImplementationBinding（仅 status=active）
4. 按 strategy 过滤：
   - 如 strategy 非空，优先返回匹配 strategy 的 Binding
   - 如无精确匹配，返回所有 active Binding（降级处理）
5. 计算综合分数（Score）：
   - 基于 priority（权重 40%）
   - 基于 performance_profile 与 strategy 的匹配度（权重 40%）
   - 基于 EvaluationProfile 最新指标（权重 20%）
6. 按 Score 降序排序返回
```

### 4.2 Control 层（运行时覆盖）

Control 层（Workflow Engine / Agent Runtime）在 Registry 层推荐的基础上，应用运行时策略覆盖。

**WorkflowDefinition.NodeConfig 中的策略偏好**：

```json
{
  "id": "detect_objects",
  "type": "algorithm",
  "algorithm_ref": {
    "algorithm_code": "object_detection",
    "version_constraint": ">=2.0",
    "strategy": "high_accuracy"
  },
  "strategy_preference": {
    "preferred_tool_ids": ["tool_yolov8_gpu"],
    "excluded_tool_ids": ["tool_deprecated_model"],
    "max_cost_per_call": 0.05,
    "max_latency_ms": 5000
  }
}
```

**WorkflowPolicy（工作流策略）**：

```go
type WorkflowPolicy struct {
    ToolWhitelist  []string  // 允许的 Tool ID 列表（为空表示不限制）
    ToolBlacklist  []string  // 禁止的 Tool ID 列表
    MaxCostPerRun  float64   // 单次 Run 最大成本（USD）
    MaxCostPerNode float64   // 单节点最大成本（USD）
    RequireHumanApproval bool // 是否需要人工审批
}
```

**运行时选择流程**：

```
1. Registry.Resolve(algorithm_ref)
   → 候选列表 [Binding-A(Score=0.9), Binding-B(Score=0.7), Binding-C(Score=0.5)]

2. 应用 WorkflowPolicy 过滤
   → 如 Binding-A 的 Tool 不在 whitelist 或在 blacklist 中，移除
   → 如 Binding-B 的 cost_per_call > max_cost_per_node，移除

3. 应用 NodeConfig.strategy_preference 过滤
   → 如有 preferred_tool_ids，匹配的 Binding 分数加权
   → 如有 excluded_tool_ids，匹配的 Binding 移除
   → 如有 max_latency_ms，超出的 Binding 移除

4. 重新排序 → 选择最优

5. 如可用列表为空 → 返回 ErrNoAvailableImplementation
```

### 4.3 策略分数计算公式

```go
func CalculateScore(binding *ImplementationBinding, strategy BindingStrategy) float64 {
    normalizedStrategy := normalizeStrategy(strategy)

    // 基础分：基于 priority（priority 越小分越高）
    // 允许 priority 输入异常时降级处理，避免除零与负数。
    priorityScore := safePriorityScore(binding.Priority)

    // 策略匹配分
    strategyScore := 0.5 // 默认分
    if normalizeStrategy(binding.Strategy) == normalizedStrategy {
        strategyScore = 1.0 // 精确匹配
    } else if normalizedStrategy == StrategyDefault {
        // default 策略对显式绑定不惩罚
        strategyScore = 0.7
    }

    // 性能画像分（根据 strategy 侧重不同指标）
    perfScore := calculatePerfScore(binding.PerformanceProfile, normalizedStrategy)

    // 加权合并
    return priorityScore*0.4 + strategyScore*0.4 + perfScore*0.2
}

func calculatePerfScore(profile PerformanceProfile, strategy BindingStrategy) float64 {
    switch strategy {
    case StrategyLowLatency:
        // 延迟越低分越高
        if profile.LatencyP50MS <= 0 {
            return 0.5 // 缺失或非法 latency，按中性分处理
        }
        return 1.0 / (1.0 + float64(profile.LatencyP50MS)/1000.0)
    case StrategyHighAccuracy:
        // 精度越高分越高
        if profile.AccuracyScore <= 0 {
            return 0.5
        }
        return profile.AccuracyScore
    case StrategyLowCost:
        // 成本越低分越高
        if profile.CostPerCall <= 0 {
            return 0.5
        }
        return 1.0 / (1.0 + profile.CostPerCall*100.0)
    case StrategyBalanced:
        // 综合评分
        latency := 0.5
        if profile.LatencyP50MS > 0 {
            latency = 1.0 / (1.0 + float64(profile.LatencyP50MS)/1000.0)
        }
        accuracy := profile.AccuracyScore
        if accuracy <= 0 {
            accuracy = 0.5
        }
        cost := 0.5
        if profile.CostPerCall > 0 {
            cost = 1.0 / (1.0 + profile.CostPerCall*100.0)
        }
        return (latency + accuracy + cost) / 3.0
    case StrategyDefault:
        // default 由 registry 默认绑定+priority 决定，性能分中性
        return 0.5
    default:
        return 0.5
    }
}

func safePriorityScore(priority int) float64 {
    if priority < 0 {
        priority = 0
    }
    return 1.0 / float64(priority+1)
}

func normalizeStrategy(strategy BindingStrategy) BindingStrategy {
    switch strategy {
    case StrategyDefault, StrategyHighAccuracy, StrategyLowCost, StrategyLowLatency, StrategyBalanced:
        return strategy
    default:
        return StrategyDefault
    }
}
```

### 4.4 评分边界处理

为避免策略打分在边界场景失真，统一采用以下规则：

1. **除零与负值保护**
   - `priority < 0` 视为 `0`；`priorityScore = 1/(priority+1)`，不会出现分母为 0。
   - 成本/延迟分数分母固定加偏移项（`1 + x`）防止除零。

2. **缺失指标降级**
   - 当 `latency_p50_ms`、`accuracy_score`、`cost_per_call` 缺失或非法时，对应维度使用中性分 `0.5`。
   - 缺失指标的绑定不会直接淘汰，避免因监控延迟导致可用实现误下线。

3. **权重分母保护**
   - 若未来支持可配置权重，要求 `sum(weights) > 0`；否则回退到默认权重 `0.4/0.4/0.2`。

4. **同分裁决（Tie-breaking）**
   - 第一优先：`priority` 更小者优先。
   - 第二优先：最近健康检查成功时间更新者优先。
   - 第三优先：`tool_id` 字典序稳定排序（保证结果可复现）。

---

## 5. 工作流集成

### 5.1 工作流节点中的算法引用

Workflow 节点通过 `type=algorithm` 声明对算法的引用，运行时由引擎解析为具体的 Tool 调用。

**节点定义示例**：

```json
{
  "id": "detect_objects",
  "type": "algorithm",
  "algorithm_ref": {
    "algorithm_code": "object_detection",
    "version_constraint": ">=2.0",
    "strategy": "high_accuracy"
  },
  "input_mapping": {
    "image_ref": "nodes.extract_frames.output.frame_refs[0]"
  },
  "output_mapping": {
    "nodes.detect_objects.detections": "detections",
    "nodes.detect_objects.processing_time": "processing_time_ms"
  },
  "param_overrides": {
    "confidence_threshold": 0.8,
    "target_classes": ["person", "vehicle"]
  },
  "retry_count": 2,
  "timeout_seconds": 30
}
```

### 5.2 运行时展开流程

当 DAG 引擎执行到 `type=algorithm` 节点时，执行以下展开流程：

```
步骤 1: 解析算法引用
    Registry.Resolve(algorithm_ref)
    → 候选列表 [ResolvedBinding...]

步骤 2: 应用工作流策略过滤
    WorkflowPolicy.filter(candidates)
    → 可用列表

步骤 3: 选择最优实现
    sort by Score → 取第一个
    → selectedBinding

步骤 4: 参数合并（优先级从低到高）
    AlgorithmVersion.default_params        (最低优先级)
    ↓ 覆盖
    ImplementationBinding.param_overrides  (中优先级)
    ↓ 覆盖
    NodeConfig.param_overrides             (最高优先级)
    → mergedParams

步骤 5: 构建执行信封
    ExecutionEnvelope {
        trace_id:    run.trace_id,
        run_id:      run.id,
        node_id:     node.id,
        tool_call_id: uuid.New(),
        tool_spec:   selectedBinding.Tool,
        input:       resolveInputMapping(contextState, node.input_mapping),
        params:      mergedParams,
        policy: {
            timeout:           node.timeout_seconds,
            domain_whitelist:  tool.data_access.domain_whitelist,
            resource_limits:   algorithmVersion.resource_profile,
        },
    }

步骤 6: 执行 Tool
    Runtime.Execute(envelope)
    → ToolResult

步骤 7: 写入上下文
    applyOutputMapping(toolResult, node.output_mapping)
    → ContextState.patch(...)

步骤 8: 产出 RunEvent
    emit RunEvent {
        type: "tool_succeeded",
        node_id: node.id,
        tool_name: selectedBinding.Tool.Name,
        algorithm_code: algorithm_ref.algorithm_code,
        algorithm_version: selectedBinding.AlgorithmVersion.Version,
        binding_strategy: selectedBinding.ImplementationBinding.Strategy,
        latency_ms: elapsed,
    }
```

### 5.3 降级策略

当选择的 Tool 执行失败时，引擎可按以下策略降级：

```
1. 重试当前 Tool（按 retry_count 配置）
2. 如重试耗尽 → 尝试候选列表中的下一个 Binding
3. 如所有候选均失败 → 节点标记为 failed
4. 产出 RunEvent: tool_failed + algorithm_fallback_attempted
```

---

## 6. 评测流程

评测是算法生命周期中的关键环节，确保算法版本在发布前经过充分验证。**评测本身作为一个特殊的 Workflow Run 执行**。

### 6.1 评测流程全景

```
步骤 1: 创建 EvaluationProfile
    status = pending
    输入: algorithm_version_id, input_dataset_ref

步骤 2: 构建评测 Workflow
    评测 Workflow 是一个预定义的 DAG 模板：
    ┌──────────┐    ┌──────────────┐    ┌──────────────┐    ┌───────────┐
    │ 加载数据集 │───►│ 遍历样本并调用 │───►│ 收集结果并对比 │───►│ 计算汇总指标│
    │          │    │ 算法版本     │    │ 真值标注     │    │           │
    └──────────┘    └──────────────┘    └──────────────┘    └───────────┘

步骤 3: 触发评测 Workflow Run
    创建 Run (type=WorkflowRun)
    EvaluationProfile.run_id = run.id
    EvaluationProfile.status = running

步骤 4: 评测执行
    - 加载数据集 Asset（structured 类型，包含样本列表与真值标注）
    - 对每个样本调用待评测的 AlgorithmVersion
    - 收集每个样本的输出结果
    - 与真值标注进行对比
    - 计算各项指标（accuracy, precision, recall, f1, latency 等）

步骤 5: 评测完成
    Workflow Run → succeeded
    → 产出 Artifacts:
       - 逐样本评测结果（structured 类型 Asset）
       - 汇总指标报告（structured 类型 Asset）
       - 可视化报告（document 类型 Asset，可选）

步骤 6: 回填 EvaluationProfile
    EvaluationProfile.status = completed
    EvaluationProfile.metrics = { accuracy: 0.95, precision: 0.93, ... }
    EvaluationProfile.completed_at = now()

步骤 7: 判断是否达标
    如果 metrics 满足预设阈值 → AlgorithmVersion 可从 draft → tested
    阈值可在 Algorithm 级别配置或由管理员人工判断
```

### 6.2 评测失败处理

```
如果评测 Workflow Run → failed:
    EvaluationProfile.status = failed
    保留失败 Run 的所有 RunEvents 用于排查
    AlgorithmVersion 保持当前状态（不自动变更）
    可重新触发评测
```

### 6.3 评测数据集规范

评测数据集作为 structured 类型 Asset 存储，格式定义：

```json
{
  "name": "COCO-val-2024-subset",
  "sample_count": 5000,
  "schema": {
    "type": "array",
    "items": {
      "type": "object",
      "required": ["id", "input", "ground_truth"],
      "properties": {
        "id": { "type": "string" },
        "input": {
          "type": "object",
          "description": "输入数据，格式与 AlgorithmVersion.input_schema 对齐"
        },
        "ground_truth": {
          "type": "object",
          "description": "真值标注，格式与 AlgorithmVersion.output_schema 对齐"
        }
      }
    }
  }
}
```

---

## 7. 接口定义

### 7.1 AlgorithmRepository（持久化接口）

```go
// AlgorithmRepository 算法仓储接口
// 位于 domain 包，由 Data Layer 实现
type AlgorithmRepository interface {
    // === Algorithm CRUD ===

    // CreateAlgorithm 创建算法
    CreateAlgorithm(ctx context.Context, alg *Algorithm) error

    // GetAlgorithm 按 ID 查询算法
    GetAlgorithm(ctx context.Context, id uuid.UUID) (*Algorithm, error)

    // GetAlgorithmByCode 按 code 查询算法
    GetAlgorithmByCode(ctx context.Context, code string) (*Algorithm, error)

    // ListAlgorithms 按条件过滤查询算法列表
    ListAlgorithms(ctx context.Context, filter AlgorithmFilter) ([]*Algorithm, int64, error)

    // UpdateAlgorithm 更新算法基本信息
    UpdateAlgorithm(ctx context.Context, alg *Algorithm) error

    // DeleteAlgorithm 删除算法（级联删除版本、绑定、评测档案）
    // 仅允许删除无 published 版本的算法
    DeleteAlgorithm(ctx context.Context, id uuid.UUID) error

    // === AlgorithmVersion CRUD ===

    // CreateVersion 创建算法版本
    CreateVersion(ctx context.Context, ver *AlgorithmVersion) error

    // GetVersion 按 ID 查询版本
    GetVersion(ctx context.Context, id uuid.UUID) (*AlgorithmVersion, error)

    // GetVersionByAlgorithmAndSemver 按算法 ID 和版本号查询
    GetVersionByAlgorithmAndSemver(ctx context.Context, algorithmID uuid.UUID, version string) (*AlgorithmVersion, error)

    // ListVersions 查询算法的所有版本
    ListVersions(ctx context.Context, algorithmID uuid.UUID) ([]*AlgorithmVersion, error)

    // ListPublishedVersions 查询算法的所有已发布版本
    ListPublishedVersions(ctx context.Context, algorithmID uuid.UUID) ([]*AlgorithmVersion, error)

    // UpdateVersion 更新版本信息（受冻结规则约束）
    UpdateVersion(ctx context.Context, ver *AlgorithmVersion) error

    // UpdateVersionStatus 更新版本生命周期状态
    UpdateVersionStatus(ctx context.Context, id uuid.UUID, status AlgorithmLifecycle) error

    // DeleteVersion 删除版本（仅允许删除 draft 状态的版本）
    DeleteVersion(ctx context.Context, id uuid.UUID) error

    // === ImplementationBinding CRUD ===

    // CreateBinding 创建实现绑定
    CreateBinding(ctx context.Context, binding *ImplementationBinding) error

    // GetBinding 按 ID 查询绑定
    GetBinding(ctx context.Context, id uuid.UUID) (*ImplementationBinding, error)

    // ListBindings 查询版本的所有绑定
    ListBindings(ctx context.Context, versionID uuid.UUID) ([]*ImplementationBinding, error)

    // ListActiveBindings 查询版本的所有活跃绑定
    ListActiveBindings(ctx context.Context, versionID uuid.UUID) ([]*ImplementationBinding, error)

    // UpdateBinding 更新绑定（param_overrides, priority, performance_profile）
    UpdateBinding(ctx context.Context, binding *ImplementationBinding) error

    // UpdateBindingStatus 更新绑定状态（active/disabled）
    UpdateBindingStatus(ctx context.Context, id uuid.UUID, status BindingStatus) error

    // DeleteBinding 删除绑定（仅允许删除 draft 状态版本的绑定）
    DeleteBinding(ctx context.Context, id uuid.UUID) error

    // === EvaluationProfile ===

    // CreateEvaluation 创建评测档案
    CreateEvaluation(ctx context.Context, eval *EvaluationProfile) error

    // GetEvaluation 按 ID 查询评测档案
    GetEvaluation(ctx context.Context, id uuid.UUID) (*EvaluationProfile, error)

    // ListEvaluations 查询版本的所有评测档案
    ListEvaluations(ctx context.Context, versionID uuid.UUID) ([]*EvaluationProfile, error)

    // UpdateEvaluation 更新评测档案（status, metrics, summary, completed_at）
    UpdateEvaluation(ctx context.Context, eval *EvaluationProfile) error

    // CountCompletedEvaluations 统计版本已完成的评测数量
    CountCompletedEvaluations(ctx context.Context, versionID uuid.UUID) (int64, error)
}
```

### 7.2 AlgorithmFilter（查询过滤器）

```go
type AlgorithmFilter struct {
    Category *AlgorithmCategory `json:"category,omitempty"` // 按分类过滤
    Tags     []string           `json:"tags,omitempty"`     // 按标签过滤（AND 语义）
    Keyword  string             `json:"keyword,omitempty"`  // 关键词搜索（name, code, description）
    TenantID *uuid.UUID         `json:"tenant_id,omitempty"`// 租户过滤
    Limit    int                `json:"limit"`
    Offset   int                `json:"offset"`
}
```

### 7.3 AlgorithmResolver（策略解析接口）

```go
// AlgorithmResolver 算法解析器接口
// 位于 Registry Layer，由 AlgorithmService 实现
type AlgorithmResolver interface {
    // Resolve 解析算法引用，返回候选实现列表
    Resolve(ctx context.Context, ref AlgorithmRef) ([]ResolvedBinding, error)

    // ResolveDefault 返回默认策略推荐的最优实现
    ResolveDefault(ctx context.Context, ref AlgorithmRef) (*ResolvedBinding, error)

    // ValidateRef 校验算法引用是否有效（算法存在、版本约束有匹配、有可用绑定）
    ValidateRef(ctx context.Context, ref AlgorithmRef) error
}
```

---

## 8. 数据库 Schema

### 8.1 algorithms 表

```sql
CREATE TABLE algorithms (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    code        VARCHAR(128) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    scene       TEXT NOT NULL DEFAULT '',
    problem     TEXT NOT NULL DEFAULT '',
    boundary    TEXT NOT NULL DEFAULT '',
    category    VARCHAR(32) NOT NULL,
    tags        JSONB NOT NULL DEFAULT '[]',
    tenant_id   UUID NOT NULL,
    created_by  UUID NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_algorithms_code UNIQUE (code),
    CONSTRAINT chk_algorithms_category CHECK (
        category IN ('detection', 'classification', 'segmentation', 'generation',
                     'transform', 'analysis', 'recognition', 'extraction')
    )
);

-- 索引
CREATE INDEX idx_algorithms_tenant_id ON algorithms (tenant_id);
CREATE INDEX idx_algorithms_category ON algorithms (category);
CREATE INDEX idx_algorithms_tags ON algorithms USING GIN (tags);
CREATE INDEX idx_algorithms_created_at ON algorithms (created_at DESC);

-- 全文搜索索引（name + description + scene + problem）
CREATE INDEX idx_algorithms_search ON algorithms USING GIN (
    to_tsvector('simple', coalesce(name, '') || ' ' || coalesce(description, '') || ' ' || coalesce(scene, '') || ' ' || coalesce(problem, ''))
);
```

### 8.2 algorithm_versions 表

```sql
CREATE TABLE algorithm_versions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    algorithm_id     UUID NOT NULL REFERENCES algorithms(id) ON DELETE CASCADE,
    version          VARCHAR(64) NOT NULL,
    input_schema     JSONB NOT NULL DEFAULT '{}',
    output_schema    JSONB NOT NULL DEFAULT '{}',
    default_params   JSONB NOT NULL DEFAULT '{}',
    resource_profile JSONB NOT NULL DEFAULT '{}',
    status           VARCHAR(16) NOT NULL DEFAULT 'draft',
    changelog        TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_algorithm_versions_alg_ver UNIQUE (algorithm_id, version),
    CONSTRAINT chk_algorithm_versions_status CHECK (
        status IN ('draft', 'tested', 'published', 'deprecated')
    )
);

-- 索引
CREATE INDEX idx_algorithm_versions_algorithm_id ON algorithm_versions (algorithm_id);
CREATE INDEX idx_algorithm_versions_status ON algorithm_versions (status);
CREATE INDEX idx_algorithm_versions_created_at ON algorithm_versions (created_at DESC);
```

### 8.3 implementation_bindings 表

```sql
CREATE TABLE implementation_bindings (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    algorithm_version_id  UUID NOT NULL REFERENCES algorithm_versions(id) ON DELETE CASCADE,
    tool_id               VARCHAR(128) NOT NULL,
    priority              INT NOT NULL DEFAULT 100,
    strategy              VARCHAR(16) NOT NULL DEFAULT 'default',
    param_overrides       JSONB NOT NULL DEFAULT '{}',
    performance_profile   JSONB NOT NULL DEFAULT '{}',
    status                VARCHAR(16) NOT NULL DEFAULT 'active',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_bindings_strategy CHECK (
        strategy IN ('default', 'high_accuracy', 'low_cost', 'low_latency', 'balanced')
    ),
    CONSTRAINT chk_bindings_status CHECK (
        status IN ('active', 'disabled')
    ),
    CONSTRAINT chk_bindings_priority CHECK (priority >= 0)
);

-- 索引
CREATE INDEX idx_bindings_version_id ON implementation_bindings (algorithm_version_id);
CREATE INDEX idx_bindings_tool_id ON implementation_bindings (tool_id);
CREATE INDEX idx_bindings_strategy ON implementation_bindings (strategy);
CREATE INDEX idx_bindings_status ON implementation_bindings (status);
-- 复合索引：按版本查找活跃绑定并按优先级排序
CREATE INDEX idx_bindings_version_active_priority ON implementation_bindings (algorithm_version_id, status, priority)
    WHERE status = 'active';
```

### 8.4 evaluation_profiles 表

```sql
CREATE TABLE evaluation_profiles (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    algorithm_version_id  UUID NOT NULL REFERENCES algorithm_versions(id) ON DELETE CASCADE,
    run_id                UUID NOT NULL,
    metrics               JSONB NOT NULL DEFAULT '{}',
    input_dataset_ref     VARCHAR(512) NOT NULL DEFAULT '',
    dataset_info          JSONB NOT NULL DEFAULT '{}',
    status                VARCHAR(16) NOT NULL DEFAULT 'pending',
    summary               TEXT NOT NULL DEFAULT '',
    completed_at          TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_eval_status CHECK (
        status IN ('pending', 'running', 'completed', 'failed')
    )
);

-- 索引
CREATE INDEX idx_eval_version_id ON evaluation_profiles (algorithm_version_id);
CREATE INDEX idx_eval_run_id ON evaluation_profiles (run_id);
CREATE INDEX idx_eval_status ON evaluation_profiles (status);
CREATE INDEX idx_eval_completed_at ON evaluation_profiles (completed_at DESC)
    WHERE completed_at IS NOT NULL;
-- 复合索引：按版本查找已完成的评测
CREATE INDEX idx_eval_version_completed ON evaluation_profiles (algorithm_version_id, status)
    WHERE status = 'completed';
```

---

## 9. 与其他模块的关系

| 关联模块 | 关系说明 |
|---------|---------|
| Tool Registry（04-tool-system.md） | ImplementationBinding.tool_id 引用 Tool Registry 中的工具 |
| Workflow Engine（06-workflow-engine.md） | 工作流节点通过 algorithm_ref 引用算法，运行时由引擎调用 AlgorithmResolver |
| Run / RunEvent（08-observability.md） | 评测作为 WorkflowRun 执行；算法选择过程产出 RunEvent |
| Asset Store（03-asset-system.md） | 评测数据集和评测结果作为 Asset 存储 |
| Policy Engine（09-security-policy.md） | WorkflowPolicy 可限制算法版本的 Tool 白名单 |

---

## 10. 开发实施建议

### 10.1 实施阶段

| 阶段 | 内容 | 依赖 |
|------|------|------|
| P1 | Algorithm + AlgorithmVersion CRUD、生命周期状态机 | 无 |
| P2 | ImplementationBinding CRUD、策略选择机制（AlgorithmResolver） | P1 + Tool Registry |
| P3 | 工作流节点 algorithm_ref 集成、运行时展开 | P2 + Workflow Engine |
| P4 | EvaluationProfile + 评测 Workflow 模板 | P3 + Run 体系 |
| P5 | 发布冻结规则强制、降级策略 | P3 |

### 10.2 测试策略

| 测试类型 | 覆盖范围 |
|---------|---------|
| 单元测试 | 生命周期状态转换规则、发布冻结校验、策略分数计算 |
| 集成测试 | AlgorithmResolver 完整解析链路、参数合并逻辑 |
| 端到端测试 | 工作流中 algorithm_ref 的运行时展开与执行 |
