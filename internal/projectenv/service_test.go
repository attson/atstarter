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
	targets      map[string]Target
	applyStatus  map[string]ApplyStatus
	doctorIssues []Issue
}

func (a *serviceAdapter) Name() string { return "test" }
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
	if len(state.Repositories) != 3 || state.Repositories[1].LastApplyResult != string(ApplyFailed) {
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
	if record.ID != scannedRepo || record.TargetID != "common-dir" || record.LastApplyResult != string(ApplySuccess) {
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
