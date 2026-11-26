package gitadapter

import (
	"context"
	"errors"
	"strings"
	"testing"

	"rootyard/internal/projectenv"
)

type signingRunner struct {
	responses map[string]struct {
		stdout string
		stderr string
		err    error
	}
}

type signingProgramRunner struct {
	lookupErr error
}

func (r signingProgramRunner) LookPath(name string) (string, error) {
	if r.lookupErr != nil {
		return "", r.lookupErr
	}
	return name, nil
}

func (r signingProgramRunner) Run(context.Context, string, string, ...string) (string, string, error) {
	return "", "", nil
}

func (r signingRunner) Run(_ context.Context, _ string, args ...string) (string, string, error) {
	key := strings.Join(args, " ")
	if response, ok := r.responses[key]; ok {
		return response.stdout, response.stderr, response.err
	}
	return "", "not found", errors.New("not found")
}

func signingResolved(format, key string, enabled bool) projectenv.ResolvedEnvironment {
	git := map[string]projectenv.ResolvedField{
		"commit.gpgSign": {Intent: projectenv.FieldIntent{Mode: projectenv.ModeSet, Value: "false"}},
		"tag.gpgSign":    {Intent: projectenv.FieldIntent{Mode: projectenv.ModeSet, Value: "false"}},
	}
	if enabled {
		git["commit.gpgSign"] = projectenv.ResolvedField{Intent: projectenv.FieldIntent{Mode: projectenv.ModeSet, Value: "true"}}
	}
	if format != "" {
		git["gpg.format"] = projectenv.ResolvedField{Intent: projectenv.FieldIntent{Mode: projectenv.ModeSet, Value: format}}
	}
	if key != "" {
		git["user.signingKey"] = projectenv.ResolvedField{Intent: projectenv.FieldIntent{Mode: projectenv.ModeSet, Value: key}}
	}
	return projectenv.ResolvedEnvironment{Git: git}
}

func TestDoctorSigningDisabledNeedsNoProgramOrKey(t *testing.T) {
	adapter := &Adapter{runner: signingRunner{responses: map[string]struct {
		stdout string
		stderr string
		err    error
	}{}}, programs: signingProgramRunner{}}
	issues := adapter.Doctor(context.Background(), projectenv.Target{WorktreePath: t.TempDir()}, signingResolved("", "", false))
	if len(issues) != 0 {
		t.Fatalf("issues = %+v", issues)
	}
}

func TestDoctorSigningRequiresKey(t *testing.T) {
	issues := New().Doctor(context.Background(), projectenv.Target{WorktreePath: t.TempDir()}, signingResolved("openpgp", "", true))
	if len(issues) == 0 || issues[0].Code != projectenv.ErrSigningKeyUnavailable || !issues[0].Blocking {
		t.Fatalf("issues = %+v", issues)
	}
}

func TestDoctorSigningReportsMissingOpenPGPProgram(t *testing.T) {
	adapter := &Adapter{runner: signingRunner{responses: map[string]struct {
		stdout string
		stderr string
		err    error
	}{
		"config --get gpg.program": {err: errors.New("absent")},
	}}, programs: signingProgramRunner{lookupErr: errors.New("missing")}}
	issues := adapter.Doctor(context.Background(), projectenv.Target{WorktreePath: t.TempDir()}, signingResolved("openpgp", "ABC123", true))
	if len(issues) == 0 || issues[0].Code != projectenv.ErrSigningProgramUnavailable {
		t.Fatalf("issues = %+v", issues)
	}
}

func TestDoctorSigningReportsMissingSSHKeyPath(t *testing.T) {
	issues := New().Doctor(context.Background(), projectenv.Target{WorktreePath: t.TempDir()}, signingResolved("ssh", "/definitely/missing/rootyard-key.pub", true))
	if len(issues) == 0 || issues[0].Code != projectenv.ErrSigningKeyUnavailable {
		t.Fatalf("issues = %+v", issues)
	}
}
