package projectenv

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func makeRepositoryMarker(t *testing.T, path string, asFile bool) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	gitPath := filepath.Join(path, ".git")
	if asFile {
		if err := os.WriteFile(gitPath, []byte("gitdir: elsewhere\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	if err := os.Mkdir(gitPath, 0o755); err != nil {
		t.Fatal(err)
	}
}

func repositoryPaths(result ScanResult) []string {
	paths := make([]string, 0, len(result.Repositories))
	for _, repository := range result.Repositories {
		paths = append(paths, repository.Path)
	}
	sort.Strings(paths)
	return paths
}

func canonicalTestPath(t *testing.T, path string) string {
	t.Helper()
	canonical, err := CanonicalRulePath(path)
	if err != nil {
		t.Fatal(err)
	}
	return canonical
}

func TestDiscoverFindsGitDirectoriesAndFilesAndPrunesRepositories(t *testing.T) {
	root := t.TempDir()
	directoryRepo := filepath.Join(root, "directory")
	fileRepo := filepath.Join(root, "linked")
	makeRepositoryMarker(t, directoryRepo, false)
	makeRepositoryMarker(t, fileRepo, true)
	makeRepositoryMarker(t, filepath.Join(directoryRepo, "nested"), false)

	result := Discover(context.Background(), []Rule{{ID: "root", Path: root, Recursive: true}}, DiscoverOptions{}, nil)
	if result.Canceled || len(result.Warnings) != 0 {
		t.Fatalf("result = %+v", result)
	}
	want := []string{canonicalTestPath(t, directoryRepo), canonicalTestPath(t, fileRepo)}
	sort.Strings(want)
	if got := repositoryPaths(result); !reflect.DeepEqual(got, want) {
		t.Fatalf("repositories = %#v, want %#v", got, want)
	}
}

func TestDiscoverExactRuleDoesNotWalkDescendants(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	makeRepositoryMarker(t, child, false)

	result := Discover(context.Background(), []Rule{{ID: "exact", Path: root}}, DiscoverOptions{}, nil)
	if len(result.Repositories) != 0 {
		t.Fatalf("exact rule found descendants: %+v", result.Repositories)
	}
	makeRepositoryMarker(t, root, false)
	result = Discover(context.Background(), []Rule{{ID: "exact", Path: root}}, DiscoverOptions{}, nil)
	if len(result.Repositories) != 1 || result.Repositories[0].Path != canonicalTestPath(t, root) {
		t.Fatalf("exact repository result = %+v", result.Repositories)
	}
}

func TestDiscoverSkipsFixedDirectoriesAndSymlinkDescendants(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"node_modules", "vendor", ".cache"} {
		makeRepositoryMarker(t, filepath.Join(root, name, "repo"), false)
	}
	external := t.TempDir()
	makeRepositoryMarker(t, filepath.Join(external, "repo"), false)
	if err := os.Symlink(external, filepath.Join(root, "linked-dir")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	result := Discover(context.Background(), []Rule{{ID: "root", Path: root, Recursive: true}}, DiscoverOptions{}, nil)
	if len(result.Repositories) != 0 {
		t.Fatalf("ignored or linked descendants were followed: %+v", result.Repositories)
	}
}

func TestDiscoverMergesOverlappingRuleIDs(t *testing.T) {
	root := t.TempDir()
	team := filepath.Join(root, "team")
	repo := filepath.Join(team, "repo")
	makeRepositoryMarker(t, repo, false)

	result := Discover(context.Background(), []Rule{
		{ID: "outer", Path: root, Recursive: true},
		{ID: "inner", Path: team, Recursive: true},
	}, DiscoverOptions{}, nil)
	if len(result.Repositories) != 1 {
		t.Fatalf("repositories = %+v", result.Repositories)
	}
	if got, want := result.Repositories[0].RuleIDs, []string{"outer", "inner"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("rule IDs = %#v, want %#v", got, want)
	}
}

func TestDiscoverFiltersRulesAndReportsProgress(t *testing.T) {
	root := t.TempDir()
	one := filepath.Join(root, "one")
	two := filepath.Join(root, "two")
	makeRepositoryMarker(t, one, false)
	makeRepositoryMarker(t, two, false)
	var progress []ScanProgress
	rules := []Rule{
		{ID: "manual", Path: one, AutoDiscover: false},
		{ID: "auto", Path: two, AutoDiscover: true},
	}
	result := Discover(context.Background(), rules, DiscoverOptions{AutoOnly: true}, func(update ScanProgress) {
		progress = append(progress, update)
	})
	if got := repositoryPaths(result); !reflect.DeepEqual(got, []string{canonicalTestPath(t, two)}) {
		t.Fatalf("repositories = %#v", got)
	}
	if len(progress) == 0 || !progress[len(progress)-1].Done || progress[len(progress)-1].RuleID != "auto" {
		t.Fatalf("progress = %+v", progress)
	}
}

func TestDiscoverWarningDoesNotStopOtherRoots(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo")
	makeRepositoryMarker(t, repo, false)
	missing := filepath.Join(t.TempDir(), "missing")

	result := Discover(context.Background(), []Rule{
		{ID: "missing", Path: missing, Recursive: true},
		{ID: "valid", Path: root, Recursive: true},
	}, DiscoverOptions{}, nil)
	if len(result.Warnings) != 1 || result.Warnings[0].Path != missing {
		t.Fatalf("warnings = %+v", result.Warnings)
	}
	if got := repositoryPaths(result); !reflect.DeepEqual(got, []string{canonicalTestPath(t, repo)}) {
		t.Fatalf("repositories = %#v", got)
	}
}

func TestDiscoverCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := Discover(ctx, []Rule{{ID: "rule", Path: t.TempDir(), Recursive: true}}, DiscoverOptions{}, nil)
	if !result.Canceled {
		t.Fatalf("result = %+v", result)
	}
}
