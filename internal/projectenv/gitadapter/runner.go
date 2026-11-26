package gitadapter

import (
	"bytes"
	"context"
	"os/exec"
	"time"
)

const defaultGitTimeout = 10 * time.Second

type Runner interface {
	Run(ctx context.Context, dir string, args ...string) (stdout, stderr string, err error)
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
