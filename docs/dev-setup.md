# Development Setup

## Monorepo commands

- `pnpm install`
- `pnpm protocol:generate`
- `pnpm dev:runtime`
- `pnpm dev:desktop`
- `pnpm dev:sync`
- `pnpm dev:hub`
- `pnpm test`

## Runtime env

Copy `runtime/python-agent/.env.example` and adjust values.

## Hub env (Phase 1 + Phase 2)

Set hub env before running `pnpm dev:hub`:

- `GOYAIS_HUB_DB_PATH=./data/hub.sqlite`
- `GOYAIS_BOOTSTRAP_TOKEN=<required>`
- `GOYAIS_ALLOW_PUBLIC_SIGNUP=false` (default)
- `GOYAIS_HUB_SECRET_KEY=<required for model-config create/update>`
- `GOYAIS_SERVER_PORT=8787`

Example key generation:

- `export GOYAIS_HUB_SECRET_KEY="$(openssl rand -base64 32)"`

Bootstrap status:

- `GET /v1/auth/bootstrap/status`
- `POST /v1/auth/bootstrap/admin` (requires valid `bootstrap_token`)

Phase 2 domain endpoints (all require `Authorization` + `workspace_id` query):

- `GET|POST /v1/projects?workspace_id=...`
- `DELETE /v1/projects/{project_id}?workspace_id=...`
- `GET|POST /v1/model-configs?workspace_id=...`
- `PUT|DELETE /v1/model-configs/{model_config_id}?workspace_id=...`

RBAC enforcement:

- Projects: `project:read` / `project:write`
- Model Configs: `modelconfig:read` / `modelconfig:manage`
- 服务端强制校验 member + permission，前端仅做 UI 级 gating。

## SQLite ownership

Runtime is the single writer for run/event/audit state.
Host reads/writes via runtime APIs only.
