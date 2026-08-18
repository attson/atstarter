//go:build darwin

package main

import (
	"strings"
	"testing"
)

func TestDarwinRelaunchAfterExitScriptWaitsThenOpensNewInstance(t *testing.T) {
	script := darwinRelaunchAfterExitScript(12345, "/Applications/AT Starter.app")

	if !strings.Contains(script, "kill -0 12345") {
		t.Fatalf("relaunch script must wait for the old app process to exit, got:\n%s", script)
	}
	if !strings.Contains(script, "/usr/bin/open -n '/Applications/AT Starter.app'") {
		t.Fatalf("relaunch script must open a new instance after exit, got:\n%s", script)
	}
}

func TestDarwinShellQuoteHandlesSingleQuotes(t *testing.T) {
	got := darwinShellQuote("/tmp/Bob's Apps/AT Starter.app")
	want := `'/tmp/Bob'\''s Apps/AT Starter.app'`
	if got != want {
		t.Errorf("darwinShellQuote = %q, want %q", got, want)
	}
}
