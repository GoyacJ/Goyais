BINARY   := hub
CMD_DIR  := ./cmd/hub
DB_DIR   := ./migrations
SQLC_CFG := sqlc.yaml

.PHONY: all build test lint migrate-up migrate-down sqlc-gen clean run

all: build

build:
	CGO_ENABLED=1 go build -o bin/$(BINARY) $(CMD_DIR)

run: build
	./bin/$(BINARY)

test:
	CGO_ENABLED=1 go test ./... -v -count=1

lint:
	golangci-lint run ./...

# SQLite migrations (local dev)
migrate-up:
	goose -dir $(DB_DIR) sqlite3 "$${GOYAIS_DB_PATH:-./data/hub.db}" up

migrate-down:
	goose -dir $(DB_DIR) sqlite3 "$${GOYAIS_DB_PATH:-./data/hub.db}" down

# Postgres migrations (remote)
migrate-up-pg:
	goose -dir $(DB_DIR) postgres "$${GOYAIS_DATABASE_URL}" up

sqlc-gen:
	sqlc generate -f $(SQLC_CFG)

clean:
	rm -rf bin/
