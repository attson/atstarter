# Rootyard Project Environments Design

**Date:** 2026-11-26

**Status:** Ready for written-spec review

**Phase:** 1 - Git identity and signing

## 1. Summary

Rootyard will add a local project-environment subsystem that resolves directory-scoped configuration,
shows the effective result for discovered Git repositories, previews native configuration changes, and
applies selected changes only after explicit user confirmation.

Phase 1 delivers the complete shared engine, an offline `rootyard env` CLI, a desktop management UI,
and a Git adapter for identity and signing configuration. Phase 2 will add Codex configuration. Phase 3
will add Claude Code configuration. Later adapters reuse the same presets, directory rules, discovery,
planning, and batch-application boundaries rather than adding separate configuration systems.

Rootyard management data remains centralized under the Rootyard user configuration directory. It does
not add `.rootyard` files to projects. Native target files such as `.git/config` remain the only files
written inside or adjacent to repositories.

## 2. Goals

- Manage project environment intent centrally without polluting project directories.
- Reuse named presets across directory rules while allowing rule-specific field overrides.
- Resolve broad directory rules before more specific directory rules deterministically.
- Automatically discover Git repositories under rules whose discovery switch is enabled.
- Show field sources, effective values, native configuration differences, drift, and blocking errors.
- Require explicit confirmation before writing any repository configuration.
- Apply a batch without allowing one repository failure to stop unrelated repositories.
- Provide the same behavior through the desktop UI and an offline command-line surface.
- Establish adapter boundaries that can add Codex and Claude Code in later phases.

## 3. Non-goals

- Codex, Claude Code, or other non-Git adapters in Phase 1.
- Automatic synchronization or writes triggered by scanning, opening, or saving a rule.
- Team sharing, cloud synchronization, or portable path mapping between computers.
- Storing Rootyard management files inside projects.
- Importing or exporting presets in Phase 1.
- Arbitrary Git key/value editing beyond the six supported identity and signing fields.
- Git network operations, commits, branch operations, or remote configuration.
- Per-worktree identities for linked worktrees.
- Arbitrary historical rollback. A failed apply is atomic; successful prior applies are not retained as
  restorable versions.
- Restricting discovery to projects already added to the Rootyard launcher.

## 4. Confirmed Product Model

### 4.1 Central configuration

Project-environment configuration is stored separately from the existing launcher `config.json`:

```text
<RootyardConfigDir>/
  config.json                 existing projects, groups, workspaces, and settings
  environments.json          user-authored project-environment intent
  environment-state.json     rebuildable discovery and apply state
```

`<RootyardConfigDir>` is the directory containing the resolved Rootyard config path. Therefore a
`ROOTYARD_CONFIG` override relocates all three files together without introducing another environment
variable.

`environments.json` is authoritative input. `environment-state.json` is derived state and never affects
resolution. If state is absent or invalid, Rootyard can rebuild it from the configuration and target
repositories.

### 4.2 Domain terms

| Term | Meaning |
|------|---------|
| Environment Preset | Named reusable adapter field values, independent of filesystem paths |
| Directory Rule | One normalized directory path, match scope, discovery setting, preset reference, and inline overrides |
| Global Defaults | Optional root layer applied before every matching directory rule |
| Field Intent | One of `inherit`, `set(value)`, or `clear` |
| Resolved Environment | Effective field intent plus source provenance for one repository |
| Repository Snapshot | Current managed Git values, configuration fingerprint, and validation results |
| Change Plan | Immutable preview of operations against one repository snapshot |
| Drift | A previously consistent native configuration now differs from the latest resolved target |
| Stale Plan | Native configuration changed after a plan was previewed and before it was applied |

The existing persisted `Project` remains a launchable directory. A discovered repository is not
automatically a `Project` and is not added to `config.json`.

## 5. Architecture

```text
environments.json
       |
       v
internal/projectenv
  Store -------- schema, locking, atomic persistence
  Resolver ----- layer merge and field provenance
  Discovery ---- repository scan and canonical identity
  Planner ------ adapter snapshot -> change plan
  Service ------ CRUD, scan, resolve, diff, apply, doctor
       |
       +---- rootyard env CLI
       |
       +---- Wails App methods
                |
                +---- Vue project-environment workspace
       |
       +---- Git adapter
              Inspect / Validate / Plan / Apply
```

`internal/projectenv` does not depend on Wails, Vue, the running desktop process, or the existing
launcher project list. The Wails binding layer and CLI are consumers of the same service.

The Git adapter owns native Git representation and operations. The resolver does not know `.git`
layout or Git command syntax. Future Codex and Claude adapters implement the same high-level adapter
contract without changing rule resolution or batch semantics.

### 5.1 Offline CLI exception

Existing `rootyard cli` and MCP operations for processes, Docker, logs, projects, and groups continue to
use the desktop control server. Project environments are different: they are offline file-backed state
with no long-lived runtime owner. Top-level `rootyard env` commands may instantiate
`internal/projectenv.Service` directly and work while the desktop app is stopped.

Cross-process file locks serialize desktop and offline CLI mutations. The implementation must update
the long-lived architecture and runtime specs to document this narrow exception. It must not be used to
bypass the desktop process for existing runtime features.

## 6. Configuration Schema

### 6.1 File shape

```json
{
  "version": 1,
  "defaults": {
    "presetId": "",
    "git": {
      "user.name": { "mode": "inherit" },
      "user.email": { "mode": "inherit" },
      "user.signingKey": { "mode": "inherit" },
      "commit.gpgSign": { "mode": "inherit" },
      "tag.gpgSign": { "mode": "inherit" },
      "gpg.format": { "mode": "inherit" }
    }
  },
  "presets": [
    {
      "id": "preset_company",
      "name": "Company identity",
      "description": "Identity used for company repositories",
      "git": {
        "user.name": { "mode": "set", "value": "Attson" },
        "user.email": { "mode": "set", "value": "attson@company.example" },
        "user.signingKey": { "mode": "set", "value": "ABCD1234EFGH5678" },
        "commit.gpgSign": { "mode": "set", "value": "true" },
        "tag.gpgSign": { "mode": "inherit" },
        "gpg.format": { "mode": "set", "value": "openpgp" }
      }
    }
  ],
  "rules": [
    {
      "id": "rule_company_projects",
      "name": "Company projects",
      "path": "/Users/example/Projects/Company",
      "recursive": true,
      "autoDiscover": true,
      "presetId": "preset_company",
      "git": {
        "user.email": { "mode": "set", "value": "team@example.com" }
      }
    }
  ]
}
```

Unknown top-level versions and unknown fields within version 1 are rejected. Rootyard must never load
an unknown field and then silently discard it during a later write. An absent file initializes an empty
version 1 model only when the first mutation is explicitly saved.

Preset and rule IDs are opaque, stable identifiers generated by the service. Names are display values
and can be changed without invalidating references. Rule paths are stored as normalized absolute paths.
Only one rule may own a normalized path; attempting to create a duplicate is a validation error. This
removes ambiguous same-path precedence.

A rule path must exist and be a directory when the rule is created. Resolution and discovery use
platform-aware absolute-path cleaning, symbolic-link evaluation for the rule root, and case handling.
A rule remains stored if its directory later disappears and reports a diagnostic until the path returns
or the user edits or removes the rule.

### 6.2 Supported Git fields

Phase 1 manages exactly:

| Key | Value validation |
|-----|------------------|
| `user.name` | non-empty string when set |
| `user.email` | non-empty string when set; conservative email-shape warning only |
| `user.signingKey` | non-empty string when set |
| `commit.gpgSign` | canonical `true` or `false` |
| `tag.gpgSign` | canonical `true` or `false` |
| `gpg.format` | `openpgp`, `ssh`, or `x509` |

The UI may use toggles or option controls but persists canonical string values expected by Git.

### 6.3 Field intent

Each layer can express:

| Mode | Resolution effect | Apply effect when final |
|------|-------------------|-------------------------|
| `inherit` | Leaves the value from the previous layer unchanged | No operation when no lower layer manages the field |
| `set` | Replaces the effective intent and records the value and source | Replace all repository-local values with one value |
| `clear` | Replaces the effective intent and records the source | Remove all repository-local values |

Missing fields are equivalent to `inherit`. `set` requires a valid value. `inherit` and `clear` must
not carry a value.

## 7. Resolution Semantics

### 7.1 Rule matching

For a canonical repository path:

1. Start with all fields unresolved.
2. Apply Global Defaults: its referenced preset first, then its inline fields.
3. Select each directory rule once. A strict ancestor matches only when `recursive` is true; a rule
   whose path equals the repository path always matches.
4. Sort matching rules from shallowest path to deepest path.
5. Apply each rule's referenced preset, then the rule's inline fields.
6. Apply optional request-scoped session overrides last.
7. Return every effective intent with its originating default, preset, rule, or session source.

`recursive: false` matches only when its path is itself a Git repository. It does not cause child
discovery. Paths use component-aware ancestry rather than string-prefix matching.

Session overrides are in-memory request options used for previews or CLI invocations through the
repeatable `--git-set`, `--git-clear`, and `--git-inherit` flags. They are not persisted and Phase 1
does not add a separate session-management UI.

### 7.2 Preset behavior

Presets do not inherit from other presets and cannot compose multiple presets directly. One default or
rule references at most one preset, then supplies inline overrides. Directory layering provides
composition without creating a preset dependency graph.

Editing a preset recalculates every referencing rule and repository. It does not write native files.
Affected repositories become `pending` or `drifted` after inspection.

A preset referenced by Global Defaults or any rule cannot be deleted. The user must first detach or
replace those references. Cloning creates a new ID and copies the field values; it does not copy rule
references.

## 8. Repository Discovery

### 8.1 Discovery behavior

Each rule has an independent `autoDiscover` switch, defaulting to false. Enabled discovery runs in the
background at desktop startup and after that rule is saved. Manual scan remains available for every
rule. Phase 1 does not add a continuous filesystem watcher or scheduled rescans.

Discovery:

- recognizes repositories with a `.git` directory;
- recognizes linked worktrees whose `.git` is a file;
- stops descending once a repository root is found;
- does not follow directory symbolic links;
- skips exactly `.git`, `node_modules`, `vendor`, and `.cache` directories while walking;
- records unreadable directories as warnings and continues elsewhere;
- supports cancellation and incremental result delivery to the desktop UI;
- never writes repository configuration.

The rule root itself is canonicalized even when the user selected it through a symbolic link. Descendant
symbolic links are not followed, preventing cycles and escape from the selected root.

### 8.2 Repository identity and worktrees

The adapter resolves both the worktree root and Git common directory through Git. Repositories sharing
the same canonical common directory are one apply target because `git config --local` is shared across
linked worktrees. Discovery may display all encountered paths. Planning and batch application
de-duplicate aliases when they resolve to the same desired target and explain the shared target in
UI/CLI output.

If aliases of one common directory match different rules and resolve to different desired values, the
target is blocked with `shared_target_conflict`. Rootyard shows every path and rule contributing to the
conflict and does not choose one path's configuration implicitly.

Per-worktree configuration through `git config --worktree` is outside Phase 1.

## 9. Git Adapter

### 9.1 Inspection

The adapter shells out to the system Git executable with bounded timeouts. It does not parse Git config
syntax itself. Inspection reads all local values for the six supported keys and distinguishes:

- key absent;
- one local value;
- repeated equal values;
- multiple conflicting values;
- invalid repository or configuration syntax;
- unreadable or unwritable configuration.

Repeated or conflicting values remain visible in the snapshot. A `set` plan replaces all local values
with one canonical value; a `clear` plan removes all local values. A conflict is a blocking issue until
the user explicitly previews an operation that resolves it.

Inspection retains Git's origin for every local-scope value. Rootyard writes only the repository's
canonical main config file. If a managed local value originates from a file included by that config,
Rootyard reports `external_local_origin` and blocks management of that key. It never edits or attempts
to override an included local file implicitly. System and global values are outside the local snapshot:
a local `set` overrides them, while `clear` intentionally exposes them again.

The snapshot includes a SHA-256 fingerprint of the native configuration bytes and target metadata
required for concurrency checks. The state file stores the fingerprint, not a duplicate copy of native
configuration contents.

### 9.2 Signing validation

When either resolved signing flag is true, `doctor` and plan validation verify that the resolved format,
signing key, and required local signing toolchain are usable. Validation is format-aware and bounded by
timeouts. A missing explicitly referenced key or unavailable required signing program is a blocking
issue. Checks do not access the network and do not create signatures or commits.

Warnings that cannot be proved locally remain warnings; Rootyard must not claim a key is usable when
the required verifier was unavailable.

### 9.3 Change plan

For every managed field the planner emits one of:

| Operation | Meaning |
|-----------|---------|
| `unchanged` | Current local values already match the final intent |
| `set` | Key is absent and one value will be added |
| `replace` | Existing local value(s) will be replaced with one value |
| `clear` | Existing local value(s) will be removed |
| `unmanaged` | Final intent is inherit; no native change is planned |
| `blocked` | Validation or repository state prevents safe application |

A plan contains repository identity, rule and preset sources, current snapshot, desired field intents,
operations, warnings, blockers, and the pre-apply fingerprint. UI and CLI render this same plan model.

### 9.4 Atomic application

Before applying a plan, the adapter:

1. Re-resolves the repository identity and target config path.
2. Acquires the native configuration lock without waiting indefinitely.
3. Re-reads and fingerprints the current configuration.
4. Rejects the plan as stale if the fingerprint differs from the preview.
5. Copies the configuration to a temporary file in the same directory while preserving safe mode bits.
6. Uses Git CLI operations against the temporary file to apply every planned set, replace, and clear.
7. Re-inspects the temporary file and verifies it matches the resolved target.
8. Flushes and atomically replaces the native configuration.
9. Releases the lock and records the result.

An error before replacement leaves the original file unchanged. Temporary files are cleaned up. The
implementation must never apply one field directly to the real file and then continue with another,
because that would expose partial per-repository state.

Batch application treats every canonical repository target as an independent unit. A failure, stale
preview, or blocker for one target does not prevent other selected targets from completing. Results
preserve input order and report success, unchanged, skipped, stale, blocked, or failed per target.

## 10. Persistence and Concurrency

### 10.1 Environment store

Desktop and CLI writers use a shared advisory lock adjacent to `environments.json`. Mutations load the
latest file only after acquiring the lock, validate the complete next model, write a temporary file,
flush it, and atomically rename it. Unix files use user-only permissions where supported.

Every load returns a content-hash revision. Desktop mutations that originate from an earlier read send
that expected revision and fail with `environment_stale` when another process changed the file. A CLI
mutation performs its read-modify-write cycle under one lock. This prevents a long-lived desktop form
from silently overwriting an offline CLI change.

If `environments.json` is syntactically invalid or has an unsupported version, Rootyard returns a
configuration error and preserves the original file. It must not replace it with defaults.

### 10.2 Derived state

`environment-state.json` may contain:

- last successful discovery time per rule;
- canonical repository identities and display paths;
- matching rule IDs and winning field-source IDs;
- last observed native fingerprint;
- last resolved target hash;
- last apply time and result code;
- scan warnings and repository blockers.

It does not contain authoritative presets or rules. If state is invalid, Rootyard moves it aside with a
timestamped corrupt suffix and rebuilds it. Rebuilding state never applies changes.
State writes use their own adjacent cross-process lock and atomic replacement so background desktop
discovery cannot corrupt a simultaneous CLI scan.

## 11. Desktop Experience

Project environments are a first-class workspace in the desktop navigation, not a modal inside app
settings. Phase 1 provides four views.

### 11.1 Overview

The overview shows counts for rules, matched repositories, pending repositories, native field changes,
and blocking issues. It provides manual scan and a direct path to pending work.

### 11.2 Presets

The preset view supports:

- create a blank preset;
- clone the selected preset;
- rename and edit its optional description;
- edit each supported field using `set`, `inherit`/"not managed", or `clear`;
- inspect referencing defaults, rules, repositories, and pending impact;
- save without applying;
- block deletion while references remain.

The UI uses "Not managed" for preset-level `inherit` where that wording is clearer, while persisting the
same `inherit` mode.

### 11.3 Directory rules

The rule view supports path selection, exact/recursive scope, the auto-discovery switch, one preset
reference, and inline field overrides. It shows matched repositories and allows navigation through
configuration, repository selection, and change preview without nesting cards.

The protected Global Defaults row is shown separately from path-backed rules. It can reference a
preset and contain inline overrides but has no path or discovery settings.

### 11.4 Repositories and apply flow

The repository view groups or filters discovered repositories by rule and status. The primary flow is:

```text
create preset
  -> create directory rule
  -> discover repositories
  -> inspect effective values and sources
  -> select repositories
  -> preview exact native changes
  -> confirm manual apply
  -> inspect per-repository results
```

`pending`, `consistent`, `drifted`, `blocked`, and `failed` are visually distinct. Blocking rows cannot
be selected for application. Long paths are single-line, truncated, monospaced, and available in full
through hover text.

Vue calls Wails bindings only. It does not open environment JSON, walk directories, or execute Git.
Pure selection, filter, and presentation transforms belong in small frontend modules with `node:test`
coverage. Components use the existing UI primitives, theme tokens, and `lucide-vue-next` icons.

## 12. CLI Surface

The top-level CLI is:

```text
rootyard env defaults get [--json]
rootyard env defaults set [--preset <id>|--no-preset] [field options] [--json]

rootyard env preset list [--json]
rootyard env preset get <id> [--json]
rootyard env preset set [--id <id>] --name <name> [--description <text>] [field options] [--json]
rootyard env preset clone <id> --name <name> [--json]
rootyard env preset delete <id> [--json]

rootyard env rule list [--json]
rootyard env rule get <id> [--json]
rootyard env rule set [--id <id>] --name <name> --path <path>
  [--recursive|--exact] [--auto-discover|--no-auto-discover]
  [--preset <id>|--no-preset] [field options] [--json]
rootyard env rule delete <id> [--json]

rootyard env scan [--rule <id>] [--json]
rootyard env resolve <repo> [session field options] [--json]
rootyard env diff <repo> [session field options] [--json]
rootyard env diff --rule <id> [session field options] [--json]
rootyard env apply <repo> [session field options] [--yes] [--json]
rootyard env apply --rule <id> [--repo <path> ...] [session field options] [--yes] [--json]
rootyard env doctor [<repo>|--rule <id>] [--json]
```

Human-readable `diff` is the default. Query commands support `--json` with a stable envelope. Apply
prints a preview and asks for confirmation when attached to an interactive terminal. Non-interactive
apply refuses to write unless `--yes` is present. `--yes` confirms the already computed plan; stale
fingerprints still fail rather than being silently refreshed.

`apply --json` never emits an interactive prompt. Without `--yes` it returns the plan with a
`confirmation_required` result and performs no write; with `--yes` it returns one JSON result envelope
after the batch completes.

Field options have one unambiguous grammar and are repeatable:

```text
--git-set <key>=<value>
--git-clear <key>
--git-inherit <key>
```

For preset, defaults, and rule `set`, unspecified fields retain their current value during an update
and default to `inherit` during creation. For resolve, diff, and apply, these flags are request-scoped
session overrides. Invalid field names or values fail before persistence or planning.

On updates, omitted non-field options also retain their current value. On creation, `recursive` and
`autoDiscover` default to false and no preset is referenced. `--exact` is the explicit inverse of
`--recursive`; `--no-auto-discover` and `--no-preset` allow existing values to be cleared.

`scan` without `--rule` manually scans every rule, including rules whose automatic-discovery switch is
off. `apply --rule` considers all currently discovered apply targets for that rule; repeatable `--repo`
narrows the selection. Repository-path commands accept any valid local Git repository and do not
require prior discovery.

## 13. Errors and Diagnostics

UI and CLI use the same structured error codes. Categories are:

| Category | Examples | Batch behavior |
|----------|----------|----------------|
| Configuration error | invalid JSON, unsupported version, broken preset reference | stop the requested operation |
| Validation error | duplicate normalized rule path, invalid field value, `environment_stale` | reject the mutation |
| Scan warning | unreadable directory, canceled subtree | continue other directories |
| Repository blocker | invalid Git config, missing key, `external_local_origin`, `shared_target_conflict` | show but do not select/apply |
| Stale plan | config fingerprint changed after preview | skip target and require new preview |
| Apply failure | lock contention, temporary write failure, atomic replace failure | fail target, continue batch |

`doctor` checks the environment store, rule paths, preset references, Git availability, repository
identity, native config readability/writability, duplicate worktree targets, signing prerequisites, and
state-file rebuildability. It is read-only.

Git command failures preserve useful Git stderr in a detail field while exposing a stable Rootyard error
code and concise message.

## 14. Testing Strategy

### 14.1 Resolver tests

Table-driven tests cover:

- defaults, outer directories, inner directories, exact rules, and session override precedence;
- preset-before-inline ordering at every layer;
- `inherit`, `set`, and `clear` behavior;
- component-aware path ancestry and normalized duplicate rejection;
- missing preset references and deterministic source provenance.

### 14.2 Discovery tests

Temporary trees cover ordinary repositories, `.git` files, linked worktrees, nested repositories,
ignored directories, unreadable paths where the platform permits, symbolic links, cancellation, and
common-directory de-duplication.

### 14.3 Git adapter integration tests

Temporary Git repositories cover absent and existing fields, repeated values, conflicting values,
replace and clear plans, invalid config, detached worktrees, linked worktrees, signing validation,
fingerprint drift, lock contention, atomic apply, and cleanup after injected failures.

### 14.4 Store and service tests

Tests cover schema validation, corrupt authoritative config preservation, atomic writes, cross-process
lock contention, state-file quarantine/rebuild, preset deletion protection, batch partial failure,
stable result ordering, and cancellation.

### 14.5 Frontend tests

Pure helpers and component tests cover preset CRUD state, mode/value enabling, rule validation,
repository filtering and selection, blocking rows, diff confirmation, stale-plan results, and batch
summary rendering. The frontend build verifies generated Wails binding integration.

### 14.6 Required verification

Implementation completion requires at least:

```bash
export GO=/home/attson/sdk/go1.24.13/bin/go
$GO test ./...
cd frontend && npm run build
node --test frontend/src/*.test.mjs frontend/src/composables/*.test.mjs
```

Focused race tests are required for the project-environment store and service if they introduce
concurrent state or background discovery workers.

## 15. Long-lived Documentation Changes

The implementation must update:

- `docs/specs/ARCHITECTURE.md` with `internal/projectenv`, the environment files, desktop view, and
  offline `rootyard env` exception;
- `docs/specs/DOMAIN_MODEL.md` with Environment Preset, Directory Rule, Resolved Environment, Change
  Plan, and discovered repository terminology;
- `docs/specs/RUNTIME_CONTRACTS.md` with service, adapter, atomic apply, and CLI safety contracts;
- `docs/specs/FRONTEND_STYLE.md` with the project-environment workspace and preview/apply interaction;
- `CLAUDE.md` with the new module and required focused tests;
- user-facing CLI and desktop documentation after behavior is implemented.

## 16. Phase Boundaries

Phase 1 is complete only when the shared service, Git adapter, offline CLI, desktop management UI,
manual batch apply, diagnostics, tests, and documentation all work as one vertical slice.

Phase 2 adds a Codex adapter for approved Codex configuration surfaces such as rules, skills, plugins,
and MCP configuration. Phase 3 adds the corresponding Claude Code adapter. Those phases require their
own design work for native file ownership and merge semantics, but they must reuse Phase 1 presets,
rules, discovery, planning, batch results, and manual-apply safety model.
