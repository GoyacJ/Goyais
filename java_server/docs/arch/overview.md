# Java Server Architecture Overview (v0.1)

## 1. Runtime Topology

- `app-auth-server`: OAuth2.1/OIDC Authorization Server。
- `app-api-server`: Resource Server + `/api/v1` business APIs。
- Shared frontend: `vue_web`。

## 2. Layered Architecture

1. Access Layer: REST + SSE。
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
- 高危动作默认策略收敛（PUBLIC、跨租户共享、策略更新）。

## 5. Deployment Profiles

- minimal: postgres + redis + local
- full: postgres + redis + kafka + minio/s3

## 6. Contract Commitment

- 路由、错误、分页、状态机语义与 Go 同构。
- Java 新增能力不得破坏 `vue_web` 契约一致性。
