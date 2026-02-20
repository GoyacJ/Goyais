# Development Setup

## Monorepo commands

- `pnpm install`
- `pnpm protocol:generate`
- `pnpm dev:runtime`
- `pnpm dev:desktop`
- `pnpm dev:sync`
- `pnpm test`

## Runtime env

Copy `runtime/python-agent/.env.example` and adjust values.

## SQLite ownership

Runtime is the single writer for run/event/audit state.
Host reads/writes via runtime APIs only.
