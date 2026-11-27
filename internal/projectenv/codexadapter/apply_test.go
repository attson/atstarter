package codexadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"rootyard/internal/projectenv"
)

func planForProject(t *testing.T, adapter *Adapter, repo string, fields projectenv.FieldMap) projectenv.ChangePlan {
	t.Helper()
	target, err := adapter.Identify(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := adapter.Inspect(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	resolved := make(map[string]projectenv.ResolvedField, len(fields))
	for key, intent := range fields {
		resolved[key] = projectenv.ResolvedField{Intent: intent, Source: projectenv.Source{Kind: projectenv.SourceSession}}
	}
	return adapter.Plan(projectenv.ResolvedEnvironment{RepoPath: repo, Codex: resolved}, snapshot)
}

func TestApplyCreatesMissingCodexConfigOnlyAfterPlanConfirmation(t *testing.T) {
	repo := makeRepository(t)
	adapter := New()
	plan := planForProject(t, adapter, repo, projectenv.FieldMap{
		"model":      {Mode: projectenv.ModeSet, Value: "custom-model"},
		"web_search": {Mode: projectenv.ModeSet, Value: "cached"},
	})
	if _, err := os.Stat(filepath.Join(repo, ".codex")); !os.IsNotExist(err) {
		t.Fatalf("planning created .codex: %v", err)
	}

	result := adapter.Apply(context.Background(), plan)
	if result.Status != projectenv.ApplySuccess || result.Fingerprint == "" {
		t.Fatalf("result = %+v", result)
	}
	data, err := os.ReadFile(plan.Target.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	assertBytesEqual(t, data, []byte("model = \"custom-model\"\nweb_search = \"cached\"\n"))
	if info, err := os.Stat(plan.Target.ConfigPath); err != nil || runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("config mode = %v, err = %v", info.Mode().Perm(), err)
	}
}

func TestApplyAcceptsDisplayedPlanAfterJSONRoundTrip(t *testing.T) {
	repo := makeRepository(t)
	adapter := New()
	plan := planForProject(t, adapter, repo, projectenv.FieldMap{
		"model": {Mode: projectenv.ModeSet, Value: "custom-model"},
	})
	encoded, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	var displayed projectenv.ChangePlan
	if err := json.Unmarshal(encoded, &displayed); err != nil {
		t.Fatal(err)
	}
	result := adapter.Apply(context.Background(), displayed)
	if result.Status != projectenv.ApplySuccess {
		t.Fatalf("result = %+v", result)
	}
}

func TestApplyAllInheritedLeavesMissingConfigAbsent(t *testing.T) {
	repo := makeRepository(t)
	adapter := New()
	plan := planForProject(t, adapter, repo, projectenv.FieldMap{})
	result := adapter.Apply(context.Background(), plan)
	if result.Status != projectenv.ApplyUnchanged {
		t.Fatalf("result = %+v", result)
	}
	if _, err := os.Stat(filepath.Join(repo, ".codex")); !os.IsNotExist(err) {
		t.Fatalf("unchanged apply created .codex: %v", err)
	}
}

func TestApplyRejectsStaleExistingAndAppearedConfigs(t *testing.T) {
	t.Run("existing changed", func(t *testing.T) {
		repo := makeRepository(t)
		path := writeCodexConfig(t, repo, "model = \"before\"\n")
		adapter := New()
		plan := planForProject(t, adapter, repo, projectenv.FieldMap{"model": {Mode: projectenv.ModeSet, Value: "planned"}})
		changed := []byte("model = \"external\"\n")
		if err := os.WriteFile(path, changed, 0o600); err != nil {
			t.Fatal(err)
		}
		result := adapter.Apply(context.Background(), plan)
		if result.Status != projectenv.ApplyStale {
			t.Fatalf("result = %+v", result)
		}
		after, _ := os.ReadFile(path)
		assertBytesEqual(t, after, changed)
	})

	t.Run("missing appeared", func(t *testing.T) {
		repo := makeRepository(t)
		adapter := New()
		plan := planForProject(t, adapter, repo, projectenv.FieldMap{"model": {Mode: projectenv.ModeSet, Value: "planned"}})
		appeared := []byte("model = \"external\"\n")
		path := writeCodexConfig(t, repo, string(appeared))
		result := adapter.Apply(context.Background(), plan)
		if result.Status != projectenv.ApplyStale {
			t.Fatalf("result = %+v", result)
		}
		after, _ := os.ReadFile(path)
		assertBytesEqual(t, after, appeared)
	})
}

func TestApplyRejectsChangedTargetAndTamperedEdits(t *testing.T) {
	t.Run("target", func(t *testing.T) {
		repo := makeRepository(t)
		adapter := New()
		plan := planForProject(t, adapter, repo, projectenv.FieldMap{"model": {Mode: projectenv.ModeSet, Value: "planned"}})
		plan.Target.ID = filepath.Join(repo, "other")
		result := adapter.Apply(context.Background(), plan)
		if result.Status != projectenv.ApplyBlocked {
			t.Fatalf("result = %+v", result)
		}
		if _, err := os.Stat(filepath.Join(repo, ".codex")); !os.IsNotExist(err) {
			t.Fatalf("target mismatch wrote config: %v", err)
		}
	})

	tests := []struct {
		name string
		edit projectenv.TextEdit
	}{
		{name: "out of bounds", edit: projectenv.TextEdit{Start: 0, End: 999, Replacement: "model = \"bad\""}},
		{name: "invalid toml", edit: projectenv.TextEdit{Start: 0, End: 0, Replacement: "model = [\n"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := makeRepository(t)
			path := writeCodexConfig(t, repo, "custom = \"keep\"\n")
			before, _ := os.ReadFile(path)
			adapter := New()
			plan := planForProject(t, adapter, repo, projectenv.FieldMap{"model": {Mode: projectenv.ModeSet, Value: "planned"}})
			plan.NativeEdits = []projectenv.TextEdit{tt.edit}
			result := adapter.Apply(context.Background(), plan)
			if result.Status != projectenv.ApplyFailed {
				t.Fatalf("result = %+v", result)
			}
			after, _ := os.ReadFile(path)
			assertBytesEqual(t, after, before)
		})
	}
}

func TestApplyRejectsInvalidManagedValueEvenWhenStoredEditMatches(t *testing.T) {
	repo := makeRepository(t)
	adapter := New()
	plan := planForProject(t, adapter, repo, projectenv.FieldMap{
		"sandbox_mode": {Mode: projectenv.ModeSet, Value: "workspace-write"},
	})
	invalid := projectenv.FieldIntent{Mode: projectenv.ModeSet, Value: "full"}
	plan.Desired["sandbox_mode"] = invalid
	for index := range plan.Changes {
		if plan.Changes[index].Key == "sandbox_mode" {
			plan.Changes[index].Desired = invalid
		}
	}
	plan.NativeEdits = []projectenv.TextEdit{{Start: 0, End: 0, Replacement: "sandbox_mode = \"full\"\n"}}

	result := adapter.Apply(context.Background(), plan)
	if result.Status != projectenv.ApplyFailed {
		t.Fatalf("result = %+v", result)
	}
	if _, err := os.Stat(filepath.Join(repo, ".codex")); !os.IsNotExist(err) {
		t.Fatalf("invalid managed value wrote config: %v", err)
	}
}

func TestApplyPreservesExistingModeAndOriginalOnAtomicWriteFailure(t *testing.T) {
	repo := makeRepository(t)
	path := writeCodexConfig(t, repo, "model = \"before\"\n")
	if runtime.GOOS != "windows" {
		if err := os.Chmod(path, 0o640); err != nil {
			t.Fatal(err)
		}
	}
	adapter := New()
	plan := planForProject(t, adapter, repo, projectenv.FieldMap{"model": {Mode: projectenv.ModeSet, Value: "after"}})
	result := adapter.Apply(context.Background(), plan)
	if result.Status != projectenv.ApplySuccess {
		t.Fatalf("result = %+v", result)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != 0o640 {
			t.Fatalf("mode = %v, err = %v", info.Mode().Perm(), err)
		}
	}

	before, _ := os.ReadFile(path)
	failing := New()
	failing.writeConfig = func(*secureConfigDir, []byte, os.FileMode) error { return errors.New("injected write failure") }
	failedPlan := planForProject(t, failing, repo, projectenv.FieldMap{"model": {Mode: projectenv.ModeSet, Value: "failed"}})
	failed := failing.Apply(context.Background(), failedPlan)
	if failed.Status != projectenv.ApplyFailed {
		t.Fatalf("failed result = %+v", failed)
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatalf("failed write changed config: before=%q after=%q", before, after)
	}
}

func TestApplyKeepsLinkedWorktreeCodexConfigsIndependent(t *testing.T) {
	repo := makeRepository(t)
	linked := filepath.Join(filepath.Dir(repo), "linked")
	runGit(t, repo, "worktree", "add", "-b", "linked-apply", linked)
	t.Cleanup(func() { _ = os.RemoveAll(linked) })
	adapter := New()
	mainPlan := planForProject(t, adapter, repo, projectenv.FieldMap{"model": {Mode: projectenv.ModeSet, Value: "main-model"}})
	linkedPlan := planForProject(t, adapter, linked, projectenv.FieldMap{"model": {Mode: projectenv.ModeSet, Value: "linked-model"}})
	if adapter.Apply(context.Background(), mainPlan).Status != projectenv.ApplySuccess || adapter.Apply(context.Background(), linkedPlan).Status != projectenv.ApplySuccess {
		t.Fatal("linked worktree apply failed")
	}
	mainData, _ := os.ReadFile(mainPlan.Target.ConfigPath)
	linkedData, _ := os.ReadFile(linkedPlan.Target.ConfigPath)
	assertBytesEqual(t, mainData, []byte("model = \"main-model\"\n"))
	assertBytesEqual(t, linkedData, []byte("model = \"linked-model\"\n"))
}

func TestApplyRecoversFromStaleLockFile(t *testing.T) {
	repo := makeRepository(t)
	adapter := New()
	plan := planForProject(t, adapter, repo, projectenv.FieldMap{
		"model": {Mode: projectenv.ModeSet, Value: "planned"},
	})
	stalePath := filepath.Join(repo, ".rootyard-codex-config.lock")
	if err := os.WriteFile(stalePath, []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	result := adapter.Apply(ctx, plan)
	if result.Status != projectenv.ApplySuccess {
		t.Fatalf("result = %+v", result)
	}
}

func TestApplyTimesOutWaitingForLock(t *testing.T) {
	repo := makeRepository(t)
	adapter := New()
	adapter.lockTimeout = 30 * time.Millisecond
	plan := planForProject(t, adapter, repo, projectenv.FieldMap{
		"model": {Mode: projectenv.ModeSet, Value: "planned"},
	})
	unlock, err := projectenv.AcquireFileLock(context.Background(), projectenv.AdapterFileLockPath("codex", plan.Target.ID))
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()

	result := adapter.Apply(context.Background(), plan)
	if result.Status != projectenv.ApplyFailed || len(result.Issues) != 1 || result.Issues[0].Code != projectenv.ErrLockTimeout {
		t.Fatalf("result = %+v", result)
	}
}

func TestApplyRejectsCodexDirectorySwap(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory replacement timing differs on Windows")
	}
	repo := makeRepository(t)
	path := writeCodexConfig(t, repo, "model = \"before\"\n")
	adapter := New()
	plan := planForProject(t, adapter, repo, projectenv.FieldMap{
		"model": {Mode: projectenv.ModeSet, Value: "after"},
	})
	originalDir := filepath.Join(repo, ".codex-original")
	replacementDir := filepath.Join(repo, ".codex")
	adapter.beforeWrite = func() {
		if err := os.Rename(filepath.Dir(path), originalDir); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(replacementDir, 0o700); err != nil {
			t.Fatal(err)
		}
	}

	result := adapter.Apply(context.Background(), plan)
	if result.Status != projectenv.ApplyFailed {
		t.Fatalf("result = %+v", result)
	}
	if _, err := os.Stat(filepath.Join(replacementDir, "config.toml")); !os.IsNotExist(err) {
		t.Fatalf("replacement directory was written: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(originalDir, "config.toml"))
	if err != nil || string(data) != "model = \"before\"\n" {
		t.Fatalf("original config changed: %q, %v", data, err)
	}
}
