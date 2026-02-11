# Goyais Java Server v0.1 Development Plan

## 1. 周期与组织

- 周期：6 Sprint，双周迭代，共 12 周。
- 团队：2 个后端小队。
- Squad-A：Auth/Security（单应用拓扑 + 动态权限）。
- Squad-B：Command/Domain/Data/Capability。

## 2. Sprint 路线图

| Sprint | 周期 | Squad-A | Squad-B | 验收门禁 | 状态 |
|---|---|---|---|---|---|
| S1 | W1-W2 | 单应用拓扑骨架、OIDC 元数据端点 | Maven 模块骨架、`/healthz` | 单应用可启动 | Done |
| S2 | W3-W4 | password/oidc 登录链路、JWT claims | command baseline + `/api/v1/commands` + error envelope | command 202 合规 | Done |
| S3 | W5-W6 | `policyVersion + Redis invalidation` | authz gate + row-level data permission | 动态权限即时生效 | In Progress (基础落地) |
| S4 | W7-W8 | `single/resource-only` 切换治理 | assets/workflow/shares 同构落地 | 契约字段同构 | Done |
| S5 | W9-W10 | 安全策略收敛与审计增强 | cache/event/messaging/storage provider 切换 | minimal/full 均可跑 | In Progress |
| S6 | W11-W12 | 安全加固与发布策略 | 回归/性能/发布回滚演练 | v0.1 gates 全绿 | Planned |

## 2.1 2026-02-11 执行结果（本次实现）

- 安全：
  - `/api/v1/**` 默认鉴权（健康检查例外）。
  - 支持 `single/resource-only` 拓扑下 OAuth2 端点开关。
  - ExecutionContext 默认从 JWT claims 解析，`GOYAIS_SECURITY_DEV_HEADER_CONTEXT_ENABLED=true` 时可回退 `X-*` 头。
- 持久化：
  - 新增 Flyway 基线迁移 `V1__baseline.sql`。
  - `commands`、`audit_events`、`policies`、`acl_entries` 表落地。
  - `infra-mybatis` 增加命令、审计、策略快照仓储实现。
- 动态权限：
  - `PolicySnapshotProvider` 升级为 Redis 优先 + DB 回源。
  - 保留 Redis invalidation 广播机制与本地缓存失效。
- 测试：
  - 新增 `DynamicAuthorizationGateTest`、`CommandPipelineTest`、`RequestExecutionContextFactoryTest`。

## 2.2 2026-02-11 N4 切片进展（本次实现）

- API：
  - 新增 `GET/POST /api/v1/assets`。
  - 新增 `GET/PATCH/DELETE /api/v1/assets/{assetId}`。
  - 新增 `GET /api/v1/assets/{assetId}/lineage`。
  - 新增 `GET/POST /api/v1/shares`。
  - 新增 `DELETE /api/v1/shares/{shareId}`。
- Command-first：
  - domain sugar 写路径统一映射到 `asset.upload/asset.update/asset.delete/share.create/share.delete`。
  - 新增 `AssetCommandHandler`、`ShareCommandHandler`，并保持 pipeline 统一审计。
- 持久化：
  - 新增 Flyway `V2__assets_shares_schema.sql`。
  - 落地 `assets`、`asset_lineage`，扩展 `acl_entries` 支持 `tenant/workspace/subjectType/expiresAt/createdBy`。
  - 落地 `MybatisAssetRepository`、`MybatisShareRepository`。

## 2.3 2026-02-11 N5 切片进展（本次实现）

- API：
  - 新增 `GET/POST /api/v1/workflow-templates`。
  - 新增 `GET /api/v1/workflow-templates/{templateId}`。
  - 新增 `POST /api/v1/workflow-templates/{templateId}:patch`。
  - 新增 `POST /api/v1/workflow-templates/{templateId}:publish`。
  - 新增 `GET/POST /api/v1/workflow-runs`。
  - 新增 `GET /api/v1/workflow-runs/{runId}`。
  - 新增 `POST /api/v1/workflow-runs/{runId}:cancel`。
  - 新增 `GET /api/v1/workflow-runs/{runId}/steps`。
  - 新增 `GET /api/v1/workflow-runs/{runId}/events`（SSE）。
- Command-first：
  - domain sugar 写路径统一映射到
    `workflow.createDraft/workflow.patch/workflow.publish/workflow.run/workflow.cancel`。
  - 新增 `WorkflowTemplateCommandHandler`、`WorkflowRunCommandHandler`。
- 持久化：
  - 新增 Flyway `V3__workflow_schema.sql`。
  - 落地 `workflow_templates/workflow_template_versions/workflow_runs/step_runs/workflow_run_events`。
  - 落地 `MybatisWorkflowTemplateRepository`、`MybatisWorkflowRunRepository`。

## 2.4 2026-02-11 N6 第一批进展（本次实现）

- 测试补齐：
  - 新增 `WorkflowPatchApplierTest`（graph patch 操作语义覆盖）。
  - 新增 `WorkflowTemplateCommandHandlerTest`（create/patch/publish 权限与约束覆盖）。
  - 新增 `WorkflowRunCommandHandlerTest`（run/cancel 权限与状态约束覆盖）。
- 风险收敛：
  - 覆盖 workflow domain sugar 的关键 deny/allow 路径，降低权限回归风险。
  - 覆盖 patch operations 的受控变更语义，降低图编辑兼容回归风险。

## 2.5 2026-02-11 N6 第二批进展（本次实现）

- 安全拓扑测试补齐：
  - 新增 `ApiSecuritySingleModeIntegrationTest`，覆盖：
    - `single` 模式 `GET /api/v1/healthz` 匿名可访问。
    - `single` 模式 `GET /oauth2/jwks` 可访问。
    - 受保护 API 无 token 返回 401。
    - 已认证但业务拒绝返回 403 且统一 error envelope。
  - 新增 `ApiSecurityResourceOnlyModeIntegrationTest`，覆盖：
    - `resource-only` 模式 `GET /oauth2/jwks` 不可用（4xx）。
    - 受保护 API 无 token 返回 401。
- 安全链路修复：
  - 修复 `single` 模式双过滤链冲突：授权链增加端点级 `securityMatcher`，避免与 API 链同时匹配 `anyRequest`。
  - 显式启用 `oauth2AuthorizationServer` 默认配置，确保 OAuth2/OIDC 标准端点在 `single` 模式可用。

## 2.6 2026-02-12 N6 第三批进展（本次实现）

- 错误语义修复：
  - 修复未匹配路由被错误映射为 500 的问题，新增 `NoResourceFoundException -> 404 NOT_FOUND` 映射。
  - 新增 `HttpRequestMethodNotSupportedException -> 405 METHOD_NOT_ALLOWED` 映射。
- 安全集成测试补齐：
  - `ApiSecuritySingleModeIntegrationTest` 新增缺失路由断言，验证 404 统一 envelope。
  - `ApiSecurityResourceOnlyModeIntegrationTest` 新增不支持 HTTP 方法断言，验证 405 统一 envelope。

## 3. 固定 DoD

- API/数据模型/状态机文档与实现同变更同步。
- JavaDoc 与文件头门禁通过。
- 单测+集成测试通过。
- Vue 联调通过且无契约破坏。
- 审计可回查 command/authz/egress/policyVersion。

## 4. 分支与 worktree 规则

- 一 story 一 worktree。
- 分支前缀：`goya/<thread-id>-<topic>`。
- 线程开启必须执行：`bash .agents/skills/goyais-worktree-flow/scripts/create_worktree.sh --topic <topic>`（默认落在 `<repo>/.worktrees/`）。
- 线程收口必须执行：`bash .agents/skills/goyais-worktree-flow/scripts/merge_thread.sh --thread-branch <goya/...>`。
- 禁止手工 `git merge` / `git branch -d` / `git worktree remove` 绕过标准收口流程。
- 提交前执行：
  - `git diff --cached --name-only`
  - `bash go_server/scripts/git/precommit_guard.sh`

## 5. 里程碑

- M0（S1-S2）：单应用框架 + command 入口 + 认证骨架。
- M1（S3-S4）：动态权限闭环 + 核心业务域。
- M2（S5）：通用能力封装 + provider 切换。
- M3（S6）：稳定性、发布、灰度就绪。

## 6. 风险与缓解

- 风险：单应用与 resource-only 配置漂移。
  - 缓解：双模式启动测试 + healthz 拓扑标识。
- 风险：动态权限缓存不一致。
  - 缓解：Redis invalidation + 本地 fallback + 演练脚本。
- 风险：注释规范回归失败。
  - 缓解：`java_javadoc_check.sh` 纳入统一回归。
