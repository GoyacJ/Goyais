# Goyais MVP-1

Local-first AI-assisted coding desktop app.

- Host: Tauri v2 + React (`/Users/goya/Repo/Git/Goyais/apps/desktop-tauri`)
- Runtime: FastAPI + LangGraph + Deep Agents (`/Users/goya/Repo/Git/Goyais/runtime/python-agent`)
- Protocol: JSON Schema + generated TS/Python types (`/Users/goya/Repo/Git/Goyais/packages/protocol`)
- Sync: single-user backup server (`/Users/goya/Repo/Git/Goyais/server/sync-server`)

License: Apache-2.0.

## Breaking change notice

- Protocol has been upgraded to `2.0.0`.
- Older clients expecting `protocol_version=1.0.0`, legacy `payload.message`, or HTTP `detail` are not supported.

## What MVP-1 includes

- SSE timeline: `plan`, `tool_call`, `tool_result`, `patch`, `error`, `done`
- Unified diff preview + approve/deny flow before `apply_patch`
- Sensitive tool confirmation + audit logs
- SQLite persistence (`projects/sessions/runs/events/artifacts/model_configs/audit_logs/tool_confirmations`)
- Single-user push/pull sync with bearer token and server-assigned `server_seq`
- Protocol v2 (`2.0.0`) with unified `GoyaisError` + required `trace_id`

## Prerequisites

- Node.js 22+
- pnpm 10+
- Python 3.11+
- Rust stable
- `uv` for Python package/runtime management

Install `uv`:

```bash
curl -LsSf https://astral.sh/uv/install.sh | sh
```

## Install

```bash
cd /Users/goya/Repo/Git/Goyais
pnpm install
pnpm protocol:generate
pnpm --filter @goyais/python-agent migrate
```

## Runtime configuration

Copy runtime env file:

```bash
cp /Users/goya/Repo/Git/Goyais/runtime/python-agent/.env.example /Users/goya/Repo/Git/Goyais/runtime/python-agent/.env
```

Important variables:

- `GOYAIS_AGENT_MODE=mock|graph|deepagents`
- `GOYAIS_DB_PATH=.goyais/runtime.db`
- `GOYAIS_SYNC_SERVER_URL=http://127.0.0.1:8140`
- `GOYAIS_SYNC_TOKEN=change-me`

### Model API keys (`secret_ref`)

`model_configs.secret_ref` supports:

- `env:OPENAI_API_KEY`
- `env:ANTHROPIC_API_KEY`
- `keychain:openai:default` -> resolves env `GOYAIS_SECRET_OPENAI_DEFAULT`
- `keychain:anthropic:default` -> resolves env `GOYAIS_SECRET_ANTHROPIC_DEFAULT`

If key resolution fails in `graph/deepagents` mode, runtime emits explicit `error` and marks run failed.

## Start demo (macOS)

Terminal 1:

```bash
cd /Users/goya/Repo/Git/Goyais
pnpm dev:runtime
```

Terminal 2:

```bash
cd /Users/goya/Repo/Git/Goyais
pnpm dev:desktop
```

In UI:

1. Create/select project (`workspace_path`)
2. Create model config (provider + model + `secret_ref`)
3. Open Run page and submit task
4. Inspect live events and patch diff
5. Approve or deny sensitive calls in Permission Center/Modal

Sample task:

```text
把 README 的标题改成 MVP-1 Demo
```

## Runtime API (host-facing)

- `POST /v1/runs`
- `GET /v1/runs/{run_id}/events` (SSE)
- `POST /v1/tool-confirmations`
- `GET /v1/projects`, `POST /v1/projects`
- `GET /v1/model-configs`, `POST /v1/model-configs`
- `GET /v1/runs?session_id=...`
- `GET /v1/runs/{run_id}/events/replay`
- `GET /v1/system-events?since_global_seq=...`
- `GET /v1/health`
- `GET /v1/version`
- `GET /v1/metrics`
- `GET /v1/diagnostics/run/{run_id}` (requires `X-Runtime-Token`)

Event envelope always includes `protocol_version=2.0.0` and `trace_id`.

Error response shape (Runtime + Sync):

```json
{
  "error": {
    "code": "E_INTERNAL",
    "message": "Internal server error.",
    "trace_id": "....",
    "retryable": false
  }
}
```

## Sync server (single-user P0)

Start local sync server:

```bash
cd /Users/goya/Repo/Git/Goyais
pnpm dev:sync
```

Or Docker:

```bash
cd /Users/goya/Repo/Git/Goyais/server/sync-server
docker compose up --build
```

Sync API:

- `POST /v1/sync/push` with `since_global_seq`
- `GET /v1/sync/pull?since_server_seq=...`
- `GET /v1/health`
- `GET /v1/version`
- `GET /v1/metrics`

Server assigns monotonic `server_seq` (server-authoritative append-only).
Request/response both support `X-Trace-Id`.

Desktop trigger:

- Settings page -> `Sync now`

## Hub server (Control Plane Phase 1)

`hub-server` 提供 remote workspace 的控制面能力：bootstrap admin、登录鉴权、工作区列表、导航与权限下发。

目录：`/Users/goya/Repo/Git/Goyais/server/hub-server`

### Hub env

```bash
export GOYAIS_HUB_DB_PATH=./data/hub.sqlite
export GOYAIS_BOOTSTRAP_TOKEN=change-me-bootstrap-token
export GOYAIS_ALLOW_PUBLIC_SIGNUP=false
export GOYAIS_SERVER_PORT=8787
```

### Start hub

```bash
cd /Users/goya/Repo/Git/Goyais
pnpm dev:hub
```

### Hub quickstart curl

```bash
# 1) setup status
curl -i http://127.0.0.1:8787/v1/auth/bootstrap/status

# 2) bootstrap admin (only once, requires bootstrap token)
curl -sS -X POST http://127.0.0.1:8787/v1/auth/bootstrap/admin \
  -H 'Content-Type: application/json' \
  -d '{"bootstrap_token":"change-me-bootstrap-token","email":"admin@example.com","password":"Passw0rd!","display_name":"Admin"}'

# 3) login
curl -sS -X POST http://127.0.0.1:8787/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com","password":"Passw0rd!"}'
```

返回 token 后可继续调用：

```bash
curl -sS http://127.0.0.1:8787/v1/me -H "Authorization: Bearer <token>"
curl -sS http://127.0.0.1:8787/v1/workspaces -H "Authorization: Bearer <token>"
curl -sS "http://127.0.0.1:8787/v1/me/navigation?workspace_id=<workspace_id>" -H "Authorization: Bearer <token>"
```

所有 hub 响应（成功/失败）都会回传 `X-Trace-Id`。

## Security model (MVP-1)

- Runtime is the single SQLite writer; Host never writes SQLite directly
- `write_file`, `apply_patch`, `run_command` require explicit approval
- Workspace path guard blocks writes outside workspace
- Command guard denylist + allowlist
- All tool calls/results/decisions are audit logged
- Runtime restart while waiting confirmation:
  - pending status is recovered as denied by system
  - runtime emits `error` + terminal `done`

## Test and verification

```bash
cd /Users/goya/Repo/Git/Goyais
pnpm typecheck
pnpm test
```

Python runtime tests include:

- path escape rejection
- command allow/deny policy
- confirmation wait/resolve and restart recovery
- protocol envelope validation

Protocol and sync server include schema and sync behavior tests (incremental + idempotent push).
