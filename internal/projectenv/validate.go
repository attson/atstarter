package projectenv

import (
	"fmt"
	"strings"
)

func ValidateConfig(cfg Config) error {
	if cfg.Version != 1 {
		return newError(ErrUnsupportedVersion, "unsupported project environment version", fmt.Sprintf("version %d", cfg.Version))
	}

	presets := make(map[string]Preset, len(cfg.Presets))
	for _, preset := range cfg.Presets {
		if strings.TrimSpace(preset.ID) == "" || strings.TrimSpace(preset.Name) == "" {
			return newError(ErrInvalidEnvironment, "preset ID and name are required", preset.ID)
		}
		if _, exists := presets[preset.ID]; exists {
			return newError(ErrInvalidEnvironment, "duplicate preset ID", preset.ID)
		}
		if err := validateFields(preset.Git); err != nil {
			return err
		}
		if err := validateCodexFields(preset.Codex); err != nil {
			return err
		}
		presets[preset.ID] = preset
	}

	if err := validatePresetReference(cfg.Defaults.PresetID, presets); err != nil {
		return err
	}
	if err := validateFields(cfg.Defaults.Git); err != nil {
		return err
	}
	if err := validateCodexFields(cfg.Defaults.Codex); err != nil {
		return err
	}

	ruleIDs := make(map[string]struct{}, len(cfg.Rules))
	rulePaths := make(map[string]string, len(cfg.Rules))
	for _, rule := range cfg.Rules {
		if strings.TrimSpace(rule.ID) == "" || strings.TrimSpace(rule.Path) == "" {
			return newError(ErrInvalidEnvironment, "rule ID and path are required", rule.ID)
		}
		if _, exists := ruleIDs[rule.ID]; exists {
			return newError(ErrInvalidEnvironment, "duplicate rule ID", rule.ID)
		}
		ruleIDs[rule.ID] = struct{}{}
		path, err := canonicalPathWithoutIO(rule.Path)
		if err != nil {
			return newError(ErrInvalidEnvironment, "invalid directory rule path", err.Error())
		}
		if existing, exists := rulePaths[path]; exists {
			return newError(ErrDuplicateRulePath, "duplicate directory rule path", existing+" and "+rule.ID)
		}
		rulePaths[path] = rule.ID
		if err := validatePresetReference(rule.PresetID, presets); err != nil {
			return err
		}
		if err := validateFields(rule.Git); err != nil {
			return err
		}
		if err := validateCodexFields(rule.Codex); err != nil {
			return err
		}
	}

	return nil
}

func ValidateEnvironmentFields(fields EnvironmentFields) error {
	if err := validateFields(fields.Git); err != nil {
		return err
	}
	return validateCodexFields(fields.Codex)
}

func validatePresetReference(id string, presets map[string]Preset) error {
	if id == "" {
		return nil
	}
	if _, ok := presets[id]; !ok {
		return newError(ErrPresetNotFound, "environment preset not found", id)
	}
	return nil
}

func validateFields(fields FieldMap) error {
	supported := make(map[string]struct{}, len(SupportedGitKeys))
	for _, key := range SupportedGitKeys {
		supported[key] = struct{}{}
	}
	for key, intent := range fields {
		if _, ok := supported[key]; !ok {
			return newError(ErrInvalidEnvironment, "unsupported Git environment field", key)
		}
		switch intent.Mode {
		case ModeSet:
			if strings.TrimSpace(intent.Value) == "" {
				return newError(ErrInvalidEnvironment, "set field requires a value", key)
			}
		case ModeInherit, ModeClear:
			if intent.Value != "" {
				return newError(ErrInvalidEnvironment, "inherit and clear fields cannot contain a value", key)
			}
		default:
			return newError(ErrInvalidEnvironment, "invalid field mode", key)
		}
		if intent.Mode != ModeSet {
			continue
		}
		switch key {
		case "commit.gpgSign", "tag.gpgSign":
			if intent.Value != "true" && intent.Value != "false" {
				return newError(ErrInvalidEnvironment, "Git signing flag must be true or false", key)
			}
		case "gpg.format":
			if intent.Value != "openpgp" && intent.Value != "ssh" && intent.Value != "x509" {
				return newError(ErrInvalidEnvironment, "unsupported Git signing format", intent.Value)
			}
		}
	}
	return nil
}

func validateCodexFields(fields FieldMap) error {
	supported := make(map[string]struct{}, len(SupportedCodexKeys))
	for _, key := range SupportedCodexKeys {
		supported[key] = struct{}{}
	}
	for key, intent := range fields {
		if _, ok := supported[key]; !ok {
			return newError(ErrInvalidEnvironment, "unsupported Codex environment field", key)
		}
		switch intent.Mode {
		case ModeSet:
			if strings.TrimSpace(intent.Value) == "" {
				return newError(ErrInvalidEnvironment, "set field requires a value", key)
			}
		case ModeInherit, ModeClear:
			if intent.Value != "" {
				return newError(ErrInvalidEnvironment, "inherit and clear fields cannot contain a value", key)
			}
		default:
			return newError(ErrInvalidEnvironment, "invalid field mode", key)
		}
		if intent.Mode != ModeSet {
			continue
		}
		switch key {
		case "sandbox_mode":
			if !oneOf(intent.Value, "read-only", "workspace-write", "danger-full-access") {
				return newError(ErrInvalidEnvironment, "unsupported Codex sandbox mode", intent.Value)
			}
		case "approval_policy":
			if !oneOf(intent.Value, "untrusted", "on-failure", "on-request", "granular", "never") {
				return newError(ErrInvalidEnvironment, "unsupported Codex approval policy", intent.Value)
			}
		case "web_search":
			if !oneOf(intent.Value, "disabled", "cached", "indexed", "live") {
				return newError(ErrInvalidEnvironment, "unsupported Codex web search mode", intent.Value)
			}
		}
	}
	return nil
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
