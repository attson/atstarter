package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	return New(path)
}

func TestLoadMissingFileReturnsEmptyConfig(t *testing.T) {
	s := newTestStore(t)
	cfg, err := s.Load()
	if err != nil {
		t.Fatalf("Load on missing file should not error: %v", err)
	}
	if cfg.Version != 1 {
		t.Errorf("Version = %d, want 1", cfg.Version)
	}
	if len(cfg.Projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(cfg.Projects))
	}
}

func TestAddAndReload(t *testing.T) {
	s := newTestStore(t)
	p := Project{Name: "toolkit", Path: "/x/ad-ai-toolkit", Command: "go", Args: []string{"run", "main.go"}}
	if err := s.Add(p); err != nil {
		t.Fatal(err)
	}
	// 新实例从磁盘重读,验证持久化。
	s2 := New(s.path)
	cfg, err := s2.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(cfg.Projects))
	}
	if cfg.Projects[0].ID == "" {
		t.Error("Add should assign an ID")
	}
	if cfg.Projects[0].Name != "toolkit" {
		t.Errorf("Name = %q", cfg.Projects[0].Name)
	}
}

func TestAddDedupBySamePath(t *testing.T) {
	s := newTestStore(t)
	_ = s.Add(Project{Name: "a", Path: "/x/proj"})
	_ = s.Add(Project{Name: "a-again", Path: "/x/proj"})
	cfg, _ := s.Load()
	if len(cfg.Projects) != 1 {
		t.Fatalf("expected dedup to 1, got %d", len(cfg.Projects))
	}
}

func TestUpdate(t *testing.T) {
	s := newTestStore(t)
	_ = s.Add(Project{Name: "a", Path: "/x/proj", Command: "go"})
	cfg, _ := s.Load()
	p := cfg.Projects[0]
	p.Command = "pnpm"
	p.AutoDetected = false
	if err := s.Update(p); err != nil {
		t.Fatal(err)
	}
	cfg2, _ := s.Load()
	if cfg2.Projects[0].Command != "pnpm" {
		t.Errorf("Command = %q, want pnpm", cfg2.Projects[0].Command)
	}
}

func TestRemove(t *testing.T) {
	s := newTestStore(t)
	_ = s.Add(Project{Name: "a", Path: "/x/proj"})
	cfg, _ := s.Load()
	id := cfg.Projects[0].ID
	if err := s.Remove(id); err != nil {
		t.Fatal(err)
	}
	cfg2, _ := s.Load()
	if len(cfg2.Projects) != 0 {
		t.Errorf("expected 0 after remove, got %d", len(cfg2.Projects))
	}
}

func TestResetProjectsClearsProjectsAndGroupsButKeepsWorkspaces(t *testing.T) {
	s := newTestStore(t)
	if err := s.SetWorkspaces([]string{"/workspace"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Add(Project{Name: "a", Path: "/x/proj"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveGroup(LaunchGroup{Name: "stack", Items: []GroupItem{{ProjectID: IDForPath("/x/proj"), CommandID: DefaultCommandID}}}); err != nil {
		t.Fatal(err)
	}

	if err := s.ResetProjects(); err != nil {
		t.Fatal(err)
	}

	cfg, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Projects) != 0 {
		t.Fatalf("expected projects cleared, got %+v", cfg.Projects)
	}
	if len(cfg.Groups) != 0 {
		t.Fatalf("expected groups cleared, got %+v", cfg.Groups)
	}
	if len(cfg.Workspaces) != 1 || cfg.Workspaces[0] != "/workspace" {
		t.Fatalf("expected workspaces preserved, got %+v", cfg.Workspaces)
	}
}

func TestLoadCorruptJSONReturnsError(t *testing.T) {
	s := newTestStore(t)
	if err := os.WriteFile(s.path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load(); err == nil {
		t.Error("expected error loading corrupt json")
	}
}

func TestLoadMigratesLegacyProjectCommand(t *testing.T) {
	s := newTestStore(t)
	legacy := Config{
		Version: 1,
		Projects: []Project{{
			ID: "p1", Name: "api", Path: "/x/api",
			Command: "go", Args: []string{"run", "main.go"}, Cwd: "/x/api", Env: map[string]string{"A": "B"},
		}},
	}
	b, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.path, b, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := s.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Projects[0].Commands) != 1 {
		t.Fatalf("expected migrated default command, got %+v", cfg.Projects[0].Commands)
	}
	cmd := cfg.Projects[0].Commands[0]
	if cmd.ID != "default" || cmd.Command != "go" || !cmd.IsDefault {
		t.Fatalf("unexpected migrated command: %+v", cmd)
	}
}

func TestSaveUpdateRemoveGroup(t *testing.T) {
	s := newTestStore(t)
	g := LaunchGroup{
		Name:  "dev stack",
		Items: []GroupItem{{ProjectID: "p1", CommandID: "serve"}},
	}
	saved, err := s.SaveGroup(g)
	if err != nil {
		t.Fatal(err)
	}
	if saved.ID == "" {
		t.Fatal("SaveGroup should assign ID")
	}
	saved.Name = "local stack"
	saved.Items = append(saved.Items, GroupItem{ProjectID: "p2", CommandID: "dev"})
	if _, err := s.SaveGroup(saved); err != nil {
		t.Fatal(err)
	}
	cfg, _ := s.Load()
	if len(cfg.Groups) != 1 || cfg.Groups[0].Name != "local stack" || len(cfg.Groups[0].Items) != 2 {
		t.Fatalf("unexpected groups after update: %+v", cfg.Groups)
	}
	if err := s.RemoveGroup(saved.ID); err != nil {
		t.Fatal(err)
	}
	cfg, _ = s.Load()
	if len(cfg.Groups) != 0 {
		t.Fatalf("expected group removed, got %+v", cfg.Groups)
	}
}

func TestSetProjectPinned(t *testing.T) {
	s := newTestStore(t)
	if err := s.Add(Project{Name: "A", Path: "/tmp/a", Command: "go"}); err != nil {
		t.Fatal(err)
	}
	if err := s.Add(Project{Name: "B", Path: "/tmp/b", Command: "go"}); err != nil {
		t.Fatal(err)
	}
	idA, idB := IDForPath("/tmp/a"), IDForPath("/tmp/b")

	// 置顶 a → order 1
	if err := s.SetProjectPinned(idA, true); err != nil {
		t.Fatal(err)
	}
	// 置顶 b → order 2(追加到末尾)
	if err := s.SetProjectPinned(idB, true); err != nil {
		t.Fatal(err)
	}
	cfg, _ := s.Load()
	if got := pinOrder(cfg, idA); got != 1 {
		t.Fatalf("a PinnedOrder = %d, want 1", got)
	}
	if got := pinOrder(cfg, idB); got != 2 {
		t.Fatalf("b PinnedOrder = %d, want 2", got)
	}

	// 重复置顶 a 不改序号(幂等)
	if err := s.SetProjectPinned(idA, true); err != nil {
		t.Fatal(err)
	}
	cfg, _ = s.Load()
	if got := pinOrder(cfg, idA); got != 1 {
		t.Fatalf("a re-pin PinnedOrder = %d, want 1", got)
	}

	// 取消 a → 0
	if err := s.SetProjectPinned(idA, false); err != nil {
		t.Fatal(err)
	}
	cfg, _ = s.Load()
	if got := pinOrder(cfg, idA); got != 0 {
		t.Fatalf("a PinnedOrder after unpin = %d, want 0", got)
	}

	// id 不存在返回错误
	if err := s.SetProjectPinned("missing", true); err == nil {
		t.Fatal("SetProjectPinned(missing) should error")
	}
}

func TestReorderPinned(t *testing.T) {
	s := newTestStore(t)
	ids := map[string]string{}
	for _, name := range []string{"a", "b", "c"} {
		if err := s.Add(Project{Name: name, Path: "/tmp/" + name, Command: "go"}); err != nil {
			t.Fatal(err)
		}
		id := IDForPath("/tmp/" + name)
		ids[name] = id
		if err := s.SetProjectPinned(id, true); err != nil {
			t.Fatal(err)
		}
	}
	// 重排为 c, a, b → order 1,2,3
	if err := s.ReorderPinned([]string{ids["c"], ids["a"], ids["b"]}); err != nil {
		t.Fatal(err)
	}
	cfg, _ := s.Load()
	if pinOrder(cfg, ids["c"]) != 1 || pinOrder(cfg, ids["a"]) != 2 || pinOrder(cfg, ids["b"]) != 3 {
		t.Fatalf("reorder wrong: c=%d a=%d b=%d", pinOrder(cfg, ids["c"]), pinOrder(cfg, ids["a"]), pinOrder(cfg, ids["b"]))
	}
}

func pinOrder(cfg Config, id string) int {
	for _, p := range cfg.Projects {
		if p.ID == id {
			return p.PinnedOrder
		}
	}
	return -1
}
