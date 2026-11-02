#!/usr/bin/env bash
set -euo pipefail

REPO="corywilkerson/webp2ascii"
RELEASE_TAG="${VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
TARGET_OS=$(uname -s | tr '[:upper:]' '[:lower:]')
TARGET_ARCH=$(uname -m)
case "$TARGET_OS" in darwin|linux) ;; *) echo "Use a release archive for this OS: $TARGET_OS" >&2; exit 1 ;; esac
case "$TARGET_ARCH" in
    x86_64) TARGET_ARCH="amd64" ;;
    aarch64|arm64) TARGET_ARCH="arm64" ;;
    *) echo "Unsupported architecture: $TARGET_ARCH" >&2; exit 1 ;;
esac

if [ "$RELEASE_TAG" = latest ]; then
    RELEASE_URL=$(curl -fsSL -o /dev/null -w '%{url_effective}' "https://github.com/$REPO/releases/latest")
    RELEASE_TAG="${RELEASE_URL##*/}"
fi
RELEASE_VERSION="${RELEASE_TAG#v}"
case "$RELEASE_VERSION" in
    ""|*[!a-zA-Z0-9._-]*) echo "Invalid release version: $RELEASE_TAG" >&2; exit 1 ;;
esac
RELEASE_TAG="v$RELEASE_VERSION"
ASSET_VERSION="$RELEASE_VERSION"
# These original releases used 1.0.0 filenames and had no checksum manifest.
case "$RELEASE_VERSION" in 1.0.0|1.0.1) ASSET_VERSION="1.0.0" ;; esac
BINARY="webp2ascii-${ASSET_VERSION}-${TARGET_OS}-${TARGET_ARCH}"
ARCHIVE="$BINARY.tar.gz"
BASE_URL="https://github.com/$REPO/releases/download/$RELEASE_TAG"
DOWNLOAD_DIR=$(mktemp -d)
trap 'rm -rf "$DOWNLOAD_DIR"' EXIT

echo "Downloading webp2ascii $RELEASE_TAG..."
curl -fsSL -o "$DOWNLOAD_DIR/$ARCHIVE" "$BASE_URL/$ARCHIVE"
if [ "$RELEASE_VERSION" != 1.0.0 ] && [ "$RELEASE_VERSION" != 1.0.1 ]; then
    curl -fsSL -o "$DOWNLOAD_DIR/SHA256SUMS" "$BASE_URL/SHA256SUMS"
    EXPECTED=$(awk -v name="./$ARCHIVE" '$2 == name {print $1}' "$DOWNLOAD_DIR/SHA256SUMS")
    if command -v sha256sum >/dev/null 2>&1; then
        ACTUAL=$(sha256sum "$DOWNLOAD_DIR/$ARCHIVE" | awk '{print $1}')
    else
        ACTUAL=$(shasum -a 256 "$DOWNLOAD_DIR/$ARCHIVE" | awk '{print $1}')
    fi
    if [ -z "$EXPECTED" ] || [ "$EXPECTED" != "$ACTUAL" ]; then
        echo "Archive checksum verification failed" >&2
        exit 1
    fi
fi

tar -xzf "$DOWNLOAD_DIR/$ARCHIVE" -C "$DOWNLOAD_DIR" "$BINARY"
mkdir -p "$INSTALL_DIR"
install -m 755 "$DOWNLOAD_DIR/$BINARY" "$INSTALL_DIR/webp2ascii"
echo "Installed to $INSTALL_DIR/webp2ascii"
echo "Add $INSTALL_DIR to PATH if needed, then run webp2ascii -h"
