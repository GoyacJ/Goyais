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

Event envelope always includes `protocol_version` (loaded from `packages/protocol/schemas/**/protocol-version.json`) and `trace_id`.

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

## Hub server (Control Plane + Remote Domain Data)

`hub-server` 提供 remote workspace 能力：

- Phase 1（控制面）：bootstrap admin、登录鉴权、工作区列表、导航与权限下发
- Phase 2（远端数据面）：Projects / Model Configs（workspace 隔离 + RBAC 强制）
- Phase 3（runtime gateway）：remote runs/SSE/tool confirmations 统一经 Hub 代理

目录：`/Users/goya/Repo/Git/Goyais/server/hub-server`

### Hub env

```bash
export GOYAIS_HUB_DB_PATH=./data/hub.sqlite
export GOYAIS_BOOTSTRAP_TOKEN=change-me-bootstrap-token
export GOYAIS_ALLOW_PUBLIC_SIGNUP=false
export GOYAIS_HUB_SECRET_KEY=<base64-32-byte-key>
export GOYAIS_HUB_RUNTIME_SHARED_SECRET=<required-shared-secret>
export GOYAIS_SERVER_PORT=8787
```

可用下面方式生成 `GOYAIS_HUB_SECRET_KEY`：

```bash
export GOYAIS_HUB_SECRET_KEY="$(openssl rand -base64 32)"
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

### Hub Phase 2 domain API curl

```bash
# Projects
curl -sS "http://127.0.0.1:8787/v1/projects?workspace_id=<workspace_id>" \
  -H "Authorization: Bearer <token>"

curl -sS -X POST "http://127.0.0.1:8787/v1/projects?workspace_id=<workspace_id>" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"name":"Demo Remote Project","root_uri":"repo://demo/main"}'

curl -sS -X DELETE "http://127.0.0.1:8787/v1/projects/<project_id>?workspace_id=<workspace_id>" \
  -H "Authorization: Bearer <token>"

# Model Configs (api_key write-only)
curl -sS "http://127.0.0.1:8787/v1/model-configs?workspace_id=<workspace_id>" \
  -H "Authorization: Bearer <token>"

curl -sS -X POST "http://127.0.0.1:8787/v1/model-configs?workspace_id=<workspace_id>" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"provider":"openai","model":"gpt-4.1-mini","temperature":0,"max_tokens":2048,"api_key":"sk-remote-xxx"}'

curl -sS -X PUT "http://127.0.0.1:8787/v1/model-configs/<model_config_id>?workspace_id=<workspace_id>" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"model":"gpt-4.1","api_key":"sk-remote-new"}'

curl -sS -X DELETE "http://127.0.0.1:8787/v1/model-configs/<model_config_id>?workspace_id=<workspace_id>" \
  -H "Authorization: Bearer <token>"
```

安全说明（Phase 2）：

- `api_key` 只允许 create/update 时写入，不会在 GET 响应中返回。
- `secrets.value_encrypted` 始终存密文字符串。
- 缺少或非法 `GOYAIS_HUB_SECRET_KEY` 时，model-config create/update 会失败（默认拒绝）。
- 所有 domain endpoint 都要求 `workspace_id` 并校验 active membership + permission。

所有 hub 响应（成功/失败）都会回传 `X-Trace-Id`。

### Hub Phase 3 runtime gateway quickstart

1) 启动 remote runtime（示例）：

```bash
cd /Users/goya/Repo/Git/Goyais/runtime/python-agent
GOYAIS_RUNTIME_REQUIRE_HUB_AUTH=true \
GOYAIS_RUNTIME_SHARED_SECRET=<required-shared-secret> \
GOYAIS_RUNTIME_WORKSPACE_ID=<workspace_id> \
GOYAIS_RUNTIME_WORKSPACE_ROOT=/srv/workspaces/<workspace_id> \
GOYAIS_HUB_BASE_URL=http://127.0.0.1:8787 \
GOYAIS_RUNTIME_PORT=19001 \
pnpm dev
```

2) 在 hub 注册 runtime（需要 `workspace:manage`）：

```bash
curl -sS -X POST "http://127.0.0.1:8787/v1/admin/workspaces/<workspace_id>/runtime" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"runtime_base_url":"http://127.0.0.1:19001"}'
```

3) 通过 Hub Gateway 发起 run / 订阅事件 / 确认：

```bash
curl -sS -X POST "http://127.0.0.1:8787/v1/runtime/runs?workspace_id=<workspace_id>" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"project_id":"project-demo","session_id":"session-demo","input":"update readme","model_config_id":"model-demo","workspace_path":"/ignored/by-remote","options":{"use_worktree":false}}'

curl -N "http://127.0.0.1:8787/v1/runtime/runs/<run_id>/events?workspace_id=<workspace_id>" \
  -H "Authorization: Bearer <token>"

curl -sS -X POST "http://127.0.0.1:8787/v1/runtime/tool-confirmations?workspace_id=<workspace_id>" \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"run_id":"<run_id>","call_id":"<call_id>","approved":true}'
```

4) 权限要求（服务端强制）：

- `POST /v1/runtime/runs` -> `run:create`
- `GET /v1/runtime/runs*` / `events` / `replay` -> `run:read`
- `POST /v1/runtime/tool-confirmations` -> `confirm:write`

5) 安全约束（Phase 3）：

- remote desktop 不直连 runtime，只能走 hub `/v1/runtime/*`
- hub 会探活并校验 runtime 自报 `workspace_id` 必须与 registry 一致，不一致返回 `E_RUNTIME_MISCONFIGURED`
- runtime 在 `GOYAIS_RUNTIME_REQUIRE_HUB_AUTH=true` 下，必须校验 `X-Hub-Auth` / `X-User-Id` / `X-Trace-Id`
- secret `secret:*` 仅由 runtime 通过 hub internal resolve 一次性解密，不回传到 desktop，不落盘

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
