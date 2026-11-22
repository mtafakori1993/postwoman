#!/usr/bin/env bash
#
# Validate release artifacts and write SHA256SUMS.txt.
#
# Usage:
#   ./scripts/verify-release-assets.sh <asset-dir> <version> [all|windows|linux]
#
# Fails if an expected asset is missing, empty, has the wrong file type, or (for
# the portable ZIP / Debian package) contains the wrong payload metadata.

set -euo pipefail

DIR="${1:-}"
VERSION="${2:-}"
SCOPE="${3:-all}"

[ -n "$DIR" ] && [ -n "$VERSION" ] || {
  echo "usage: $0 <asset-dir> <version> [all|windows|linux]" >&2
  exit 2
}
[ -d "$DIR" ] || { echo "error: asset directory not found: $DIR" >&2; exit 1; }
[[ "$SCOPE" =~ ^(all|windows|linux)$ ]] || { echo "error: scope must be all, windows or linux" >&2; exit 1; }

WIN_ZIP="Postwoman-v${VERSION}-windows-x64-portable.zip"
WIN_SETUP="Postwoman-v${VERSION}-windows-x64-setup.exe"
DEB="postwoman_${VERSION}_amd64.deb"
APPIMAGE="Postwoman-${VERSION}-x86_64.AppImage"

need() {
  local file="$DIR/$1"
  [ -f "$file" ] || { echo "error: missing required asset $1" >&2; exit 1; }
  [ -s "$file" ] || { echo "error: asset is empty: $1" >&2; exit 1; }
  printf '%s' "$file"
}

check_pe() {
  local file="$1" label="$2"
  local magic
  magic="$(od -An -tx1 -N2 "$file" | tr -d '[:space:]')"
  [ "$magic" = "4d5a" ] || { echo "error: $label is not a Windows PE file ($magic)" >&2; exit 1; }
  local size
  size="$(wc -c < "$file" | tr -d '[:space:]')"
  [ "$size" -gt 1000000 ] || { echo "error: $label is unexpectedly small ($size bytes)" >&2; exit 1; }
}

if [ "$SCOPE" = "all" ] || [ "$SCOPE" = "windows" ]; then
  command -v unzip >/dev/null 2>&1 || { echo "error: unzip is required" >&2; exit 1; }
  ZIP_PATH="$(need "$WIN_ZIP")"
  SETUP_PATH="$(need "$WIN_SETUP")"
  check_pe "$SETUP_PATH" "Windows installer"
  entries="$(unzip -Z1 "$ZIP_PATH")"
  for required in \
    "Postwoman-v${VERSION}-windows-x64-portable/Postwoman.exe" \
    "Postwoman-v${VERSION}-windows-x64-portable/portable.txt" \
    "Postwoman-v${VERSION}-windows-x64-portable/README.txt" \
    "Postwoman-v${VERSION}-windows-x64-portable/LICENSE"; do
    grep -Fxq "$required" <<<"$entries" || {
      echo "error: $WIN_ZIP is missing $required" >&2
      exit 1
    }
  done
  tmp="$(mktemp -d)"
  trap 'rm -rf "$tmp"' EXIT
  unzip -q "$ZIP_PATH" "Postwoman-v${VERSION}-windows-x64-portable/Postwoman.exe" -d "$tmp"
  check_pe "$tmp/Postwoman-v${VERSION}-windows-x64-portable/Postwoman.exe" "portable Postwoman.exe"
fi

if [ "$SCOPE" = "all" ] || [ "$SCOPE" = "linux" ]; then
  command -v dpkg-deb >/dev/null 2>&1 || { echo "error: dpkg-deb is required" >&2; exit 1; }
  DEB_PATH="$(need "$DEB")"
  APPIMAGE_PATH="$(need "$APPIMAGE")"
  deb_version="$(dpkg-deb --field "$DEB_PATH" Version)"
  [ "$deb_version" = "$VERSION" ] || {
    echo "error: Debian package Version is $deb_version, expected $VERSION" >&2
    exit 1
  }
  magic="$(od -An -tx1 -N4 "$APPIMAGE_PATH" | tr -d '[:space:]')"
  [ "$magic" = "7f454c46" ] || { echo "error: $APPIMAGE is not an ELF/AppImage ($magic)" >&2; exit 1; }
  [ -x "$APPIMAGE_PATH" ] || { echo "error: $APPIMAGE is not executable" >&2; exit 1; }
fi

# Generate checksums in a stable order over every asset except the sums file.
(
  cd "$DIR"
  find . -maxdepth 1 -type f ! -name 'SHA256SUMS.txt' -exec basename {} \; |
    LC_ALL=C sort |
    while IFS= read -r file; do sha256sum "$file"; done > SHA256SUMS.txt
)
[ -s "$DIR/SHA256SUMS.txt" ] || { echo "error: SHA256SUMS.txt was not created" >&2; exit 1; }
(
  cd "$DIR"
  sha256sum -c SHA256SUMS.txt >/dev/null
)

echo "Validated $SCOPE release assets for v$VERSION:"
cat "$DIR/SHA256SUMS.txt"