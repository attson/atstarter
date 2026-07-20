#!/usr/bin/env bash
# Install a new atstarter linux build. Invoked by the running app right
# before it exits. Arguments:
#   $1 asset path (validated .tar.gz on disk)
#   $2 target binary path (absolute; typically the running binary)
#   $3 exec path used to relaunch (same as $2 in practice)
set -euo pipefail

asset="$1"
target="$2"
exec_path="$3"

log() { echo "[atstarter-install] $*" >&2; }

test -f "$asset" || { log "asset missing: $asset"; exit 1; }

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

tar -xzf "$asset" -C "$tmp"

# Tarball contains a single binary named "AT Starter".
src=""
while IFS= read -r -d '' candidate; do
  src="$candidate"
  break
done < <(find "$tmp" -maxdepth 2 -type f \( -name "AT Starter" -o -name "atstarter" \) -print0)

test -n "$src" || { log "no binary found in tarball"; exit 1; }
chmod +x "$src"

# Prefer writing over the existing target; fall back to ~/.local/bin when
# not writable (system-managed install).
dest="$target"
if [ ! -w "$(dirname "$target")" ]; then
  mkdir -p "$HOME/.local/bin"
  dest="$HOME/.local/bin/$(basename "$target")"
fi

install -m 0755 "$src" "$dest"
log "installed at $dest"

# Detach and relaunch so the parent can exit cleanly.
setsid "$dest" </dev/null >/dev/null 2>&1 &
