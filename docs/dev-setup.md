# Development Setup (v0.4.0)

This guide reflects the current codebase in `main` for Goyais v0.4.0.

## 1. Prerequisites

- Node.js 22+ (CI uses 24)
- pnpm 10.11+
- Go 1.24+
- Python 3.11+
- [uv](https://docs.astral.sh/uv/)
- Rust stable (Tauri toolchain)

Install dependencies once at repo root:

```bash
pnpm install
```

## 2. Local Desktop Mode (recommended)

Run desktop directly:

```bash
pnpm run dev:desktop
```

What this does:

1. Runs `apps/desktop` Tauri dev mode.
2. Auto-prepares sidecar binaries when missing:
   - `goyais-hub-<target-triple>`
   - `goyais-worker-<target-triple>`
3. Desktop starts local Hub (`8787`) and Worker (`8788`) sidecars.

Sidecar preparation script:

```bash
scripts/release/ensure-local-sidecars.sh
```

Force rebuild local sidecars:

```bash
GOYAIS_FORCE_SIDECAR_REBUILD=1 pnpm --filter @goyais/desktop sidecar:prepare
```

## 3. Split-Service Debug Mode (optional)

Useful when debugging Hub or Worker independently.

### Hub

```bash
make dev-hub
```

### Worker

```bash
make dev-worker
```

### Desktop web-only

```bash
make dev-web
```

### Quick command helper

```bash
pnpm run dev
```

This prints the command set for 3-terminal startup.

## 4. Health Checks

```bash
curl http://127.0.0.1:8787/health
curl http://127.0.0.1:8788/health
```

Smoke script:

```bash
make health
```

## 5. Tests and Quality Gates

Run all tests:

```bash
make test
```

Run lint:

```bash
make lint
```

Desktop strict/runtime gates:

```bash
pnpm --filter @goyais/desktop test:strict
pnpm --filter @goyais/desktop check:tokens
pnpm --filter @goyais/desktop check:size
pnpm --filter @goyais/desktop check:complexity
pnpm --filter @goyais/desktop coverage:gate
```

Hub and Worker standalone:

```bash
cd services/hub && go test ./...
cd services/worker && uv sync && uv run pytest
```

## 6. Environment Notes

### Hub

- `PORT` (default: `8787`)
- `HUB_DB_PATH` (optional; defaults to user config dir db)
- `HUB_INTERNAL_TOKEN` (required for Worker internal auth in sidecar/remote setups)

### Worker

- `PORT` (default: `8788`)
- `HUB_BASE_URL` (default: `http://127.0.0.1:8787`)
- `HUB_INTERNAL_TOKEN`
- `WORKER_MAX_CONCURRENCY` (default: `3`)
- `WORKER_DISABLE_CLAIM_LOOP` (test/debug)
- TLS related:
  - `WORKER_TLS_CA_FILE`
  - `WORKER_TLS_INSECURE_SKIP_VERIFY=1` (debug only)

### Desktop frontend

- `VITE_HUB_BASE_URL` (default: `http://127.0.0.1:8787`)
- `VITE_API_MODE` (`real` / `hybrid` / `mock`)
- `VITE_ENABLE_MOCK_FALLBACK` (`false` recommended for strict verification)

## 7. Logs and Troubleshooting

Desktop sidecar log (macOS):

```text
~/Library/Application Support/com.goyais.desktop/sidecar.log
```

Common checks:

1. Missing sidecar binary: run `pnpm --filter @goyais/desktop sidecar:prepare`.
2. Port conflict (`8787`/`8788`): stop stale processes and relaunch desktop.
3. Worker start timeout: inspect `sidecar.log` for worker stderr and health probe details.

## 8. Packaging and Release

### Build local package (current host target)

```bash
TARGET_TRIPLE="$(rustc -vV | awk '/^host:/ {print $2}')"
pnpm --filter @goyais/desktop sidecar:prepare
cd apps/desktop
VITE_API_MODE=strict VITE_ENABLE_MOCK_FALLBACK=false pnpm tauri build -- --target "$TARGET_TRIPLE" --no-sign
```

### GitHub Release via tag

Workflow file:

- `.github/workflows/release.yml`

Trigger release draft build:

```bash
git tag -a v0.4.0 -m "v0.4.0"
git push origin v0.4.0
```

The release workflow builds matrix artifacts for:

- macOS arm64
- macOS x64
- Linux x64
- Windows x64
