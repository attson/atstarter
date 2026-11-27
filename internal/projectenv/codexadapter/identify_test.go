package codexadapter

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestIdentifyUsesEachLinkedWorktreeAsAnIndependentTarget(t *testing.T) {
	repo := makeRepository(t)
	linked := filepath.Join(filepath.Dir(repo), "linked")
	runGit(t, repo, "worktree", "add", "-b", "linked-test", linked)
	t.Cleanup(func() { _ = os.RemoveAll(linked) })

	adapter := New()
	nested := filepath.Join(repo, "internal", "package")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	mainTarget, err := adapter.Identify(context.Background(), nested)
	if err != nil {
		t.Fatal(err)
	}
	linkedTarget, err := adapter.Identify(context.Background(), linked)
	if err != nil {
		t.Fatal(err)
	}
	mainRoot, _ := filepath.EvalSymlinks(repo)
	linkedRoot, _ := filepath.EvalSymlinks(linked)
	if mainTarget.ID != mainRoot || mainTarget.WorktreePath != mainRoot {
		t.Fatalf("main target = %+v", mainTarget)
	}
	if linkedTarget.ID != linkedRoot || linkedTarget.WorktreePath != linkedRoot {
		t.Fatalf("linked target = %+v", linkedTarget)
	}
	if mainTarget.ID == linkedTarget.ID {
		t.Fatalf("linked worktrees shared Codex target: %+v %+v", mainTarget, linkedTarget)
	}
	if mainTarget.CommonDir != "" || linkedTarget.CommonDir != "" {
		t.Fatalf("Codex targets exposed Git common directory: %+v %+v", mainTarget, linkedTarget)
	}
	if mainTarget.ConfigPath != filepath.Join(mainRoot, ".codex", "config.toml") {
		t.Fatalf("config path = %q", mainTarget.ConfigPath)
	}
}
