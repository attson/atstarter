package docker

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestOrderServiceNamesUsesComposeDeclarationOrder(t *testing.T) {
	dir := t.TempDir()
	writeComposeFile(t, dir, "compose.yaml", `services:
  worker:
    image: worker
  api:
    image: api
  database:
    image: postgres
`)

	got := orderServiceNames(dir, "", []string{"database", "worker", "api"})
	want := []string{"worker", "api", "database"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("orderServiceNames() = %v, want %v", got, want)
	}
}

func TestListServiceNamesUsesComposeDeclarationOrder(t *testing.T) {
	dir := t.TempDir()
	writeComposeFile(t, dir, "compose.yaml", `services:
  worker:
    image: worker
  api:
    image: api
  database:
    image: postgres
`)
	c := newWithExec(func(context.Context, string, ...string) execResult {
		return execResult{Stdout: "api\ndatabase\nworker\n"}
	})

	got, err := c.ListServiceNames(context.Background(), dir, "")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"worker", "api", "database"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ListServiceNames() = %v, want %v", got, want)
	}
}

func TestOrderServiceNamesAppendsUnknownServicesStably(t *testing.T) {
	dir := t.TempDir()
	writeComposeFile(t, dir, "custom.yml", `services:
  web:
    image: nginx
`)

	got := orderServiceNames(dir, "custom.yml", []string{"worker", "web", "api", "web"})
	want := []string{"web", "api", "worker"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("orderServiceNames() = %v, want %v", got, want)
	}
}

func writeComposeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
