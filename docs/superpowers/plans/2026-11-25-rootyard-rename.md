# Rootyard Rename Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the AT Starter product and technical identity with Rootyard across the desktop app, CLI/MCP, AI integrations, installers, updater, release pipeline, website, and documentation without changing existing product behavior.

**Architecture:** Treat the rename as a coordinated contract migration. Establish the runtime identity first, then migrate each independently testable integration boundary, and finish with documentation plus a whole-repository legacy-name audit. Keep configuration and runtime schemas unchanged; Rootyard uses new paths and identifiers without reading or deleting legacy state.

**Tech Stack:** Go 1.24, Wails v2, Vue 3/Vite, Node.js 20 test runner, npm, VitePress, shell/PowerShell packaging scripts, NSIS, GitHub Actions.

**Spec:** `docs/superpowers/specs/2026-11-25-rootyard-rename-design.md`

## Global Constraints

- Canonical product name: `Rootyard`; canonical executable and CLI: `rootyard` (`rootyard.exe` on Windows).
- Canonical positioning: `本地项目工作台`; canonical tagline: `Your local project workspace`.
- Canonical Go module: `rootyard`; bundle IDs: `com.attson.rootyard` and `com.attson.rootyard.dev`.
- Canonical config: `<UserConfigDir>/rootyard/config.json`; override: `ROOTYARD_CONFIG`.
- Canonical AI identities: `rootyard_*`, `@attson/rootyard-mcp`, `rootyard-control`, and `use-rootyard`.
- Canonical repository/site: `attson/rootyard` and `https://attson.github.io/rootyard/`.
- Canonical update secrets: `ROOTYARD_UPDATE_VERIFY_PUBLIC_KEY` and `ROOTYARD_UPDATE_SIGNING_PRIVATE_KEY`.
- Do not add command, environment-variable, MCP-tool, npm-package, plugin, or config-path compatibility aliases.
- Do not read, migrate, rename, or delete legacy config, cache, installation, or shortcut files.
- Do not change persisted data schemas, Wails method signatures, CLI JSON envelopes, MCP schemas, or existing runtime behavior.
- Keep Git history and semantic version progression; the first Rootyard release is `v0.6.0`.
- Do not perform GitHub repository rename, secret changes, npm publication, Pages settings changes, tagging, or release publication from the implementation branch.
- Use `/home/attson/sdk/go1.24.13/bin/go` for repository Go commands, as required by `CLAUDE.md`.
- Commit messages must not contain `Co-Authored-By`.

## File and Responsibility Map

| Boundary | Files | Responsibility |
|----------|-------|----------------|
| Core identity | `go.mod`, Go importers, `wails.json`, `main.go`, `app.go`, `internal/control/protocol.go` | Module name, display name, config path, control errors |
| CLI/MCP | `cli.go`, `mcp.go`, `control_server_test.go`, new `mcp_test.go` | Automation command copy, MCP server/tool identity |
| npm launcher | `npm/rootyard-mcp/**` | Locate the installed Rootyard binary and launch `rootyard mcp` |
| Agent plugin | `plugins/rootyard-control/**`, marketplace manifests, validator | Rootyard MCP/plugin/skill installation contract |
| Native packaging | platform installers, Wails metadata, native tray bridge, `release_contract_test.go` | Installed executable, application paths, PATH, bundle IDs |
| Update/release | `updater.go`, updater tests, GitHub workflow, signing script | Repository lookup, cache, release assets, verification secrets, npm publication path |
| Product surfaces | frontend title/copy, VitePress site/config/assets/tests | Rootyard user-facing web and desktop identity |
| Project guidance | `README.md`, `CLAUDE.md`, `docs/**` | Current product positioning, commands, long-lived contracts, historical file names |

---

### Task 1: Establish the Core Rootyard Runtime Identity

**Files:**
- Modify: `go.mod`
- Modify: `wails.json`
- Modify: `app.go`
- Modify: `app_test.go`
- Modify: `main.go`
- Modify: `internal/control/protocol.go`
- Create: `internal/control/protocol_test.go`
- Modify imports in: `app_docker_test.go`, `assethandler.go`, `cli.go`, `control_server.go`, `control_server_test.go`, `internal/scanner/scanner.go`, `log_batcher.go`, `log_batcher_test.go`, `mcp.go`
- Modify: `release_contract_test.go`

**Interfaces:**
- Consumes: the existing `defaultConfigPath() string`, `control.AppNotRunningError`, Wails metadata, and current internal package APIs.
- Produces: Go module `rootyard`; `defaultConfigPath()` honoring only `ROOTYARD_CONFIG`; default path `<UserConfigDir>/rootyard/config.json`; Wails name/output `rootyard`; display name `Rootyard`.

- [ ] **Step 1: Change the core identity tests to the Rootyard contract**

Update `release_contract_test.go` so its Wails assertions are exact:

```go
if config.Name != "rootyard" {
    t.Errorf("name = %q, want rootyard", config.Name)
}
if config.OutputFilename != "rootyard" {
    t.Errorf("outputfilename = %q, want rootyard", config.OutputFilename)
}
if config.Info.ProductName != "Rootyard" {
    t.Errorf("productName = %q, want Rootyard", config.Info.ProductName)
}
```

Extend the decoded test struct with `Name string`. In `app_test.go`, replace the old override test and add an explicit legacy-variable isolation test:

```go
func TestDefaultConfigPathHonorsRootyardEnvOverride(t *testing.T) {
    want := filepath.Join(t.TempDir(), "custom.json")
    t.Setenv("ROOTYARD_CONFIG", want)
    if got := defaultConfigPath(); got != want {
        t.Fatalf("defaultConfigPath() = %q, want %q", got, want)
    }
}

func TestDefaultConfigPathIgnoresLegacyEnvOverride(t *testing.T) {
    configHome := t.TempDir()
    t.Setenv("XDG_CONFIG_HOME", configHome)
    t.Setenv("ATSTARTER_CONFIG", filepath.Join(t.TempDir(), "legacy.json"))
    want := filepath.Join(configHome, "rootyard", "config.json")
    if got := defaultConfigPath(); got != want {
        t.Fatalf("defaultConfigPath() = %q, want %q", got, want)
    }
}
```

Change `TestNewAppUsesUserConfigOutsideDev` to expect `rootyard/config.json`. Add `internal/control/protocol_test.go`:

```go
package control

import "testing"

func TestAppNotRunningErrorUsesRootyardName(t *testing.T) {
    if got, want := (AppNotRunningError{}).Error(), "rootyard desktop app is not running"; got != want {
        t.Fatalf("Error() = %q, want %q", got, want)
    }
}
```

- [ ] **Step 2: Run the focused tests and verify they fail against the old identity**

Run:

```bash
export GO=/home/attson/sdk/go1.24.13/bin/go
$GO test . ./internal/control -run 'Test(WailsNaming|DefaultConfigPath|NewAppUsesUserConfig|AppNotRunningError)' -count=1
```

Expected: failures show `atstarter`, `AT Starter`, `ATSTARTER_CONFIG`, or the old config directory where Rootyard values are required.

- [ ] **Step 3: Patch the runtime identity and module imports**

Set `go.mod` to:

```go
module rootyard
```

Replace every import beginning with `atstarter/internal/` in the files listed above with
`rootyard/internal/`. Patch `wails.json` to:

```json
{
  "name": "rootyard",
  "outputfilename": "rootyard",
  "info": {
    "productName": "Rootyard"
  }
}
```

Preserve all unrelated Wails keys. In `app.go`, implement only the new override and directory:

```go
if override := os.Getenv("ROOTYARD_CONFIG"); override != "" {
    return override
}
return filepath.Join(dir, "rootyard", "config.json")
```

Change the window title in `main.go` to `Rootyard`, including the version suffix. Change
`AppNotRunningError.Error()` to `rootyard desktop app is not running`.

- [ ] **Step 4: Format and run the complete Go suite**

Run:

```bash
gofmt -w app.go app_test.go main.go internal/control/protocol.go internal/control/protocol_test.go release_contract_test.go
export GO=/home/attson/sdk/go1.24.13/bin/go
$GO test ./...
```

Expected: all Go packages compile under module `rootyard`, and all tests pass. Do not regenerate Wails bindings because no exported Wails method signature changed.

- [ ] **Step 5: Commit the core identity**

```bash
git add go.mod wails.json app.go app_test.go main.go internal/control/protocol.go internal/control/protocol_test.go release_contract_test.go app_docker_test.go assethandler.go cli.go control_server.go control_server_test.go internal/scanner/scanner.go log_batcher.go log_batcher_test.go mcp.go
git commit -m "refactor: establish Rootyard runtime identity"
```

---

### Task 2: Rename the CLI and MCP Automation Surface

**Files:**
- Modify: `cli.go`
- Modify: `control_server_test.go`
- Modify: `mcp.go`
- Create: `mcp_test.go`

**Interfaces:**
- Consumes: unchanged CLI JSON envelope and local control RPC method names such as `app.ping` and `project.start`.
- Produces: Rootyard CLI help/hints; MCP server name `rootyard`; exact existing tool set under `rootyard_*`; helper `rootyardMCPTools() []mcpTool`.

- [ ] **Step 1: Add CLI and MCP identity tests**

Add this CLI test to `control_server_test.go`:

```go
func TestCLIHelpUsesOnlyRootyardCommand(t *testing.T) {
    var stdout, stderr bytes.Buffer
    if code := runCLIWithIO([]string{"help"}, &stdout, &stderr); code != 0 {
        t.Fatalf("exit = %d stderr=%s", code, stderr.String())
    }
    if !strings.Contains(stdout.String(), `rootyard cli`) {
        t.Fatalf("help = %s, want rootyard cli usage", stdout.String())
    }
    if strings.Contains(strings.ToLower(stdout.String()), "atstarter") {
        t.Fatalf("help contains legacy command: %s", stdout.String())
    }
}
```

Create `mcp_test.go` with exact identity checks:

```go
package main

import (
    "encoding/json"
    "strings"
    "testing"
)

func TestMCPInitializeUsesRootyardServerName(t *testing.T) {
    response := (&mcpServer{}).handle(mcpRequest{
        JSONRPC: "2.0",
        ID:      json.RawMessage("1"),
        Method:  "initialize",
    })
    result := response.Result.(map[string]interface{})
    info := result["serverInfo"].(map[string]interface{})
    if info["name"] != "rootyard" {
        t.Fatalf("server name = %v, want rootyard", info["name"])
    }
}

func TestRootyardMCPToolsUseOnlyRootyardPrefix(t *testing.T) {
    tools := rootyardMCPTools()
    if len(tools) == 0 {
        t.Fatal("tool list is empty")
    }
    for _, tool := range tools {
        if !strings.HasPrefix(tool.Name, "rootyard_") {
            t.Errorf("tool %q does not use rootyard_ prefix", tool.Name)
        }
        if strings.HasPrefix(tool.Name, "atstarter_") {
            t.Errorf("legacy tool registered: %q", tool.Name)
        }
    }
}
```

- [ ] **Step 2: Run the identity tests and verify they fail**

Run:

```bash
export GO=/home/attson/sdk/go1.24.13/bin/go
$GO test . -run 'Test(CLIHelpUsesOnlyRootyardCommand|MCPInitializeUsesRootyardServerName|RootyardMCPToolsUseOnlyRootyardPrefix)' -count=1
```

Expected: CLI output still names the old command, MCP initialize still names the old server, and `rootyardMCPTools` is undefined.

- [ ] **Step 3: Rename CLI copy and every MCP tool without changing schemas**

Patch `cli.go` so all usage and recovery hints use `rootyard cli`, including:

```go
Hint: "run: rootyard cli app start --wait"
```

```go
return control.RemoteError{Code: "usage", Message: message, Hint: "run: rootyard cli help"}
```

```go
"usage": "rootyard cli [--config path] <resource> <command> [args]",
```

In `mcp.go`:

- change server info name to `rootyard`;
- rename `atstarterMCPTools` to `rootyardMCPTools`;
- replace every dispatch name and generated tool name prefix `atstarter_` with `rootyard_`;
- change brand-bearing descriptions to `Rootyard` or `rootyard` as grammatically appropriate;
- leave control RPC names, parameter schemas, result shapes, and error codes unchanged.

- [ ] **Step 4: Run focused and regression tests**

Run:

```bash
gofmt -w cli.go control_server_test.go mcp.go mcp_test.go
export GO=/home/attson/sdk/go1.24.13/bin/go
$GO test . -run 'Test(CLI|RunCLI|MCP)' -count=1
$GO test ./...
```

Expected: all tests pass and `rg -n 'atstarter_|atstarter cli' cli.go mcp.go` prints no matches.

- [ ] **Step 5: Commit the automation rename**

```bash
git add cli.go control_server_test.go mcp.go mcp_test.go
git commit -m "refactor: rename CLI and MCP surface to Rootyard"
```

---

### Task 3: Rename the npm MCP Launcher

**Files:**
- Rename: `npm/atstarter-mcp/` to `npm/rootyard-mcp/`
- Rename: `npm/rootyard-mcp/bin/atstarter-mcp.mjs` to `npm/rootyard-mcp/bin/rootyard-mcp.mjs`
- Modify: `npm/rootyard-mcp/package.json`
- Modify: `npm/rootyard-mcp/lib/locate.mjs`
- Modify: `npm/rootyard-mcp/test/locate.test.mjs`
- Modify: `npm/rootyard-mcp/README.md`

**Interfaces:**
- Consumes: installed Rootyard layouts from the packaging contract.
- Produces: package `@attson/rootyard-mcp`, executable `rootyard-mcp`, exported `locateRootyardBinary()`, and a launcher that executes `[rootyardBinary, "mcp"]`.

- [ ] **Step 1: Move the package and rewrite its tests to the new public API**

Use `git mv` for the package directory and bin file. Change the test import and expectations to:

```js
import { locateRootyardBinary } from '../lib/locate.mjs'

test('ROOTYARD_BIN wins over PATH and known locations', () => {
  const found = locateRootyardBinary({
    platform: 'linux',
    env: { ROOTYARD_BIN: '/opt/custom/rootyard', PATH: '/usr/bin' },
    existsSync: existsSet(['/opt/custom/rootyard', '/usr/bin/rootyard']),
    delimiter: ':',
  })
  assert.equal(found, '/opt/custom/rootyard')
})

test('legacy override is ignored', () => {
  const found = locateRootyardBinary({
    platform: 'linux',
    env: { ATSTARTER_BIN: '/opt/legacy/atstarter', PATH: '/usr/bin' },
    existsSync: existsSet(['/opt/legacy/atstarter', '/usr/bin/rootyard']),
    delimiter: ':',
  })
  assert.equal(found, '/usr/bin/rootyard')
})
```

Change all remaining platform expectations to `/usr/bin/rootyard`,
`/usr/local/bin/rootyard`, `/Applications/Rootyard.app/Contents/MacOS/rootyard`, and
`C:\\Program Files\\Rootyard\\rootyard.exe`.

- [ ] **Step 2: Run the moved launcher tests and verify they fail**

Run:

```bash
node --test npm/rootyard-mcp/test/locate.test.mjs
```

Expected: import failure because `locateRootyardBinary` is not exported, followed by old-path failures after the export is renamed.

- [ ] **Step 3: Implement the Rootyard-only locator and launcher metadata**

Rename the locator export and use only the new variable:

```js
export function locateRootyardBinary({
  platform = process.platform,
  env = process.env,
  existsSync = nodeExistsSync,
  delimiter = nodeDelimiter,
} = {}) {
  const name = platform === 'win32' ? 'rootyard.exe' : 'rootyard'
  const override = env.ROOTYARD_BIN
  // Preserve the existing override -> PATH -> known locations algorithm.
}
```

Known locations must be exactly:

```js
const macOSLocations = [
  '/usr/local/bin/rootyard',
  '/Applications/Rootyard.app/Contents/MacOS/rootyard',
]
const linuxLocations = ['/usr/bin/rootyard', '/usr/local/bin/rootyard']
```

Use `Rootyard` as the Windows install-directory component. Update the bin script to import
`locateRootyardBinary`, mention `ROOTYARD_BIN`, and spawn the located executable with `mcp`.

Set `package.json` identity to:

```json
{
  "name": "@attson/rootyard-mcp",
  "bin": {
    "rootyard-mcp": "./bin/rootyard-mcp.mjs"
  },
  "homepage": "https://github.com/attson/rootyard",
  "repository": {
    "type": "git",
    "url": "https://github.com/attson/rootyard.git",
    "directory": "npm/rootyard-mcp"
  }
}
```

Preserve the current package version until the release workflow derives `0.6.0` from the tag. Update
the package README examples to `npx -y @attson/rootyard-mcp` and Rootyard paths only.

- [ ] **Step 4: Verify tests and package contents**

Run:

```bash
node --test npm/rootyard-mcp/test/locate.test.mjs
cd npm/rootyard-mcp && npm pack --dry-run
```

Expected: all tests pass; dry-run output contains `bin/rootyard-mcp.mjs`, `lib/locate.mjs`, README,
and package metadata, with no old package/bin paths.

- [ ] **Step 5: Commit the npm launcher rename**

```bash
git add -A npm
git commit -m "refactor: rename npm MCP launcher to Rootyard"
```

---

### Task 4: Rename the Agent Plugin and Marketplaces

**Files:**
- Rename: `plugins/atstarter-control/` to `plugins/rootyard-control/`
- Rename: `plugins/rootyard-control/skills/use-atstarter/` to `plugins/rootyard-control/skills/use-rootyard/`
- Modify: `plugins/rootyard-control/.claude-plugin/plugin.json`
- Modify: `plugins/rootyard-control/.codex-plugin/plugin.json`
- Modify: `plugins/rootyard-control/.mcp.json`
- Modify: `plugins/rootyard-control/README.md`
- Modify: `plugins/rootyard-control/skills/use-rootyard/SKILL.md`
- Modify: `.agents/plugins/marketplace.json`
- Modify: `.claude-plugin/marketplace.json`
- Modify: `scripts/validate-ai-plugin.mjs`

**Interfaces:**
- Consumes: `rootyard mcp`, the `rootyard_*` MCP surface, and `@attson/rootyard-mcp` documentation contract.
- Produces: plugin `rootyard-control` with skill `use-rootyard`; Codex and Claude marketplace entries pointing only to the Rootyard plugin.

- [ ] **Step 1: Change the validator to assert the Rootyard plugin contract**

Patch `scripts/validate-ai-plugin.mjs` to use:

```js
const pluginName = 'rootyard-control'
const pluginPath = join(root, 'plugins', pluginName)
assert.equal(codexMarketplace.name, 'rootyard')
assert.equal(codexMarketplace.interface?.displayName, 'Rootyard')
assert.equal(codexEntry.source.path, './plugins/rootyard-control')
assert.equal(claudeMarketplace.name, 'rootyard')
assert.equal(claudeEntry.source, './plugins/rootyard-control')
assert.equal(claudePlugin.skills?.[0], './skills/use-rootyard')
assert.equal(claudePlugin.mcpServers?.rootyard?.command, 'rootyard')
assert.deepEqual(claudePlugin.mcpServers.rootyard.args, ['mcp'])
assertFile(join(pluginPath, 'skills', 'use-rootyard', 'SKILL.md'))
```

Apply the same `rootyard` MCP server assertions to `.mcp.json`.

- [ ] **Step 2: Run the validator and verify it fails on the old plugin paths**

Run:

```bash
node scripts/validate-ai-plugin.mjs
```

Expected: failure reports missing `plugins/rootyard-control` or legacy marketplace values.

- [ ] **Step 3: Move and patch the plugin bundle**

Use `git mv` for the plugin and skill directories. Update both marketplace catalogs and both plugin
manifests to `rootyard-control`. Configure the MCP server as:

```json
{
  "mcpServers": {
    "rootyard": {
      "command": "rootyard",
      "args": ["mcp"]
    }
  }
}
```

Update `use-rootyard/SKILL.md` so every tool call uses `rootyard_*`, the non-default config variable
is `ROOTYARD_CONFIG`, and the product/command is Rootyard/rootyard. Update the plugin README with the
new installation name and MCP package examples.

- [ ] **Step 4: Validate manifests and scan the plugin tree**

Run:

```bash
node scripts/validate-ai-plugin.mjs
rg -n -i 'atstarter|at starter|at-starter' plugins .agents .claude-plugin scripts/validate-ai-plugin.mjs
```

Expected: validator exits zero; the scan prints no matches.

- [ ] **Step 5: Commit the plugin rename**

```bash
git add -A plugins .agents .claude-plugin scripts/validate-ai-plugin.mjs
git commit -m "refactor: rename AI plugin to Rootyard"
```

---

### Task 5: Rename Native Applications, Installers, and Package Payloads

**Files:**
- Modify: `.github/scripts/package-linux-deb.sh`
- Modify: `.github/scripts/package-macos-dmg.sh`
- Modify: `scripts/install-linux.sh`
- Modify: `scripts/install-windows.ps1`
- Modify: `build/darwin/Info.plist`
- Modify: `build/darwin/Info.dev.plist`
- Modify: `build/windows/installer/project.nsi`
- Modify: `build/windows/installer/update-path.ps1`
- Modify: `tray_darwin.go`
- Modify: `tray_darwin.m`
- Modify: `install_darwin.go`
- Modify: `install_darwin_test.go`
- Modify: `install_macos_package.go`
- Modify: `install_macos_package_test.go`
- Modify: `Makefile`
- Modify: `.gitignore`
- Modify: `release_contract_test.go`

**Interfaces:**
- Consumes: Wails output `rootyard` and product name `Rootyard` from Task 1.
- Produces: `/usr/bin/rootyard`, `/Applications/Rootyard.app`, `/usr/local/bin/rootyard`, `rootyard.exe`, Rootyard bundle IDs, and Rootyard-only uninstall ownership.

- [ ] **Step 1: Rewrite platform release-contract tests before changing scripts**

Change `release_contract_test.go` assertions to require these exact payload strings:

```go
linuxWants := []string{
    `bin="build/bin/rootyard"`,
    `install -Dm755 "$bin" "$root/usr/bin/rootyard"`,
    `Exec=rootyard`,
    `Package: rootyard`,
}
macWants := []string{
    `/Applications/Rootyard.app/Contents/MacOS/rootyard`,
    `cp "$component_pkg" "$staging/Rootyard.pkg"`,
    `--identifier "com.attson.rootyard"`,
}
windowsWants := []string{
    `-Action add -Directory "$INSTDIR"`,
    `-Action remove -Directory "$INSTDIR"`,
    `Rootyard`,
}
```

Extend the test to read both plist files and assert `com.attson.rootyard` and
`com.attson.rootyard.dev`. Keep the existing PATH mutation and broadcast assertions. Remove workflow
asset-name assertions from the platform payload tests in this task; Task 6 adds them back under a
dedicated `TestReleaseWorkflowUsesRootyardIdentity`, after the workflow itself is changed. This keeps
the Task 5 commit independently green.

- [ ] **Step 2: Run package-contract tests and verify they fail**

Run:

```bash
export GO=/home/attson/sdk/go1.24.13/bin/go
$GO test . -run 'Test(WailsNaming|LinuxPackages|WindowsInstaller|MacOSDMG|Bundle)' -count=1
```

Expected: failures identify old executable paths, product names, package name, and bundle IDs.

- [ ] **Step 3: Patch Linux and macOS package/install paths**

Make the Linux package script use:

```bash
bin="build/bin/rootyard"
install -Dm755 "$bin" "$root/usr/bin/rootyard"
install -Dm644 build/appicon.png "$root/usr/share/pixmaps/Rootyard.png"
install -Dm644 /dev/stdin "$root/usr/share/applications/Rootyard.desktop" <<DESKTOP
[Desktop Entry]
Type=Application
Name=Rootyard
Exec=rootyard
Icon=Rootyard
Terminal=false
Categories=Development;Utility;
DESKTOP
```

Set the Debian control fields to `Package: rootyard`, `Description: Rootyard desktop app`, and the
local-project-workspace description. The embedded Linux update script must locate a single file named
`rootyard`, install it under that basename, log with `[rootyard-install]`, and never search for the old
binary.

Make the macOS script package `/Applications/Rootyard.app`, executable `rootyard`, symlink
`/usr/local/bin/rootyard`, component `Rootyard.pkg`, volume `Rootyard`, and identifier
`com.attson.rootyard`. Apply production/development bundle IDs in the plist files. Update inline
macOS installer path constants and their tests to Rootyard.

- [ ] **Step 4: Patch Windows packaging and native symbol names**

Change NSIS error/detail copy to Rootyard and rely on Wails `INFO_PRODUCTNAME=Rootyard` plus
`PRODUCT_EXECUTABLE=rootyard.exe`. Keep PATH add/remove and environment broadcast semantics intact.
Change PowerShell comments/logging and self-test fixtures to Rootyard paths without changing
`Update-PathValue` behavior.

Rename the native bridge symbols consistently in `tray_darwin.go` and `tray_darwin.m`:

```c
void rootyard_start_tray(const char *iconBytes, int length);
void rootyard_update_running(int n);
void rootyard_update_visible(bool visible);
```

Update every Go `C.` call to the new symbols. Update `Makefile` clean targets and `.gitignore` build
patterns to Rootyard output names only.

- [ ] **Step 5: Run platform contract and installer unit tests**

Run:

```bash
gofmt -w tray_darwin.go install_darwin.go install_darwin_test.go install_macos_package.go install_macos_package_test.go release_contract_test.go
export GO=/home/attson/sdk/go1.24.13/bin/go
$GO test . -run 'Test(WailsNaming|LinuxPackages|WindowsInstaller|MacOSDMG|Bundle|InstallDarwin|MacOSPackage)' -count=1
powershell.exe -NoProfile -ExecutionPolicy Bypass -File build/windows/installer/update-path.ps1 -SelfTest
```

Expected: Go tests and the PowerShell self-test pass. If `powershell.exe` is unavailable on the
current OS, rely on the Windows CI job for that one command and record it as CI-only verification.

- [ ] **Step 6: Commit the native packaging rename**

```bash
git add .github/scripts/package-linux-deb.sh .github/scripts/package-macos-dmg.sh scripts/install-linux.sh scripts/install-windows.ps1 build/darwin build/windows/installer tray_darwin.go tray_darwin.m install_darwin.go install_darwin_test.go install_macos_package.go install_macos_package_test.go Makefile .gitignore release_contract_test.go
git commit -m "refactor: rename native packages to Rootyard"
```

---

### Task 6: Rename the Updater and Release Pipeline

**Files:**
- Modify: `updater.go`
- Modify: `updater_test.go`
- Modify: `.github/scripts/sign-release-checksums.go`
- Modify: `.github/workflows/build.yml`
- Modify: `release_contract_test.go`

**Interfaces:**
- Consumes: Rootyard package output paths from Task 5 and npm directory from Task 3.
- Produces: update lookup at `attson/rootyard`; cache `<UserCacheDir>/rootyard/updates`; Rootyard assets; Rootyard verification secrets; npm publication from `npm/rootyard-mcp`.

- [ ] **Step 1: Change updater and workflow tests to Rootyard values**

Update the URL fixtures in `updater_test.go`:

```go
raw := "https://github.com/attson/rootyard/releases/download/v0.6.0/Rootyard-linux-amd64.tar.gz"
```

Change the released-name fixtures to `Rootyard-linux-amd64.tar.gz`,
`Rootyard_0.6.0_arm64.dmg`, and `Rootyard_0.6.0_amd64.exe`, while preserving the existing matcher
semantics:

```go
cases := []struct{ os, arch, want string }{
    {"linux", "amd64", "-linux-amd64.tar.gz"},
    {"linux", "arm64", "-linux-arm64.tar.gz"},
    {"darwin", "arm64", "_arm64.dmg"},
    {"darwin", "amd64", "_amd64.dmg"},
    {"windows", "amd64", "_amd64.exe"},
}
```

Add assertions in `release_contract_test.go` that the workflow contains:

```go
workflowWants := []string{
    `APP_ARTIFACT_NAME: 'Rootyard'`,
    `secrets.ROOTYARD_UPDATE_VERIFY_PUBLIC_KEY`,
    `ROOTYARD_UPDATE_SIGNING_PRIVATE_KEY: ${{ secrets.ROOTYARD_UPDATE_SIGNING_PRIVATE_KEY }}`,
    `working-directory: npm/rootyard-mcp`,
    `tar -czf "Rootyard-linux-${{ matrix.arch }}.tar.gz" rootyard`,
    `7z a "Rootyard-windows-amd64.zip" "rootyard.exe"`,
}
```

- [ ] **Step 2: Run updater/release tests and verify they fail**

Run:

```bash
export GO=/home/attson/sdk/go1.24.13/bin/go
$GO test . -run 'Test(Asset|Mirror|FetchText|ReleaseWorkflow)' -count=1
```

Expected: URL/path fixtures and workflow assertions fail against old repository, asset, secret, and
npm-directory values.

- [ ] **Step 3: Patch updater identity without changing verification behavior**

Set:

```go
const updateRepoOwner = "attson"
const updateRepoName = "rootyard"
```

Change the temporary installer prefix to `rootyard-install-*` and update cache directory to:

```go
dir := filepath.Join(cacheRoot, "rootyard", "updates")
```

Update comments and error text that name the product. Do not change mirror ordering, HTTP behavior,
Ed25519 verification, SHA-256 verification, progress reporting, platform install dispatch, or update
state shape.

Change `.github/scripts/sign-release-checksums.go` to read only:

```go
os.Getenv("ROOTYARD_UPDATE_SIGNING_PRIVATE_KEY")
```

and report that exact name when missing.

- [ ] **Step 4: Patch CI build, package, publish, and release paths**

In `.github/workflows/build.yml`:

- set `APP_ARTIFACT_NAME: 'Rootyard'`;
- change all verify-public-key secret references to `ROOTYARD_UPDATE_VERIFY_PUBLIC_KEY`;
- change the signing environment variable and secret to `ROOTYARD_UPDATE_SIGNING_PRIVATE_KEY`;
- run npm tests and publication from `npm/rootyard-mcp`;
- package `rootyard`, `rootyard.exe`, and `Rootyard.app`;
- name/upload portable assets `Rootyard-linux-<arch>.tar.gz`, `Rootyard-darwin-<arch>.zip`, and
  `Rootyard-windows-amd64.zip`;
- verify Rootyard installer payload paths from Task 5;
- preserve platform matrices, tag guards, checksum generation, signing, and release upload behavior.

- [ ] **Step 5: Run updater, release-contract, plugin, and npm checks**

Run:

```bash
gofmt -w updater.go updater_test.go .github/scripts/sign-release-checksums.go release_contract_test.go
export GO=/home/attson/sdk/go1.24.13/bin/go
$GO test ./...
node --test npm/rootyard-mcp/test/locate.test.mjs
node scripts/validate-ai-plugin.mjs
git diff --check
```

Expected: all commands pass. A scan of the workflow and signing script has no old secret or npm path:

```bash
rg -n 'ATSTARTER_UPDATE|npm/atstarter|AT-Starter|atstarter\.exe|\batstarter\b' .github/workflows/build.yml .github/scripts/sign-release-checksums.go
```

Expected: no matches.

- [ ] **Step 6: Commit updater and release pipeline identity**

```bash
git add updater.go updater_test.go .github/scripts/sign-release-checksums.go .github/workflows/build.yml release_contract_test.go
git commit -m "refactor: rename update and release pipeline to Rootyard"
```

---

### Task 7: Rename Desktop and Website Product Surfaces

**Files:**
- Modify: `frontend/index.html`
- Modify: `frontend/src/App.vue`
- Modify: `frontend/src/components/AddProjectDialog.vue`
- Modify: `frontend/src/components/fileExplorer/fsBridge.ts`
- Modify: `frontend/src/components/fileExplorer/theme-bridge.css`
- Modify: `frontend/src/composables/useTheme.js`
- Modify: `frontend/src/projectTree.test.mjs`
- Modify: `frontend/vite.config.js`
- Modify: `site/package.json`
- Modify: `site/package-lock.json`
- Modify: `site/docs/.vitepress/config.mjs`
- Modify: `site/docs/.vitepress/theme/index.js`
- Modify: `site/docs/.vitepress/theme/components/HomeDemo.vue`
- Modify: `site/docs/.vitepress/theme/components/mockWailsApp.mjs`
- Modify: `site/docs/.vitepress/theme/components/homeDemoSource.test.mjs`
- Modify: `site/docs/index.md`
- Modify: `site/docs/guide/index.md`
- Modify: `site/docs/guide/ai-cli.md`
- Modify: `site/docs/guide/faq.md`
- Rename: `site/docs/public/atstarter-icon.svg` to `site/docs/public/rootyard-icon.svg`

**Interfaces:**
- Consumes: Rootyard commands, npm package, URLs, and product positioning from prior tasks.
- Produces: desktop and VitePress user surfaces that identify only as Rootyard; VitePress base `/rootyard/`.

- [ ] **Step 1: Change source-based frontend/site tests to Rootyard expectations**

Update affected source tests so expected product references use Rootyard. Add explicit config assertions
to `homeDemoSource.test.mjs`:

```js
test('site uses the Rootyard Pages base and repository links', () => {
  assert.match(configSource, /base:\s*['"]\/rootyard\/['"]/)
  assert.match(configSource, /github\.com\/attson\/rootyard/)
  assert.doesNotMatch(configSource.toLowerCase(), /atstarter|at-starter/)
})
```

Preserve tests that protect demo bindings, source parity, and mock coverage.

- [ ] **Step 2: Run affected tests and verify they fail**

Run:

```bash
node --test frontend/src/projectTree.test.mjs frontend/src/composables/useTheme.test.mjs
node --test site/docs/.vitepress/theme/components/homeDemoSource.test.mjs
```

Expected: source assertions find old titles, storage/theme keys, links, or Pages base.

- [ ] **Step 3: Patch frontend-visible identity and internal product-specific keys**

Change document/app titles and user-facing copy to Rootyard. Rename product-specific CSS classes,
storage keys, bridge identifiers, and test fixtures that contain the old slug to `rootyard`, preserving
their behavior. Do not change Wails exported method names or event names such as `logs:batch` and
`update:state`.

- [ ] **Step 4: Patch VitePress identity, content, and public asset**

Use `git mv` for the icon. Set site metadata to:

```js
export default defineConfig({
  base: '/rootyard/',
  title: 'Rootyard',
  description: 'Your local project workspace',
})
```

Point favicon, repository, release, and Pages links to Rootyard. Set the site package name to
`rootyard-site` and refresh only the corresponding root-package metadata in `site/package-lock.json`
using `npm install --package-lock-only` from `site/`. Update the home/guide copy to the approved
workspace positioning while retaining accurate descriptions of current features. Update CLI/AI guide
examples to `rootyard`, `ROOTYARD_CONFIG`, `ROOTYARD_BIN`, `rootyard_*`, and
`@attson/rootyard-mcp`.

- [ ] **Step 5: Run complete frontend and site verification**

Run:

```bash
cd frontend
npm run build
node --test $(find src -name '*.test.mjs' | sort)
cd ../site
npm run docs:build
node --test $(find docs/.vitepress/theme/components -name '*.test.mjs' | sort)
```

Expected: both builds and all Node tests pass. Then run:

```bash
cd ..
rg -n -i 'atstarter|at starter|at-starter' frontend site --glob '!**/node_modules/**' --glob '!**/.vitepress/dist/**'
```

Expected: no matches.

- [ ] **Step 6: Commit product-surface changes**

```bash
git add -A frontend site
git commit -m "docs: rename product surfaces to Rootyard"
```

---

### Task 8: Rename Current Guidance and Historical Working Documents

**Files:**
- Modify: `README.md`
- Modify: `CLAUDE.md`
- Rename: `docs/ai/atstarter-cli.md` to `docs/ai/rootyard-cli.md`
- Modify: `docs/specs/ARCHITECTURE.md`
- Modify: `docs/specs/DOMAIN_MODEL.md`
- Modify: `docs/specs/RUNTIME_CONTRACTS.md`
- Modify: `docs/specs/SITE.md`
- Modify residual brand-bearing comments/fixtures in: `app_docker_test.go`, `assethandler.go`, `internal/docker/parse.go`, `internal/docker/parse_test.go`, `internal/filetree/bytesio.go`, `internal/git/git.go`, `internal/runner/process_kill_unix_test.go`, `internal/runner/runner_test.go`, `internal/scanner/scanner.go`, `internal/store/model.go`, `log_batcher.go`, `log_batcher_test.go`, `tray.go`
- Rename: `docs/superpowers/plans/2026-07-16-atstarter.md` to `docs/superpowers/plans/2026-07-16-rootyard.md`
- Rename: `docs/superpowers/specs/2026-07-16-atstarter-design.md` to `docs/superpowers/specs/2026-07-16-rootyard-design.md`
- Rename: `docs/superpowers/specs/2026-07-16-atstarter-implementation-notes.md` to `docs/superpowers/specs/2026-07-16-rootyard-implementation-notes.md`
- Modify other tracked source files reported by the final audit, excluding only the approved rename design and this implementation plan.

**Interfaces:**
- Consumes: the final command, package, plugin, asset, repository, and site contracts established in Tasks 1-7.
- Produces: current guidance that consistently describes Rootyard and preserves no operational dependency on a legacy identity.

- [ ] **Step 1: Rename product-specific documentation paths with Git history preserved**

Use `git mv` for the four files listed above. Do not rename the approved Rootyard design or this plan.
Replace current-project titles, module paths, commands, URLs, package names, config paths, MCP names,
plugin paths, release assets, and architecture labels inside the moved files.

- [ ] **Step 2: Update long-lived product and agent guidance**

In `README.md`, use this identity block:

```markdown
# Rootyard

Your local project workspace.

Rootyard 是面向 AI 开发时代的本地项目工作台。
```

Describe only features that already exist. Use `rootyard`, `@attson/rootyard-mcp`, and Rootyard URLs
in installation and AI integration examples.

In `CLAUDE.md`, change the project overview and Git/release naming rules to Rootyard while preserving
all architectural, runtime, frontend, safety, and required-command rules. In long-lived specs, update
product/module/CLI/MCP/site naming without changing domain terms such as `Project`, `LaunchCommand`,
`LaunchGroup`, `Workspace`, `RunID`, `ComposeService`, `ContainerState`, `FileEntry`, `EditorTab`, or
`Branch`.

- [ ] **Step 3: Run a full content and filename audit**

Run:

```bash
rg -uuu -n -i 'atstarter|at starter|at-starter|@attson/atstarter' \
  -g '!frontend/node_modules/**' \
  -g '!site/node_modules/**' \
  -g '!site/docs/.vitepress/dist/**' \
  -g '!build/bin/**' \
  -g '!.git/**' \
  | rg -v 'docs/superpowers/specs/2026-11-25-rootyard-rename-design.md|docs/superpowers/plans/2026-11-25-rootyard-rename.md'
```

Expected: no output. If matches remain in current code, comments, tests, fixtures, file names, or docs,
patch each file listed in this task to the corresponding Rootyard identity and rerun the command.

Run the filename audit:

```bash
rg --files -uuu \
  -g '!frontend/node_modules/**' \
  -g '!site/node_modules/**' \
  -g '!site/docs/.vitepress/dist/**' \
  -g '!build/bin/**' \
  -g '!.git/**' \
  | rg -i 'atstarter|at-starter|at starter'
```

Expected: no output.

- [ ] **Step 4: Verify documentation references resolve and the tree is cleanly formatted**

Run:

```bash
rg -n 'docs/ai/atstarter-cli|npm/atstarter-mcp|plugins/atstarter-control|use-atstarter|/atstarter/' . \
  -g '!frontend/node_modules/**' -g '!site/node_modules/**' -g '!.git/**' \
  -g '!docs/superpowers/specs/2026-11-25-rootyard-rename-design.md' \
  -g '!docs/superpowers/plans/2026-11-25-rootyard-rename.md'
git diff --check
```

Expected: no stale references and no whitespace errors.

- [ ] **Step 5: Commit documentation and final legacy-reference cleanup**

```bash
git add README.md CLAUDE.md docs/ai/rootyard-cli.md docs/specs \
  app_docker_test.go assethandler.go internal/docker/parse.go internal/docker/parse_test.go \
  internal/filetree/bytesio.go internal/git/git.go internal/runner/process_kill_unix_test.go \
  internal/runner/runner_test.go internal/scanner/scanner.go internal/store/model.go \
  log_batcher.go log_batcher_test.go tray.go
git add -u docs
git commit -m "docs: complete Rootyard identity migration"
```

---

### Task 9: Run Full Integration Verification

**Files:**
- Verify only: all changed files from Tasks 1-8
- Do not create release tags or modify external services.

**Interfaces:**
- Consumes: the complete Rootyard tree.
- Produces: evidence that the branch satisfies the design before review/merge.

- [ ] **Step 1: Verify there are no unintended legacy identifiers**

Run both audit commands from Task 8 again. Expected output is empty except the two explicitly excluded
process documents, whose purpose is to record the old-to-new mapping.

- [ ] **Step 2: Run all Go and race tests**

Run:

```bash
export GO=/home/attson/sdk/go1.24.13/bin/go
$GO test ./...
$GO test -race ./internal/runner/
```

Expected: all tests pass with no package importing `atstarter/internal/...`.

- [ ] **Step 3: Run frontend and site builds/tests**

Run:

```bash
cd frontend
npm run build
node --test $(find src -name '*.test.mjs' | sort)
cd ../site
npm run docs:build
node --test $(find docs/.vitepress/theme/components -name '*.test.mjs' | sort)
cd ..
```

Expected: both builds and all Node tests pass.

- [ ] **Step 4: Run AI integration and package checks**

Run:

```bash
node scripts/validate-ai-plugin.mjs
node --test npm/rootyard-mcp/test/locate.test.mjs
cd npm/rootyard-mcp && npm pack --dry-run && cd ../..
```

Expected: plugin validation, launcher tests, and package dry-run succeed with Rootyard-only names.

- [ ] **Step 5: Check release contracts, diff hygiene, and commit history**

Run:

```bash
export GO=/home/attson/sdk/go1.24.13/bin/go
$GO test . -run 'Test(WailsNaming|LinuxPackages|WindowsInstaller|MacOSDMG|Bundle|Asset|Mirror|Release)' -count=1
git diff --check main...HEAD
git status --short
git log --oneline main..HEAD
```

Expected: release-contract tests pass; diff check is clean; status is clean; history contains the
small, boundary-oriented commits from this plan and no release/tag commit.

- [ ] **Step 6: Record CI-only and external release checks in the handoff**

The implementation handoff must explicitly list these post-merge operations without performing them:

```text
1. Rename attson/atstarter to attson/rootyard on GitHub.
2. Update the local origin URL to git@github.com:attson/rootyard.git.
3. Configure ROOTYARD_UPDATE_VERIFY_PUBLIC_KEY and ROOTYARD_UPDATE_SIGNING_PRIVATE_KEY.
4. Confirm GitHub Pages serves /rootyard/.
5. Publish @attson/rootyard-mcp.
6. Tag v0.6.0 from merged main and verify all Linux/macOS/Windows assets.
```

Do not mark the external operations complete until they are actually performed after merge.
