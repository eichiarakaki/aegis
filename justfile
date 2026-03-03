set shell := ["sh", "-c"]

# Project binaries
DAEMON := "./cmd/aegisd/*"
CLI    := "./cmd/aegis/*"

INSTALL_DIR := "~/.local/bin"

# --- Build Commands ---
# Build everything
all: build-daemon build-cli

# Build only the daemon (aegisd)
build-daemon:
    go build -o bin/aegisd {{DAEMON}}

# Build only the CLI (aegis)
build-cli:
    go build -o bin/aegis {{CLI}}

# Install both binaries to ~/go/bin (or change INSTALL_DIR to /usr/local/bin etc.)
install: build-daemon build-cli
    @mkdir -p {{INSTALL_DIR}}
    cp bin/aegisd {{INSTALL_DIR}}/aegisd
    cp bin/aegis  {{INSTALL_DIR}}/aegis
    @echo "Installed:"
    @which aegisd  || echo "  → aegisd not found in PATH"
    @which aegis   || echo "  → aegis not found in PATH"
    @echo ""
    @echo "Make sure {{INSTALL_DIR}} is in your PATH"

# Uninstall (remove from ~/go/bin or /usr/local/bin)
uninstall:
    rm -f {{INSTALL_DIR}}/aegisd
    rm -f {{INSTALL_DIR}}/aegis
    @echo "Removed aegisd and aegis from {{INSTALL_DIR}}"

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