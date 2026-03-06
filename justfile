set shell := ["sh", "-c"]

# Project binaries
DAEMON := "./cmd/aegisd/*"
CTL    := "./cmd/aegisctl"

INSTALL_DIR := "~/.local/bin"

# --- Build Commands ---
# Build everything
all: build-daemon build-ctl

# Build only the daemon (aegisd)
build-daemon:
    go build -o bin/aegisd {{DAEMON}}

# Build only the ctl (aegis)
build-ctl:
    go build -o bin/aegisctl {{CTL}}

# Install both binaries to ~/go/bin (or change INSTALL_DIR to /usr/local/bin etc.)
install: build-daemon build-ctl
    @mkdir -p {{INSTALL_DIR}}
    cp bin/aegisd {{INSTALL_DIR}}/aegisd
    cp bin/aegisctl  {{INSTALL_DIR}}/aegisctl
    @echo "Installed:"
    @which aegisd  || echo "  → aegisd not found in PATH"
    @which aegisctl   || echo "  → aegisctl not found in PATH"
    @echo ""
    @echo "Make sure {{INSTALL_DIR}} is in your PATH"

# Uninstall (remove from ~/go/bin or /usr/local/bin)
uninstall:
    rm -f {{INSTALL_DIR}}/aegisd
    rm -f {{INSTALL_DIR}}/aegisctl
    @echo "Removed aegisd and aegisctl from {{INSTALL_DIR}}"

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