package scanner

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanFindsProjectsInDirectChildren(t *testing.T) {
	root := t.TempDir()
	// 子目录 a:go 项目
	write(t, filepath.Join(root, "a", "go.mod"), "module a\n")
	write(t, filepath.Join(root, "a", "main.go"), "package main\nfunc main(){}\n")
	// 子目录 b:pnpm 项目
	write(t, filepath.Join(root, "b", "package.json"), `{"scripts":{"dev":"vite"}}`)
	write(t, filepath.Join(root, "b", "pnpm-lock.yaml"), "")
	// 子目录 c:无法识别,仍应列出(unknown)
	write(t, filepath.Join(root, "c", "README.md"), "hi")
	// 文件(非目录)应被忽略
	write(t, filepath.Join(root, "loose.txt"), "x")

	got := Scan([]string{root})
	if len(got) != 3 {
		t.Fatalf("expected 3 candidates, got %d: %+v", len(got), got)
	}
	sort.Slice(got, func(i, j int) bool { return got[i].Name < got[j].Name })
	if got[0].Name != "a" || got[0].DetectedType != "go" {
		t.Errorf("a: got %+v", got[0])
	}
	if got[0].Command != "go" || len(got[0].Args) != 2 {
		t.Errorf("a command/args: got cmd=%q args=%v", got[0].Command, got[0].Args)
	}
	if got[1].Name != "b" || got[1].DetectedType != "node-pnpm" {
		t.Errorf("b: got %+v", got[1])
	}
	if got[2].Name != "c" || got[2].DetectedType != "unknown" {
		t.Errorf("c: got %+v", got[2])
	}
}

func TestScanSkipsMissingRoot(t *testing.T) {
	got := Scan([]string{"/nonexistent/path/xyz"})
	if len(got) != 0 {
		t.Errorf("expected 0 for missing root, got %d", len(got))
	}
}

func TestScanAddsDetectionOptionsForComposeFallback(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "svc", "docker-compose.yml"), "services: {}\n")
	write(t, filepath.Join(root, "svc", "go.mod"), "module svc\n")
	write(t, filepath.Join(root, "svc", "main.go"), "package main\nfunc main(){}\n")

	got := Scan([]string{root})
	if len(got) != 1 {
		t.Fatalf("expected 1 candidate, got %d: %+v", len(got), got)
	}
	if got[0].DetectedType != "compose" {
		t.Fatalf("DetectedType = %q, want compose", got[0].DetectedType)
	}
	if len(got[0].DetectionOptions) != 2 {
		t.Fatalf("DetectionOptions = %+v, want compose and go", got[0].DetectionOptions)
	}
	if got[0].DetectionOptions[1].Type != "go" || got[0].DetectionOptions[1].Command != "go" {
		t.Fatalf("fallback option = %+v, want go command", got[0].DetectionOptions[1])
	}
}

func TestScanIncludesWorktreeDirectories(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, ".worktrees", "feature-a", "go.mod"), "module a\n")
	write(t, filepath.Join(root, ".worktrees", "feature-a", "main.go"), "package main\nfunc main(){}\n")
	write(t, filepath.Join(root, ".claude", "worktrees", "review-b", "package.json"), `{"scripts":{"dev":"vite"}}`)
	write(t, filepath.Join(root, ".claude", "worktrees", "review-b", "pnpm-lock.yaml"), "")

	got := Scan([]string{root})
	sort.Slice(got, func(i, j int) bool { return got[i].Name < got[j].Name })
	if len(got) != 2 {
		t.Fatalf("expected 2 worktree candidates, got %d: %+v", len(got), got)
	}
	if got[0].Name != "feature-a" || got[0].DetectedType != "go" {
		t.Errorf("feature-a candidate = %+v", got[0])
	}
	if got[1].Name != "review-b" || got[1].DetectedType != "node-pnpm" {
		t.Errorf("review-b candidate = %+v", got[1])
	}
}

func TestScanIncludesWorktreesInsideDirectChildProjects(t *testing.T) {
	root := t.TempDir()
	write(t, filepath.Join(root, "repo", "go.mod"), "module repo\n")
	write(t, filepath.Join(root, "repo", "main.go"), "package main\nfunc main(){}\n")
	write(t, filepath.Join(root, "repo", ".claude", "worktrees", "budget-usage-proxy", "go.mod"), "module wt\n")
	write(t, filepath.Join(root, "repo", ".claude", "worktrees", "budget-usage-proxy", "main.go"), "package main\nfunc main(){}\n")
	write(t, filepath.Join(root, "repo", ".worktrees", "material-tag-proxy", "package.json"), `{"scripts":{"dev":"vite"}}`)
	write(t, filepath.Join(root, "repo", ".worktrees", "material-tag-proxy", "pnpm-lock.yaml"), "")

	got := Scan([]string{root})
	sort.Slice(got, func(i, j int) bool { return got[i].Name < got[j].Name })
	if len(got) != 3 {
		t.Fatalf("expected repo plus 2 nested worktrees, got %d: %+v", len(got), got)
	}
	if got[0].Name != "budget-usage-proxy" || got[0].DetectedType != "go" {
		t.Errorf("budget-usage-proxy candidate = %+v", got[0])
	}
	if got[1].Name != "material-tag-proxy" || got[1].DetectedType != "node-pnpm" {
		t.Errorf("material-tag-proxy candidate = %+v", got[1])
	}
	if got[2].Name != "repo" || got[2].DetectedType != "go" {
		t.Errorf("repo candidate = %+v", got[2])
	}
}
