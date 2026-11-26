package projectenv

import (
	"path/filepath"
	"testing"
)

func TestResolveLayersPresetRuleAndSession(t *testing.T) {
	cfg := Config{
		Version: 1,
		Defaults: LayerConfig{Git: FieldMap{
			"user.name": {Mode: ModeSet, Value: "Default Name"},
		}},
		Presets: []Preset{{ID: "company", Name: "Company", Git: FieldMap{
			"user.email": {Mode: ModeSet, Value: "company@example.com"},
		}}},
		Rules: []Rule{
			{ID: "outer", Path: "/work", Recursive: true, PresetID: "company"},
			{ID: "exact", Path: "/work/api", Git: FieldMap{
				"user.email": {Mode: ModeSet, Value: "api@example.com"},
			}},
		},
	}
	got, err := Resolve(cfg, "/work/api", FieldMap{
		"commit.gpgSign": {Mode: ModeSet, Value: "true"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Git["user.name"].Intent.Value != "Default Name" {
		t.Fatal(got)
	}
	if got.Git["user.email"].Intent.Value != "api@example.com" {
		t.Fatal(got)
	}
	if got.Git["user.email"].Source.ID != "exact" {
		t.Fatal(got)
	}
	if got.Git["commit.gpgSign"].Source.Kind != SourceSession {
		t.Fatal(got)
	}
}

func TestResolveAppliesDefaultsPresetBeforeInlineAndTracksSource(t *testing.T) {
	cfg := Config{
		Version: 1,
		Presets: []Preset{{ID: "base", Name: "Base", Git: FieldMap{
			"user.name":  {Mode: ModeSet, Value: "Preset Name"},
			"user.email": {Mode: ModeSet, Value: "preset@example.com"},
		}}},
		Defaults: LayerConfig{PresetID: "base", Git: FieldMap{
			"user.name": {Mode: ModeSet, Value: "Inline Name"},
		}},
	}

	got, err := Resolve(cfg, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if field := got.Git["user.name"]; field.Intent.Value != "Inline Name" || field.Source.Kind != SourceDefault {
		t.Fatalf("user.name = %+v", field)
	}
	if field := got.Git["user.email"]; field.Intent.Value != "preset@example.com" || field.Source.Kind != SourcePreset || field.Source.ID != "base" {
		t.Fatalf("user.email = %+v", field)
	}
}

func TestResolveOrdersMatchingRulesOuterToInner(t *testing.T) {
	root := t.TempDir()
	inner := filepath.Join(root, "team", "api")
	cfg := Config{Version: 1, Rules: []Rule{
		{ID: "inner", Name: "Inner", Path: filepath.Join(root, "team"), Recursive: true, Git: FieldMap{
			"user.email": {Mode: ModeSet, Value: "inner@example.com"},
		}},
		{ID: "outer", Name: "Outer", Path: root, Recursive: true, Git: FieldMap{
			"user.email": {Mode: ModeSet, Value: "outer@example.com"},
		}},
	}}

	got, err := Resolve(cfg, inner, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Git["user.email"].Intent.Value != "inner@example.com" {
		t.Fatalf("got %+v", got.Git["user.email"])
	}
	if len(got.MatchingRuleIDs) != 2 || got.MatchingRuleIDs[0] != "outer" || got.MatchingRuleIDs[1] != "inner" {
		t.Fatalf("matching rules = %#v", got.MatchingRuleIDs)
	}
}

func TestResolveExactRuleDoesNotMatchDescendant(t *testing.T) {
	root := t.TempDir()
	cfg := Config{Version: 1, Rules: []Rule{{
		ID: "exact", Path: root, Recursive: false,
		Git: FieldMap{"user.name": {Mode: ModeSet, Value: "Exact"}},
	}}}

	exact, err := Resolve(cfg, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if exact.Git["user.name"].Intent.Value != "Exact" {
		t.Fatalf("exact result = %+v", exact)
	}
	descendant, err := Resolve(cfg, filepath.Join(root, "child"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := descendant.Git["user.name"]; ok {
		t.Fatalf("exact rule matched descendant: %+v", descendant)
	}
}

func TestResolveInheritLeavesPriorValueAndClearReplacesIt(t *testing.T) {
	root := t.TempDir()
	cfg := Config{
		Version: 1,
		Defaults: LayerConfig{Git: FieldMap{
			"user.name":  {Mode: ModeSet, Value: "Default"},
			"user.email": {Mode: ModeSet, Value: "default@example.com"},
		}},
		Rules: []Rule{{ID: "rule", Path: root, Git: FieldMap{
			"user.name":  {Mode: ModeInherit},
			"user.email": {Mode: ModeClear},
		}}},
	}

	got, err := Resolve(cfg, root, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Git["user.name"].Intent.Value != "Default" {
		t.Fatalf("inherit changed value: %+v", got.Git["user.name"])
	}
	if field := got.Git["user.email"]; field.Intent.Mode != ModeClear || field.Source.ID != "rule" {
		t.Fatalf("clear result = %+v", field)
	}
}

func TestResolveRejectsMissingPreset(t *testing.T) {
	cfg := Config{Version: 1, Defaults: LayerConfig{PresetID: "missing"}}
	if _, err := Resolve(cfg, t.TempDir(), nil); !IsCode(err, ErrPresetNotFound) {
		t.Fatalf("err = %v", err)
	}
}
