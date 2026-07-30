# Runtime Contracts Spec

## Document Goal

This file describes backend behavior that must remain stable across detector, store, runner, Docker,
updater, Wails bindings, and events.

## 1. Core Interfaces

```go
// Detection -> parse -> persistence -> runtime
detector.Detect(dir) Result
cmdparse.Parse(line) (command string, args []string, err error)
store.NormalizeProjectCommands(project) store.Project
runner.Start(runner.Spec{ID, Command, Args, Dir, Env})
```

```go
// Wails-facing runtime events
log:<runID>       {stream,text}
status:<runID>    {state,pid,exitCode}
update:state      UpdateState
docker:available  docker.Info
docker:state      []docker.ContainerState
fs:dir-changed    string // changed relative directory
```

```go
// Local control discovery for CLI/MCP
control.WriteState(<config>.control.json, {url, token, pid, version})
control.Client.Call(method, params, out)
```

### Interface Constraints

- Detector suggestions are defaults, not hidden rules. Users can override command, args, cwd, env, and
  detection mode.
- Store normalization must keep legacy single-command configs readable and writable.
- Wails bindings are the only frontend/backend API. Do not make frontend code depend on local files
  outside Wails methods.
- CLI and MCP must call the desktop control server. They must not instantiate their own long-lived
  runner, Docker poller, or store mutation path.

## 2. Dispatch Rules

| Key/Condition | Implementation | Description |
|---------------|----------------|-------------|
| Project ID | `store.IDForPath(normalizedPath)` | De-duplicates project config |
| Command ID `default` | normalized default command | Mirrors legacy `Project.Command` fields |
| Run ID project default | `<projectID>:default` | Status/log identity for default command |
| Run ID project command | `<projectID>:<commandID>` | Status/log identity for non-default command |
| Run ID container logs | `container:<containerID>` | Runner-managed `docker logs -f` |
| Run ID compose logs | `compose:<projectID>[:<service>]` | Runner-managed `docker compose logs -f` |
| Compose lifecycle | docker CLI detached commands | Not persisted as runner processes |
| Control state path | `<config path>.control.json` | Runtime-only CLI/MCP discovery file |

### Dispatch Constraints

- `StartProject` and `StopProject` target the normalized default command.
- `StartProjectCommand` and `StopProjectCommand` target explicit command IDs.
- `StartGroup` and `StopGroup` operate on stored `GroupItem{projectId,commandId}` references.
- Compose services are resolved from docker at runtime and must not be serialized into groups.
- Control RPC project, command, and group targets may accept IDs or names, but ambiguous name matches
  must fail and ask for an ID.
- Control RPC scan/add/switch/group-management methods must delegate to `App`/`store` helpers that
  already normalize projects, commands, detection options, and group item command IDs.

## 3. Implementation Patterns

### 3.1 Store Write Pattern

Applicable to config changes.

1. Normalize paths before generating IDs.
2. Normalize project commands before save/list return paths.
3. Write config through temporary file plus rename.

Reference implementation: `internal/store/store.go`, `internal/store/model.go`.

### 3.2 Runner Start Pattern

Applicable to all project commands and log follow commands.

1. Build `runner.Spec` with stable run ID.
2. On Unix, wrap command with `$SHELL -l -i -c` and shell-quote each token after `~` expansion.
3. Build process env from `os.Environ()` plus command env overrides; expand override values with
   process env first so values like `PATH=/custom/bin:$PATH` work without executing shell syntax.
4. On Unix, re-apply valid shell variable env overrides inside the `-c` line after shell startup,
   so login/interactive rc files and version managers cannot prepend older tools ahead of a
   command-specific `PATH`.
5. Set process attributes so Stop can clean process descendants.
6. Start stdout/stderr pumps before wait.
7. Emit status/log callbacks outside the runner lock.

Reference implementation: `internal/runner/runner.go`, `internal/runner/process_unix.go`.

### 3.3 Runner Stop Pattern

Applicable to user Stop, group stop, app exit, and log follow cleanup.

1. Read managed process under lock.
2. Release lock before slow process-tree cleanup.
3. On Unix, terminate descendants first with SIGTERM, kill top interactive shell, then SIGKILL fallback.
4. Treat unknown or already-stopped run IDs as success.

Reference implementation: `internal/runner/process_unix.go`.

### 3.4 Docker Pattern

Applicable to container panel and compose detail.

1. Resolve docker CLI through PATH plus common Docker Desktop locations.
2. Use injected `execFunc` in tests; parsers stay pure.
3. Use short app-layer timeouts for detection/list/config.
4. Use lifecycle commands detached where applicable (`compose up -d`).
5. Poll snapshots and emit only meaningful availability/state updates.

Reference implementation: `internal/docker`, `app.go`.

### 3.5 Updater Pattern

Applicable to release checks and installs.

1. Fetch GitHub latest release state.
2. Select asset by platform/arch.
3. Expand standard GitHub release URLs through configured mirrors, with original URL as final fallback.
4. Verify `SHA256SUMS.sig` with Ed25519 public key and verify asset SHA256.
5. Run embedded installer script detached, then quit the app.

Reference implementation: `updater.go`, `scripts/install-*`.

### 3.6 Local Control Pattern

Applicable to `control_server.go`, `cli.go`, `mcp.go`, and `internal/control`.

1. Desktop startup opens a localhost listener on `127.0.0.1:0`.
2. The desktop process writes `<config>.control.json` with URL, token, PID, and version using `0600`
   permissions.
3. Every RPC request must include `Authorization: Bearer <token>`.
4. RPC methods delegate to existing `App` methods and preserve existing Run ID rules.
5. Project detection switching uses the same data transformation as the UI: switching to `compose`
   clears `LaunchCommand` entries; switching to an ordinary detection option creates a default
   `LaunchCommand`.
6. CLI output uses `{ok,data,error}` JSON envelopes. `--follow` log commands emit one envelope per
   update as JSON Lines.
7. MCP tools wrap the same control client and return the same envelope as text content.
8. Shutdown removes the control state file.

Reference implementation: `control_server.go`, `cli.go`, `mcp.go`, `internal/control/protocol.go`.

## 4. Allowed And Forbidden

### Allowed Changes

- Add detector rules when tests cover expected files and suggested command.
- Add frontend-only transformation helpers when covered by `node:test`.
- Add Wails methods when a frontend feature needs a clear backend boundary.
- Add docker parser support through pure parser tests with fake CLI output.

### Forbidden Changes

- Do not bypass `cmdparse` with ad-hoc command splitting.
- Do not read/write `runner.managed.status` outside `r.mu`.
- Do not call status/log callbacks while holding `r.mu`.
- Do not run `killTree` or other slow process cleanup under `r.mu`.
- Do not remove Unix login shell wrapping without solving GUI PATH and `~` expansion.
- Do not trust frontend-provided file paths outside the project root.
- Do not push release tags from a non-main commit.
- Do not let CLI/MCP bypass the desktop process for runner, Docker, project, group, or log state.
- Do not persist control URL/token/PID/version into `config.json`.

## 5. New Implementation Checklist

### Required

- [ ] Preserve project ID and command ID stability.
- [ ] Preserve old config compatibility through normalization.
- [ ] Keep runtime-only state out of `config.json`.
- [ ] Add or update Go tests for `internal/*` behavior changes.
- [ ] Regenerate Wails bindings after exported App signature changes.

### Conditional

- [ ] If changing `internal/runner` concurrency, run `$GO test -race ./internal/runner/`.
- [ ] If changing Wails build commands, keep `-tags webkit2_41` for Ubuntu 24.04.
- [ ] If changing docker behavior, cover parser/command construction with fake exec tests.
- [ ] If changing updater install/download, cover cancellation, mirror fallback, and verification paths.

### Verification

- [ ] `export GO=/home/attson/sdk/go1.24.13/bin/go`
- [ ] `$GO test ./...`
- [ ] Frontend pure tests with `node --test ...`
- [ ] `cd frontend && npm run build` for UI or binding changes.
