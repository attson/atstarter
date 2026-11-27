package codexadapter

import (
	"context"
	"errors"
	"testing"

	"rootyard/internal/projectenv"
)

func TestDoctorWarnsWhenCodexExecutableIsUnavailableWithoutBlocking(t *testing.T) {
	repo := makeRepository(t)
	adapter := New()
	lookups := 0
	adapter.lookPath = func(name string) (string, error) {
		lookups++
		if name != "codex" {
			t.Fatalf("lookup name = %q", name)
		}
		return "", errors.New("missing")
	}
	target, _ := adapter.Identify(context.Background(), repo)
	issues := adapter.Doctor(context.Background(), target, projectenv.ResolvedEnvironment{})
	if lookups != 1 || len(issues) != 1 || issues[0].Code != projectenv.ErrCodexUnavailable || issues[0].Blocking {
		t.Fatalf("lookups = %d, issues = %+v", lookups, issues)
	}
}

func TestDoctorBlocksInvalidProjectTOMLAndWarnsForDangerousPermissions(t *testing.T) {
	t.Run("invalid", func(t *testing.T) {
		repo := makeRepository(t)
		writeCodexConfig(t, repo, "model = [\n")
		adapter := New()
		adapter.lookPath = func(string) (string, error) { return "/usr/bin/codex", nil }
		target, _ := adapter.Identify(context.Background(), repo)
		issues := adapter.Doctor(context.Background(), target, projectenv.ResolvedEnvironment{})
		if len(issues) != 1 || issues[0].Code != projectenv.ErrInvalidCodexConfig || !issues[0].Blocking {
			t.Fatalf("issues = %+v", issues)
		}
	})

	t.Run("dangerous", func(t *testing.T) {
		repo := makeRepository(t)
		adapter := New()
		adapter.lookPath = func(string) (string, error) { return "/usr/bin/codex", nil }
		target, _ := adapter.Identify(context.Background(), repo)
		resolved := projectenv.ResolvedEnvironment{Codex: map[string]projectenv.ResolvedField{
			"sandbox_mode":    {Intent: projectenv.FieldIntent{Mode: projectenv.ModeSet, Value: "danger-full-access"}},
			"approval_policy": {Intent: projectenv.FieldIntent{Mode: projectenv.ModeSet, Value: "never"}},
		}}
		issues := adapter.Doctor(context.Background(), target, resolved)
		if len(issues) != 1 || issues[0].Code != projectenv.ErrDangerousCodexPermissions || issues[0].Blocking {
			t.Fatalf("issues = %+v", issues)
		}
	})
}

func TestDoctorWarnsForMixedCurrentAndManagedDangerousPermissions(t *testing.T) {
	repo := makeRepository(t)
	writeCodexConfig(t, repo, "sandbox_mode = \"danger-full-access\"\n")
	adapter := New()
	adapter.lookPath = func(string) (string, error) { return "/usr/bin/codex", nil }
	target, _ := adapter.Identify(context.Background(), repo)
	resolved := projectenv.ResolvedEnvironment{Codex: map[string]projectenv.ResolvedField{
		"approval_policy": {Intent: projectenv.FieldIntent{Mode: projectenv.ModeSet, Value: "never"}},
	}}

	issues := adapter.Doctor(context.Background(), target, resolved)
	if len(issues) != 1 || issues[0].Code != projectenv.ErrDangerousCodexPermissions || issues[0].Blocking {
		t.Fatalf("issues = %+v", issues)
	}
}
