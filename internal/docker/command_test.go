package docker

import (
	"context"
	"errors"
	"testing"
)

func TestResolveDockerCommandFallsBackToDockerDesktopPaths(t *testing.T) {
	got := resolveDockerCommandWith(
		func(name string) (string, error) { return "", errors.New("not found") },
		func(path string) bool { return path == "/usr/local/bin/docker" },
	)
	if got != "/usr/local/bin/docker" {
		t.Fatalf("resolveDockerCommandWith() = %q, want /usr/local/bin/docker", got)
	}
}

func TestResolveDockerCommandPrefersPATH(t *testing.T) {
	got := resolveDockerCommandWith(
		func(name string) (string, error) { return "/custom/bin/docker", nil },
		func(path string) bool { return true },
	)
	if got != "/custom/bin/docker" {
		t.Fatalf("resolveDockerCommandWith() = %q, want PATH result", got)
	}
}

func TestClientUsesResolvedDockerCommand(t *testing.T) {
	var gotName string
	c := NewWithCommandForTest("/usr/local/bin/docker", func(ctx context.Context, name string, args ...string) execResult {
		gotName = name
		return execResult{Stdout: "27.0.3\n"}
	})

	info := c.Detect(context.Background())
	if !info.Available {
		t.Fatalf("Available = false, want true; reason=%q", info.Reason)
	}
	if gotName != "/usr/local/bin/docker" {
		t.Fatalf("docker command = %q, want /usr/local/bin/docker", gotName)
	}
}
