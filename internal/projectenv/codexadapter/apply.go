package codexadapter

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"time"

	"rootyard/internal/projectenv"
)

func (a *Adapter) Apply(ctx context.Context, plan projectenv.ChangePlan) projectenv.ApplyResult {
	result := projectenv.ApplyResult{Target: plan.Target, Changes: plan.Changes, Issues: []projectenv.Issue{}}
	if len(plan.Blockers) > 0 {
		result.Status = projectenv.ApplyBlocked
		result.Issues = append(result.Issues, plan.Blockers...)
		return result
	}
	currentTarget, err := a.Identify(ctx, plan.Target.WorktreePath)
	if err != nil {
		return failedResult(result, projectenv.ErrApplyFailed, "re-identify Codex project", err.Error())
	}
	if currentTarget.ID != plan.Target.ID || !normalizedPathEqual(currentTarget.ConfigPath, plan.Target.ConfigPath) {
		return blockedResult(result, projectenv.ErrInvalidRepository, "Codex project target changed after preview", currentTarget.WorktreePath)
	}

	unlock, err := acquireConfigLock(ctx, filepath.Join(currentTarget.WorktreePath, ".rootyard-codex-config.lock"))
	if err != nil {
		return failedResult(result, projectenv.ErrApplyFailed, "acquire Codex configuration lock", err.Error())
	}
	defer unlock()

	current, err := a.Inspect(ctx, currentTarget)
	if err != nil {
		return blockedErrorResult(result, err)
	}
	if current.Fingerprint != plan.Fingerprint {
		result.Status = projectenv.ApplyStale
		result.Issues = append(result.Issues, projectenv.Issue{
			Code: projectenv.ErrStalePlan, Message: "Codex project configuration changed after preview",
			Path: currentTarget.ConfigPath, Blocking: true,
		})
		return result
	}
	if err := projectenv.ValidateEnvironmentFields(projectenv.EnvironmentFields{Codex: plan.Desired}); err != nil {
		return failedResult(result, projectenv.ErrInvalidCodexConfig, "validate stored Codex plan", err.Error())
	}
	if !a.storedPlanMatches(plan, current) {
		return failedResult(result, projectenv.ErrApplyFailed, "validate stored Codex plan", "displayed plan does not match the current managed edits")
	}
	if len(plan.NativeEdits) == 0 {
		result.Status = projectenv.ApplyUnchanged
		result.Fingerprint = current.Fingerprint
		return result
	}

	proposed, err := applyTextEdits(current.NativeContent, plan.NativeEdits)
	if err != nil {
		return failedResult(result, projectenv.ErrApplyFailed, "apply stored Codex text edits", err.Error())
	}
	parsed, err := parseDocument(proposed)
	if err != nil {
		return failedResult(result, projectenv.ErrInvalidCodexConfig, "validate proposed Codex configuration", err.Error())
	}
	if err := verifyDesired(parsed, plan.Desired); err != nil {
		return failedResult(result, projectenv.ErrInvalidCodexConfig, "verify proposed Codex configuration", err.Error())
	}

	mode := os.FileMode(0o600)
	createdDir := false
	if current.Exists {
		info, statErr := os.Stat(currentTarget.ConfigPath)
		if statErr != nil {
			return failedResult(result, projectenv.ErrApplyFailed, "inspect Codex configuration mode", statErr.Error())
		}
		mode = info.Mode().Perm()
	} else {
		createdDir, err = ensureConfigDirectory(filepath.Dir(currentTarget.ConfigPath))
		if err != nil {
			return failedResult(result, projectenv.ErrApplyFailed, "create Codex configuration directory", err.Error())
		}
		if _, statErr := os.Lstat(currentTarget.ConfigPath); statErr == nil || !errors.Is(statErr, os.ErrNotExist) {
			if createdDir {
				_ = os.Remove(filepath.Dir(currentTarget.ConfigPath))
			}
			result.Status = projectenv.ApplyStale
			result.Issues = append(result.Issues, projectenv.Issue{
				Code: projectenv.ErrStalePlan, Message: "Codex project configuration appeared after preview",
				Path: currentTarget.ConfigPath, Blocking: true,
			})
			return result
		}
	}
	if err := a.atomicWrite(currentTarget.ConfigPath, proposed, mode); err != nil {
		if createdDir {
			_ = os.Remove(filepath.Dir(currentTarget.ConfigPath))
		}
		return failedResult(result, projectenv.ErrApplyFailed, "write Codex project configuration", err.Error())
	}
	result.Status = projectenv.ApplySuccess
	result.Fingerprint = configFingerprint(true, proposed)
	return result
}

func (a *Adapter) storedPlanMatches(plan projectenv.ChangePlan, current projectenv.Snapshot) bool {
	resolved := projectenv.ResolvedEnvironment{RepoPath: plan.RepoPath, Codex: map[string]projectenv.ResolvedField{}}
	sources := map[string]projectenv.Source{}
	for _, change := range plan.Changes {
		sources[change.Key] = change.Source
	}
	for key, intent := range plan.Desired {
		if intent.Mode != projectenv.ModeInherit {
			resolved.Codex[key] = projectenv.ResolvedField{Intent: intent, Source: sources[key]}
		}
	}
	expected := a.Plan(resolved, current)
	return slices.Equal(expected.NativeEdits, plan.NativeEdits) &&
		slices.Equal(expected.NativeDiff, plan.NativeDiff) &&
		changesEqual(expected.Changes, plan.Changes) &&
		slices.Equal(expected.Warnings, plan.Warnings)
}

func changesEqual(left, right []projectenv.Change) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		l, r := left[index], right[index]
		if l.Key != r.Key || l.Operation != r.Operation || l.Desired != r.Desired || l.Source != r.Source ||
			!slices.Equal(l.Current, r.Current) || !slices.Equal(l.Issues, r.Issues) {
			return false
		}
	}
	return true
}

func verifyDesired(doc document, desired map[string]projectenv.FieldIntent) error {
	for key, intent := range desired {
		assignment, exists := doc.Assignments[key]
		switch intent.Mode {
		case projectenv.ModeSet:
			if !exists || assignment.Value != intent.Value {
				return errors.New("managed value does not match plan for " + key)
			}
		case projectenv.ModeClear:
			if exists {
				return errors.New("managed value was not cleared for " + key)
			}
		}
	}
	return nil
}

func ensureConfigDirectory(dir string) (bool, error) {
	err := os.Mkdir(dir, 0o700)
	if err == nil {
		return true, nil
	}
	if !errors.Is(err, os.ErrExist) {
		return false, err
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return false, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return false, errors.New("Codex configuration directory is not a real directory")
	}
	return false, nil
}

func atomicWriteConfig(path string, data []byte, mode os.FileMode) (returnErr error) {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".rootyard-codex-config-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	defer func() {
		_ = temp.Close()
		if returnErr != nil {
			_ = os.Remove(tempPath)
		}
	}()
	if err := temp.Chmod(mode.Perm()); err != nil {
		return err
	}
	if _, err := temp.Write(data); err != nil {
		return err
	}
	if err := temp.Sync(); err != nil {
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := replaceConfigFile(tempPath, path); err != nil {
		return err
	}
	if err := syncConfigDirectory(dir); err != nil {
		return err
	}
	return nil
}

func acquireConfigLock(ctx context.Context, path string) (func(), error) {
	for {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			_ = file.Close()
			return func() { _ = os.Remove(path) }, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, err
		}
		timer := time.NewTimer(10 * time.Millisecond)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func syncConfigDirectory(dir string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	file, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer file.Close()
	return file.Sync()
}

func blockedErrorResult(result projectenv.ApplyResult, err error) projectenv.ApplyResult {
	code := projectenv.ErrApplyFailed
	var typed *projectenv.Error
	if errors.As(err, &typed) {
		code = typed.Code
	}
	return blockedResult(result, code, "inspect Codex project configuration", err.Error())
}

func blockedResult(result projectenv.ApplyResult, code projectenv.ErrorCode, message, detail string) projectenv.ApplyResult {
	result.Status = projectenv.ApplyBlocked
	result.Issues = append(result.Issues, projectenv.Issue{Code: code, Message: message, Detail: detail, Blocking: true})
	return result
}

func failedResult(result projectenv.ApplyResult, code projectenv.ErrorCode, message, detail string) projectenv.ApplyResult {
	result.Status = projectenv.ApplyFailed
	result.Issues = append(result.Issues, projectenv.Issue{Code: code, Message: message, Detail: detail, Blocking: true})
	return result
}
