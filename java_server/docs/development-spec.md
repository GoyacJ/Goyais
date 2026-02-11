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
- 授权顺序：Tenant -> Visibility -> ACL -> RBAC -> Egress。
- 统一执行上下文：`tenantId/workspaceId/userId/roles/policyVersion/traceId`。

## 4. 安全与认证 | Security and Authentication

- Spring Security + Spring Authorization Server。
- OAuth2.1 + OIDC。
- 单应用模式：同进程暴露 OAuth2/OIDC 标准端点与 `/api/v1/*`。
- `resource-only` 模式：关闭授权端点，仅保留资源服务器能力。
- 登录方式：password/sms/oidc/social（v0.1 先落地 password+oidc 基础链路）。
- JWT claims 对齐 ExecutionContext。

## 5. 动态权限与数据权限 | Dynamic AuthZ + Data Permission

- 动态权限：`policyVersion` + Redis invalidation。
- 缓存优先级：Redis -> 本地内存 fallback。
- 失效通道默认：`goyais:policy:invalidate`。
- 失效事件字段：`tenantId/workspaceId/userId/policyVersion/traceId/changedAt`。
- 行级数据权限：SQL 层注入过滤，不在 handler/service 手写分支。

## 6. 通用能力封装 | Capability Wrappers

- Cache：`CacheFacade` + `LockFacade`。
- Policy invalidation：`PolicyInvalidationPublisher` + `PolicyInvalidationSubscriber`。
- Event：`DomainEventPublisher` + Outbox。
- Messaging：`MessageBus`（memory default, kafka optional）。
- Storage：`ObjectStorage`（local default, minio/s3 optional）。

## 7. 配置策略 | Configuration Strategy

- 优先级：`ENV > YAML > default`。
- 统一命名：`GOYAIS_*`。
- 新增关键配置：
  - `GOYAIS_SECURITY_TOPOLOGY_MODE=single|resource-only`
  - `GOYAIS_AUTHZ_DYNAMIC_ENABLED=true|false`
  - `GOYAIS_AUTHZ_POLICY_CACHE_TTL=30s`
  - `GOYAIS_AUTHZ_POLICY_INVALIDATION_CHANNEL=goyais:policy:invalidate`

## 8. 质量门禁 | Quality Gates

- 文件头检查：`bash go_server/scripts/ci/source_header_check.sh`
- JavaDoc 检查：`bash java_server/scripts/ci/java_javadoc_check.sh`
- Java 构建：`mvn -f java_server/pom.xml -DskipTests verify`
- Java 测试：`mvn -f java_server/pom.xml test`
