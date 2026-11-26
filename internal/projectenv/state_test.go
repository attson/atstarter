package projectenv

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStateStoreLoadsAbsentAndRoundTrips(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	store := NewStateStore(configPath)
	state, err := store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if state.Version != 2 || len(state.Repositories) != 0 {
		t.Fatalf("state = %+v", state)
	}
	state.Repositories = append(state.Repositories, RepositoryRecord{
		ID: "repo", Aliases: []RepositoryAlias{{Path: "/work/repo", RuleIDs: []string{"rule"}}},
	})
	if err := store.Save(context.Background(), state); err != nil {
		t.Fatal(err)
	}
	reloaded, err := NewStateStore(configPath).Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Repositories) != 1 || reloaded.Repositories[0].ID != "repo" {
		t.Fatalf("reloaded = %+v", reloaded)
	}
}

func TestStateStoreQuarantinesCorruptState(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	path := PathsForConfig(configPath).State
	if err := os.WriteFile(path, []byte("not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	state, err := NewStateStore(configPath).Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if state.Version != 2 {
		t.Fatalf("state = %+v", state)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "environment-state.json.corrupt-") {
			found = true
		}
	}
	if !found {
		t.Fatalf("corrupt state was not quarantined: %+v", entries)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("corrupt state remains at active path: %v", err)
	}
}

func TestStateStoreMigratesVersionOneRepositoryTargetToGitAdapter(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	path := PathsForConfig(configPath).State
	legacy := `{
  "version": 1,
  "ruleScans": [{"ruleId":"rule","lastScanAt":"2026-11-26T10:00:00Z"}],
  "repositories": [{
    "id": "/work/repo",
    "targetId": "/work/repo/.git",
    "aliases": [{"path":"/work/repo","ruleIds":["rule"]}],
    "matchingRuleIds": ["rule"],
    "fingerprint": "before",
    "status": "blocked",
    "lastApplyAt": "2026-11-26T11:00:00Z",
    "lastApplyResult": "stale",
    "blockers": [{"code":"stale_plan","message":"changed","blocking":true}]
  }]
}`
	if err := os.WriteFile(path, []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}

	state, err := NewStateStore(configPath).Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if state.Version != 2 || len(state.RuleScans) != 1 || len(state.Repositories) != 1 {
		t.Fatalf("state = %+v", state)
	}
	record := state.Repositories[0]
	gitState, ok := record.Adapters["git"]
	if !ok || gitState.TargetID != "/work/repo/.git" || gitState.Fingerprint != "before" || gitState.LastApplyResult != "stale" {
		t.Fatalf("git adapter state = %+v", gitState)
	}
	if record.Status != "blocked" || len(gitState.Blockers) != 1 || record.Aliases[0].RuleIDs[0] != "rule" {
		t.Fatalf("record = %+v", record)
	}
}
