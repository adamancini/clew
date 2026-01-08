.PHONY: build clean test test-unit test-e2e test-all lint install plugin plugin-binaries plugin-clean

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

# Default target
all: build

# Build the binary
build:
	go build $(LDFLAGS) -o clew ./cmd/clew

# Install to GOPATH/bin
install:
	go install $(LDFLAGS) ./cmd/clew

# Run all tests (unit + e2e)
test:
	go test -v ./...

# Run only unit tests
test-unit:
	go test -v ./internal/...

# Run only e2e tests
test-e2e:
	go test -v ./test/e2e/...

# Run all tests with coverage
test-all: test-unit test-e2e

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./internal/config ./internal/diff ./internal/state ./internal/sync
	go tool cover -html=coverage.out -o coverage.html

# Run e2e tests with verbose output
test-e2e-verbose:
	go test -v -count=1 ./test/e2e/...

# Lint
lint:
	golangci-lint run

# Clean build artifacts
clean:
	rm -f clew
	rm -f coverage.out coverage.html
	rm -rf dist/
	rm -rf bin/

# Build for multiple platforms
build-all: clean
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/clew-darwin-arm64 ./cmd/clew
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/clew-darwin-amd64 ./cmd/clew
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/clew-linux-amd64 ./cmd/clew
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/clew-linux-arm64 ./cmd/clew

# Format code
fmt:
	go fmt ./...

# Tidy dependencies
tidy:
	go mod tidy

# Plugin targets
# Build binaries for plugin distribution
plugin-binaries:
	@mkdir -p bin
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/clew-darwin-arm64 ./cmd/clew
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/clew-darwin-amd64 ./cmd/clew
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/clew-linux-amd64 ./cmd/clew
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/clew-linux-arm64 ./cmd/clew
	@chmod +x bin/*
	@echo "Plugin binaries built in bin/"

# Clean plugin binaries
plugin-clean:
	rm -rf bin/

# Build complete plugin package (binaries + structure)
plugin: plugin-binaries
	@echo "Plugin structure ready:"
	@echo "  .claude-plugin/plugin.json"
	@echo "  skills/clew/SKILL.md"
	@echo "  hooks/hooks.json"
	@echo "  hooks/session_start.sh"
	@echo "  bin/clew-{darwin,linux}-{arm64,amd64}"
	@echo ""
	@echo "To test locally:"
	@echo "  claude --plugin-dir ."
