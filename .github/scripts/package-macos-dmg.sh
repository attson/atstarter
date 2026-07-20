#!/usr/bin/env bash
# Package the wails .app bundle into a Finder-friendly DMG with an
# /Applications alias so users can drag-drop install.
set -euo pipefail

version_no_v="${VERSION#v}"
app="build/bin/AT Starter.app"
out="build/bin/${ARTIFACT_NAME}_${version_no_v}_${ARCH}.dmg"

test -d "$app"
rm -f "$out"

staging="$(mktemp -d "${TMPDIR:-/tmp}/atstarter-dmg.XXXXXX")"
cleanup() { rm -rf "$staging"; }
trap cleanup EXIT

cp -R "$app" "$staging/AT Starter.app"
ln -s /Applications "$staging/Applications"

hdiutil create -volname "AT Starter" -srcfolder "$staging" -ov -format UDZO "$out"
ls -la "$out"
