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

func TestListContainersBinding(t *testing.T) {
	app := fakeDockerApp(t, map[string]docker.ExecResult{
		"docker ps": {Stdout: `{"ID":"abc","Names":"redis","Image":"redis:7","State":"running","Status":"Up 1m","Ports":"","Labels":""}`},
	})
	got, err := app.ListContainers()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "redis" {
		t.Errorf("got = %+v", got)
	}
}

func TestRemoveContainerForce(t *testing.T) {
	var gotKey string
	app := newTestApp(t)
	app.docker = docker.NewWithExecForTest(func(ctx context.Context, name string, args ...string) docker.ExecResult {
		gotKey = name
		for _, a := range args {
			gotKey += " " + a
		}
		return docker.ExecResult{}
	})
	if err := app.RemoveContainer("abc", true); err != nil {
		t.Fatal(err)
	}
	if gotKey != "docker rm -f abc" {
		t.Errorf("key = %q", gotKey)
	}
}
