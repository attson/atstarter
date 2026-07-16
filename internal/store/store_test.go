package store

import (
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

func TestLoadCorruptJSONReturnsError(t *testing.T) {
	s := newTestStore(t)
	if err := os.WriteFile(s.path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load(); err == nil {
		t.Error("expected error loading corrupt json")
	}
}
