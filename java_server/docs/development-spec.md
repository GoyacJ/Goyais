# Goyais Java Server v0.1 Development Spec

## 1. 目标与约束 | Goals and Constraints

- 对齐 PRD：`/Users/goya/Repo/Git/Goyais/docs/prd.md`。
- 对齐 Go 契约语义（Command-first、错误模型、分页、状态机、可见性/ACL）。
- Java 默认单应用拓扑：`app-api-server` 同进程承载 Resource + Auth。
- 保留 `resource-only` 模式，支持多资源服务器外接统一授权中心。
- 最小运行模式固定为：PostgreSQL + Redis + Local File Store。
- 数据权限 v0.1 固定为行级（tenant/workspace/visibility/acl 过滤）。

## 2. 模块架构 | Maven Multi-Module Architecture

- `bom`: 统一依赖版本管理。
- `parent`: 统一构建插件、JDK 与质量门禁。
- `contract-api`: OpenAPI 同构 DTO、错误模型、分页语义模型。
- `kernel-core`: ExecutionContext 与公共协议。
- `kernel-web`: 全局异常与统一响应规范。
- `kernel-security`: 鉴权 SPI + policyVersion 动态权限核心。
- `kernel-mybatis`: MyBatisPlus 基础设施与数据权限拦截器。
- `capability-cache`: Cache/Lock 抽象 + Redis policy invalidation 实现。
- `capability-event`: Domain Event + Outbox 抽象。
- `capability-messaging`: MessageBus 抽象（memory/kafka）。
- `capability-storage`: ObjectStorage 抽象（local/minio/s3）。
- `domain`: 领域模型与状态机对象。
- `application`: Command Pipeline 与应用用例。
- `infra-mybatis`: Mapper/Repository SQL 落地。
- `adapter-rest`: `/api/v1` HTTP 适配层与 domain sugar 映射。
- `app-auth-server`: Auth capability 模块（作为能力被装配，不作为默认启动入口）。
- `app-api-server`: 默认可执行应用（single mode）。

## 3. 关键契约 | Public Contract Commitments

- API Prefix：`/api/v1`。
- 写路径：统一 command-first。
- 错误模型：`error: { code, messageKey, details }`。
- 已落地业务面（2026-02-11）：
  - `commands*`
  - `assets*`
  - `shares*`
- 鉴权默认：除健康检查外，`/api/v1/**` 默认要求 Bearer Token。
- 授权顺序：Tenant -> Visibility -> ACL -> RBAC -> Egress。
- 统一执行上下文：`tenantId/workspaceId/userId/roles/policyVersion/traceId`。

## 4. 安全与认证 | Security and Authentication

- Spring Security + Spring Authorization Server。
- OAuth2.1 + OIDC。
- 单应用模式：同进程暴露 OAuth2/OIDC 标准端点与 `/api/v1/*`。
- `resource-only` 模式：关闭授权端点，仅保留资源服务器能力。
- 登录方式：password/sms/oidc/social（v0.1 先落地 password+oidc 基础链路）。
- JWT claims 对齐 ExecutionContext。
- 开发回退：仅当 `GOYAIS_SECURITY_DEV_HEADER_CONTEXT_ENABLED=true` 时允许 `X-*` 请求头构建 ExecutionContext。

## 5. 动态权限与数据权限 | Dynamic AuthZ + Data Permission

- 动态权限：`policyVersion` + Redis invalidation。
- 缓存优先级：Redis -> PostgreSQL policy store -> 本地内存 fallback。
- 失效通道默认：`goyais:policy:invalidate`。
- 失效事件字段：`tenantId/workspaceId/userId/policyVersion/traceId/changedAt`。
- 行级数据权限：SQL 层注入过滤，不在 handler/service 手写分支。
- 行级查询通过 `DataPermissionResolver` 在 repository SQL 中生成谓词。

## 6. 通用能力封装 | Capability Wrappers

- Cache：`CacheFacade` + `LockFacade`。
- Policy invalidation：`PolicyInvalidationPublisher` + `PolicyInvalidationSubscriber`。
- Policy snapshot：`PolicySnapshotProvider` + `PolicySnapshotStore`（Redis 缓存 + DB 回源）。
- Event：`DomainEventPublisher` + Outbox。
- Messaging：`MessageBus`（memory default, kafka optional）。
- Storage：`ObjectStorage`（local default, minio/s3 optional）。

## 7.1 数据持久化落地（2026-02-11）

- Flyway 迁移脚本：`app-api-server/src/main/resources/db/migration/V1__baseline.sql`。
- 增量迁移脚本：`app-api-server/src/main/resources/db/migration/V2__assets_shares_schema.sql`。
- 基线表：
  - `commands`
  - `audit_events`
  - `policies`
  - `acl_entries`
- 本次新增：
  - `assets`
  - `asset_lineage`
  - `acl_entries` 扩展字段：`tenant_id/workspace_id/subject_type/expires_at/created_by`
- 命令与审计落地实现：
  - `MybatisCommandRepository`
  - `MybatisAuditEventStore`
  - `MybatisPolicySnapshotStore`
- 资产与分享落地实现：
  - `MybatisAssetRepository`
  - `MybatisShareRepository`

## 7. 配置策略 | Configuration Strategy

- 优先级：`ENV > YAML > default`。
- 统一命名：`GOYAIS_*`。
- 新增关键配置：
  - `GOYAIS_SECURITY_TOPOLOGY_MODE=single|resource-only`
  - `GOYAIS_SECURITY_DEV_HEADER_CONTEXT_ENABLED=false|true`
  - `GOYAIS_AUTHZ_DYNAMIC_ENABLED=true|false`
  - `GOYAIS_AUTHZ_POLICY_CACHE_TTL=30s`
  - `GOYAIS_AUTHZ_POLICY_INVALIDATION_CHANNEL=goyais:policy:invalidate`
  - `GOYAIS_DB_URL`
  - `GOYAIS_DB_USERNAME`
  - `GOYAIS_DB_PASSWORD`
  - `GOYAIS_DB_FLYWAY_ENABLED`
  - `GOYAIS_RESOURCE_SERVER_JWK_SET_URI`
  - `GOYAIS_STORAGE_PROVIDER`
  - `GOYAIS_STORAGE_LOCAL_ROOT`
  - `GOYAIS_STORAGE_BUCKET`
  - `GOYAIS_FEATURE_ASSET_LIFECYCLE`
  - `GOYAIS_FEATURE_ACL_ROLE_SUBJECT`

## 8. 质量门禁 | Quality Gates

- 文件头检查：`bash go_server/scripts/ci/source_header_check.sh`
- JavaDoc 检查：`bash java_server/scripts/ci/java_javadoc_check.sh`
- Java 构建：`mvn -f java_server/pom.xml -DskipTests verify`
- Java 测试：`mvn -f java_server/pom.xml test`
- 关键单测覆盖：
  - `DynamicAuthorizationGateTest`
  - `CommandPipelineTest`
  - `RequestExecutionContextFactoryTest`
