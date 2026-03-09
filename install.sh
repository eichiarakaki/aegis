#!/bin/sh
set -e

REPO="eichiarakaki/aegis"
INSTALL_DIR="${HOME}/.local/bin"
CONFIG_DIR="${HOME}/.config/aegis"
CONFIG_FILE="${CONFIG_DIR}/aegis.yaml"

# Detect OS and arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case $ARCH in
    x86_64)  ARCH="amd64" ;;
    aarch64) ARCH="arm64" ;;
    arm64)   ARCH="arm64" ;;
    *)       echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Get latest release version
VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' | cut -d'"' -f4)

echo "Installing Aegis ${VERSION} (${OS}/${ARCH})..."

# Download binaries
mkdir -p "${INSTALL_DIR}"
for BINARY in aegisctl aegisd aegis-fetcher; do
    URL="https://github.com/${REPO}/releases/download/${VERSION}/${BINARY}-${OS}-${ARCH}"
    curl -fsSL "${URL}" -o "${INSTALL_DIR}/${BINARY}"
    chmod +x "${INSTALL_DIR}/${BINARY}"
    echo "Installed ${BINARY}"
done

# Install default config (never overwrite existing)
mkdir -p "${CONFIG_DIR}"
if [ ! -f "${CONFIG_FILE}" ]; then
    curl -fsSL "https://raw.githubusercontent.com/${REPO}/main/config/aegis.yaml" \
        -o "${CONFIG_FILE}"
    echo "Created ${CONFIG_FILE}"
else
    echo "Skipped ${CONFIG_FILE} (already exists)"
fi

# PATH check
case ":${PATH}:" in
    *":${INSTALL_DIR}:"*) ;;
    *)
        echo ""
        echo "Add this to your shell profile (~/.bashrc or ~/.zshrc):"
        echo "  export PATH=\"\${HOME}/.local/bin:\${PATH}\""
        ;;
esac

echo ""
echo "Done. Edit ${CONFIG_FILE} to set your data_path."
echo "Run 'aegisctl --help' to get started."