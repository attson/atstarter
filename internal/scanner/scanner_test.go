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
