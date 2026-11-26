package gitadapter

import (
	"testing"

	"rootyard/internal/projectenv"
)

func TestPlanEmitsEveryOperation(t *testing.T) {
	target := projectenv.Target{ID: "target", ConfigPath: "/repo/.git/config"}
	snapshot := projectenv.Snapshot{
		Target:      target,
		Fingerprint: "fingerprint",
		Values: map[string][]projectenv.OriginValue{
			"user.name":       {},
			"user.email":      {{Value: "same@example.com", MainConfig: true}},
			"user.signingKey": {{Value: "old", MainConfig: true}},
			"commit.gpgSign":  {{Value: "true", MainConfig: true}},
			"tag.gpgSign":     {{Value: "true", MainConfig: true}},
			"gpg.format":      {{Value: "ssh", Origin: "/included", MainConfig: false}},
		},
		Issues: []projectenv.Issue{{
			Code: projectenv.ErrExternalLocalOrigin, Key: "gpg.format", Blocking: true,
		}},
	}
	resolved := projectenv.ResolvedEnvironment{RepoPath: "/repo", Git: map[string]projectenv.ResolvedField{
		"user.name":       {Intent: projectenv.FieldIntent{Mode: projectenv.ModeSet, Value: "Ada"}},
		"user.email":      {Intent: projectenv.FieldIntent{Mode: projectenv.ModeSet, Value: "same@example.com"}},
		"user.signingKey": {Intent: projectenv.FieldIntent{Mode: projectenv.ModeSet, Value: "new"}},
		"commit.gpgSign":  {Intent: projectenv.FieldIntent{Mode: projectenv.ModeClear}},
		"gpg.format":      {Intent: projectenv.FieldIntent{Mode: projectenv.ModeSet, Value: "openpgp"}},
	}}

	plan := New().Plan(resolved, snapshot)
	want := map[string]projectenv.Operation{
		"user.name":       projectenv.OperationSet,
		"user.email":      projectenv.OperationUnchanged,
		"user.signingKey": projectenv.OperationReplace,
		"commit.gpgSign":  projectenv.OperationClear,
		"tag.gpgSign":     projectenv.OperationUnmanaged,
		"gpg.format":      projectenv.OperationBlocked,
	}
	for _, change := range plan.Changes {
		if change.Operation != want[change.Key] {
			t.Errorf("%s operation = %q, want %q", change.Key, change.Operation, want[change.Key])
		}
		delete(want, change.Key)
	}
	if len(want) != 0 {
		t.Fatalf("missing changes: %+v", want)
	}
	if plan.Fingerprint != snapshot.Fingerprint || plan.Desired["user.name"].Value != "Ada" {
		t.Fatalf("plan = %+v", plan)
	}
	if len(plan.Blockers) != 1 || plan.Blockers[0].Code != projectenv.ErrExternalLocalOrigin {
		t.Fatalf("blockers = %+v", plan.Blockers)
	}
}

func TestPlanReplacesRepeatedEqualValues(t *testing.T) {
	snapshot := projectenv.Snapshot{
		Values: map[string][]projectenv.OriginValue{
			"user.email": {
				{Value: "same@example.com", MainConfig: true},
				{Value: "same@example.com", MainConfig: true},
			},
		},
	}
	resolved := projectenv.ResolvedEnvironment{Git: map[string]projectenv.ResolvedField{
		"user.email": {Intent: projectenv.FieldIntent{Mode: projectenv.ModeSet, Value: "same@example.com"}},
	}}
	plan := New().Plan(resolved, snapshot)
	for _, change := range plan.Changes {
		if change.Key == "user.email" && change.Operation != projectenv.OperationReplace {
			t.Fatalf("change = %+v", change)
		}
	}
}
