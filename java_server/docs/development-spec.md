# Goyais Java Server v0.1 Development Spec

## 1. 目标与约束 | Goals and Constraints

- 对齐 PRD：`/Users/goya/Repo/Git/Goyais/docs/prd.md`。
- 对齐 Go 契约语义（Command-first、错误模型、分页、状态机、可见性/ACL）。
- Java 采用双应用拓扑：`app-auth-server` + `app-api-server`。
- 最小运行模式固定为：PostgreSQL + Redis + Local File Store。
- 数据权限 v0.1 固定为行级（tenant/workspace/visibility/acl 过滤）。

## 2. 模块架构 | Maven Multi-Module Architecture

- `bom`: 统一依赖版本管理。
- `parent`: 统一构建插件、JDK 与质量门禁。
- `contract-api`: OpenAPI 同构 DTO、错误模型、分页语义模型。
- `kernel-core`: ExecutionContext 与公共协议。
- `kernel-web`: 全局异常与统一响应规范。
- `kernel-security`: 鉴权 SPI（RBAC/ACL/Egress）。
- `kernel-mybatis`: MyBatisPlus 基础设施与数据权限拦截器。
- `capability-cache`: Cache/Lock 抽象（Redisson + Spring Cache 预留）。
- `capability-event`: Domain Event + Outbox 抽象。
- `capability-messaging`: MessageBus 抽象（memory/kafka）。
- `capability-storage`: ObjectStorage 抽象（local/minio/s3）。
- `domain`: 领域模型与状态机对象。
- `application`: Command Pipeline 与应用用例。
- `infra-mybatis`: Mapper/Repository SQL 落地。
- `adapter-rest`: `/api/v1` HTTP 适配层与 domain sugar 映射。
- `app-api-server`: Resource Server 业务入口。
- `app-auth-server`: Authorization Server 认证授权入口。

## 3. 关键契约 | Public Contract Commitments

- API Prefix：`/api/v1`。
- 写路径：统一 command-first。
- 错误模型：`error: { code, messageKey, details }`。
- 授权顺序：Tenant -> Visibility -> ACL -> RBAC -> Egress。
- 统一执行上下文：`tenantId/workspaceId/userId/roles/policyVersion/traceId`。

## 4. 安全与认证 | Security and Authentication

- Spring Security + Spring Authorization Server。
- OAuth2.1 + OIDC。
- 登录方式：password/sms/oidc/social。
- JWT claims 对齐 ExecutionContext。
- 动态权限：`policyVersion` + Redis 缓存失效策略。

## 5. 数据与持久化 | Data and Persistence

- DB：PostgreSQL（v0.1 基线）。
- ORM：MyBatisPlus。
- Migration：Flyway（forward + rollback scripts）。
- 行级数据权限：SQL 层注入过滤，不在 handler/service 手写分支。
- ACL：JSONB 存储，统一 operator 判定。

## 6. 通用能力封装 | Capability Wrappers

- Cache：`CacheFacade` + `LockFacade`。
- Event：`DomainEventPublisher` + Outbox。
- Messaging：`MessageBus`（memory default, kafka optional）。
- Storage：`ObjectStorage`（local default, minio/s3 optional）。

## 7. 配置策略 | Configuration Strategy

- 优先级：`ENV > YAML > default`。
- 统一命名：`GOYAIS_*`。
- profile:
  - minimal: postgres + redis + local
  - full: postgres + redis + kafka + minio/s3
- Feature gates 与 Go 语义对齐，支持快速回滚。

## 8. 设计阶段交付门禁 | Design-Phase Gate Deliverables

- API 草案：`java_server/docs/api/openapi-java-draft.yaml`
- 架构草案：`java_server/docs/arch/overview.md`
- 数据模型草案：`java_server/docs/arch/data-model.md`
- 状态机草案：`java_server/docs/arch/state-machines.md`
- 验收草案：`java_server/docs/acceptance.md`
- 开发规格：`java_server/docs/development-spec.md`
- 开发计划：`java_server/docs/development-plan.md`
