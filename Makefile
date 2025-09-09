PROJECT_PKG := github.com/choreoatlas2025/cli
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS     := -s -w -X '$(PROJECT_PKG)/internal/cli.Version=$(VERSION)' -X '$(PROJECT_PKG)/internal/cli.BuildEdition=ce'
BIN         := bin/choreoatlas

.PHONY: help build test lint clean

help:
	@echo "ChoreoAtlas CLI - Community Edition Build"
	@echo ""
	@echo "Available targets:"
	@echo "  build         - Build Community Edition"
	@echo "  test          - Run tests"
	@echo "  lint          - Run linter"
	@echo "  clean         - Clean build artifacts"

build:
	@echo "Building ChoreoAtlas CLI (Community Edition)..."
	go build -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/choreoatlas
	@echo "Creating default symlinks..."
	cd bin && ln -sf choreoatlas ca

test:
	go test ./...

lint:
	golangci-lint run || true

clean:
	rm -rf bin