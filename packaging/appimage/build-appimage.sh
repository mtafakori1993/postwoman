#!/usr/bin/env bash
#
# Build a portable Linux AppImage for Postwoman.
#
# Usage:
#   packaging/appimage/build-appimage.sh   # -> Postwoman-<version>-x86_64.AppImage
#
# The AppImage type-2 runtime is vendored in packaging/appimage/runtime-x86_64,
# so no external appimagetool is required (the one previously bundled in this
# repository was truncated and could not run). The payload is a plain SquashFS
# image produced with mksquashfs and prepended with the runtime.
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

VERSION="$(tr -d '[:space:]' < VERSION)"
[ -n "$VERSION" ] || { echo "error: could not read VERSION" >&2; exit 1; }
OUT_DIR="${OUT_DIR:-.}"
mkdir -p "$OUT_DIR"

ARCH="x86_64"
RUNTIME="packaging/appimage/runtime-x86_64"
[ -f "$RUNTIME" ] || { echo "error: missing $RUNTIME" >&2; exit 1; }

BIN="build/bin/Postwoman"
if [ ! -x "$BIN" ]; then
  echo "==> $BIN not found, building it"
  go build -tags desktop,production,webkit2_41 -trimpath -ldflags "-w -s -X main.appVersion=$VERSION" -o "$BIN" .
fi

echo "==> refreshing AppDir"
install -d AppDir/usr/bin AppDir/usr/share/applications \
  AppDir/usr/share/icons/hicolor/256x256/apps

install -m 0755 "$BIN" AppDir/usr/bin/Postwoman
rm -f AppDir/Postwoman-*-x86_64.desktop AppDir/usr/share/applications/Postwoman-*-x86_64.desktop
install -m 0644 packaging/debian/postwoman.desktop "AppDir/Postwoman-${VERSION}-${ARCH}.desktop"
install -m 0644 packaging/debian/postwoman.desktop AppDir/usr/share/applications/postwoman.desktop
install -m 0644 packaging/debian/postwoman.desktop \
  "AppDir/usr/share/applications/Postwoman-${VERSION}-${ARCH}.desktop"
install -m 0644 packaging/debian/icons/postwoman-256.png AppDir/postwoman.png
cp AppDir/postwoman.png AppDir/.DirIcon
install -m 0644 packaging/debian/icons/postwoman-256.png \
  AppDir/usr/share/icons/hicolor/256x256/apps/postwoman.png

if [ ! -f AppDir/AppRun ]; then
  cat > AppDir/AppRun <<'APPRUN'
#!/bin/sh
HERE="$(dirname "$(readlink -f "$0")")"
export LD_LIBRARY_PATH="$HERE/usr/lib:$LD_LIBRARY_PATH"
exec "$HERE/usr/bin/Postwoman" "$@"
APPRUN
  chmod 0755 AppDir/AppRun
fi

echo "==> packing SquashFS"
SQUASH="$(mktemp)"
trap 'rm -f "$SQUASH"' EXIT
mksquashfs AppDir "$SQUASH" -root-owned -noappend -comp gzip -no-progress >/dev/null

OUT="${OUT_DIR}/Postwoman-${VERSION}-${ARCH}.AppImage"
rm -f "$OUT"
cat "$RUNTIME" "$SQUASH" > "$OUT"
chmod 0755 "$OUT"

echo "==> built $OUT"
