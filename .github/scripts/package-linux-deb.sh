#!/usr/bin/env bash
# Build a minimal .deb around the wails linux binary. Depends on the same
# GTK/WebKit runtime libs the CI builder used at compile time.
set -euo pipefail

version_no_v="${VERSION#v}"
bin="build/bin/rootyard"
out="build/bin/${ARTIFACT_NAME}_${version_no_v}_${ARCH}.deb"
root="$(mktemp -d)"
trap 'rm -rf "$root"' EXIT

test -f "$bin"

install -Dm755 "$bin" "$root/usr/bin/rootyard"
install -Dm644 build/appicon.png "$root/usr/share/pixmaps/Rootyard.png"
install -Dm644 build/appicon.png "$root/usr/share/icons/hicolor/1024x1024/apps/Rootyard.png"
install -Dm644 /dev/stdin "$root/usr/share/applications/Rootyard.desktop" <<DESKTOP
[Desktop Entry]
Type=Application
Name=Rootyard
Exec=rootyard
Icon=Rootyard
Terminal=false
Categories=Development;Utility;
DESKTOP

installed_size="$(du -sk "$root/usr" | awk '{print $1}')"
mkdir -p "$root/DEBIAN"
cat > "$root/DEBIAN/control" <<CONTROL
Package: rootyard
Version: ${version_no_v}
Section: devel
Priority: optional
Architecture: ${ARCH}
Maintainer: liuzaisen <liuzaisen@wanxinbuzhi.com>
Installed-Size: ${installed_size}
Depends: libgtk-3-0, libwebkit2gtk-4.1-0
Description: Rootyard desktop app
 Local project workspace (Wails + Vue3) for projects, launch commands,
 logs, Docker, files, Git state, and AI-facing controls.
CONTROL

dpkg-deb --build --root-owner-group "$root" "$out"
ls -la "$out"
