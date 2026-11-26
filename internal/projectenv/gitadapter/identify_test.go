package gitadapter

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestIdentifyOrdinaryRepository(t *testing.T) {
	repo := newGitRepository(t)
	target, err := New().Identify(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	if target.ID == "" || target.WorktreePath == "" || target.CommonDir == "" || target.ConfigPath == "" {
		t.Fatalf("target = %+v", target)
	}
	if target.WorktreePath != repo {
		real, realErr := filepath.EvalSymlinks(repo)
		if realErr != nil || target.WorktreePath != real {
			t.Fatalf("worktree path = %q, repo = %q", target.WorktreePath, repo)
		}
	}
	if target.ID != target.CommonDir {
		t.Fatalf("ID = %q, common dir = %q", target.ID, target.CommonDir)
	}
	if _, err := os.Stat(target.ConfigPath); err != nil {
		t.Fatalf("config path = %q: %v", target.ConfigPath, err)
	}
}

func TestIdentifyLinkedWorktreesShareTarget(t *testing.T) {
	repo := newGitRepository(t)
	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("root\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "add", "README.md")
	runGit(t, repo, "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-m", "initial")
	linked := filepath.Join(t.TempDir(), "linked")
	runGit(t, repo, "worktree", "add", "-b", "linked-test", linked)

	adapter := New()
	primary := identifyTarget(t, adapter, repo)
	secondary := identifyTarget(t, adapter, linked)
	if primary.ID != secondary.ID || primary.ConfigPath != secondary.ConfigPath {
		t.Fatalf("primary = %+v, linked = %+v", primary, secondary)
	}
	if primary.WorktreePath == secondary.WorktreePath {
		t.Fatalf("worktree paths should differ: %+v %+v", primary, secondary)
	}
}
