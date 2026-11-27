package gitadapter

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"rootyard/internal/projectenv"
)

func TestInspectRecordsAbsentSingleRepeatedAndConflictingValues(t *testing.T) {
	repo := newGitRepository(t)
	runGit(t, repo, "config", "--local", "user.name", "Ada")
	runGit(t, repo, "config", "--local", "--add", "user.email", "one@example.com")
	runGit(t, repo, "config", "--local", "--add", "user.email", "one@example.com")
	runGit(t, repo, "config", "--local", "--add", "user.signingKey", "first")
	runGit(t, repo, "config", "--local", "--add", "user.signingKey", "second")
	adapter := New()
	target := identifyTarget(t, adapter, repo)

	snapshot, err := adapter.Inspect(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Fingerprint == "" {
		t.Fatal("missing fingerprint")
	}
	if len(snapshot.Values["commit.gpgSign"]) != 0 {
		t.Fatalf("absent values = %+v", snapshot.Values["commit.gpgSign"])
	}
	if got := snapshot.Values["user.name"]; len(got) != 1 || got[0].Value != "Ada" || !got[0].MainConfig {
		t.Fatalf("single value = %+v", got)
	}
	if got := snapshot.Values["user.email"]; len(got) != 2 || got[0].Value != got[1].Value {
		t.Fatalf("repeated values = %+v", got)
	}
	if got := snapshot.Values["user.signingKey"]; len(got) != 2 || got[0].Value == got[1].Value {
		t.Fatalf("conflicting values = %+v", got)
	}
}

func TestInspectFingerprintTracksMainConfigBytes(t *testing.T) {
	repo := newGitRepository(t)
	adapter := New()
	target := identifyTarget(t, adapter, repo)
	before, err := adapter.Inspect(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "config", "--local", "user.name", "Changed")
	after, err := adapter.Inspect(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	if before.Fingerprint == after.Fingerprint {
		t.Fatal("fingerprint did not change with config bytes")
	}
}

func TestInspectBlocksManagedValuesFromIncludedLocalConfig(t *testing.T) {
	repo := newGitRepository(t)
	adapter := New()
	target := identifyTarget(t, adapter, repo)
	included := filepath.Join(filepath.Dir(target.ConfigPath), "identity.inc")
	if err := os.WriteFile(included, []byte("[user]\n\temail = included@example.com\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, repo, "config", "--local", "include.path", "identity.inc")

	snapshot, err := adapter.Inspect(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	values := snapshot.Values["user.email"]
	if len(values) != 1 || values[0].MainConfig {
		t.Fatalf("included values = %+v", values)
	}
	found := false
	for _, issue := range snapshot.Issues {
		if issue.Code == projectenv.ErrExternalLocalOrigin && issue.Key == "user.email" && issue.Blocking {
			found = true
		}
	}
	if !found {
		t.Fatalf("issues = %+v", snapshot.Issues)
	}
}
