# Rootyard Rename Design

**Date:** 2026-11-25

**Status:** Ready for written-spec review

**Release:** v0.6.0

## 1. Summary

Rename the current AT Starter product to **Rootyard** and reposition it as a local project
workspace for AI-assisted development. This change establishes one consistent product and technical
identity across the desktop app, command line, MCP surface, packages, plugins, installers, updater,
documentation, website, and repository metadata.

This is a clean identity cutover. Rootyard does not provide compatibility aliases for AT Starter,
does not read or migrate AT Starter configuration, and does not remove AT Starter files. Git history
and the existing semantic version sequence are preserved; the first Rootyard release is `v0.6.0`.

The rename does not implement the proposed hierarchical project-environment system. That system
will be designed separately after the rename is complete.

## 2. Product Positioning

Rootyard is a **local project workspace**. It brings local projects, launch commands, processes,
logs, Docker, files, Git state, and AI-facing controls into one desktop application. The future
project-environment subsystem will extend that workspace with directory-scoped AI and development
configuration, but it is not part of this change.

The canonical positioning is:

| Item | Value |
|------|-------|
| Product name | `Rootyard` |
| Chinese category | `本地项目工作台` |
| English tagline | `Your local project workspace` |

User-facing copy should describe Rootyard as a workspace rather than only as a launcher. It may
still describe project launching as an existing capability.

## 3. Goals

- Establish `Rootyard` as the only current product name in the checked-in product surface.
- Establish `rootyard` as the only installed executable and CLI command.
- Rename all machine-visible identifiers that form part of configuration, integration, packaging,
  or release contracts.
- Keep existing project launch, process, Docker, log, file, Git, control-server, and update behavior
  functionally unchanged after the identity cutover.
- Preserve Git history and continue the current version sequence with `v0.6.0`.
- Make the rename complete enough that later project-environment work starts from Rootyard naming
  and does not need to carry dual-brand compatibility.

## 4. Non-goals

- Implementing global, parent-directory, project, local, or session configuration layers.
- Adding `resolve`, `diff`, `sync`, or `doctor` commands for project environments.
- Changing the persisted config schema or existing domain model.
- Redesigning the desktop UI, logo, or interaction model beyond required names and product copy.
- Migrating, importing, deleting, or linking AT Starter configuration and cache files.
- Providing an `atstarter` command alias, old MCP tool aliases, old environment-variable fallbacks,
  redirects inside the binary, or a compatibility npm package.
- Renaming a developer's local checkout directory as part of a Git commit.
- Publishing packages, renaming the GitHub repository, changing secrets, or creating a release from
  the implementation branch.

## 5. Canonical Naming Contract

### 5.1 Product and code identity

| Surface | Old identity | Rootyard identity |
|---------|--------------|-------------------|
| Product display name | `AT Starter` | `Rootyard` |
| Product slug | `atstarter` / `at-starter` | `rootyard` |
| Executable and CLI | `atstarter` | `rootyard` |
| Windows executable | `atstarter.exe` | `rootyard.exe` |
| Go module | `atstarter` | `rootyard` |
| Go internal imports | `atstarter/internal/...` | `rootyard/internal/...` |
| macOS bundle ID | `com.attson.atstarter` | `com.attson.rootyard` |
| macOS development bundle ID | `com.attson.atstarter.dev` | `com.attson.rootyard.dev` |
| Native helper symbol prefix | `atstarter_` | `rootyard_` |
| Linux package name | `at-starter` | `rootyard` |
| Linux desktop/icon name | `AT-Starter` | `Rootyard` |
| Windows install/product name | `AT Starter` | `Rootyard` |

`wails.json` uses `rootyard` for `name` and `outputfilename`, and `Rootyard` for `productName`.
Window titles, tray labels, shortcuts, installer UI, package descriptions, and application paths use
`Rootyard` consistently.

### 5.2 Runtime files and environment variables

| Purpose | Old identity | Rootyard identity |
|---------|--------------|-------------------|
| Default config | `<UserConfigDir>/atstarter/config.json` | `<UserConfigDir>/rootyard/config.json` |
| Config override | `ATSTARTER_CONFIG` | `ROOTYARD_CONFIG` |
| Runtime control state | `<config>.control.json` | `<Rootyard config>.control.json` |
| Update cache | `<UserCacheDir>/atstarter/updates` | `<UserCacheDir>/rootyard/updates` |
| Temporary installer prefix | `atstarter-install-*` | `rootyard-install-*` |
| npm binary override | `ATSTARTER_BIN` | `ROOTYARD_BIN` |
| Update verification secret | `ATSTARTER_UPDATE_VERIFY_PUBLIC_KEY` | `ROOTYARD_UPDATE_VERIFY_PUBLIC_KEY` |
| Update signing secret | `ATSTARTER_UPDATE_SIGNING_PRIVATE_KEY` | `ROOTYARD_UPDATE_SIGNING_PRIVATE_KEY` |

Test-only identifiers that contain the product prefix also move to `ROOTYARD_*` so current source and
diagnostics use one identity. The generic workflow-local variable `UPDATE_VERIFY_PUBLIC_KEY` may
remain as an implementation detail, but its GitHub secret source is the Rootyard secret.

### 5.3 AI integration identity

| Surface | Rootyard identity |
|---------|-------------------|
| MCP server name | `rootyard` |
| MCP tool prefix | `rootyard_*` |
| npm package | `@attson/rootyard-mcp` |
| npm executable | `rootyard-mcp` |
| npm source directory | `npm/rootyard-mcp/` |
| Agent/Claude plugin | `rootyard-control` |
| Plugin source directory | `plugins/rootyard-control/` |
| Plugin skill | `use-rootyard` |
| Marketplace name | `rootyard` |

Every existing MCP tool keeps its operation and schema and changes only its prefix. For example,
`atstarter_project_start` becomes `rootyard_project_start`. The implementation must not register both
names.

The npm launcher locates `rootyard` or `rootyard.exe`, honors only `ROOTYARD_BIN`, and executes
`rootyard mcp`. It does not search for the old executable or inspect the old override variable.

The plugin manifests, MCP command configuration, skill instructions, validation script, marketplace
catalogs, and onboarding examples all use the Rootyard package, command, plugin, skill, and tool
names.

### 5.4 Repository and public distribution identity

| Surface | Rootyard identity |
|---------|-------------------|
| GitHub repository | `attson/rootyard` |
| Repository URL | `https://github.com/attson/rootyard` |
| Git remote after external rename | `git@github.com:attson/rootyard.git` |
| GitHub Pages | `https://attson.github.io/rootyard/` |
| VitePress base | `/rootyard/` |
| Public icon filename | `rootyard-icon.svg` |
| Release asset prefix | `Rootyard-` or `Rootyard_` as defined below |

The checked-in code and documentation target the final Rootyard URLs even before the external
repository rename occurs. GitHub's repository redirect may keep old hyperlinks reachable, but that
redirect is not an application compatibility feature.

## 6. Source-tree Rename Scope

The implementation updates all current checked-in product references, including:

- Wails metadata, Go module/imports, application title, tray integration, control protocol errors,
  CLI help, MCP server metadata, MCP tool names, updater repository/cache paths, and tests.
- npm launcher directory, files, package metadata, binary lookup logic, tests, and documentation.
- plugin directory, manifests, skill directory, marketplace catalogs, validation, and documentation.
- macOS bundle metadata and packaging, Linux package/desktop integration, Windows installer and PATH
  integration, install scripts, release-contract tests, and CI workflow paths.
- frontend and site titles, URLs, public asset names, demo content, user guides, README, AI/CLI docs,
  long-lived specs, and agent guidance.
- checked-in historical design and plan filenames/content where they identify the current project.
  Git history remains the source of the original AT Starter-era documents.

Filesystem names containing the old slug are renamed, not merely left with updated content. The local
checkout directory itself is outside the repository and may be renamed manually after the branch is
merged.

The application icon artwork may remain visually unchanged for this release. Its product-specific
filename and references change to Rootyard where applicable.

## 7. Behavioral Invariants

The rename must not alter these contracts:

- Project, launch-command, launch-group, workspace, and settings data shapes remain unchanged.
- Path normalization and `store.IDForPath` behavior remain unchanged.
- Process start/stop/restart, shell invocation, logs, status events, and cleanup remain unchanged.
- Docker and compose lifecycle behavior remain unchanged.
- File browser and editor security boundaries remain unchanged.
- Local Git operations remain unchanged and do not add network operations.
- The desktop process remains the source of truth for the CLI and MCP control surface.
- CLI JSON envelope shape and error codes remain unchanged; only brand-bearing messages and hints
  change.
- MCP tool argument/result schemas remain unchanged; only tool names and brand-bearing descriptions
  change.
- Release verification remains Ed25519 plus SHA-256, using newly named GitHub secrets.
- Update checks continue to use GitHub Releases, now under `attson/rootyard`.

The persisted configuration version stays at its current value because the data schema does not
change.

## 8. Configuration and Data Cutover

Rootyard starts with its own default config path. On first launch without `ROOTYARD_CONFIG`, it creates
or reads `<UserConfigDir>/rootyard/config.json` and its adjacent runtime control file. It does not probe
`<UserConfigDir>/atstarter/config.json`.

The following rules are explicit:

1. `ATSTARTER_CONFIG` is ignored by Rootyard.
2. Existing AT Starter config, control-state, cache, application, executable, shortcuts, and package
   files are not copied, renamed, imported, or deleted.
3. `ROOTYARD_CONFIG` can point to a user-selected file, including an old file if the user explicitly
   chooses it. This is ordinary path override behavior, not automatic migration.
4. Rootyard writes runtime discovery next to the config file it actually uses, so Rootyard CLI and
   MCP clients cannot accidentally attach to an AT Starter desktop process.
5. Uninstallers only remove paths owned by their own product identity.

This policy is acceptable because the product currently has no meaningful installed user base and
avoids carrying permanent dual-brand logic into the new product.

## 9. CLI, MCP, npm, and Plugin Cutover

The installed command becomes:

```text
rootyard
rootyard cli ...
rootyard mcp
```

CLI examples, error hints, process-launch lookup, package installers, and PATH verification must all
use that command. No `atstarter` shim is installed.

The MCP server exposes the same set of capabilities under `rootyard_*`. Existing client
configurations that name old tools or execute the old npm package require manual replacement. The
canonical portable MCP setup is:

```text
npx -y @attson/rootyard-mcp
```

The Rootyard control plugin embeds that launcher configuration and a `use-rootyard` skill describing
the renamed tools. Old package and plugin names are not published as forwarding wrappers.

## 10. Packaging, Updates, and Release Assets

Every formal installer exposes `rootyard` in a newly opened terminal:

- Linux installs `/usr/bin/rootyard`, a `Rootyard.desktop` entry, and Rootyard icons.
- macOS installs `/Applications/Rootyard.app` with executable `rootyard` and symlink
  `/usr/local/bin/rootyard`.
- Windows installs `rootyard.exe`, adds the Rootyard install directory to PATH, and creates Rootyard
  shortcuts.

Portable archives contain only the canonical Rootyard executable and do not modify PATH.

Release assets retain the platform coverage of the current pipeline and use Rootyard prefixes:

| Platform | Asset contract |
|----------|----------------|
| Linux portable | `Rootyard-linux-<arch>.tar.gz` |
| Linux package | `Rootyard_<version>_<arch>.deb` |
| macOS portable | `Rootyard-darwin-<arch>.zip` |
| macOS installer | `Rootyard_<version>_<arch>.dmg` |
| Windows portable | `Rootyard-windows-amd64.zip` |
| Windows installer | `Rootyard_<version>_amd64.exe` |
| Verification | `SHA256SUMS` and `SHA256SUMS.sig` |

The Rootyard updater changes its repository and expected asset names together. Its download,
signature verification, progress, install, and relaunch semantics otherwise remain unchanged.

AT Starter `v0.5.x` to Rootyard `v0.6.0` is not a supported in-place update contract. Because the
repository is renamed in place, GitHub may redirect the old update endpoint and an old build may see
the new release. Rootyard does not add old executable names or legacy archive members to make that
path succeed. The supported cutover is a separate Rootyard installation; old installations remain
untouched and can be removed manually.

## 11. Documentation and Long-lived Project Rules

The implementation updates `CLAUDE.md`, `README.md`, and relevant `docs/specs/*.md` so future work
uses Rootyard vocabulary and the new release contract. The architecture and domain boundaries remain
the same for this release; only the product positioning and identities change.

Current documentation must not instruct users or agents to run old commands, install old packages,
use old URLs, or set old environment variables. The new rename design itself may mention old values
when documenting the cutover, and Git history naturally retains prior names.

## 12. External Operations Boundary

The implementation branch may change only repository content and local Git history. It must not:

- rename the GitHub repository;
- push directly to `main`;
- create or modify repository/environment secrets;
- publish the npm package;
- publish a GitHub release or tag;
- delete old application data or packages from any machine.

After the implementation is reviewed, merged, and verified, external operations occur in this order:

1. Rename `attson/atstarter` to `attson/rootyard` on GitHub.
2. Update the local Git remote to `git@github.com:attson/rootyard.git`.
3. Configure `ROOTYARD_UPDATE_VERIFY_PUBLIC_KEY` and
   `ROOTYARD_UPDATE_SIGNING_PRIVATE_KEY` in the required GitHub environments.
4. Confirm GitHub Pages serves the repository at `/rootyard/`.
5. Publish `@attson/rootyard-mcp`.
6. Tag `v0.6.0` from the merged `main` branch and let CI publish the Rootyard release.

The external sequence is part of the release checklist, not something automated by the rename
implementation.

## 13. Verification Strategy

### 13.1 Static identity audit

Run a case-insensitive repository scan over tracked and hidden source files while excluding generated
dependencies, build output, and `.git`. Every remaining old-brand occurrence must be one of:

- an intentional old-to-new mapping in this design document; or
- an explicitly justified release note describing the cutover.

There must be no operational code, manifest, current documentation, package path, plugin path, site
asset, or test fixture that still depends on the old identity.

Also scan filenames for old slugs and verify all current product-specific paths have been renamed.

### 13.2 Automated tests

Run the repository's required checks:

```bash
export GO=/home/attson/sdk/go1.24.13/bin/go
$GO test ./...
$GO test -race ./internal/runner/
cd frontend && npm run build
node --test frontend/src/projectTree.test.mjs frontend/src/commandForms.test.mjs frontend/src/composables/useTheme.test.mjs frontend/src/dockerState.test.mjs frontend/src/envVars.test.mjs frontend/src/projectDetection.test.mjs frontend/src/updateSchedule.test.mjs frontend/src/workspaceRoots.test.mjs
```

In addition:

- run all npm launcher tests from `npm/rootyard-mcp/test`;
- run the AI plugin manifest validator;
- build the VitePress site;
- run or extend release-contract tests for Wails names, package payloads, executable paths, bundle
  identifiers, asset names, updater repository, and signing-secret names;
- regenerate Wails bindings only if exported method signatures change. A pure rename should not
  change those signatures.

### 13.3 Manual smoke checks

- Launch the desktop app and confirm Rootyard names in the window, tray, settings/about surfaces,
  and error messages.
- Confirm a fresh launch uses the Rootyard config path and does not read an existing AT Starter
  config.
- Confirm `ROOTYARD_CONFIG` selects a custom config and `ATSTARTER_CONFIG` has no effect.
- Confirm `rootyard cli app ping` reaches the running desktop app and returns the unchanged JSON
  envelope.
- Confirm `rootyard mcp` advertises only `rootyard_*` tools.
- Confirm the npm launcher locates the Rootyard binary through `ROOTYARD_BIN`, PATH, and known
  platform locations.
- Inspect Linux, macOS, and Windows release artifacts in CI to confirm canonical executable names,
  application paths, PATH integration, and Rootyard asset names.

## 14. Acceptance Criteria

The rename is complete when all of the following are true:

1. The desktop app, installed command, CLI/MCP surface, npm package, plugin, website, installers,
   updater, and current docs consistently identify as Rootyard.
2. `rootyard`, `ROOTYARD_*`, `rootyard_*`, `@attson/rootyard-mcp`, `rootyard-control`,
   `com.attson.rootyard`, and `attson/rootyard` are the canonical machine identities.
3. Rootyard creates and uses its own config and cache paths without automatically touching old data.
4. No compatibility alias or automatic migration exists for old commands, tools, variables,
   packages, plugins, or config paths.
5. Existing non-brand behavior and persisted data schemas remain unchanged.
6. All required automated checks and release-contract tests pass.
7. The repository is ready for the external rename, secrets, npm publication, Pages deployment, and
   `v0.6.0` release steps, but none of those external mutations has been performed from the feature
   branch.

## 15. Follow-up Work

After the Rootyard rename ships, design the project-environment subsystem as a separate architectural
change. Its intended direction remains:

- configuration precedence from global to directory/project/local/session layers;
- a standalone configuration engine and CLI usable without the desktop app;
- `resolve` for read-only effective configuration, `diff` for proposed native-file changes, `sync`
  for explicit writes, and `doctor` for diagnostics;
- native synchronization for Git identity and supported AI-tool rules, skills, plugins, and MCP
  configuration;
- Rootyard desktop UI as the workspace-oriented management surface over that engine.

These points preserve product direction but create no implementation requirement for the rename.
