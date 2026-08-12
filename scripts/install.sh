#!/usr/bin/env bash
# unleash installer (Linux / macOS)
# Usage: curl -fsSL https://raw.githubusercontent.com/NetVar1337/unleash/main/scripts/install.sh | bash
set -euo pipefail

REPO="NetVar1337/unleash"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
mkdir -p "$INSTALL_DIR"

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$os" in
  linux)  plat="linux" ;;
  darwin) plat="darwin" ;;
  *) echo "unsupported OS: $os" >&2; exit 1 ;;
esac
case "$arch" in
  x86_64|amd64)  suffix="amd64" ;;
  arm64|aarch64) suffix="arm64" ;;
  *) echo "unsupported arch: $arch" >&2; exit 1 ;;
esac

api="https://api.github.com/repos/$REPO/releases?per_page=30"
json=$(curl -fsSL "$api")

latest_tag() { # $1 = tag prefix
  printf '%s' "$json" | grep -o "\"tag_name\": *\"[^\"]*\"" | sed 's/.*"\([^"]*\)"$/\1/' \
    | grep "^$1" | head -1 || true
}

asset_url() { # $1 = tag, $2 = asset name
  curl -fsSL "https://api.github.com/repos/$REPO/releases/tags/$1" \
    | grep -o "\"browser_download_url\": *\"[^\"]*$2\"" | head -1 | sed 's/.*"\(http[^"]*\)"$/\1/'
}

install_one() { # $1 = prefix, $2 = asset, $3 = dest name
  local tag url
  tag=$(latest_tag "$1")
  if [ -z "$tag" ]; then
    echo "  ! no release for tag prefix '$1' — skipped $3" >&2
    return
  fi
  url=$(asset_url "$tag" "$2")
  if [ -z "$url" ]; then
    echo "  ! release $tag has no asset '$2' — skipped $3" >&2
    return
  fi
  echo "  downloading $2 ($tag) -> $INSTALL_DIR/$3"
  curl -fsSL "$url" -o "$INSTALL_DIR/$3"
  chmod +x "$INSTALL_DIR/$3"
}

echo "unleash installer ($plat-$suffix)"
echo "  install dir: $INSTALL_DIR"
install_one "cc-v"  "unleash-$plat-$suffix"      "unleash"
install_one "gpt-v" "unleash-gpt-$plat-$suffix"  "unleash-gpt"
install_one "omp-v" "unleash-omp-$plat-$suffix"  "unleash-omp"

case ":$PATH:" in
  *":$INSTALL_DIR:"*) ;;
  *) echo "  note: add $INSTALL_DIR to PATH" ;;
esac

echo
echo "Done. Next:"
echo "  unleash setup        # Claude Code"
echo "  unleash-gpt setup    # Codex CLI"
echo "  unleash-omp setup    # Oh-My-Pi"
