package main

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	return NewAppWithConfig(cfgPath)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestAppAddProjectDetectsCommand(t *testing.T) {
	app := newTestApp(t)
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "package.json"), `{"scripts":{"dev":"vite"}}`)
	writeFile(t, filepath.Join(dir, "pnpm-lock.yaml"), "")

	p, err := app.AddProject(dir)
	if err != nil {
		t.Fatal(err)
	}
	if p.Command != "pnpm" {
		t.Errorf("Command = %q, want pnpm", p.Command)
	}
	if p.DetectedType != "node-pnpm" {
		t.Errorf("DetectedType = %q", p.DetectedType)
	}

	list, _ := app.ListProjects()
	if len(list) != 1 {
		t.Errorf("expected 1 persisted project, got %d", len(list))
	}
}

func TestAppUpdateProjectCommandFromLine(t *testing.T) {
	app := newTestApp(t)
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module x\n")
	writeFile(t, filepath.Join(dir, "main.go"), "package main\nfunc main(){}\n")
	p, _ := app.AddProject(dir)

	updated, err := app.UpdateProjectCommand(p.ID, "/home/attson/sdk/go1.23.12/bin/go run main.go serve")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Command != "/home/attson/sdk/go1.23.12/bin/go" {
		t.Errorf("Command = %q", updated.Command)
	}
	wantArgs := []string{"run", "main.go", "serve"}
	if len(updated.Args) != len(wantArgs) {
		t.Fatalf("Args = %v", updated.Args)
	}
	if updated.AutoDetected != false {
		t.Errorf("AutoDetected should be false after manual edit")
	}
}

func TestAppAddProjectNormalizesPath(t *testing.T) {
	app := newTestApp(t)
	dir := t.TempDir() // 绝对路径
	writeFile(t, filepath.Join(dir, "go.mod"), "module x\n")
	writeFile(t, filepath.Join(dir, "main.go"), "package main\nfunc main(){}\n")

	// 第一次:干净绝对路径
	p1, err := app.AddProject(dir)
	if err != nil {
		t.Fatal(err)
	}
	// 第二次:同一目录的非规范写法(末尾加 /. )
	p2, err := app.AddProject(dir + string(filepath.Separator) + ".")
	if err != nil {
		t.Fatal(err)
	}

	// 两次应生成相同 ID(去重)
	if p1.ID != p2.ID {
		t.Errorf("same dir should yield same ID: %q vs %q", p1.ID, p2.ID)
	}
	// 存储里应只有 1 条
	list, _ := app.ListProjects()
	if len(list) != 1 {
		t.Fatalf("expected 1 project after adding same dir twice, got %d", len(list))
	}
	// 存储的 Path 应是干净的绝对路径(等于 filepath.Clean(dir))
	if list[0].Path != filepath.Clean(dir) {
		t.Errorf("stored Path = %q, want cleaned abs %q", list[0].Path, filepath.Clean(dir))
	}
}
