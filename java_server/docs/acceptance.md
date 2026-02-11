# Java Server v0.1 Acceptance Draft

## 1. Contract Compatibility

- `/api/v1` prefix fixed.
- error envelope fixed: `error: { code, messageKey, details }`.
- write path follows command-first.

## 2. Topology Mode

- `single` 模式下：
  - `GET /api/v1/healthz` 可用。
  - OAuth2/OIDC metadata endpoints 可用。
- `resource-only` 模式下：
  - `/api/v1/*` 可用。
  - OAuth2/OIDC metadata endpoints 不可用（deny/404）。

## 3. Minimal Runtime

- PostgreSQL + Redis + Local file store starts successfully.
- `GET /api/v1/healthz` and `GET /api/v1/system/healthz` return provider readiness.
- Flyway migration creates `commands/audit_events/policies/acl_entries/assets/asset_lineage`.

## 4. Security and Auth

- OAuth2.1/OIDC usable for web login。
- Password/OIDC login paths pass e2e checks。
- JWT claims map to ExecutionContext without drift。
- `GET /api/v1/commands` without token returns 401。
- `GOYAIS_SECURITY_DEV_HEADER_CONTEXT_ENABLED=false` 时，`X-*` 头不会绕过认证。

## 5. Dynamic Authorization

- Authorization order fixed: Tenant -> Visibility -> ACL -> RBAC -> Egress。
- `policyVersion` 过期请求会在无重启场景下按新策略生效。
- Redis invalidation 广播可使多节点权限缓存同步失效。

## 6. Data Scope

- Row-level data permission SQL filtering is enforced。
- owner/WORKSPACE/ACL.READ 三类路径过滤语义正确。
- SQL 过滤谓词由 `DataPermissionResolver` 在 repository 层注入。

## 7. Capability Wrappers

- cache/event/messaging/storage SPI are available and swappable.
- memory/local fallback paths work when kafka/minio/s3 are unavailable.

## 7.1 Assets and Shares

- `assets`:
  - `POST /api/v1/assets` returns 202 and `WriteResponseAsset` payload.
  - `GET /api/v1/assets` returns deterministic ordering `created_at DESC, id DESC`.
  - `PATCH/DELETE /api/v1/assets/{assetId}` obey `GOYAIS_FEATURE_ASSET_LIFECYCLE`.
  - `GET /api/v1/assets/{assetId}/lineage` returns `assetId + edges`.
- `shares`:
  - `POST /api/v1/shares` returns 202 and `WriteResponseShare`.
  - `DELETE /api/v1/shares/{shareId}` returns 202 and `status=deleted`.
  - `GOYAIS_FEATURE_ACL_ROLE_SUBJECT=false` 时，`subjectType=role` 返回 `INVALID_SHARE_REQUEST`.

## 8. Comment and CI Gates

- `bash go_server/scripts/ci/source_header_check.sh` 通过。
- `bash java_server/scripts/ci/java_javadoc_check.sh` 通过。
- `bash go_server/scripts/ci/contract_regression.sh` 通过。
- `mvn -f java_server/pom.xml test` 通过（含 `DynamicAuthorizationGateTest`、`CommandPipelineTest`、`RequestExecutionContextFactoryTest`）。

## 9. Rollback

- Feature gates can disable high-risk domains without full rollback.
- DB migration rollback scripts are executable.
