# Project Environments Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver Rootyard's complete Phase 1 project-environment workflow: centralized presets and directory rules, Git repository discovery, safe Git identity/signing previews and manual application, an offline CLI, and the desktop management UI.

**Architecture:** Add a Wails-independent `internal/projectenv` core for persistence, resolution, discovery, state, and orchestration. Implement Git-native behavior behind a `projectenv.Adapter` in `internal/projectenv/gitadapter`; compose the same service into the top-level `rootyard env` CLI and Wails bindings, then build a Vue workspace over those bindings.

**Tech Stack:** Go 1.23 module using the system `git` executable and existing `golang.org/x/sys`; Wails v2 bindings/events; Vue 3 Composition API; Vite 3; existing CSS tokens and `lucide-vue-next`; Go `testing` and Node `node:test`.

**Spec:** [`docs/superpowers/specs/2026-11-26-project-environments-design.md`](../specs/2026-11-26-project-environments-design.md) - read before starting.

## Global Constraints

- Phase 1 manages exactly `user.name`, `user.email`, `user.signingKey`, `commit.gpgSign`, `tag.gpgSign`, and `gpg.format`.
- Persist field intent as `inherit`, `set(value)`, or `clear`; never infer deletion from a missing field.
- Persist authoritative data in sibling `environments.json`; keep `environment-state.json` rebuildable and non-authoritative.
- Never write Git configuration during discovery, resolution, preset/rule save, or diff. Only an explicit apply writes.
- Git operations are local-only and bounded by timeouts. Do not add fetch, pull, push, merge, commit, or stash behavior.
- `rootyard env` works without the desktop process. Existing `rootyard cli` and MCP runtime operations continue through the desktop control server.
- The existing launcher `config.json` schema remains unchanged.
- No third-party UI component library. Use `components/ui/*`, existing theme variables, and `lucide-vue-next`.
- Wails frontend code calls generated bindings only; it does not read JSON, walk directories, or invoke Git.
- Regenerate Wails bindings after adding exported `App` methods.
- Local Go commands use `GO=/home/attson/sdk/go1.24.13/bin/go`.
- On Ubuntu 24.04 Wails dev/build uses `-tags webkit2_41` through the existing Makefile targets.
- Commit after every task using the exact commit message shown.

## File Structure

### New Go core files

- `internal/projectenv/model.go` - persisted config types, runtime plan/result types, status constants.
- `internal/projectenv/errors.go` - stable `Error` and `Issue` codes shared by CLI and Wails.
- `internal/projectenv/paths.go` - platform-aware canonical paths, ancestry, and environment file paths.
- `internal/projectenv/validate.go` - strict schema, field, preset-reference, and duplicate-rule validation.
- `internal/projectenv/resolver.go` - defaults -> outer rules -> inner/exact rule -> session merge with provenance.
- `internal/projectenv/filelock.go`, `filelock_unix.go`, `filelock_windows.go` - context-bounded cross-process locks.
- `internal/projectenv/store.go` - strict JSON load plus revision-checked atomic config mutation.
- `internal/projectenv/state.go` - rebuildable state load/quarantine/atomic save.
- `internal/projectenv/discovery.go` - cancellable recursive Git candidate discovery.
- `internal/projectenv/adapter.go` - adapter interface and generic target/snapshot/change-plan contracts.
- `internal/projectenv/service.go` - preset/rule CRUD, scan, resolve, plan, batch apply, and doctor orchestration.
- Corresponding `*_test.go` files beside each responsibility.

### New Git adapter files

- `internal/projectenv/gitadapter/runner.go` - timeout-bound injectable Git/process runner.
- `internal/projectenv/gitadapter/identify.go` - worktree/common-dir/config-path identity.
- `internal/projectenv/gitadapter/inspect.go` - local values, origins, fingerprint, and blockers.
- `internal/projectenv/gitadapter/plan.go` - set/replace/clear/unchanged/unmanaged operations.
- `internal/projectenv/gitadapter/apply.go` - lock, stale check, temporary config mutation, verification, atomic replace.
- `internal/projectenv/gitadapter/signing.go` - format-aware, local-only signing diagnostics.
- Corresponding `*_test.go` integration tests using temporary repositories.

### Main-package integration

- `env_cli.go`, `env_cli_test.go` - top-level offline `rootyard env` parser, rendering, and confirmation.
- `app_environments.go`, `app_environments_test.go` - Wails bridge and cancellable scan manager.
- `main.go` - dispatch top-level `env` before desktop startup.
- `app.go` - compose the environment service and start/cancel automatic discovery.
- `frontend/wailsjs/go/main/App.js`, `App.d.ts`, `frontend/wailsjs/go/models.ts` - generated bindings.

### New frontend files

- `frontend/src/projectEnvironmentUi.js` and `.test.mjs` - mode editing, status filtering, selection, summaries.
- `frontend/src/components/environments/EnvironmentWorkspace.vue` - data owner and view navigation.
- `EnvironmentOverview.vue` - summary and scan entry points.
- `EnvironmentPresetManager.vue` - preset CRUD, field editor, usage impact.
- `EnvironmentRuleManager.vue` - defaults and path-backed rule editor.
- `EnvironmentRepositoryManager.vue` - repository filter/selection/status table.
- `EnvironmentDiffPanel.vue` - operation preview, explicit confirmation, and batch results.
- `frontend/src/App.vue` - add the Environments top-level workspace tab.

### Documentation updates

- `CLAUDE.md`
- `docs/specs/ARCHITECTURE.md`
- `docs/specs/DOMAIN_MODEL.md`
- `docs/specs/RUNTIME_CONTRACTS.md`
- `docs/specs/FRONTEND_STYLE.md`
- `docs/ai/rootyard-cli.md`
- `README.md`

---

## Task 1: Core model, validation, paths, and resolver

**Files:**
- Create: `internal/projectenv/model.go`
- Create: `internal/projectenv/errors.go`
- Create: `internal/projectenv/paths.go`
- Create: `internal/projectenv/validate.go`
- Create: `internal/projectenv/resolver.go`
- Test: `internal/projectenv/paths_test.go`
- Test: `internal/projectenv/validate_test.go`
- Test: `internal/projectenv/resolver_test.go`

**Interfaces:**
- Consumes: none.
- Produces: `Config`, `Document`, `FieldIntent`, `FieldMap`, `ResolvedEnvironment`, `Resolve`, `ValidateConfig`, `PathsForConfig`, and stable error codes used by every later task.

- [ ] **Step 1: Write failing resolver and validation tests**

Create table-driven tests that prove layer order, provenance, exact-rule behavior, duplicate normalized path rejection, missing preset rejection, and session overrides. Use this core case verbatim:

```go
func TestResolveLayersPresetRuleAndSession(t *testing.T) {
	cfg := Config{
		Version: 1,
		Defaults: LayerConfig{Git: FieldMap{
			"user.name": {Mode: ModeSet, Value: "Default Name"},
		}},
		Presets: []Preset{{ID: "company", Name: "Company", Git: FieldMap{
			"user.email": {Mode: ModeSet, Value: "company@example.com"},
		}}},
		Rules: []Rule{
			{ID: "outer", Path: "/work", Recursive: true, PresetID: "company"},
			{ID: "exact", Path: "/work/api", Git: FieldMap{
				"user.email": {Mode: ModeSet, Value: "api@example.com"},
			}},
		},
	}
	got, err := Resolve(cfg, "/work/api", FieldMap{
		"commit.gpgSign": {Mode: ModeSet, Value: "true"},
	})
	if err != nil { t.Fatal(err) }
	if got.Git["user.name"].Intent.Value != "Default Name" { t.Fatal(got) }
	if got.Git["user.email"].Intent.Value != "api@example.com" { t.Fatal(got) }
	if got.Git["user.email"].Source.ID != "exact" { t.Fatal(got) }
	if got.Git["commit.gpgSign"].Source.Kind != SourceSession { t.Fatal(got) }
}
```

- [ ] **Step 2: Run the focused tests and verify red**

Run:

```bash
export GO=/home/attson/sdk/go1.24.13/bin/go
$GO test ./internal/projectenv -run 'TestResolve|TestValidate|TestCanonical' -count=1
```

Expected: FAIL because the package/types do not exist.

- [ ] **Step 3: Implement the persisted and runtime model**

Define these exact core types and constants in `model.go`; add JSON tags shown by the design spec:

```go
type Mode string
const ( ModeInherit Mode = "inherit"; ModeSet Mode = "set"; ModeClear Mode = "clear" )

type FieldIntent struct { Mode Mode `json:"mode"`; Value string `json:"value,omitempty"` }
type FieldMap map[string]FieldIntent
type LayerConfig struct { PresetID string `json:"presetId,omitempty"`; Git FieldMap `json:"git,omitempty"` }
type Preset struct { ID, Name, Description string; Git FieldMap }
type Rule struct { ID, Name, Path string; Recursive, AutoDiscover bool; PresetID string; Git FieldMap }
type Config struct { Version int; Defaults LayerConfig; Presets []Preset; Rules []Rule }
type Document struct { Config Config `json:"config"`; Revision string `json:"revision"` }
type Source struct { Kind, ID, Name string }
type ResolvedField struct { Intent FieldIntent; Source Source }
type ResolvedEnvironment struct { RepoPath string; MatchingRuleIDs []string; Git map[string]ResolvedField }
```

Use named source constants `default`, `preset`, `rule`, and `session`. Define the six supported Git keys once as `SupportedGitKeys` and make validation reject unknown keys, invalid modes, invalid booleans, invalid `gpg.format`, values on inherit/clear, and missing values on set.

- [ ] **Step 4: Implement paths and resolver**

`PathsForConfig(configPath)` returns sibling `environments.json`, `environment-state.json`, and their lock paths. `CanonicalRulePath` requires an existing directory, makes it absolute, evaluates the selected root symlink, and cleans platform case. `PathContains` compares components, not string prefixes.

Implement `Resolve` with one `applyLayer(presetID string, inline FieldMap, source Source)` helper. Apply defaults, sorted matching rules, then session fields. An exact rule matches once regardless of `Recursive`; a strict ancestor requires `Recursive`.

- [ ] **Step 5: Run tests and commit**

Run `$GO test ./internal/projectenv -count=1`. Expected: PASS.

```bash
git add internal/projectenv
git commit -m "feat: add project environment resolver"
```

## Task 2: Cross-process locking and atomic stores

**Files:**
- Create: `internal/projectenv/filelock.go`
- Create: `internal/projectenv/filelock_unix.go`
- Create: `internal/projectenv/filelock_windows.go`
- Create: `internal/projectenv/store.go`
- Create: `internal/projectenv/state.go`
- Test: `internal/projectenv/filelock_test.go`
- Test: `internal/projectenv/store_test.go`
- Test: `internal/projectenv/state_test.go`

**Interfaces:**
- Consumes: Task 1 `Config`, `Document`, `ValidateConfig`, `PathsForConfig`, `Error`.
- Produces: `Store.Load`, `Store.Update`, `StateStore.Load`, `StateStore.Save`, and `StateDocument`.

- [ ] **Step 1: Write failing persistence tests**

Cover absent config without eager file creation, strict unknown-field rejection, corrupt authoritative file preservation, revision mismatch, atomic update, lock timeout, state quarantine, and state reload. The stale test must assert the stable code:

```go
_, err = store.Update(ctx, first.Revision, func(cfg *Config) error {
	cfg.Presets = append(cfg.Presets, Preset{ID: "stale", Name: "Stale"})
	return nil
})
if !IsCode(err, ErrEnvironmentStale) { t.Fatalf("err = %v", err) }
```

- [ ] **Step 2: Run tests and verify red**

Run `$GO test ./internal/projectenv -run 'TestStore|TestState|TestFileLock' -count=1`. Expected: FAIL with undefined store/lock symbols.

- [ ] **Step 3: Implement the platform lock**

Expose one internal API:

```go
func withFileLock(ctx context.Context, path string, fn func() error) error
```

On Unix use `unix.Flock(fd, LOCK_EX|LOCK_NB)` with a short retry timer until context cancellation. On Windows use `windows.LockFileEx`/`UnlockFileEx` with the same context-bounded retry contract. Keep the common retry/error mapping in `filelock.go`; build-tag only the OS syscalls.

- [ ] **Step 4: Implement strict config storage**

Create:

```go
type Store struct { paths Paths }
func NewStore(configPath string) *Store
func (s *Store) Load(ctx context.Context) (Document, error)
func (s *Store) Update(ctx context.Context, expectedRevision string, mutate func(*Config) error) (Document, error)
```

Use `json.Decoder.DisallowUnknownFields`, reject trailing JSON, hash exact bytes with SHA-256, and return an empty version 1 document when absent. Under the config lock, reload, compare non-empty expected revision, mutate, validate the complete model, marshal indented JSON, write a same-directory temporary file with `0600`, `Sync`, rename, and sync the directory on Unix.

For an absent file, calculate the revision from the canonical indented empty version 1 document rather
than returning an empty revision. That lets the first desktop save detect a file created concurrently by
the CLI.

- [ ] **Step 5: Implement rebuildable state storage**

Define `StateDocument`, `RuleScanState`, `RepositoryRecord`, and `RepositoryAlias` in `state.go`. `StateStore.Load` moves invalid state to `environment-state.json.corrupt-<UTC timestamp>` under the state lock and returns an empty version 1 state. `Save` uses its own lock and the same atomic writer.

- [ ] **Step 6: Verify race behavior and commit**

Run:

```bash
$GO test -race ./internal/projectenv -run 'TestStore|TestState|TestFileLock' -count=1
```

Expected: PASS with no race reports.

```bash
git add internal/projectenv
git commit -m "feat: persist project environment state safely"
```

## Task 3: Cancellable repository discovery

**Files:**
- Create: `internal/projectenv/discovery.go`
- Test: `internal/projectenv/discovery_test.go`

**Interfaces:**
- Consumes: Task 1 `Rule`, canonical paths, and Task 2 repository state types.
- Produces: `Discover(ctx, rules, options, progress) ScanResult`, `ScanProgress`, `ScanWarning`, and `RepositoryCandidate`.

- [ ] **Step 1: Write discovery tests**

Build temporary trees covering `.git` directories, `.git` files, nested repositories, exact rules, recursive rules, symlink descendants, ignored `.git/node_modules/vendor/.cache` directories, overlapping rules, and cancellation. Assert that reaching a repository prunes its subtree and that warning paths do not stop other roots.

- [ ] **Step 2: Run the discovery tests and verify red**

Run `$GO test ./internal/projectenv -run TestDiscover -count=1`. Expected: FAIL because `Discover` is undefined.

- [ ] **Step 3: Implement discovery**

Use these contracts:

```go
type DiscoverOptions struct { RuleIDs []string; AutoOnly bool }
type RepositoryCandidate struct { Path string; RuleIDs []string }
type ScanProgress struct { RuleID, Path string; Repositories, Warnings int; Done bool }
type ScanResult struct { Repositories []RepositoryCandidate; Warnings []Issue; Canceled bool }
func Discover(ctx context.Context, rules []Rule, opts DiscoverOptions, emit func(ScanProgress)) ScanResult
```

Use `filepath.WalkDir`, check `ctx.Err()` on every callback, never follow symlink directories, prune the four fixed ignored names only for descendants, detect either kind of `.git`, and merge candidates by canonical worktree path while preserving all matching rule IDs.

- [ ] **Step 4: Run tests and commit**

Run `$GO test ./internal/projectenv -run TestDiscover -count=1`. Expected: PASS.

```bash
git add internal/projectenv/discovery.go internal/projectenv/discovery_test.go
git commit -m "feat: discover environment repositories"
```

## Task 4: Git target identity, inspection, and planning

**Files:**
- Create: `internal/projectenv/adapter.go`
- Create: `internal/projectenv/gitadapter/runner.go`
- Create: `internal/projectenv/gitadapter/identify.go`
- Create: `internal/projectenv/gitadapter/inspect.go`
- Create: `internal/projectenv/gitadapter/plan.go`
- Test: `internal/projectenv/gitadapter/identify_test.go`
- Test: `internal/projectenv/gitadapter/inspect_test.go`
- Test: `internal/projectenv/gitadapter/plan_test.go`

**Interfaces:**
- Consumes: Task 1 resolved fields and stable issues.
- Produces: the `projectenv.Adapter` contract and `gitadapter.New()` used by Task 6.

- [ ] **Step 1: Define adapter contracts and failing tests**

Add this interface and generic models in `adapter.go`:

```go
type Adapter interface {
	Name() string
	Identify(context.Context, string) (Target, error)
	Inspect(context.Context, Target) (Snapshot, error)
	Plan(ResolvedEnvironment, Snapshot) ChangePlan
	Apply(context.Context, ChangePlan) ApplyResult
	Doctor(context.Context, Target, ResolvedEnvironment) []Issue
}
```

Define `Target`, `OriginValue`, `Snapshot`, `Operation`, `Change`, `ChangePlan`, and `ApplyResult` with JSON tags. Tests must cover ordinary repositories, linked worktrees sharing a target ID, absent/single/repeated/conflicting values, local include origins, fingerprints, and every plan operation.

- [ ] **Step 2: Run adapter tests and verify red**

Run `$GO test ./internal/projectenv/gitadapter -run 'TestIdentify|TestInspect|TestPlan' -count=1`. Expected: FAIL because the adapter is not implemented.

- [ ] **Step 3: Implement the injected runner and identity**

Use a private runner interface so error and timeout paths are testable:

```go
type Runner interface { Run(ctx context.Context, dir string, args ...string) (stdout, stderr string, err error) }
func New() *Adapter
```

`Identify` calls `git rev-parse --show-toplevel`, `--git-common-dir`, and `--git-path config`, converts results to canonical absolute paths, and uses the common directory as `Target.ID`.

- [ ] **Step 4: Implement inspection and origin safety**

Query each supported key with local scope and origin information. Record all values, classify values from the main config versus included local files, fingerprint the main config bytes, and emit `external_local_origin` when an included file owns a managed key. Preserve useful Git stderr in `Issue.Detail`.

- [ ] **Step 5: Implement pure planning**

For each supported key compare final intent with snapshot values and emit exactly one operation: `unmanaged`, `unchanged`, `set`, `replace`, `clear`, or `blocked`. Put the snapshot fingerprint and complete desired intents in the plan so apply never silently re-resolves a different target.

- [ ] **Step 6: Run tests and commit**

Run `$GO test ./internal/projectenv/gitadapter -run 'TestIdentify|TestInspect|TestPlan' -count=1`. Expected: PASS.

```bash
git add internal/projectenv/adapter.go internal/projectenv/gitadapter
git commit -m "feat: inspect and plan Git environment changes"
```

## Task 5: Atomic Git apply and signing diagnostics

**Files:**
- Create: `internal/projectenv/gitadapter/apply.go`
- Create: `internal/projectenv/gitadapter/signing.go`
- Test: `internal/projectenv/gitadapter/apply_test.go`
- Test: `internal/projectenv/gitadapter/signing_test.go`

**Interfaces:**
- Consumes: Task 4 `Target`, `Snapshot`, `ChangePlan`, and runner.
- Produces: working `Adapter.Apply` and `Adapter.Doctor` implementations.

- [ ] **Step 1: Write failing atomicity and diagnostic tests**

Test set, replace-all, clear-all, unchanged, stale fingerprint, pre-existing config lock, injected Git failure, linked-worktree target, temp cleanup, missing OpenPGP program/key, missing SSH key path, and valid disabled-signing behavior. The injected-failure test must compare original bytes before and after:

```go
before, _ := os.ReadFile(plan.Target.ConfigPath)
result := adapter.Apply(ctx, plan)
after, _ := os.ReadFile(plan.Target.ConfigPath)
if result.Status != projectenv.ApplyFailed { t.Fatal(result) }
if !bytes.Equal(before, after) { t.Fatal("failed apply changed native config") }
```

- [ ] **Step 2: Run tests and verify red**

Run `$GO test ./internal/projectenv/gitadapter -run 'TestApply|TestDoctorSigning' -count=1`. Expected: FAIL because apply/doctor are incomplete.

- [ ] **Step 3: Implement atomic apply**

Implement the nine-step algorithm from spec section 9.4. Acquire `<config>.lock` with context, re-fingerprint, copy the target to a same-directory temp file, run `git config --file <temp> --replace-all` or `--unset-all` for planned keys, inspect the temp, sync, preserve safe mode bits, rename, and clean up on every exit. Map outcomes to `success`, `unchanged`, `stale`, `blocked`, or `failed` without panics.

- [ ] **Step 4: Implement local signing diagnostics**

When both signing flags resolve false or unmanaged, return no key blocker. For active signing, validate `gpg.format`, require `user.signingKey`, and use bounded local commands: OpenPGP through the configured/default GPG binary, SSH path readability or `ssh-keygen` inspection, and X.509 through the configured/default GPGSM tool. Never fetch keys or create a signature.

- [ ] **Step 5: Run integration tests and commit**

Run `$GO test ./internal/projectenv/gitadapter -count=1`. Expected: PASS (Git-dependent tests skip only when Git is unavailable).

```bash
git add internal/projectenv/gitadapter
git commit -m "feat: apply Git environments atomically"
```

## Task 6: Project-environment service and batch orchestration

**Files:**
- Create: `internal/projectenv/service.go`
- Test: `internal/projectenv/service_test.go`

**Interfaces:**
- Consumes: Tasks 1-5 stores, discovery, resolver, adapter models, and `gitadapter.New()` supplied by composition roots.
- Produces: the complete `Service` API consumed by CLI and Wails.

- [ ] **Step 1: Write service-level failing tests**

Cover blank/clone/update/delete preset, deletion protection, defaults mutation, rule normalization/duplicate rejection, scan-state persistence, resolve, shared-target dedupe, `shared_target_conflict`, batch partial failure with stable ordering, state updates after apply, doctor aggregation, and context cancellation.

- [ ] **Step 2: Define the exact service request API**

Add request structs with JSON tags and these methods:

```go
func NewService(configPath string, adapters ...Adapter) *Service
func (s *Service) GetDocument(ctx context.Context) (Document, error)
func (s *Service) SaveDefaults(ctx context.Context, SaveDefaultsRequest) (Document, error)
func (s *Service) SavePreset(ctx context.Context, SavePresetRequest) (Document, error)
func (s *Service) ClonePreset(ctx context.Context, ClonePresetRequest) (Document, error)
func (s *Service) DeletePreset(ctx context.Context, DeletePresetRequest) (Document, error)
func (s *Service) SaveRule(ctx context.Context, SaveRuleRequest) (Document, error)
func (s *Service) DeleteRule(ctx context.Context, DeleteRuleRequest) (Document, error)
func (s *Service) Scan(ctx context.Context, ScanRequest, func(ScanProgress)) (ScanResult, error)
func (s *Service) State(ctx context.Context) (StateDocument, error)
func (s *Service) Resolve(ctx context.Context, ResolveRequest) (ResolvedEnvironment, error)
func (s *Service) Plan(ctx context.Context, PlanRequest) ([]ChangePlan, error)
func (s *Service) Apply(ctx context.Context, []ChangePlan) []ApplyResult
func (s *Service) Doctor(ctx context.Context, DoctorRequest) ([]Issue, error)
```

- [ ] **Step 3: Run service tests and verify red**

Run `$GO test ./internal/projectenv -run TestService -count=1`. Expected: FAIL because service methods are not implemented.

- [ ] **Step 4: Implement CRUD and scanning**

Use `Store.Update` for every mutation, assign opaque IDs only when input IDs are empty, reject referenced preset deletion, preserve unspecified update values, and write discovery results through `StateStore`. `ScanRequest{AutoOnly:true}` selects only enabled rules; an empty manual rule selection scans all rules.

Generate IDs with `crypto/rand` as `preset_<32 lowercase hex chars>`,
`rule_<32 lowercase hex chars>`, and `scan_<32 lowercase hex chars>`; inject the generator in tests so
assertions remain deterministic. Never derive identity from a mutable name or path.

- [ ] **Step 5: Implement resolve, plan, apply, and doctor**

Resolve every alias path before grouping by `Target.ID`. Equal desired hashes dedupe; unequal hashes return a blocked plan with every alias/rule and `shared_target_conflict`. Apply plans in input order, continue after individual failure, then persist each result/fingerprint. Doctor combines store, rule-path, Git, target, signing, and state diagnostics without mutation.

- [ ] **Step 6: Verify race behavior and commit**

Run:

```bash
$GO test -race ./internal/projectenv/... -count=1
```

Expected: PASS with no races.

```bash
git add internal/projectenv
git commit -m "feat: orchestrate project environments"
```

## Task 7: Offline `rootyard env` CLI

**Files:**
- Create: `env_cli.go`
- Create: `env_cli_test.go`
- Modify: `main.go`

**Interfaces:**
- Consumes: Task 6 `Service` and request/result models plus Task 5 `gitadapter.New()`.
- Produces: the complete spec section 12 command surface without a running desktop process.

- [ ] **Step 1: Write failing CLI contract tests**

Test top-level dispatch, sibling file paths under `--config`, defaults/preset/rule CRUD, repeatable `--git-set/--git-clear/--git-inherit`, scan/resolve/diff/doctor, human confirmation refusal, `--json` confirmation-required output, `--yes` apply, and stale-plan failure. At least one test must prove no control state file or desktop server exists.

- [ ] **Step 2: Run CLI tests and verify red**

Run `$GO test . -run 'TestRunEnvCLI|TestMainEnv' -count=1`. Expected: FAIL because `runEnvCLI` is undefined and `main` lacks the dispatch.

- [ ] **Step 3: Implement parser and output contracts**

In `main.go`, add:

```go
case "env":
	os.Exit(runEnvCLI(os.Args[2:], os.Stdin, os.Stdout, os.Stderr))
```

In `env_cli.go`, parse global `--config` and `--json` before command groups, build `projectenv.NewService(configPath, gitadapter.New())`, and map `projectenv.Error` to `{ok:false,error:{code,message,detail}}`. Implement field flags as repeatable typed values rather than ad-hoc splitting beyond the first `=`.

Dispatch every specified command explicitly: `defaults get/set`, `preset list/get/set/clone/delete`,
`rule list/get/set/delete`, and top-level `scan`, `resolve`, `diff`, `apply`, and `doctor`. Unknown groups,
subcommands, positional arguments, and flags return code `usage` without touching either environment
file.

- [ ] **Step 4: Implement preview and confirmation**

Human `diff` renders repository, source, old values, operations, and desired values. Human `apply` prints the same plan then reads an exact `y`/`yes` confirmation from a terminal; non-terminal input requires `--yes`. JSON apply without `--yes` returns the plan and `confirmation_required` without prompting or writing.

- [ ] **Step 5: Run tests and commit**

Run `$GO test . -run 'TestRunEnvCLI|TestMainEnv' -count=1`. Expected: PASS.

```bash
git add main.go env_cli.go env_cli_test.go
git commit -m "feat: add offline project environment CLI"
```

## Task 8: Wails bridge, scan lifecycle, and generated bindings

**Files:**
- Create: `app_environments.go`
- Create: `app_environments_test.go`
- Modify: `app.go`
- Modify generated: `frontend/wailsjs/go/main/App.js`
- Modify generated: `frontend/wailsjs/go/main/App.d.ts`
- Modify generated: `frontend/wailsjs/go/models.ts`

**Interfaces:**
- Consumes: Task 6 service and Task 5 Git adapter.
- Produces: Wails methods and `projectenv:scan-progress` / `projectenv:scan-complete` events for frontend tasks.

- [ ] **Step 1: Write failing App bridge tests**

Use `newTestApp(t)` to cover document CRUD delegation, plan/apply, unique scan IDs, scan cancellation, shutdown cancellation, and automatic discovery selecting only `AutoDiscover` rules.

- [ ] **Step 2: Run bridge tests and verify red**

Run `$GO test . -run TestEnvironmentApp -count=1`. Expected: FAIL because the App bridge does not exist.

- [ ] **Step 3: Compose service and expose bindings**

Add `envService *projectenv.Service` and a mutex-protected `map[string]context.CancelFunc` to `App`. Construct the service in `NewAppWithConfig`. Put exported methods in `app_environments.go`:

```go
GetEnvironmentDocument
SaveEnvironmentDefaults
SaveEnvironmentPreset
CloneEnvironmentPreset
DeleteEnvironmentPreset
SaveEnvironmentRule
DeleteEnvironmentRule
StartEnvironmentScan
CancelEnvironmentScan
GetEnvironmentState
ResolveEnvironment
PlanEnvironmentRepositories
ApplyEnvironmentPlans
DoctorEnvironments
```

`StartEnvironmentScan` returns a generated scan ID immediately, emits progress and one terminal complete event, and always removes its cancel function. Startup launches one `AutoOnly` scan; shutdown cancels all scans before closing other services.

- [ ] **Step 4: Run Go tests and regenerate bindings**

Run `$GO test . -run TestEnvironmentApp -count=1`. Expected: PASS.

Then run:

```bash
export GO=/home/attson/sdk/go1.24.13/bin/go
$($GO env GOPATH)/bin/wails generate module
```

Verify generated files expose every method above and projectenv models.

- [ ] **Step 5: Commit**

```bash
git add app.go app_environments.go app_environments_test.go frontend/wailsjs
git commit -m "feat: expose project environments to Wails"
```

## Task 9: Frontend environment shell and pure UI state

**Files:**
- Create: `frontend/src/projectEnvironmentUi.js`
- Create: `frontend/src/projectEnvironmentUi.test.mjs`
- Create: `frontend/src/components/environments/EnvironmentWorkspace.vue`
- Create: `frontend/src/components/environments/EnvironmentOverview.vue`
- Modify: `frontend/src/App.vue`

**Interfaces:**
- Consumes: Task 8 generated bindings/events.
- Produces: top-level Environments tab, workspace data owner, overview, and tested pure selection/status helpers.

- [ ] **Step 1: Write failing pure helper tests**

Test `normalizeEnvironmentDocument`, `repositoryStatus`, `environmentSummary`, `toggleRepositorySelection`, `selectableRepositoryIDs`, and `fieldRows`. Include blocked rows and shared-target aliases:

```js
test('blocked repositories cannot be selected', () => {
  const repos = [{ id: 'ok', status: 'pending' }, { id: 'bad', status: 'blocked' }]
  assert.deepEqual(toggleRepositorySelection(new Set(), 'bad', repos), new Set())
  assert.deepEqual([...toggleRepositorySelection(new Set(), 'ok', repos)], ['ok'])
})
```

- [ ] **Step 2: Run Node test and verify red**

Run `node --test frontend/src/projectEnvironmentUi.test.mjs`. Expected: FAIL because the helper module is missing.

- [ ] **Step 3: Implement helpers and workspace shell**

Keep helpers pure and tolerant of Wails' Go-shaped and JSON-shaped empty values. `EnvironmentWorkspace.vue` owns `document`, `state`, `activeView`, loading/error state, scan state, and event subscriptions. It imports only generated Wails bindings and passes data/actions to children.

Add an `Environments` top-level tab to `App.vue` beside Projects and Containers. When active, render only `EnvironmentWorkspace`; do not run project status filtering against it. Use a compact sidebar for Overview, Presets, Directory Rules, and Repositories.

When Environments is active, hide the process running/exited summary pills and the project-only New
Group, Scan, and Add actions. Environment scanning and creation actions live inside
`EnvironmentWorkspace`; switching back restores the existing Projects/Containers top bar unchanged.

- [ ] **Step 4: Implement overview and event cleanup**

`EnvironmentOverview.vue` renders rules, repositories, pending changes, and blockers using stable grid tracks. Manual scan calls `StartEnvironmentScan([])`; progress updates counters. On unmount, remove both Wails event subscriptions and cancel only a scan started by this mounted workspace.

- [ ] **Step 5: Run tests/build and commit**

Run:

```bash
node --test frontend/src/projectEnvironmentUi.test.mjs
cd frontend && npm run build
```

Expected: PASS and successful Vite build.

```bash
git add frontend/src/App.vue frontend/src/projectEnvironmentUi.js frontend/src/projectEnvironmentUi.test.mjs frontend/src/components/environments
git commit -m "feat: add project environment workspace"
```

## Task 10: Preset and directory-rule management UI

**Files:**
- Create: `frontend/src/components/environments/EnvironmentFieldEditor.vue`
- Create: `frontend/src/components/environments/EnvironmentPresetManager.vue`
- Create: `frontend/src/components/environments/EnvironmentRuleManager.vue`
- Modify: `frontend/src/components/environments/EnvironmentWorkspace.vue`
- Modify: `frontend/src/projectEnvironmentUi.js`
- Modify: `frontend/src/projectEnvironmentUi.test.mjs`

**Interfaces:**
- Consumes: Task 9 workspace data/actions and Task 8 CRUD bindings.
- Produces: complete preset/default/rule editing with revision handling and deletion protection.

- [ ] **Step 1: Extend failing helper tests**

Add tests for `editFieldMode`, `editFieldValue`, `presetUsage`, `validatePresetDraft`, `validateRuleDraft`, and stale-revision error mapping. Assert that changing a field to inherit or clear removes its value and that set requires a value.

- [ ] **Step 2: Run tests and verify red**

Run `node --test frontend/src/projectEnvironmentUi.test.mjs`. Expected: FAIL on the new helper exports.

- [ ] **Step 3: Implement the shared field editor**

Render the six fixed keys. Each row uses a select for Set / Not managed / Clear; value controls enable only for Set. Use checkboxes/toggles for booleans and a select for `gpg.format`. Emit an immutable `FieldMap` update and never save automatically.

- [ ] **Step 4: Implement preset management**

Build list/search, blank create, clone, name/description edit, usage tab, explicit save, and referenced-delete blocking. Pass `document.revision` on every mutation. On `environment_stale`, reload the document, preserve the unsaved draft, and show a conflict banner instead of overwriting.

- [ ] **Step 5: Implement defaults and rule management**

Show Global Defaults as a protected row. Rule editing includes native directory picker, exact/recursive segmented control, auto-discovery toggle, one preset select, inline fields, save, delete, matched-repository count, and scan action. Long paths use monospaced ellipsis plus `title`.

- [ ] **Step 6: Run tests/build and commit**

Run Node tests plus `cd frontend && npm run build`. Expected: PASS.

```bash
git add frontend/src/projectEnvironmentUi* frontend/src/components/environments
git commit -m "feat: manage environment presets and rules"
```

## Task 11: Repository preview, manual apply, and results UI

**Files:**
- Create: `frontend/src/components/environments/EnvironmentRepositoryManager.vue`
- Create: `frontend/src/components/environments/EnvironmentDiffPanel.vue`
- Modify: `frontend/src/components/environments/EnvironmentWorkspace.vue`
- Modify: `frontend/src/projectEnvironmentUi.js`
- Modify: `frontend/src/projectEnvironmentUi.test.mjs`
- Test static contract: `frontend/src/environmentWorkspaceUi.test.mjs`

**Interfaces:**
- Consumes: Task 8 plan/apply/doctor bindings and Task 9-10 workspace state.
- Produces: complete repository selection -> exact diff -> explicit confirmation -> per-target result flow.

- [ ] **Step 1: Add failing selection/result tests**

Test status filters, select-all excluding blocked rows, deduped target counts, change counts, result summaries, stale plan handling, and shared-target conflict display. The static contract test reads the Vue source and asserts confirmation text, disabled blocked checkboxes, and lucide icon imports.

- [ ] **Step 2: Run tests and verify red**

Run:

```bash
node --test frontend/src/projectEnvironmentUi.test.mjs frontend/src/environmentWorkspaceUi.test.mjs
```

Expected: FAIL on missing helpers/components/contracts.

- [ ] **Step 3: Implement repository manager**

Render filter/search, rule grouping, alias paths, effective preset, pending/consistent/drifted/blocked/failed status, blocker details, and stable checkboxes. Selecting Preview calls `PlanEnvironmentRepositories` with exactly the selected canonical targets.

- [ ] **Step 4: Implement diff and confirmation panel**

Show per repository and per key: source, existing local values, operation, and desired value. Keep Apply disabled until the user checks an explicit confirmation checkbox. Call `ApplyEnvironmentPlans` with the displayed plans, never a newly hidden re-plan. Render success/unchanged/stale/blocked/failed rows and offer re-preview for stale targets.

- [ ] **Step 5: Verify desktop layout and behavior**

Run all frontend Node tests and `npm run build`. Start `make dev`, inspect light and dark themes at 1024x768 and 1440x900, and exercise preset -> rule -> scan -> preview against a disposable Git repository. Confirm no overlap, clipped button text, nested cards, accidental write before confirmation, or console error.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/projectEnvironmentUi* frontend/src/environmentWorkspaceUi.test.mjs frontend/src/components/environments
git commit -m "feat: preview and apply Git environments"
```

## Task 12: Long-lived docs and end-to-end verification

**Files:**
- Modify: `CLAUDE.md`
- Modify: `docs/specs/ARCHITECTURE.md`
- Modify: `docs/specs/DOMAIN_MODEL.md`
- Modify: `docs/specs/RUNTIME_CONTRACTS.md`
- Modify: `docs/specs/FRONTEND_STYLE.md`
- Modify: `docs/ai/rootyard-cli.md`
- Modify: `README.md`

**Interfaces:**
- Consumes: all implemented behavior from Tasks 1-11.
- Produces: durable engineering and user documentation matching the shipped feature.

- [ ] **Step 1: Update long-lived engineering contracts**

Document `internal/projectenv`, Environment Preset/Directory Rule/Resolved Environment/Change Plan terms, the two environment files, adapter boundaries, atomic apply, scan events, the offline `rootyard env` exception, Wails workspace, and focused test commands. Keep existing runtime CLI/MCP control-server rules unchanged.

- [ ] **Step 2: Update user documentation**

Add the preset -> directory rule -> discover -> preview -> manual apply workflow, all CLI command groups and field flags, worktree/common-directory behavior, signing diagnostics, error meanings, and examples that never imply automatic writes.

- [ ] **Step 3: Run the full fresh verification suite**

Run:

```bash
export GO=/home/attson/sdk/go1.24.13/bin/go
$GO test ./...
$GO test -race ./internal/projectenv/...
node --test frontend/src/*.test.mjs frontend/src/composables/*.test.mjs frontend/src/components/fileExplorer/*.test.mjs
cd frontend && npm run build
```

Expected: every command exits 0; no Go race report; Vite build succeeds.

- [ ] **Step 4: Audit scope and repository state**

Run:

```bash
git diff --check
git status --short
rg -n 'TO''DO|FI''XME|TB''D' internal/projectenv env_cli.go app_environments.go frontend/src/components/environments frontend/src/projectEnvironmentUi.js
```

Expected: no whitespace errors, no uncommitted generated output omitted from the intended diff, and no implementation placeholders in the new subsystem.

- [ ] **Step 5: Commit documentation**

```bash
git add CLAUDE.md README.md docs/specs docs/ai/rootyard-cli.md
git commit -m "docs: document project environments"
```

- [ ] **Step 6: Request final review**

Summarize the 12 task commits, verification outputs, any Git/signing tools unavailable on the current platform, and the manual UI smoke result. Do not merge, push, tag, or release without a separate user request.
