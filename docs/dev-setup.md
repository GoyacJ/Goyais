# Development Setup

## Monorepo commands

- `pnpm install`
- `pnpm protocol:generate`
- `pnpm dev:hub`
- `pnpm dev:runtime`
- `pnpm dev:desktop`
- `pnpm test`

## v0.2.0 auth modes

Hub supports exactly two modes:

- `GOYAIS_AUTH_MODE=local_open`
  - desktop local workspace does **not** use token/login
  - single local workspace returned from `/v1/workspaces`
- `GOYAIS_AUTH_MODE=remote_auth`
  - desktop remote workspace must provide bearer token
  - workspace membership + RBAC enforced on workspace-scoped routes

## Local development (recommended)

### 1) Start Hub in `local_open`

```bash
GOYAIS_AUTH_MODE=local_open \
GOYAIS_RUNTIME_SHARED_SECRET=dev-shared \
pnpm dev:hub
```

### 2) Start Runtime (Hub-auth only)

```bash
GOYAIS_RUNTIME_REQUIRE_HUB_AUTH=true \
GOYAIS_RUNTIME_SHARED_SECRET=dev-shared \
GOYAIS_HUB_BASE_URL=http://127.0.0.1:8787 \
pnpm dev:runtime
```

### 3) Start Desktop

```bash
pnpm dev:desktop
```

## Hub-first routing rules

- Desktop must only call Hub APIs.
- Hub forwards runtime requests with headers:
  - `X-Hub-Auth`
  - `X-User-Id`
  - `X-Trace-Id`
- Runtime should reject direct desktop calls when `GOYAIS_RUNTIME_REQUIRE_HUB_AUTH=true`.

## Key Hub APIs (workspace-scoped)

All routes below require `workspace_id` query and RBAC in `remote_auth` mode:

- `GET|POST /v1/projects`
- `DELETE /v1/projects/{project_id}`
- `GET|POST|PUT|DELETE /v1/model-configs`
- `GET /v1/runtime/model-configs/{model_config_id}/models`
- `GET /v1/runtime/health`

## Notes

- Local mode removes old local token/bootstrap/auto-login flow.
- Desktop settings no longer expose Runtime URL or SyncNow entry.
- If you need remote auth testing, switch Hub to `GOYAIS_AUTH_MODE=remote_auth` and log in via remote workspace profile.
