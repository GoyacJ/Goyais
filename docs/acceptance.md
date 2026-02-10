# Goyais v0.1 验收清单

> 说明：本清单用于 v0.1 文档到实现阶段的统一验收。所有条目默认在同一租户内执行，且必须保留审计记录。

## 1. 基础验收条件

- [ ] API 前缀统一为 `/api/v1`。
- [ ] 所有副作用动作可通过 `/api/v1/commands` 表达并追踪。
- [ ] 错误模型统一为 `error: { code, messageKey, details }`。
- [ ] 关键对象包含通用字段：`id/tenantId/workspaceId/ownerId/visibility/acl/createdAt/updatedAt/status`。

## 2. 最小化运行模式验收（无外部依赖）

### 2.1 启动配置
- [ ] 使用 `sqlite + mediamtx + local object store + memory cache` 组合可启动。
- [ ] 在无 Postgres/Redis/MinIO 条件下，系统仍可完成基础闭环。

### 2.2 闭环能力
- [ ] 上传一个资产后可查询到元数据。
- [ ] 创建并运行一个最小 workflow（至少 1 个 step）可产生 run 记录。
- [ ] run/step 状态可查询，且审计中可看到对应 command。

## 3. Provider 切换验收

### 3.1 DB provider
- [ ] `db.driver=sqlite` 时迁移与查询可用。
- [ ] `db.driver=postgres` 时迁移与查询可用。
- [ ] 两种模式字段语义一致（时间/状态/visibility/acl 不漂移）。

### 3.2 Object store provider
- [ ] `object_store.provider=local` 可上传/读取/删除。
- [ ] `object_store.provider=minio` 可上传/读取/删除。
- [ ] `object_store.provider=s3` 可上传/读取/删除。

### 3.3 Cache + Vector provider
- [ ] `cache.provider=memory` 下系统可运行。
- [ ] `cache.provider=redis` 下缓存命中与过期符合预期。
- [ ] `vector.provider=redis_stack` 可写入并检索向量。
- [ ] 无 Redis 时 `vector.provider=sqlite` fallback 可检索。

## 4. Single Binary Packaging 验收

### 4.1 构建与独立运行
- [ ] 执行 `make build` 后产出单个可执行文件。
- [ ] 改名或删除 `web/dist`（可选强化：改名或删除 `web/`）后，启动该二进制。
- [ ] 访问 `/` 返回 200。
- [ ] 访问 `/canvas` 返回 200（SPA fallback 生效）。
- [ ] 访问 `/api/v1/healthz` 返回 200。

### 4.2 静态路由与特殊路径
- [ ] `/api/v1/*` 不被 SPA fallback 覆盖。
- [ ] 未提供占位文件时，`/favicon.ico` 返回 404。
- [ ] 未提供占位文件时，`/robots.txt` 返回 404。

### 4.3 响应头与类型
- [ ] `GET /` 返回 `Content-Type: text/html`（可含 charset）。
- [ ] `GET /canvas` 返回 `Content-Type: text/html`（可含 charset）。
- [ ] `GET /` 与 `GET /canvas` 的 `Cache-Control` 精确为 `no-store`。
- [ ] 首页引用的 `/assets/*.js` 返回 JavaScript 类型（`application/javascript` 或兼容值）。
- [ ] 静态文件不出现 `application/octet-stream` 误配（除确实二进制资源外）。

## 5. Command-first 与 AI/UI 一致性验收

- [ ] UI 触发 `workflow.run` 与 AI 触发同动作时，落库 command 形态一致。
- [ ] Domain 写接口响应包含：`resource + commandRef { commandId, status, acceptedAt }`。
- [ ] 通过 `GET /api/v1/commands/{commandId}` 可追踪最终执行结果。

### 5.1 A2 最小闭环（Thread #3）
- [ ] `POST /api/v1/commands`（携带 `X-Tenant-Id/X-Workspace-Id/X-User-Id`）返回 `202` 且包含 `resource + commandRef`。
- [ ] 缺少任一上下文 header 返回 `400`，错误为 `MISSING_CONTEXT + error.context.missing`，并在 `details.missingHeaders` 返回缺失项列表。
- [ ] `GET /api/v1/commands` 返回 `items`，并固定按 `created_at DESC, id DESC` 排序。
- [ ] cursor 模式 token 基于 `(created_at,id)`，若请求带 `cursor` 则忽略 `page/pageSize`。
- [ ] 同 `(tenant,workspace,owner,idempotency_key)` 且同请求哈希复用同一 `commandId`。
- [ ] 同 `(tenant,workspace,owner,idempotency_key)` 但不同请求哈希返回 `409 IDEMPOTENCY_KEY_CONFLICT`。
- [ ] `Idempotency-Key` 缺失时仍可创建新命令，并保留审计记录。
- [ ] SQLite（minimal）可完成 create/get/list + 状态流转 + 审计落库。
- [ ] Postgres（full）可连接并在 healthz 回显 provider；commands 业务接口可统一返回 `501 NOT_IMPLEMENTED`（本轮非阻塞）。

## 6. Visibility/ACL 与隔离验收

- [ ] 资产、工作流、算法、插件、流对象均支持 `PRIVATE/WORKSPACE/TENANT/PUBLIC`。
- [ ] ACL 可赋予 `READ/WRITE/EXECUTE/MANAGE/SHARE`。
- [ ] 无权限用户访问资源返回拒绝，并包含明确 `messageKey`。
- [ ] `PRIVATE` 输入默认不得直接产生 `PUBLIC` 输出（除非策略放开且权限满足）。

## 7. Workflow/Run 回放验收

- [ ] WorkflowTemplate 支持 Draft/Published 版本。
- [ ] WorkflowRun/StepRun 状态机符合约定（含 failed/canceled/retry）。
- [ ] 可查询 step 输入输出摘要与产物引用。
- [ ] 回放时可叠加节点状态与耗时信息。

## 8. Plugin Market 验收

- [ ] 插件包可上传、安装、启用、禁用。
- [ ] 升级与回滚路径可执行。
- [ ] 依赖缺失时返回可理解错误（含 `messageKey`）。
- [ ] 权限 ceiling 生效，超界安装/启用会被拒绝并审计。

## 9. Stream + MediaMTX 验收

- [ ] 可创建/更新/删除 StreamingAsset/path。
- [ ] 可执行录制开始与停止，录制结果资产化并建立 lineage。
- [ ] `onPublish` 事件能触发一次 workflow run。
- [ ] 流对象的 visibility/ACL 判定与其他对象一致。

## 10. 前端主题与国际化验收

- [ ] 前端使用 Vue + Vite + TypeScript + TailwindCSS。
- [ ] 深色/浅色主题可手动切换并持久化（至少会话级）。
- [ ] `vue-i18n` 至少提供 `zh-CN` 与 `en-US`。
- [ ] 后端 `messageKey` 能正确映射到前端本地化文案。
- [ ] 缺失翻译键时有兜底显示策略（键名或默认文案）。

## 11. 审计与可观测性验收

- [ ] 每次 command 执行记录发起人、上下文、授权结果、资源影响。
- [ ] 外发调用记录目的地、策略结果、摘要信息（不泄露敏感原文）。
- [ ] run/step 关联 traceId 可串联查询。

## 12. 结果判定

- [ ] P0 条目（2、4、5、6）全部通过。
- [ ] 其余条目无阻断性失败。
- [ ] 失败项形成缺陷清单并绑定后续里程碑。
