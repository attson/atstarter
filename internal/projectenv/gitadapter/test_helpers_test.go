package gitadapter

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"rootyard/internal/projectenv"
)

func requireGit(t *testing.T) string {
	t.Helper()
	path, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is unavailable")
	}
	return path
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	git := requireGit(t)
	cmd := exec.Command(git, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL="+filepath.Join(t.TempDir(), "global.gitconfig"),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return string(output)
}

func newGitRepository(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init")
	return dir
}

func identifyTarget(t *testing.T, adapter *Adapter, dir string) projectenv.Target {
	t.Helper()
	target, err := adapter.Identify(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	return target
}
