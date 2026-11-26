package gitadapter

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"rootyard/internal/projectenv"
)

func planForRepository(t *testing.T, adapter *Adapter, repo string, fields projectenv.FieldMap) projectenv.ChangePlan {
	t.Helper()
	target := identifyTarget(t, adapter, repo)
	snapshot, err := adapter.Inspect(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	resolvedFields := make(map[string]projectenv.ResolvedField, len(fields))
	for key, intent := range fields {
		resolvedFields[key] = projectenv.ResolvedField{Intent: intent, Source: projectenv.Source{Kind: projectenv.SourceSession}}
	}
	return adapter.Plan(projectenv.ResolvedEnvironment{RepoPath: repo, Git: resolvedFields}, snapshot)
}

func TestApplySetsReplacesAndClearsAtomically(t *testing.T) {
	repo := newGitRepository(t)
	runGit(t, repo, "config", "--local", "--add", "user.email", "old-one@example.com")
	runGit(t, repo, "config", "--local", "--add", "user.email", "old-two@example.com")
	runGit(t, repo, "config", "--local", "commit.gpgSign", "true")
	adapter := New()
	plan := planForRepository(t, adapter, repo, projectenv.FieldMap{
		"user.name":      {Mode: projectenv.ModeSet, Value: "Ada"},
		"user.email":     {Mode: projectenv.ModeSet, Value: "new@example.com"},
		"commit.gpgSign": {Mode: projectenv.ModeClear},
	})

	result := adapter.Apply(context.Background(), plan)
	if result.Status != projectenv.ApplySuccess {
		t.Fatalf("result = %+v", result)
	}
	snapshot, err := adapter.Inspect(context.Background(), plan.Target)
	if err != nil {
		t.Fatal(err)
	}
	if got := snapshot.Values["user.name"]; len(got) != 1 || got[0].Value != "Ada" {
		t.Fatalf("user.name = %+v", got)
	}
	if got := snapshot.Values["user.email"]; len(got) != 1 || got[0].Value != "new@example.com" {
		t.Fatalf("user.email = %+v", got)
	}
	if got := snapshot.Values["commit.gpgSign"]; len(got) != 0 {
		t.Fatalf("commit.gpgSign = %+v", got)
	}
}

func TestApplyReturnsUnchangedWithoutRewriting(t *testing.T) {
	repo := newGitRepository(t)
	runGit(t, repo, "config", "--local", "user.name", "Ada")
	adapter := New()
	plan := planForRepository(t, adapter, repo, projectenv.FieldMap{
		"user.name": {Mode: projectenv.ModeSet, Value: "Ada"},
	})
	before, err := os.ReadFile(plan.Target.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	result := adapter.Apply(context.Background(), plan)
	after, err := os.ReadFile(plan.Target.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if result.Status != projectenv.ApplyUnchanged || !bytes.Equal(before, after) {
		t.Fatalf("result = %+v, bytes changed = %v", result, !bytes.Equal(before, after))
	}
}

func TestApplyRejectsStaleFingerprintWithoutWriting(t *testing.T) {
	repo := newGitRepository(t)
	adapter := New()
	plan := planForRepository(t, adapter, repo, projectenv.FieldMap{
		"user.name": {Mode: projectenv.ModeSet, Value: "Planned"},
	})
	runGit(t, repo, "config", "--local", "user.name", "Concurrent")
	before, _ := os.ReadFile(plan.Target.ConfigPath)
	result := adapter.Apply(context.Background(), plan)
	after, _ := os.ReadFile(plan.Target.ConfigPath)
	if result.Status != projectenv.ApplyStale || !bytes.Equal(before, after) {
		t.Fatalf("result = %+v, config changed = %v", result, !bytes.Equal(before, after))
	}
}

func TestApplyHonorsPreexistingConfigLock(t *testing.T) {
	repo := newGitRepository(t)
	adapter := New()
	plan := planForRepository(t, adapter, repo, projectenv.FieldMap{
		"user.name": {Mode: projectenv.ModeSet, Value: "Ada"},
	})
	if err := os.WriteFile(plan.Target.ConfigPath+".lock", []byte("held"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	result := adapter.Apply(ctx, plan)
	if result.Status != projectenv.ApplyFailed {
		t.Fatalf("result = %+v", result)
	}
	if data, err := os.ReadFile(plan.Target.ConfigPath + ".lock"); err != nil || string(data) != "held" {
		t.Fatalf("pre-existing lock was changed: %q %v", data, err)
	}
}

type failingRunner struct {
	delegate Runner
}

func (r failingRunner) Run(ctx context.Context, dir string, args ...string) (string, string, error) {
	if len(args) > 2 && args[0] == "config" && args[1] == "--file" {
		return "", "injected Git failure", errors.New("injected Git failure")
	}
	return r.delegate.Run(ctx, dir, args...)
}

func TestApplyInjectedGitFailurePreservesOriginalAndCleansTemp(t *testing.T) {
	repo := newGitRepository(t)
	base := New()
	plan := planForRepository(t, base, repo, projectenv.FieldMap{
		"user.name": {Mode: projectenv.ModeSet, Value: "Ada"},
	})
	adapter := &Adapter{runner: failingRunner{delegate: base.runner}}
	before, _ := os.ReadFile(plan.Target.ConfigPath)
	result := adapter.Apply(context.Background(), plan)
	after, _ := os.ReadFile(plan.Target.ConfigPath)
	if result.Status != projectenv.ApplyFailed {
		t.Fatal(result)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("failed apply changed native config")
	}
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(plan.Target.ConfigPath), ".rootyard-config-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files remain: %+v", matches)
	}
}

func TestApplyLinkedWorktreeUsesSharedConfigTarget(t *testing.T) {
	repo := newGitRepository(t)
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("root\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "README.md")
	runGit(t, repo, "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-m", "initial")
	linked := filepath.Join(t.TempDir(), "linked")
	runGit(t, repo, "worktree", "add", "-b", "apply-linked", linked)
	adapter := New()
	plan := planForRepository(t, adapter, linked, projectenv.FieldMap{
		"user.email": {Mode: projectenv.ModeSet, Value: "linked@example.com"},
	})
	result := adapter.Apply(context.Background(), plan)
	if result.Status != projectenv.ApplySuccess {
		t.Fatalf("result = %+v", result)
	}
	if got := strings.TrimSpace(runGit(t, repo, "config", "--local", "--get", "user.email")); got != "linked@example.com" {
		t.Fatalf("shared user.email = %q", got)
	}
}
