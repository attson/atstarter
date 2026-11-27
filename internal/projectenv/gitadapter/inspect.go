package gitadapter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"rootyard/internal/projectenv"
)

func (a *Adapter) Inspect(ctx context.Context, target projectenv.Target) (projectenv.Snapshot, error) {
	data, err := os.ReadFile(target.ConfigPath)
	if err != nil {
		return projectenv.Snapshot{}, &projectenv.Error{Code: projectenv.ErrInvalidGitConfig, Message: "read Git configuration", Detail: err.Error(), Cause: err}
	}
	sum := sha256.Sum256(data)
	snapshot := projectenv.Snapshot{
		Target:      target,
		Values:      make(map[string][]projectenv.OriginValue, len(projectenv.SupportedGitKeys)),
		Fingerprint: hex.EncodeToString(sum[:]),
		Issues:      []projectenv.Issue{},
	}
	for _, key := range projectenv.SupportedGitKeys {
		stdout, stderr, runErr := a.runner.Run(ctx, target.WorktreePath, "config", "--local", "--includes", "--show-origin", "--get-all", key)
		if runErr != nil {
			var exitErr *exec.ExitError
			if errors.As(runErr, &exitErr) && exitErr.ExitCode() == 1 && strings.TrimSpace(stdout) == "" {
				snapshot.Values[key] = []projectenv.OriginValue{}
				continue
			}
			detail := strings.TrimSpace(stderr)
			if detail == "" {
				detail = runErr.Error()
			}
			return projectenv.Snapshot{}, &projectenv.Error{Code: projectenv.ErrInvalidGitConfig, Message: "inspect Git configuration", Detail: detail, Cause: runErr}
		}
		values := parseOriginValues(stdout, target)
		snapshot.Values[key] = values
		for _, value := range values {
			if value.MainConfig {
				continue
			}
			snapshot.Issues = append(snapshot.Issues, projectenv.Issue{
				Code:     projectenv.ErrExternalLocalOrigin,
				Message:  "managed Git value originates from an included local file",
				Detail:   value.Origin,
				Path:     target.WorktreePath,
				Key:      key,
				Blocking: true,
			})
			break
		}
	}
	return snapshot, nil
}

func parseOriginValues(output string, target projectenv.Target) []projectenv.OriginValue {
	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return []projectenv.OriginValue{}
	}
	values := make([]projectenv.OriginValue, 0, len(lines))
	for _, line := range lines {
		origin, value, ok := strings.Cut(line, "\t")
		if !ok {
			origin, value, ok = strings.Cut(line, " ")
		}
		if !ok {
			continue
		}
		origin = strings.TrimPrefix(origin, "file:")
		canonical := canonicalOriginPath(target.WorktreePath, origin)
		values = append(values, projectenv.OriginValue{
			Value:      value,
			Origin:     canonical,
			MainConfig: samePath(canonical, target.ConfigPath),
		})
	}
	return values
}

func canonicalOriginPath(base, path string) string {
	if !filepath.IsAbs(path) {
		path = filepath.Join(base, path)
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return filepath.Clean(resolved)
	}
	return filepath.Clean(abs)
}

func samePath(left, right string) bool {
	leftInfo, leftErr := os.Stat(left)
	rightInfo, rightErr := os.Stat(right)
	if leftErr == nil && rightErr == nil {
		return os.SameFile(leftInfo, rightInfo)
	}
	return filepath.Clean(left) == filepath.Clean(right)
}
