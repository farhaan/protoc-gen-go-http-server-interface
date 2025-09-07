# Makefile for protoc-gen-go-http-server-interface
.PHONY: test build install clean

# Version information
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS = -X github.com/farhaan/protoc-gen-go-http-server-interface/version.Version=$(VERSION) \
          -X github.com/farhaan/protoc-gen-go-http-server-interface/version.GitCommit=$(GIT_COMMIT) \
          -X github.com/farhaan/protoc-gen-go-http-server-interface/version.BuildTime=$(BUILD_TIME)

# Test with parallel execution
test:
	go test -timeout=300s -race -parallel=8 ./...

# Build the plugin  
build:
	mkdir -p ./bin
	go build -ldflags "$(LDFLAGS)" -o ./bin/protoc-gen-go-http-server-interface .

# Install the plugin locally
install:
	go install -ldflags "$(LDFLAGS)" .

# Install as go tool (for users)
install-tool:
	go install .

# Clean up
clean:
	rm -rf ./bin/
	go clean -testcache

