#!/usr/bin/env bash
# Package the wails .app bundle into a Finder-friendly DMG with an
# /Applications alias so users can drag-drop install.
set -euo pipefail

version_no_v="${VERSION#v}"
app="build/bin/AT Starter.app"
out="build/bin/${ARTIFACT_NAME}_${version_no_v}_${ARCH}.dmg"

# Ensure the user-facing bundle name exists. Wails writes atstarter.app
# based on wails.json:name; workflow's earlier zip step already copies
# it, but running the script alone (local build) shouldn't fail.
if [ ! -d "$app" ] && [ -d "build/bin/atstarter.app" ]; then
  cp -R "build/bin/atstarter.app" "$app"
fi
test -d "$app"
rm -f "$out"

staging="$(mktemp -d "${TMPDIR:-/tmp}/atstarter-dmg.XXXXXX")"
cleanup() { rm -rf "$staging"; }
trap cleanup EXIT

cp -R "$app" "$staging/AT Starter.app"
ln -s /Applications "$staging/Applications"

hdiutil create -volname "AT Starter" -srcfolder "$staging" -ov -format UDZO "$out"
ls -la "$out"
