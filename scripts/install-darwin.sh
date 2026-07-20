#!/usr/bin/env bash
# Install a new atstarter darwin build. Invoked by the running app right
# before it exits. Arguments:
#   $1 asset path (validated .dmg on disk)
#   $2 target .app path (absolute, e.g. /Applications/AT Starter.app)
#   $3 exec path used to relaunch (usually "$target/Contents/MacOS/AT Starter")
set -euo pipefail

asset="$1"
target="$2"
exec_path="$3"

log() { echo "[atstarter-install] $*" >&2; }

test -f "$asset" || { log "asset missing: $asset"; exit 1; }
test -d "$target" || { log "target .app missing: $target — dropping into ~/Applications"; }

# Mount the DMG read-only; we cp the .app out, then unmount.
mount_point="$(hdiutil attach -nobrowse -noverify -readonly "$asset" | awk 'END{print $NF}')"
trap 'hdiutil detach "$mount_point" -quiet -force >/dev/null 2>&1 || true' EXIT

src_app=""
while IFS= read -r -d '' candidate; do
  src_app="$candidate"
  break
done < <(find "$mount_point" -maxdepth 2 -type d -name "*.app" -print0)

test -n "$src_app" || { log "no .app found in dmg"; exit 1; }

# Fall back to ~/Applications if we can't write to the existing location.
if [ -d "$target" ] && [ -w "$(dirname "$target")" ]; then
  rm -rf "$target"
  dest="$target"
elif [ -w "/Applications" ]; then
  dest="/Applications/$(basename "$src_app")"
  rm -rf "$dest"
else
  mkdir -p "$HOME/Applications"
  dest="$HOME/Applications/$(basename "$src_app")"
  rm -rf "$dest"
fi

cp -R "$src_app" "$dest"
log "installed at $dest"

# macOS quarantines DMG contents; strip so the app opens without a warning.
xattr -dr com.apple.quarantine "$dest" >/dev/null 2>&1 || true

# Relaunch by opening the destination bundle so Launch Services picks up
# the new binary. `open -n` starts a new instance.
open -n "$dest" || true
