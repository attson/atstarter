package projectenv

import (
	"errors"
	"fmt"
)

type ErrorCode string

const (
	ErrInvalidEnvironment        ErrorCode = "invalid_environment"
	ErrUnsupportedVersion        ErrorCode = "unsupported_environment_version"
	ErrDuplicateRulePath         ErrorCode = "duplicate_rule_path"
	ErrPresetNotFound            ErrorCode = "preset_not_found"
	ErrEnvironmentStale          ErrorCode = "environment_stale"
	ErrLockTimeout               ErrorCode = "lock_timeout"
	ErrExternalLocalOrigin       ErrorCode = "external_local_origin"
	ErrSharedTargetConflict      ErrorCode = "shared_target_conflict"
	ErrScanWarning               ErrorCode = "scan_warning"
	ErrGitUnavailable            ErrorCode = "git_unavailable"
	ErrInvalidRepository         ErrorCode = "invalid_repository"
	ErrInvalidGitConfig          ErrorCode = "invalid_git_config"
	ErrInvalidCodexConfig        ErrorCode = "invalid_codex_config"
	ErrUnsafeCodexConfigPath     ErrorCode = "unsafe_codex_config_path"
	ErrStalePlan                 ErrorCode = "stale_plan"
	ErrApplyFailed               ErrorCode = "apply_failed"
	ErrSigningKeyUnavailable     ErrorCode = "signing_key_unavailable"
	ErrSigningProgramUnavailable ErrorCode = "signing_program_unavailable"
	ErrPresetInUse               ErrorCode = "preset_in_use"
	ErrRuleNotFound              ErrorCode = "rule_not_found"
)

type Error struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Detail  string    `json:"detail,omitempty"`
	Cause   error     `json:"-"`
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Detail != "" {
		return fmt.Sprintf("%s: %s", e.Message, e.Detail)
	}
	return e.Message
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func IsCode(err error, code ErrorCode) bool {
	var target *Error
	return errors.As(err, &target) && target.Code == code
}

type Issue struct {
	Code     ErrorCode `json:"code"`
	Message  string    `json:"message"`
	Detail   string    `json:"detail,omitempty"`
	Path     string    `json:"path,omitempty"`
	Key      string    `json:"key,omitempty"`
	Blocking bool      `json:"blocking,omitempty"`
}

func newError(code ErrorCode, message, detail string) error {
	return &Error{Code: code, Message: message, Detail: detail}
}
