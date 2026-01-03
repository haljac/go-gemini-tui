#!/bin/bash
#
# Gemini TUI Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/haljac/go-gemini-tui/master/install.sh | bash
#
set -euo pipefail

REPO="haljac/go-gemini-tui"
BINARY_NAME="gemini-tui"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

info() {
    echo -e "${GREEN}==>${NC} $1"
}

warn() {
    echo -e "${YELLOW}Warning:${NC} $1"
}

error() {
    echo -e "${RED}Error:${NC} $1" >&2
    exit 1
}

# Check for required commands
command -v curl >/dev/null 2>&1 || error "curl is required but not installed"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    linux)
        OS="linux"
        ;;
    darwin)
        OS="darwin"
        ;;
    mingw*|msys*|cygwin*)
        error "Windows is not supported. Please build from source or use WSL."
        ;;
    *)
        error "Unsupported operating system: $OS"
        ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        error "Unsupported architecture: $ARCH"
        ;;
esac

info "Detected platform: ${OS}/${ARCH}"

# Get latest release version
info "Fetching latest release..."
LATEST=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/') || error "Failed to fetch latest release. Check your internet connection."

if [ -z "$LATEST" ]; then
    error "Could not determine latest version. The repository may not have any releases yet."
fi

info "Latest version: ${LATEST}"

# Construct download URL
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST}/${BINARY_NAME}-${OS}-${ARCH}"

# Download binary
info "Downloading ${BINARY_NAME} ${LATEST}..."
TMP_FILE=$(mktemp)
trap "rm -f ${TMP_FILE}" EXIT

if ! curl -fsSL "$DOWNLOAD_URL" -o "$TMP_FILE"; then
    error "Failed to download from ${DOWNLOAD_URL}"
fi

# Make executable
chmod +x "$TMP_FILE"

# Verify it runs
if ! "$TMP_FILE" --version >/dev/null 2>&1; then
    warn "Binary verification failed, but continuing with installation"
fi

# Create install directory if needed
mkdir -p "$INSTALL_DIR"

# Install
mv "$TMP_FILE" "${INSTALL_DIR}/${BINARY_NAME}"
trap - EXIT  # Clear the trap since we moved the file

info "Installed ${BINARY_NAME} to ${INSTALL_DIR}/${BINARY_NAME}"

# Check if install dir is in PATH
if ! echo "$PATH" | tr ':' '\n' | grep -q "^${INSTALL_DIR}$"; then
    echo ""
    warn "${INSTALL_DIR} is not in your PATH"
    echo ""
    echo "Add it to your shell configuration:"
    echo ""
    echo "  For bash (~/.bashrc):"
    echo "    export PATH=\"\$PATH:${INSTALL_DIR}\""
    echo ""
    echo "  For zsh (~/.zshrc):"
    echo "    export PATH=\"\$PATH:${INSTALL_DIR}\""
    echo ""
    echo "Then restart your shell or run: source ~/.bashrc (or ~/.zshrc)"
    echo ""
fi

# Final message
echo ""
info "Installation complete!"
echo ""
echo "To get started:"
echo "  1. Set your API key: export GOOGLE_API_KEY=\"your-api-key\""
echo "  2. Run: ${BINARY_NAME}"
echo ""
echo "Get an API key at: https://aistudio.google.com/apikey"
