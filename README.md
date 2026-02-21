# Goyais v0.2.0

Local-first + Hub-First AI-assisted coding desktop app.

- Host: Tauri v2 + React (`/Users/goya/Repo/Git/Goyais/apps/desktop-tauri`)
- Hub Server: Go 1.23+ + chi + sqlc (`/Users/goya/Repo/Git/Goyais/server/hub-server-go`)
- Runtime: Python FastAPI + LangGraph (`/Users/goya/Repo/Git/Goyais/runtime/python-agent`)
- Protocol: JSON Schema + generated TS/Python/Go types (`/Users/goya/Repo/Git/Goyais/packages/protocol`)

License: Apache-2.0.

## Breaking change notice (v0.2.0)

- **Architecture Shift**: "Run-Centric" has been fully replaced by "Session-Centric + Hub-First".
- **Auth Mode Split**:
  - `local_open`: local workspace, no desktop token/login flow.
  - `remote_auth`: bearer auth + membership + RBAC.
- **Hub Server Authority**: The Hub (now written in Go) is the source of truth for Sessions, Executions, Events, Skills, MCP, and remote Git projects.
- **Local SQLite DB**: Local mode now runs a local Hub which manages the SQLite database. The Python Runtime no longer holds any persisted state (memory buffer only).
- **Execution Mutex**: Only one execution can be active per session at any time.

## What v0.2.0 includes

- **Hub-First Only**: Desktop -> Go Hub -> Runtime/Worker (desktop no direct runtime calls)
- **Session Modes**: Agent mode (autonomous execution with confirmation for sensitive tools) and Plan mode (design plan first, require approval, then execute).
- **Worktree Isolation**: Operations run in an isolated Git worktree `goyais-exec-<id>` and support proper local Git commits via UI.
- **Skills & MCP Integration**: Full CRUD and execution injection for custom Skills and Model Context Protocol (MCP) servers.
- **Remote Git Projects**: Full sync/clone support for remote repositories via Git URL and Auth References.
- **Robustness**: Watchdog timeouts for stalled executions, SSE auto-reconnect, detailed Audit Logs.

## Prerequisites

- Node.js 22+
- pnpm 10+
- Python 3.11+
- Go 1.23+
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
cd server/hub-server-go && make migrate-up
```

## Runtime configuration

Copy runtime env file:

```bash
cp /Users/goya/Repo/Git/Goyais/runtime/python-agent/.env.example /Users/goya/Repo/Git/Goyais/runtime/python-agent/.env
```

Important variables:

- `GOYAIS_HUB_BASE_URL=http://127.0.0.1:8787` (The Python worker must point to the Go Hub)
- `GOYAIS_RUNTIME_REQUIRE_HUB_AUTH=true|false`
- `GOYAIS_RUNTIME_SHARED_SECRET=<shared with hub>`
- `GOYAIS_AGENT_MODE=plan|agent`

### Model API keys (`secret_ref`)

`model_configs.secret_ref` supports:

- `env:OPENAI_API_KEY`
- `env:ANTHROPIC_API_KEY`
- `keychain:openai:default` -> resolves env `GOYAIS_SECRET_OPENAI_DEFAULT`
- `keychain:anthropic:default` -> resolves env `GOYAIS_SECRET_ANTHROPIC_DEFAULT`

## Start demo (macOS)

Terminal 1 (Go Hub):

```bash
cd /Users/goya/Repo/Git/Goyais
GOYAIS_AUTH_MODE=local_open GOYAIS_RUNTIME_SHARED_SECRET=dev-shared pnpm dev:hub
```

Terminal 2 (Python Worker):

```bash
cd /Users/goya/Repo/Git/Goyais
GOYAIS_RUNTIME_REQUIRE_HUB_AUTH=true GOYAIS_RUNTIME_SHARED_SECRET=dev-shared GOYAIS_HUB_BASE_URL=http://127.0.0.1:8787 pnpm dev:runtime
```

Terminal 3 (Tauri Desktop):

```bash
cd /Users/goya/Repo/Git/Goyais
pnpm dev:desktop
```

In UI:

1. Add a Local or Remote Git Project.
2. Configure Settings -> Models (Provider + secret_ref).
3. Optionally configure Skills & MCP servers.
4. Create a new Session (choose Mode, Model, Skills).
5. Submit a task, review Plan (if in Plan mode), inspect diff, commit or discard changes.

## Hub Server API (Go)

- `GET/POST/PATCH/DELETE /v1/sessions`
- `POST /v1/sessions/{id}/execute` -> Returns 409 if busy
- `GET /v1/sessions/{id}/events` (SSE)
- `POST /v1/executions/{id}/commit`
- `GET /v1/executions/{id}/patch`
- `POST /v1/confirmations`
- `GET/POST/DELETE /v1/projects` & `POST /v1/projects/{id}/sync`
- `GET/POST/PUT/DELETE /v1/model-configs`
- `GET /v1/runtime/model-configs/{id}/models`
- `GET /v1/runtime/health`
- `GET/POST/PUT/DELETE /v1/skill-sets` & `skills`
- `GET/POST/PUT/DELETE /v1/mcp-connectors`

## Security model (v0.2.0)

- Go Hub is the single database writer and authority.
- `write_fs`, `exec`, `network`, `delete` require explicit confirmation (Agent Mode).
- Git operations are isolated in ephemeral worktrees to prevent accidental destruction of uncommitted work.
- Hub watchdog automatically reclaims session mutexes from crashed Python workers.
- All operations (session CRUD, git commit, project sync, tool calls, confirmations) are recorded in `audit_logs`.

## Test and verification

```bash
cd /Users/goya/Repo/Git/Goyais
pnpm typecheck
pnpm test

# Test Go Hub
cd server/hub-server-go
go test ./...
```
