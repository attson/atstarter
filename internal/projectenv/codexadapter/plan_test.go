package codexadapter

import (
	"context"
	"strings"
	"testing"

	"rootyard/internal/projectenv"
)

func TestPlanProducesMinimalEditsAndPreservesUnmanagedTOML(t *testing.T) {
	repo := makeRepository(t)
	fixture := "# heading\r\nmodel   = \"old-model\" # keep inline\r\nweb_search = \"live\" # rationale\r\ncustom = { token = \"secret\" }\r\n\r\n[features]\r\nmulti_agent = true\r\n"
	writeCodexConfig(t, repo, fixture)
	adapter := New()
	target, _ := adapter.Identify(context.Background(), repo)
	snapshot, err := adapter.Inspect(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	resolved := projectenv.ResolvedEnvironment{RepoPath: repo, Codex: map[string]projectenv.ResolvedField{
		"model":           {Intent: projectenv.FieldIntent{Mode: projectenv.ModeSet, Value: "new-model"}, Source: projectenv.Source{Kind: projectenv.SourcePreset, ID: "p", Name: "Company"}},
		"web_search":      {Intent: projectenv.FieldIntent{Mode: projectenv.ModeClear}, Source: projectenv.Source{Kind: projectenv.SourceRule, ID: "r", Name: "Work"}},
		"sandbox_mode":    {Intent: projectenv.FieldIntent{Mode: projectenv.ModeSet, Value: "danger-full-access"}, Source: projectenv.Source{Kind: projectenv.SourceRule, ID: "r", Name: "Work"}},
		"approval_policy": {Intent: projectenv.FieldIntent{Mode: projectenv.ModeSet, Value: "never"}, Source: projectenv.Source{Kind: projectenv.SourceRule, ID: "r", Name: "Work"}},
	}}

	plan := adapter.Plan(resolved, snapshot)
	if plan.Adapter != "codex" || plan.Fingerprint != snapshot.Fingerprint || len(plan.NativeEdits) != 3 {
		t.Fatalf("plan = %+v", plan)
	}
	operations := map[string]projectenv.Operation{}
	for _, change := range plan.Changes {
		operations[change.Key] = change.Operation
	}
	if operations["model"] != projectenv.OperationReplace || operations["web_search"] != projectenv.OperationClear ||
		operations["sandbox_mode"] != projectenv.OperationSet || operations["approval_policy"] != projectenv.OperationSet {
		t.Fatalf("operations = %+v", operations)
	}
	if len(plan.Warnings) != 1 || plan.Warnings[0].Code != projectenv.ErrDangerousCodexPermissions || plan.Warnings[0].Blocking {
		t.Fatalf("warnings = %+v", plan.Warnings)
	}
	got, err := applyTextEdits(snapshot.NativeContent, plan.NativeEdits)
	if err != nil {
		t.Fatal(err)
	}
	want := "# heading\r\nmodel   = \"new-model\" # keep inline\r\n# rationale\r\ncustom = { token = \"secret\" }\r\n\r\nsandbox_mode = \"danger-full-access\"\r\napproval_policy = \"never\"\r\n[features]\r\nmulti_agent = true\r\n"
	assertBytesEqual(t, got, []byte(want))
	if len(plan.NativeDiff) == 0 {
		t.Fatal("managed native diff is empty")
	}
	for _, line := range plan.NativeDiff {
		if strings.Contains(line, "secret") || strings.Contains(line, "multi_agent") {
			t.Fatalf("native diff exposed unmanaged content: %#v", plan.NativeDiff)
		}
	}
}

func TestPlanAppendsManagedValueWithoutAddingAnExtraFinalNewline(t *testing.T) {
	repo := makeRepository(t)
	writeCodexConfig(t, repo, `custom = "keep"`)
	adapter := New()
	target, _ := adapter.Identify(context.Background(), repo)
	snapshot, err := adapter.Inspect(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	plan := adapter.Plan(projectenv.ResolvedEnvironment{RepoPath: repo, Codex: map[string]projectenv.ResolvedField{
		"model": {Intent: projectenv.FieldIntent{Mode: projectenv.ModeSet, Value: "custom-model"}},
	}}, snapshot)
	got, err := applyTextEdits(snapshot.NativeContent, plan.NativeEdits)
	if err != nil {
		t.Fatal(err)
	}
	assertBytesEqual(t, got, []byte("custom = \"keep\"\nmodel = \"custom-model\""))
}

func TestPlanAllInheritedDoesNotCreateMissingConfig(t *testing.T) {
	repo := makeRepository(t)
	adapter := New()
	target, _ := adapter.Identify(context.Background(), repo)
	snapshot, err := adapter.Inspect(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	plan := adapter.Plan(projectenv.ResolvedEnvironment{RepoPath: repo, Codex: map[string]projectenv.ResolvedField{}}, snapshot)
	if len(plan.NativeEdits) != 0 || len(plan.NativeDiff) != 0 || len(plan.Blockers) != 0 {
		t.Fatalf("plan = %+v", plan)
	}
	for _, change := range plan.Changes {
		if change.Operation != projectenv.OperationUnmanaged {
			t.Fatalf("change = %+v", change)
		}
	}
}
