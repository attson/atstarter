package projectenv

import (
	"path/filepath"
	"testing"
)

func TestValidateConfigRejectsDuplicateNormalizedRulePaths(t *testing.T) {
	root := t.TempDir()
	cfg := Config{Version: 1, Rules: []Rule{
		{ID: "one", Name: "One", Path: root},
		{ID: "two", Name: "Two", Path: filepath.Join(root, ".")},
	}}

	err := ValidateConfig(cfg)
	if !IsCode(err, ErrDuplicateRulePath) {
		t.Fatalf("err = %v", err)
	}
}

func TestValidateConfigRejectsMissingPresetReference(t *testing.T) {
	cfg := Config{
		Version:  1,
		Defaults: LayerConfig{PresetID: "missing"},
	}
	if err := ValidateConfig(cfg); !IsCode(err, ErrPresetNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestValidateConfigRejectsInvalidFields(t *testing.T) {
	tests := []struct {
		name   string
		key    string
		intent FieldIntent
	}{
		{name: "unknown key", key: "core.editor", intent: FieldIntent{Mode: ModeSet, Value: "vim"}},
		{name: "unknown mode", key: "user.name", intent: FieldIntent{Mode: Mode("merge"), Value: "A"}},
		{name: "set missing value", key: "user.name", intent: FieldIntent{Mode: ModeSet}},
		{name: "inherit with value", key: "user.name", intent: FieldIntent{Mode: ModeInherit, Value: "A"}},
		{name: "clear with value", key: "user.name", intent: FieldIntent{Mode: ModeClear, Value: "A"}},
		{name: "invalid boolean", key: "commit.gpgSign", intent: FieldIntent{Mode: ModeSet, Value: "yes"}},
		{name: "invalid format", key: "gpg.format", intent: FieldIntent{Mode: ModeSet, Value: "pem"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{Version: 1, Defaults: LayerConfig{Git: FieldMap{tt.key: tt.intent}}}
			if err := ValidateConfig(cfg); !IsCode(err, ErrInvalidEnvironment) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}

func TestValidateConfigAcceptsSupportedFieldValues(t *testing.T) {
	cfg := Config{Version: 1, Defaults: LayerConfig{Git: FieldMap{
		"user.name":       {Mode: ModeSet, Value: "Ada"},
		"user.email":      {Mode: ModeSet, Value: "ada@example.com"},
		"user.signingKey": {Mode: ModeClear},
		"commit.gpgSign":  {Mode: ModeSet, Value: "true"},
		"tag.gpgSign":     {Mode: ModeSet, Value: "false"},
		"gpg.format":      {Mode: ModeSet, Value: "ssh"},
	}}}
	if err := ValidateConfig(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestValidateConfigAcceptsSupportedCodexFieldValues(t *testing.T) {
	cfg := Config{Version: 1, Defaults: LayerConfig{Codex: FieldMap{
		"model":                  {Mode: ModeSet, Value: "custom-model"},
		"model_reasoning_effort": {Mode: ModeSet, Value: "future-effort"},
		"sandbox_mode":           {Mode: ModeSet, Value: "workspace-write"},
		"approval_policy":        {Mode: ModeSet, Value: "granular"},
		"web_search":             {Mode: ModeSet, Value: "indexed"},
	}}}
	if err := ValidateConfig(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestValidateConfigRejectsInvalidCodexFields(t *testing.T) {
	tests := []struct {
		name   string
		key    string
		intent FieldIntent
	}{
		{name: "unknown key", key: "mcp_servers", intent: FieldIntent{Mode: ModeSet, Value: "enabled"}},
		{name: "blank model", key: "model", intent: FieldIntent{Mode: ModeSet, Value: "  "}},
		{name: "blank reasoning", key: "model_reasoning_effort", intent: FieldIntent{Mode: ModeSet, Value: ""}},
		{name: "invalid sandbox", key: "sandbox_mode", intent: FieldIntent{Mode: ModeSet, Value: "full"}},
		{name: "invalid approval", key: "approval_policy", intent: FieldIntent{Mode: ModeSet, Value: "always"}},
		{name: "boolean web search", key: "web_search", intent: FieldIntent{Mode: ModeSet, Value: "true"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Config{Version: 1, Defaults: LayerConfig{Codex: FieldMap{tt.key: tt.intent}}}
			if err := ValidateConfig(cfg); !IsCode(err, ErrInvalidEnvironment) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}
