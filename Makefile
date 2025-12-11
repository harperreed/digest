# ABOUTME: Makefile for building digest CLI
# ABOUTME: Provides build, test, install, and clean targets

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildDate=$(BUILD_DATE)"

.PHONY: all build test install clean

all: build

build:
	go build $(LDFLAGS) -o digest ./cmd/digest

test:
	go test -v ./...

test-short:
	go test -short -v ./...

install:
	go install $(LDFLAGS) ./cmd/digest

clean:
	rm -f digest
	go clean ./...

# Run integration tests (requires network)
test-integration:
	go test -v -run Integration ./test/...
