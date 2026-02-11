# Goyais v0.1 验收清单

> 说明：本清单用于 v0.1 文档到实现阶段的统一验收。所有条目默认在同一租户内执行。

## 1. 基础验收条件

- [x] API 前缀统一为 `/api/v1`。
- [x] 所有副作用动作可通过 `/api/v1/commands` 表达并追踪（含 domain sugar：assets/workflow/shares）。
- [x] 错误模型统一为 `error: { code, messageKey, details }`。
- [x] 关键对象（commands/assets/workflow templates+runs+steps）包含通用字段：`id/tenantId/workspaceId/ownerId/visibility/acl/createdAt/updatedAt/status`。

## 2. 最小化运行模式验收（无外部依赖）

### 2.1 启动配置
- [x] 使用 `sqlite + mediamtx + local object store + memory cache` 组合可启动。
- [x] 在无 Postgres/Redis/MinIO 条件下，系统仍可完成基础闭环。

### 2.2 闭环能力
- [x] 上传一个资产后可查询到元数据。
- [x] 创建并运行一个最小 workflow（至少 1 个 step）可产生 run 记录。
- [x] run/step 状态可查询，且审计中可看到对应 command。

## 3. Provider 切换验收

### 3.1 DB provider
- [x] `db.driver=sqlite` 时迁移与查询可用。
- [x] `db.driver=postgres` 时迁移与查询可用。
- [x] 两种模式字段语义一致（时间/状态/visibility/acl 不漂移）。

### 3.2 Object store provider
- [x] `object_store.provider=local` 可上传/读取/删除。
- [x] `object_store.provider=minio` 可上传/读取/删除。
- [x] `object_store.provider=s3` 可上传/读取/删除。

### 3.3 Cache + Vector provider
- [x] `cache.provider=memory` 下系统可运行。
- [x] `cache.provider=redis`（含 `cache.redis_password`）下缓存命中与过期符合预期。
- [x] `vector.provider=redis_stack`（含 `vector.redis_password`）可写入并检索向量。
- [x] 无 Redis 时 `vector.provider=sqlite` fallback 可检索。

### 3.4 Event Bus provider
- [x] `event_bus.provider=memory|kafka` 配置可解析，且遵循 `ENV > YAML > 默认值`。
- [x] `GET /api/v1/healthz` / `GET /api/v1/system/healthz` 返回 `details.providers.event_bus.status`（`ready/degraded`）。

## 4. Single Binary Packaging 验收

### 4.1 构建与独立运行
- [x] 执行 `make build` 后产出单个可执行文件。
- [x] 改名或删除 `vue_web/dist`（可选强化：改名或删除 `vue_web/`）后，启动该二进制。
- [x] 访问 `/` 返回 200。
- [x] 访问 `/canvas` 返回 200（SPA fallback 生效）。
- [x] 访问 `/api/v1/healthz` 返回 200。

### 4.2 静态路由与特殊路径
- [x] `/api/v1/*` 不被 SPA fallback 覆盖。
- [x] 未提供占位文件时，`/favicon.ico` 返回 404。
- [x] 未提供占位文件时，`/robots.txt` 返回 404。

### 4.3 响应头与类型
- [x] `GET /` 返回 `Content-Type: text/html`（可含 charset）。
- [x] `GET /canvas` 返回 `Content-Type: text/html`（可含 charset）。
- [x] `GET /` 与 `GET /canvas` 的 `Cache-Control` 精确为 `no-store`。
- [x] 首页引用的 `/assets/*.js` 返回 JavaScript 类型（`application/javascript` 或兼容值）。
- [x] 静态文件不出现 `application/octet-stream` 误配（除确实二进制资源外）。

## 5. Command-first 与 AI/UI 一致性验收

- [x] UI 触发 `workflow.run` 与 AI 触发同动作时，落库 command 形态一致。
- [x] Domain 写接口响应包含：`resource + commandRef { commandId, status, acceptedAt }`。
- [x] `GET /api/v1/commands` 与 `GET /api/v1/commands/{commandId}` 返回 `acceptedAt` 与可追踪的 `traceId`。
- [x] 通过 `GET /api/v1/commands/{commandId}` 可追踪最终执行结果。

### 5.1 A2 最小闭环（Thread #3）
- [x] `POST /api/v1/commands`（携带 `X-Tenant-Id/X-Workspace-Id/X-User-Id`）返回 `202` 且包含 `resource + commandRef`。
- [x] 缺少任一上下文 header 返回 `400`，错误为 `MISSING_CONTEXT + error.context.missing`，并在 `details.missingHeaders` 返回缺失项列表。
- [x] `GOYAIS_AUTH_CONTEXT_MODE=jwt_or_header` 时，携带有效 Bearer JWT 可在不提供 `X-Tenant-Id/X-Workspace-Id/X-User-Id` 的情况下创建 command。
- [x] `GOYAIS_AUTH_CONTEXT_MODE=jwt_or_header` 时，`X-Tenant-Id/X-Workspace-Id/X-Roles` 仅允许在 JWT claims 可访问范围内覆盖，越权返回 `403 FORBIDDEN + error.authz.forbidden`。
- [x] Bearer JWT 非法时返回 `400 INVALID_TOKEN + error.context.invalid_token`。
- [x] `GOYAIS_AUTH_CONTEXT_MODE=header_only` 时，携带 JWT 但缺少上下文 header 仍返回 `400 MISSING_CONTEXT`。
- [x] `GET /api/v1/commands` 返回 `items`，并固定按 `created_at DESC, id DESC` 排序。
- [x] cursor 模式 token 基于 `(created_at,id)`，若请求带 `cursor` 则忽略 `page/pageSize`。
- [x] 同 `(tenant,workspace,owner,idempotency_key)` 且同请求哈希复用同一 `commandId`。
- [x] 同 `(tenant,workspace,owner,idempotency_key)` 但不同请求哈希返回 `409 IDEMPOTENCY_KEY_CONFLICT`。
- [x] `Idempotency-Key` 缺失时仍可创建新命令，并保留审计记录。
- [x] SQLite（minimal）可完成 create/get/list + 状态流转 + 审计落库。
- [x] Postgres（full）可连接并在 healthz 回显 provider；commands 业务接口与 sqlite 语义等价（非 `501` 占位）。

## 6. Visibility/ACL 与隔离验收

- [x] 已实现对象（commands/assets/workflow/shares）支持 `PRIVATE/WORKSPACE/TENANT/PUBLIC`。
- [x] 已实现对象（commands/assets/workflow/shares/registry/plugin/stream）按当前阶段支持 visibility/ACL 判定。
- [x] ACL 可赋予 `READ/WRITE/EXECUTE/MANAGE/SHARE`。
- [x] 无权限用户访问资源返回拒绝，并包含明确 `messageKey`。
- [x] `PRIVATE` 输入默认不得直接产生 `PUBLIC` 输出（除非策略放开且权限满足）。

### 6.1 A3 最小闭环（Thread #4）
- [x] `POST /api/v1/shares` 仅允许 `resourceType=command|asset`，其他值返回 `400 INVALID_SHARE_REQUEST`。
- [x] `POST /api/v1/shares` 仅允许 `subjectType=user` 且 `permissions` 仅来自 `READ/WRITE/EXECUTE/MANAGE/SHARE`，非法值返回 `400 INVALID_SHARE_REQUEST`。
- [x] `POST /api/v1/shares` 创建前必须校验同资源 SHARE 权限：owner 或该资源上已有 `ACL.SHARE`。
- [x] 非 owner 且无该资源 `SHARE` 权限时，`POST /api/v1/shares` 返回 `403 FORBIDDEN + messageKey=error.authz.forbidden`。
- [x] `POST /api/v1/shares` 与 `DELETE /api/v1/shares/{shareId}` 走 command-first，返回 `202 + resource + commandRef`，且可由 `GET /api/v1/commands/{commandId}` 追踪。
- [x] SQLite 模式下，`GET /api/v1/commands` 的可读过滤在 SQL 层完成（`owner OR visibility=WORKSPACE OR ACL.READ`），分页基于过滤后结果且排序固定 `created_at DESC,id DESC`。

## 7. Workflow/Run 回放验收

- [x] WorkflowTemplate 支持 Draft/Published 版本。
- [x] WorkflowRun/StepRun 状态机符合约定（含 failed/canceled/retry）。
- [x] 可查询 step 输入输出摘要与产物引用。
- [x] 回放时可叠加节点状态与耗时信息。

## 7.1 Registry C1 Read-only 验收

- [x] `GET /api/v1/registry/capabilities` 不再返回 `501`，返回 `200 + items + pageInfo/cursorInfo`。
- [x] `GET /api/v1/registry/capabilities/{capabilityId}` 对不存在资源返回 `404` + `messageKey=error.registry.not_found`。
- [x] `GET /api/v1/registry/algorithms`、`GET /api/v1/registry/providers` 不再返回 `501`，分页语义与全局约定一致（`cursor` 优先）。
- [x] SQLite/Postgres 下 registry 读接口均可用，且保持 tenant/workspace 隔离与 ACL.READ 过滤语义。

## 8. Plugin Market 验收

- [x] 插件包可上传、安装、启用、禁用。
- [x] 升级与回滚路径可执行。
- [x] 依赖缺失时返回可理解错误（含 `messageKey`）。
- [x] 权限 ceiling 生效，超界安装/启用会被拒绝并审计。

## 8.1 Algorithm Library MVP 验收

- [x] `GET /api/v1/registry/algorithms/{algorithmId}` 可查询算法详情（非 501）。
- [x] `POST /api/v1/algorithms/{algorithmId}:run` 走 command-first，返回 `202 + resource + commandRef`。
- [x] `commandType=algorithm.run` 可通过 `GET /api/v1/commands/{commandId}` 完整回查。
- [x] 至少 2 个 `algo-pack` 安装后可运行，且每次运行输出结构化结果 + 至少 1 个资产。
- [x] `algorithm.run` 结果与 workflow 执行链路一致，包含 `workflowRunId` 追踪关联。

## 9. Stream + MediaMTX 验收

- [x] 可创建并查询 StreamingAsset/path（`POST /streams`、`GET /streams*`）。
- [x] 可执行录制开始与停止，录制结果资产化并建立 lineage。
- [x] `onPublish` 事件能触发一次 workflow run（经 command gate）。
- [x] 在 `event_bus.provider=kafka` 下，`stream.on_publish` 消费链路通过 command gate 触发 `workflow.run`，重复事件按 `stream-onpublish-<recordingId>` 幂等收敛。
- [x] 流对象的 visibility/ACL 判定与其他对象一致（owner/ACL.READ；当策略允许提升可见性时支持 WORKSPACE 读，写动作受 EXECUTE/MANAGE 约束）。
- [x] `GOYAIS_FEATURE_STREAM_CONTROL_PLANE=false` 时，`stream.updateAuth`、`stream.delete` 以及对应 domain sugar 路径返回 `501 NOT_IMPLEMENTED`（含 `/api/v1/commands` 直调）。

## 10. 前端主题与国际化验收

- [x] 前端使用 Vue + Vite + TypeScript + TailwindCSS。
- [x] 深色/浅色主题可手动切换并持久化（至少会话级）。
- [x] `vue-i18n` 至少提供 `zh-CN` 与 `en-US`。
- [x] 后端 `messageKey` 能正确映射到前端本地化文案。
- [x] 缺失翻译键时有兜底显示策略（键名或默认文案）。
- [x] 多布局模式 `console/topnav/focus` 可切换，且 `auto` 能按路由默认生效。
- [x] 三布局在 desktop 下都支持窗口拖拽/缩放/置顶；mobile 自动降级为单列卡片。
- [x] 窗口布局按 `route+layout` 独立持久化；切换布局或路由不会污染彼此状态。
- [x] `pnpm -C /Users/goya/Repo/Git/Goyais/vue_web test --run` 通过（包含 layout/window 核心用例）。
- [x] 本轮执行 `bash .agents/skills/goyais-vue-asset-governance/scripts/validate-assets.sh` 并通过（若有素材变更同样适用）。

## 11. 审计与可观测性验收

- [x] 每次 command 执行记录发起人、上下文、授权结果、资源影响。
- [x] 外发调用记录目的地、策略结果、摘要信息（不泄露敏感原文）。
- [x] run/step 关联 traceId 可串联查询。

## 12. B1 Asset 最小闭环验收（Thread #5）

### 12.1 SQLite minimal（必须通过）
- [x] `POST /api/v1/assets` 使用 multipart 上传成功，返回 `202`，响应包含 `resource + commandRef`。
- [x] owner 访问 `GET /api/v1/assets/{assetId}` 返回 `200`。
- [x] 非 owner 且无 share 时访问 `GET /api/v1/assets/{assetId}` 返回 `403 FORBIDDEN` + `messageKey=error.authz.forbidden`。
- [x] owner 对同一 `asset` 创建 `READ` share 后，非 owner 访问 `GET /api/v1/assets/{assetId}` 返回 `200`。
- [x] `GET /api/v1/assets` 在 SQL 层完成可读过滤（tenant/workspace 限定 + owner/WORKSPACE/ACL.READ），并保持 `created_at DESC,id DESC` 稳定排序。
- [x] cursor 模式下 `cursor` 优先于 `page/pageSize`，分页无重复/漏项。

### 12.2 Shares（asset）规则（必须通过）
- [x] `POST /api/v1/shares` 支持 `resourceType=asset`，并沿用同资源 `SHARE` 权限校验。
- [x] `subjectType` 仅支持 `user`；非法值返回 `400 INVALID_SHARE_REQUEST`。
- [x] `permissions` 仅支持 `READ/WRITE/EXECUTE/MANAGE/SHARE`；非法值返回 `400 INVALID_SHARE_REQUEST`。
- [x] 非 owner 且无 `asset` 的 `SHARE` 权限时，`POST /api/v1/shares` 返回 `403 FORBIDDEN + messageKey=error.authz.forbidden`。

### 12.3 Postgres full（本轮收口）
- [x] `GET /api/v1/healthz` 与 `GET /api/v1/system/healthz` 返回 `200`，且 `providers.db=postgres`。
- [x] `POST/GET /api/v1/assets*`、`/commands*`、`/workflow*`、`/shares*` 与 sqlite 语义等价（非 `501` 占位）。

### 12.4 回归（必须通过）
- [x] `make build` 通过。
- [x] `verify_single_binary.sh` 返回 `0`（含 no-store、favicon/robots 404、JS Content-Type、移除 vue_web/dist 后可运行）。

### 12.5 Asset 生命周期（feature gate）
- [x] `GOYAIS_FEATURE_ASSET_LIFECYCLE=true` 时，`PATCH /api/v1/assets/{assetId}` 走 command-first，返回 `202 + resource + commandRef`。
- [x] `GOYAIS_FEATURE_ASSET_LIFECYCLE=true` 时，`DELETE /api/v1/assets/{assetId}` 走 command-first，返回 `202 + resource + commandRef`，资源状态为 `deleted`。
- [x] `GOYAIS_FEATURE_ASSET_LIFECYCLE=true` 时，`GET /api/v1/assets/{assetId}/lineage` 返回 `200` 与 lineage edges。
- [x] `GOYAIS_FEATURE_ASSET_LIFECYCLE=false`（默认）时，上述三条路径返回 `501 NOT_IMPLEMENTED`，用于安全回滚。

## 13. 结果判定

- [x] P0 条目（2、4、5、6）全部通过。
- [x] 其余条目无阻断性失败（M2 占位项已标注 deferred）。
- [x] 失败项形成缺陷清单并绑定后续里程碑（见 7/8/9/11 与 M2 规划）。
- [x] “v0.1 冻结验收通过”与“PRD 严格口径不满足”可并存：冻结验收用于当前里程碑收口，PRD 严格口径用于后续重构闭环判定。

## 14. 本轮证据命令（2026-02-10）

- `go test ./...`
- `GOYAIS_IT_POSTGRES_DSN='<dsn>' go test ./internal/integration -run TestPostgresCommandAssetWorkflowContract -v`
- `GOYAIS_IT_OBJECT_STORE_ENDPOINT=<endpoint> GOYAIS_IT_OBJECT_STORE_ACCESS_KEY=<ak> GOYAIS_IT_OBJECT_STORE_SECRET_KEY=<sk> GOYAIS_IT_OBJECT_STORE_BUCKET=<bucket> GOYAIS_IT_OBJECT_STORE_USE_SSL=false go test ./internal/asset -run TestS3CompatibleStoreIntegration -v`
- `GOYAIS_IT_REDIS_ADDR='<host:port>' GOYAIS_IT_REDIS_PASSWORD='<password>' go test ./internal/platform/cache -run TestRedisProviderIntegration -v`
- `GOYAIS_IT_POSTGRES_DSN='<dsn>' GOYAIS_IT_KAFKA_BROKERS='<host:port>' go test ./internal/integration -run KafkaStreamTrigger -v`
- `go test ./internal/config ./internal/access/http -v`
- `pnpm -C /Users/goya/Repo/Git/Goyais/vue_web typecheck`
- `pnpm -C /Users/goya/Repo/Git/Goyais/vue_web test:run`
- `bash go_server/scripts/ci/source_header_check.sh`
- `bash .agents/skills/goyais-vue-asset-governance/scripts/validate-assets.sh`
- `make build`
- `GOYAIS_VERIFY_BASE_URL=http://127.0.0.1:18080 GOYAIS_START_CMD='GOYAIS_SERVER_ADDR=:18080 ./go_server/build/goyais' bash .agents/skills/goyais-release-regression/scripts/verify_single_binary.sh`
  - 说明：默认 `:8080` 被本机其他进程占用，验收脚本改用 `:18080` 执行并通过。

## 15. 自动化回归与 Git 护栏验收

- [x] 存在统一回归脚本 `go_server/scripts/ci/contract_regression.sh`，串行执行 `path_migration_audit`、`source_header_check`、`go test ./...`、`pnpm -C /Users/goya/Repo/Git/Goyais/vue_web typecheck`、`pnpm -C /Users/goya/Repo/Git/Goyais/vue_web test:run`、`make build`、single-binary verify。
- [x] 存在迁移残留审计脚本 `go_server/scripts/ci/path_migration_audit.sh`，用于扫描旧路径引用是否清零。
- [x] 存在源码头校验脚本 `go_server/scripts/ci/source_header_check.sh`：非 Java 校验 SPDX/版权/作者/时间/版本/描述字段；Java 校验 `SPDX + <p> + @author + @since(yyyy-MM-dd HH:mm:ss)`。
- [x] 存在提交前防呆脚本 `go_server/scripts/git/precommit_guard.sh`，阻断 `data/objects/`、`*.db`、`go_server/build/`、`go_server/internal/access/webstatic/dist/assets/`、`go_server/internal/access/webstatic/dist/index.html`、`vue_web/dist/`、`vue_web/node_modules/` 被 staged。
- [x] 存在 worktree 审计脚本 `go_server/scripts/git/worktree_audit.sh`，阻断 detached worktree 与重复分支绑定。
- [x] CI workflow `.github/workflows/contract-regression.yml` 调用统一回归脚本，作为 PR/master 门禁入口。

## 16. PRD 严格口径重构启动验收（S0-S6）

> 本章节用于“严格口径重构计划”追踪，不覆盖 1-15 的冻结结论。

- [x] S0 契约同步：`openapi/data-model/state-machines/overview/acceptance` 已同步新增 API、commandType、实体口径。
- [x] S0 路由可达：`openapi_reachability` 覆盖新增参数 `sessionId/packageId/bundleId`，新增路径均非 `API_NOT_FOUND`。
- [x] S1 Workflow Engine：完成 DAG 拓扑校验、并发调度、重试退避、Tool Gate 与 run/step 事件流。
- [x] S2 Canvas：满足 PRD 8.9 五条验收（typed ports/minimap/undo-redo/run-from-here/test-node）。
- [x] S2.1 Canvas 图编辑能力：`typed ports`、`minimap`、`undo/redo`、`run from here`、`test node` 已在 `vue_web/src/views/CanvasView.vue` 落地。
- [x] S2.2 Canvas 运行态可视化：节点运行状态/耗时/产物数/错误码与步骤侧栏联动，运行中自动轮询刷新（`vue_web/src/views/CanvasView.vue`、`vue_web/src/components/canvas/TypedPortNode.vue`）。
- [x] S2.3 Canvas 回归：`pnpm -C /Users/goya/Repo/Git/Goyais/vue_web typecheck`、`pnpm -C /Users/goya/Repo/Git/Goyais/vue_web test:run -- src/views/CanvasView.spec.ts src/components/canvas/TypedPortNode.spec.ts` 通过。
- [x] S2.4 Canvas AI patch 闭环：前端通过 `POST /api/v1/ai/plans:preview` 获取 `workflow.patch` 计划，服务端按受控 `operations` 校验并应用，失败返回 `INVALID_WORKFLOW_REQUEST`。
- [x] S3 AI 工作台：会话/turn/计划/执行反馈闭环，且 AI/UI 同动作 command 同形。
- [x] S3.0 AI 计划预览：`POST /api/v1/ai/plans:preview` 可返回 planner/reason/payload/suggestions，且不产生副作用写入。
- [x] S3.1 AI 事件反馈：`/ai/sessions/{id}/events` 输出 `ai.turn.*` + `command.*` + `workflow.*` 摘要事件，turn `commandIds` 绑定真实 commandId。
- [x] S3.2 AI 前端闭环：计划预览、执行反馈时间线、失败错误码展示与会话事件自动轮询已落地（`vue_web/src/views/AIWorkbenchView.vue`）。
- [x] S3.3 AI 回归：`go test ./internal/access/http -run TestAPIContractRegression -count=1`、`pnpm -C /Users/goya/Repo/Git/Goyais/vue_web test:run -- src/views/AIWorkbenchView.spec.ts` 通过。
- [x] S3.4 AI planner 语义增强：新增自然语言 intent strategy（中英关键词）、domain-aware reject reason/alternatives 与 explainability confidence；回归覆盖 `go_server/internal/ai/planner/planner_test.go`。
- [x] S4 MediaMTX 控制面：`update-auth/delete`、录制资产化、onPublish 事件触发 workflow（经 command gate）。
- [x] S4.1 算法库页面闭环：`Algorithm Library` 支持输入 JSON 参数、触发 `algorithm.run`、展示 `workflowRunId/assetIds/commandId` 结果。
- [x] S4.2 Run Center 深化：支持 step 详情输入/输出/错误上下文，并提供 `logRef/artifacts` 引用的复制与新标签打开入口（`vue_web/src/views/RunCenterView.vue`、`vue_web/src/views/RunCenterView.spec.ts`）。
- [x] S4.3 权限管理闭环：支持 ACL 策略授予/撤销（`share.create/share.delete`）、策略列表浏览，以及 share.* 命令审计并行视图（`vue_web/src/views/PermissionManagementView.vue`、`vue_web/src/api/shares.ts`、`vue_web/src/views/PermissionManagementView.spec.ts`）。
- [x] S5 插件市场生命周期：`download/upgrade` 与 `uploaded->validating->installing->enabled` 全链路一致。
- [x] S6 ContextBundle + ACL role：`context-bundles` 读接口与 `acl_entries.subject_type=user|role` 落地。
- [x] 每切片均具备全量回归证据（`go test ./...`、`pnpm -C /Users/goya/Repo/Git/Goyais/vue_web typecheck`、`pnpm -C /Users/goya/Repo/Git/Goyais/vue_web test:run`、`make build`、single-binary verify）。
