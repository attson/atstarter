package gitadapter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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
		return failedApplyResult(result, projectenv.ErrApplyFailed, "re-identify Git repository", err.Error())
	}
	if currentTarget.ID != plan.Target.ID || !samePath(currentTarget.ConfigPath, plan.Target.ConfigPath) {
		result.Status = projectenv.ApplyBlocked
		result.Issues = append(result.Issues, projectenv.Issue{
			Code: projectenv.ErrInvalidRepository, Message: "Git repository target changed after preview", Blocking: true,
		})
		return result
	}

	lockPath := currentTarget.ConfigPath + ".lock"
	unlock, err := acquireConfigLock(ctx, lockPath)
	if err != nil {
		return failedApplyResult(result, projectenv.ErrApplyFailed, "acquire Git configuration lock", err.Error())
	}
	defer unlock()

	original, err := os.ReadFile(currentTarget.ConfigPath)
	if err != nil {
		return failedApplyResult(result, projectenv.ErrApplyFailed, "read Git configuration", err.Error())
	}
	if fingerprintBytes(original) != plan.Fingerprint {
		result.Status = projectenv.ApplyStale
		result.Issues = append(result.Issues, projectenv.Issue{
			Code: projectenv.ErrStalePlan, Message: "Git configuration changed after preview", Path: currentTarget.ConfigPath, Blocking: true,
		})
		return result
	}

	if !planHasWrites(plan) {
		result.Status = projectenv.ApplyUnchanged
		result.Fingerprint = plan.Fingerprint
		return result
	}

	info, err := os.Stat(currentTarget.ConfigPath)
	if err != nil {
		return failedApplyResult(result, projectenv.ErrApplyFailed, "inspect Git configuration", err.Error())
	}
	temp, err := os.CreateTemp(filepath.Dir(currentTarget.ConfigPath), ".rootyard-config-*")
	if err != nil {
		return failedApplyResult(result, projectenv.ErrApplyFailed, "create temporary Git configuration", err.Error())
	}
	tempPath := temp.Name()
	cleanup := true
	defer func() {
		_ = temp.Close()
		if cleanup {
			_ = os.Remove(tempPath)
		}
	}()
	if err := temp.Chmod(info.Mode().Perm()); err != nil {
		return failedApplyResult(result, projectenv.ErrApplyFailed, "set temporary Git configuration permissions", err.Error())
	}
	if _, err := temp.Write(original); err != nil {
		return failedApplyResult(result, projectenv.ErrApplyFailed, "copy Git configuration", err.Error())
	}
	if err := temp.Sync(); err != nil {
		return failedApplyResult(result, projectenv.ErrApplyFailed, "flush temporary Git configuration", err.Error())
	}
	if err := temp.Close(); err != nil {
		return failedApplyResult(result, projectenv.ErrApplyFailed, "close temporary Git configuration", err.Error())
	}

	for _, change := range plan.Changes {
		var args []string
		switch change.Operation {
		case projectenv.OperationSet, projectenv.OperationReplace:
			args = []string{"config", "--file", tempPath, "--replace-all", change.Key, change.Desired.Value}
		case projectenv.OperationClear:
			args = []string{"config", "--file", tempPath, "--unset-all", change.Key}
		default:
			continue
		}
		_, stderr, runErr := a.runner.Run(ctx, currentTarget.WorktreePath, args...)
		if runErr != nil && !(change.Operation == projectenv.OperationClear && isExitCode(runErr, 1)) {
			detail := strings.TrimSpace(stderr)
			if detail == "" {
				detail = runErr.Error()
			}
			return failedApplyResult(result, projectenv.ErrApplyFailed, "update temporary Git configuration", detail)
		}
	}
	if err := a.verifyTemporaryConfig(ctx, currentTarget.WorktreePath, tempPath, plan.Desired); err != nil {
		return failedApplyResult(result, projectenv.ErrApplyFailed, "verify temporary Git configuration", err.Error())
	}
	file, err := os.OpenFile(tempPath, os.O_RDWR, info.Mode().Perm())
	if err != nil {
		return failedApplyResult(result, projectenv.ErrApplyFailed, "open temporary Git configuration", err.Error())
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return failedApplyResult(result, projectenv.ErrApplyFailed, "flush temporary Git configuration", err.Error())
	}
	if err := file.Close(); err != nil {
		return failedApplyResult(result, projectenv.ErrApplyFailed, "close temporary Git configuration", err.Error())
	}
	finalBytes, err := os.ReadFile(tempPath)
	if err != nil {
		return failedApplyResult(result, projectenv.ErrApplyFailed, "read temporary Git configuration", err.Error())
	}
	if err := replaceConfigFile(tempPath, currentTarget.ConfigPath); err != nil {
		return failedApplyResult(result, projectenv.ErrApplyFailed, "replace Git configuration", err.Error())
	}
	cleanup = false
	if err := syncConfigDirectory(filepath.Dir(currentTarget.ConfigPath)); err != nil {
		return failedApplyResult(result, projectenv.ErrApplyFailed, "flush Git configuration directory", err.Error())
	}
	result.Status = projectenv.ApplySuccess
	result.Fingerprint = fingerprintBytes(finalBytes)
	return result
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

func (a *Adapter) verifyTemporaryConfig(ctx context.Context, dir, path string, desired map[string]projectenv.FieldIntent) error {
	for _, key := range projectenv.SupportedGitKeys {
		intent := desired[key]
		if intent.Mode == projectenv.ModeInherit {
			continue
		}
		stdout, stderr, err := a.runner.Run(ctx, dir, "config", "--file", path, "--get-all", key)
		if err != nil && !isExitCode(err, 1) {
			if strings.TrimSpace(stderr) != "" {
				return errors.New(strings.TrimSpace(stderr))
			}
			return err
		}
		values := splitConfigValues(stdout)
		switch intent.Mode {
		case projectenv.ModeSet:
			if len(values) != 1 || values[0] != intent.Value {
				return errors.New("temporary Git value does not match plan for " + key)
			}
		case projectenv.ModeClear:
			if len(values) != 0 {
				return errors.New("temporary Git value was not cleared for " + key)
			}
		}
	}
	return nil
}

func splitConfigValues(output string) []string {
	output = strings.TrimSuffix(output, "\n")
	if output == "" {
		return nil
	}
	return strings.Split(output, "\n")
}

func isExitCode(err error, code int) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr) && exitErr.ExitCode() == code
}

func planHasWrites(plan projectenv.ChangePlan) bool {
	for _, change := range plan.Changes {
		switch change.Operation {
		case projectenv.OperationSet, projectenv.OperationReplace, projectenv.OperationClear:
			return true
		}
	}
	return false
}

func fingerprintBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func failedApplyResult(result projectenv.ApplyResult, code projectenv.ErrorCode, message, detail string) projectenv.ApplyResult {
	result.Status = projectenv.ApplyFailed
	result.Issues = append(result.Issues, projectenv.Issue{Code: code, Message: message, Detail: detail, Blocking: true})
	return result
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
