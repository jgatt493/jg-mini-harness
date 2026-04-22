#!/usr/bin/env bash
set -euo pipefail

# Install jg-mini-harness
# Requires: Go 1.21+

INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
REPO_URL="https://github.com/jgatt493/jg-mini-harness.git"
TMP_DIR="$(mktemp -d)"

cleanup() { rm -rf "$TMP_DIR"; }
trap cleanup EXIT

echo "Installing jg-mini-harness..."

# Check Go is available
if ! command -v go &>/dev/null; then
  echo "Error: Go is required but not installed." >&2
  echo "Install Go from https://go.dev/dl/" >&2
  exit 1
fi

# Clone and build
git clone --quiet "$REPO_URL" "$TMP_DIR/jg-mini-harness"
cd "$TMP_DIR/jg-mini-harness"
go build -o harness ./cmd/harness

# Install
mkdir -p "$INSTALL_DIR"
cp harness "$INSTALL_DIR/harness"
echo "Installed harness to $INSTALL_DIR/harness"

# Verify
if command -v harness &>/dev/null; then
  echo "$(harness version)"
else
  echo ""
  echo "Note: $INSTALL_DIR is not in your PATH."
  echo "Add it with: export PATH=\"$INSTALL_DIR:\$PATH\""
fi
