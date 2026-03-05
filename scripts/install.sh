#!/bin/bash
#
# Aetheris Quick Install Script
# Usage: curl -sSL https://raw.githubusercontent.com/Colin4k1024/Aetheris/main/scripts/install.sh | bash
#

set -e

VERSION="v2.3.0"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.aetheris}"
BIN_DIR="$INSTALL_DIR/bin"

# Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

# Map architecture
case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Map OS
case "$OS" in
    darwin) OS="darwin" ;;
    linux) OS="linux" ;;
    *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

TAR_NAME="aetheris-${VERSION}-${OS}-${ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/Colin4k1024/Aetheris/releases/download/${VERSION}/${TAR_NAME}"

echo "Installing Aetheris ${VERSION}..."

# Create installation directory
mkdir -p "$BIN_DIR"

# Download and extract
if command -v curl &> /dev/null; then
    DOWNLOAD_CMD="curl -sSL"
elif command -v wget &> /dev/null; then
    DOWNLOAD_CMD="wget -q -O -"
else
    echo "Error: curl or wget is required"
    exit 1
fi

echo "Downloading from $DOWNLOAD_URL..."

# Try to download, fall back to building from source if release not available
if ! $DOWNLOAD_CMD -o /tmp/aetheris.tar.gz "$DOWNLOAD_URL" 2>/dev/null; then
    echo "Release not found, building from source..."
    cd /tmp
    rm -rf Aetheris
    git clone --depth 1 --branch "${VERSION}" https://github.com/Colin4k1024/Aetheris.git
    cd Aetheris
    go build -o "$BIN_DIR/aetheris" ./cmd/cli
    echo "Build complete!"
else
    tar -xzf /tmp/aetheris.tar.gz -C "$BIN_DIR"
    rm /tmp/aetheris.tar.gz
fi

# Add to PATH
SHELL_RC="$HOME/.bashrc"
if [ -f "$HOME/.zshrc" ]; then
    SHELL_RC="$HOME/.zshrc"
fi

if ! grep -q "$BIN_DIR" "$SHELL_RC" 2>/dev/null; then
    echo "" >> "$SHELL_RC"
    echo "# Aetheris" >> "$SHELL_RC"
    echo "export PATH=\"$BIN_DIR:\$PATH\"" >> "$SHELL_RC"
fi

export PATH="$BIN_DIR:$PATH"

# Verify installation
if command -v aetheris &> /dev/null; then
    echo ""
    echo "✅ Aetheris installed successfully!"
    echo ""
    aetheris --version
    echo ""
    echo "Run 'aetheris init' to scaffold a new agent project"
    echo "Or start the API: go run ./cmd/api"
else
    echo "⚠️  Please add $BIN_DIR to your PATH:"
    echo "   export PATH=\"$BIN_DIR:\$PATH\""
fi
