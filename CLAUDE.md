# CLAUDE.md

This file provides guidance to AI coding agents working in this repository.

> Always read `docs/specs/*.md` for the subsystem you touch. `AGENTS.md` is a thin cross-tool entry point that points here.

## Project Overview

atstarter is a local project launcher desktop app built with Wails v2, Go, and Vue 3. It scans local workspaces, detects project types, stores project launch commands, starts/stops projects, shows logs, manages Docker/compose, provides a file browser/editor, and self-updates from GitHub Releases.

## Required Commands

The system `go` is too old locally. Use the explicit Go binary for local Go commands:

```bash
export GO=/home/attson/sdk/go1.24.13/bin/go
$GO test ./...
$GO test -race ./internal/runner/
cd frontend && npm run build
node --test frontend/src/projectTree.test.mjs frontend/src/commandForms.test.mjs frontend/src/composables/useTheme.test.mjs frontend/src/dockerState.test.mjs frontend/src/envVars.test.mjs frontend/src/projectDetection.test.mjs frontend/src/updateSchedule.test.mjs frontend/src/workspaceRoots.test.mjs
```

Wails build/dev on Ubuntu 24.04 must use WebKitGTK 4.1:

```bash
make dev
make build
wails build -tags webkit2_41
```

## Architecture

```text
main.go                 Wails entry, embedded frontend, Version/UpdateVerifyPublicKey ldflags
app.go                  Wails binding layer, module assembly, event bridge, 64 App methods
tray.go                 system tray, close-to-tray, running count, quit flow
updater.go              GitHub release self-update, mirrors, Ed25519 + SHA256 verification, 5 update methods
internal/
  cmdparse/             shell-like command line <-> command+args
  detector/             project type detection and suggested commands
  scanner/              workspace direct-child scan including worktrees
  store/                JSON config, path IDs, commands, groups
  runner/               process lifecycle, logs, login shell, process tree cleanup
  docker/               docker/compose CLI facade and parsers
  filetree/             project-scoped file browser, metadata, write, trash, watch
frontend/src/           Vue3 app, custom UI components, theme tokens, file explorer
```

## Spec Index

| File | Content |
|------|---------|
| `docs/specs/ARCHITECTURE.md` | system overview and subsystem map |
| `docs/specs/DOMAIN_MODEL.md` | domain terms, aggregates, IDs, states |
| `docs/specs/RUNTIME_CONTRACTS.md` | detector/store/runner/docker/updater/filetree contracts |
| `docs/specs/FRONTEND_STYLE.md` | UI structure, design tokens, interaction rules |
| `docs/specs/FILE_BROWSER.md` | file explorer security, editing, preview, watch behavior |
| `docs/specs/SITE.md` | VitePress site, embedded app demo, mock data, screenshots, theme bridge |

## Coding Norms

- Prefer existing module boundaries and helper APIs. Do not move behavior across backend/frontend layers without a clear contract reason.
- Business logic changes in `internal/{runner,store,detector,cmdparse,scanner,docker,filetree}` need focused tests.
- Frontend pure transformations belong in small `.js` modules with `node:test` coverage, not buried only in Vue templates.
- Regenerate Wails bindings after changing exported `App` method signatures: `$($GO env GOPATH)/bin/wails generate module`.
- Commit messages must not include `Co-Authored-By`.

## Must-Follow Rules

### Domain Terms

- Use these terms consistently: `Project`, `LaunchCommand`, `LaunchGroup`, `Workspace`, `RunID`, `ComposeService`, `ContainerState`, `FileEntry`.
- Do not call a `LaunchCommand` a "script" in persisted models; "command" means structured `command + args + cwd + env`.

### Runtime Boundaries

- `store.IDForPath(path)` is the project de-duplication key. Preserve path normalization before storing.
- Default command ID is exactly `default`. Non-default command IDs must not collide with `default`.
- `runner` status reads/writes stay under `r.mu`; callbacks run outside the lock with copied values; slow process-tree cleanup stays outside locks.
- Unix process launch uses login interactive shell and process group cleanup. Preserve `$SHELL -l -i -c` and `setsid` unless replacing the full contract.
- File browser APIs must resolve all relative paths inside project root and reject path traversal.

### Frontend Rules

- No third-party UI component library. Use `components/ui/*`, CSS variables, scoped CSS, and `lucide-vue-next`.
- Theme colors, spacing, radii, shadows, and typography come from `styles/tokens.css` plus `theme.light.css`/`theme.dark.css`.
- Project detail path is a full-width single-line row with `title` for hover full path. Command edit belongs on the command row.
- EditProjectDialog command forms round-trip `env` as one `KEY=value` per line; empty command `cwd` displays `project.path` as editable value.

### Git And Release

- Do not push directly to `main`. Use feature branch -> GitHub PR -> green CI -> merge -> tag from `main`.
- Release by annotated `v*` tag on `main`; CI builds Linux/macOS/Windows assets, signs checksums, and publishes GitHub Release.
