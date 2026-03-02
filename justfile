set shell := ["sh", "-c"]

# Project binaries
DAEMON := "./cmd/aegisd/*"
CLI    := "./cmd/aegis/*"

# --- Build Commands ---

# Build everything
all: build-daemon build-cli

# Build only the daemon (aegisd)
build-daemon:
    go build -o bin/aegisd {{DAEMON}}

# Build only the CLI (aegis)
build-cli:
    go build -o bin/aegis {{CLI}}

# --- Development Commands ---

# Run the daemon with hot-reload (watches cmd/aegisd)
dev-daemon:
    air --build.cmd "go build -o bin/aegisd {{DAEMON}}" --build.bin "./bin/aegisd"

# Run tests across the whole workspace
test:
    go test -v -race ./...

# Full project check (lint + test)
check:
    golangci-lint run
    go test -v ./...

# Clean binaries
clean:
    rm -rf bin/
    go clean

# List available commands
help:
    @just --list