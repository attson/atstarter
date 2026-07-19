# Local build helpers. CI uses .github/workflows/build.yml for release artifacts.

# Ubuntu 24.04 ships libwebkit2gtk-4.1-dev only, but Wails 2.12 links against
# 4.0 by default — the 4_41 tag switches to the 4.1 pkg-config module. macOS
# and other Linux distros can drop the flag if they still have 4.0.
WAILS_TAGS ?= webkit2_41
VERSION ?= dev
LDFLAGS := -X main.Version=$(VERSION)

.PHONY: dev build build-linux build-darwin-arm64 build-darwin-amd64 build-windows test test-race clean

dev:
	wails dev -tags "$(WAILS_TAGS)"

build:
	wails build -tags "$(WAILS_TAGS)" -s -ldflags "$(LDFLAGS)"

build-linux:
	wails build -tags "$(WAILS_TAGS)" -platform linux/amd64 -s -ldflags "$(LDFLAGS)"

build-darwin-arm64:
	wails build -platform darwin/arm64 -s -ldflags "$(LDFLAGS)"

build-darwin-amd64:
	wails build -platform darwin/amd64 -s -ldflags "$(LDFLAGS)"

build-windows:
	wails build -platform windows/amd64 -nsis -s -ldflags "$(LDFLAGS)"

test:
	go test ./...
	node --test frontend/src/projectTree.test.mjs
	node --test frontend/src/composables/useTheme.test.mjs

test-race:
	go test -race ./internal/runner/

clean:
	rm -rf build/bin/atstarter build/bin/atstarter.app build/bin/atstarter.exe \
	  build/bin/atstarter-*.tar.gz build/bin/atstarter-*.zip \
	  build/bin/atstarter_*.dmg build/bin/atstarter_*.deb build/bin/atstarter_*_amd64.exe
