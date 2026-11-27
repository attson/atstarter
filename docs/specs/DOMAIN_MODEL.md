# Rootyard Domain Model Spec

## Document Goal

This file fixes the vocabulary and aggregate boundaries used by Rootyard. Code changes, UI labels, and
AI-generated plans must use these terms consistently.

## 1. Problem Domain

Rootyard manages local project launchability and project environments: it turns filesystem directories
into persisted launch targets and runtime processes while centrally resolving repository-local Git and
Codex intent. Docker, files, logs, groups, environments, and updates remain in one desktop app.

## 2. Terminology

| Term | Code Identifier | Meaning | Code Location |
|------|-----------------|---------|---------------|
| Project | `store.Project` | Persisted local directory with launch configuration and detection metadata | `internal/store/model.go` |
| Launch Command | `store.LaunchCommand` | One runnable command configuration under a project | `internal/store/model.go` |
| Launch Group | `store.LaunchGroup` | Persisted list of project command references for batch start/stop | `internal/store/model.go` |
| Workspace | `Config.Workspaces` | Root directory scanned for project candidates | `internal/store/model.go` |
| Run ID | `runner.Spec.ID` | Runtime identity used for status and logs | `internal/runner/runner.go`, `app.go` |
| Detection Option | `store.DetectionOption` | Alternative detected type/command for one directory | `internal/store/model.go` |
| Compose Service | `docker.ComposeService` | Runtime aggregate view of one docker compose service | `internal/docker/docker.go` |
| Container State | `docker.ContainerState` | Runtime snapshot from `docker ps -a` | `internal/docker/docker.go` |
| File Entry | `filetree.Entry` | Direct child item in a project directory | `internal/filetree/filetree.go` |
| Editor Tab | `editorTabs` state | One open file in the project file editor, with its dirty flag and view mode | `frontend/src/components/fileExplorer/editorTabs.js` |
| Branch | `git.Branches` | Local/remote branch list plus worktree status for one project | `internal/git/git.go` |
| Control State | `control.State` | Runtime-only localhost control discovery for CLI/MCP | `internal/control/protocol.go` |
| App Settings | `store.AppSettings` | Cross-project shell and log typography preferences | `internal/store/model.go` |
| Environment Preset | `projectenv.Preset` | Named, reusable Git and Codex field intents independent of filesystem paths | `internal/projectenv/model.go` |
| Directory Rule | `projectenv.Rule` | Canonical directory path, match/discovery scope, optional preset, and inline overrides | `internal/projectenv/model.go` |
| Resolved Environment | `projectenv.ResolvedEnvironment` | Effective field intents and source provenance for one repository path | `internal/projectenv/model.go` |
| Change Plan | `projectenv.ChangePlan` | Immutable, fingerprint-bound preview of native adapter operations for one canonical target | `internal/projectenv/adapter.go` |

### Terminology Constraints

- Use "command" for persisted `LaunchCommand`; do not call it a "script" in config-facing code.
- Use "project" only for persisted launchable directories. A detected scan result becomes a project only
  after it is added to the store.
- Use "run ID" for runtime log/status IDs. Project IDs and command IDs are stable configuration IDs.
- Use "compose service" for Docker compose children. They are not `LaunchCommand` values.
- Use "environment preset" for reusable values and "directory rule" for path matching. Neither save
  operation implies repository mutation.
- A discovered repository is not a launcher `Project`; it becomes an environment target independently.

## 3. Bounded Contexts

### 3.1 Core Domain

The core domain is project launch management: `Project`, `LaunchCommand`, `LaunchGroup`, runtime
`runner.Status`, and ring-buffered logs.

### 3.2 Supporting Domains

| Context | Collaboration | Description |
|---------|---------------|-------------|
| Detection | Feeds store with suggested project type and command | Does not persist by itself |
| Docker | Provides runtime state and lifecycle for containers/compose | Does not write project config except compose detection metadata |
| File Browser | Reads and mutates files under a project root | Does not alter launch semantics |
| Git Branch | Reports and switches the project's git branch | Read-only toward launch config; local git operations only, never network |
| Updater | Manages application binary update state | Does not touch project config |
| Tray | Displays runtime count and exit controls | Delegates process state to runner/app |
| Local Control | Lets CLI/MCP call desktop runtime operations | Does not own business state |
| Project Environment | Resolves presets/rules and orchestrates repository plans | Separate authoritative config; no launcher-project dependency |
| Git Environment Adapter | Maps resolved intent to native Git inspection and apply | Owns Git layout and system `git`; resolver stays adapter-agnostic |
| Codex Environment Adapter | Maps resolved intent to project-level Codex TOML | Owns worktree identity, minimal TOML edits, and atomic replacement |

### 3.3 Generic Domains

| Context | Collaboration | Description |
|---------|---------------|-------------|
| Command Parsing | Used by detection and command edit save paths | Pure parser/serializer boundary |
| Theme System | Used by all frontend components | CSS variable contract, no business state |

### 3.4 Boundary Constraints

- `internal/store` owns persisted project/group/config shape. Runtime-only Docker/service/status values
  must not be added to config unless they become explicit product requirements.
- `internal/runner` owns process lifecycle. UI components must use Wails methods/events rather than
  assuming OS process behavior.
- `internal/filetree` always operates relative to a project root. It must not accept arbitrary absolute
  user paths from the frontend.
- `internal/projectenv` owns `environments.json` and rebuildable `environment-state.json`, not launcher
  `config.json`. Its core is independent of Wails and the desktop process.
- `internal/projectenv/gitadapter` alone interprets common directories, config origins, fingerprints,
  signing tools, and native Git mutations.
- `internal/projectenv/codexadapter` alone interprets `.codex/config.toml`, validates safe paths and
  managed values, and applies parser-validated minimal text edits. It never reads or writes global
  Codex configuration.

## 4. Aggregates

### 4.1 Project

| Attribute | Type | Meaning |
|-----------|------|---------|
| `ID` | string | `sha1(normalizedPath)` hex; de-duplication key |
| `Name` | string | Display name, usually directory basename |
| `Path` | string | Normalized absolute project root |
| `Command`, `Args`, `Cwd`, `Env` | legacy fields | Mirror default command for backward compatibility |
| `DetectedType` | string | Detector-selected type, for UI and compose mode |
| `AutoDetected` | bool | Whether type/command came from detector |
| `Commands` | `[]LaunchCommand` | Multi-command configuration |
| `ComposeFile` | string | Optional compose file path relative to project root |
| `DetectionOptions` | `[]DetectionOption` | Alternative project modes available in UI |

#### State Machine

```text
scan candidate -> persisted project -> missing project
```

| State | Meaning | Constraint |
|-------|---------|------------|
| scan candidate | Detector output not yet saved | May be discarded without config changes |
| persisted project | Exists in config | Must have stable `ID` and normalized command model |
| missing project | Config path no longer exists | Config is preserved until user removes it |

#### Unique Key Rules

| Condition | Participating Fields | Formula |
|-----------|----------------------|---------|
| Project de-duplication | normalized absolute path | `store.IDForPath(path)` |
| Store add idempotency | project ID | Existing path updates are not duplicated |

### 4.2 LaunchCommand

| Attribute | Type | Meaning |
|-----------|------|---------|
| `ID` | string | `default` for default command, otherwise 12-char stable hash |
| `Name` | string | Display label in command picker |
| `Command` | string | Executable token |
| `Args` | `[]string` | Argument tokens |
| `Cwd` | string | Working directory; empty is allowed in legacy data |
| `Env` | `map[string]string` | Environment overrides |
| `IsDefault` | bool | Exactly one normalized command should be default |

#### State Machine

```text
new command -> normalized command -> selected runtime command
```

| State | Meaning | Constraint |
|-------|---------|------------|
| new command | Frontend form row, may have empty ID | Save path assigns ID |
| normalized command | Stored command after `NormalizeProjectCommands` | One default; default ID is `default` |
| selected runtime command | User-selected command for start/log/status | Run ID includes project ID and command ID |

#### Unique Key Rules

| Condition | Participating Fields | Formula |
|-----------|----------------------|---------|
| Default command | default status | `ID == "default"` |
| Non-default command | project ID, command name, command token | first 12 chars of `IDForPath("command:"+projectID+":"+name+":"+line)` |
| Legacy upgrade | old `Project.Command` fields | converted into one default command |

### 4.3 LaunchGroup

| Attribute | Type | Meaning |
|-----------|------|---------|
| `ID` | string | Group identity |
| `Name` | string | Display label |
| `Items` | `[]GroupItem` | References to project command pairs |

| Associated Entity | Type | Load Mode | Meaning |
|-------------------|------|-----------|---------|
| `GroupItem.ProjectID` | string | Resolved at start/stop time | Target project |
| `GroupItem.CommandID` | string | Resolved at start/stop time | Target command |

#### State Machine

```text
saved -> starting -> partially started/running -> stopped
```

| State | Meaning | Constraint |
|-------|---------|------------|
| saved | Group exists in config | Items are references, not embedded command copies |
| starting | `StartGroup` iterates items | Result list reports per-item success/failure |
| running | One or more referenced run IDs are running | Group does not own process lifecycle beyond referenced runs |
| stopped | `StopGroup` stops referenced run IDs | Missing project/command references are skipped or reported by app layer |

### 4.4 Runtime Process

Runtime processes are not persisted aggregates. They are identified by run ID and represented by
`runner.Status`.

```text
stopped -> running -> exited
stopped -> error
running -> error
```

| State | Meaning | Constraint |
|-------|---------|------------|
| stopped | No known running process | Unknown run IDs return stopped |
| running | Process started and PID is known | Duplicate `Start` for same running ID is rejected |
| exited | Process returned an exit code | Runner appends an exit marker to logs |
| error | Start or wait failed outside ordinary exit code | Error status is reported through status event |

### 4.5 Project Environment

An Environment Preset is a reusable value object with a stable opaque ID, display name, optional
description, and independent Git and Codex `FieldMap` values. It has no path and does not inherit from
another preset. A preset in use by Global Defaults or a Directory Rule cannot be deleted.

A Directory Rule has one stable ID and one canonical absolute path. It can match only that exact path
or, when `Recursive` is true, repositories below it. It may reference one preset and then override
fields inline. At most one rule owns a normalized path. `AutoDiscover` controls startup/after-save
discovery only; it does not permit writes.

Resolution applies layers in this order:

```text
global defaults preset -> global defaults inline fields
-> matching rules, shallowest to deepest (each preset then inline fields)
-> request-scoped session fields
```

A Resolved Environment records every effective `FieldIntent` and its default, preset, rule, or session
source for each adapter. Request-scoped session overrides use `EnvironmentFields{Git, Codex}`. A
Change Plan binds that desired result to a repository snapshot fingerprint and displays each
`unchanged`, `set`, `replace`, `clear`, `unmanaged`, or `blocked` operation. Apply consumes that exact
displayed plan; it does not silently resolve or plan again.

#### Field Intent

| Intent | Resolution meaning | Final adapter behavior |
|--------|--------------------|--------------------|
| `inherit` | Preserve the lower layer's effective field; missing fields are equivalent | Unmanaged when no lower layer manages the key |
| `set(value)` | Replace the effective field and provenance | Set the validated project-local value |
| `clear` | Replace the effective field and provenance with explicit removal | Remove the project-local value for the key |

The Git adapter supports exactly `user.name`, `user.email`, `user.signingKey`, `commit.gpgSign`,
`tag.gpgSign`, and `gpg.format`.

The Codex adapter supports exactly `model`, `model_reasoning_effort`, `sandbox_mode`,
`approval_policy`, and `web_search`. Model and reasoning values are non-empty free-form strings.
Sandbox values are `read-only`, `workspace-write`, or `danger-full-access`; approval values are
`untrusted`, `on-failure`, `on-request`, `granular`, or `never`; web search values are `disabled`,
`cached`, `indexed`, or `live`.

#### Target Identity

The canonical Git common directory, not an encountered worktree path, identifies a Git apply target.
Linked worktrees sharing a common directory therefore share one local Git configuration and one target.
Their paths remain aliases for display and rule resolution. If aliases resolve to different desired
values, `shared_target_conflict` blocks the target rather than choosing one alias implicitly.

Each canonical worktree root identifies one Codex target whose config path is
`<worktree>/.codex/config.toml`. Linked worktrees therefore share one Git plan but retain independent
Codex plans. A plan for a missing Codex file fingerprints the absence distinctly from an empty file.

## 5. Value Objects

| Value Object | Attributes | Notes |
|--------------|------------|-------|
| `runner.Spec` | `ID`, `Command`, `Args`, `Dir`, `Env` | One start request |
| `runner.LogLine` | `ID`, `Stream`, `Text` | `Stream` is `stdout` or `stderr` |
| `docker.Info` | `Available`, `Version`, `Reason` | Availability probe result |
| `filetree.FileBytes` | `Data`, `ModTime`, `IsBinary`, `TruncatedAt` | Used for editor/preview bytes |
| `projectenv.FieldIntent` | `Mode`, `Value` | Explicit `inherit`, `set(value)`, or `clear` intent |
| `projectenv.Snapshot` | `Target`, `Fingerprint`, `Values`, `Issues`, `Exists` | Current adapter-native state and blockers |
| `projectenv.Target` | `ID`, `WorktreePath`, `CommonDir`, `ConfigPath` | Canonical adapter target plus an access path |
| `projectenv.TextEdit` | `Start`, `End`, `Replacement` | Exact parser-guided Codex edit stored in a displayed plan |

## 6. Data Mapping

| Storage | Entity | Description |
|---------|--------|-------------|
| `config.json.workspaces[]` | Workspace | Scan roots |
| `config.json.projects[]` | Project + LaunchCommand | Project config and commands |
| `config.json.groups[]` | LaunchGroup | Batch start/stop definitions |
| runner maps | Runtime Process | In-memory only |
| docker CLI snapshots | ContainerState, ComposeService | Runtime only |
| `<config>.control.json` | Control State | Runtime-only discovery, removed on shutdown |
| `config.json.settings` | App Settings | Cross-project runtime and log display preferences |
| `environments.json.defaults` | Global Defaults | First environment resolution layer |
| `environments.json.presets[]` | Environment Preset | Authoritative reusable field intents |
| `environments.json.rules[]` | Directory Rule | Authoritative path-scoped field intents |
| `environment-state.json.repositories[]` | Repository records, aliases, and per-adapter states | Rebuildable version 2 scan/apply status; never resolution input; valid version 1 state migrates into the Git adapter entry |

## 7. Extension Registry

| Field | Scenario | Affects Unique Key | Purpose |
|-------|----------|--------------------|---------|
| `Project.ComposeFile` | Compose file disambiguation | No | Pass `-f` relative file to compose |
| `Project.DetectionOptions` | Switch between compose and ordinary command modes | No | UI type switch |
| `LaunchCommand.Env` | Per-command environment overrides | No | Launch-time env overlay |
| `LaunchCommand.Cwd` | Per-command working directory | No | Launch-time working dir |
