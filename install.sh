#!/bin/bash
# AgLight install script
# Usage: curl -fsSL https://raw.githubusercontent.com/ryubyte/aglight/master/install.sh | sh

set -e

REPO="ryubyte/aglight"
INSTALL_DIR="$HOME/.local/bin"
BINARY="aglight"

echo "AgLight — AI Agent 状态红绿灯"

# Check macOS
if [ "$(uname)" != "Darwin" ]; then
    echo "Error: AgLight only supports macOS"
    exit 1
fi

# Detect architecture
ARCH=$(uname -m)
if [ "$ARCH" = "arm64" ]; then
    ASSET="aglight-darwin-arm64"
elif [ "$ARCH" = "x86_64" ]; then
    ASSET="aglight-darwin-amd64"
else
    echo "Error: Unsupported architecture: $ARCH"
    exit 1
fi

# Get latest version
LATEST=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null | grep '"tag_name"' | head -1 | sed 's/.*"v\(.*\)".*/\1/')
if [ -z "$LATEST" ]; then
    echo "Error: Failed to get latest version from GitHub"
    exit 1
fi

DOWNLOAD_URL="https://github.com/$REPO/releases/download/v${LATEST}/${ASSET}"

echo "Installing v${LATEST} for ${ARCH}..."

# Download
echo "Downloading..."
curl -fsSL -o /tmp/${BINARY} "$DOWNLOAD_URL"
chmod +x /tmp/${BINARY}

# Install binary
echo "Installing to ${INSTALL_DIR}..."
mkdir -p "$INSTALL_DIR"
mv /tmp/${BINARY} "${INSTALL_DIR}/${BINARY}"

# Ensure INSTALL_DIR is in PATH
SHELL_RC=""
if [ -f "$HOME/.zshrc" ]; then
    SHELL_RC="$HOME/.zshrc"
elif [ -f "$HOME/.bashrc" ]; then
    SHELL_RC="$HOME/.bashrc"
fi
if [ -n "$SHELL_RC" ] && ! grep -q "$INSTALL_DIR" "$SHELL_RC" 2>/dev/null; then
    echo "" >> "$SHELL_RC"
    echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$SHELL_RC"
    echo "Added $INSTALL_DIR to PATH in $(basename $SHELL_RC)"
fi

# Start
echo "Starting AgLight..."
nohup "${INSTALL_DIR}/${BINARY}" >/dev/null 2>&1 &

echo ""
echo "✅ AgLight v${LATEST} installed!"
echo "   菜单栏应该已经出现红绿灯图标。"
echo ""
echo "Run 'aglight' to start, or add it to your login items for auto-start."
echo ""
echo "Uninstall: rm ${INSTALL_DIR}/${BINARY}"
