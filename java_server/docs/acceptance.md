# Java Server v0.1 Acceptance Draft

## 1. Contract Compatibility

- `/api/v1` prefix fixed.
- error envelope fixed: `error: { code, messageKey, details }`.
- write path follows command-first.

## 2. Minimal Runtime

- PostgreSQL + Redis + Local file store starts successfully.
- `GET /api/v1/healthz` and `GET /api/v1/system/healthz` return provider readiness.

## 3. Security and Auth

- OAuth2.1/OIDC usable for web login。
- Password/SMS/OIDC/Social login paths pass e2e checks。
- JWT claims map to ExecutionContext without drift。

## 4. Authorization and Data Scope

- Authorization order fixed: Tenant -> Visibility -> ACL -> RBAC -> Egress。
- Row-level data permission SQL filtering is enforced。

## 5. Command Pipeline

- `POST /api/v1/commands` returns `202 + resource + commandRef`。
- command traces and audit events are queryable。

## 6. Capability Wrappers

- cache/event/messaging/storage SPI are available and swappable.
- memory/local fallback paths work when kafka/minio/s3 are unavailable.

## 7. Rollback

- Feature gates can disable high-risk domains without full rollback.
- DB migration rollback scripts are executable.
