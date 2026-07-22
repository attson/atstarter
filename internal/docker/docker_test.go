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

func TestListContainers(t *testing.T) {
	sample := `{"ID":"abc","Names":"redis","Image":"redis:7","State":"running","Status":"Up 1m","Ports":"","Labels":""}`
	c := newWithExec(fakeExec(map[string]execResult{
		"docker ps": {Stdout: sample},
	}))
	got, err := c.ListContainers(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "redis" {
		t.Errorf("got = %+v", got)
	}
}

func TestContainerLifecycleArgs(t *testing.T) {
	var gotArgs []string
	rec := func(ctx context.Context, name string, args ...string) execResult {
		gotArgs = append([]string{name}, args...)
		return execResult{}
	}
	c := newWithExec(rec)
	ctx := context.Background()

	c.StartContainer(ctx, "abc")
	if got := join(gotArgs); got != "docker start abc" {
		t.Errorf("start = %q", got)
	}
	c.StopContainer(ctx, "abc")
	if got := join(gotArgs); got != "docker stop abc" {
		t.Errorf("stop = %q", got)
	}
	c.RestartContainer(ctx, "abc")
	if got := join(gotArgs); got != "docker restart abc" {
		t.Errorf("restart = %q", got)
	}
	c.RemoveContainer(ctx, "abc", false)
	if got := join(gotArgs); got != "docker rm abc" {
		t.Errorf("rm = %q", got)
	}
	c.RemoveContainer(ctx, "abc", true)
	if got := join(gotArgs); got != "docker rm -f abc" {
		t.Errorf("rm -f = %q", got)
	}
}

func join(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += " "
		}
		out += p
	}
	return out
}
