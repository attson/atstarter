package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"atstarter/internal/control"
	"atstarter/internal/store"
)

func TestResolveProjectFromRejectsAmbiguousNames(t *testing.T) {
	projects := []store.Project{
		{ID: "p1", Name: "api"},
		{ID: "p2", Name: "API"},
	}

	if _, err := resolveProjectFrom(projects, "api"); err == nil {
		t.Fatal("expected ambiguous project error")
	} else {
		var remote control.RemoteError
		if !errors.As(err, &remote) || remote.Code != "ambiguous_project" {
			t.Fatalf("error = %v, want ambiguous_project", err)
		}
	}

	got, err := resolveProjectFrom(projects, "p2")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "p2" {
		t.Fatalf("resolved ID = %q, want p2", got.ID)
	}
}

func TestResolveCommandDefaultsAndDetectsAmbiguousNames(t *testing.T) {
	project := store.Project{
		ID: "p1",
		Commands: []store.LaunchCommand{
			{ID: "default", Name: "Dev", Command: "npm", IsDefault: true},
			{ID: "serve", Name: "dev", Command: "go"},
		},
	}

	got, err := resolveCommand(project, "")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != store.DefaultCommandID {
		t.Fatalf("default command ID = %q", got.ID)
	}

	if _, err := resolveCommand(project, "dev"); err == nil {
		t.Fatal("expected ambiguous command error")
	} else {
		var remote control.RemoteError
		if !errors.As(err, &remote) || remote.Code != "ambiguous_command" {
			t.Fatalf("error = %v, want ambiguous_command", err)
		}
	}
}

func TestControlServerWritesStateAndRequiresToken(t *testing.T) {
	app := newTestApp(t)
	statePath := filepath.Join(t.TempDir(), "control.json")
	srv, err := startControlServer(app, statePath)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	state, err := control.ReadState(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if state.URL == "" || state.Token == "" || state.PID != os.Getpid() {
		t.Fatalf("state = %+v", state)
	}

	client := control.Client{StatePath: statePath, HTTP: &http.Client{Timeout: time.Second}}
	var ping map[string]interface{}
	if err := client.Call("app.ping", nil, &ping); err != nil {
		t.Fatal(err)
	}
	if ping["version"] == "" {
		t.Fatalf("ping = %#v", ping)
	}

	body := bytes.NewBufferString(`{"method":"app.ping"}`)
	req, err := http.NewRequest(http.MethodPost, state.URL+"/rpc", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer wrong-token")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var cr control.Response
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		t.Fatal(err)
	}
	if cr.OK || cr.Error == nil || cr.Error.Code != "unauthorized" {
		t.Fatalf("unauthorized response = %+v", cr)
	}

	srv.Close()
	if _, err := os.Stat(statePath); !os.IsNotExist(err) {
		t.Fatalf("state file should be removed on close, stat err = %v", err)
	}
}

func TestRunCLIProjectListUsesControlServer(t *testing.T) {
	app := newTestApp(t)
	if err := app.store.Add(store.Project{ID: "p1", Name: "api", Path: "/tmp/api"}); err != nil {
		t.Fatal(err)
	}
	srv, err := startControlServer(app, controlStatePath(app.configPath))
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := runCLIWithIO([]string{"--config", app.configPath, "project", "list"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	if !strings.Contains(stdout.String(), `"ok":true`) || !strings.Contains(stdout.String(), `"api"`) {
		t.Fatalf("stdout = %s", stdout.String())
	}
}

func TestRunCLIScanAddsDetectedProjects(t *testing.T) {
	app := newTestApp(t)
	root := t.TempDir()
	projectDir := filepath.Join(root, "svc")
	writeFile(t, filepath.Join(projectDir, "go.mod"), "module svc\n")
	writeFile(t, filepath.Join(projectDir, "main.go"), "package main\nfunc main(){}\n")
	srv, err := startControlServer(app, controlStatePath(app.configPath))
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := runCLIWithIO([]string{"--config", app.configPath, "scan", root, "--add"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	projects, err := app.ListProjects()
	if err != nil {
		t.Fatal(err)
	}
	if len(projects) != 1 || projects[0].Name != "svc" {
		t.Fatalf("projects = %+v", projects)
	}
}

func TestRunCLIGroupCreateAndAddItem(t *testing.T) {
	app := newTestApp(t)
	if err := app.store.Add(store.Project{
		ID:      "ignored",
		Name:    "api",
		Path:    "/tmp/api",
		Command: "go",
		Args:    []string{"run", "."},
	}); err != nil {
		t.Fatal(err)
	}
	srv, err := startControlServer(app, controlStatePath(app.configPath))
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := runCLIWithIO([]string{"--config", app.configPath, "group", "create", "dev", "--item", "api:default"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("create exit = %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	groups, err := app.ListGroups()
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || groups[0].Name != "dev" || len(groups[0].Items) != 1 {
		t.Fatalf("groups after create = %+v", groups)
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLIWithIO([]string{"--config", app.configPath, "group", "update", "dev", "--name", "local dev"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("update exit = %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	groups, err = app.ListGroups()
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || groups[0].Name != "local dev" || len(groups[0].Items) != 1 {
		t.Fatalf("groups after rename = %+v", groups)
	}

	stdout.Reset()
	stderr.Reset()
	code = runCLIWithIO([]string{"--config", app.configPath, "group", "remove-item", "local dev", "--item", "api:default"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("remove-item exit = %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	groups, err = app.ListGroups()
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) != 1 || len(groups[0].Items) != 0 {
		t.Fatalf("groups after remove item = %+v", groups)
	}
}

func TestRunCLIProjectSwitchTypeUsesDetectionOptions(t *testing.T) {
	app := newTestApp(t)
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "docker-compose.yml"), "services: {}\n")
	writeFile(t, filepath.Join(dir, "go.mod"), "module svc\n")
	writeFile(t, filepath.Join(dir, "main.go"), "package main\nfunc main(){}\n")
	p, err := app.AddProject(dir)
	if err != nil {
		t.Fatal(err)
	}
	if p.DetectedType != "compose" {
		t.Fatalf("initial detectedType = %q, want compose", p.DetectedType)
	}
	srv, err := startControlServer(app, controlStatePath(app.configPath))
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	var stdout, stderr bytes.Buffer
	code := runCLIWithIO([]string{"--config", app.configPath, "project", "switch-type", p.Name, "--type", "go"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit = %d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
	projects, err := app.ListProjects()
	if err != nil {
		t.Fatal(err)
	}
	if projects[0].DetectedType != "go" || projects[0].Command != "go" || len(projects[0].Commands) != 1 {
		t.Fatalf("switched project = %+v", projects[0])
	}
}
