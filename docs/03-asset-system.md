# Goyais 资产系统设计

> 本文档定义 Goyais 资产系统（Asset System）的类型体系、存储抽象、生命周期、派生关系与流媒体集成方案，是 `Asset` 一等对象在工程上的落地规范。

最后更新：2026-02-09

---

## 1. 目标与范围

### 1.1 目标

资产系统负责回答三个问题：

1. 如何统一表达多模态输入/输出
2. 如何在不复制原始内容的前提下保证可追溯
3. 如何把离线文件与实时流纳入同一数据模型

### 1.2 设计边界

资产系统只负责：

- 资产元数据管理
- 原始内容引用与存储抽象
- 资产派生链路与可追溯关系
- 流媒体接入与切片索引
- 为 Intent/Workflow/Agent 提供统一可复用的资产引用入口

资产系统不负责：

- 内容标注与数据集管理（可对接外部标注平台）
- 模型训练集编排
- 通用文件协同编辑

---

## 2. 统一资产模型

### 2.1 类型体系

| 类型 | 说明 | 必需元数据 |
|------|------|-----------|
| `video` | 视频文件 | `duration`,`width`,`height`,`fps`,`codec`,`bitrate` |
| `image` | 图片文件 | `width`,`height`,`color_space`,`dpi` |
| `audio` | 音频文件 | `duration`,`sample_rate`,`channels`,`codec`,`bitrate` |
| `document` | 文档文件（PDF/Word/Excel） | `page_count` 或 `sheet_names` |
| `stream` | 实时流媒体 | `source_id`,`mediamtx_path`,`protocol` |
| `structured` | JSON/CSV/表格型结构化结果 | `schema`,`row_count` |
| `text` | 纯文本内容 | `encoding`,`language` |

### 2.2 核心字段约束

`Asset` 字段规范与 `02-domain-model.md` 一致，并追加以下约束：

- `uri`：必须为可解析的稳定引用，禁止写入临时 URL
- `store_ref`：仅作为历史兼容别名（API 输出可保留），落库字段统一为 `uri`
- `digest`：推荐使用 `sha256:<hex>`，用于去重与完整性校验
- `parent_id`：仅允许单父节点；多来源产物通过 `ArtifactLineage` 表达

### 2.3 不可变原则

Asset 元数据可补充，但内容引用不可覆写：

- 禁止更新 `uri` 指向新内容
- 若内容变化，必须新建 Asset，并设置 `parent_id`
- 同一 `digest` 可被多个 Asset 复用，但每个 Asset 保留独立业务语义

---

## 3. Store 抽象

### 3.1 抽象接口

```go
type AssetStore interface {
    Put(ctx context.Context, key string, r io.Reader, opts PutOptions) (StoreObject, error)
    Get(ctx context.Context, key string) (io.ReadCloser, error)
    Head(ctx context.Context, key string) (StoreObject, error)
    Delete(ctx context.Context, key string) error
    PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error)
}

type StoreObject struct {
    Key       string
    ETag      string
    SizeBytes int64
    MimeType  string
    CreatedAt time.Time
}
```

### 3.2 适配实现

| 实现 | 场景 | 说明 |
|------|------|------|
| `local` | 开发/单机部署 | 文件系统落盘，路径规则 `data/assets/<tenant>/<yyyy>/<mm>/<dd>/...` |
| `s3` | 生产云部署 | AWS S3 或兼容接口 |
| `minio` | 私有化部署 | S3 协议兼容，支持本地对象存储 |

### 3.3 Key 规范

统一 key 结构：

```text
{tenant_id}/{asset_type}/{yyyy}/{mm}/{dd}/{asset_id}/{filename}
```

示例：

```text
8f1.../video/2026/02/09/5fd.../camera-01.mp4
```

---

## 4. 资产接入链路

### 4.1 文件上传接入

```text
Client Upload -> Access API -> MIME/大小校验 -> AssetStore.Put
             -> 写入 assets 表 -> 产出 asset_created 事件
```

### 4.2 外部引用导入

对外部 URL 导入使用“先拉取后托管”策略：

1. 校验域名白名单（见 `09-security-policy.md`）
2. 拉取内容至临时区并做病毒/格式校验
3. 写入 Asset Store
4. 生成正式 Asset

### 4.3 流媒体接入

`stream` 类型通过 MediaMTX 管理：

- 记录 `source_id` 与 `mediamtx_path`
- 维护当前在线状态、码率、最近心跳
- 可按策略启动录制并自动生成 `video` 子 Asset

---

## 5. StreamAsset 与 MediaMTX 集成

### 5.1 关键字段

```go
type StreamMetadata struct {
    SourceID     string   `json:"source_id"`
    Protocol     string   `json:"protocol"`      // rtsp|rtmp|hls|webrtc
    MediaMTXPath string   `json:"mediamtx_path"`
    RecordEnable bool     `json:"record_enable"`
    SliceWindowS int      `json:"slice_window_s"`
    Labels       []string `json:"labels,omitempty"`
}
```

### 5.2 时间切片索引（TimeSliceIndex）

流媒体按固定窗口生成可检索切片：

| 字段 | 说明 |
|------|------|
| `slice_id` | 切片唯一标识 |
| `stream_asset_id` | 所属流资产 |
| `start_at`/`end_at` | 时间范围 |
| `uri` | 对应录制段地址 |
| `keyframe_index` | 关键帧索引（可选） |

查询示例：

```sql
SELECT * FROM stream_slices
WHERE stream_asset_id = $1
  AND start_at < $3
  AND end_at > $2
ORDER BY start_at;
```

切片创建算法（规范）：

1. 录制流按 GOP 边界切片，默认窗口 `slice_window_s=6` 秒。
2. 若检测到关键帧间隔过大（>`slice_window_s * 1.5`），允许按时间窗口强制切片并记录 `non_keyframe_start=true`。
3. 每个切片写入时必须附带：
   - `uri`
   - `start_at` / `end_at`
   - `seq_no`（单流递增序号）
   - `keyframe_index`（切片内关键帧时间偏移）
4. MediaMTX 断连处理：
   - 连续 `heartbeat_timeout`（默认 10s）未收到流心跳，生成 `stream_disconnected` 内部事件
   - 当前未完成切片以 `partial=true` 落盘，仍可回放但默认不参与算法主链
   - 重连后切片序号延续，且新切片 `discontinuity=true`，便于下游去抖与拼接
5. 下游读取切片时默认过滤 `partial=true`，可通过调试参数显式包含。

### 5.3 录制派生

当录制开启时：

- 每个分段文件注册为新 `video` Asset
- `parent_id` 指向原 `stream` Asset
- 同步写入 `artifact_created` / `asset_created` 事件

---

## 6. 派生链与血缘关系

### 6.1 单步派生

典型链路：

```text
stream -> video segment -> image frame -> structured detection result -> document report
```

### 6.2 血缘查询

支持两类查询：

1. 正向追踪：输入资产产生了哪些衍生产物
2. 反向溯源：某个报告源自哪些原始资产

推荐使用递归 CTE：

```sql
WITH RECURSIVE lineage AS (
  SELECT id, parent_id, 0 AS depth
  FROM assets
  WHERE id = $1
  UNION ALL
  SELECT a.id, a.parent_id, l.depth + 1
  FROM assets a
  JOIN lineage l ON a.parent_id = l.id
)
SELECT * FROM lineage;
```

### 6.3 跨资产聚合

对于多输入节点（例如两个视频拼接），不在 `parent_id` 建立多父关系，而通过 Artifact 元数据记录：

```json
{
  "lineage": {
    "source_assets": ["asset-a", "asset-b"],
    "merge_strategy": "timeline_concat"
  }
}
```

---

## 7. 生命周期与治理

### 7.1 生命周期状态

| 状态 | 说明 | 可读 | 可写 |
|------|------|------|------|
| `active` | 正常可用 | 是 | 允许派生 |
| `archived` | 归档 | 是 | 不允许新派生（默认） |
| `deleted` | 软删除 | 管控可读 | 禁止 |

### 7.2 清理策略

- 软删除延迟清理：默认保留 30 天
- 硬删除前检查引用计数与合规保留策略
- 任何硬删除行为必须写审计日志

### 7.3 保留策略

按租户可配置：

```yaml
asset_retention:
  default_days: 180
  stream_slice_days: 30
  audit_hold_tags: ["legal-hold", "compliance"]
```

---

## 8. 权限与安全控制

### 8.1 访问控制

资产访问受 RBAC + Scope 双重约束：

- `asset:read:{tenant}/{asset_id}`
- `asset:write:{tenant}/*`
- `asset:delete:{tenant}/{asset_id}`

### 8.2 数据访问声明联动

Tool 执行时，`ToolSpec.DataAccess.ReadScopes/WriteScopes` 必须与目标 Asset 的租户范围与访问 scope 匹配；不匹配则在 Policy Engine 阶段拒绝执行。

### 8.3 下载链路保护

- 默认返回短期预签名 URL（TTL 60s-10m）
- 对公网下载可启用一次性 token
- 对高敏内容启用水印与访问审计

---

## 9. 数据库设计

### 9.1 assets 表（补充约束）

```sql
CREATE TABLE assets (
    id           UUID PRIMARY KEY,
    tenant_id    UUID         NOT NULL,
    owner_id     UUID         NOT NULL,
    name         VARCHAR(512) NOT NULL,
    type         VARCHAR(32)  NOT NULL,
    uri          TEXT         NOT NULL,
    digest       VARCHAR(128) NOT NULL,
    size         BIGINT       NOT NULL DEFAULT 0,
    mime_type    VARCHAR(128) NOT NULL DEFAULT '',
    metadata     JSONB        NOT NULL DEFAULT '{}',
    parent_id    UUID         NULL REFERENCES assets(id),
    tags         TEXT[]       NOT NULL DEFAULT '{}',
    status       VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_assets_tenant_type ON assets(tenant_id, type);
CREATE INDEX idx_assets_parent_id ON assets(parent_id) WHERE parent_id IS NOT NULL;
CREATE INDEX idx_assets_digest ON assets(digest);
CREATE INDEX idx_assets_created_at ON assets(created_at DESC);
```

### 9.2 stream_slices 表

```sql
CREATE TABLE stream_slices (
    id               UUID PRIMARY KEY,
    stream_asset_id  UUID NOT NULL REFERENCES assets(id),
    start_at         TIMESTAMPTZ NOT NULL,
    end_at           TIMESTAMPTZ NOT NULL,
    uri              TEXT NOT NULL,
    keyframe_index   JSONB NOT NULL DEFAULT '{}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_stream_slices_range ON stream_slices(stream_asset_id, start_at, end_at);
```

---

## 10. 运行时接口

### 10.1 资产服务接口

```go
type AssetService interface {
    Create(ctx context.Context, in CreateAssetInput) (*Asset, error)
    Get(ctx context.Context, id uuid.UUID) (*Asset, error)
    List(ctx context.Context, q AssetQuery) ([]Asset, int64, error)
    Derive(ctx context.Context, parentID uuid.UUID, in DeriveAssetInput) (*Asset, error)
    Archive(ctx context.Context, id uuid.UUID) error
    SoftDelete(ctx context.Context, id uuid.UUID) error
    FindLineage(ctx context.Context, id uuid.UUID, depth int) ([]Asset, error)
}
```

### 10.2 事件约定

| 事件 | 触发时机 |
|------|---------|
| `asset_created` | 新资产写入成功 |
| `asset_archived` | 资产归档 |
| `asset_deleted` | 资产软删除 |
| `asset_derived` | 派生资产创建 |
| `stream_slice_created` | 流切片写入 |

---

## 11. 与其他模块关系

| 模块 | 关系 |
|------|------|
| `04-tool-system.md` | Tool 输入输出大量依赖 AssetRef |
| `05-algorithm-library.md` | 算法评测数据与结果以 Asset 持久化 |
| `06-workflow-engine.md` | Workflow 的输入起点与输出终点均为 Asset |
| `08-observability.md` | 资产关键操作产出 RunEvent/Audit |
| `09-security-policy.md` | Asset 访问受策略引擎限制 |
| `10-api-design.md` | Asset REST/SSE 接口定义 |

---

## 12. 实施建议

### 12.1 分阶段落地

| 阶段 | 内容 |
|------|------|
| P1 | `assets` 表 + LocalStore + 上传/下载 API |
| P2 | S3/MinIO 适配 + digest 去重 + 派生关系 |
| P3 | StreamAsset + MediaMTX + 切片索引 |
| P4 | 生命周期治理（归档/清理/保留策略） |
| P5 | 血缘查询优化 + 审计联动 |

### 12.2 测试重点

- 大文件上传与断点重试
- 录制分段丢失/乱序恢复
- 深层派生链递归查询性能
- 多租户越权访问拦截
