package projectenv

import "sort"

func Resolve(cfg Config, repoPath string, session FieldMap) (ResolvedEnvironment, error) {
	if err := ValidateConfig(cfg); err != nil {
		return ResolvedEnvironment{}, err
	}
	repoPath, err := canonicalPathWithoutIO(repoPath)
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
	}
	applyFields := func(fields FieldMap, source Source) {
		for key, intent := range fields {
			if intent.Mode == ModeInherit {
				continue
			}
			result.Git[key] = ResolvedField{Intent: intent, Source: source}
		}
	}
	applyLayer := func(presetID string, inline FieldMap, source Source) {
		if presetID != "" {
			preset := presets[presetID]
			applyFields(preset.Git, Source{Kind: SourcePreset, ID: preset.ID, Name: preset.Name})
		}
		applyFields(inline, source)
	}

	applyLayer(cfg.Defaults.PresetID, cfg.Defaults.Git, Source{Kind: SourceDefault, ID: "defaults", Name: "Global Defaults"})

	matching := make([]Rule, 0, len(cfg.Rules))
	for _, rule := range cfg.Rules {
		rulePath, pathErr := canonicalPathWithoutIO(rule.Path)
		if pathErr != nil {
			return ResolvedEnvironment{}, newError(ErrInvalidEnvironment, "invalid directory rule path", pathErr.Error())
		}
		exact := rulePath == repoPath
		if exact || (rule.Recursive && PathContains(rulePath, repoPath)) {
			matching = append(matching, rule)
		}
	}
	sort.SliceStable(matching, func(i, j int) bool {
		left, _ := canonicalPathWithoutIO(matching[i].Path)
		right, _ := canonicalPathWithoutIO(matching[j].Path)
		if len(left) == len(right) {
			return left < right
		}
		return len(left) < len(right)
	})
	for _, rule := range matching {
		result.MatchingRuleIDs = append(result.MatchingRuleIDs, rule.ID)
		applyLayer(rule.PresetID, rule.Git, Source{Kind: SourceRule, ID: rule.ID, Name: rule.Name})
	}

	if err := validateFields(session); err != nil {
		return ResolvedEnvironment{}, err
	}
	applyFields(session, Source{Kind: SourceSession, ID: "session", Name: "Session overrides"})
	return result, nil
}
