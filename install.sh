#!/bin/sh
# install.sh - Install bb (Bitbucket CLI) from GitHub Releases.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/PhilipKram/Bitbucket-CLI/main/install.sh | sh
#
# Environment variables:
#   BB_INSTALL_DIR  - directory to install into (default: /usr/local/bin)
#   BB_VERSION      - version to install (default: latest)

set -e

REPO="PhilipKram/Bitbucket-CLI"
BINARY="bb"
INSTALL_DIR="${BB_INSTALL_DIR:-/usr/local/bin}"

# Detect OS
OS="$(uname -s)"
case "$OS" in
  Linux*)  OS="linux" ;;
  Darwin*) OS="darwin" ;;
  *)       echo "Error: unsupported OS: $OS" >&2; exit 1 ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64)  ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *)             echo "Error: unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

# Determine version
if [ -n "$BB_VERSION" ]; then
  VERSION="$BB_VERSION"
else
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": *"//;s/".*//')"
  if [ -z "$VERSION" ]; then
    echo "Error: could not determine latest version. No releases found." >&2
    echo "Install from source instead: go install github.com/PhilipKram/bitbucket-cli@latest" >&2
    exit 1
  fi
fi

VERSION_NUM="${VERSION#v}"
ARCHIVE="${BINARY}_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${ARCHIVE}"

echo "Installing ${BINARY} ${VERSION} (${OS}/${ARCH})..."

TMPDIR="$(mktemp -d)"
trap 'rm -rf "$TMPDIR"' EXIT

curl -fsSL "$URL" -o "${TMPDIR}/${ARCHIVE}"
tar -xzf "${TMPDIR}/${ARCHIVE}" -C "$TMPDIR"

if [ -w "$INSTALL_DIR" ]; then
  mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
  echo "Installing to ${INSTALL_DIR} (requires sudo)..."
  sudo mv "${TMPDIR}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

chmod +x "${INSTALL_DIR}/${BINARY}"

echo "Installed ${BINARY} to ${INSTALL_DIR}/${BINARY}"
"${INSTALL_DIR}/${BINARY}" --version
