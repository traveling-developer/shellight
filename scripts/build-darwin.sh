#!/usr/bin/env bash
# Cross build shellight for darwin/arm64 inside the devcontainer and assemble
# build/bin/shellight.app.
#
# Env:
#   MACOS_SDK_PATH     SDK root (default /opt/macos-sdk)
#   MACOS_MIN_VERSION  minimum macOS version (default 27.0)
#   BUILD_ID           shown in the menu (default: UTC timestamp)
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

export MACOS_SDK_PATH="${MACOS_SDK_PATH:-/opt/macos-sdk}"
export MACOS_MIN_VERSION="${MACOS_MIN_VERSION:-27.0}"

[[ -d "$MACOS_SDK_PATH/System/Library/Frameworks/AppKit.framework" ]] ||
  { echo "macOS SDK not found at $MACOS_SDK_PATH" >&2; exit 1; }

out="build/bin"
app="$out/shellight.app"
bin="$out/shellight-darwin-arm64"
mkdir -p "$out"

build_id="${BUILD_ID:-$(date -u +%Y%m%d-%H%M%S)}"

echo "==> go build darwin/arm64 (min macOS $MACOS_MIN_VERSION, build $build_id)"
GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 \
CC="$root/scripts/zcc-darwin-arm64" \
CGO_CFLAGS="-w" \
MACOSX_DEPLOYMENT_TARGET="$MACOS_MIN_VERSION" \
  go build -tags production -trimpath -buildvcs=false -ldflags="-s -w -X main.buildID=$build_id" \
    -o "$bin" ./cmd/shellight

echo "==> bundle $app"
rm -rf -- "$app"
mkdir -p "$app/Contents/MacOS" "$app/Contents/Resources"
cp "$bin" "$app/Contents/MacOS/shellight"
cp build/darwin/Info.plist "$app/Contents/Info.plist"
printf 'APPL????' > "$app/Contents/PkgInfo"

echo "==> done: $app"
