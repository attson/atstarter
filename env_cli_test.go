package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"rootyard/internal/projectenv"
)

func runEnvironmentCLIForTest(t *testing.T, configPath string, args ...string) (int, string, string) {
	t.Helper()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	full := append([]string{"--config", configPath}, args...)
	code := runEnvCLI(full, strings.NewReader(""), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func decodeEnvironmentEnvelope(t *testing.T, output string) map[string]any {
	t.Helper()
	var envelope map[string]any
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		t.Fatalf("decode %q: %v", output, err)
	}
	return envelope
}

func TestRunEnvCLIPresetDefaultsAndRuleCRUD(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	code, output, _ := runEnvironmentCLIForTest(t, configPath,
		"--json", "preset", "set", "--name", "Company",
		"--git-set", "user.name=Ada", "--git-clear", "user.signingKey",
		"--codex-set", "model=company-model", "--codex-clear", "web_search")
	if code != 0 {
		t.Fatalf("code = %d, output = %s", code, output)
	}
	envelope := decodeEnvironmentEnvelope(t, output)
	if envelope["ok"] != true {
		t.Fatalf("envelope = %+v", envelope)
	}
	doc, err := projectenv.NewStore(configPath).Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	preset := doc.Config.Presets[0]
	if preset.Git["user.name"].Value != "Ada" || preset.Git["user.signingKey"].Mode != projectenv.ModeClear ||
		preset.Codex["model"].Value != "company-model" || preset.Codex["web_search"].Mode != projectenv.ModeClear {
		t.Fatalf("preset = %+v", preset)
	}

	code, output, _ = runEnvironmentCLIForTest(t, configPath, "defaults", "set", "--preset", preset.ID,
		"--codex-set", "model_reasoning_effort=high")
	if code != 0 {
		t.Fatalf("code = %d, output = %s", code, output)
	}
	root := t.TempDir()
	code, output, _ = runEnvironmentCLIForTest(t, configPath,
		"rule", "set", "--name", "Work", "--path", root, "--recursive", "--auto-discover", "--preset", preset.ID,
		"--codex-set", "approval_policy=on-request")
	if code != 0 {
		t.Fatalf("code = %d, output = %s", code, output)
	}
	doc, _ = projectenv.NewStore(configPath).Load(context.Background())
	if doc.Config.Defaults.PresetID != preset.ID || doc.Config.Defaults.Codex["model_reasoning_effort"].Value != "high" ||
		len(doc.Config.Rules) != 1 || !doc.Config.Rules[0].Recursive || !doc.Config.Rules[0].AutoDiscover ||
		doc.Config.Rules[0].Codex["approval_policy"].Value != "on-request" {
		t.Fatalf("document = %+v", doc.Config)
	}
}

func TestRunEnvCLICodexSessionDiffAndApply(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is unavailable")
	}
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	repo := filepath.Join(dir, "repo")
	if err := os.Mkdir(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init")
	cmd.Dir = repo
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}

	code, output, _ := runEnvironmentCLIForTest(t, configPath,
		"diff", repo, "--codex-set", "model=custom-model")
	if code != 0 || !strings.Contains(output, "codex") || !strings.Contains(output, "model") {
		t.Fatalf("diff: %d %s", code, output)
	}
	if _, err := os.Stat(filepath.Join(repo, ".codex")); !os.IsNotExist(err) {
		t.Fatalf("diff created project config: %v", err)
	}

	code, output, _ = runEnvironmentCLIForTest(t, configPath,
		"--json", "apply", repo, "--codex-set", "model=custom-model")
	if code == 0 || !strings.Contains(output, "confirmation_required") {
		t.Fatalf("unconfirmed apply: %d %s", code, output)
	}
	code, output, _ = runEnvironmentCLIForTest(t, configPath,
		"--json", "apply", repo, "--codex-set", "model=custom-model", "--yes")
	if code != 0 || !strings.Contains(output, "success") {
		t.Fatalf("confirmed apply: %d %s", code, output)
	}
	data, err := os.ReadFile(filepath.Join(repo, ".codex", "config.toml"))
	if err != nil || string(data) != "model = \"custom-model\"\n" {
		t.Fatalf("config = %q, err = %v", data, err)
	}
}

func TestRunEnvCLIDangerousCodexApplyRequiresAdditionalConfirmation(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is unavailable")
	}
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	repo := filepath.Join(dir, "repo")
	if err := os.Mkdir(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init")
	cmd.Dir = repo
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	args := []string{
		"--json", "apply", repo,
		"--codex-set", "sandbox_mode=danger-full-access",
		"--codex-set", "approval_policy=never", "--yes",
	}
	code, output, _ := runEnvironmentCLIForTest(t, configPath, args...)
	if code == 0 || !strings.Contains(output, "dangerous_confirmation_required") {
		t.Fatalf("dangerous apply: %d %s", code, output)
	}
	if _, err := os.Stat(filepath.Join(repo, ".codex")); !os.IsNotExist(err) {
		t.Fatalf("unconfirmed dangerous apply wrote config: %v", err)
	}
	args = append(args, "--accept-dangerous")
	code, output, _ = runEnvironmentCLIForTest(t, configPath, args...)
	if code != 0 || !strings.Contains(output, "success") {
		t.Fatalf("confirmed dangerous apply: %d %s", code, output)
	}
}

func TestConfirmEnvironmentApplyConsumesBothInteractiveAnswersFromOneReader(t *testing.T) {
	plans := []projectenv.ChangePlan{{Warnings: []projectenv.Issue{{Code: projectenv.ErrDangerousCodexPermissions}}}}
	var output bytes.Buffer
	err := confirmEnvironmentApply(strings.NewReader("yes\nAPPLY DANGEROUS\n"), &output, plans, false, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "Apply these changes") || !strings.Contains(output.String(), "APPLY DANGEROUS") {
		t.Fatalf("prompts = %q", output.String())
	}
}

func TestConfirmEnvironmentApplyRequiresTypedDangerousAcknowledgementInInteractiveMode(t *testing.T) {
	plans := []projectenv.ChangePlan{{Warnings: []projectenv.Issue{{Code: projectenv.ErrDangerousCodexPermissions}}}}
	var output bytes.Buffer
	err := confirmEnvironmentApply(strings.NewReader(""), &output, plans, true, true, true)
	var cliErr envCLIError
	if !errors.As(err, &cliErr) || cliErr.code != "dangerous_confirmation_required" {
		t.Fatalf("err = %#v", err)
	}
	if !strings.Contains(output.String(), "APPLY DANGEROUS") {
		t.Fatalf("prompt = %q", output.String())
	}
}

func TestRunEnvCLIScanResolveAndDiffOffline(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is unavailable")
	}
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	repo := filepath.Join(dir, "repo")
	if err := os.Mkdir(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init")
	cmd.Dir = repo
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	code, output, _ := runEnvironmentCLIForTest(t, configPath,
		"rule", "set", "--name", "Repos", "--path", dir, "--recursive", "--git-set", "user.email=base@example.com")
	if code != 0 {
		t.Fatalf("rule set: %d %s", code, output)
	}
	code, output, _ = runEnvironmentCLIForTest(t, configPath, "--json", "scan")
	if code != 0 || decodeEnvironmentEnvelope(t, output)["ok"] != true {
		t.Fatalf("scan: %d %s", code, output)
	}
	code, output, _ = runEnvironmentCLIForTest(t, configPath, "--json", "resolve", repo, "--git-set", "user.email=session@example.com")
	if code != 0 || !strings.Contains(output, "session@example.com") {
		t.Fatalf("resolve: %d %s", code, output)
	}
	code, output, _ = runEnvironmentCLIForTest(t, configPath, "diff", repo)
	if code != 0 || !strings.Contains(output, "user.email") || !strings.Contains(output, "set") {
		t.Fatalf("diff: %d %s", code, output)
	}
	if _, err := os.Stat(controlStatePath(configPath)); !os.IsNotExist(err) {
		t.Fatalf("offline env command touched control state: %v", err)
	}
}

func TestRunEnvCLIJSONApplyRequiresConfirmationThenYesApplies(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is unavailable")
	}
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	repo := filepath.Join(dir, "repo")
	if err := os.Mkdir(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init")
	cmd.Dir = repo
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, output)
	}
	code, output, _ := runEnvironmentCLIForTest(t, configPath,
		"--json", "apply", repo, "--git-set", "user.name=Ada")
	if code == 0 || !strings.Contains(output, "confirmation_required") {
		t.Fatalf("unconfirmed apply: %d %s", code, output)
	}
	check := exec.Command("git", "config", "--local", "--get", "user.name")
	check.Dir = repo
	if value, err := check.Output(); err == nil || len(value) != 0 {
		t.Fatalf("unconfirmed apply wrote user.name: %q %v", value, err)
	}
	code, output, _ = runEnvironmentCLIForTest(t, configPath,
		"--json", "apply", repo, "--git-set", "user.name=Ada", "--yes")
	if code != 0 || !strings.Contains(output, "success") {
		t.Fatalf("confirmed apply: %d %s", code, output)
	}
	check = exec.Command("git", "config", "--local", "--get", "user.name")
	check.Dir = repo
	value, err := check.Output()
	if err != nil || strings.TrimSpace(string(value)) != "Ada" {
		t.Fatalf("user.name = %q, err = %v", value, err)
	}
}

func TestRunEnvCLIUnknownCommandDoesNotTouchEnvironmentFiles(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	code, output, _ := runEnvironmentCLIForTest(t, configPath, "--json", "unknown")
	if code != 2 || !strings.Contains(output, "usage") {
		t.Fatalf("code = %d, output = %s", code, output)
	}
	paths := projectenv.PathsForConfig(configPath)
	for _, path := range []string{paths.Config, paths.State} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("unknown command touched %s: %v", path, err)
		}
	}
}

func TestMainEnvDispatchesBeforeDesktop(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.json")
	var stdout bytes.Buffer
	handled, code := dispatchTopLevel([]string{"rootyard", "env", "--config", configPath, "--json", "preset", "list"}, strings.NewReader(""), &stdout, &bytes.Buffer{})
	if !handled || code != 0 || decodeEnvironmentEnvelope(t, stdout.String())["ok"] != true {
		t.Fatalf("handled = %v code = %d output = %s", handled, code, stdout.String())
	}
}
