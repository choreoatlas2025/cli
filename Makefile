# SPDX-FileCopyrightText: 2025 ChoreoAtlas contributors
# SPDX-License-Identifier: Apache-2.0
PROJECT_PKG := github.com/choreoatlas2025/cli
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT  ?= $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
LDFLAGS     := -s -w -X '$(PROJECT_PKG)/internal/cli.Version=$(VERSION)' -X '$(PROJECT_PKG)/internal/cli.GitCommit=$(GIT_COMMIT)' -X '$(PROJECT_PKG)/internal/cli.BuildEdition=ce' -X '$(PROJECT_PKG)/internal/cli.BuildChannel=make'
BIN         := bin/choreoatlas
GOLANGCI_LINT ?= golangci-lint

.PHONY: help build test lint clean deps run-example

help:
	@echo "ChoreoAtlas CLI - Community Edition Build"
	@echo ""
	@echo "Available targets:"
	@echo "  build         - Build Community Edition"
	@echo "  deps          - Install dependencies"
	@echo "  test          - Run tests"
	@echo "  lint          - Run linter"
	@echo "  run-example   - Run example validation"
	@echo "  clean         - Clean build artifacts"

build:
	@echo "Building ChoreoAtlas CLI (Community Edition)..."
	go build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/choreoatlas
	@echo "Creating default symlinks..."
	cd bin && ln -sf choreoatlas ca

test:
	go test ./...

lint:
	$(GOLANGCI_LINT) run --timeout=5m

deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

run-example: build
	@echo "Running example validation..."
	$(BIN) lint --flow examples/flows/order-fulfillment.flowspec.yaml
	$(BIN) validate --flow examples/flows/order-fulfillment.flowspec.yaml --trace examples/traces/successful-order.trace.json

clean:
	rm -rf bin
