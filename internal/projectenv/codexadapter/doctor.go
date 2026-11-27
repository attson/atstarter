package codexadapter

import (
	"context"
	"errors"

	"rootyard/internal/projectenv"
)

func (a *Adapter) Doctor(ctx context.Context, target projectenv.Target, resolved projectenv.ResolvedEnvironment) []projectenv.Issue {
	snapshot, err := a.Inspect(ctx, target)
	if err != nil {
		code := projectenv.ErrInvalidCodexConfig
		var typed *projectenv.Error
		if errors.As(err, &typed) {
			code = typed.Code
		}
		return []projectenv.Issue{{Code: code, Message: "inspect Codex project configuration", Detail: err.Error(), Path: target.ConfigPath, Blocking: true}}
	}
	issues := []projectenv.Issue{}
	if _, err := a.lookPath("codex"); err != nil {
		issues = append(issues, projectenv.Issue{
			Code: projectenv.ErrCodexUnavailable, Message: "Codex executable is unavailable", Path: target.WorktreePath,
		})
	}
	if dangerousCodexPermissions(resolved.Codex, snapshot) {
		issues = append(issues, projectenv.Issue{
			Code:    projectenv.ErrDangerousCodexPermissions,
			Message: "Codex will run without sandboxing or approval prompts", Path: target.ConfigPath,
		})
	}
	return issues
}
