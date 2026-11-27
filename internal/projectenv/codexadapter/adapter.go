package codexadapter

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"time"

	"rootyard/internal/projectenv"
)

const defaultGitTimeout = 10 * time.Second

type runner interface {
	Run(context.Context, string, ...string) (string, string, error)
}

type commandRunner struct {
	path    string
	timeout time.Duration
}

func (r commandRunner) Run(ctx context.Context, dir string, args ...string) (string, string, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, r.path, args...)
	cmd.Dir = dir
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

type Adapter struct {
	runner      runner
	lookPath    func(string) (string, error)
	atomicWrite func(string, []byte, os.FileMode) error
}

var _ projectenv.Adapter = (*Adapter)(nil)

func New() *Adapter {
	path, err := exec.LookPath("git")
	if err != nil {
		path = "git"
	}
	return &Adapter{
		runner:   commandRunner{path: path, timeout: defaultGitTimeout},
		lookPath: exec.LookPath, atomicWrite: atomicWriteConfig,
	}
}

func (a *Adapter) Name() string { return "codex" }
