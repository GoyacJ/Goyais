# Goyais 领域模型设计

> **最后更新**: 2026-02-09

---

## 1. 概述

本文档定义 Goyais 中所有一等公民对象（First-class Entity）的完整结构、枚举常量与接口契约。核心设计决策是**将 domain 包同时承担数据结构定义与接口定义（Port 角色）**，所有层（Control、Registry、Runtime、Data、Access）均实现 domain 中声明的接口，从而将依赖关系严格指向内层。

### 1.1 设计原则

- **domain 包零外部依赖**：仅依赖标准库与 `github.com/google/uuid`。
- **接口即契约**：所有跨层交互通过 domain 包中定义的接口完成，禁止层间直接依赖。
- **值对象不可变**：Spec / Policy / Config 类型作为值对象，创建后语义上不可变。
- **实体有身份**：所有一等实体均以 `uuid.UUID` 为主键。
- **枚举穷尽**：所有状态/类型使用 `type Xxx string` 定义，配合 `const` 块穷尽所有合法值。

### 1.2 包结构

```
internal/domain/
├── types.go           # 共享类型（JSONMap, JSONSchema, ResourceProfile 等）
├── asset.go           # Asset + StreamAsset
├── tool.go            # ToolSpec + CostHint + RetryPolicy
├── run.go             # Run + RunError + RunBudget
├── run_event.go       # RunEvent
├── workflow.go        # WorkflowDefinition + NodeDefinition + EdgeDefinition
├── agent.go           # AgentProfile + AgentState + AgentPlan
├── algorithm.go       # Algorithm + AlgorithmVersion + ImplementationBinding + EvaluationProfile
├── context.go         # ContextSpec + ContextState + ContextPatch
├── intent.go          # Intent + IntentPlan + IntentAction
├── identity.go        # User + Role + RoleBinding
├── envelope.go        # ExecutionEnvelope + ToolResult
├── interfaces.go      # 所有 Repository / Service / Engine 接口
└── errors.go          # 领域级错误定义
```

---

## 2. 共享类型（Shared Types）

```go
package domain

import (
    "time"
)

// JSONMap 是通用的 JSON 对象类型，用于存储非结构化数据。
type JSONMap map[string]any

// JSONSchema 表示 JSON Schema 定义，用于输入输出校验。
// 内部结构遵循 JSON Schema Draft 2020-12 规范。
type JSONSchema map[string]any

// ResourceProfile 描述计算资源需求画像。
type ResourceProfile struct {
    CPUCores   float64 `json:"cpu_cores,omitempty"`   // CPU 核数需求
    MemoryMB   int     `json:"memory_mb,omitempty"`   // 内存需求（MB）
    GPUType    string  `json:"gpu_type,omitempty"`    // GPU 型号要求（如 "nvidia-a100"）
    GPUCount   int     `json:"gpu_count,omitempty"`   // GPU 数量
    DiskMB     int     `json:"disk_mb,omitempty"`     // 临时磁盘需求（MB）
    NetworkMBs int     `json:"network_mbs,omitempty"` // 网络带宽需求（MB/s）
}

// ResourceLimits 描述运行时资源硬限制。
type ResourceLimits struct {
    MaxCPUCores   float64       `json:"max_cpu_cores,omitempty"`
    MaxMemoryMB   int           `json:"max_memory_mb,omitempty"`
    MaxGPUCount   int           `json:"max_gpu_count,omitempty"`
    MaxDiskMB     int           `json:"max_disk_mb,omitempty"`
    MaxDuration   time.Duration `json:"max_duration,omitempty"`
    MaxOutputSize int64         `json:"max_output_size,omitempty"` // 最大输出大小（字节）
}

// DataAccessSpec 定义数据访问域限制。
type DataAccessSpec struct {
    BucketPrefixes    []string `json:"bucket_prefixes,omitempty"`    // 允许访问的存储桶前缀
    DBScopes          []string `json:"db_scopes,omitempty"`          // 允许访问的数据库域
    DomainWhitelist   []string `json:"domain_whitelist,omitempty"`   // 允许访问的网络域名
    ReadScopes        []string `json:"read_scopes,omitempty"`        // 读权限范围
    WriteScopes       []string `json:"write_scopes,omitempty"`       // 写权限范围
}

// BudgetStatus 预算状态枚举。
type BudgetStatus string

const (
    BudgetStatusHealthy  BudgetStatus = "healthy"  // 预算健康（< 80%）
    BudgetStatusWarning  BudgetStatus = "warning"  // 预算告警（>= 80%）
    BudgetStatusExceeded BudgetStatus = "exceeded" // 预算超限（>= 100%）
)

// BudgetState 预算运行态。
// 供 PolicyCheckRequest、Run 治理与可观测事件统一引用。
type BudgetState struct {
    MaxCostUSD      float64      `json:"max_cost_usd,omitempty"`
    UsedCostUSD     float64      `json:"used_cost_usd,omitempty"`
    MaxToolCalls    int          `json:"max_tool_calls,omitempty"`
    UsedToolCalls   int          `json:"used_tool_calls,omitempty"`
    MaxTokens       int          `json:"max_tokens,omitempty"`
    UsedTokens      int          `json:"used_tokens,omitempty"`
    MaxDurationSec  int64        `json:"max_duration_sec,omitempty"`
    UsedDurationSec int64        `json:"used_duration_sec,omitempty"`
    Status          BudgetStatus `json:"status"`
    UpdatedAt       time.Time    `json:"updated_at"`
}

// ApprovalStatus 审批状态枚举。
type ApprovalStatus string

const (
    ApprovalStatusNotRequired ApprovalStatus = "not_required"
    ApprovalStatusPending     ApprovalStatus = "pending"
    ApprovalStatusApproved    ApprovalStatus = "approved"
    ApprovalStatusRejected    ApprovalStatus = "rejected"
    ApprovalStatusExpired     ApprovalStatus = "expired"
)

// ApprovalMode 审批模式枚举。
type ApprovalMode string

const (
    ApprovalModeSingle ApprovalMode = "single" // 单人审批（high 风险默认）
    ApprovalModeDual   ApprovalMode = "dual"   // 双人审批（critical 风险默认）
)

// ApprovalVote 单次审批投票记录。
type ApprovalVote struct {
    ApproverID string         `json:"approver_id"`      // 审批人 ID
    Decision   ApprovalStatus `json:"decision"`         // approved/rejected
    Comment    string         `json:"comment,omitempty"`
    VotedAt    time.Time      `json:"voted_at"`
}

// ApprovalState 审批运行态。
// 用于表达高风险动作的人审门禁状态。
type ApprovalState struct {
    Required          bool           `json:"required"`
    Mode              ApprovalMode   `json:"mode"`                             // single/dual
    RequiredApprovers int            `json:"required_approvers,omitempty"`     // 1/2...
    ApprovedBy        []string       `json:"approved_by,omitempty"`            // 已通过审批人
    RejectedBy        []string       `json:"rejected_by,omitempty"`            // 已拒绝审批人
    QuorumReached     bool           `json:"quorum_reached"`                   // 是否满足法定通过人数
    Votes             []ApprovalVote `json:"votes,omitempty"`                  // 审批投票明细
    TicketID          string         `json:"ticket_id,omitempty"`
    Status            ApprovalStatus `json:"status"`
    RequestedBy       string         `json:"requested_by,omitempty"`
    RequestedAt       *time.Time     `json:"requested_at,omitempty"`
    ResolvedBy        string         `json:"resolved_by,omitempty"`            // 最后使状态收敛的审批人
    ResolvedAt        *time.Time     `json:"resolved_at,omitempty"`
    TimeoutAt         *time.Time     `json:"timeout_at,omitempty"`
    Reason            string         `json:"reason,omitempty"`
}

// TriggerConfig 定义工作流触发配置。
type TriggerConfig struct {
    Type        TriggerType    `json:"type"`                    // manual/schedule/event
    Schedule    string         `json:"schedule,omitempty"`      // cron 表达式
    IntervalSec int            `json:"interval_sec,omitempty"`  // 间隔秒数
    EventType   string         `json:"event_type,omitempty"`    // 事件类型
    EventFilter JSONMap        `json:"event_filter,omitempty"`  // 事件过滤条件
}

// TriggerType 触发类型枚举。
type TriggerType string

const (
    TriggerManual   TriggerType = "manual"
    TriggerSchedule TriggerType = "schedule"
    TriggerEvent    TriggerType = "event"
)

// MappingRules 定义输入输出映射规则。
// Key 为目标路径，Value 为源路径表达式。
// 路径格式：ctx.vars.<key> | ctx.assets.<key> | ctx.artifacts.<key> | ctx.nodes.<node_id>.<key> | input.<key>
type MappingRules map[string]string

// WritePolicy 定义节点对共享上下文键的写入策略。
type WritePolicy struct {
    AllowedKeys      []string `json:"allowed_keys,omitempty"`       // 允许写入的共享键列表
    RequireCAS       bool     `json:"require_cas,omitempty"`        // 是否强制使用 CAS
    ConflictStrategy string   `json:"conflict_strategy,omitempty"`  // reject/overwrite/merge/append
}

// OutputMapping 定义工作流输出映射。
// Key 为工作流输出字段名，Value 为 artifact/context 路径表达式。
type OutputMapping map[string]string

// AssetRef 资产引用。
type AssetRef struct {
    AssetID  string `json:"asset_id"`
    URI      string `json:"uri,omitempty"`
    MimeType string `json:"mime_type,omitempty"`
    Summary  string `json:"summary,omitempty"` // 摘要，用于上下文记忆
}

// ArtifactRef 产物引用。
type ArtifactRef struct {
    ArtifactID string `json:"artifact_id"`
    NodeID     string `json:"node_id"`
    Type       string `json:"type"`
    URI        string `json:"uri,omitempty"`
    Summary    string `json:"summary,omitempty"`
}

// NodeState 节点隔离状态空间。
type NodeState struct {
    Status    string  `json:"status"`
    Output    JSONMap `json:"output,omitempty"`
    Error     string  `json:"error,omitempty"`
    Custom    JSONMap `json:"custom,omitempty"`    // 节点自定义状态
}

// ContextMeta 上下文元信息。
type ContextMeta struct {
    TraceID     string   `json:"trace_id"`
    TenantID    string   `json:"tenant_id,omitempty"`
    ProjectID   string   `json:"project_id,omitempty"`
    Permissions []string `json:"permissions,omitempty"`
}

// ArtifactOutput 工具执行产物输出。
type ArtifactOutput struct {
    Type     string `json:"type"`               // asset/result/timeline/report
    URI      string `json:"uri,omitempty"`       // 文件类产物的存储地址
    Data     JSONMap `json:"data,omitempty"`      // 结构化产物的数据
    MimeType string `json:"mime_type,omitempty"`
    Size     int64  `json:"size,omitempty"`
    Summary  string `json:"summary,omitempty"`   // 产物摘要
}

// Diagnostics 执行诊断信息。
type Diagnostics struct {
    LatencyMs    int64  `json:"latency_ms,omitempty"`
    ModelVersion string `json:"model_version,omitempty"`
    Device       string `json:"device,omitempty"`
    TokensUsed   int    `json:"tokens_used,omitempty"`
    CostUSD      float64 `json:"cost_usd,omitempty"`
    Extra        JSONMap `json:"extra,omitempty"`
}

// ContextRef 上下文引用（用于 ExecutionEnvelope，仅传引用不传全量）。
type ContextRef struct {
    Path    string `json:"path"`              // 上下文路径
    Version int64  `json:"version,omitempty"` // 引用时的版本号
}
```

---

## 3. Asset（资产）

资产是系统的核心数据对象，代表系统中所有可被引用、处理、流转的媒体与数据实体。`Asset` 为统一资产模型，流媒体资产通过 `StreamAsset` 嵌入扩展。

### 3.1 实体定义

```go
package domain

import (
    "time"

    "github.com/google/uuid"
)

// AssetType 资产类型枚举。
type AssetType string

const (
    AssetTypeVideo      AssetType = "video"      // 视频文件
    AssetTypeImage      AssetType = "image"      // 图片文件
    AssetTypeAudio      AssetType = "audio"      // 音频文件
    AssetTypeDocument   AssetType = "document"   // 文档（PDF/Word/Excel 等）
    AssetTypeStream     AssetType = "stream"     // 实时流媒体
    AssetTypeStructured AssetType = "structured" // 结构化数据（JSON/CSV/数据库导出）
    AssetTypeText       AssetType = "text"       // 纯文本
)

// AssetStatus 资产状态枚举。
type AssetStatus string

const (
    AssetStatusActive   AssetStatus = "active"   // 正常可用
    AssetStatusArchived AssetStatus = "archived" // 已归档（仍可读取，不参与新任务）
    AssetStatusDeleted  AssetStatus = "deleted"  // 已标记删除（软删除，等待清理）
)

// Asset 资产实体。
// 代表系统中所有可被引用、处理和流转的媒体与数据资源。
// URI 格式约定：
//   - 对象存储：s3://bucket/path/to/file
//   - 本地文件：file:///absolute/path
//   - 流媒体：rtsp://host/path 或 rtmp://host/path
//   - 外部链接：https://example.com/resource
type Asset struct {
    ID        uuid.UUID   `gorm:"type:uuid;primary_key;default:gen_random_uuid()"` // 资产唯一标识
    TenantID  uuid.UUID   `gorm:"type:uuid;not null;index"`                        // 租户 ID
    OwnerID   uuid.UUID   `gorm:"type:uuid;not null;index"`                        // 所有者 ID
    Name      string      `gorm:"type:varchar(500);not null"`                      // 资产名称
    Type      AssetType   `gorm:"type:varchar(50);not null;index"`                 // 资产类型
    URI       string      `gorm:"type:varchar(2000);not null"`                     // 资源定位符
    Digest    string      `gorm:"type:varchar(128)"`                               // 内容摘要（推荐 sha256）
    MimeType  string      `gorm:"type:varchar(200)"`                               // MIME 类型（如 video/mp4）
    Size      int64       `gorm:"type:bigint;default:0"`                           // 文件大小（字节）
    Metadata  JSONMap     `gorm:"type:jsonb"`                                      // 扩展元数据（按 Type 不同含义不同）
    ParentID  *uuid.UUID  `gorm:"type:uuid;index"`                                // 父资产 ID（衍生资产追踪）
    Tags      StringArray `gorm:"type:text[]"`                                     // 标签列表
    Status    AssetStatus `gorm:"type:varchar(20);not null;default:'active';index"` // 资产状态
    CreatedAt time.Time   `gorm:"not null;index"`                                  // 创建时间
    UpdatedAt time.Time   `gorm:"not null"`                                        // 更新时间
}

// StringArray 自定义类型，用于 PostgreSQL text[] 类型映射。
type StringArray []string

// TableName 指定数据库表名。
func (Asset) TableName() string {
    return "assets"
}

// IsStream 判断是否为流媒体资产。
func (a *Asset) IsStream() bool {
    return a.Type == AssetTypeStream
}

// HasParent 判断是否为衍生资产。
func (a *Asset) HasParent() bool {
    return a.ParentID != nil
}

// IsActive 判断资产是否处于活跃状态。
func (a *Asset) IsActive() bool {
    return a.Status == AssetStatusActive
}

// Archive 归档资产。
func (a *Asset) Archive() {
    a.Status = AssetStatusArchived
    a.UpdatedAt = time.Now()
}

// SoftDelete 软删除资产。
func (a *Asset) SoftDelete() {
    a.Status = AssetStatusDeleted
    a.UpdatedAt = time.Now()
}
```

### 3.2 Metadata 结构约定

`Metadata` 字段为 JSONMap，其内部结构因 `AssetType` 不同而不同：

| AssetType | Metadata 关键字段 | 说明 |
|-----------|-----------------|------|
| `video` | `duration`, `width`, `height`, `fps`, `codec`, `bitrate` | 视频元信息 |
| `image` | `width`, `height`, `color_space`, `dpi` | 图片元信息 |
| `audio` | `duration`, `sample_rate`, `channels`, `codec`, `bitrate` | 音频元信息 |
| `document` | `page_count`, `sheet_names`, `encoding` | 文档元信息 |
| `stream` | 见 StreamAsset 扩展 | 流媒体扩展信息 |
| `structured` | `schema`, `row_count`, `column_count` | 结构化数据元信息 |
| `text` | `encoding`, `line_count`, `language` | 文本元信息 |

### 3.3 StreamAsset 扩展

```go
// StreamAsset 流媒体资产扩展。
// 嵌入基础 Asset，增加流媒体专属字段。
// 在数据库层面通过 assets 表的 Metadata JSONB 存储扩展字段；
// 在领域层面提供类型安全的访问方式。
type StreamAsset struct {
    Asset                                                                     // 嵌入基础资产
    StreamProtocol StreamProtocol `json:"stream_protocol"`                    // 流协议
    MediaMTXPath   string         `json:"mediamtx_path"`                     // MediaMTX 路由路径
    SliceIndex     JSONMap        `json:"slice_index,omitempty"`             // 时间轴切片索引
    RecordingPath  string         `json:"recording_path,omitempty"`          // 录制文件路径模式
}

// StreamProtocol 流协议枚举。
type StreamProtocol string

const (
    StreamProtocolRTSP   StreamProtocol = "rtsp"
    StreamProtocolRTMP   StreamProtocol = "rtmp"
    StreamProtocolHLS    StreamProtocol = "hls"
    StreamProtocolWebRTC StreamProtocol = "webrtc"
)

// ToStreamAsset 从 Asset 构造 StreamAsset。
// 从 Asset.Metadata 中解析流媒体扩展字段。
func ToStreamAsset(a *Asset) *StreamAsset {
    if a == nil || a.Type != AssetTypeStream {
        return nil
    }
    sa := &StreamAsset{Asset: *a}
    if v, ok := a.Metadata["stream_protocol"].(string); ok {
        sa.StreamProtocol = StreamProtocol(v)
    }
    if v, ok := a.Metadata["mediamtx_path"].(string); ok {
        sa.MediaMTXPath = v
    }
    if v, ok := a.Metadata["slice_index"].(map[string]any); ok {
        sa.SliceIndex = v
    }
    if v, ok := a.Metadata["recording_path"].(string); ok {
        sa.RecordingPath = v
    }
    return sa
}
```

### 3.4 过滤器

```go
// AssetFilter 资产查询过滤条件。
type AssetFilter struct {
    TenantID *uuid.UUID   // 租户过滤
    OwnerID  *uuid.UUID   // 所有者过滤
    Type     *AssetType   // 类型过滤
    Status   *AssetStatus // 状态过滤
    ParentID *uuid.UUID   // 父资产过滤
    Tags     []string     // 标签过滤（AND 语义）
    Keyword  string       // 名称关键字搜索
    URILike  string       // URI 前缀匹配
    From     *time.Time   // 创建时间起始
    To       *time.Time   // 创建时间截止
    Limit    int          // 分页大小
    Offset   int          // 分页偏移
}
```

### 3.5 接口

```go
// AssetRepository 资产持久化接口。
type AssetRepository interface {
    Create(ctx context.Context, asset *Asset) error
    GetByID(ctx context.Context, id uuid.UUID) (*Asset, error)
    List(ctx context.Context, filter AssetFilter) ([]*Asset, int64, error)
    Update(ctx context.Context, asset *Asset) error
    Delete(ctx context.Context, id uuid.UUID) error
    ListByParentID(ctx context.Context, parentID uuid.UUID) ([]*Asset, error)
    CountByType(ctx context.Context, tenantID uuid.UUID) (map[AssetType]int64, error)
}

// AssetStore 资产存储接口（文件存储层抽象）。
// 实现方包括 MinIO/S3、本地文件系统等。
type AssetStore interface {
    // Upload 上传文件到存储后端。
    // path: 存储路径（不含 bucket 前缀）
    // reader: 文件内容读取器
    // metadata: 自定义元数据（如 content-type）
    // 返回完整 URI（如 s3://bucket/path）
    Upload(ctx context.Context, path string, reader io.Reader, metadata map[string]string) (uri string, err error)

    // Download 从存储后端下载文件。
    Download(ctx context.Context, uri string) (io.ReadCloser, error)

    // Delete 删除存储后端中的文件。
    Delete(ctx context.Context, uri string) error

    // GetPresignedURL 生成带时效的预签名访问 URL。
    GetPresignedURL(ctx context.Context, uri string, expiry time.Duration) (string, error)

    // Exists 检查文件是否存在。
    Exists(ctx context.Context, uri string) (bool, error)
}
```

---

## 4. Tool（工具）

工具是系统的能力原子单元。`ToolSpec` 统一涵盖算子、MCP 工具、HTTP API、CLI 命令、AI 模型调用、复合工具等六种执行类别。每个 ToolSpec 是一份自描述的能力契约，包含输入输出 Schema、副作用声明、风险等级、执行模式与成本估算。

### 4.1 实体定义

```go
package domain

import (
    "time"

    "github.com/google/uuid"
)

// ToolCategory 工具类别枚举。
type ToolCategory string

const (
    ToolCategoryOperator  ToolCategory = "operator"  // 传统算子（HTTP/CLI）
    ToolCategoryMCP       ToolCategory = "mcp"       // MCP 协议工具
    ToolCategoryHTTP      ToolCategory = "http"      // 通用 HTTP API
    ToolCategoryCLI       ToolCategory = "cli"       // 命令行工具
    ToolCategoryModel     ToolCategory = "model"     // AI 模型调用
    ToolCategoryComposite ToolCategory = "composite" // 复合工具（内部编排多个子工具）
)

// ExecutionMode 执行模式枚举。
type ExecutionMode string

const (
    ExecModeInProcess  ExecutionMode = "in_process"  // 进程内执行
    ExecModeSubprocess ExecutionMode = "subprocess"  // 子进程执行
    ExecModeContainer  ExecutionMode = "container"   // 容器执行
    ExecModeRemote     ExecutionMode = "remote"      // 远程调用
)

// RiskLevel 风险等级枚举。
type RiskLevel string

const (
    RiskLevelLow    RiskLevel = "low"    // 低风险：只读操作、无副作用
    RiskLevelMedium RiskLevel = "medium" // 中风险：有写入副作用、可恢复
    RiskLevelHigh   RiskLevel = "high"   // 高风险：不可逆操作、需审批
    RiskLevelCritical RiskLevel = "critical" // 严重风险：跨租户/高破坏操作，需强审批
)

// SideEffect 副作用类型枚举。
type SideEffect string

const (
    SideEffectRead    SideEffect = "read"    // 读取外部数据
    SideEffectWrite   SideEffect = "write"   // 写入外部数据
    SideEffectNetwork SideEffect = "network" // 网络请求
    SideEffectDelete  SideEffect = "delete"  // 删除操作
    SideEffectNotify  SideEffect = "notify"  // 发送通知
)

// DeterminismType 确定性类型枚举。
type DeterminismType string

const (
    DeterminismDeterministic DeterminismType = "deterministic" // 确定性：相同输入产生相同输出
    DeterminismStochastic    DeterminismType = "stochastic"    // 随机性：相同输入可能产生不同输出
)

// ToolStatus 工具状态枚举。
type ToolStatus string

const (
    ToolStatusActive     ToolStatus = "active"     // 正常可用
    ToolStatusDeprecated ToolStatus = "deprecated" // 已弃用（仍可调用，但不推荐）
    ToolStatusDisabled   ToolStatus = "disabled"   // 已禁用（不可调用）
)

// ToolSpec 工具规范实体。
// 是系统中所有可调用能力的统一描述格式。
type ToolSpec struct {
    ID                  uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    TenantID            uuid.UUID       `gorm:"type:uuid;not null;index"`
    Name                string          `gorm:"type:varchar(200);not null"`
    Code                string          `gorm:"type:varchar(200);not null;uniqueIndex"`
    Description         string          `gorm:"type:text"`
    Category            ToolCategory    `gorm:"type:varchar(50);not null;index"`
    InputSchema         JSONSchema      `gorm:"type:jsonb"`
    OutputSchema        JSONSchema      `gorm:"type:jsonb"`
    SideEffects         SideEffectArray `gorm:"type:text[]"`
    RiskLevel           RiskLevel       `gorm:"type:varchar(20);not null;default:'low'"`
    RequiredPermissions StringArray     `gorm:"type:text[]"`
    ExecutionMode       ExecutionMode   `gorm:"type:varchar(50);not null;default:'remote'"`
    TimeoutMs           int64           `gorm:"type:bigint;default:30000"`
    RetryPolicy         RetryPolicy     `gorm:"type:jsonb"`
    Idempotent          bool            `gorm:"type:boolean;default:false"`
    CostHint            CostHint        `gorm:"type:jsonb"`
    Determinism         DeterminismType `gorm:"type:varchar(50);not null;default:'deterministic'"`
    DataAccess          DataAccessSpec  `gorm:"type:jsonb"`
    Version             string          `gorm:"type:varchar(50)"`
    Status              ToolStatus      `gorm:"type:varchar(20);not null;default:'active';index"`
    Config              JSONMap         `gorm:"type:jsonb"`
    CreatedAt           time.Time       `gorm:"not null"`
    UpdatedAt           time.Time       `gorm:"not null"`
}

// SideEffectArray 副作用数组类型。
type SideEffectArray []SideEffect

func (ToolSpec) TableName() string {
    return "tools"
}

// IsActive 判断工具是否处于活跃状态。
func (t *ToolSpec) IsActive() bool {
    return t.Status == ToolStatusActive
}

// IsHighRisk 判断工具是否为高风险。
func (t *ToolSpec) IsHighRisk() bool {
    return t.RiskLevel == RiskLevelHigh || t.RiskLevel == RiskLevelCritical
}

// HasSideEffect 判断工具是否具有指定副作用。
func (t *ToolSpec) HasSideEffect(effect SideEffect) bool {
    for _, e := range t.SideEffects {
        if e == effect {
            return true
        }
    }
    return false
}

// CostHint 成本估算提示。
// 用于 Agent 规划阶段的成本预判与预算管控。
type CostHint struct {
    EstTokens    int     `json:"est_tokens,omitempty"`     // 预估 Token 消耗
    EstPriceUSD  float64 `json:"est_price_usd,omitempty"`  // 预估美元成本
    EstLatencyMs int     `json:"est_latency_ms,omitempty"` // 预估延迟（毫秒）
    RequiresCPU  bool    `json:"requires_cpu,omitempty"`   // 是否需要 CPU 密集计算
    RequiresGPU  bool    `json:"requires_gpu,omitempty"`   // 是否需要 GPU
}

// RetryPolicy 重试策略。
type RetryPolicy struct {
    MaxAttempts int    `json:"max_attempts"`           // 最大重试次数（含首次）
    BackoffMs   int64  `json:"backoff_ms"`             // 初始退避时间（毫秒）
    BackoffType string `json:"backoff_type"`           // fixed/exponential
    MaxBackoffMs int64 `json:"max_backoff_ms,omitempty"` // 最大退避时间（毫秒）
}
```

### 4.2 过滤器

```go
// ToolFilter 工具查询过滤条件。
type ToolFilter struct {
    TenantID    *uuid.UUID    // 租户过滤
    Category    *ToolCategory // 类别过滤
    Status      *ToolStatus   // 状态过滤
    RiskLevel   *RiskLevel    // 风险等级过滤
    Determinism *DeterminismType // 确定性过滤
    Codes       []string      // 按 code 列表过滤
    Keyword     string        // 名称/描述关键字搜索
    Limit       int           // 分页大小
    Offset      int           // 分页偏移
}
```

### 4.3 接口

```go
// ToolRegistry 工具注册表接口。
// 管理 ToolSpec 的生命周期（注册、查询、更新、弃用）。
type ToolRegistry interface {
    // Register 注册新工具。Code 必须全局唯一。
    Register(ctx context.Context, spec *ToolSpec) error

    // Get 按 ID 获取工具规范。
    Get(ctx context.Context, id uuid.UUID) (*ToolSpec, error)

    // GetByCode 按唯一编码获取工具规范。
    GetByCode(ctx context.Context, code string) (*ToolSpec, error)

    // List 按条件查询工具列表。
    List(ctx context.Context, filter ToolFilter) ([]*ToolSpec, int64, error)

    // Update 更新工具规范。
    Update(ctx context.Context, spec *ToolSpec) error

    // Deprecate 弃用工具。状态变更为 deprecated。
    Deprecate(ctx context.Context, id uuid.UUID) error

    // Disable 禁用工具。状态变更为 disabled，不可再被调用。
    Disable(ctx context.Context, id uuid.UUID) error
}

// ToolExecutor 工具执行器接口。
// 负责接收执行信封并完成实际工具调用。
type ToolExecutor interface {
    // Execute 执行工具调用。
    // 接收完整的 ExecutionEnvelope（含安全策略、资源限制等），返回 ToolResult。
    Execute(ctx context.Context, envelope ExecutionEnvelope) (*ToolResult, error)

    // Validate 校验工具输入是否符合 InputSchema。
    Validate(ctx context.Context, spec *ToolSpec, input JSONMap) error

    // HealthCheck 工具健康检查。
    HealthCheck(ctx context.Context, spec *ToolSpec) error
}
```

---

## 5. Run（运行）

运行是系统的执行实例。`Run` 统一表示三种执行类型：工作流运行（workflow）、Agent 会话运行（agent）、单次工具调用（tool_call）。通过 `ParentRunID` 支持嵌套执行（如工作流中的 Agent 节点、Agent 发起的子工作流）。

### 5.1 实体定义

```go
package domain

import (
    "time"

    "github.com/google/uuid"
)

// RunType 运行类型枚举。
type RunType string

const (
    RunTypeWorkflow RunType = "workflow"  // 工作流运行
    RunTypeAgent    RunType = "agent"     // Agent 会话运行
    RunTypeToolCall RunType = "tool_call" // 单次工具调用
)

// RunStatus 运行状态枚举。
type RunStatus string

const (
    RunStatusPending   RunStatus = "pending"   // 待执行
    RunStatusRunning   RunStatus = "running"   // 执行中
    RunStatusPaused    RunStatus = "paused"    // 已暂停（等待人工干预或外部信号）
    RunStatusCompleted RunStatus = "completed" // 已完成
    RunStatusFailed    RunStatus = "failed"    // 已失败
    RunStatusCancelled RunStatus = "cancelled" // 已取消
)

// ErrorCategory 错误类别枚举。
// 用于结构化错误分类，驱动自动化恢复策略。
type ErrorCategory string

const (
    ErrCatValidation    ErrorCategory = "VALIDATION"      // 输入校验失败
    ErrCatPolicyDeny    ErrorCategory = "POLICY_DENY"     // 策略拒绝（权限/预算/黑名单）
    ErrCatTransient     ErrorCategory = "TRANSIENT"       // 临时性错误（网络抖动、服务暂不可用）
    ErrCatDependency    ErrorCategory = "DEPENDENCY"      // 依赖错误（上游服务/数据不可用）
    ErrCatTimeout       ErrorCategory = "TIMEOUT"         // 超时
    ErrCatResourceLimit ErrorCategory = "RESOURCE_LIMIT"  // 资源超限（内存/GPU/配额）
    ErrCatToolBug       ErrorCategory = "TOOL_BUG"        // 工具内部缺陷
    ErrCatModelReasoning ErrorCategory = "MODEL_REASONING" // 模型推理异常（幻觉/拒答/格式错误）
)

// Run 运行实体。
// 统一表示工作流运行、Agent 会话运行、单次工具调用。
type Run struct {
    ID            uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    TenantID      uuid.UUID  `gorm:"type:uuid;not null;index"`
    Type          RunType    `gorm:"type:varchar(20);not null;index"`
    ParentRunID   *uuid.UUID `gorm:"type:uuid;index"`
    WorkflowID    *uuid.UUID `gorm:"type:uuid;index"`
    AgentProfile  *string    `gorm:"type:varchar(200)"`
    ToolID        *uuid.UUID `gorm:"type:uuid"`
    Status        RunStatus  `gorm:"type:varchar(20);not null;default:'pending';index"`
    TraceID       string     `gorm:"type:varchar(100);not null;index"`
    Input         JSONMap    `gorm:"type:jsonb"`
    Output        JSONMap    `gorm:"type:jsonb"`
    Error         *RunError  `gorm:"type:jsonb"`
    Budget        RunBudget  `gorm:"type:jsonb"`
    BudgetUsed    RunBudgetUsed `gorm:"type:jsonb"`
    NestingLevel  int        `gorm:"type:int;default:0"`
    ContextState  *ContextState `gorm:"-"`
    StartedAt     *time.Time `gorm:"index"`
    CompletedAt   *time.Time
    CreatedAt     time.Time  `gorm:"not null;index"`
    UpdatedAt     time.Time  `gorm:"not null"`

    // 触发者信息
    TriggeredBy   *uuid.UUID `gorm:"type:uuid"`
    TriggerSource string     `gorm:"type:varchar(50)"` // manual/schedule/event/parent_run
}

func (Run) TableName() string {
    return "runs"
}

// IsTerminal 判断是否为终态。
func (r *Run) IsTerminal() bool {
    return r.Status == RunStatusCompleted ||
        r.Status == RunStatusFailed ||
        r.Status == RunStatusCancelled
}

// IsRunning 判断是否正在执行。
func (r *Run) IsRunning() bool {
    return r.Status == RunStatusRunning
}

// CanPause 判断是否可以暂停。
func (r *Run) CanPause() bool {
    return r.Status == RunStatusRunning
}

// CanCancel 判断是否可以取消。
func (r *Run) CanCancel() bool {
    return r.Status == RunStatusPending || r.Status == RunStatusRunning || r.Status == RunStatusPaused
}

// Start 开始运行。
func (r *Run) Start() {
    now := time.Now()
    r.Status = RunStatusRunning
    r.StartedAt = &now
    r.UpdatedAt = now
}

// Complete 完成运行。
func (r *Run) Complete(output JSONMap) {
    now := time.Now()
    r.Status = RunStatusCompleted
    r.Output = output
    r.CompletedAt = &now
    r.UpdatedAt = now
}

// Fail 标记运行失败。
func (r *Run) Fail(runErr *RunError) {
    now := time.Now()
    r.Status = RunStatusFailed
    r.Error = runErr
    r.CompletedAt = &now
    r.UpdatedAt = now
}

// Cancel 取消运行。
func (r *Run) Cancel() {
    now := time.Now()
    r.Status = RunStatusCancelled
    r.CompletedAt = &now
    r.UpdatedAt = now
}

// Pause 暂停运行。
func (r *Run) Pause() {
    r.Status = RunStatusPaused
    r.UpdatedAt = time.Now()
}

// Resume 恢复运行。
func (r *Run) Resume() {
    r.Status = RunStatusRunning
    r.UpdatedAt = time.Now()
}

// IsBudgetExceeded 判断是否超出预算。
func (r *Run) IsBudgetExceeded() bool {
    if r.Budget.MaxSteps > 0 && r.BudgetUsed.Steps >= r.Budget.MaxSteps {
        return true
    }
    if r.Budget.MaxToolCalls > 0 && r.BudgetUsed.ToolCalls >= r.Budget.MaxToolCalls {
        return true
    }
    if r.Budget.MaxCostUSD > 0 && r.BudgetUsed.CostUSD >= r.Budget.MaxCostUSD {
        return true
    }
    if r.Budget.MaxTokens > 0 && r.BudgetUsed.Tokens >= r.Budget.MaxTokens {
        return true
    }
    return false
}

// RunError 运行错误。
// 结构化错误信息，支持自动化恢复策略匹配。
type RunError struct {
    Category    ErrorCategory `json:"category"`              // 错误类别
    Message     string        `json:"message"`               // 人类可读的错误信息
    NodeID      string        `json:"node_id,omitempty"`     // 出错节点 ID
    ToolCallID  string        `json:"tool_call_id,omitempty"` // 出错工具调用 ID
    Recoverable bool          `json:"recoverable"`           // 是否可恢复
    Details     JSONMap       `json:"details,omitempty"`     // 额外详情
}

// RunBudget 运行预算。
// 用于限制单次运行的资源消耗上限。
type RunBudget struct {
    MaxSteps       int     `json:"max_steps,omitempty"`        // 最大步骤数
    MaxToolCalls   int     `json:"max_tool_calls,omitempty"`   // 最大工具调用次数
    MaxDurationSec int64   `json:"max_duration_sec,omitempty"` // 最大持续时间（秒）
    MaxCostUSD     float64 `json:"max_cost_usd,omitempty"`     // 最大成本（美元）
    MaxTokens      int     `json:"max_tokens,omitempty"`       // 最大 Token 消耗
}

// RunBudgetUsed 运行预算消耗。
type RunBudgetUsed struct {
    Steps       int     `json:"steps"`        // 已执行步骤数
    ToolCalls   int     `json:"tool_calls"`   // 已调用工具次数
    DurationSec int64   `json:"duration_sec"` // 已持续时间（秒）
    CostUSD     float64 `json:"cost_usd"`     // 已消耗成本（美元）
    Tokens      int     `json:"tokens"`       // 已消耗 Token 数
}
```

### 5.2 过滤器

```go
// RunFilter 运行查询过滤条件。
type RunFilter struct {
    TenantID    *uuid.UUID // 租户过滤
    Type        *RunType   // 类型过滤
    Status      *RunStatus // 状态过滤
    WorkflowID  *uuid.UUID // 工作流过滤
    ParentRunID *uuid.UUID // 父运行过滤
    TriggeredBy *uuid.UUID // 触发者过滤
    TraceID     string     // 追踪 ID 过滤
    From        *time.Time // 开始时间起始
    To          *time.Time // 开始时间截止
    Limit       int        // 分页大小
    Offset      int        // 分页偏移
}
```

### 5.3 接口

```go
// RunRepository 运行持久化接口。
type RunRepository interface {
    // Create 创建运行记录。
    Create(ctx context.Context, run *Run) error

    // GetByID 按 ID 获取运行记录。
    GetByID(ctx context.Context, id uuid.UUID) (*Run, error)

    // List 按条件查询运行列表。
    List(ctx context.Context, filter RunFilter) ([]*Run, int64, error)

    // Update 更新运行记录。
    Update(ctx context.Context, run *Run) error

    // UpdateStatus 原子更新运行状态。
    UpdateStatus(ctx context.Context, id uuid.UUID, status RunStatus) error

    // UpdateBudgetUsed 原子更新预算消耗。
    UpdateBudgetUsed(ctx context.Context, id uuid.UUID, used RunBudgetUsed) error

    // ListByParent 查询某运行的所有子运行。
    ListByParent(ctx context.Context, parentRunID uuid.UUID) ([]*Run, error)

    // GetStats 获取运行统计信息。
    GetStats(ctx context.Context, filter RunFilter) (*RunStats, error)
}

// RunStats 运行统计信息。
type RunStats struct {
    Total     int64 `json:"total"`
    Pending   int64 `json:"pending"`
    Running   int64 `json:"running"`
    Paused    int64 `json:"paused"`
    Completed int64 `json:"completed"`
    Failed    int64 `json:"failed"`
    Cancelled int64 `json:"cancelled"`
}
```

---

## 6. RunEvent（运行事件）

运行事件是系统可观测性的基石。每个运行实例产生的所有状态变更、工具调用、上下文修改、错误与恢复动作均以事件形式记录，支持全链路追踪与事后回放。

### 6.1 实体定义

```go
package domain

import (
    "time"

    "github.com/google/uuid"
)

// RunEventType 运行事件类型枚举。
// 约定：
//   - Go 常量名使用 PascalCase（如 EventRunStarted）
//   - 线上协议（JSON/SSE）使用 snake_case（如 run_started）
type RunEventType string

const (
    EventRunStarted           RunEventType = "run_started"
    EventRunFinished          RunEventType = "run_finished"
    EventRunPaused            RunEventType = "run_paused"
    EventRunResumed           RunEventType = "run_resumed"
    EventRunCancelled         RunEventType = "run_cancelled"
    EventRunRetried           RunEventType = "run_retried"
    EventNodeStarted          RunEventType = "node_started"
    EventNodeFinished         RunEventType = "node_finished"
    EventNodeFailed           RunEventType = "node_failed"
    EventNodeSkipped          RunEventType = "node_skipped"
    EventNodeRetry            RunEventType = "node_retry"
    EventSubWorkflowStarted   RunEventType = "sub_workflow_started"
    EventSubWorkflowFinished  RunEventType = "sub_workflow_finished"
    EventToolCalled           RunEventType = "tool_called"
    EventToolSucceeded        RunEventType = "tool_succeeded"
    EventToolFailed           RunEventType = "tool_failed"
    EventToolTimedOut         RunEventType = "tool_timed_out"
    EventToolRetryScheduled   RunEventType = "tool_retry_scheduled"
    EventContextPatchApplied  RunEventType = "context_patch_applied"
    EventContextConflict      RunEventType = "context_conflict"
    EventContextSnapshotCreated RunEventType = "context_snapshot_created"
    EventAgentPlan            RunEventType = "agent_plan"
    EventAgentAct             RunEventType = "agent_act"
    EventAgentObserve         RunEventType = "agent_observe"
    EventAgentRecover         RunEventType = "agent_recover"
    EventAgentEscalation      RunEventType = "agent_escalation"
    EventAgentSessionStarted  RunEventType = "agent_session_started"
    EventAgentSessionFinished RunEventType = "agent_session_finished"
    EventIntentReceived       RunEventType = "intent_received"
    EventIntentParsed         RunEventType = "intent_parsed"
    EventIntentPlanned        RunEventType = "intent_planned"
    EventIntentPlanAdjusted   RunEventType = "intent_plan_adjusted"
    EventIntentConfirmed      RunEventType = "intent_confirmed"
    EventIntentRejected       RunEventType = "intent_rejected"
    EventIntentExecutionStarted RunEventType = "intent_execution_started"
    EventIntentExecutionFinished RunEventType = "intent_execution_finished"
    EventIntentExecutionFailed RunEventType = "intent_execution_failed"
    EventPolicyEvaluated      RunEventType = "policy_evaluated"
    EventPolicyBlocked        RunEventType = "policy_blocked"
    EventApprovalRequested    RunEventType = "approval_requested"
    EventApprovalResolved     RunEventType = "approval_resolved"
    EventAssetCreated         RunEventType = "asset_created"
    EventAssetDerived         RunEventType = "asset_derived"
    EventStreamSliceCreated   RunEventType = "stream_slice_created"
    EventBudgetWarning        RunEventType = "budget_warning"
    EventBudgetExceeded       RunEventType = "budget_exceeded"
)

// EventSource 事件来源层级。
type EventSource string

const (
    EventSourceAccess  EventSource = "access"
    EventSourceControl EventSource = "control"
    EventSourceRuntime EventSource = "runtime"
    EventSourcePolicy  EventSource = "policy"
    EventSourceData    EventSource = "data"
)

// EventError 标准事件错误结构。
type EventError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Type    string `json:"type,omitempty"` // validation|policy|runtime|dependency|timeout
}

// RunEvent 运行事件实体。
// 采用 append-only 模式存储，不可修改。
type RunEvent struct {
    ID          uuid.UUID    `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    TraceID     string       `gorm:"type:varchar(100);not null;index"`
    RunID       uuid.UUID    `gorm:"type:uuid;not null;index"`
    Seq         int64        `gorm:"type:bigint;not null;index"` // 单 Run 内单调递增序号
    ParentRunID *uuid.UUID   `gorm:"type:uuid;index"`
    NodeID      string       `gorm:"type:varchar(200)"`
    StepID      string       `gorm:"type:varchar(200)"`
    ToolCallID  string       `gorm:"type:varchar(200)"`
    Type        RunEventType `gorm:"type:varchar(50);not null;index"`
    Status      string       `gorm:"type:varchar(32);not null;default:''"`
    Timestamp   time.Time    `gorm:"not null;index"`
    Source      EventSource  `gorm:"type:varchar(32);not null;index"`
    Payload     JSONMap      `gorm:"type:jsonb;not null;default:'{}'"`
    Error       *EventError  `gorm:"type:jsonb"`
    TenantID    uuid.UUID    `gorm:"type:uuid;not null;index"`
    ActorID     *uuid.UUID   `gorm:"type:uuid;index"`
}

func (RunEvent) TableName() string {
    return "run_events"
}
```

### 6.2 事件数据结构约定

各事件类型对应的 `Payload` 字段结构：

| 事件类别 | 事件类型（snake_case） | Payload 关键字段 |
|---------|------------------------|------------------|
| Run 生命周期 | `run_started`,`run_finished`,`run_paused`,`run_resumed`,`run_cancelled`,`run_retried` | `status`,`reason`,`duration_ms`,`from_node`,`budget_used` |
| Workflow 节点 | `node_started`,`node_finished`,`node_failed`,`node_skipped`,`node_retry`,`sub_workflow_started`,`sub_workflow_finished` | `node_id`,`child_run_id`,`inputs`,`outputs`,`error`,`attempt`,`delay_ms`,`condition_result` |
| Tool 调用 | `tool_called`,`tool_succeeded`,`tool_failed`,`tool_timed_out`,`tool_retry_scheduled` | `tool_id`,`tool_code`,`input`,`output_summary`,`diagnostics`,`error`,`attempt` |
| Context 变更 | `context_patch_applied`,`context_conflict`,`context_snapshot_created` | `before_version`,`after_version`,`operations`,`conflict_keys`,`strategy`,`snapshot_version` |
| Agent 决策 | `agent_plan`,`agent_act`,`agent_observe`,`agent_recover`,`agent_escalation`,`agent_session_started`,`agent_session_finished` | `step_id`,`goal`,`action`,`observation`,`recovery_action`,`approval_ticket`,`output` |
| Intent 编排 | `intent_received`,`intent_parsed`,`intent_planned`,`intent_plan_adjusted`,`intent_confirmed`,`intent_rejected`,`intent_execution_started`,`intent_execution_finished`,`intent_execution_failed` | `intent_id`,`source_type`,`goal`,`action_count`,`requires_confirmation`,`linked_run_id`,`error`,`clarification_questions` |
| Policy 与审批 | `policy_evaluated`,`policy_blocked`,`approval_requested`,`approval_resolved` | `decision`,`reason_code`,`violations`,`ticket_id`,`approval_status`,`comment` |
| Asset 链路 | `asset_created`,`asset_derived`,`stream_slice_created` | `asset_id`,`parent_id`,`asset_type`,`slice_id`,`start_at`,`end_at` |
| Budget 治理 | `budget_warning`,`budget_exceeded` | `metric`,`current`,`limit`,`percentage` |

### 6.3 查询选项

```go
// EventQueryOpts 事件查询选项。
type EventQueryOpts struct {
    Types      []RunEventType // 按事件类型过滤
    NodeID     string         // 按节点 ID 过滤
    ToolCallID string         // 按工具调用 ID 过滤
    FromSeq    *int64         // 按序号起点过滤（用于补拉）
    From       *time.Time     // 时间范围起始
    To         *time.Time     // 时间范围截止
    Limit      int            // 分页大小
    Offset     int            // 分页偏移
    OrderAsc   bool           // 是否按时间正序（默认倒序）
}
```

### 6.4 接口

```go
// RunEventStore 运行事件存储接口。
// 采用 append-only 模式，不支持修改和删除。
type RunEventStore interface {
    // Append 追加一条运行事件。
    Append(ctx context.Context, event RunEvent) error

    // AppendBatch 批量追加运行事件。
    AppendBatch(ctx context.Context, events []RunEvent) error

    // GetEvents 按条件查询运行事件。
    GetEvents(ctx context.Context, runID uuid.UUID, opts EventQueryOpts) ([]RunEvent, int64, error)

    // Subscribe 订阅指定运行的实时事件流。
    // 返回事件通道，调用方通过 context 取消订阅。
    Subscribe(ctx context.Context, runID uuid.UUID) (<-chan RunEvent, error)

    // CountByType 按事件类型统计数量。
    CountByType(ctx context.Context, runID uuid.UUID) (map[RunEventType]int64, error)
}
```

---

## 7. Workflow（工作流）

工作流定义了 DAG（有向无环图）编排逻辑。`WorkflowDefinition` 内聚所有版本化信息，节点类型支持 tool、algorithm、agent 和 sub_workflow 四种。

### 7.1 实体定义

```go
package domain

import (
    "time"

    "github.com/google/uuid"
)

// WorkflowStatus 工作流状态枚举。
type WorkflowStatus string

const (
    WorkflowStatusDraft    WorkflowStatus = "draft"    // 草稿（编辑中）
    WorkflowStatusActive   WorkflowStatus = "active"   // 活跃（可被触发执行）
    WorkflowStatusDisabled WorkflowStatus = "disabled" // 已禁用（暂停触发）
    WorkflowStatusArchived WorkflowStatus = "archived" // 已归档（不可编辑、不可触发）
)

// NodeType 节点类型枚举。
type NodeType string

const (
    NodeTypeTool        NodeType = "tool"         // 工具节点：调用单个 ToolSpec
    NodeTypeAlgorithm   NodeType = "algorithm"    // 算法节点：引用 Algorithm，运行时解析实现
    NodeTypeAgent       NodeType = "agent"        // Agent 节点：启动一个 Agent Run Loop
    NodeTypeSubWorkflow NodeType = "sub_workflow"  // 子工作流节点：嵌套执行另一个工作流
)

// WorkflowDefinition 工作流定义实体。
// 包含完整的 DAG 结构、触发配置、上下文规范与执行策略。
type WorkflowDefinition struct {
    ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    TenantID    uuid.UUID      `gorm:"type:uuid;not null;index"`
    OwnerID     uuid.UUID      `gorm:"type:uuid;not null"`
    Name        string         `gorm:"type:varchar(500);not null"`
    Code        string         `gorm:"type:varchar(200);not null;uniqueIndex"`
    Description string         `gorm:"type:text"`
    Version     string         `gorm:"type:varchar(50);not null"`
    Inputs      JSONSchema     `gorm:"type:jsonb"`
    Outputs     OutputMapping  `gorm:"type:jsonb"`
    ContextSpec ContextSpec    `gorm:"type:jsonb"`
    Nodes       NodeDefArray   `gorm:"type:jsonb"`
    Edges       EdgeDefArray   `gorm:"type:jsonb"`
    Policy      WorkflowPolicy `gorm:"type:jsonb"`
    Triggers    TriggerArray   `gorm:"type:jsonb"`
    Status      WorkflowStatus `gorm:"type:varchar(20);not null;default:'draft';index"`
    Tags        StringArray    `gorm:"type:text[]"`
    CreatedAt   time.Time      `gorm:"not null;index"`
    UpdatedAt   time.Time      `gorm:"not null"`
}

// NodeDefArray 节点定义数组类型。
type NodeDefArray []NodeDefinition

// EdgeDefArray 边定义数组类型。
type EdgeDefArray []EdgeDefinition

// TriggerArray 触发配置数组类型。
type TriggerArray []TriggerConfig

func (WorkflowDefinition) TableName() string {
    return "workflow_definitions"
}

// IsActive 判断工作流是否处于活跃状态。
func (w *WorkflowDefinition) IsActive() bool {
    return w.Status == WorkflowStatusActive
}

// CanTrigger 判断工作流是否可被触发。
func (w *WorkflowDefinition) CanTrigger() bool {
    return w.Status == WorkflowStatusActive
}

// Activate 激活工作流。
func (w *WorkflowDefinition) Activate() error {
    if len(w.Nodes) == 0 {
        return ErrWorkflowNoNodes
    }
    w.Status = WorkflowStatusActive
    w.UpdatedAt = time.Now()
    return nil
}

// Disable 禁用工作流。
func (w *WorkflowDefinition) Disable() {
    w.Status = WorkflowStatusDisabled
    w.UpdatedAt = time.Now()
}

// Archive 归档工作流。
func (w *WorkflowDefinition) Archive() {
    w.Status = WorkflowStatusArchived
    w.UpdatedAt = time.Now()
}

// NodeDefinition DAG 节点定义。
type NodeDefinition struct {
    ID             string         `json:"id"`                            // 节点唯一标识（工作流内唯一）
    Name           string         `json:"name"`                          // 节点名称
    Type           NodeType       `json:"type"`                          // 节点类型
    ToolID         *uuid.UUID     `json:"tool_id,omitempty"`             // 工具 ID（Type=tool 时）
    AlgorithmRef   *AlgorithmRef  `json:"algorithm_ref,omitempty"`       // 算法引用（Type=algorithm 时）
    AgentProfileID *string        `json:"agent_profile_id,omitempty"`    // Agent 配置 ID（Type=agent 时）
    SubWorkflowID  *uuid.UUID     `json:"sub_workflow_id,omitempty"`     // 子工作流 ID（Type=sub_workflow 时）
    InputMapping   MappingRules   `json:"input_mapping,omitempty"`       // 输入映射规则
    OutputMapping  MappingRules   `json:"output_mapping,omitempty"`      // 输出映射规则
    RetryPolicy    RetryPolicy    `json:"retry_policy,omitempty"`        // 重试策略
    TimeoutMs      int64          `json:"timeout_ms,omitempty"`          // 超时时间（毫秒）
    WritePolicy    *WritePolicy   `json:"write_policy,omitempty"`        // 共享上下文写入策略
    Condition      *string        `json:"condition,omitempty"`           // 条件表达式（空表示无条件执行）
    Position       *NodePosition  `json:"position,omitempty"`            // 前端布局位置
}

// AlgorithmRef 算法引用。
// 工作流节点通过此结构引用算法库中的算法版本。
type AlgorithmRef struct {
    AlgorithmID   *uuid.UUID `json:"algorithm_id,omitempty"`    // 算法 ID
    AlgorithmCode string     `json:"algorithm_code,omitempty"`  // 算法编码（与 ID 二选一）
    Version       string     `json:"version,omitempty"`         // 指定版本（空表示使用最新 published 版本）
    Strategy      string     `json:"strategy,omitempty"`        // 实现选择策略：default/high_accuracy/low_cost/low_latency/balanced
    ParamOverrides JSONMap   `json:"param_overrides,omitempty"` // 参数覆盖
}

// EdgeDefinition DAG 边定义。
type EdgeDefinition struct {
    ID        string  `json:"id"`                   // 边唯一标识
    FromNode  string  `json:"from_node"`            // 源节点 ID
    ToNode    string  `json:"to_node"`              // 目标节点 ID
    Condition *string `json:"condition,omitempty"`   // 条件表达式（空表示无条件流转）
}

// NodePosition 节点前端布局位置。
type NodePosition struct {
    X float64 `json:"x"`
    Y float64 `json:"y"`
}

// WorkflowPolicy 工作流执行策略。
type WorkflowPolicy struct {
    ToolWhitelist    []string       `json:"tool_whitelist,omitempty"`    // 允许使用的工具编码列表
    BudgetLimit      RunBudget      `json:"budget_limit,omitempty"`     // 预算上限
    NetworkWhitelist []string       `json:"network_whitelist,omitempty"` // 允许访问的域名列表
    DataScope        DataAccessSpec `json:"data_scope,omitempty"`       // 数据访问域限制
    MaxNestingLevel  int            `json:"max_nesting_level,omitempty"` // 最大嵌套层级（默认 2）
    MaxParallelNodes int            `json:"max_parallel_nodes,omitempty"` // 最大并行节点数
}
```

### 7.2 过滤器

```go
// WorkflowFilter 工作流查询过滤条件。
type WorkflowFilter struct {
    TenantID *uuid.UUID      // 租户过滤
    OwnerID  *uuid.UUID      // 所有者过滤
    Status   *WorkflowStatus // 状态过滤
    Tags     []string        // 标签过滤
    Keyword  string          // 名称/描述关键字搜索
    Limit    int             // 分页大小
    Offset   int             // 分页偏移
}
```

### 7.3 接口

```go
// WorkflowRepository 工作流持久化接口。
type WorkflowRepository interface {
    Create(ctx context.Context, wf *WorkflowDefinition) error
    GetByID(ctx context.Context, id uuid.UUID) (*WorkflowDefinition, error)
    GetByCode(ctx context.Context, code string) (*WorkflowDefinition, error)
    List(ctx context.Context, filter WorkflowFilter) ([]*WorkflowDefinition, int64, error)
    Update(ctx context.Context, wf *WorkflowDefinition) error
    Delete(ctx context.Context, id uuid.UUID) error
    ListActive(ctx context.Context) ([]*WorkflowDefinition, error)
    ListByTriggerType(ctx context.Context, triggerType TriggerType) ([]*WorkflowDefinition, error)
}

// WorkflowEngine DAG 工作流执行引擎接口。
type WorkflowEngine interface {
    // Execute 执行工作流。
    // 解析 DAG 拓扑，按依赖顺序执行节点，管理上下文读写。
    Execute(ctx context.Context, wf *WorkflowDefinition, run *Run) error

    // Cancel 取消正在执行的工作流运行。
    Cancel(ctx context.Context, runID uuid.UUID) error

    // Pause 暂停正在执行的工作流运行。
    Pause(ctx context.Context, runID uuid.UUID) error

    // Resume 恢复暂停的工作流运行。
    Resume(ctx context.Context, runID uuid.UUID) error

    // ValidateDAG 校验 DAG 结构有效性（无环、连通性等）。
    ValidateDAG(ctx context.Context, nodes []NodeDefinition, edges []EdgeDefinition) error
}
```

---

## 8. Agent（智能体）

Agent 是系统的自主执行实体，遵循 Plan-Act-Observe-Reflect-Finish 循环。Goyais 将 Agent 工程化，赋予可描述、可控、可观测、可限制的完整能力。AgentProfile 定义 Agent 的行为契约（静态），AgentState 表示运行时状态（动态）。

### 8.1 实体定义

```go
package domain

import (
    "time"
)

// AgentPhase Agent 运行阶段枚举。
type AgentPhase string

const (
    AgentPhasePlan    AgentPhase = "plan"    // 规划阶段：分析目标、制定计划
    AgentPhaseAct     AgentPhase = "act"     // 执行阶段：调用工具
    AgentPhaseObserve AgentPhase = "observe" // 观察阶段：分析工具返回结果
    AgentPhaseReflect AgentPhase = "reflect" // 反思阶段：评估进展、调整计划
    AgentPhaseFinish  AgentPhase = "finish"  // 完成阶段：汇总输出
)

// AgentProfile Agent 配置实体。
// 定义 Agent 的行为契约、工具权限、预算与恢复策略。
// 属于定义态，创建后不应在运行中修改。
type AgentProfile struct {
    ID              string            `gorm:"type:varchar(200);primary_key"` // 唯一标识符
    TenantID        string            `gorm:"type:varchar(100);not null;index"`
    Name            string            `gorm:"type:varchar(500);not null"`
    Description     string            `gorm:"type:text"`
    SystemPrompt    string            `gorm:"type:text;not null"`            // 系统指令模板
    ModelRef        string            `gorm:"type:varchar(200);not null"`    // 固定模型引用（从 Model Registry 获取）
    ToolScope       ToolScope         `gorm:"type:jsonb"`                    // 工具权限范围
    Budget          RunBudget         `gorm:"type:jsonb"`                    // 预算配置
    MemoryConfig    MemoryConfig      `gorm:"type:jsonb"`                    // 记忆配置
    FailurePolicy   FailurePolicyMap  `gorm:"type:jsonb"`                    // 错误恢复策略映射
    OutputSchema    JSONSchema        `gorm:"type:jsonb"`                    // 强制结构化输出 Schema
    MaxNestingLevel int               `gorm:"type:int;default:2"`            // 最大嵌套层级
    CreatedAt       time.Time         `gorm:"not null"`
    UpdatedAt       time.Time         `gorm:"not null"`
}

// FailurePolicyMap 错误恢复策略映射类型。
type FailurePolicyMap map[ErrorCategory]FailureAction

func (AgentProfile) TableName() string {
    return "agent_profiles"
}

// ToolScope 工具权限范围。
// Allowed 和 Denied 互斥使用：
//   - 仅设置 Allowed：白名单模式，只允许使用列出的工具
//   - 仅设置 Denied：黑名单模式，禁止使用列出的工具
//   - 两者均空：无工具限制
type ToolScope struct {
    Allowed []string `json:"allowed,omitempty"` // 允许的工具编码列表
    Denied  []string `json:"denied,omitempty"`  // 禁止的工具编码列表
}

// IsToolAllowed 判断指定工具是否被允许。
func (ts ToolScope) IsToolAllowed(toolCode string) bool {
    if len(ts.Denied) > 0 {
        for _, d := range ts.Denied {
            if d == toolCode {
                return false
            }
        }
    }
    if len(ts.Allowed) > 0 {
        for _, a := range ts.Allowed {
            if a == toolCode {
                return true
            }
        }
        return false
    }
    return true
}

// MemoryConfig Agent 记忆配置。
type MemoryConfig struct {
    TokenBudget int    `json:"token_budget"` // 记忆 Token 预算（如 2048）
    Strategy    string `json:"strategy"`     // 记忆策略：summary_append/sliding_window/key_facts
}

// FailureAction 错误恢复动作。
type FailureAction struct {
    Action      string `json:"action"`       // retry/fallback/replan/escalate
    MaxAttempts int    `json:"max_attempts"` // 最大尝试次数
}

// AgentState Agent 运行时状态。
// 属于运行态，跟随 Run 的生命周期。
type AgentState struct {
    Phase      AgentPhase   `json:"phase"`                // 当前阶段
    StepCount  int          `json:"step_count"`           // 已执行步骤数
    Plan       *AgentPlan   `json:"plan,omitempty"`       // 当前计划
    Memory     JSONMap      `json:"memory,omitempty"`     // 累积记忆（在 Token 预算内）
    LastAction *AgentAction `json:"last_action,omitempty"` // 最近一次动作
    LastResult *ToolResult  `json:"last_result,omitempty"` // 最近一次工具调用结果
}

// AgentPlan Agent 执行计划。
type AgentPlan struct {
    Goal      string     `json:"goal"`      // 目标描述
    Steps     []PlanStep `json:"steps"`     // 计划步骤
    Reasoning string     `json:"reasoning"` // 推理过程
}

// PlanStep 计划步骤。
type PlanStep struct {
    Index       int    `json:"index"`                  // 步骤序号
    Description string `json:"description"`            // 步骤描述
    ToolCode    string `json:"tool_code,omitempty"`    // 预计使用的工具编码
    Status      string `json:"status"`                 // pending/running/completed/skipped
}

// AgentAction Agent 动作。
type AgentAction struct {
    Type       string `json:"type"`                 // tool_call/plan_update/finish
    ToolCode   string `json:"tool_code,omitempty"`  // 工具编码
    ToolCallID string `json:"tool_call_id,omitempty"` // 工具调用 ID
    Input      JSONMap `json:"input,omitempty"`      // 工具输入
    Reasoning  string `json:"reasoning,omitempty"`  // 动作理由
}
```

### 8.2 接口

```go
// AgentRuntime Agent 运行时接口。
// 负责驱动 Agent 的 Plan-Act-Observe-Reflect-Finish 循环。
type AgentRuntime interface {
    // Start 启动 Agent 运行。
    // 创建新的 Run（type=agent），初始化 AgentState。
    Start(ctx context.Context, profile *AgentProfile, input JSONMap, parentRunID *uuid.UUID) (*Run, error)

    // Step 执行 Agent 的下一步。
    // 根据当前 AgentState.Phase 决定执行动作：
    //   plan    → 调用模型生成计划
    //   act     → 调用工具
    //   observe → 分析工具结果
    //   reflect → 评估进展
    //   finish  → 汇总输出
    Step(ctx context.Context, runID uuid.UUID) (*AgentState, error)

    // Stop 停止 Agent 运行。
    Stop(ctx context.Context, runID uuid.UUID) error

    // GetState 获取 Agent 当前状态。
    GetState(ctx context.Context, runID uuid.UUID) (*AgentState, error)
}
```

---

## 9. Algorithm（算法）

算法是系统的意图层抽象。区别于 Tool（能力原子），Algorithm 代表一个完整的"问题域 + 解法描述"，可以绑定多个实现（ImplementationBinding），支持按策略选择最优实现。通过 EvaluationProfile 提供评测证据，形成可治理的算法资产。

### 9.1 实体定义

```go
package domain

import (
    "time"

    "github.com/google/uuid"
)

// AlgorithmLifecycle 算法版本生命周期枚举。
type AlgorithmLifecycle string

const (
    AlgoLifecycleDraft     AlgorithmLifecycle = "draft"     // 草稿
    AlgoLifecycleTested    AlgorithmLifecycle = "tested"    // 已测试
    AlgoLifecyclePublished AlgorithmLifecycle = "published" // 已发布（依赖冻结）
    AlgoLifecycleDeprecated AlgorithmLifecycle = "deprecated" // 已弃用
)

// Algorithm 算法实体。
// 代表一个完整的算法意图（场景 + 问题 + 边界），
// 通过版本化管理实现演进与兼容。
type Algorithm struct {
    ID          uuid.UUID  `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    TenantID    uuid.UUID  `gorm:"type:uuid;not null;index"`
    OwnerID     uuid.UUID  `gorm:"type:uuid;not null"`
    Name        string     `gorm:"type:varchar(500);not null"`
    Code        string     `gorm:"type:varchar(200);not null;uniqueIndex"`
    Description string     `gorm:"type:text"`
    Scene       string     `gorm:"type:text"`                             // 场景描述
    Problem     string     `gorm:"type:text"`                             // 问题定义
    Boundary    string     `gorm:"type:text"`                             // 边界说明
    Category    string     `gorm:"type:varchar(100);not null;index"`      // detection/classification/segmentation/generation/transform/analysis
    Tags        StringArray `gorm:"type:text[]"`
    CreatedAt   time.Time  `gorm:"not null;index"`
    UpdatedAt   time.Time  `gorm:"not null"`

    // 关联（非数据库列）
    Versions []AlgorithmVersion `gorm:"-"`
}

func (Algorithm) TableName() string {
    return "algorithms"
}

// AlgorithmVersion 算法版本实体。
// 每个版本冻结了输入输出 Schema 与资源画像。
// 当 Status=published 时，版本及其绑定的 ImplementationBinding 不可再修改。
type AlgorithmVersion struct {
    ID              uuid.UUID          `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    AlgorithmID     uuid.UUID          `gorm:"type:uuid;not null;index"`
    Version         string             `gorm:"type:varchar(50);not null"`
    InputSchema     JSONSchema         `gorm:"type:jsonb"`
    OutputSchema    JSONSchema         `gorm:"type:jsonb"`
    DefaultParams   JSONMap            `gorm:"type:jsonb"`
    ResourceProfile ResourceProfile    `gorm:"type:jsonb"`
    Status          AlgorithmLifecycle `gorm:"type:varchar(20);not null;default:'draft'"`
    PublishedAt     *time.Time
    CreatedAt       time.Time          `gorm:"not null"`
    UpdatedAt       time.Time          `gorm:"not null"`

    // 关联（非数据库列）
    Implementations []ImplementationBinding `gorm:"-"`
    Evaluations     []EvaluationProfile     `gorm:"-"`
}

func (AlgorithmVersion) TableName() string {
    return "algorithm_versions"
}

// IsPublished 判断版本是否已发布。
func (v *AlgorithmVersion) IsPublished() bool {
    return v.Status == AlgoLifecyclePublished
}

// CanModify 判断版本是否可修改。
// 已发布的版本不可修改（依赖冻结原则）。
func (v *AlgorithmVersion) CanModify() bool {
    return v.Status == AlgoLifecycleDraft || v.Status == AlgoLifecycleTested
}

// ImplementationBinding 算法实现绑定。
// 将算法版本与具体的 Tool 关联，支持多实现策略选择。
type ImplementationBinding struct {
    ID                 uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    AlgorithmVersionID uuid.UUID `gorm:"type:uuid;not null;index"`
    ToolID             uuid.UUID `gorm:"type:uuid;not null"`
    Priority           int       `gorm:"type:int;default:0"`
    Strategy           string    `gorm:"type:varchar(50)"`             // default/high_accuracy/low_cost/low_latency/balanced
    ParamOverrides     JSONMap   `gorm:"type:jsonb"`
    Status             string    `gorm:"type:varchar(20);default:'active'"` // active/disabled
    CreatedAt          time.Time `gorm:"not null"`
}

func (ImplementationBinding) TableName() string {
    return "implementation_bindings"
}

// EvaluationProfile 算法评测档案。
// 记录算法版本的评测结果，为实现选择提供数据依据。
// 评测本身作为一次特殊的 Workflow Run 执行。
type EvaluationProfile struct {
    ID                 uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    AlgorithmVersionID uuid.UUID `gorm:"type:uuid;not null;index"`
    RunID              uuid.UUID `gorm:"type:uuid;not null"`
    Metrics            JSONMap   `gorm:"type:jsonb"`
    InputDatasetRef    string    `gorm:"type:varchar(500)"`
    Status             string    `gorm:"type:varchar(20);default:'pending'"` // pending/running/completed/failed
    CompletedAt        *time.Time
    CreatedAt          time.Time `gorm:"not null"`
}

func (EvaluationProfile) TableName() string {
    return "evaluation_profiles"
}
```

### 9.2 过滤器

```go
// AlgorithmFilter 算法查询过滤条件。
type AlgorithmFilter struct {
    TenantID *uuid.UUID // 租户过滤
    Category string     // 类别过滤
    Tags     []string   // 标签过滤
    Keyword  string     // 名称/描述关键字搜索
    Limit    int        // 分页大小
    Offset   int        // 分页偏移
}
```

### 9.3 关键业务规则

1. **依赖冻结**：当 `AlgorithmVersion.Status = published` 时，该版本的 InputSchema、OutputSchema、DefaultParams 以及所有关联的 `ImplementationBinding` 均不可修改。
2. **多实现选择**：同一个 AlgorithmVersion 可绑定多个 ImplementationBinding，运行时由 `Strategy` 字段决定选择策略：
   - `default`：优先选择版本默认绑定
   - `high_accuracy`：选择评测指标中准确率最高的实现
   - `low_cost`：选择 CostHint 成本最低的实现
   - `low_latency`：选择延迟最低的实现
   - `balanced`：在精度/成本/延迟之间综合平衡
3. **评测即运行**：EvaluationProfile 的评测过程本身作为一个特殊的 Workflow Run 执行，结果存储为 Artifact 并关联到 EvaluationProfile。
4. **Registry 提供候选，Control 层决策**：AlgorithmRepository 负责返回候选实现列表和默认策略；工作流引擎（Control 层）可在 `NodeDefinition.AlgorithmRef` 中覆盖策略。

### 9.4 接口

```go
// AlgorithmRepository 算法持久化接口。
type AlgorithmRepository interface {
    // Algorithm CRUD
    Create(ctx context.Context, algo *Algorithm) error
    GetByID(ctx context.Context, id uuid.UUID) (*Algorithm, error)
    GetByCode(ctx context.Context, code string) (*Algorithm, error)
    List(ctx context.Context, filter AlgorithmFilter) ([]*Algorithm, int64, error)
    Update(ctx context.Context, algo *Algorithm) error
    Delete(ctx context.Context, id uuid.UUID) error

    // AlgorithmVersion 管理
    CreateVersion(ctx context.Context, version *AlgorithmVersion) error
    GetVersionByID(ctx context.Context, id uuid.UUID) (*AlgorithmVersion, error)
    ListVersions(ctx context.Context, algorithmID uuid.UUID) ([]*AlgorithmVersion, error)
    UpdateVersion(ctx context.Context, version *AlgorithmVersion) error
    GetLatestPublishedVersion(ctx context.Context, algorithmID uuid.UUID) (*AlgorithmVersion, error)

    // ImplementationBinding 管理
    CreateBinding(ctx context.Context, binding *ImplementationBinding) error
    ListBindings(ctx context.Context, versionID uuid.UUID) ([]*ImplementationBinding, error)
    UpdateBinding(ctx context.Context, binding *ImplementationBinding) error
    DeleteBinding(ctx context.Context, id uuid.UUID) error
    ResolveImplementation(ctx context.Context, versionID uuid.UUID, strategy string) (*ImplementationBinding, error)

    // EvaluationProfile 管理
    CreateEvaluation(ctx context.Context, eval *EvaluationProfile) error
    GetEvaluation(ctx context.Context, id uuid.UUID) (*EvaluationProfile, error)
    ListEvaluations(ctx context.Context, versionID uuid.UUID) ([]*EvaluationProfile, error)
    UpdateEvaluation(ctx context.Context, eval *EvaluationProfile) error
}
```

---

## 10. Context（上下文）

上下文是工作流与 Agent 运行时的共享状态容器。系统严格区分定义态（ContextSpec）和运行态（ContextState），通过 RFC 6902 JSON Patch 实现可追溯的状态变更，支持 CAS 并发控制与回放。

> 上下文系统的详细设计见 `03-context-system.md`。

### 10.1 定义态

```go
package domain

// ContextSpec 上下文规范（定义态）。
// 属于 WorkflowDefinition / AgentProfile，版本化且不可变。
// 声明所有变量的类型、默认值、大小限制，以及共享键的冲突策略。
type ContextSpec struct {
    Vars       map[string]VarSpec       `json:"vars,omitempty"`        // 声明变量
    SharedKeys map[string]SharedKeySpec `json:"shared_keys,omitempty"` // 共享键声明
}

// VarSpec 变量规范。
type VarSpec struct {
    Type        string `json:"type"`                    // string/number/boolean/object/array
    Default     any    `json:"default,omitempty"`       // 默认值
    Description string `json:"description,omitempty"`   // 变量描述
    MaxSize     int    `json:"max_size,omitempty"`      // 最大大小（字节）
}

// SharedKeySpec 共享键规范。
// 共享键需要在 ContextSpec 中声明才能被节点写入。
type SharedKeySpec struct {
    ConflictStrategy string `json:"conflict_strategy"` // reject/overwrite/merge/append
    MaxSize          int    `json:"max_size,omitempty"` // 最大大小（字节）
    Description      string `json:"description,omitempty"`
}
```

### 10.2 运行态

```go
// ContextState 上下文状态（运行态）。
// 属于 Run，是工作流/Agent 执行过程中的实时状态快照。
type ContextState struct {
    Meta      ContextMeta            `json:"meta"`                // 元信息
    Vars      map[string]any         `json:"vars"`                // 结构化变量
    Assets    map[string]AssetRef    `json:"assets"`              // 资产引用索引
    Artifacts map[string]ArtifactRef `json:"artifacts"`           // 产物引用索引
    Nodes     map[string]NodeState   `json:"nodes"`               // 节点隔离状态空间
    Version   int64                  `json:"version"`             // 递增版本号
}

// NewContextState 创建初始上下文状态。
func NewContextState(spec ContextSpec, meta ContextMeta) *ContextState {
    state := &ContextState{
        Meta:      meta,
        Vars:      make(map[string]any),
        Assets:    make(map[string]AssetRef),
        Artifacts: make(map[string]ArtifactRef),
        Nodes:     make(map[string]NodeState),
        Version:   1,
    }
    // 使用 ContextSpec 中的默认值初始化变量
    for key, spec := range spec.Vars {
        if spec.Default != nil {
            state.Vars[key] = spec.Default
        }
    }
    return state
}
```

### 10.3 变更记录

```go
// ContextPatch 上下文变更补丁。
// 记录每次上下文写入的 RFC 6902 JSON Patch 操作。
// append-only 存储，用于审计追溯与回放。
type ContextPatch struct {
    ID         uuid.UUID        `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    RunID      uuid.UUID        `gorm:"type:uuid;not null;index"`
    NodeID     string           `gorm:"type:varchar(200);not null"`
    Version    int64            `gorm:"type:bigint;not null"`
    Operations PatchOpArray     `gorm:"type:jsonb;not null"`
    Timestamp  time.Time        `gorm:"not null;index"`
}

// PatchOpArray Patch 操作数组类型。
type PatchOpArray []PatchOperation

func (ContextPatch) TableName() string {
    return "context_patches"
}

// PatchOperation RFC 6902 JSON Patch 操作。
type PatchOperation struct {
    Op    string `json:"op"`              // add/remove/replace/move/copy/test
    Path  string `json:"path"`            // JSON Pointer 路径
    Value any    `json:"value,omitempty"` // 操作值
    From  string `json:"from,omitempty"`  // move/copy 的源路径
}

// ContextSnapshot 上下文快照。
// 定期对 ContextState 进行全量快照，加速回放。
type ContextSnapshot struct {
    ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    RunID     uuid.UUID `gorm:"type:uuid;not null;index"`
    Version   int64     `gorm:"type:bigint;not null"`
    State     JSONMap   `gorm:"type:jsonb;not null"`
    CreatedAt time.Time `gorm:"not null"`
}

func (ContextSnapshot) TableName() string {
    return "context_snapshots"
}
```

### 10.4 接口

```go
// ContextRepository 上下文持久化接口。
type ContextRepository interface {
    // State 管理
    InitializeState(ctx context.Context, runID uuid.UUID, state *ContextState) error
    GetState(ctx context.Context, runID uuid.UUID) (*ContextState, error)
    UpdateState(ctx context.Context, runID uuid.UUID, state *ContextState) error

    // Patch 管理
    AppendPatch(ctx context.Context, patch *ContextPatch) error
    ListPatches(ctx context.Context, runID uuid.UUID, fromVersion int64, limit int) ([]*ContextPatch, error)
    GetPatchCount(ctx context.Context, runID uuid.UUID) (int64, error)

    // Snapshot 管理
    CreateSnapshot(ctx context.Context, snapshot *ContextSnapshot) error
    GetLatestSnapshot(ctx context.Context, runID uuid.UUID) (*ContextSnapshot, error)
    GetSnapshotByVersion(ctx context.Context, runID uuid.UUID, version int64) (*ContextSnapshot, error)
}
```

---

## 11. ExecutionEnvelope（执行信封）

执行信封是工具调用的统一传输协议。它封装了工具规范、输入数据、上下文引用与安全策略，确保每次工具调用都具有完整的执行上下文与安全边界。

### 11.1 实体定义

```go
package domain

import (
    "github.com/google/uuid"
)

// ExecutionEnvelope 执行信封。
// 封装一次工具调用的所有必要信息。
// ToolExecutor 接收信封并执行实际调用。
type ExecutionEnvelope struct {
    TraceID     string          `json:"trace_id"`               // 追踪 ID
    RunID       uuid.UUID       `json:"run_id"`                 // 所属运行 ID
    NodeID      string          `json:"node_id"`                // 所属节点 ID
    ToolCallID  string          `json:"tool_call_id"`           // 工具调用唯一 ID
    ToolSpec    *ToolSpec       `json:"tool_spec"`              // 工具规范
    Input       JSONMap         `json:"input"`                  // 工具输入（已映射、已校验）
    ContextRefs []ContextRef    `json:"context_refs,omitempty"` // 上下文引用（仅引用，不含全量）
    Policy      ExecutionPolicy `json:"policy"`                 // 执行策略
}

// ExecutionPolicy 执行策略。
// 控制单次工具调用的安全边界与资源限制。
type ExecutionPolicy struct {
    NetworkWhitelist []string       `json:"network_whitelist,omitempty"` // 允许访问的域名
    ResourceLimits   ResourceLimits `json:"resource_limits,omitempty"`   // 资源硬限制
    PermissionToken  string         `json:"permission_token,omitempty"`  // 短时权限令牌
    TimeoutMs        int64          `json:"timeout_ms,omitempty"`        // 超时时间（毫秒）
    DataScope        DataAccessSpec `json:"data_scope,omitempty"`        // 数据访问域限制
}

// ToolResult 工具执行结果。
type ToolResult struct {
    Success     bool             `json:"success"`                  // 是否成功
    Output      JSONMap          `json:"output,omitempty"`         // 输出数据
    Artifacts   []ArtifactOutput `json:"artifacts,omitempty"`      // 产物列表
    Diagnostics Diagnostics      `json:"diagnostics,omitempty"`    // 诊断信息
    Error       *ToolError       `json:"error,omitempty"`          // 错误信息（失败时）
}

// ToolError 工具错误。
type ToolError struct {
    Category     ErrorCategory `json:"category"`                  // 错误类别
    Message      string        `json:"message"`                   // 错误信息
    RootCause    string        `json:"root_cause,omitempty"`      // 根因分析
    ActionHint   string        `json:"action_hint,omitempty"`     // 恢复建议
    Retryable    bool          `json:"retryable"`                 // 是否可重试
    ProviderCode string        `json:"provider_code,omitempty"`   // 提供方错误码
}

// Error 实现 error 接口。
func (e *ToolError) Error() string {
    if e == nil {
        return ""
    }
    return string(e.Category) + ": " + e.Message
}
```

---

## 12. Intent（意图任务）

Intent 是平台的“全 AI 入口”对象。用户通过文本/语音/视频表达目标后，系统将其解析为结构化动作计划（IntentPlan），并在策略审批通过后执行。Intent 不替代 Workflow/Run，而是负责将自然语言目标映射为可执行操作链。

### 12.1 核心类型

```go
package domain

import (
    "time"

    "github.com/google/uuid"
)

// IntentSourceType 意图输入来源类型。
type IntentSourceType string

const (
    IntentSourceText  IntentSourceType = "text"  // 纯文本输入
    IntentSourceVoice IntentSourceType = "voice" // 语音输入（通常先转写为文本）
    IntentSourceVideo IntentSourceType = "video" // 视频输入（通常先提取转写/描述）
    IntentSourceMixed IntentSourceType = "mixed" // 多模态混合输入
)

// IntentStatus 意图生命周期状态。
type IntentStatus string

const (
    IntentStatusReceived          IntentStatus = "received"
    IntentStatusParsing           IntentStatus = "parsing"
    IntentStatusPlanned           IntentStatus = "planned"
    IntentStatusWaitingConfirm    IntentStatus = "waiting_confirmation"
    IntentStatusApproved          IntentStatus = "approved"
    IntentStatusExecuting         IntentStatus = "executing"
    IntentStatusSucceeded         IntentStatus = "succeeded"
    IntentStatusFailed            IntentStatus = "failed"
    IntentStatusRejected          IntentStatus = "rejected"
    IntentStatusCancelled         IntentStatus = "cancelled"
)

// IntentExecutionMode 意图执行模式。
type IntentExecutionMode string

const (
    IntentExecutionModeDryRun             IntentExecutionMode = "dry_run"              // 仅生成计划，不执行
    IntentExecutionModeConfirmThenExecute IntentExecutionMode = "confirm_then_execute" // 默认：高风险动作需确认
    IntentExecutionModeAutoExecute        IntentExecutionMode = "auto_execute"         // 低风险且授权充分时自动执行
)

// IntentActionType 平台动作类型（可扩展）。
// 覆盖“创建用户/角色/权限、更改设置、上传资源”等管理与执行行为。
type IntentActionType string

const (
    IntentActionCreateUser            IntentActionType = "identity.user.create"
    IntentActionCreateRole            IntentActionType = "identity.role.create"
    IntentActionCreatePermission      IntentActionType = "identity.permission.create"
    IntentActionUpdateRolePermissions IntentActionType = "identity.role.permissions.update"
    IntentActionBindRole              IntentActionType = "identity.role.bind"
    IntentActionUpdateSettings        IntentActionType = "settings.update"
    IntentActionUploadAsset           IntentActionType = "asset.upload"
    IntentActionRunWorkflow           IntentActionType = "workflow.run"
    IntentActionInvokeTool            IntentActionType = "tool.invoke"
)

// IntentActionStatus 单个动作状态。
type IntentActionStatus string

const (
    IntentActionPending   IntentActionStatus = "pending"
    IntentActionReady     IntentActionStatus = "ready"
    IntentActionRunning   IntentActionStatus = "running"
    IntentActionSucceeded IntentActionStatus = "succeeded"
    IntentActionFailed    IntentActionStatus = "failed"
    IntentActionSkipped   IntentActionStatus = "skipped"
    IntentActionRejected  IntentActionStatus = "rejected"
)

// IntentAction 计划中的单个可执行动作。
type IntentAction struct {
    ID                  string             `json:"id"`                             // 计划内唯一 step_id
    Type                IntentActionType   `json:"type"`                           // 动作类型
    Resource            string             `json:"resource"`                       // 资源域，如 users/roles/settings/assets
    Params              JSONMap            `json:"params"`                         // 动作参数（已结构化）
    DependsOn           []string           `json:"depends_on,omitempty"`           // 依赖步骤
    RiskLevel           RiskLevel          `json:"risk_level"`                     // 风险等级
    RequiredPermissions []string           `json:"required_permissions,omitempty"` // 所需权限
    NeedConfirmation    bool               `json:"need_confirmation"`              // 是否需要用户确认
    Status              IntentActionStatus `json:"status"`                         // 执行状态
    Error               string             `json:"error,omitempty"`                // 失败原因
}

// IntentPlan 意图解析后的执行计划。
type IntentPlan struct {
    Version              int            `json:"version"`                            // 计划版本，重规划递增
    Goal                 string         `json:"goal"`                               // 解析后的目标描述
    Summary              string         `json:"summary"`                            // 计划摘要（用于确认 UI）
    Actions              []IntentAction `json:"actions"`                            // 动作 DAG/序列
    InputAssets          []AssetRef     `json:"input_assets,omitempty"`             // 引用的已存在资产
    OutputExpectations   []string       `json:"output_expectations,omitempty"`      // 预期产出
    GeneratedWorkflowID  *uuid.UUID     `json:"generated_workflow_id,omitempty"`    // 自动生成工作流（可选）
    GeneratedWorkflowRev *int           `json:"generated_workflow_revision,omitempty"`
}

// Intent 意图任务实体。
type Intent struct {
    ID            uuid.UUID            `json:"id"`
    TenantID      uuid.UUID            `json:"tenant_id"`
    ActorID       uuid.UUID            `json:"actor_id"`
    SourceType    IntentSourceType     `json:"source_type"`
    RawInput      string               `json:"raw_input"`                  // 原始输入（文本或转写结果）
    InputAssets   []AssetRef           `json:"input_assets,omitempty"`     // 用户附加或自动匹配的资产
    Constraints   JSONMap              `json:"constraints,omitempty"`      // 业务/安全约束
    ExecutionMode IntentExecutionMode  `json:"execution_mode"`
    Status        IntentStatus         `json:"status"`
    Plan          *IntentPlan          `json:"plan,omitempty"`
    LinkedRunID   *uuid.UUID           `json:"linked_run_id,omitempty"`    // 实际执行对应 Run（若有）
    Result        JSONMap              `json:"result,omitempty"`            // 执行结果摘要
    CreatedAt     time.Time            `json:"created_at"`
    UpdatedAt     time.Time            `json:"updated_at"`
}
```

### 12.2 Intent 接口

```go
// IntentFilter 意图查询过滤条件。
type IntentFilter struct {
    TenantID   uuid.UUID
    ActorID    *uuid.UUID
    SourceType *IntentSourceType
    Status     *IntentStatus
    Keyword    string
    CreatedFrom *time.Time
    CreatedTo   *time.Time
    Limit      int
    Offset     int
}

// IntentRepository 意图持久化接口。
type IntentRepository interface {
    Create(ctx context.Context, intent *Intent) error
    GetByID(ctx context.Context, id uuid.UUID) (*Intent, error)
    List(ctx context.Context, filter IntentFilter) ([]*Intent, int64, error)
    Update(ctx context.Context, intent *Intent) error
    AppendActions(ctx context.Context, intentID uuid.UUID, actions []IntentAction) error
    UpdateActionStatus(ctx context.Context, intentID uuid.UUID, actionID string, status IntentActionStatus, errMsg string) error
}

// IntentPlanner 负责将自然语言/语音转写结果编译为 IntentPlan。
type IntentPlanner interface {
    Plan(ctx context.Context, intent *Intent) (*IntentPlan, error)
    Replan(ctx context.Context, intent *Intent, feedback JSONMap) (*IntentPlan, error)
}

// IntentExecutor 负责执行 IntentPlan（包含确认、审批与恢复流程）。
type IntentExecutor interface {
    Execute(ctx context.Context, intentID uuid.UUID) (*Run, error)
    Approve(ctx context.Context, intentID uuid.UUID, actorID uuid.UUID, comment string) error
    Reject(ctx context.Context, intentID uuid.UUID, actorID uuid.UUID, reason string) error
    Cancel(ctx context.Context, intentID uuid.UUID, reason string) error
}
```

---

## 13. 全局接口定义

除了各实体章节中已定义的 Repository 接口外，以下是系统级的服务接口定义。所有接口均在 domain 包中声明，由 Adapter 层实现。

### 13.1 ModelRegistry（模型注册表）

```go
// ModelRegistry 模型注册表接口。
// 管理 AI 模型的连接配置与可用性。
type ModelRegistry interface {
    // GetModel 获取模型配置。
    GetModel(ctx context.Context, modelRef string) (*ModelConfig, error)

    // ListModels 列出所有可用模型。
    ListModels(ctx context.Context) ([]*ModelConfig, error)

    // HealthCheck 模型健康检查。
    HealthCheck(ctx context.Context, modelRef string) error
}

// ModelConfig 模型配置。
type ModelConfig struct {
    Ref         string  `json:"ref"`          // 模型引用标识
    Provider    string  `json:"provider"`     // 提供方（openai/anthropic/local）
    Model       string  `json:"model"`        // 模型名称
    Endpoint    string  `json:"endpoint"`     // API 端点
    MaxTokens   int     `json:"max_tokens"`   // 最大 Token 数
    CostPerToken float64 `json:"cost_per_token"` // 每 Token 成本（美元）
    Config      JSONMap `json:"config,omitempty"` // 额外配置
}
```

### 13.2 PolicyEngine（策略引擎）

```go
// PolicyEngine 策略引擎接口。
// 在工具调用前进行权限校验与策略评估。
type PolicyEngine interface {
    // Evaluate 评估执行策略。
    // 返回 nil 表示通过，返回 error 表示策略拒绝。
    Evaluate(ctx context.Context, envelope *ExecutionEnvelope) error

    // CheckToolPermission 检查工具调用权限。
    CheckToolPermission(ctx context.Context, toolCode string, callerID string, permissions []string) error

    // CheckBudget 检查预算是否充足。
    CheckBudget(ctx context.Context, runID uuid.UUID) error

    // CheckDataAccess 检查数据访问权限。
    CheckDataAccess(ctx context.Context, toolCode string, dataAccess DataAccessSpec) error
}
```

### 13.3 AuditLogger（审计日志）

```go
// AuditLogger 审计日志接口。
// 记录系统中的关键操作，用于安全审计与合规。
type AuditLogger interface {
    // Log 记录审计事件。
    Log(ctx context.Context, entry AuditEntry) error

    // Query 查询审计日志。
    Query(ctx context.Context, filter AuditFilter) ([]AuditEntry, int64, error)
}

// AuditEntry 审计日志条目。
type AuditEntry struct {
    ID        uuid.UUID `json:"id"`
    TenantID  string    `json:"tenant_id"`
    ActorID   string    `json:"actor_id"`   // 操作者 ID
    ActorType string    `json:"actor_type"` // user/agent/system
    Action    string    `json:"action"`     // create/update/delete/execute/access
    Resource  string    `json:"resource"`   // 资源类型
    ResourceID string   `json:"resource_id"` // 资源 ID
    Details   JSONMap   `json:"details,omitempty"`
    Timestamp time.Time `json:"timestamp"`
}

// AuditFilter 审计日志查询过滤条件。
type AuditFilter struct {
    TenantID   string
    ActorID    string
    Action     string
    Resource   string
    ResourceID string
    From       *time.Time
    To         *time.Time
    Limit      int
    Offset     int
}
```

### 13.4 EventBus（事件总线）

```go
// EventBus 内部事件总线接口。
// 用于模块间异步通信（如资产就绪事件触发工作流）。
type EventBus interface {
    // Publish 发布事件。
    Publish(ctx context.Context, event DomainEvent) error

    // Subscribe 订阅事件。
    // handler 在事件到达时被调用，返回 error 表示处理失败。
    Subscribe(ctx context.Context, eventType string, handler EventHandler) error

    // Unsubscribe 取消订阅。
    Unsubscribe(ctx context.Context, eventType string, handler EventHandler) error
}

// DomainEvent 领域事件。
type DomainEvent struct {
    ID        uuid.UUID `json:"id"`
    Type      string    `json:"type"`       // asset.created/asset.ready/run.completed 等
    Source    string    `json:"source"`     // 事件来源模块
    TenantID  string    `json:"tenant_id"`
    Payload   JSONMap   `json:"payload"`
    Timestamp time.Time `json:"timestamp"`
}

// EventHandler 事件处理函数类型。
type EventHandler func(ctx context.Context, event DomainEvent) error
```

### 13.5 Scheduler（调度器）

```go
// Scheduler 调度器接口。
// 管理定时触发与事件触发的工作流执行。
type Scheduler interface {
    // RegisterWorkflow 注册工作流到调度器。
    // 根据 TriggerConfig 创建定时任务或事件监听。
    RegisterWorkflow(ctx context.Context, wf *WorkflowDefinition) error

    // UnregisterWorkflow 从调度器移除工作流。
    UnregisterWorkflow(ctx context.Context, workflowID uuid.UUID) error

    // Start 启动调度器。
    Start(ctx context.Context) error

    // Stop 停止调度器。
    Stop(ctx context.Context) error

    // ListScheduled 列出所有已调度的工作流。
    ListScheduled(ctx context.Context) ([]ScheduledWorkflow, error)
}

// ScheduledWorkflow 已调度的工作流信息。
type ScheduledWorkflow struct {
    WorkflowID uuid.UUID   `json:"workflow_id"`
    Code       string      `json:"code"`
    TriggerType TriggerType `json:"trigger_type"`
    NextRunAt  *time.Time  `json:"next_run_at,omitempty"` // 下次执行时间（仅 schedule 类型）
    Status     string      `json:"status"`                // active/paused
}
```

### 13.6 UnitOfWork（工作单元）

```go
// UnitOfWork 工作单元接口。
// 提供跨 Repository 的事务管理能力。
type UnitOfWork interface {
    // Begin 开始事务，返回事务上下文。
    Begin(ctx context.Context) (context.Context, error)

    // Commit 提交事务。
    Commit(ctx context.Context) error

    // Rollback 回滚事务。
    Rollback(ctx context.Context) error

    // RunInTransaction 在事务中执行函数。
    // 自动处理 Commit/Rollback。
    RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
```

### 13.7 IdentityRepository（用户与角色）

```go
// UserStatus 用户状态枚举。
type UserStatus string

const (
    UserStatusActive   UserStatus = "active"
    UserStatusDisabled UserStatus = "disabled"
)

// RoleScope 角色作用域枚举。
type RoleScope string

const (
    RoleScopeTenant RoleScope = "tenant"
    RoleScopeSystem RoleScope = "system"
)

// User 用户实体（RBAC 主体）。
type User struct {
    ID        uuid.UUID   `json:"id"`
    TenantID  uuid.UUID   `json:"tenant_id"`
    Name      string      `json:"name"`
    Email     string      `json:"email"`
    Status    UserStatus  `json:"status"`
    CreatedAt time.Time   `json:"created_at"`
    UpdatedAt time.Time   `json:"updated_at"`
}

// Role 角色实体（RBAC 权限集合）。
type Role struct {
    ID          uuid.UUID `json:"id"`
    TenantID    *uuid.UUID `json:"tenant_id,omitempty"` // system 角色为空
    Scope       RoleScope `json:"scope"`                // tenant/system
    Name        string    `json:"name"`
    Code        string    `json:"code"`
    Permissions []string  `json:"permissions"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

// RoleBinding 用户与角色绑定关系。
type RoleBinding struct {
    ID        uuid.UUID `json:"id"`
    UserID    uuid.UUID `json:"user_id"`
    RoleID    uuid.UUID `json:"role_id"`
    GrantedBy uuid.UUID `json:"granted_by"`
    CreatedAt time.Time `json:"created_at"`
}

// Permission 权限模板实体（可选，作为平台可见权限目录）。
// 若不启用独立权限表，也可仅作为内存/配置层定义。
type Permission struct {
    ID          uuid.UUID  `json:"id"`
    TenantID    *uuid.UUID `json:"tenant_id,omitempty"` // nil 表示系统级权限模板
    Code        string     `json:"code"`                // 例如 tool:invoke:tenant/*
    Name        string     `json:"name"`
    Description string     `json:"description"`
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
}

// IdentityRepository 用户与角色持久化接口。
type IdentityRepository interface {
    // User
    CreateUser(ctx context.Context, user *User) error
    GetUser(ctx context.Context, id uuid.UUID) (*User, error)
    ListUsers(ctx context.Context, tenantID uuid.UUID, status *UserStatus, keyword string, limit, offset int) ([]*User, int64, error)
    UpdateUser(ctx context.Context, user *User) error

    // Role
    CreateRole(ctx context.Context, role *Role) error
    GetRole(ctx context.Context, id uuid.UUID) (*Role, error)
    ListRoles(ctx context.Context, tenantID *uuid.UUID, scope *RoleScope, keyword string, limit, offset int) ([]*Role, int64, error)
    UpdateRole(ctx context.Context, role *Role) error

    // Binding
    BindRole(ctx context.Context, binding *RoleBinding) error
    UnbindRole(ctx context.Context, bindingID uuid.UUID) error
    ListUserRoles(ctx context.Context, userID uuid.UUID) ([]*Role, error)

    // Permission（可选实现）
    CreatePermission(ctx context.Context, permission *Permission) error
    GetPermission(ctx context.Context, id uuid.UUID) (*Permission, error)
    ListPermissions(ctx context.Context, tenantID *uuid.UUID, keyword string, limit, offset int) ([]*Permission, int64, error)
}
```

---

## 14. 领域错误定义

```go
package domain

import "errors"

// 资产相关错误
var (
    ErrAssetNotFound      = errors.New("asset not found")
    ErrAssetAlreadyExists = errors.New("asset already exists")
    ErrAssetNotActive     = errors.New("asset is not active")
    ErrAssetTypeInvalid   = errors.New("invalid asset type")
)

// 工具相关错误
var (
    ErrToolNotFound       = errors.New("tool not found")
    ErrToolCodeDuplicate  = errors.New("tool code already exists")
    ErrToolDisabled       = errors.New("tool is disabled")
    ErrToolDeprecated     = errors.New("tool is deprecated")
    ErrToolValidationFail = errors.New("tool input validation failed")
)

// 运行相关错误
var (
    ErrRunNotFound        = errors.New("run not found")
    ErrRunAlreadyTerminal = errors.New("run is already in terminal state")
    ErrRunBudgetExceeded  = errors.New("run budget exceeded")
    ErrRunNestingExceeded = errors.New("max nesting level exceeded")
)

// 工作流相关错误
var (
    ErrWorkflowNotFound   = errors.New("workflow not found")
    ErrWorkflowNoNodes    = errors.New("workflow must have at least one node")
    ErrWorkflowHasCycle   = errors.New("workflow DAG contains cycle")
    ErrWorkflowNotActive  = errors.New("workflow is not active")
)

// 算法相关错误
var (
    ErrAlgorithmNotFound         = errors.New("algorithm not found")
    ErrAlgorithmVersionNotFound  = errors.New("algorithm version not found")
    ErrAlgorithmVersionFrozen    = errors.New("published algorithm version cannot be modified")
    ErrNoImplementationAvailable = errors.New("no active implementation available for algorithm")
)

// 上下文相关错误
var (
    ErrContextNotInitialized    = errors.New("context state not initialized")
    ErrContextVersionConflict   = errors.New("context version conflict (CAS failure)")
    ErrContextSharedKeyUndeclared = errors.New("shared key not declared in context spec")
    ErrContextSizeExceeded      = errors.New("context state size exceeded limit")
)

// 策略相关错误
var (
    ErrPolicyDenied           = errors.New("policy denied")
    ErrPermissionInsufficient = errors.New("insufficient permission")
    ErrDataAccessDenied       = errors.New("data access denied")
)

// Agent 相关错误
var (
    ErrAgentProfileNotFound = errors.New("agent profile not found")
    ErrAgentToolDenied      = errors.New("tool not allowed by agent tool scope")
)

// Intent 相关错误
var (
    ErrIntentNotFound             = errors.New("intent not found")
    ErrIntentParseFailed          = errors.New("intent parse failed")
    ErrIntentPlanEmpty            = errors.New("intent plan is empty")
    ErrIntentConfirmationRequired = errors.New("intent confirmation required")
    ErrIntentActionUnsupported    = errors.New("intent action unsupported")
)
```

---

## 15. 实体关系总览

```
                        ┌─────────────────────┐
                        │    Asset             │
                        │  (所有媒体与数据)     │
                        └──────┬──────────────┘
                               │ ParentID (衍生)
                               ▼
                        ┌─────────────────────┐
                        │    Asset (child)     │
                        └─────────────────────┘

┌─────────────────┐     ┌─────────────────────┐     ┌─────────────────────┐
│   ToolSpec      │     │  Algorithm          │     │  AgentProfile       │
│  (能力原子)      │◄────│  (意图层)            │     │  (行为契约)          │
│                 │     │  └─ AlgorithmVersion │     └──────┬──────────────┘
│                 │     │      └─ ImplBinding  │            │
│                 │     │      └─ EvalProfile  │            │
└────────┬────────┘     └─────────────────────┘            │
         │                                                  │
         │              ┌─────────────────────┐            │
         └──────────────│  WorkflowDefinition │◄───────────┘
                        │  (DAG 编排)          │  NodeType=agent
                        │  └─ NodeDefinition  │
                        │  └─ EdgeDefinition  │
                        │  └─ ContextSpec     │
                        └──────┬──────────────┘
                               │ Execute
                               ▼
                        ┌─────────────────────┐
                        │    Run              │
                        │  (执行实例)          │◄──── ParentRunID (嵌套)
                        │  └─ RunBudget       │
                        │  └─ RunError        │
                        └──────┬──────────────┘
                               │
                    ┌──────────┼──────────┐
                    ▼          ▼          ▼
            ┌────────────┐ ┌──────────┐ ┌─────────────┐
            │ RunEvent   │ │ Context  │ │ Artifact    │
            │ (观测事件)  │ │ State    │ │ (执行产物)   │
            └────────────┘ │ Patch    │ └─────────────┘
                           │ Snapshot │
                           └──────────┘
```

### 15.1 核心关系说明

| 关系 | 描述 |
|------|------|
| `Asset.ParentID → Asset` | 衍生资产追踪链 |
| `WorkflowDefinition.Nodes[].ToolID → ToolSpec` | 工作流节点引用工具 |
| `WorkflowDefinition.Nodes[].AlgorithmRef → Algorithm` | 工作流节点引用算法 |
| `WorkflowDefinition.Nodes[].AgentProfileID → AgentProfile` | 工作流节点引用 Agent |
| `ImplementationBinding.ToolID → ToolSpec` | 算法实现绑定到工具 |
| `Intent.InputAssets[] → Asset` | 意图执行可复用项目内已存在资产 |
| `Intent.Plan.Actions[] → User/Role/Tool/Workflow/Settings` | 意图动作可覆盖平台管理与执行行为 |
| `Intent.LinkedRunID → Run` | 意图执行绑定到统一运行容器 |
| `Run.WorkflowID → WorkflowDefinition` | 运行对应的工作流 |
| `Run.ParentRunID → Run` | 子运行与父运行的嵌套关系 |
| `RunEvent.RunID → Run` | 事件归属的运行实例 |
| `ContextPatch.RunID → Run` | 上下文补丁归属的运行实例 |
| `EvaluationProfile.RunID → Run` | 评测运行实例 |
| `RoleBinding.UserID → User` | 用户与角色绑定关系 |
| `RoleBinding.RoleID → Role` | 绑定到具体角色 |
| `Role.Permissions[] → Permission.Code` | 角色权限集引用权限模板编码 |

---

## 16. 数据库表一览

| 表名 | 对应实体 | 说明 |
|------|---------|------|
| `assets` | `Asset` | 统一资产表 |
| `tools` | `ToolSpec` | 工具规范表 |
| `runs` | `Run` | 运行实例表 |
| `run_events` | `RunEvent` | 运行事件表（append-only） |
| `workflow_definitions` | `WorkflowDefinition` | 工作流定义表 |
| `agent_profiles` | `AgentProfile` | Agent 配置表 |
| `intents` | `Intent` | 意图任务表（含计划与执行状态） |
| `intent_actions` | `IntentAction` | 意图动作明细表 |
| `users` | `User` | 租户用户表 |
| `roles` | `Role` | RBAC 角色表（租户/系统） |
| `role_bindings` | `RoleBinding` | 用户-角色绑定表 |
| `permissions` | `Permission` | 权限模板表（系统/租户可选） |
| `algorithms` | `Algorithm` | 算法表 |
| `algorithm_versions` | `AlgorithmVersion` | 算法版本表 |
| `implementation_bindings` | `ImplementationBinding` | 算法实现绑定表 |
| `evaluation_profiles` | `EvaluationProfile` | 算法评测档案表 |
| `context_states` | `ContextState` | 上下文物化状态表 |
| `context_patches` | `ContextPatch` | 上下文补丁表（append-only） |
| `context_snapshots` | `ContextSnapshot` | 上下文快照表 |

### 16.1 关键索引

```sql
-- assets
CREATE INDEX idx_assets_tenant_type ON assets(tenant_id, type);
CREATE INDEX idx_assets_parent ON assets(parent_id) WHERE parent_id IS NOT NULL;
CREATE INDEX idx_assets_status ON assets(status);
CREATE INDEX idx_assets_created ON assets(created_at DESC);

-- tools
CREATE UNIQUE INDEX idx_tools_code ON tools(code);
CREATE INDEX idx_tools_category ON tools(category);
CREATE INDEX idx_tools_status ON tools(status);

-- runs
CREATE INDEX idx_runs_tenant_type ON runs(tenant_id, type);
CREATE INDEX idx_runs_workflow ON runs(workflow_id) WHERE workflow_id IS NOT NULL;
CREATE INDEX idx_runs_parent ON runs(parent_run_id) WHERE parent_run_id IS NOT NULL;
CREATE INDEX idx_runs_status ON runs(status);
CREATE INDEX idx_runs_trace ON runs(trace_id);
CREATE INDEX idx_runs_created ON runs(created_at DESC);

-- run_events
CREATE INDEX idx_run_events_run ON run_events(run_id);
CREATE INDEX idx_run_events_type ON run_events(type);
CREATE INDEX idx_run_events_timestamp ON run_events(timestamp);
CREATE UNIQUE INDEX idx_run_events_run_seq ON run_events(run_id, seq);

-- workflow_definitions
CREATE UNIQUE INDEX idx_workflows_code ON workflow_definitions(code);
CREATE INDEX idx_workflows_tenant_status ON workflow_definitions(tenant_id, status);

-- intents
CREATE INDEX idx_intents_tenant_status ON intents(tenant_id, status);
CREATE INDEX idx_intents_actor_created ON intents(actor_id, created_at DESC);
CREATE INDEX idx_intent_actions_intent_status ON intent_actions(intent_id, status);

-- users / roles
CREATE INDEX idx_users_tenant_status ON users(tenant_id, status);
CREATE UNIQUE INDEX idx_roles_scope_code ON roles(scope, code);
CREATE INDEX idx_role_bindings_user ON role_bindings(user_id);
CREATE UNIQUE INDEX idx_permissions_code ON permissions(code);

-- context_patches
CREATE INDEX idx_context_patches_run_version ON context_patches(run_id, version);

-- algorithm_versions
CREATE INDEX idx_algo_versions_algorithm ON algorithm_versions(algorithm_id);

-- implementation_bindings
CREATE INDEX idx_impl_bindings_version ON implementation_bindings(algorithm_version_id);
```

---

最后更新：2026-02-09
