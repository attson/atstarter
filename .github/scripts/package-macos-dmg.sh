#!/usr/bin/env bash
# Package the Wails app and its CLI symlink into a PKG, then wrap that PKG in
# the release DMG. A drag-copied app cannot install /usr/local/bin/atstarter.
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

work="$(mktemp -d "${TMPDIR:-/tmp}/atstarter-dmg.XXXXXX")"
staging="$work/dmg"
pkgroot="$work/pkgroot"
scripts_dir="$work/scripts"
component_pkg="$work/AT Starter.pkg"
cleanup() { rm -rf "$work"; }
trap cleanup EXIT

mkdir -p "$staging" "$pkgroot/Applications" "$pkgroot/usr/local/bin" "$scripts_dir"
ditto "$app" "$pkgroot/Applications/AT Starter.app"
ln -s "/Applications/AT Starter.app/Contents/MacOS/atstarter" "$pkgroot/usr/local/bin/atstarter"

cat > "$scripts_dir/preinstall" <<'SCRIPT'
#!/bin/sh
set -eu

command_path="/usr/local/bin/atstarter"
expected_target="/Applications/AT Starter.app/Contents/MacOS/atstarter"
if [ -L "$command_path" ]; then
  actual_target="$(readlink "$command_path")"
  if [ "$actual_target" != "$expected_target" ]; then
    echo "refusing to replace existing symlink: $command_path -> $actual_target" >&2
    exit 1
  fi
elif [ -e "$command_path" ]; then
  echo "refusing to replace existing file: $command_path" >&2
  exit 1
fi
SCRIPT
chmod 0755 "$scripts_dir/preinstall"

pkgbuild --root "$pkgroot" \
  --scripts "$scripts_dir" \
  --identifier "com.attson.atstarter" \
  --version "$version_no_v" \
  --install-location / \
  "$component_pkg"

cp "$component_pkg" "$staging/AT Starter.pkg"

hdiutil create -volname "AT Starter" -srcfolder "$staging" -ov -format UDZO "$out"
ls -la "$out"
