# Makefile for azure2aws
# Build, test, lint, clean and cross compilation targets.

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(DATE)"

BINARY := azure2aws
BIN_DIR := bin
DIST_DIR := dist

.PHONY: all build build-all test lint clean

all: build

# Build for the host platform
build:
	@mkdir -p $(BIN_DIR)
	go build $(LDFLAGS) -o $(BIN_DIR)/$(BINARY) ./cmd/azure2aws

# Cross build for linux/darwin/windows x amd64/arm64
build-all:
	@mkdir -p $(DIST_DIR)
	@for GOOS in linux darwin windows; do \
	  for GOARCH in amd64 arm64; do \
	    OUT="$(DIST_DIR)/$(BINARY)-$${GOOS}-$${GOARCH}"; \
	    if [ "$${GOOS}" = "windows" ]; then OUT="$${OUT}.exe"; fi; \
	    echo "Building $${OUT} ..."; \
	    GOOS=$${GOOS} GOARCH=$${GOARCH} go build $(LDFLAGS) -o "$${OUT}" ./cmd/azure2aws || exit $$?; \
	  done; \
	done

test:
	go test ./... -coverprofile=coverage.out

lint:
	golangci-lint run --timeout=5m

clean:
	rm -rf $(BIN_DIR) $(DIST_DIR) coverage.out
