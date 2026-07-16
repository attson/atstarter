package cmdparse

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		wantCmd  string
		wantArgs []string
		wantErr  bool
	}{
		{"simple", "pnpm run dev", "pnpm", []string{"run", "dev"}, false},
		{"go run", "go run main.go", "go", []string{"run", "main.go"}, false},
		{"go subcommand", "go run main.go serve", "go", []string{"run", "main.go", "serve"}, false},
		{"absolute runtime", "/home/attson/sdk/go1.23.12/bin/go run main.go serve",
			"/home/attson/sdk/go1.23.12/bin/go", []string{"run", "main.go", "serve"}, false},
		{"quoted arg", `node -e "console.log('hi there')"`, "node",
			[]string{"-e", "console.log('hi there')"}, false},
		{"trailing spaces", "  cargo run  ", "cargo", []string{"run"}, false},
		{"only command", "make", "make", []string{}, false},
		{"empty", "   ", "", nil, true},
		{"unbalanced quote", `node -e "oops`, "", nil, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cmd, args, err := Parse(c.input)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got cmd=%q args=%v", cmd, args)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cmd != c.wantCmd {
				t.Errorf("cmd = %q, want %q", cmd, c.wantCmd)
			}
			if !reflect.DeepEqual(args, c.wantArgs) {
				t.Errorf("args = %v, want %v", args, c.wantArgs)
			}
		})
	}
}

func TestJoin(t *testing.T) {
	got := Join("go", []string{"run", "main.go", "serve"})
	want := "go run main.go serve"
	if got != want {
		t.Errorf("Join = %q, want %q", got, want)
	}
}
