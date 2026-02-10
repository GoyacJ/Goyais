BINARY ?= build/goyais
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X 'goyais/internal/buildinfo.Version=$(VERSION)' -X 'goyais/internal/buildinfo.Commit=$(COMMIT)' -X 'goyais/internal/buildinfo.BuildTime=$(BUILD_TIME)'

.PHONY: web-install web-build web-dev web-sync build test clean

web-install:
	pnpm -C web install --frozen-lockfile

web-build: web-install
	pnpm -C web build

web-dev:
	pnpm -C web dev

web-sync:
	rm -rf internal/access/webstatic/dist
	mkdir -p internal/access/webstatic/dist
	cp -R web/dist/. internal/access/webstatic/dist/
	touch internal/access/webstatic/dist/.keep

build: web-build web-sync
	mkdir -p build
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/api

test:
	go test ./...

clean:
	rm -rf build
