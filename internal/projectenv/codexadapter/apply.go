package codexadapter

import (
	"context"
	"errors"
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

	lockCtx, cancelLock := context.WithTimeout(ctx, adapterLockTimeout(a.lockTimeout))
	defer cancelLock()
	unlock, err := projectenv.AcquireFileLock(lockCtx, projectenv.AdapterFileLockPath(a.Name(), currentTarget.ID))
	if err != nil {
		return failedResult(result, errorCode(err, projectenv.ErrApplyFailed), "acquire Codex configuration lock", err.Error())
	}
	defer unlock()

	secure, err := openSecureConfigDir(currentTarget.WorktreePath)
	if err != nil {
		return blockedResult(result, projectenv.ErrUnsafeCodexConfigPath, "open Codex configuration directory", err.Error())
	}
	defer secure.Close()
	current, mode, err := inspectSecureConfig(ctx, currentTarget, secure)
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

	if !current.Exists {
		_, err = secure.ensureConfigDir()
		if err != nil {
			return failedResult(result, projectenv.ErrApplyFailed, "create Codex configuration directory", err.Error())
		}
		_, _, appeared, readErr := secure.readConfig()
		if readErr != nil {
			return failedResult(result, projectenv.ErrApplyFailed, "re-inspect Codex project configuration", readErr.Error())
		}
		if appeared {
			result.Status = projectenv.ApplyStale
			result.Issues = append(result.Issues, projectenv.Issue{
				Code: projectenv.ErrStalePlan, Message: "Codex project configuration appeared after preview",
				Path: currentTarget.ConfigPath, Blocking: true,
			})
			return result
		}
	}
	if a.beforeWrite != nil {
		a.beforeWrite()
	}
	latest, _, latestExists, err := secure.readConfig()
	if err != nil {
		return failedResult(result, projectenv.ErrApplyFailed, "re-inspect Codex project configuration", err.Error())
	}
	if configFingerprint(latestExists, latest) != current.Fingerprint {
		result.Status = projectenv.ApplyStale
		result.Issues = append(result.Issues, projectenv.Issue{
			Code: projectenv.ErrStalePlan, Message: "Codex project configuration changed after preview",
			Path: currentTarget.ConfigPath, Blocking: true,
		})
		return result
	}
	if err := a.writeConfig(secure, proposed, mode); err != nil {
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

func adapterLockTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return 2 * time.Second
	}
	return timeout
}

func errorCode(err error, fallback projectenv.ErrorCode) projectenv.ErrorCode {
	var typed *projectenv.Error
	if errors.As(err, &typed) {
		return typed.Code
	}
	return fallback
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
