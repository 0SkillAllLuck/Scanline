#!/usr/bin/env bash
# Packages a Scanline.app bundle into a drag-to-install .dmg.
#
# Usage: ./scripts/build-dmg.sh <Scanline.app> <output.dmg>
#   e.g. ./scripts/build-dmg.sh ./result/Scanline.app dev.skillless.Scanline.aarch64.dmg
#
# Stages the .app alongside an Applications symlink so users see the classic
# drag-to-install layout, then writes a UDZO-compressed DMG via hdiutil.
set -euo pipefail

if [ $# -ne 2 ]; then
    echo "Usage: $0 <Scanline.app> <output.dmg>" >&2
    exit 1
fi

APP_PATH="$1"
DMG_PATH="$2"

if [ ! -d "$APP_PATH" ]; then
    echo "Error: $APP_PATH is not a directory" >&2
    exit 1
fi

if ! command -v hdiutil >/dev/null 2>&1; then
    echo "Error: hdiutil not found (this script must run on macOS)" >&2
    exit 1
fi

STAGE_DIR=$(mktemp -d)
# Files copied from /nix/store are read-only, so rm -rf alone fails on them;
# strip write protection before cleanup.
trap 'chmod -R u+w "$STAGE_DIR" 2>/dev/null; rm -rf "$STAGE_DIR"' EXIT

cp -R "$APP_PATH" "$STAGE_DIR/"
ln -s /Applications "$STAGE_DIR/Applications"

rm -f "$DMG_PATH"
hdiutil create \
    -volname Scanline \
    -srcfolder "$STAGE_DIR" \
    -fs HFS+ \
    -format UDZO \
    -imagekey zlib-level=9 \
    "$DMG_PATH"

echo "Wrote $DMG_PATH"
