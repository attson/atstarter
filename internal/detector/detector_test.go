package detector

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// mkProject 在临时目录建一个项目,files 是 相对路径→内容 的映射。
func mkProject(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestDetect(t *testing.T) {
	pkg := func(scripts map[string]string) string {
		m := map[string]any{"name": "x", "scripts": scripts}
		b, _ := json.Marshal(m)
		return string(b)
	}
	cases := []struct {
		name     string
		files    map[string]string
		wantType string
		wantCmd  string
	}{
		{"pnpm", map[string]string{"package.json": pkg(map[string]string{"dev": "vite"}), "pnpm-lock.yaml": ""},
			"node-pnpm", "pnpm run dev"},
		{"yarn", map[string]string{"package.json": pkg(map[string]string{"dev": "vite"}), "yarn.lock": ""},
			"node-yarn", "yarn dev"},
		{"bun", map[string]string{"package.json": pkg(map[string]string{"dev": "vite"}), "bun.lockb": ""},
			"node-bun", "bun run dev"},
		{"npm default", map[string]string{"package.json": pkg(map[string]string{"dev": "vite"})},
			"node-npm", "npm run dev"},
		{"npm serve fallback", map[string]string{"package.json": pkg(map[string]string{"serve": "vue-cli-service serve"})},
			"node-npm", "npm run serve"},
		{"go root main", map[string]string{"go.mod": "module x\n", "main.go": "package main\nfunc main(){}\n"},
			"go", "go run main.go"},
		{"go cmd", map[string]string{"go.mod": "module x\n", "cmd/server/main.go": "package main\nfunc main(){}\n"},
			"go", "go run ./cmd/server"},
		{"rust", map[string]string{"Cargo.toml": "[package]\nname=\"x\"\n"},
			"rust", "cargo run"},
		{"django", map[string]string{"manage.py": "", "requirements.txt": ""},
			"python-django", "python manage.py runserver"},
		{"python main", map[string]string{"main.py": "", "requirements.txt": ""},
			"python", "python main.py"},
		{"compose yml", map[string]string{"docker-compose.yml": "services:\n  web:\n    image: nginx\n"},
			"compose", ""},
		{"compose yaml", map[string]string{"compose.yaml": "services: {}\n"},
			"compose", ""},
		{"compose over node", map[string]string{"package.json": pkg(map[string]string{"dev": "vite"}), "docker-compose.yml": "services: {}\n"},
			"compose", ""},
		{"unknown", map[string]string{"README.md": "hi"},
			"unknown", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := mkProject(t, c.files)
			res := Detect(dir)
			if res.Type != c.wantType {
				t.Errorf("Type = %q, want %q", res.Type, c.wantType)
			}
			if res.Command != c.wantCmd {
				t.Errorf("Command = %q, want %q", res.Command, c.wantCmd)
			}
		})
	}
}

func TestDetectOptionsIncludesComposeAndFallback(t *testing.T) {
	dir := mkProject(t, map[string]string{
		"docker-compose.yml": "services:\n  web:\n    image: nginx\n",
		"go.mod":             "module x\n",
		"main.go":            "package main\nfunc main(){}\n",
	})

	got := DetectOptions(dir)
	if len(got) != 2 {
		t.Fatalf("DetectOptions len = %d, want 2: %+v", len(got), got)
	}
	if got[0].Type != "compose" || got[0].Command != "" {
		t.Fatalf("first option = %+v, want compose with empty command", got[0])
	}
	if got[1].Type != "go" || got[1].Command != "go run main.go" {
		t.Fatalf("fallback option = %+v, want go run main.go", got[1])
	}
}
