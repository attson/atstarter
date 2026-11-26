package projectenv

import "sort"

func Resolve(cfg Config, repoPath string, session EnvironmentFields) (ResolvedEnvironment, error) {
	if err := ValidateConfig(cfg); err != nil {
		return ResolvedEnvironment{}, err
	}
	repoPath, err := canonicalResolutionPath(repoPath)
	if err != nil {
		return ResolvedEnvironment{}, newError(ErrInvalidEnvironment, "invalid repository path", err.Error())
	}

	presets := make(map[string]Preset, len(cfg.Presets))
	for _, preset := range cfg.Presets {
		presets[preset.ID] = preset
	}
	result := ResolvedEnvironment{
		RepoPath: repoPath,
		Git:      make(map[string]ResolvedField),
		Codex:    make(map[string]ResolvedField),
	}
	applyFields := func(destination map[string]ResolvedField, fields FieldMap, source Source) {
		for key, intent := range fields {
			if intent.Mode == ModeInherit {
				continue
			}
			destination[key] = ResolvedField{Intent: intent, Source: source}
		}
	}
	applyLayer := func(presetID string, git, codex FieldMap, source Source) {
		if presetID != "" {
			preset := presets[presetID]
			presetSource := Source{Kind: SourcePreset, ID: preset.ID, Name: preset.Name}
			applyFields(result.Git, preset.Git, presetSource)
			applyFields(result.Codex, preset.Codex, presetSource)
		}
		applyFields(result.Git, git, source)
		applyFields(result.Codex, codex, source)
	}

	applyLayer(cfg.Defaults.PresetID, cfg.Defaults.Git, cfg.Defaults.Codex, Source{Kind: SourceDefault, ID: "defaults", Name: "Global Defaults"})

	matching := make([]Rule, 0, len(cfg.Rules))
	for _, rule := range cfg.Rules {
		rulePath, pathErr := canonicalResolutionPath(rule.Path)
		if pathErr != nil {
			return ResolvedEnvironment{}, newError(ErrInvalidEnvironment, "invalid directory rule path", pathErr.Error())
		}
		exact := rulePath == repoPath
		if exact || (rule.Recursive && PathContains(rulePath, repoPath)) {
			matching = append(matching, rule)
		}
	}
	sort.SliceStable(matching, func(i, j int) bool {
		left, _ := canonicalResolutionPath(matching[i].Path)
		right, _ := canonicalResolutionPath(matching[j].Path)
		if len(left) == len(right) {
			return left < right
		}
		return len(left) < len(right)
	})
	for _, rule := range matching {
		result.MatchingRuleIDs = append(result.MatchingRuleIDs, rule.ID)
		applyLayer(rule.PresetID, rule.Git, rule.Codex, Source{Kind: SourceRule, ID: rule.ID, Name: rule.Name})
	}

	if err := validateFields(session.Git); err != nil {
		return ResolvedEnvironment{}, err
	}
	if err := validateCodexFields(session.Codex); err != nil {
		return ResolvedEnvironment{}, err
	}
	sessionSource := Source{Kind: SourceSession, ID: "session", Name: "Session overrides"}
	applyFields(result.Git, session.Git, sessionSource)
	applyFields(result.Codex, session.Codex, sessionSource)
	return result, nil
}
