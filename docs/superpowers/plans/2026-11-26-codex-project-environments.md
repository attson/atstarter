# Codex Project Environments Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend Rootyard Project Environments so unified presets and rules can safely preview and apply five project-level Codex settings in each Git worktree's `.codex/config.toml`.

**Architecture:** Add a Codex field set to the existing resolver and a focused `projectenv.Adapter` implementation whose target identity is the canonical worktree root. Parse complete TOML for safety, retain source bytes for planning, serialize only minimal managed text edits, and replay those edits after fingerprint validation during atomic apply. Keep repository discovery shared while derived state records independent Git and Codex adapter outcomes.

**Tech Stack:** Go 1.23, `github.com/pelletier/go-toml/v2`, Wails v2 generated bindings, Vue 3, Node test runner, Vite.

**Spec:** `docs/superpowers/specs/2026-11-26-codex-project-environments-design.md`

## Global Constraints

- Manage only project-level `<worktree>/.codex/config.toml`; never write global Codex configuration or profile files.
- Manage exactly `model`, `model_reasoning_effort`, `sandbox_mode`, `approval_policy`, and `web_search`.
- Preserve unmanaged TOML keys, tables, ordering, whitespace, and comments; malformed or unsafe TOML blocks apply.
- A missing project config may be created only through displayed-plan apply.
- `clear` removes the root-level project key and exposes Codex global or built-in fallback behavior.
- Git targets remain deduplicated by common directory; Codex targets remain independent per linked worktree.
- `model` and `model_reasoning_effort` accept non-empty free-form strings; the other fields use the exact enums in the spec.
- Doctor performs no network requests and does not invoke `codex doctor`.
- `danger-full-access` plus `never` is a non-blocking high-risk warning with an additional explicit confirmation gate.
- No automatic native writes; apply must replay the displayed plan after fingerprint and target validation.
- Keep top-level Projects / Containers / Environments navigation order and geometry stable.
- Use `/Users/attson/sdk/go1.23.12/bin/go` when `go` is absent from `PATH`.

---

### Task 1: Add Codex Fields to the Persisted and Resolved Domain Model

**Files:**
- Modify: `internal/projectenv/model.go`
- Modify: `internal/projectenv/validate.go`
- Modify: `internal/projectenv/validate_test.go`
- Modify: `internal/projectenv/resolver.go`
- Modify: `internal/projectenv/resolver_test.go`
- Modify: `internal/projectenv/service.go`
- Modify: `internal/projectenv/service_test.go`

**Interfaces:**
- Produces: `SupportedCodexKeys []string` and `EnvironmentFields { Git FieldMap; Codex FieldMap }`.
- Produces: `LayerConfig.Codex`, `Preset.Codex`, `Rule.Codex`, and `ResolvedEnvironment.Codex`.
- Produces: request `Session EnvironmentFields` and save-request `Codex FieldMap` properties.
- Preserves: persisted environment configuration version 1 and existing Git resolution semantics.

- [x] **Step 1: Write failing validation and resolution tests**

Add table tests that prove all five keys are accepted, unsupported keys are rejected, fixed enums are strict, and model/reasoning values are non-empty but otherwise free-form. Add a layered resolution test with a preset Codex model, rule sandbox override, and session approval override:

```go
func TestResolveLayersGitAndCodexIndependently(t *testing.T) {
    cfg := Config{
        Version: 1,
        Presets: []Preset{{
            ID: "preset_company", Name: "Company",
            Git: FieldMap{"user.email": {Mode: ModeSet, Value: "dev@example.com"}},
            Codex: FieldMap{"model": {Mode: ModeSet, Value: "company-model"}},
        }},
        Rules: []Rule{{
            ID: "rule_company", Name: "Company", Path: t.TempDir(), Recursive: true,
            PresetID: "preset_company",
            Codex: FieldMap{"sandbox_mode": {Mode: ModeSet, Value: "workspace-write"}},
        }},
    }
    resolved, err := Resolve(cfg, cfg.Rules[0].Path, EnvironmentFields{Codex: FieldMap{
        "approval_policy": {Mode: ModeSet, Value: "on-request"},
    }})
    if err != nil { t.Fatal(err) }
    if resolved.Codex["model"].Source.Kind != SourcePreset { t.Fatalf("model = %+v", resolved.Codex["model"]) }
    if resolved.Codex["sandbox_mode"].Source.Kind != SourceRule { t.Fatalf("sandbox = %+v", resolved.Codex["sandbox_mode"]) }
    if resolved.Codex["approval_policy"].Source.Kind != SourceSession { t.Fatalf("approval = %+v", resolved.Codex["approval_policy"]) }
}
```

- [x] **Step 2: Run focused tests and confirm the compile failure**

Run:

```bash
/Users/attson/sdk/go1.23.12/bin/go test ./internal/projectenv -run 'Test(Validate|Resolve)' -count=1
```

Expected: FAIL because Codex fields and `EnvironmentFields` do not exist.

- [x] **Step 3: Implement explicit field sets and validation**

Add:

```go
var SupportedCodexKeys = []string{
    "model", "model_reasoning_effort", "sandbox_mode", "approval_policy", "web_search",
}

type EnvironmentFields struct {
    Git   FieldMap `json:"git,omitempty"`
    Codex FieldMap `json:"codex,omitempty"`
}
```

Add `Codex FieldMap` to each persisted layer and `map[string]ResolvedField` to the resolved result. Split validation into `validateGitFields` and `validateCodexFields`; validate the exact enums from the spec and require a trimmed non-empty value for every `set`. Update save methods, clone behavior, and merge helpers so Git and Codex maps are copied independently. Update `Resolve` to apply both maps through the same source layer in deterministic order.

- [x] **Step 4: Run domain tests**

Run:

```bash
/Users/attson/sdk/go1.23.12/bin/go test ./internal/projectenv -run 'Test(Validate|Resolve|Save|Clone)' -count=1
```

Expected: PASS.

- [x] **Step 5: Commit the domain extension**

```bash
git add internal/projectenv/model.go internal/projectenv/validate.go internal/projectenv/validate_test.go internal/projectenv/resolver.go internal/projectenv/resolver_test.go internal/projectenv/service.go internal/projectenv/service_test.go
git commit -m "feat: resolve Codex environment fields"
```

---

### Task 2: Migrate Derived State to Per-Adapter Results

**Files:**
- Modify: `internal/projectenv/state.go`
- Modify: `internal/projectenv/state_test.go`
- Modify: `internal/projectenv/service.go`
- Modify: `internal/projectenv/service_test.go`

**Interfaces:**
- Consumes: adapter names and targets already present on plans/results.
- Produces: `AdapterState` and `RepositoryRecord.Adapters map[string]AdapterState`.
- Produces: state version 2 with automatic version 1 migration.
- Preserves: repository aliases, rule matches, scan timestamps, and rebuildable-state quarantine behavior.

- [x] **Step 1: Write failing migration and aggregation tests**

Cover a version 1 JSON document whose legacy target fields migrate into `Adapters["git"]`, a new repository with both adapter states, and severity aggregation:

```go
func TestStateStoreMigratesVersionOneRepositoryTargetToGitAdapter(t *testing.T) {
    path := filepath.Join(t.TempDir(), "config.json")
    paths := PathsForConfig(path)
    if err := os.MkdirAll(filepath.Dir(paths.State), 0o755); err != nil { t.Fatal(err) }
    legacy := `{"version":1,"ruleScans":[],"repositories":[{"id":"/repo","targetId":"/common","aliases":[{"path":"/repo"}],"status":"consistent","fingerprint":"abc"}]}`
    if err := os.WriteFile(paths.State, []byte(legacy), 0o600); err != nil { t.Fatal(err) }
    state, err := NewStateStore(path).Load(context.Background())
    if err != nil { t.Fatal(err) }
    if state.Version != 2 || state.Repositories[0].Adapters["git"].TargetID != "/common" {
        t.Fatalf("state = %+v", state)
    }
}
```

- [x] **Step 2: Run state tests and verify failure**

```bash
/Users/attson/sdk/go1.23.12/bin/go test ./internal/projectenv -run 'Test(State|PersistApply)' -count=1
```

Expected: FAIL because state version 2 and adapter states are absent.

- [x] **Step 3: Implement migration and aggregate status**

Define:

```go
type AdapterState struct {
    TargetID        string  `json:"targetId,omitempty"`
    Fingerprint     string  `json:"fingerprint,omitempty"`
    Status          string  `json:"status,omitempty"`
    LastApplyAt     string  `json:"lastApplyAt,omitempty"`
    LastApplyResult string  `json:"lastApplyResult,omitempty"`
    Blockers        []Issue `json:"blockers,omitempty"`
}
```

Load raw version metadata first. Decode version 1 into a private legacy struct, migrate its target fields to `git`, and save version 2 only on the next state mutation. Normalize nil adapter maps. Update `persistApplyResults` to locate records by plan aliases, write `record.Adapters[plan.Adapter]`, and recompute repository status using `failed > blocked > drifted > pending > consistent`.

- [x] **Step 4: Run state and service tests**

```bash
/Users/attson/sdk/go1.23.12/bin/go test ./internal/projectenv -run 'Test(State|PersistApply|Service)' -count=1
```

Expected: PASS with both legacy migration and independent adapter results.

- [x] **Step 5: Commit state version 2**

```bash
git add internal/projectenv/state.go internal/projectenv/state_test.go internal/projectenv/service.go internal/projectenv/service_test.go
git commit -m "feat: track environment state per adapter"
```

---

### Task 3: Parse and Inspect Project Codex TOML Without Reformatting

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`
- Modify: `internal/projectenv/adapter.go`
- Modify: `internal/projectenv/errors.go`
- Create: `internal/projectenv/codexadapter/adapter.go`
- Create: `internal/projectenv/codexadapter/identify.go`
- Create: `internal/projectenv/codexadapter/toml.go`
- Create: `internal/projectenv/codexadapter/inspect.go`
- Create: `internal/projectenv/codexadapter/test_helpers_test.go`
- Create: `internal/projectenv/codexadapter/identify_test.go`
- Create: `internal/projectenv/codexadapter/inspect_test.go`

**Interfaces:**
- Produces: `codexadapter.New() *Adapter`, `Name() string`, `Identify`, and `Inspect`.
- Produces: `TextEdit { Start, End int; Replacement string }` on plans and private parsed assignment spans.
- Extends: `Snapshot.NativeContent []byte` with `json:"-"` for same-process planning only.
- Uses: `github.com/pelletier/go-toml/v2` for complete-document syntax and type validation.

- [x] **Step 1: Add the TOML dependency**

Run:

```bash
/Users/attson/sdk/go1.23.12/bin/go get github.com/pelletier/go-toml/v2@v2.2.2
```

Expected: `go.mod` and `go.sum` record the pinned parser.

- [x] **Step 2: Write failing identify and inspection tests**

Tests must initialize a normal repository and linked worktree, then prove Codex IDs are distinct worktree roots while Git common directories are irrelevant. Inspection cases must cover missing file, root values, a table after root values, wrong managed type, malformed TOML, duplicate managed key, `.codex` symlink, and config symlink.

Use byte-preservation fixtures such as:

```go
const preservedFixture = `# keep heading
model   = "old-model" # keep inline
custom = { token = "secret" }

[features]
multi_agent = true # keep table comment
`
```

Assert that the snapshot reports only managed values, retains the exact fixture in `NativeContent`, and emits `ErrInvalidCodexConfig` or `ErrUnsafeCodexConfigPath` as blocking issues without exposing `secret` in the issue detail.

- [x] **Step 3: Run adapter tests and confirm failure**

```bash
/Users/attson/sdk/go1.23.12/bin/go test ./internal/projectenv/codexadapter -run 'Test(Identify|Inspect)' -count=1
```

Expected: FAIL because the package does not exist.

- [x] **Step 4: Implement identity, safe path checks, and parser-backed inspection**

The default adapter runs bounded `git rev-parse --show-toplevel`, canonicalizes the returned root, and returns:

```go
projectenv.Target{
    Adapter: "codex", ID: worktree, WorktreePath: worktree,
    ConfigPath: filepath.Join(worktree, ".codex", "config.toml"),
}
```

Implement a private parsed representation:

```go
type assignment struct {
    Key string
    Value string
    StatementStart int
    ValueStart int
    ValueEnd int
    StatementEnd int
    TrailingComment string
}

type document struct {
    Data []byte
    Newline string
    FirstTable int
    Assignments map[string]assignment
}
```

Use `go-toml/v2` to reject invalid full documents and decode root managed values as strings. Use its parser tokens/positions, rather than regex alone, to find top-level assignment/value spans. Treat missing files as a distinct fingerprint state. Check each path component with `Lstat` before reading and block symlinks or non-regular configuration files.

- [x] **Step 5: Run inspect tests**

```bash
/Users/attson/sdk/go1.23.12/bin/go test ./internal/projectenv/codexadapter -run 'Test(Identify|Inspect)' -count=1
```

Expected: PASS.

- [x] **Step 6: Commit safe Codex inspection**

```bash
git add go.mod go.sum internal/projectenv/adapter.go internal/projectenv/errors.go internal/projectenv/codexadapter
git commit -m "feat: inspect project Codex configuration"
```

---

### Task 4: Plan Minimal TOML Edits and High-Risk Warnings

**Files:**
- Create: `internal/projectenv/codexadapter/edit.go`
- Create: `internal/projectenv/codexadapter/edit_test.go`
- Create: `internal/projectenv/codexadapter/plan.go`
- Create: `internal/projectenv/codexadapter/plan_test.go`
- Modify: `internal/projectenv/adapter.go`

**Interfaces:**
- Produces: `ChangePlan.NativeEdits []TextEdit` and `ChangePlan.NativeDiff []string`.
- Produces: `applyTextEdits(original []byte, edits []TextEdit) ([]byte, error)`.
- Produces: non-blocking `ErrDangerousCodexPermissions` warning.
- Consumes: `ResolvedEnvironment.Codex` and `Snapshot.NativeContent` from Tasks 1 and 3.

- [x] **Step 1: Write failing edit-planning tests**

Cover these exact transformations:

```text
replace: model   = "old" # keep -> model   = "new" # keep
clear:   web_search = "live" # rationale -> # rationale
insert:  before the first [features] table
append:  preserve CRLF and missing-final-newline policy
escape:  quote backslash, quote, tab, and newline in free-form strings
```

Assert that bytes outside the returned edit ranges are identical. Add plan tests for `set`, `replace`, `clear`, `unchanged`, all-inherit with a missing file, and final effective `danger-full-access + never` warning.

- [x] **Step 2: Run plan tests and verify failure**

```bash
/Users/attson/sdk/go1.23.12/bin/go test ./internal/projectenv/codexadapter -run 'Test(Edit|Plan)' -count=1
```

Expected: FAIL because edit planning is absent.

- [x] **Step 3: Implement deterministic minimal edits**

Add serializable edits:

```go
type TextEdit struct {
    Start       int    `json:"start"`
    End         int    `json:"end"`
    Replacement string `json:"replacement"`
}
```

Encode every managed value as a TOML basic string. Replace value tokens only. For clear, replace the whole statement with its trailing comment and original newline. Insert missing assignments before the first table, or at EOF, using the detected newline convention. Sort edits by descending `Start`, reject overlap/out-of-bounds edits, and generate a zero-context managed diff without unrelated TOML values.

`Plan` must fill all five managed changes, desired intents, edits, diff, warnings, blockers, and the original fingerprint. When every effective field is inherited, it must return no edits and must not plan file creation.

- [x] **Step 4: Run edit and plan tests**

```bash
/Users/attson/sdk/go1.23.12/bin/go test ./internal/projectenv/codexadapter -run 'Test(Edit|Plan)' -count=1
```

Expected: PASS, including exact byte assertions.

- [x] **Step 5: Commit Codex planning**

```bash
git add internal/projectenv/adapter.go internal/projectenv/codexadapter/edit.go internal/projectenv/codexadapter/edit_test.go internal/projectenv/codexadapter/plan.go internal/projectenv/codexadapter/plan_test.go
git commit -m "feat: plan minimal Codex configuration edits"
```

---

### Task 5: Apply Codex Plans Atomically and Diagnose Offline

**Files:**
- Create: `internal/projectenv/codexadapter/apply.go`
- Create: `internal/projectenv/codexadapter/apply_replace_unix.go`
- Create: `internal/projectenv/codexadapter/apply_replace_windows.go`
- Create: `internal/projectenv/codexadapter/apply_test.go`
- Create: `internal/projectenv/codexadapter/doctor.go`
- Create: `internal/projectenv/codexadapter/doctor_test.go`

**Interfaces:**
- Produces: `Adapter.Apply(context.Context, projectenv.ChangePlan) projectenv.ApplyResult`.
- Produces: `Adapter.Doctor(context.Context, projectenv.Target, projectenv.ResolvedEnvironment) []projectenv.Issue`.
- Consumes: exact serialized edits and fingerprint from Task 4.

- [x] **Step 1: Write failing atomic apply tests**

Cover creation, replacement, clear, no-op, target substitution, stale existing file, file appearing after a missing-file preview, invalid stored edit bounds, invalid rendered TOML, preservation of existing mode, temporary-write failure, and linked-worktree independence. For each failure before rename, assert original bytes remain unchanged.

Add Doctor tests with an injected executable lookup function so they deterministically prove:

- missing `codex` is `ErrCodexUnavailable` and non-blocking;
- invalid project TOML is blocking;
- dangerous final permissions warn without blocking;
- Doctor never runs a subprocess or network request.

- [x] **Step 2: Run apply and Doctor tests and verify failure**

```bash
/Users/attson/sdk/go1.23.12/bin/go test ./internal/projectenv/codexadapter -run 'Test(Apply|Doctor)' -count=1
```

Expected: FAIL because `Apply` and `Doctor` are absent.

- [x] **Step 3: Implement stale-safe atomic apply**

Apply must:

```text
re-identify -> compare target -> acquire <config>.rootyard.lock
-> inspect -> compare fingerprint -> replay stored edits
-> parse/validate rendered TOML -> create .codex if required
-> same-directory temp -> chmod -> write -> fsync -> close
-> atomic replace -> directory fsync -> return fingerprint
```

Use mode `0600` for a new file and preserve `Perm()` for an existing file. Create `.codex` only after all pre-write validation passes. Reject plans with blockers, unsafe paths, changed targets, or stale fingerprints. Ensure the absent-file fingerprint cannot equal an empty existing file.

- [x] **Step 4: Implement offline Doctor**

Use injected `exec.LookPath` only. Reuse inspection and final resolved-field validation. Return typed issues with paths and managed keys but no unrelated TOML values. Emit the dangerous warning based on final resolved values even when those values came from different layers.

- [x] **Step 5: Run the complete adapter suite with race detection**

```bash
/Users/attson/sdk/go1.23.12/bin/go test -race ./internal/projectenv/codexadapter -count=1
```

Expected: PASS.

- [x] **Step 6: Commit apply and Doctor**

```bash
git add internal/projectenv/codexadapter
git commit -m "feat: apply Codex environments atomically"
```

---

### Task 6: Wire Codex Through Service, Desktop, and Offline CLI

**Files:**
- Modify: `app.go`
- Modify: `env_cli.go`
- Modify: `env_cli_test.go`
- Modify: `internal/projectenv/service.go`
- Modify: `internal/projectenv/service_test.go`
- Modify generated: `frontend/wailsjs/go/models.ts`
- Modify generated: `frontend/wailsjs/go/main/App.js`
- Modify generated: `frontend/wailsjs/go/main/App.d.ts`

**Interfaces:**
- Consumes: `codexadapter.New()` and structured session fields.
- Produces: `--codex-set`, `--codex-clear`, `--codex-inherit`, and `--accept-dangerous`.
- Preserves: existing Git flags and offline `rootyard env` dispatch.

- [ ] **Step 1: Write failing service and CLI tests**

Extend the service test adapter matrix to assert one Git/common target and two Codex/worktree targets for linked worktrees. Add CLI tests that persist Codex preset/rule fields, resolve session overrides, render adapter labels in diff, apply a normal Codex plan with `--yes`, and reject dangerous non-interactive apply unless both flags are present:

```go
code, output, _ := runEnvironmentCLIForTest(t, configPath,
    "--json", "apply", repo,
    "--codex-set", "sandbox_mode=danger-full-access",
    "--codex-set", "approval_policy=never",
    "--yes")
if code == 0 || !strings.Contains(output, "dangerous_confirmation_required") {
    t.Fatalf("dangerous apply = %d %s", code, output)
}
```

- [ ] **Step 2: Run service and CLI tests and verify failure**

```bash
/Users/attson/sdk/go1.23.12/bin/go test ./internal/projectenv ./... -run 'Test(Service|RunEnvCLI)' -count=1
```

Expected: FAIL because the service is not wired and the CLI does not parse Codex flags.

- [ ] **Step 3: Register both adapters**

Construct services in `app.go` and `env_cli.go` with:

```go
projectenv.NewService(configPath, gitadapter.New(), codexadapter.New())
```

Keep service grouping by `adapter.Name() + target.ID`. Ensure planning and Doctor append adapter-specific errors rather than labeling every inspection error as invalid Git config.

- [ ] **Step 4: Implement structured field flags and confirmation**

Replace `parseEnvironmentFields` with a parser returning `projectenv.EnvironmentFields`. Route `--git-*` and `--codex-*` into their corresponding maps and validate both. Pass the structured session through resolve, plan, apply, and Doctor. Parse `--accept-dangerous` only for apply. In interactive mode require a distinct dangerous acknowledgement after the ordinary plan confirmation; in non-interactive/JSON mode require both `--yes` and `--accept-dangerous`.

Render plan adapter, worktree, and config path before changes. Never print full TOML or serialized native edits in human output.

- [ ] **Step 5: Regenerate Wails bindings**

Run:

```bash
GO=/Users/attson/sdk/go1.23.12/bin/go
$($GO env GOPATH)/bin/wails generate module
```

Expected: generated models include Codex maps, structured sessions, text edits, adapter states, and state version 2.

- [ ] **Step 6: Run backend and CLI tests**

```bash
/Users/attson/sdk/go1.23.12/bin/go test ./internal/projectenv/... ./... -count=1
```

Expected: PASS.

- [ ] **Step 7: Commit integration**

```bash
git add app.go env_cli.go env_cli_test.go internal/projectenv frontend/wailsjs
git commit -m "feat: expose Codex project environments"
```

---

### Task 7: Add Codex Controls to Unified Presets and Rules

**Files:**
- Modify: `frontend/src/projectEnvironmentUi.js`
- Modify: `frontend/src/projectEnvironmentUi.test.mjs`
- Create: `frontend/src/components/environments/CodexEnvironmentFieldEditor.vue`
- Modify: `frontend/src/components/environments/EnvironmentPresetManager.vue`
- Modify: `frontend/src/components/environments/EnvironmentRuleManager.vue`
- Modify: `frontend/src/components/environments/EnvironmentWorkspace.vue`

**Interfaces:**
- Produces: normalized `codex` maps for defaults, presets, rules, and save requests.
- Produces: `environmentCodexKeys`, `codexFieldRows`, `codexModelSuggestions`, and `dangerousCodexCombination` helpers.
- Preserves: existing `EnvironmentFieldEditor` as the Git editor.

- [ ] **Step 1: Write failing frontend model tests**

Add tests proving normalization retains Codex fields, recent model suggestions are deduplicated from managed documents/plans, free-form reasoning survives edits, fixed enums reject invalid draft values, and dangerous combinations are detected only from the effective pair.

```js
test('normalizes and validates Codex fields independently from Git', () => {
  const document = normalizeEnvironmentDocument({ Config: { Version: 1, Presets: [{
    ID: 'p', Name: 'Company', Codex: { model: { mode: 'set', value: 'custom-model' } },
  }] } })
  assert.equal(document.config.presets[0].codex.model.value, 'custom-model')
  assert.deepEqual(codexModelSuggestions(document), ['custom-model'])
})
```

- [ ] **Step 2: Run frontend tests and verify failure**

```bash
cd frontend && node --test src/projectEnvironmentUi.test.mjs src/environmentWorkspaceUi.test.mjs
```

Expected: FAIL because Codex helpers and controls do not exist.

- [ ] **Step 3: Implement frontend normalization and validation**

Add explicit Codex key metadata with labels, descriptions, suggestion values, and fixed option lists. Normalize and save `codex` alongside `git`. Keep model/reasoning validation to trimmed non-empty values; validate exact sandbox, approval, and search enums. Derive recent model suggestions only from Rootyard documents and current inspected plan values.

- [ ] **Step 4: Build the Codex field editor and unified tabs**

Create a row editor that keeps the existing three intent modes and uses:

- editable `input` plus `datalist` for model and reasoning;
- select controls for sandbox, approval, and web search;
- inline high-risk warning when the current draft sets the dangerous pair.

Add compact `Git` and `Codex` tabs to preset/default/rule field sections. Clone and save both maps. Use stable grid tracks and existing radii/colors; do not alter top-level navigation.

- [ ] **Step 5: Run frontend tests and build**

```bash
cd frontend
node --test src/projectEnvironmentUi.test.mjs src/environmentWorkspaceUi.test.mjs
npm run build
```

Expected: all tests PASS and Vite build succeeds.

- [ ] **Step 6: Commit unified editors**

```bash
git add frontend/src/projectEnvironmentUi.js frontend/src/projectEnvironmentUi.test.mjs frontend/src/components/environments
git commit -m "feat: edit Codex environment presets and rules"
```

---

### Task 8: Show Adapter-Aware Plans, State, and Dangerous Confirmation

**Files:**
- Modify: `frontend/src/projectEnvironmentUi.js`
- Modify: `frontend/src/projectEnvironmentUi.test.mjs`
- Modify: `frontend/src/environmentWorkspaceUi.test.mjs`
- Modify: `frontend/src/components/environments/EnvironmentRepositoryManager.vue`
- Modify: `frontend/src/components/environments/EnvironmentDiffPanel.vue`
- Modify: `frontend/src/components/environments/EnvironmentOverview.vue`
- Modify: `frontend/src/components/environments/EnvironmentWorkspace.vue`

**Interfaces:**
- Consumes: adapter state maps, plan adapter names, native diff lines, and dangerous warning codes.
- Produces: aggregate repository status and two-stage confirmation UI.
- Preserves: repository path-oriented selection and stale re-preview.

- [ ] **Step 1: Write failing UI behavior tests**

Update the source-level UI test and helper tests to require:

- generic `Native environment changes` wording;
- visible Git/Codex adapter labels and config paths;
- state status derived from `repository.adapters`;
- ordinary exact-change checkbox for every apply;
- second checkbox only when `dangerous_codex_permissions` is present;
- apply disabled until every required confirmation is checked;
- unchanged top-level navigation order and stable narrow layout.

- [ ] **Step 2: Run UI tests and verify failure**

```bash
cd frontend && node --test src/projectEnvironmentUi.test.mjs src/environmentWorkspaceUi.test.mjs
```

Expected: FAIL on Git-only labels and missing high-risk acknowledgement.

- [ ] **Step 3: Implement adapter-aware state and preview helpers**

Normalize version 2 adapter states while retaining read compatibility for an in-memory version 1 state returned during migration. Compute repository status from the worst adapter outcome. Count changes by plan and adapter. Add helpers that identify high-risk plans by issue code without inferring risk from display strings.

- [ ] **Step 4: Implement preview and confirmation UI**

Group displayed plans by worktree, with separate Git and Codex subsections. Show adapter, configuration path, key-level before/operation/after/source, and `nativeDiff` lines for Codex. Change generic confirmation copy to `I reviewed these exact native environment changes`. Show a warning band and `I understand this disables both sandboxing and approvals` checkbox only for high-risk plans.

Do not add a card inside the diff panel. Keep responsive panel dimensions stable and prevent long paths, model IDs, or warning text from widening the workspace.

- [ ] **Step 5: Run frontend tests and production build**

```bash
cd frontend
node --test src/*.test.mjs
npm run build
```

Expected: all frontend tests PASS and build succeeds.

- [ ] **Step 6: Commit adapter-aware preview**

```bash
git add frontend/src/projectEnvironmentUi.js frontend/src/projectEnvironmentUi.test.mjs frontend/src/environmentWorkspaceUi.test.mjs frontend/src/components/environments
git commit -m "feat: preview Codex environment changes safely"
```

---

### Task 9: Update Long-Lived Contracts and User Documentation

**Files:**
- Modify: `docs/specs/DOMAIN_MODEL.md`
- Modify: `docs/specs/ARCHITECTURE.md`
- Modify: `docs/specs/RUNTIME_CONTRACTS.md`
- Modify: `docs/specs/FRONTEND_STYLE.md`
- Modify: `README.md`

**Interfaces:**
- Documents: unified schema, state version 2, Codex target identity, TOML preservation, CLI flags, offline Doctor, and dangerous confirmation.
- Preserves: Phase 1 Git guarantees and the offline CLI exception.

- [ ] **Step 1: Update authoritative contracts**

Document the exact five Codex keys and values, structured session overrides, `TextEdit`/native diff plan data, worktree target identity, per-adapter state, state migration, stale absent-file behavior, and issue severity. Replace any statement that implies `projectenv` has only a Git adapter.

- [ ] **Step 2: Update UI and CLI documentation**

Document the Git/Codex sub-tabs, adapter-grouped preview, two confirmations for high-risk plans, and these commands:

```bash
rootyard env preset set --name Company --codex-set model=custom-model
rootyard env diff /path/to/repo --codex-set sandbox_mode=workspace-write
rootyard env apply /path/to/repo --codex-clear model --yes
rootyard env apply /path/to/repo --codex-set sandbox_mode=danger-full-access --codex-set approval_policy=never --yes --accept-dangerous
```

State explicitly that Rootyard does not modify global Codex config or verify account/model availability.

- [ ] **Step 3: Check docs for stale Git-only claims**

Run:

```bash
rg -n "Git adapter|Native Git changes|Session overrides|environment-state|projectenv.Adapter" README.md docs/specs
```

Expected: every remaining Git-only phrase is intentionally scoped to the Git adapter, not the whole subsystem.

- [ ] **Step 4: Commit documentation**

```bash
git add README.md docs/specs
git commit -m "docs: document Codex project environments"
```

---

### Task 10: Complete Regression, Race, and Browser Verification

**Files:**
- Modify only if verification exposes a defect: files already listed in Tasks 1-9.

**Interfaces:**
- Verifies: every design requirement and Phase 1 regression constraint.
- Produces: a clean worktree with passing automated and live UI checks.

- [ ] **Step 1: Run formatting and generated-file checks**

```bash
/Users/attson/sdk/go1.23.12/bin/gofmt -w internal/projectenv/*.go internal/projectenv/gitadapter/*.go internal/projectenv/codexadapter/*.go app.go app_environments.go env_cli.go env_cli_test.go
git diff --check
git status --short
```

Expected: no formatting errors or unexpected generated artifacts.

- [ ] **Step 2: Run all Go tests**

```bash
/Users/attson/sdk/go1.23.12/bin/go test ./... -count=1
```

Expected: PASS.

- [ ] **Step 3: Run project-environment race tests**

```bash
/Users/attson/sdk/go1.23.12/bin/go test -race ./internal/projectenv/... -count=1
```

Expected: PASS with no race reports.

- [ ] **Step 4: Run all frontend tests and build**

```bash
cd frontend
node --test src/*.test.mjs
npm run build
```

Expected: all tests PASS and Vite build succeeds.

- [ ] **Step 5: Run focused manual CLI smoke tests in a temporary repository**

Create a temporary Git repository and use an isolated `--config` path. Verify `diff` does not create `.codex`, normal `apply --yes` creates only project `.codex/config.toml`, `clear` removes only the managed assignment, malformed TOML blocks without byte changes, stale apply refuses, and dangerous apply requires `--accept-dangerous`. Remove the temporary directory after evidence is recorded.

- [ ] **Step 6: Start the development server and inspect the live UI**

Run the existing development command on an unused port. Use AIPageGo against the live localhost page at desktop and narrow viewport sizes. Verify:

- Projects / Containers / Environments stay in the same order and x-position across tabs;
- preset and rule Git/Codex tabs work;
- all five Codex controls fit without overlap;
- long model IDs and paths wrap or truncate intentionally;
- adapter plans and both confirmation gates render correctly;
- no horizontal page scroll, console errors, or failed assets occur.

- [ ] **Step 7: Record final evidence**

Run:

```bash
git status --short
git log --oneline --max-count=12
```

Expected: clean worktree and a reviewable sequence of focused Phase 2 commits.
