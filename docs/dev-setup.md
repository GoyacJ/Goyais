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
- `GOYAIS_HUB_RUNTIME_SHARED_SECRET=<required for runtime gateway>`
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

## Runtime gateway env (Phase 3)

Remote runtime（由 hub 代理访问）建议最小配置：

- `GOYAIS_RUNTIME_REQUIRE_HUB_AUTH=true`
- `GOYAIS_RUNTIME_SHARED_SECRET=<must match GOYAIS_HUB_RUNTIME_SHARED_SECRET>`
- `GOYAIS_RUNTIME_WORKSPACE_ID=<workspace_id>`
- `GOYAIS_RUNTIME_WORKSPACE_ROOT=<workspace_root_path>`
- `GOYAIS_HUB_BASE_URL=http://127.0.0.1:8787`
- `GOYAIS_RUNTIME_PORT=<workspace_runtime_port>`

注意：

- `workspace_id` 仅通过 Hub 外部 API query 参数传递（`?workspace_id=...`）。
- Hub 到 runtime 的上下文通过 header 注入（`X-Hub-Auth`、`X-User-Id`、`X-Trace-Id`）。
- runtime `/v1/health` 必须返回当前 `workspace_id`，Hub 会校验与 registry 一致，不一致会拒绝流量。

## Phase 3 quick path

1. 启动 hub：`pnpm dev:hub`
2. 启动 workspace runtime（按上述 env）
3. 注册 runtime：
   `POST /v1/admin/workspaces/{workspace_id}/runtime` with `runtime_base_url`
4. 验证 gateway：
   - `GET /v1/runtime/health?workspace_id=...`
   - `POST /v1/runtime/runs?workspace_id=...`
   - `GET /v1/runtime/runs/{run_id}/events?workspace_id=...`
   - `POST /v1/runtime/tool-confirmations?workspace_id=...`

## SQLite ownership

Runtime is the single writer for run/event/audit state.
Host reads/writes via runtime APIs only.
