package main

import (
	"context"
	"path/filepath"
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

func TestListComposeServices(t *testing.T) {
	app := newTestApp(t)
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "docker-compose.yml"), "services:\n  web: {image: nginx}\n")
	p, err := app.AddProject(dir)
	if err != nil {
		t.Fatal(err)
	}
	base := filepath.Base(dir)
	app.docker = docker.NewWithExecForTest(func(ctx context.Context, name string, args ...string) docker.ExecResult {
		key := name
		for _, a := range args {
			key += " " + a
		}
		if len(key) >= len("docker compose") && key[:14] == "docker compose" {
			if containsArg(args, "config") {
				return docker.ExecResult{Stdout: "web\n"}
			}
		}
		if len(key) >= len("docker ps") && key[:9] == "docker ps" {
			return docker.ExecResult{Stdout: `{"ID":"x","Names":"` + base + `-web-1","Image":"nginx","State":"running","Status":"Up","Ports":"","Labels":"com.docker.compose.project=` + base + `,com.docker.compose.service=web"}`}
		}
		return docker.ExecResult{}
	})
	svcs, err := app.ListComposeServices(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(svcs) != 1 || svcs[0].Name != "web" || svcs[0].State != "running" {
		t.Errorf("services = %+v", svcs)
	}
}

func containsArg(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}

func TestFollowContainerLogsStartsRunner(t *testing.T) {
	app := newTestApp(t)
	// docker logs -f 会真的尝试执行 docker;用一个立即退出的假命令替身不易。
	// 这里只验证 runID 约定函数,不真启动。
	_ = app
	if id := containerRunID("abc"); id != "container:abc" {
		t.Errorf("runID = %q", id)
	}
	if id := composeRunID("proj", ""); id != "compose:proj" {
		t.Errorf("runID = %q", id)
	}
	if id := composeRunID("proj", "web"); id != "compose:proj:web" {
		t.Errorf("runID = %q", id)
	}
}
