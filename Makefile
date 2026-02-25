HUB_PORT ?= 8787
DESKTOP_PORT ?= 5173

.PHONY: dev dev-hub dev-desktop dev-web health health-app test lint test-hub test-desktop lint-hub lint-desktop

dev:
	@scripts/dev/print_commands.sh $(HUB_PORT) $(DESKTOP_PORT)

dev-hub:
	@cd services/hub && PORT=$(HUB_PORT) HUB_INTERNAL_TOKEN=$${HUB_INTERNAL_TOKEN:?HUB_INTERNAL_TOKEN is required} go run ./cmd/hub

dev-desktop:
	@cd apps/desktop && pnpm tauri:dev

dev-web:
	@cd apps/desktop && DESKTOP_PORT=$(DESKTOP_PORT) pnpm dev --port $(DESKTOP_PORT)

health:
	@HUB_PORT=$(HUB_PORT) DESKTOP_PORT=$(DESKTOP_PORT) scripts/smoke/health_check.sh

health-app:
	@DESKTOP_PORT=$(DESKTOP_PORT) scripts/smoke/health_app_check.sh

test: test-hub test-desktop

test-hub:
	@cd services/hub && go test ./...

test-desktop:
	@pnpm --filter @goyais/desktop test

lint: lint-hub lint-desktop

lint-hub:
	@cd services/hub && go vet ./...

lint-desktop:
	@pnpm --filter @goyais/desktop lint
