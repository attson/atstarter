package projectenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPathsForConfigUsesSiblingEnvironmentFiles(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "nested", "config.json")
	paths := PathsForConfig(configPath)
	dir := filepath.Dir(configPath)

	if paths.Config != filepath.Join(dir, "environments.json") {
		t.Fatalf("Config = %q", paths.Config)
	}
	if paths.State != filepath.Join(dir, "environment-state.json") {
		t.Fatalf("State = %q", paths.State)
	}
	if paths.ConfigLock != paths.Config+".lock" || paths.StateLock != paths.State+".lock" {
		t.Fatalf("unexpected lock paths: %+v", paths)
	}
}

func TestCanonicalRulePathResolvesSelectedRootSymlink(t *testing.T) {
	realRoot := filepath.Join(t.TempDir(), "real")
	if err := os.Mkdir(realRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	linkRoot := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(realRoot, linkRoot); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	got, err := CanonicalRulePath(linkRoot)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(realRoot)
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Clean(want) {
		t.Fatalf("CanonicalRulePath() = %q, want %q", got, want)
	}
}

func TestCanonicalRulePathRejectsMissingAndFilePaths(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{filepath.Join(t.TempDir(), "missing"), file} {
		if _, err := CanonicalRulePath(path); err == nil {
			t.Fatalf("CanonicalRulePath(%q) succeeded", path)
		}
	}
}

func TestPathContainsUsesPathComponents(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "work", "api")
	if !PathContains(root, filepath.Join(root, "nested")) {
		t.Fatal("expected descendant to be contained")
	}
	if !PathContains(root, root) {
		t.Fatal("expected path to contain itself")
	}
	if PathContains(root, filepath.Join(string(filepath.Separator), "work", "api-old")) {
		t.Fatal("string prefix must not count as path ancestry")
	}
}
