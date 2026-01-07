.PHONY: build clean test lint install

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

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Lint
lint:
	golangci-lint run

# Clean build artifacts
clean:
	rm -f clew
	rm -f coverage.out coverage.html
	rm -rf dist/

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
