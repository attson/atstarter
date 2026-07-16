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
