#!/usr/bin/env bash
# Build a minimal .deb around the wails linux binary. Depends on the same
# GTK/WebKit runtime libs the CI builder used at compile time.
set -euo pipefail

version_no_v="${VERSION#v}"
bin="build/bin/atstarter"
out="build/bin/${ARTIFACT_NAME}_${version_no_v}_${ARCH}.deb"
root="$(mktemp -d)"
trap 'rm -rf "$root"' EXIT

test -f "$bin"

install -Dm755 "$bin" "$root/usr/bin/atstarter"
install -Dm644 build/appicon.png "$root/usr/share/pixmaps/atstarter.png"
install -Dm644 build/appicon.png "$root/usr/share/icons/hicolor/1024x1024/apps/atstarter.png"
install -Dm644 /dev/stdin "$root/usr/share/applications/atstarter.desktop" <<DESKTOP
[Desktop Entry]
Type=Application
Name=atstarter
Exec=atstarter
Icon=atstarter
Terminal=false
Categories=Development;Utility;
DESKTOP

installed_size="$(du -sk "$root/usr" | awk '{print $1}')"
mkdir -p "$root/DEBIAN"
cat > "$root/DEBIAN/control" <<CONTROL
Package: atstarter
Version: ${version_no_v}
Section: devel
Priority: optional
Architecture: ${ARCH}
Maintainer: liuzaisen <liuzaisen@wanxinbuzhi.com>
Installed-Size: ${installed_size}
Depends: libgtk-3-0, libwebkit2gtk-4.1-0
Description: atstarter desktop app
 Local project launcher (Wails + Vue3) that scans workspaces, detects
 project types, and starts / stops them from a single window.
CONTROL

dpkg-deb --build --root-owner-group "$root" "$out"
ls -la "$out"
