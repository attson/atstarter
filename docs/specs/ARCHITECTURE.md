# atstarter Architecture Spec

## Document Goal

This file is the high-level architecture map for atstarter. Keep detailed domain rules in
`DOMAIN_MODEL.md`, runtime contracts in `RUNTIME_CONTRACTS.md`, frontend interaction/style rules in
`FRONTEND_STYLE.md`, and file browser rules in `FILE_BROWSER.md`.

Process drafts under `docs/superpowers/` are not formal specs. Long-lived project rules belong in
`CLAUDE.md` and `docs/specs/`.

## 1. System Overview

atstarter is a local desktop project launcher. It turns local directories into managed launch targets:
detect project type, store launch commands, start or stop child processes, show logs, manage
Docker/compose, browse project files, and self-update from GitHub Releases.

```text
Wails desktop process
  frontend: Vue 3 + Vite + custom CSS/theme tokens
    App.vue
    ProjectList / ProjectDetail / GroupDetail / ComposeDetail / ContainerPanel
    FileBrowser and fileExplorer preview/editor components
      calls Wails bindings and subscribes to runtime events

  backend: Go
    main.go       Wails entry, embedded frontend, ldflags
    app.go        module assembly, Wails bindings, event bridge, 63 App methods
    updater.go    GitHub release update flow, 5 update methods
    tray.go       system tray and close-to-tray behavior
    internal/     pure and stateful subsystem packages
```

The app is intentionally local-first. Configuration is a JSON file under the platform config
directory. There is no server-side state.

## 2. Backend Modules

| Module | Responsibility | Boundary |
|--------|----------------|----------|
| `internal/cmdparse` | Convert one command line into `command + args` using shlex rules | Pure transformation |
| `internal/detector` | Detect project type and suggested command from files | Read-only filesystem |
| `internal/scanner` | Scan workspace direct children and worktree directories | Read-only filesystem |
| `internal/store` | Persist config, normalize paths, de-duplicate projects, commands, groups | File-backed state |
| `internal/runner` | Start/stop processes, capture logs, maintain status, clean process trees | Concurrent runtime |
| `internal/docker` | Wrap docker/compose CLI, parse snapshots, aggregate service state | CLI facade, injectable exec |
| `internal/filetree` | Project-scoped file listing, preview, edit, trash, watch | Filesystem state within root |
| `app.go` | Compose modules and expose Wails methods/events | Binding layer only |
| `updater.go` | Check/download/verify/install releases | Network and installer state |
| `tray.go` | Tray menu, close-to-tray, running count, exit gate | Desktop integration |

## 3. Data Flow

```text
scanner.Scan(root)
  -> detector.Detect(dir)
  -> cmdparse.Parse(suggestedLine)
  -> store.Project{commands: []LaunchCommand}
  -> app.StartProjectCommand(projectID, commandID)
  -> runner.Spec{ID, Command, Args, Dir, Env}
  -> runner.Start
  -> log:<runID> and status:<runID> events
```

Docker compose projects share the project tree but do not use `runner.Spec` for lifecycle commands.
Compose lifecycle is detached through docker CLI; compose logs still reuse runner with `compose:*`
run IDs.

## 4. Wails Binding Surface

`app.go` exposes project, filetree, workspace, group, runtime, Docker, and compose methods. `updater.go`
adds update methods. Changing exported method names, parameters, or return types requires regenerating
bindings:

```bash
export GO=/home/attson/sdk/go1.24.13/bin/go
$($GO env GOPATH)/bin/wails generate module
```

Frontend imports generated bindings from `frontend/wailsjs/go/main/App` and runtime event helpers from
`frontend/wailsjs/runtime/runtime`.

## 5. Runtime Events

| Event | Payload | Producer | Consumer |
|-------|---------|----------|----------|
| `log:<runID>` | `{stream,text}` | runner/app | `LogPanel`, Docker log followers |
| `status:<runID>` | `{state,pid,exitCode}` | runner/app | project and command status maps |
| `update:state` | update state object | updater/app | `UpdateBanner` |
| `docker:available` | `{available,version,reason}` | docker poller | Docker UI |
| `docker:state` | container snapshot | docker poller | project tree and container panel |
| `fs:dir-changed` | relative directory string | filetree watcher/app | `FileBrowser` |

Frontend status maps use Go-shaped keys from polling (`State`, `PID`, `ExitCode`) and normalize
event payloads from lower-case keys.

## 6. Configuration

The config file shape is:

```json
{
  "version": 1,
  "workspaces": [],
  "projects": [],
  "groups": []
}
```

All writes use temporary file plus rename. `store.NormalizeProjectCommands` keeps old config files
compatible with the multi-command model.

## 7. Engineering Constraints

- Local Go commands must use `/home/attson/sdk/go1.24.13/bin/go`; the system `go` is too old.
- Wails dev/build on Ubuntu 24.04 must use `-tags webkit2_41`; Makefile targets already include it.
- Linux tray builds require `libayatana-appindicator3-dev`.
- Do not push directly to `main`. Use feature branch, GitHub PR, green CI, merge, then tag release from
  `main`.
- Commit messages must not include `Co-Authored-By`.

## 8. Known Limits

- Windows process tree cleanup is a process-kill fallback, not a full Job Object implementation.
- Processes that deliberately detach with a new session and no useful parent relationship can outlive
  Stop.
- Compose services are runtime children of compose projects, not persisted group items.
- `ComposeFile` stores one compose file path; multi-file override behavior is left to docker compose
  default discovery.
