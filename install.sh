#!/usr/bin/env bash
# shellodex installer — https://github.com/ripnet/shellodex
# Usage: curl -fsSL https://raw.githubusercontent.com/ripnet/shellodex/main/install.sh | bash
set -euo pipefail

REPO="ripnet/shellodex"
BINARY="shellodex"
INSTALL_DIR="$HOME/.local/bin"

# ── OS / arch detection ───────────────────────────────────────────────────────

case "$(uname -s)" in
    Linux*)  OS="linux"  ;;
    Darwin*) OS="darwin" ;;
    *) echo "Error: unsupported OS: $(uname -s)" >&2; exit 1 ;;
esac

case "$(uname -m)" in
    x86_64)        ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Error: unsupported architecture: $(uname -m)" >&2; exit 1 ;;
esac

# ── Resolve latest version ────────────────────────────────────────────────────

echo "Fetching latest release..."
API_RESPONSE=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest")
if command -v jq >/dev/null 2>&1; then
    VERSION=$(echo "$API_RESPONSE" | jq -r '.tag_name')
else
    VERSION=$(echo "$API_RESPONSE" | grep '"tag_name"' | head -1 \
        | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
fi
if [ -z "$VERSION" ] || [ "$VERSION" = "null" ]; then
    echo "Error: could not determine latest version." >&2
    exit 1
fi

echo "Installing ${BINARY} ${VERSION} (${OS}/${ARCH})..."

# ── Download and install ──────────────────────────────────────────────────────

ARCHIVE="${BINARY}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

curl -fsSL "$URL" -o "$TMPDIR/$ARCHIVE"
tar -xzf "$TMPDIR/$ARCHIVE" -C "$TMPDIR"

mkdir -p "$INSTALL_DIR"
mv "$TMPDIR/$BINARY" "$INSTALL_DIR/$BINARY"
chmod +x "$INSTALL_DIR/$BINARY"

echo "  Installed: $INSTALL_DIR/$BINARY"

# ── PATH setup ────────────────────────────────────────────────────────────────

if ! echo ":${PATH}:" | grep -q ":${INSTALL_DIR}:"; then
    SHELL_CONFIG="$HOME/.bashrc"
    if [ "$(basename "${SHELL:-bash}")" = "zsh" ] || [ -f "$HOME/.zshrc" ]; then
        SHELL_CONFIG="$HOME/.zshrc"
    fi
    {
        echo ""
        echo "# Added by shellodex installer"
        echo 'export PATH="$HOME/.local/bin:$PATH"'
    } >> "$SHELL_CONFIG"
    echo "  PATH updated in $SHELL_CONFIG"
fi

# ── Optional alias ────────────────────────────────────────────────────────────

add_alias() {
    SHELL_CONFIG="$HOME/.bashrc"
    if [ "$(basename "${SHELL:-bash}")" = "zsh" ] || [ -f "$HOME/.zshrc" ]; then
        SHELL_CONFIG="$HOME/.zshrc"
    fi
    {
        echo ""
        echo "alias s='shellodex'"
    } >> "$SHELL_CONFIG"
    echo "  Alias added to $SHELL_CONFIG"
}

ALIAS_REPLY="n"
if [ -t 0 ]; then
    read -r -p "Add alias 's' for shellodex to your shell config? [y/N] " ALIAS_REPLY
else
    read -r -p "Add alias 's' for shellodex to your shell config? [y/N] " ALIAS_REPLY </dev/tty || true
fi

if [[ "${ALIAS_REPLY:-n}" =~ ^[Yy]$ ]]; then
    add_alias
fi

# ── Done ──────────────────────────────────────────────────────────────────────

echo ""
echo "Done! shellodex ${VERSION} is ready."
echo ""
echo "  Restart your terminal or run:  source ~/.zshrc  (or ~/.bashrc)"
echo "  Then launch with:              shellodex"
if [[ "${ALIAS_REPLY:-n}" =~ ^[Yy]$ ]]; then
    echo "  Or use the alias:              s"
fi
echo ""
echo "  To update later, re-run this script or run:  shellodex --update"
