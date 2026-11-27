package codexadapter

import (
	"context"
	"path/filepath"
	"strings"

	"rootyard/internal/projectenv"
)

func (a *Adapter) Identify(ctx context.Context, path string) (projectenv.Target, error) {
	stdout, stderr, err := a.runner.Run(ctx, path, "rev-parse", "--show-toplevel")
	if err != nil {
		detail := strings.TrimSpace(stderr)
		if detail == "" {
			detail = err.Error()
		}
		return projectenv.Target{}, &projectenv.Error{Code: projectenv.ErrInvalidRepository, Message: "identify Codex project worktree", Detail: detail, Cause: err}
	}
	worktree := strings.TrimSpace(stdout)
	if !filepath.IsAbs(worktree) {
		worktree = filepath.Join(path, worktree)
	}
	abs, err := filepath.Abs(worktree)
	if err != nil {
		return projectenv.Target{}, &projectenv.Error{Code: projectenv.ErrInvalidRepository, Message: "resolve Codex project worktree", Detail: err.Error(), Cause: err}
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return projectenv.Target{}, &projectenv.Error{Code: projectenv.ErrInvalidRepository, Message: "resolve Codex project worktree", Detail: err.Error(), Cause: err}
	}
	worktree = filepath.Clean(resolved)
	return projectenv.Target{
		Adapter: a.Name(), ID: worktree, WorktreePath: worktree,
		ConfigPath: filepath.Join(worktree, ".codex", "config.toml"),
	}, nil
}
