#!/usr/bin/env bash
#
# Build the Linux release assets (Debian package + AppImage) from one version.
#
# Usage:
#   ./scripts/build-linux.sh
#   OUT_DIR=/tmp/postwoman-assets ./scripts/build-linux.sh
#
# Requires Go, Node.js, the Wails CLI, GTK 3 / WebKitGTK 4.1 development
# headers, dpkg-deb and mksquashfs (squashfs-tools).

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

VERSION="$(tr -d '[:space:]' < VERSION)"
[ -n "$VERSION" ] || { echo "error: VERSION is empty" >&2; exit 1; }
OUT_DIR="${OUT_DIR:-dist/linux}"

if ! command -v wails >/dev/null 2>&1; then
  gopath_bin="$(go env GOPATH 2>/dev/null || true)/bin"
  if [ -x "$gopath_bin/wails" ]; then
    PATH="$gopath_bin:$PATH"
    export PATH
  fi
fi

for tool in node npm wails dpkg-deb mksquashfs; do
  command -v "$tool" >/dev/null 2>&1 || { echo "error: $tool is required" >&2; exit 1; }
done

node scripts/update-version.mjs --check "$VERSION"

if [ "${SKIP_FRONTEND:-0}" != "1" ]; then
  echo "==> building frontend"
  (
    cd frontend
    npm ci --no-audit --no-fund
    npm run build
  )
fi

echo "==> building Linux application"
wails build \
  -clean \
  -platform linux/amd64 \
  -trimpath \
  -tags webkit2_41 \
  -ldflags "-X main.appVersion=$VERSION" \
  -m \
  -s

[ -x "build/bin/Postwoman" ] || { echo "error: build/bin/Postwoman was not produced" >&2; exit 1; }

mkdir -p "$OUT_DIR"
echo "==> building Debian package"
OUT_DIR="$OUT_DIR" ./packaging/debian/build-deb.sh
echo "==> building AppImage"
OUT_DIR="$OUT_DIR" ./packaging/appimage/build-appimage.sh

DEB="$OUT_DIR/postwoman_${VERSION}_amd64.deb"
APPIMAGE="$OUT_DIR/Postwoman-${VERSION}-x86_64.AppImage"
for artifact in "$DEB" "$APPIMAGE"; do
  [ -s "$artifact" ] || { echo "error: missing or empty $artifact" >&2; exit 1; }
done

file "$DEB" "$APPIMAGE"
echo "Linux assets in $OUT_DIR:"
ls -lh "$DEB" "$APPIMAGE"