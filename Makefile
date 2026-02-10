SHELL := /bin/bash

WEB_DIR := web
BINARY := build/goyais

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X goyais/internal/buildinfo.Version=$(VERSION) -X goyais/internal/buildinfo.Commit=$(COMMIT) -X goyais/internal/buildinfo.BuildTime=$(BUILD_TIME)

.PHONY: web-install web-build web-dev prepare-embed build clean

web-install:
	pnpm -C $(WEB_DIR) install --frozen-lockfile

web-build: web-install
	pnpm -C $(WEB_DIR) build

web-dev: web-install
	pnpm -C $(WEB_DIR) dev

prepare-embed: web-build
	rm -rf internal/access/webstatic/dist
	mkdir -p internal/access/webstatic
	cp -R $(WEB_DIR)/dist internal/access/webstatic/dist

build: prepare-embed
	mkdir -p $(dir $(BINARY))
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/api
	chmod +x $(BINARY)

clean:
	rm -rf build
