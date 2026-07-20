#!/usr/bin/env bash
# Build a minimal .deb around the wails linux binary. Depends on the same
# GTK/WebKit runtime libs the CI builder used at compile time.
set -euo pipefail

version_no_v="${VERSION#v}"
bin="build/bin/AT Starter"
out="build/bin/${ARTIFACT_NAME}_${version_no_v}_${ARCH}.deb"
root="$(mktemp -d)"
trap 'rm -rf "$root"' EXIT

test -f "$bin"

install -Dm755 "$bin" "$root/usr/bin/AT-Starter"
install -Dm644 build/appicon.png "$root/usr/share/pixmaps/AT-Starter.png"
install -Dm644 build/appicon.png "$root/usr/share/icons/hicolor/1024x1024/apps/AT-Starter.png"
install -Dm644 /dev/stdin "$root/usr/share/applications/AT-Starter.desktop" <<DESKTOP
[Desktop Entry]
Type=Application
Name=AT Starter
Exec=AT-Starter
Icon=AT-Starter
Terminal=false
Categories=Development;Utility;
DESKTOP

installed_size="$(du -sk "$root/usr" | awk '{print $1}')"
mkdir -p "$root/DEBIAN"
cat > "$root/DEBIAN/control" <<CONTROL
Package: at-starter
Version: ${version_no_v}
Section: devel
Priority: optional
Architecture: ${ARCH}
Maintainer: liuzaisen <liuzaisen@wanxinbuzhi.com>
Installed-Size: ${installed_size}
Depends: libgtk-3-0, libwebkit2gtk-4.1-0
Description: AT Starter desktop app
 Local project launcher (Wails + Vue3) that scans workspaces, detects
 project types, and starts / stops them from a single window.
CONTROL

dpkg-deb --build --root-owner-group "$root" "$out"
ls -la "$out"
