# Makefile for testing
.PHONY: test test-unit test-integration

# Test commands
test: test-unit test-integration

# Run unit tests
test-unit:
	go test -timeout=300s -coverprofile=cover.out -race -gcflags=all=-l ./...

# Run integration tests
test-integration:
	INTEGRATION_TEST=1 go test -v ./tests/...

# Full test and build workflow
all: test build

# Build targets
build: build-plugin

build-plugin:
	go build -o ./bin/protoc-gen-httpinterface ./plugin

# Install the plugin locally for testing
install:
	go install ./plugin

# Clean up
clean:
	rm -f ./bin/protoc-gen-httpinterface
	go clean -testcache	-