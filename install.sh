#!/usr/bin/env bash
set -e

echo -e "\033[35m═══════════════════════════════════════════\033[0m"
echo -e "\033[36m         Installing agpass CLI             \033[0m"
echo -e "\033[35m═══════════════════════════════════════════\033[0m\n"

INSTALL_DIR="$HOME/.agpass/bin"
mkdir -p "$INSTALL_DIR"
TARGET="$INSTALL_DIR/agpass"

if [ -f "main.go" ]; then
    echo "Building from local repository..."
    go build -ldflags="-s -w" -o "$TARGET" .
elif command -v go >/dev/null 2>&1; then
    echo "Installing via 'go install'..."
    go install agpass@latest
    GOPATH_BIN="$(go env GOPATH)/bin/agpass"
    if [ -f "$GOPATH_BIN" ]; then
        cp "$GOPATH_BIN" "$TARGET"
    fi
else
    echo "Go is required. Please install Go or download pre-built binary."
    exit 1
fi

chmod +x "$TARGET"

# Install AI Agent Skill
"$TARGET" install-skill || true

echo -e "\n\033[32m✓ Installation complete! Add $INSTALL_DIR to your PATH if not already present:\033[0m"
echo "  export PATH=\"\$HOME/.agpass/bin:\$PATH\""