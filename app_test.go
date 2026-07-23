package main

import (
	"os"
	"path/filepath"
	"testing"

	"atstarter/internal/store"
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

func chdir(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(old); err != nil {
			t.Fatal(err)
		}
	})
}

func TestNewAppUsesDevConfigInWorkingDirectory(t *testing.T) {
	oldVersion := Version
	Version = "dev"
	t.Cleanup(func() { Version = oldVersion })

	dir := t.TempDir()
	chdir(t, dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))

	app := NewApp()
	if err := app.SetWorkspaces([]string{"/tmp/workspace"}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".dev")); err != nil {
		t.Fatalf("expected dev config at .dev: %v", err)
	}
}

func TestNewAppUsesUserConfigOutsideDev(t *testing.T) {
	oldVersion := Version
	Version = "v1.2.3"
	t.Cleanup(func() { Version = oldVersion })

	dir := t.TempDir()
	configHome := filepath.Join(t.TempDir(), "config")
	chdir(t, dir)
	t.Setenv("XDG_CONFIG_HOME", configHome)

	app := NewApp()
	if err := app.SetWorkspaces([]string{"/tmp/workspace"}); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(dir, ".dev")); !os.IsNotExist(err) {
		t.Fatalf("expected no .dev config outside dev, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(configHome, "atstarter", "config.json")); err != nil {
		t.Fatalf("expected user config outside dev: %v", err)
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

func TestAppResetProjectsClearsProjectsAndGroupsButKeepsWorkspaces(t *testing.T) {
	app := newTestApp(t)
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module x\n")
	writeFile(t, filepath.Join(dir, "main.go"), "package main\nfunc main(){}\n")

	p, err := app.AddProject(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.SetWorkspaces([]string{"/workspace"}); err != nil {
		t.Fatal(err)
	}
	if _, err := app.SaveGroup(store.LaunchGroup{Name: "stack", Items: []store.GroupItem{{ProjectID: p.ID, CommandID: store.DefaultCommandID}}}); err != nil {
		t.Fatal(err)
	}

	if err := app.ResetProjects(); err != nil {
		t.Fatal(err)
	}

	projects, err := app.ListProjects()
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 0 {
		t.Fatalf("expected projects cleared, got %+v", projects)
	}
	groups, err := app.ListGroups()
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 0 {
		t.Fatalf("expected groups cleared, got %+v", groups)
	}
	workspaces, err := app.GetWorkspaces()
	if err != nil {
		t.Fatal(err)
	}
	if len(workspaces) != 1 || workspaces[0] != "/workspace" {
		t.Fatalf("expected workspaces preserved, got %+v", workspaces)
	}
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home dir")
	}
	cases := []struct {
		in   string
		want string
	}{
		{"~", home},
		{"~/GolandProjects", filepath.Join(home, "GolandProjects")},
		{"~/a/b", filepath.Join(home, "a", "b")},
		{"/absolute/path", "/absolute/path"}, // 非 ~ 前缀原样返回
		{"relative/x", "relative/x"},
		{"~notme/x", "~notme/x"}, // ~ 后非 / 的不展开(不是当前用户家目录写法)
		{"", ""},
	}
	for _, c := range cases {
		if got := expandHome(c.in); got != c.want {
			t.Errorf("expandHome(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestScanWorkspacesExpandsHome(t *testing.T) {
	// 在临时"家目录"下造一个含 go 项目的工作区,用 ~ 路径扫描应能找到。
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeFile(t, filepath.Join(home, "ws", "proj", "go.mod"), "module x\n")
	writeFile(t, filepath.Join(home, "ws", "proj", "main.go"), "package main\nfunc main(){}\n")

	app := newTestApp(t)
	got := app.ScanWorkspaces([]string{"~/ws"})
	if len(got) != 1 {
		t.Fatalf("expected 1 candidate under ~/ws, got %d: %+v", len(got), got)
	}
	if got[0].Name != "proj" || got[0].DetectedType != "go" {
		t.Errorf("candidate = %+v", got[0])
	}
}

func TestListProjectsAddsDetectionOptionsForSavedComposeProject(t *testing.T) {
	app := newTestApp(t)
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "docker-compose.yml"), "services: {}\n")
	writeFile(t, filepath.Join(dir, "go.mod"), "module x\n")
	writeFile(t, filepath.Join(dir, "main.go"), "package main\nfunc main(){}\n")

	p, err := app.AddProject(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.DetectionOptions) != 2 {
		t.Fatalf("AddProject DetectionOptions = %+v, want compose and go", p.DetectionOptions)
	}

	list, err := app.ListProjects()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("ListProjects len = %d, want 1", len(list))
	}
	if len(list[0].DetectionOptions) != 2 {
		t.Fatalf("ListProjects DetectionOptions = %+v, want compose and go", list[0].DetectionOptions)
	}
	if list[0].DetectionOptions[1].Type != "go" {
		t.Fatalf("fallback option = %+v, want go", list[0].DetectionOptions[1])
	}
}

func TestAppUpdateProjectCommands(t *testing.T) {
	app := newTestApp(t)
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module x\n")
	writeFile(t, filepath.Join(dir, "main.go"), "package main\nfunc main(){}\n")
	p, _ := app.AddProject(dir)

	updated, err := app.UpdateProjectCommands(p.ID, "api", []CommandInput{
		{Name: "Serve", Line: "go run main.go serve", IsDefault: true},
		{Name: "Worker", Line: "go run main.go worker", Cwd: dir},
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "api" {
		t.Errorf("Name = %q, want api", updated.Name)
	}
	if len(updated.Commands) != 2 {
		t.Fatalf("Commands = %+v", updated.Commands)
	}
	if updated.Commands[0].ID == "" || updated.Commands[1].ID == "" || updated.Commands[0].ID == updated.Commands[1].ID {
		t.Fatalf("commands should have distinct IDs: %+v", updated.Commands)
	}
	if updated.Command != "go" || len(updated.Args) != 3 || updated.Args[2] != "serve" {
		t.Fatalf("legacy default fields not updated: %+v", updated)
	}
}

func TestRunIDForCommand(t *testing.T) {
	if got := runIDForCommand("p1", "serve"); got != "p1:serve" {
		t.Fatalf("runIDForCommand = %q", got)
	}
}
