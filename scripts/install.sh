#!/bin/sh
# mifo installer: downloads the latest release binary into your PATH.
set -eu

REPO="literalmock/mifo"
VERSION="${MIFO_VERSION:-latest}"

[ "$(uname -s)" = "Darwin" ] || { echo "mifo is macOS only (uwu sorry)"; exit 1; }

case "$(uname -m)" in
  arm64) ARCH=arm64 ;;
  x86_64) ARCH=amd64 ;;
  *) echo "unsupported arch: $(uname -m)"; exit 1 ;;
esac

if [ "$VERSION" = "latest" ]; then
  URL="https://github.com/$REPO/releases/latest/download/mifo-darwin-$ARCH"
else
  URL="https://github.com/$REPO/releases/download/$VERSION/mifo-darwin-$ARCH"
fi

DIR="${MIFO_BIN_DIR:-/usr/local/bin}"
if ! mkdir -p "$DIR" 2>/dev/null || [ ! -w "$DIR" ]; then
  [ -n "${MIFO_BIN_DIR:-}" ] && { echo "cannot write to $DIR"; exit 1; }
  DIR="$HOME/.local/bin"
  mkdir -p "$DIR"
fi

TMP="$(mktemp)"
trap 'rm -f "$TMP"' EXIT
echo "-> downloading mifo ($ARCH)"
curl -fsSL "$URL" -o "$TMP"
chmod +x "$TMP"
mv "$TMP" "$DIR/mifo"
echo "installed: $DIR/mifo"

case ":$PATH:" in
  *":$DIR:"*) echo "run it: mifo  ✨" ;;
  *) echo "add to PATH: echo 'export PATH=\"$DIR:\$PATH\"' >> ~/.zshrc && source ~/.zshrc" ;;
esac
