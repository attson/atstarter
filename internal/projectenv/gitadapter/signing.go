package gitadapter

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"time"

	"rootyard/internal/projectenv"
)

type ProgramRunner interface {
	LookPath(name string) (string, error)
	Run(ctx context.Context, dir, program string, args ...string) (stdout, stderr string, err error)
}

type execProgramRunner struct {
	timeout time.Duration
}

func (r execProgramRunner) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func (r execProgramRunner) Run(ctx context.Context, dir, program string, args ...string) (string, string, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, program, args...)
	cmd.Dir = dir
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func (a *Adapter) Doctor(ctx context.Context, target projectenv.Target, resolved projectenv.ResolvedEnvironment) []projectenv.Issue {
	if !signingEnabled(resolved) {
		return []projectenv.Issue{}
	}
	keyField, ok := resolved.Git["user.signingKey"]
	if !ok || keyField.Intent.Mode != projectenv.ModeSet || strings.TrimSpace(keyField.Intent.Value) == "" {
		return []projectenv.Issue{signingIssue(projectenv.ErrSigningKeyUnavailable, "signing is enabled but user.signingKey is not set", "")}
	}
	format := "openpgp"
	if formatField, ok := resolved.Git["gpg.format"]; ok && formatField.Intent.Mode == projectenv.ModeSet {
		format = formatField.Intent.Value
	}
	key := keyField.Intent.Value
	switch format {
	case "openpgp":
		return a.doctorSigningProgram(ctx, target, "gpg.program", "gpg", key, "--list-secret-keys", "--with-colons", key)
	case "ssh":
		if _, err := os.Stat(key); err != nil {
			return []projectenv.Issue{signingIssue(projectenv.ErrSigningKeyUnavailable, "SSH signing key is not readable", err.Error())}
		}
		return a.doctorSigningProgram(ctx, target, "gpg.ssh.program", "ssh-keygen", key, "-lf", key)
	case "x509":
		return a.doctorSigningProgram(ctx, target, "gpg.x509.program", "gpgsm", key, "--list-secret-keys", key)
	default:
		return []projectenv.Issue{signingIssue(projectenv.ErrInvalidEnvironment, "unsupported Git signing format", format)}
	}
}

func signingEnabled(resolved projectenv.ResolvedEnvironment) bool {
	for _, key := range []string{"commit.gpgSign", "tag.gpgSign"} {
		field, ok := resolved.Git[key]
		if ok && field.Intent.Mode == projectenv.ModeSet && field.Intent.Value == "true" {
			return true
		}
	}
	return false
}

func (a *Adapter) doctorSigningProgram(ctx context.Context, target projectenv.Target, configKey, fallback, signingKey string, args ...string) []projectenv.Issue {
	program := fallback
	stdout, _, err := a.runner.Run(ctx, target.WorktreePath, "config", "--get", configKey)
	if err == nil && strings.TrimSpace(stdout) != "" {
		program = strings.TrimSpace(stdout)
	}
	programs := a.programs
	if programs == nil {
		programs = execProgramRunner{timeout: defaultGitTimeout}
	}
	resolvedProgram, err := programs.LookPath(program)
	if err != nil {
		return []projectenv.Issue{signingIssue(projectenv.ErrSigningProgramUnavailable, "required signing program is unavailable", program)}
	}
	_, stderr, err := programs.Run(ctx, target.WorktreePath, resolvedProgram, args...)
	if err != nil {
		detail := strings.TrimSpace(stderr)
		if detail == "" {
			detail = err.Error()
		}
		return []projectenv.Issue{signingIssue(projectenv.ErrSigningKeyUnavailable, "signing key could not be verified locally", signingKey+": "+detail)}
	}
	return []projectenv.Issue{}
}

func signingIssue(code projectenv.ErrorCode, message, detail string) projectenv.Issue {
	return projectenv.Issue{Code: code, Message: message, Detail: detail, Blocking: true}
}
