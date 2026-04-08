#!/usr/bin/env sh

set -eu

REPO_OWNER="${OMNETHDB_REPO_OWNER:-ubcent}"
REPO_NAME="${OMNETHDB_REPO_NAME:-omnethdb}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
VERSION="${VERSION:-latest}"

usage() {
  cat <<EOF
Install OmnethDB release binaries.

Usage:
  sh scripts/install.sh [--version v0.1.0] [--install-dir /path/to/bin]

Environment:
  VERSION      Release tag to install. Default: latest
  INSTALL_DIR  Target directory for binaries. Default: \$HOME/.local/bin

Examples:
  sh scripts/install.sh
  sh scripts/install.sh --version v0.1.0
  INSTALL_DIR=/usr/local/bin sh scripts/install.sh --version v0.1.0
EOF
}

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "error: required command not found: $1" >&2
    exit 1
  fi
}

resolve_os() {
  os="$(uname -s | tr '[:upper:]' '[:lower:]')"
  case "$os" in
    darwin|linux)
      printf '%s' "$os"
      ;;
    mingw*|msys*|cygwin*)
      printf 'windows'
      ;;
    *)
      echo "error: unsupported OS: $os" >&2
      exit 1
      ;;
  esac
}

resolve_arch() {
  arch="$(uname -m)"
  case "$arch" in
    x86_64|amd64)
      printf 'amd64'
      ;;
    arm64|aarch64)
      printf 'arm64'
      ;;
    *)
      echo "error: unsupported architecture: $arch" >&2
      exit 1
      ;;
  esac
}

resolve_latest_version() {
  need_cmd curl
  api_url="https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/releases/latest"
  tag="$(curl -fsSL "$api_url" | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)"
  if [ -z "$tag" ]; then
    echo "error: could not resolve latest release tag from $api_url" >&2
    exit 1
  fi
  printf '%s' "$tag"
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)
      VERSION="$2"
      shift 2
      ;;
    --install-dir)
      INSTALL_DIR="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "error: unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

need_cmd curl
need_cmd mktemp
need_cmd tar
OS="$(resolve_os)"
ARCH="$(resolve_arch)"

if [ "$VERSION" = "latest" ]; then
  VERSION="$(resolve_latest_version)"
fi

ARCHIVE_BASE="${REPO_NAME}_${VERSION}_${OS}_${ARCH}"
case "$OS" in
  windows)
    ARCHIVE_NAME="${ARCHIVE_BASE}.zip"
    ;;
  *)
    ARCHIVE_NAME="${ARCHIVE_BASE}.tar.gz"
    ;;
esac

DOWNLOAD_URL="https://github.com/$REPO_OWNER/$REPO_NAME/releases/download/$VERSION/$ARCHIVE_NAME"
TMP_DIR="$(mktemp -d)"
ARCHIVE_PATH="$TMP_DIR/$ARCHIVE_NAME"

cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT INT TERM

echo "==> Installing OmnethDB $VERSION for $OS/$ARCH"
echo "==> Downloading $DOWNLOAD_URL"
curl -fL "$DOWNLOAD_URL" -o "$ARCHIVE_PATH"

mkdir -p "$TMP_DIR/unpack"
case "$OS" in
  windows)
    need_cmd unzip
    unzip -q "$ARCHIVE_PATH" -d "$TMP_DIR/unpack"
    ;;
  *)
    tar -xzf "$ARCHIVE_PATH" -C "$TMP_DIR/unpack"
    ;;
esac

mkdir -p "$INSTALL_DIR"

BIN_EXT=""
if [ "$OS" = "windows" ]; then
  BIN_EXT=".exe"
fi

cp "$TMP_DIR/unpack/omnethdb$BIN_EXT" "$INSTALL_DIR/omnethdb$BIN_EXT"
cp "$TMP_DIR/unpack/omnethdb-mcp$BIN_EXT" "$INSTALL_DIR/omnethdb-mcp$BIN_EXT"

if [ "$OS" != "windows" ]; then
  chmod +x "$INSTALL_DIR/omnethdb" "$INSTALL_DIR/omnethdb-mcp"
fi

echo "==> Installed:"
echo "    $INSTALL_DIR/omnethdb$BIN_EXT"
echo "    $INSTALL_DIR/omnethdb-mcp$BIN_EXT"
echo "==> Make sure $INSTALL_DIR is on your PATH"
