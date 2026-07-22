package main

import (
	"context"
	"testing"

	"atstarter/internal/docker"
)

func fakeDockerApp(t *testing.T, routes map[string]docker.ExecResult) *App {
	t.Helper()
	app := newTestApp(t)
	app.docker = docker.NewWithExecForTest(func(ctx context.Context, name string, args ...string) docker.ExecResult {
		key := name
		for _, a := range args {
			key += " " + a
		}
		for prefix, res := range routes {
			if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
				return res
			}
		}
		return docker.ExecResult{}
	})
	return app
}

func TestDockerAvailable(t *testing.T) {
	app := fakeDockerApp(t, map[string]docker.ExecResult{
		"docker version": {Stdout: "27.0.3"},
	})
	info := app.DockerAvailable()
	if !info.Available {
		t.Fatalf("Available = false; reason=%q", info.Reason)
	}
}
