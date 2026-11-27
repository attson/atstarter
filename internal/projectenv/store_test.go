package projectenv

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreLoadAbsentReturnsRevisionWithoutCreatingFile(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	store := NewStore(configPath)
	doc, err := store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if doc.Config.Version != 1 || doc.Revision == "" {
		t.Fatalf("document = %+v", doc)
	}
	if _, err := os.Stat(PathsForConfig(configPath).Config); !os.IsNotExist(err) {
		t.Fatalf("absent load created file or returned unexpected stat error: %v", err)
	}
}

func TestStoreRejectsUnknownFieldsAndPreservesAuthoritativeFile(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	path := PathsForConfig(configPath).Config
	input := []byte(`{"version":1,"defaults":{},"presets":[],"rules":[],"future":true}`)
	if err := os.WriteFile(path, input, 0o600); err != nil {
		t.Fatal(err)
	}
	store := NewStore(configPath)
	if _, err := store.Load(context.Background()); !IsCode(err, ErrInvalidEnvironment) {
		t.Fatalf("err = %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(input) {
		t.Fatal("invalid authoritative file was modified")
	}
}

func TestStoreRejectsTrailingJSON(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	path := PathsForConfig(configPath).Config
	if err := os.WriteFile(path, []byte(`{"version":1,"defaults":{},"presets":[],"rules":[]} {}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := NewStore(configPath).Load(context.Background()); !IsCode(err, ErrInvalidEnvironment) {
		t.Fatalf("err = %v", err)
	}
}

func TestStoreUpdateIsAtomicAndReloads(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	store := NewStore(configPath)
	first, err := store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	updated, err := store.Update(context.Background(), first.Revision, func(cfg *Config) error {
		cfg.Presets = append(cfg.Presets, Preset{ID: "personal", Name: "Personal"})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Revision == first.Revision {
		t.Fatal("revision did not change")
	}
	reloaded, err := NewStore(configPath).Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Config.Presets) != 1 || reloaded.Config.Presets[0].ID != "personal" {
		t.Fatalf("reloaded = %+v", reloaded)
	}
	if _, err := os.Stat(PathsForConfig(configPath).Config + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("temporary file remains: %v", err)
	}
}

func TestStoreUpdateRejectsStaleRevision(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	store := NewStore(configPath)
	first, err := store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(context.Background(), first.Revision, func(cfg *Config) error {
		cfg.Presets = append(cfg.Presets, Preset{ID: "current", Name: "Current"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	_, err = store.Update(context.Background(), first.Revision, func(cfg *Config) error {
		cfg.Presets = append(cfg.Presets, Preset{ID: "stale", Name: "Stale"})
		return nil
	})
	if !IsCode(err, ErrEnvironmentStale) {
		t.Fatalf("err = %v", err)
	}
	doc, err := store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Config.Presets) != 1 || doc.Config.Presets[0].ID != "current" {
		t.Fatalf("stale update changed config: %+v", doc.Config.Presets)
	}
}

func TestStoreFirstUpdateDetectsConcurrentFileCreation(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	store := NewStore(configPath)
	absent, err := store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	path := PathsForConfig(configPath).Config
	if err := os.WriteFile(path, []byte("{\n  \"version\": 1,\n  \"defaults\": {},\n  \"presets\": [],\n  \"rules\": []\n}"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = store.Update(context.Background(), absent.Revision, func(cfg *Config) error { return nil })
	if !IsCode(err, ErrEnvironmentStale) {
		t.Fatalf("err = %v", err)
	}
}
