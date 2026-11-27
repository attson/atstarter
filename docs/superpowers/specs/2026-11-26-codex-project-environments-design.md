# Rootyard Codex Project Environments Design

**Date:** 2026-11-26

**Status:** Ready for written-spec review

**Phase:** 2 - Codex project configuration

## 1. Summary

Rootyard will extend the existing Project Environments subsystem with a Codex adapter that manages a
small, explicit set of root-level values in each Git worktree's `.codex/config.toml`. Git and Codex
intent remains in one Environment Preset and one Directory Rule model. The existing resolver,
discovery, preview, explicit-confirmation, stale-plan, atomic-apply, desktop, and offline CLI behavior
continues to be shared.

Phase 2 manages only `model`, `model_reasoning_effort`, `sandbox_mode`, `approval_policy`, and
`web_search`. It never modifies global Codex configuration or profiles. It preserves every unmanaged
project-level TOML key, table, comment, and ordering decision, and blocks invalid or unsafe-to-edit
TOML rather than rewriting it.

The official OpenAI Codex configuration reference returned an HTTP 403 from Vercel during design on
2026-11-26. Compatibility was therefore checked against the locally installed `codex-cli 0.170.0`
configuration loader. Rootyard must not claim model or account availability and must keep future-
fragile string fields open to free input. This limitation is a design constraint, not a reason to use
network access during normal Rootyard operation.

## 2. Goals

- Manage the five approved Codex project settings through the same presets and directory rules as Git.
- Create `.codex/config.toml` after explicit confirmation when it does not exist.
- Preserve unmanaged TOML content byte-for-byte wherever no managed edit overlaps it.
- Show the exact managed changes before apply and write only the displayed plan.
- Treat each linked worktree as an independent Codex target.
- Preserve Phase 1 stale detection, atomic replacement, batch isolation, and offline CLI behavior.
- Warn prominently about dangerous sandbox and approval combinations without silently blocking a
  deliberate choice.
- Keep Doctor offline and useful even when the Codex executable is absent.

## 3. Non-goals

- Modifying `~/.codex/config.toml`, `$CODEX_HOME/config.toml`, named profile files, or global profiles.
- Managing MCP servers, hooks, agents, skills, feature flags, arbitrary TOML keys, or nested tables.
- Discovering arbitrary non-Git project directories in Phase 2.
- Querying OpenAI services, account entitlements, model availability, or authentication state.
- Validating that a free-form model or reasoning value is supported by the user's account.
- Calling `codex doctor` or using network reachability as part of Rootyard Doctor.
- Repairing malformed TOML or normalizing a whole TOML document.
- Automatically applying changes after a preset, rule, scan, or Codex update.

## 4. Confirmed Product Model

### 4.1 Unified presets and rules

One preset may contain both Git and Codex fields:

```json
{
  "id": "preset_company",
  "name": "Company",
  "git": {
    "user.email": { "mode": "set", "value": "dev@company.example" }
  },
  "codex": {
    "model": { "mode": "set", "value": "company-approved-model" },
    "sandbox_mode": { "mode": "set", "value": "workspace-write" }
  }
}
```

Global Defaults and Directory Rules use the same `git` and `codex` field maps. A rule references one
preset, then may override Git and Codex fields independently. Presets do not inherit from other
presets. Missing fields remain equivalent to `inherit`.

### 4.2 Managed fields

Phase 2 manages exactly these root-level TOML keys:

| Key | Rootyard control | Validation |
|-----|------------------|------------|
| `model` | Suggestions plus free-form text | Non-empty string when set |
| `model_reasoning_effort` | `low`, `medium`, `high`, `xhigh` suggestions plus free-form text | Non-empty string when set |
| `sandbox_mode` | Select | `read-only`, `workspace-write`, `danger-full-access` |
| `approval_policy` | Select | `untrusted`, `on-failure`, `on-request`, `granular`, `never` |
| `web_search` | Select | `disabled`, `cached`, `indexed`, `live` |

The enumerated values above were accepted by the local Codex configuration loader during design.
`web_search` is a string enum, not a boolean. `model` and `model_reasoning_effort` remain free-form
because model catalogs and accepted reasoning values can change independently of Rootyard releases.

Model suggestions are distinct values already present in Rootyard-managed presets, rules, or inspected
project files. They are labeled as recent values, not available models. Rootyard does not read or
modify global Codex configuration to populate suggestions.

### 4.3 Field intent

Codex fields use the Phase 1 intent semantics:

| Mode | Resolution | Native project apply |
|------|------------|----------------------|
| `inherit` | Leaves the lower layer unchanged | No operation if no lower layer manages the field |
| `set` | Replaces the effective value and source | Set or replace the root-level TOML assignment |
| `clear` | Replaces the effective intent and source | Remove the root-level TOML assignment |

`clear` affects only `.codex/config.toml`. It intentionally exposes whatever Codex would inherit from
global configuration or built-in defaults.

## 5. Architecture

The recommended architecture adds a focused adapter without replacing the existing domain model:

```text
environments.json
       |
       v
Resolver -> ResolvedEnvironment { Git, Codex }
       |
       v
Project Environment Service
       |
       +---- Git adapter ---- common-dir target ---- .git/config
       |
       +---- Codex adapter -- worktree target ------ .codex/config.toml
       |
       +---- offline CLI / Wails / Vue workspace
```

`internal/projectenv/codexadapter` implements the existing `projectenv.Adapter` contract. It owns
Codex target identity, TOML inspection, validation, planning, native edits, apply, and Doctor checks.
The resolver owns only layering and provenance. The service continues to orchestrate adapters without
knowing either Git syntax or TOML syntax.

Two rejected alternatives are:

1. A separate Codex preset and rule subsystem. It reduces initial coupling but duplicates resolution,
   discovery, confirmation, and state management and contradicts the unified-preset decision.
2. A dynamic `map[adapter]map[field]intent` refactor. It is more generic but makes persisted schema,
   generated Wails types, validation, and UI metadata less explicit. Phase 2 has only two known
   adapters, so the migration cost is not justified.

### 5.1 Domain model changes

`LayerConfig`, `Preset`, and `Rule` gain `Codex FieldMap`. `ResolvedEnvironment` gains a Codex resolved
field map. Request-scoped overrides become an explicit structure with `git` and `codex` maps rather
than leaving the current `session` field implicitly Git-only. Desktop calls and CLI parsing migrate to
the structured shape together.

Supported-field validation takes an adapter field definition instead of assuming every `FieldMap` is
Git. Unsupported keys remain rejected when saving `environments.json`.

### 5.2 Target identity

The Codex adapter canonicalizes the supplied repository path to its Git worktree root and uses that
root as `Target.ID` and `Target.WorktreePath`. `Target.ConfigPath` is
`<worktree>/.codex/config.toml`. `Target.CommonDir` is empty for Codex.

Unlike Git, Codex targets are never deduplicated by Git common directory. Two linked worktrees produce
one shared Git plan and two independent Codex plans. The existing service grouping key of
`adapter name + target ID` remains valid because the two adapters deliberately return different IDs.

Phase 2 continues to discover Git repositories only. Supporting Codex configuration for arbitrary
non-Git directories requires a separate project-target discovery model and is deferred.

## 6. TOML Inspection and Preservation

### 6.1 Inspection

Inspection treats a missing `.codex/config.toml` as a valid empty snapshot. For an existing file it:

1. Uses `Lstat` on `.codex` and `config.toml` and blocks symbolic links or non-regular targets.
2. Reads the original bytes without normalization.
3. Parses the full document with a standards-compliant TOML parser.
4. Rejects syntax errors, duplicate definitions, invalid managed value types, and unsupported enum
   values already present in managed fields.
5. Locates only root-level assignments for the five managed keys.
6. Records current managed values, source locations, file metadata, and a SHA-256 fingerprint.

Invalid TOML is a blocking issue. Rootyard never tries to recover a partially parsed document and
never invokes Codex as the primary validator. `codex app-server --strict-config` is not suitable for
this purpose because it also validates unrelated global configuration.

The fingerprint distinguishes a missing file from an empty file and includes the exact file bytes and
relevant target metadata. If a missing file appears after preview, apply is stale rather than treating
it as the previewed empty document.

### 6.2 Parser-guided text edits

Rootyard uses the TOML parser to identify syntax and byte spans, then performs minimal text edits:

- Replacing a managed value changes only its value token. Key spelling, spacing, and trailing comment
  remain unchanged.
- Clearing a managed assignment removes its statement. A trailing comment is retained as a standalone
  comment at the same location; preceding comments remain untouched.
- Adding a managed root key inserts a canonical quoted-string assignment immediately before the first
  table header. If there is no table, it appends at end of file with the existing newline convention.
- Multiple edits are applied from the highest byte offset to the lowest so earlier spans remain valid.
- Unmanaged byte ranges are copied unchanged.

The resulting plan contains the managed before/after values, exact text edits, and a zero-context
native diff of changed assignments. It does not serialize the whole configuration file to the
frontend, avoiding disclosure of unrelated project secrets. Apply rechecks the target and fingerprint,
then applies the stored edits to the same bytes. It does not recalculate a different desired document.

## 7. Planning and Apply

Planning produces one `ChangePlan` per adapter target. Codex operations use the same generic operation
names as Git: `unmanaged`, `unchanged`, `set`, `replace`, `clear`, and `blocked`. Plans identify their
adapter so the preview can group Git and Codex changes correctly.

Apply preserves the Phase 1 contract:

1. Re-identify the target and reject target substitution.
2. Acquire the target configuration lock.
3. Re-inspect the file and compare its fingerprint with the displayed plan.
4. Reject stale or blocked plans without writing.
5. Apply the exact stored text edits to the inspected bytes.
6. Parse and strictly validate the proposed bytes again.
7. Create `.codex` only at apply time if necessary.
8. Write a same-directory temporary file, preserve an existing file's mode, flush, atomically rename,
   and flush the directory.
9. Return the resulting fingerprint and per-target status.

For a new file, Rootyard creates `.codex` with normal private-user-safe directory permissions and
`config.toml` with mode `0600`, subject to the platform's permission model. A failure before rename
leaves the original file unchanged. One target failure does not prevent independent batch targets from
applying.

## 8. Risk and Doctor Behavior

The combination `sandbox_mode = "danger-full-access"` and `approval_policy = "never"` is allowed but
high risk. Resolver provenance does not change this assessment: the warning is based on the final
effective combination.

- Plan and Doctor emit a prominent non-blocking `dangerous_codex_permissions` issue.
- The desktop apply panel requires both the normal exact-change checkbox and a separate high-risk
  acknowledgement when any displayed plan contains this warning.
- Interactive CLI apply prints the warning and requires the explicit dangerous acknowledgement.
- Non-interactive `--yes` is insufficient; `--accept-dangerous` is also required.

Doctor remains offline. It validates saved intent, target paths, TOML syntax, managed value types,
write safety, and dangerous combinations. It checks for the Codex executable with a local path lookup
only. A missing executable is a non-blocking warning because a valid project configuration may be
prepared before Codex is installed. Doctor does not run `codex doctor`, authenticate, query models, or
perform network requests.

## 9. Derived State

Phase 1 stores one target result directly on each repository record, which is insufficient once one
repository has both Git and Codex targets. Phase 2 migrates derived state to adapter-specific entries:

```json
{
  "id": "/path/to/worktree",
  "aliases": [{ "path": "/path/to/worktree" }],
  "adapters": {
    "git": { "targetId": "/path/to/common-dir", "status": "consistent" },
    "codex": { "targetId": "/path/to/worktree", "status": "pending" }
  },
  "status": "pending"
}
```

The repository status is the most severe adapter status in this order: `failed`, `blocked`, `drifted`,
`pending`, `consistent`. Adapter fingerprints, last apply time, result, and blockers live in the
adapter entry. The state store migrates valid version 1 data into the Git adapter entry; corrupt state
remains rebuildable and quarantined under the existing rules.

Repository selection remains path-oriented. Previewing one selected repository asks every registered
adapter for a plan. Shared Git plans are deduplicated, while Codex plans remain worktree-specific.

## 10. CLI Contract

Persistent and request-scoped commands accept:

```text
--codex-set key=value
--codex-clear key
--codex-inherit key
```

These flags parallel `--git-set`, `--git-clear`, and `--git-inherit` for `defaults set`, `preset set`,
`rule set`, `resolve`, `diff`, `apply`, and `doctor`. Repeated operations for one key use the last
command-line occurrence, matching ordinary override semantics. Unsupported keys and invalid enum
values are usage errors before persistence or planning.

`diff` prints separate Git and Codex target sections. `apply` continues to apply the displayed plans,
requires `--yes` in non-interactive mode, and additionally requires `--accept-dangerous` for a high-
risk Codex plan. JSON output exposes adapter names, warnings, blockers, changes, and results without
including the full unmanaged TOML document.

## 11. Desktop Interaction

Preset and Directory Rule editors add `Git` and `Codex` sub-tabs inside their existing field area.
This does not change top-level application navigation or the stable Projects / Containers /
Environments positions.

Codex controls are type-appropriate:

- model and reasoning effort use editable suggestion controls;
- sandbox, approval, and web search use selects with all locally verified values;
- every row retains the `Not managed / Set / Clear` mode control;
- dangerous effective combinations show a warning near the fields and in preview.

The diff panel title becomes `Native environment changes`. Each target section shows its adapter,
worktree, configuration path, exact managed before/after values, source layer, warnings, and blockers.
The ordinary confirmation text no longer says Git. A second acknowledgement appears only for high-risk
Codex plans.

Model and reasoning suggestions do not imply availability. The UI accepts a non-empty free-form value
and explains availability only through neutral labeling, not feature-description text or a promise
that Rootyard can verify the account.

## 12. Error Handling

New typed issues distinguish at least:

- invalid Codex TOML;
- unsafe Codex configuration path or symbolic link;
- invalid managed Codex value;
- stale Codex plan;
- unavailable Codex executable;
- dangerous Codex permissions.

Invalid syntax, unsafe paths, invalid managed values, and stale plans block the affected target.
Missing Codex and dangerous permissions are warnings. Errors include the adapter, target path, managed
key where applicable, and a concise local detail. They must not include unrelated TOML values.

## 13. Testing and Verification

Backend tests cover:

- independent Git and Codex layering and provenance;
- field-key and enum validation;
- missing file inspection and explicit creation;
- root keys before and around tables;
- quoted strings, comments, CRLF/LF, and files without a final newline;
- byte preservation for unmanaged keys, tables, ordering, whitespace, and comments;
- clearing a managed field with a trailing comment;
- malformed TOML, duplicate keys, wrong types, and symbolic links;
- exact edit replay, stale fingerprints, file appearance races, atomic replacement, and permissions;
- linked worktrees producing one Git target and independent Codex targets;
- state migration and aggregate repository status;
- offline Doctor, missing executable warning, and dangerous-combination warning;
- CLI flags, JSON output, ordinary confirmation, and dangerous confirmation.

Frontend tests cover normalization, editor controls, recent model suggestions, adapter-grouped previews,
warning rendering, both confirmation gates, and stable top-level navigation. Browser verification covers
desktop and narrow viewports, overflow, text fitting, console errors, and that Git-only Phase 1 flows
remain usable.

Required completion checks include focused project-environment Go tests, race tests, CLI tests,
frontend unit tests, production build, and live browser checks against the development server.

## 14. Compatibility and Rollout

The persisted environment configuration remains version 1 because adding optional `codex` maps is a
backward-compatible schema extension for this application. Strict decoding still rejects unknown
fields. Derived state advances to version 2 because its target-result shape changes; version 1 state is
migrated locally and can always be rebuilt.

Existing presets and rules load with empty Codex maps. Existing Git behavior and native files are not
changed until a user explicitly configures Codex intent, previews it, and confirms apply. A plan with
all Codex fields inherited is unchanged and does not create `.codex/config.toml`.

The Codex value lists are compatibility observations for `codex-cli 0.170.0`, not a permanent promise
about future Codex releases. Free-form model and reasoning inputs and adapter-scoped validation keep a
future update narrow when official documentation or local behavior changes.
