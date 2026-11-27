package gitadapter

import (
	"context"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"rootyard/internal/projectenv"
)

type Adapter struct {
	runner      Runner
	programs    ProgramRunner
	lockTimeout time.Duration
}

var _ projectenv.Adapter = (*Adapter)(nil)

func New() *Adapter {
	path, err := exec.LookPath("git")
	if err != nil {
		path = "git"
	}
	return &Adapter{
		runner:      commandRunner{path: path, timeout: defaultGitTimeout},
		programs:    execProgramRunner{timeout: defaultGitTimeout},
		lockTimeout: 2 * time.Second,
	}
}

func (a *Adapter) Name() string {
	return "git"
}

func (a *Adapter) Identify(ctx context.Context, path string) (projectenv.Target, error) {
	worktree, err := a.gitPath(ctx, path, "--show-toplevel")
	if err != nil {
		return projectenv.Target{}, err
	}
	worktree, err = canonicalGitPath(path, worktree)
	if err != nil {
		return projectenv.Target{}, invalidRepositoryError("resolve Git worktree", err.Error(), err)
	}
	commonDir, err := a.gitPath(ctx, worktree, "--git-common-dir")
	if err != nil {
		return projectenv.Target{}, err
	}
	commonDir, err = canonicalGitPath(worktree, commonDir)
	if err != nil {
		return projectenv.Target{}, invalidRepositoryError("resolve Git common directory", err.Error(), err)
	}
	configPath, err := a.gitPath(ctx, worktree, "--git-path", "config")
	if err != nil {
		return projectenv.Target{}, err
	}
	configPath, err = canonicalGitPath(worktree, configPath)
	if err != nil {
		return projectenv.Target{}, invalidRepositoryError("resolve Git config path", err.Error(), err)
	}
	return projectenv.Target{
		Adapter:      a.Name(),
		ID:           commonDir,
		WorktreePath: worktree,
		CommonDir:    commonDir,
		ConfigPath:   configPath,
	}, nil
}

func (a *Adapter) gitPath(ctx context.Context, dir string, args ...string) (string, error) {
	stdout, stderr, err := a.runner.Run(ctx, dir, append([]string{"rev-parse"}, args...)...)
	if err != nil {
		detail := strings.TrimSpace(stderr)
		if detail == "" {
			detail = err.Error()
		}
		return "", invalidRepositoryError("identify Git repository", detail, err)
	}
	value := strings.TrimSpace(stdout)
	if value == "" {
		return "", invalidRepositoryError("identify Git repository", "Git returned an empty path", nil)
	}
	return value, nil
}

func canonicalGitPath(base, path string) (string, error) {
	if !filepath.IsAbs(path) {
		path = filepath.Join(base, path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	return filepath.Clean(resolved), nil
}

func invalidRepositoryError(message, detail string, cause error) error {
	return &projectenv.Error{Code: projectenv.ErrInvalidRepository, Message: message, Detail: detail, Cause: cause}
}
