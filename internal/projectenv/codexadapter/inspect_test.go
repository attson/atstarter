package codexadapter

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"rootyard/internal/projectenv"
)

func TestInspectTreatsMissingProjectConfigAsDistinctEmptySnapshot(t *testing.T) {
	repo := makeRepository(t)
	adapter := New()
	target, err := adapter.Identify(context.Background(), repo)
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := adapter.Inspect(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.Exists || snapshot.Fingerprint == "" || len(snapshot.Values) != 0 || snapshot.NativeContent != nil {
		t.Fatalf("snapshot = %+v", snapshot)
	}
}

func TestInspectReadsOnlyManagedRootStringsAndRetainsExactBytes(t *testing.T) {
	repo := makeRepository(t)
	fixture := "# keep heading\nmodel   = \"old-model\" # keep inline\ncustom = { token = \"secret\" }\n\n[features]\nmulti_agent = true # keep table comment\n"
	writeCodexConfig(t, repo, fixture)
	adapter := New()
	target, _ := adapter.Identify(context.Background(), repo)
	snapshot, err := adapter.Inspect(context.Background(), target)
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.Exists || !bytes.Equal(snapshot.NativeContent, []byte(fixture)) {
		t.Fatalf("snapshot did not retain native bytes: %+v", snapshot)
	}
	values := snapshot.Values["model"]
	if len(values) != 1 || values[0].Value != "old-model" || values[0].Origin != target.ConfigPath || !values[0].MainConfig {
		t.Fatalf("model values = %+v", values)
	}
	if _, exists := snapshot.Values["custom"]; exists {
		t.Fatalf("unmanaged value leaked into snapshot: %+v", snapshot.Values)
	}
}

func TestInspectRejectsInvalidOrUnsafeCodexConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    projectenv.ErrorCode
	}{
		{name: "syntax", content: "model = [\n", want: projectenv.ErrInvalidCodexConfig},
		{name: "duplicate", content: "model = \"one\"\nmodel = \"two\"\n", want: projectenv.ErrInvalidCodexConfig},
		{name: "wrong managed type", content: "web_search = true\n", want: projectenv.ErrInvalidCodexConfig},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := makeRepository(t)
			writeCodexConfig(t, repo, tt.content)
			adapter := New()
			target, _ := adapter.Identify(context.Background(), repo)
			_, err := adapter.Inspect(context.Background(), target)
			if !projectenv.IsCode(err, tt.want) {
				t.Fatalf("err = %v", err)
			}
			if strings.Contains(err.Error(), "secret") {
				t.Fatalf("error exposed unmanaged value: %v", err)
			}
		})
	}
}

func TestInspectRejectsCodexDirectoryAndConfigSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior differs on Windows")
	}
	t.Run("directory", func(t *testing.T) {
		repo := makeRepository(t)
		outside := t.TempDir()
		if err := os.Symlink(outside, filepath.Join(repo, ".codex")); err != nil {
			t.Fatal(err)
		}
		adapter := New()
		target, _ := adapter.Identify(context.Background(), repo)
		_, err := adapter.Inspect(context.Background(), target)
		if !projectenv.IsCode(err, projectenv.ErrUnsafeCodexConfigPath) {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("config", func(t *testing.T) {
		repo := makeRepository(t)
		dir := filepath.Join(repo, ".codex")
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		outside := filepath.Join(t.TempDir(), "config.toml")
		if err := os.WriteFile(outside, []byte("model = \"outside\"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(dir, "config.toml")); err != nil {
			t.Fatal(err)
		}
		adapter := New()
		target, _ := adapter.Identify(context.Background(), repo)
		_, err := adapter.Inspect(context.Background(), target)
		if !projectenv.IsCode(err, projectenv.ErrUnsafeCodexConfigPath) {
			t.Fatalf("err = %v", err)
		}
	})
}
