#!/usr/bin/env bash
#
# Build a Debian/Ubuntu installable package (.deb) for Postwoman.
#
# Usage:
#   packaging/debian/build-deb.sh      # -> postwoman_<version>_<arch>.deb
#
# The version is read from the single source of truth (app.go) so the package
# always matches the application. The icon is taken from the committed PNGs and
# falls back to the base64 blob in assets/ so the logo can never be lost.
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

PKG="postwoman"
ARCH="$(dpkg --print-architecture 2>/dev/null || echo amd64)"
VERSION="$(sed -n 's/^const appVersion = "\(.*\)"/\1/p' app.go | head -1)"
if [ -z "$VERSION" ]; then
  echo "error: could not read appVersion from app.go" >&2
  exit 1
fi

BIN="build/bin/Postwoman"
if [ ! -x "$BIN" ]; then
  echo "==> $BIN not found, building it"
  go build -tags desktop,production -ldflags "-w -s" -o "$BIN" .
fi

STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT
ROOTDIR="$STAGE/${PKG}_${VERSION}_${ARCH}"

install -d "$ROOTDIR/DEBIAN" \
  "$ROOTDIR/usr/bin" \
  "$ROOTDIR/usr/share/applications" \
  "$ROOTDIR/usr/share/doc/$PKG"

# --- payload -----------------------------------------------------------------
install -m 0755 "$BIN" "$ROOTDIR/usr/bin/$PKG"
install -m 0644 packaging/debian/postwoman.desktop "$ROOTDIR/usr/share/applications/$PKG.desktop"

# Icons: prefer the committed PNGs, fall back to the base64 blob, then the PNG.
ICONS="packaging/debian/icons"
for size in 16 32 48 64 128 256 512; do
  dir="$ROOTDIR/usr/share/icons/hicolor/${size}x${size}/apps"
  install -d "$dir"
  src="$ICONS/${PKG}-${size}.png"
  if [ -f "$src" ]; then
    install -m 0644 "$src" "$dir/$PKG.png"
  elif [ -f "assets/${PKG}-logo.base64" ]; then
    base64 -d "assets/${PKG}-logo.base64" > "$dir/$PKG.png"
  elif [ -f "assets/${PKG}-logo.png" ]; then
    install -m 0644 "assets/${PKG}-logo.png" "$dir/$PKG.png"
  fi
done

install -m 0644 packaging/debian/copyright "$ROOTDIR/usr/share/doc/$PKG/copyright"
if [ -f README.md ]; then
  gzip -9 -n -c README.md > "$ROOTDIR/usr/share/doc/$PKG/README.md.gz"
fi

# --- control -----------------------------------------------------------------
INSTALLED_KB="$(du -sk "$ROOTDIR" | cut -f1)"
cat > "$ROOTDIR/DEBIAN/control" <<CONTROL
Package: $PKG
Version: $VERSION
Architecture: $ARCH
Maintainer: Postwoman <mtafakori1993@users.noreply.github.com>
Section: devel
Priority: optional
Installed-Size: $INSTALLED_KB
Depends: libc6, libgtk-3-0, libglib2.0-0, libpango-1.0-0, libcairo2, libsoup-3.0-0, libwebkit2gtk-4.1-0, libjavascriptcoregtk-4.1-0
Homepage: https://github.com/mtafakori1993/postwoman
Description: API Development Platform (HTTP client)
 Postwoman is a desktop API development platform for building, sending and
 testing HTTP requests, with collections, environments, request scripts, mock
 servers and WebSocket sessions.
 .
 This package installs the desktop application (GTK 3 / WebKitGTK).
CONTROL

# --- maintainer scripts ------------------------------------------------------
cat > "$ROOTDIR/DEBIAN/postinst" <<'POSTINST'
#!/bin/sh
set -e
if command -v update-desktop-database >/dev/null 2>&1; then
  update-desktop-database -q /usr/share/applications || true
fi
if command -v gtk-update-icon-cache >/dev/null 2>&1 && [ -d /usr/share/icons/hicolor ]; then
  gtk-update-icon-cache -q -t -f /usr/share/icons/hicolor || true
fi
exit 0
POSTINST

cat > "$ROOTDIR/DEBIAN/postrm" <<'POSTRM'
#!/bin/sh
set -e
if [ "$1" = "remove" ] || [ "$1" = "purge" ]; then
  if command -v update-desktop-database >/dev/null 2>&1; then
    update-desktop-database -q /usr/share/applications || true
  fi
  if command -v gtk-update-icon-cache >/dev/null 2>&1 && [ -d /usr/share/icons/hicolor ]; then
    gtk-update-icon-cache -q -t -f /usr/share/icons/hicolor || true
  fi
fi
exit 0
POSTRM

chmod 0755 "$ROOTDIR/DEBIAN/postinst" "$ROOTDIR/DEBIAN/postrm"

# --- build -------------------------------------------------------------------
OUT="${PKG}_${VERSION}_${ARCH}.deb"
rm -f "$OUT"
dpkg-deb --build --root-owner-group "$ROOTDIR" "$OUT" >/dev/null
echo "==> built $OUT"
