# Java Server Architecture Overview (v0.1)

## 1. Runtime Topology

- 默认：`app-api-server` 单应用运行（Auth + Resource 同进程）。
- 扩展：`resource-only` 模式用于水平扩容多个资源服务器。
- Shared frontend: `vue_web`。

## 2. Layered Architecture

1. Access Layer: REST + SSE + OAuth2/OIDC endpoints。
2. Application Layer: Command pipeline + use cases。
3. Domain Layer: aggregate/state machine/policy。
4. Infrastructure Layer: MyBatis/Flyway/Redis/ObjectStorage/MessageBus。

## 3. Command Pipeline

1. Validate
2. Authorize (Tenant -> Visibility -> ACL -> RBAC -> Egress)
3. Execute
4. Audit
5. Event

## 4. Security Model

- Agent-as-User execution context。
- JWT claims 必含 `tenantId/workspaceId/userId/roles/policyVersion/traceId`。
- 动态权限：`policyVersion + Redis invalidation`。
- Redis 不可用时降级本地缓存并标记 `healthz` degraded。

## 5. Deployment Profiles

- minimal: postgres + redis + local
- full: postgres + redis + kafka + minio/s3

## 6. Contract Commitment

- 路由、错误、分页、状态机语义与 Go 同构。
- Java 新增能力不得破坏 `vue_web` 契约一致性。
