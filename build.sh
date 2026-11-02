#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

APP_NAME="webp2ascii"
BUILD_VERSION="${VERSION:-dev}"
BUILD_VERSION="${BUILD_VERSION#v}"
case "$BUILD_VERSION" in
    ""|*[!a-zA-Z0-9._-]*) echo "Invalid VERSION: $BUILD_VERSION" >&2; exit 1 ;;
esac

BUILD_DIR=$(mktemp -d)
trap 'rm -rf "$BUILD_DIR"' EXIT
rm -rf dist/
mkdir -p dist/

for PLATFORM in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64; do
    TARGET_OS="${PLATFORM%/*}"
    TARGET_ARCH="${PLATFORM#*/}"
    BINARY="${APP_NAME}-${BUILD_VERSION}-${TARGET_OS}-${TARGET_ARCH}"
    if [ "$TARGET_OS" = windows ]; then BINARY="${BINARY}.exe"; fi
    echo "Building $PLATFORM ($BUILD_VERSION)..."
    CGO_ENABLED=0 GOOS="$TARGET_OS" GOARCH="$TARGET_ARCH" go build \
        -trimpath -ldflags="-s -w -X main.version=$BUILD_VERSION" -o "$BUILD_DIR/$BINARY" .
    tar -czf "dist/${BINARY}.tar.gz" -C "$BUILD_DIR" "$BINARY"
done

(cd dist && shasum -a 256 ./*.tar.gz > SHA256SUMS)
echo "Builds and SHA256SUMS are in dist/"
