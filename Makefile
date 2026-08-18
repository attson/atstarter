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

# frontend 测试用 find 展开而不是逐个列名:手工列表漏登记过 fileBrowserSearch /
# groupMemberStatus / typeLabel 三个文件,新增的测试从此自动纳入。
# 用 find 而非 `node --test <dir>`,因为 CI 的 Node 20 不支持目录参数。
FRONTEND_TESTS = $(shell find frontend/src -name '*.test.mjs' | sort)

# 演示站的测试单独点名:同目录下的 homeDemoBundle 需要先构建站点,不适合放进
# 这个目标。
DEMO_TESTS = site/docs/.vitepress/theme/components/mockWailsCoverage.test.mjs

test:
	go test ./...
	node --test $(FRONTEND_TESTS)
	node --test $(DEMO_TESTS)

test-race:
	go test -race ./internal/runner/

clean:
	rm -rf build/bin/atstarter build/bin/atstarter.app build/bin/atstarter.exe \
	  "build/bin/AT Starter" "build/bin/AT Starter.app" "build/bin/AT Starter.exe" \
	  build/bin/AT-Starter-*.tar.gz build/bin/AT-Starter-*.zip \
	  build/bin/AT-Starter_*.dmg build/bin/AT-Starter_*.deb build/bin/AT-Starter_*_amd64.exe
