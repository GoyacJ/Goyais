HUB_PORT ?= 8787
WORKER_PORT ?= 8788
DESKTOP_PORT ?= 5173

.PHONY: dev dev-hub dev-worker dev-desktop dev-web health health-app test lint test-hub test-worker test-desktop lint-hub lint-worker lint-desktop

dev:
	@scripts/dev/print_commands.sh $(HUB_PORT) $(WORKER_PORT) $(DESKTOP_PORT)

dev-hub:
	@cd services/hub && PORT=$(HUB_PORT) HUB_INTERNAL_TOKEN=$${HUB_INTERNAL_TOKEN:?HUB_INTERNAL_TOKEN is required} go run ./cmd/hub

dev-worker:
	@cd services/worker && PORT=$(WORKER_PORT) HUB_INTERNAL_TOKEN=$${HUB_INTERNAL_TOKEN:?HUB_INTERNAL_TOKEN is required} uv run python -m app.main

dev-desktop:
	@cd apps/desktop && pnpm tauri:dev

dev-web:
	@cd apps/desktop && DESKTOP_PORT=$(DESKTOP_PORT) pnpm dev --port $(DESKTOP_PORT)

health:
	@HUB_PORT=$(HUB_PORT) WORKER_PORT=$(WORKER_PORT) DESKTOP_PORT=$(DESKTOP_PORT) scripts/smoke/health_check.sh

health-app:
	@DESKTOP_PORT=$(DESKTOP_PORT) scripts/smoke/health_app_check.sh

test: test-hub test-worker test-desktop

test-hub:
	@cd services/hub && go test ./...

test-worker:
	@cd services/worker && uv sync && uv run pytest

test-desktop:
	@pnpm --filter @goyais/desktop test

lint: lint-hub lint-worker lint-desktop

lint-hub:
	@cd services/hub && go vet ./...

lint-worker:
	@cd services/worker && uv sync && uv run ruff check .

lint-desktop:
	@pnpm --filter @goyais/desktop lint
