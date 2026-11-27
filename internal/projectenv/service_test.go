package projectenv

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func stringPointer(value string) *string { return &value }
func boolPointer(value bool) *bool       { return &value }

type serviceAdapter struct {
	name               string
	targets            map[string]Target
	applyStatus        map[string]ApplyStatus
	doctorIssues       []Issue
	inspectErr         error
	inspectErrByTarget map[string]error
}

func (a *serviceAdapter) Name() string {
	if a.name != "" {
		return a.name
	}
	return "test"
}
func (a *serviceAdapter) Identify(ctx context.Context, path string) (Target, error) {
	if err := ctx.Err(); err != nil {
		return Target{}, err
	}
	if target, ok := a.targets[path]; ok {
		return target, nil
	}
	return Target{}, errors.New("unknown repository: " + path)
}
func (a *serviceAdapter) Inspect(ctx context.Context, target Target) (Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	if a.inspectErr != nil {
		return Snapshot{}, a.inspectErr
	}
	if err := a.inspectErrByTarget[target.ID]; err != nil {
		return Snapshot{}, err
	}
	return Snapshot{Target: target, Fingerprint: "fp-" + target.ID, Values: map[string][]OriginValue{}}, nil
}
func (a *serviceAdapter) Plan(resolved ResolvedEnvironment, snapshot Snapshot) ChangePlan {
	desired := map[string]FieldIntent{}
	for key, field := range resolved.Git {
		desired[key] = field.Intent
	}
	return ChangePlan{
		Adapter: a.Name(), Target: snapshot.Target, RepoPath: resolved.RepoPath,
		Fingerprint: snapshot.Fingerprint, Snapshot: snapshot, Desired: desired,
	}
}
func (a *serviceAdapter) Apply(_ context.Context, plan ChangePlan) ApplyResult {
	status := a.applyStatus[plan.Target.ID]
	if status == "" {
		status = ApplySuccess
	}
	return ApplyResult{Target: plan.Target, Status: status, Fingerprint: "after-" + plan.Target.ID}
}
func (a *serviceAdapter) Doctor(context.Context, Target, ResolvedEnvironment) []Issue {
	return append([]Issue(nil), a.doctorIssues...)
}

func newTestEnvironmentService(t *testing.T, adapters ...Adapter) (*Service, string) {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "config.json")
	service := NewService(configPath, adapters...)
	next := 0
	service.idGenerator = func(prefix string) (string, error) {
		next++
		return prefix + "test" + string(rune('0'+next)), nil
	}
	return service, configPath
}

func TestServicePresetCRUDAndDeletionProtection(t *testing.T) {
	service, _ := newTestEnvironmentService(t)
	ctx := context.Background()
	doc, err := service.GetDocument(ctx)
	if err != nil {
		t.Fatal(err)
	}
	name := "Company"
	doc, err = service.SavePreset(ctx, SavePresetRequest{
		Revision: doc.Revision, Name: &name,
		Git:   FieldMap{"user.email": {Mode: ModeSet, Value: "company@example.com"}},
		Codex: FieldMap{"model": {Mode: ModeSet, Value: "company-model"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Config.Presets) != 1 || doc.Config.Presets[0].ID == "" {
		t.Fatalf("presets = %+v", doc.Config.Presets)
	}
	presetID := doc.Config.Presets[0].ID
	doc, err = service.ClonePreset(ctx, ClonePresetRequest{Revision: doc.Revision, ID: presetID, Name: "Clone"})
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Config.Presets) != 2 ||
		doc.Config.Presets[1].Git["user.email"].Value != "company@example.com" ||
		doc.Config.Presets[1].Codex["model"].Value != "company-model" {
		t.Fatalf("presets = %+v", doc.Config.Presets)
	}
	doc, err = service.SaveDefaults(ctx, SaveDefaultsRequest{Revision: doc.Revision, PresetID: stringPointer(presetID)})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.DeletePreset(ctx, DeletePresetRequest{Revision: doc.Revision, ID: presetID}); !IsCode(err, ErrPresetInUse) {
		t.Fatalf("err = %v", err)
	}
	doc, err = service.SaveDefaults(ctx, SaveDefaultsRequest{Revision: doc.Revision, PresetID: stringPointer("")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.DeletePreset(ctx, DeletePresetRequest{Revision: doc.Revision, ID: presetID}); err != nil {
		t.Fatal(err)
	}
}

func TestServiceRuleNormalizesPathRejectsDuplicatesAndPersistsScan(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	makeRepositoryMarker(t, repo, false)
	service, _ := newTestEnvironmentService(t)
	ctx := context.Background()
	doc, _ := service.GetDocument(ctx)
	name := "Root"
	doc, err := service.SaveRule(ctx, SaveRuleRequest{
		Revision: doc.Revision, Name: &name, Path: stringPointer(root), Recursive: boolPointer(true),
	})
	if err != nil {
		t.Fatal(err)
	}
	canonicalRoot, _ := CanonicalRulePath(root)
	if doc.Config.Rules[0].Path != canonicalRoot {
		t.Fatalf("path = %q", doc.Config.Rules[0].Path)
	}
	otherName := "Duplicate"
	if _, err := service.SaveRule(ctx, SaveRuleRequest{
		Revision: doc.Revision, Name: &otherName, Path: stringPointer(filepath.Join(root, ".")),
	}); !IsCode(err, ErrDuplicateRulePath) {
		t.Fatalf("err = %v", err)
	}
	result, err := service.Scan(ctx, ScanRequest{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Repositories) != 1 {
		t.Fatalf("scan = %+v", result)
	}
	state, err := service.State(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Repositories) != 1 || len(state.RuleScans) != 1 {
		t.Fatalf("state = %+v", state)
	}
}

func TestServicePlanDeduplicatesSharedTargetAndBlocksConflict(t *testing.T) {
	root := t.TempDir()
	one := filepath.Join(root, "one")
	two := filepath.Join(root, "two")
	if err := os.MkdirAll(one, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(two, 0o755); err != nil {
		t.Fatal(err)
	}
	adapter := &serviceAdapter{targets: map[string]Target{
		one: {Adapter: "test", ID: "shared", WorktreePath: one, ConfigPath: filepath.Join(root, "config")},
		two: {Adapter: "test", ID: "shared", WorktreePath: two, ConfigPath: filepath.Join(root, "config")},
	}}
	service, _ := newTestEnvironmentService(t, adapter)
	ctx := context.Background()
	doc, _ := service.GetDocument(ctx)
	defaultsName := "Same"
	doc, err := service.SavePreset(ctx, SavePresetRequest{Revision: doc.Revision, Name: &defaultsName, Git: FieldMap{
		"user.email": {Mode: ModeSet, Value: "same@example.com"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	doc, err = service.SaveDefaults(ctx, SaveDefaultsRequest{Revision: doc.Revision, PresetID: stringPointer(doc.Config.Presets[0].ID)})
	if err != nil {
		t.Fatal(err)
	}
	plans, err := service.Plan(ctx, PlanRequest{RepositoryPaths: []string{one, two}})
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 || len(plans[0].Aliases) != 2 || len(plans[0].Blockers) != 0 {
		t.Fatalf("deduped plans = %+v", plans)
	}

	ruleName := "Different"
	doc, err = service.SaveRule(ctx, SaveRuleRequest{
		Revision: doc.Revision, Name: &ruleName, Path: stringPointer(two), Git: FieldMap{
			"user.email": {Mode: ModeSet, Value: "different@example.com"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	plans, err = service.Plan(ctx, PlanRequest{RepositoryPaths: []string{one, two}})
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 || len(plans[0].Blockers) == 0 || plans[0].Blockers[0].Code != ErrSharedTargetConflict {
		t.Fatalf("conflict plans = %+v", plans)
	}
}

func TestServicePlanDeduplicatesGitTargetButKeepsCodexWorktreesIndependent(t *testing.T) {
	root := t.TempDir()
	one := filepath.Join(root, "one")
	two := filepath.Join(root, "two")
	if err := os.MkdirAll(one, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(two, 0o755); err != nil {
		t.Fatal(err)
	}
	gitAdapter := &serviceAdapter{name: "git", targets: map[string]Target{
		one: {Adapter: "git", ID: "shared-common", WorktreePath: one},
		two: {Adapter: "git", ID: "shared-common", WorktreePath: two},
	}}
	codexAdapter := &serviceAdapter{name: "codex", targets: map[string]Target{
		one: {Adapter: "codex", ID: one, WorktreePath: one},
		two: {Adapter: "codex", ID: two, WorktreePath: two},
	}}
	service, _ := newTestEnvironmentService(t, gitAdapter, codexAdapter)
	plans, err := service.Plan(context.Background(), PlanRequest{RepositoryPaths: []string{one, two}})
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 3 {
		t.Fatalf("plans = %+v", plans)
	}
	counts := map[string]int{}
	for _, plan := range plans {
		counts[plan.Adapter]++
	}
	if counts["git"] != 1 || counts["codex"] != 2 {
		t.Fatalf("adapter plan counts = %+v", counts)
	}
}

func TestServiceDoctorPreservesTypedAdapterInspectionError(t *testing.T) {
	repo := t.TempDir()
	adapter := &serviceAdapter{
		name:       "codex",
		targets:    map[string]Target{repo: {Adapter: "codex", ID: repo, WorktreePath: repo, ConfigPath: filepath.Join(repo, ".codex", "config.toml")}},
		inspectErr: &Error{Code: ErrUnsafeCodexConfigPath, Message: "unsafe Codex project configuration path", Detail: "symlink"},
	}
	service, _ := newTestEnvironmentService(t, adapter)
	issues, err := service.Doctor(context.Background(), DoctorRequest{RepositoryPaths: []string{repo}})
	if err != nil {
		t.Fatal(err)
	}
	if len(issues) != 1 || issues[0].Code != ErrUnsafeCodexConfigPath {
		t.Fatalf("issues = %+v", issues)
	}
}

func TestServicePlanContinuesAfterAdapterFailure(t *testing.T) {
	repo := t.TempDir()
	bad := &serviceAdapter{
		name: "codex",
		targets: map[string]Target{repo: {
			Adapter: "codex", ID: repo, WorktreePath: repo, ConfigPath: filepath.Join(repo, ".codex", "config.toml"),
		}},
		inspectErrByTarget: map[string]error{repo: &Error{
			Code: ErrInvalidCodexConfig, Message: "invalid Codex project configuration", Detail: "line 1, column 9",
		}},
	}
	good := &serviceAdapter{name: "git", targets: map[string]Target{repo: {
		Adapter: "git", ID: "common", WorktreePath: repo, ConfigPath: filepath.Join(repo, ".git", "config"),
	}}}
	service, _ := newTestEnvironmentService(t, bad, good)

	plans, err := service.Plan(context.Background(), PlanRequest{RepositoryPaths: []string{repo}})
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 2 {
		t.Fatalf("plans = %+v", plans)
	}
	if plans[0].Adapter != "codex" || len(plans[0].Blockers) != 1 || plans[0].Blockers[0].Code != ErrInvalidCodexConfig {
		t.Fatalf("blocked plan = %+v", plans[0])
	}
	if plans[1].Adapter != "git" || len(plans[1].Blockers) != 0 {
		t.Fatalf("independent plan = %+v", plans[1])
	}
}

func TestServiceScopedScanPreservesUnselectedState(t *testing.T) {
	root := t.TempDir()
	shared := filepath.Join(root, "shared")
	manualOnly := filepath.Join(t.TempDir(), "manual-only")
	makeRepositoryMarker(t, shared, false)
	makeRepositoryMarker(t, manualOnly, false)
	shared = canonicalTestPath(t, shared)
	manualOnly = canonicalTestPath(t, manualOnly)
	service, configPath := newTestEnvironmentService(t)
	ctx := context.Background()
	doc, _ := service.GetDocument(ctx)
	outerName := "Outer"
	doc, err := service.SaveRule(ctx, SaveRuleRequest{
		Revision: doc.Revision, Name: &outerName, Path: stringPointer(root), Recursive: boolPointer(true), AutoDiscover: boolPointer(false),
	})
	if err != nil {
		t.Fatal(err)
	}
	autoName := "Auto"
	doc, err = service.SaveRule(ctx, SaveRuleRequest{
		Revision: doc.Revision, Name: &autoName, Path: stringPointer(shared), AutoDiscover: boolPointer(true),
	})
	if err != nil {
		t.Fatal(err)
	}
	manualName := "Manual"
	doc, err = service.SaveRule(ctx, SaveRuleRequest{
		Revision: doc.Revision, Name: &manualName, Path: stringPointer(manualOnly), AutoDiscover: boolPointer(false),
	})
	if err != nil {
		t.Fatal(err)
	}
	outerID, autoID, manualID := doc.Config.Rules[0].ID, doc.Config.Rules[1].ID, doc.Config.Rules[2].ID
	seed := StateDocument{Version: 2, RuleScans: []RuleScanState{
		{RuleID: outerID, ScanID: "outer-before"}, {RuleID: autoID, ScanID: "auto-before"}, {RuleID: manualID, ScanID: "manual-before"},
	}, Repositories: []RepositoryRecord{
		{ID: shared, Aliases: []RepositoryAlias{{Path: shared, RuleIDs: []string{outerID, autoID}}}, MatchingRuleIDs: []string{outerID, autoID}, Status: "consistent", Adapters: map[string]AdapterState{"git": {TargetID: "shared-common", Fingerprint: "keep"}}},
		{ID: manualOnly, Aliases: []RepositoryAlias{{Path: manualOnly, RuleIDs: []string{manualID}}}, MatchingRuleIDs: []string{manualID}, Status: "consistent", Adapters: map[string]AdapterState{"git": {TargetID: "manual-common", Fingerprint: "keep"}}},
	}}
	if err := NewStateStore(configPath).Save(ctx, seed); err != nil {
		t.Fatal(err)
	}

	if _, err := service.Scan(ctx, ScanRequest{AutoOnly: true}, nil); err != nil {
		t.Fatal(err)
	}
	state, err := service.State(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Repositories) != 2 || len(state.RuleScans) != 3 {
		t.Fatalf("state = %+v", state)
	}
	records := map[string]RepositoryRecord{}
	for _, record := range state.Repositories {
		records[record.ID] = record
	}
	sharedRecord := records[shared]
	if got := sharedRecord.Aliases[0].RuleIDs; !reflect.DeepEqual(got, []string{outerID, autoID}) || sharedRecord.Adapters["git"].Fingerprint != "keep" {
		t.Fatalf("shared record = %+v", sharedRecord)
	}
	if records[manualOnly].Adapters["git"].Fingerprint != "keep" {
		t.Fatalf("manual record = %+v", records[manualOnly])
	}
	for _, scan := range state.RuleScans {
		if scan.RuleID != autoID && scan.ScanID != map[string]string{outerID: "outer-before", manualID: "manual-before"}[scan.RuleID] {
			t.Fatalf("unselected scan was replaced: %+v", scan)
		}
	}
}

func TestServiceApplyContinuesInOrderAndUpdatesState(t *testing.T) {
	root := t.TempDir()
	paths := []string{filepath.Join(root, "a"), filepath.Join(root, "b"), filepath.Join(root, "c")}
	adapter := &serviceAdapter{targets: map[string]Target{}, applyStatus: map[string]ApplyStatus{"b": ApplyFailed}}
	plans := make([]ChangePlan, 0, len(paths))
	for _, path := range paths {
		id := filepath.Base(path)
		target := Target{Adapter: "test", ID: id, WorktreePath: path}
		adapter.targets[path] = target
		plans = append(plans, ChangePlan{Adapter: "test", Target: target, RepoPath: path})
	}
	service, _ := newTestEnvironmentService(t, adapter)
	results := service.Apply(context.Background(), plans)
	got := make([]string, 0, len(results))
	for _, result := range results {
		got = append(got, result.Target.ID+":"+string(result.Status))
	}
	want := []string{"a:success", "b:failed", "c:success"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("results = %#v", got)
	}
	state, err := service.State(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	sort.Slice(state.Repositories, func(i, j int) bool { return state.Repositories[i].ID < state.Repositories[j].ID })
	if len(state.Repositories) != 3 || state.Repositories[1].Adapters["test"].LastApplyResult != string(ApplyFailed) {
		t.Fatalf("state = %+v", state)
	}
}

func TestServiceApplyUpdatesScannedRepositoryRecord(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	makeRepositoryMarker(t, repo, false)
	adapter := &serviceAdapter{targets: map[string]Target{}}
	service, _ := newTestEnvironmentService(t, adapter)
	ctx := context.Background()
	doc, err := service.GetDocument(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ruleName := "Root"
	if _, err := service.SaveRule(ctx, SaveRuleRequest{
		Revision: doc.Revision, Name: &ruleName, Path: stringPointer(root), Recursive: boolPointer(true),
	}); err != nil {
		t.Fatal(err)
	}
	scan, err := service.Scan(ctx, ScanRequest{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	scannedRepo := scan.Repositories[0].Path
	adapter.targets[scannedRepo] = Target{Adapter: "test", ID: "common-dir", WorktreePath: scannedRepo}
	plans, err := service.Plan(ctx, PlanRequest{RepositoryPaths: []string{scannedRepo}})
	if err != nil {
		t.Fatal(err)
	}
	service.Apply(ctx, plans)
	state, err := service.State(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Repositories) != 1 {
		t.Fatalf("repositories = %+v", state.Repositories)
	}
	record := state.Repositories[0]
	if record.ID != scannedRepo || record.Adapters["test"].TargetID != "common-dir" ||
		record.Adapters["test"].LastApplyResult != string(ApplySuccess) || record.Status != "consistent" {
		t.Fatalf("repository = %+v", record)
	}
}

func TestServiceApplyStoresIndependentAdapterResultsAndWorstStatus(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	makeRepositoryMarker(t, repo, false)
	gitAdapter := &serviceAdapter{
		name: "git",
		targets: map[string]Target{repo: {
			Adapter: "git", ID: "shared-common-dir", WorktreePath: repo,
		}},
		applyStatus: map[string]ApplyStatus{"shared-common-dir": ApplyFailed},
	}
	codexAdapter := &serviceAdapter{
		name: "codex",
		targets: map[string]Target{repo: {
			Adapter: "codex", ID: repo, WorktreePath: repo,
		}},
	}
	service, _ := newTestEnvironmentService(t, gitAdapter, codexAdapter)
	ctx := context.Background()
	doc, err := service.GetDocument(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ruleName := "Root"
	if _, err := service.SaveRule(ctx, SaveRuleRequest{
		Revision: doc.Revision, Name: &ruleName, Path: stringPointer(root), Recursive: boolPointer(true),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Scan(ctx, ScanRequest{}, nil); err != nil {
		t.Fatal(err)
	}
	plans, err := service.Plan(ctx, PlanRequest{RepositoryPaths: []string{repo}})
	if err != nil {
		t.Fatal(err)
	}
	if len(plans) != 2 {
		t.Fatalf("plans = %+v", plans)
	}
	service.Apply(ctx, plans)
	state, err := service.State(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Repositories) != 1 {
		t.Fatalf("repositories = %+v", state.Repositories)
	}
	record := state.Repositories[0]
	if record.Adapters["git"].LastApplyResult != string(ApplyFailed) ||
		record.Adapters["codex"].LastApplyResult != string(ApplySuccess) ||
		record.Status != "failed" {
		t.Fatalf("repository = %+v", record)
	}
}

func TestServiceHonorsCanceledContext(t *testing.T) {
	service, _ := newTestEnvironmentService(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.GetDocument(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}
