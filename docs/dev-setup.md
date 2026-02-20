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

## Hub env (Phase 1 control plane)

Set hub env before running `pnpm dev:hub`:

- `GOYAIS_HUB_DB_PATH=./data/hub.sqlite`
- `GOYAIS_BOOTSTRAP_TOKEN=<required>`
- `GOYAIS_ALLOW_PUBLIC_SIGNUP=false` (default)
- `GOYAIS_SERVER_PORT=8787`

Bootstrap status:

- `GET /v1/auth/bootstrap/status`
- `POST /v1/auth/bootstrap/admin` (requires valid `bootstrap_token`)

## SQLite ownership

Runtime is the single writer for run/event/audit state.
Host reads/writes via runtime APIs only.
