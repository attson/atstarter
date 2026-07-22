package docker

import (
	"context"
	"testing"
)

// fakeExec 按 (name+args) 前缀匹配预设结果。
func fakeExec(routes map[string]execResult) execFunc {
	return func(ctx context.Context, name string, args ...string) execResult {
		key := name
		for _, a := range args {
			key += " " + a
		}
		for prefix, res := range routes {
			if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
				return res
			}
		}
		return execResult{Err: context.Canceled}
	}
}

func TestDetectAvailable(t *testing.T) {
	c := newWithExec(fakeExec(map[string]execResult{
		"docker version": {Stdout: "Docker version 27.0.3, build abc"},
	}))
	info := c.Detect(context.Background())
	if !info.Available {
		t.Fatalf("Available = false, want true; reason=%q", info.Reason)
	}
	if info.Version == "" {
		t.Errorf("Version empty")
	}
}

func TestDetectDaemonDown(t *testing.T) {
	c := newWithExec(fakeExec(map[string]execResult{
		"docker version": {Stderr: "Cannot connect to the Docker daemon. Is the docker daemon running?", ExitCode: 1},
	}))
	info := c.Detect(context.Background())
	if info.Available {
		t.Fatalf("Available = true, want false")
	}
	if info.Reason != "docker daemon 未运行" {
		t.Errorf("Reason = %q", info.Reason)
	}
}
