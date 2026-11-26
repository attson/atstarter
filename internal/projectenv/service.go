package projectenv

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"time"
)

type SaveDefaultsRequest struct {
	Revision string   `json:"revision"`
	PresetID *string  `json:"presetId,omitempty"`
	Git      FieldMap `json:"git,omitempty"`
}

type SavePresetRequest struct {
	Revision    string   `json:"revision"`
	ID          string   `json:"id,omitempty"`
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	Git         FieldMap `json:"git,omitempty"`
}

type ClonePresetRequest struct {
	Revision string `json:"revision"`
	ID       string `json:"id"`
	Name     string `json:"name"`
}

type DeletePresetRequest struct {
	Revision string `json:"revision"`
	ID       string `json:"id"`
}

type SaveRuleRequest struct {
	Revision     string   `json:"revision"`
	ID           string   `json:"id,omitempty"`
	Name         *string  `json:"name,omitempty"`
	Path         *string  `json:"path,omitempty"`
	Recursive    *bool    `json:"recursive,omitempty"`
	AutoDiscover *bool    `json:"autoDiscover,omitempty"`
	PresetID     *string  `json:"presetId,omitempty"`
	Git          FieldMap `json:"git,omitempty"`
}

type DeleteRuleRequest struct {
	Revision string `json:"revision"`
	ID       string `json:"id"`
}

type ScanRequest struct {
	RuleIDs  []string `json:"ruleIds,omitempty"`
	AutoOnly bool     `json:"autoOnly,omitempty"`
}

type ResolveRequest struct {
	RepoPath string   `json:"repoPath"`
	Session  FieldMap `json:"session,omitempty"`
}

type PlanRequest struct {
	RepositoryPaths []string `json:"repositoryPaths,omitempty"`
	RuleIDs         []string `json:"ruleIds,omitempty"`
	Session         FieldMap `json:"session,omitempty"`
}

type DoctorRequest struct {
	RepositoryPaths []string `json:"repositoryPaths,omitempty"`
	RuleIDs         []string `json:"ruleIds,omitempty"`
	Session         FieldMap `json:"session,omitempty"`
}

type Service struct {
	store         *Store
	state         *StateStore
	adapters      []Adapter
	adapterByName map[string]Adapter
	idGenerator   func(string) (string, error)
	now           func() time.Time
}

func NewService(configPath string, adapters ...Adapter) *Service {
	byName := make(map[string]Adapter, len(adapters))
	for _, adapter := range adapters {
		byName[adapter.Name()] = adapter
	}
	return &Service{
		store:         NewStore(configPath),
		state:         NewStateStore(configPath),
		adapters:      append([]Adapter(nil), adapters...),
		adapterByName: byName,
		idGenerator:   generateOpaqueID,
		now:           time.Now,
	}
}

func (s *Service) GetDocument(ctx context.Context) (Document, error) {
	return s.store.Load(ctx)
}

func (s *Service) SaveDefaults(ctx context.Context, request SaveDefaultsRequest) (Document, error) {
	return s.store.Update(ctx, request.Revision, func(cfg *Config) error {
		if request.PresetID != nil {
			cfg.Defaults.PresetID = *request.PresetID
		}
		cfg.Defaults.Git = mergeFieldMap(cfg.Defaults.Git, request.Git)
		return nil
	})
}

func (s *Service) SavePreset(ctx context.Context, request SavePresetRequest) (Document, error) {
	return s.store.Update(ctx, request.Revision, func(cfg *Config) error {
		if request.ID == "" {
			id, err := s.idGenerator("preset_")
			if err != nil {
				return err
			}
			preset := Preset{ID: id, Git: FieldMap{}}
			applyPresetRequest(&preset, request)
			cfg.Presets = append(cfg.Presets, preset)
			return nil
		}
		for index := range cfg.Presets {
			if cfg.Presets[index].ID == request.ID {
				applyPresetRequest(&cfg.Presets[index], request)
				return nil
			}
		}
		return newError(ErrPresetNotFound, "environment preset not found", request.ID)
	})
}

func (s *Service) ClonePreset(ctx context.Context, request ClonePresetRequest) (Document, error) {
	return s.store.Update(ctx, request.Revision, func(cfg *Config) error {
		for _, preset := range cfg.Presets {
			if preset.ID != request.ID {
				continue
			}
			id, err := s.idGenerator("preset_")
			if err != nil {
				return err
			}
			clone := preset
			clone.ID = id
			clone.Name = request.Name
			clone.Git = cloneFieldMap(preset.Git)
			cfg.Presets = append(cfg.Presets, clone)
			return nil
		}
		return newError(ErrPresetNotFound, "environment preset not found", request.ID)
	})
}

func (s *Service) DeletePreset(ctx context.Context, request DeletePresetRequest) (Document, error) {
	return s.store.Update(ctx, request.Revision, func(cfg *Config) error {
		if cfg.Defaults.PresetID == request.ID {
			return newError(ErrPresetInUse, "environment preset is referenced by global defaults", request.ID)
		}
		for _, rule := range cfg.Rules {
			if rule.PresetID == request.ID {
				return newError(ErrPresetInUse, "environment preset is referenced by a directory rule", rule.ID)
			}
		}
		for index, preset := range cfg.Presets {
			if preset.ID == request.ID {
				cfg.Presets = append(cfg.Presets[:index], cfg.Presets[index+1:]...)
				return nil
			}
		}
		return newError(ErrPresetNotFound, "environment preset not found", request.ID)
	})
}

func (s *Service) SaveRule(ctx context.Context, request SaveRuleRequest) (Document, error) {
	return s.store.Update(ctx, request.Revision, func(cfg *Config) error {
		if request.ID == "" {
			id, err := s.idGenerator("rule_")
			if err != nil {
				return err
			}
			rule := Rule{ID: id, Git: FieldMap{}}
			if err := applyRuleRequest(&rule, request); err != nil {
				return err
			}
			cfg.Rules = append(cfg.Rules, rule)
			return nil
		}
		for index := range cfg.Rules {
			if cfg.Rules[index].ID == request.ID {
				return applyRuleRequest(&cfg.Rules[index], request)
			}
		}
		return newError(ErrRuleNotFound, "directory rule not found", request.ID)
	})
}

func (s *Service) DeleteRule(ctx context.Context, request DeleteRuleRequest) (Document, error) {
	return s.store.Update(ctx, request.Revision, func(cfg *Config) error {
		for index, rule := range cfg.Rules {
			if rule.ID == request.ID {
				cfg.Rules = append(cfg.Rules[:index], cfg.Rules[index+1:]...)
				return nil
			}
		}
		return newError(ErrRuleNotFound, "directory rule not found", request.ID)
	})
}

func (s *Service) Scan(ctx context.Context, request ScanRequest, emit func(ScanProgress)) (ScanResult, error) {
	doc, err := s.store.Load(ctx)
	if err != nil {
		return ScanResult{}, err
	}
	result := Discover(ctx, doc.Config.Rules, DiscoverOptions{RuleIDs: request.RuleIDs, AutoOnly: request.AutoOnly}, emit)
	state, err := s.state.Load(ctx)
	if err != nil {
		return result, err
	}
	state.Repositories = repositoryRecordsFromCandidates(result.Repositories)
	selectedRules := selectedRules(doc.Config.Rules, request.RuleIDs, request.AutoOnly)
	scanID, err := s.idGenerator("scan_")
	if err != nil {
		return result, err
	}
	when := s.now().UTC().Format(time.RFC3339Nano)
	state.RuleScans = make([]RuleScanState, 0, len(selectedRules))
	for _, rule := range selectedRules {
		state.RuleScans = append(state.RuleScans, RuleScanState{RuleID: rule.ID, ScanID: scanID, LastScanAt: when, Warnings: warningsForRule(result.Warnings, rule)})
	}
	if err := s.state.Save(ctx, state); err != nil {
		return result, err
	}
	return result, nil
}

func (s *Service) State(ctx context.Context) (StateDocument, error) {
	return s.state.Load(ctx)
}

func (s *Service) Resolve(ctx context.Context, request ResolveRequest) (ResolvedEnvironment, error) {
	doc, err := s.store.Load(ctx)
	if err != nil {
		return ResolvedEnvironment{}, err
	}
	return Resolve(doc.Config, request.RepoPath, request.Session)
}

func (s *Service) Plan(ctx context.Context, request PlanRequest) ([]ChangePlan, error) {
	doc, err := s.store.Load(ctx)
	if err != nil {
		return nil, err
	}
	paths, err := s.planPaths(ctx, request.RepositoryPaths, request.RuleIDs)
	if err != nil {
		return nil, err
	}
	type groupedPlan struct {
		index       int
		desiredHash string
	}
	groups := make(map[string]groupedPlan)
	plans := make([]ChangePlan, 0, len(paths)*max(1, len(s.adapters)))
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		resolved, err := Resolve(doc.Config, path, request.Session)
		if err != nil {
			return nil, err
		}
		for _, adapter := range s.adapters {
			target, err := adapter.Identify(ctx, path)
			if err != nil {
				return nil, err
			}
			snapshot, err := adapter.Inspect(ctx, target)
			if err != nil {
				return nil, err
			}
			plan := adapter.Plan(resolved, snapshot)
			plan.Adapter = adapter.Name()
			plan.Aliases = []RepositoryAlias{{Path: path, RuleIDs: append([]string(nil), resolved.MatchingRuleIDs...)}}
			hash := desiredHash(plan.Desired)
			groupKey := adapter.Name() + "\x00" + target.ID
			if group, exists := groups[groupKey]; exists {
				existing := &plans[group.index]
				existing.Aliases = append(existing.Aliases, plan.Aliases...)
				if group.desiredHash != hash && !hasIssueCode(existing.Blockers, ErrSharedTargetConflict) {
					existing.Blockers = append(existing.Blockers, Issue{
						Code: ErrSharedTargetConflict, Message: "linked worktrees resolve to different environment targets", Detail: target.ID, Blocking: true,
					})
				}
				continue
			}
			groups[groupKey] = groupedPlan{index: len(plans), desiredHash: hash}
			plans = append(plans, plan)
		}
	}
	return plans, nil
}

func (s *Service) Apply(ctx context.Context, plans []ChangePlan) []ApplyResult {
	results := make([]ApplyResult, 0, len(plans))
	for _, plan := range plans {
		adapter := s.adapterByName[plan.Adapter]
		if adapter == nil {
			results = append(results, ApplyResult{Target: plan.Target, Status: ApplyFailed, Issues: []Issue{{
				Code: ErrApplyFailed, Message: "environment adapter is unavailable", Detail: plan.Adapter, Blocking: true,
			}}})
			continue
		}
		results = append(results, adapter.Apply(ctx, plan))
	}
	s.persistApplyResults(ctx, plans, results)
	return results
}

func (s *Service) Doctor(ctx context.Context, request DoctorRequest) ([]Issue, error) {
	doc, err := s.store.Load(ctx)
	if err != nil {
		return nil, err
	}
	issues := []Issue{}
	for _, rule := range doc.Config.Rules {
		if _, err := os.Stat(rule.Path); err != nil {
			issues = append(issues, Issue{Code: ErrInvalidEnvironment, Message: "directory rule path is unavailable", Detail: err.Error(), Path: rule.Path, Blocking: true})
		}
	}
	paths, err := s.planPaths(ctx, request.RepositoryPaths, request.RuleIDs)
	if err != nil {
		return issues, err
	}
	for _, path := range paths {
		resolved, err := Resolve(doc.Config, path, request.Session)
		if err != nil {
			return issues, err
		}
		for _, adapter := range s.adapters {
			target, err := adapter.Identify(ctx, path)
			if err != nil {
				issues = append(issues, Issue{Code: ErrInvalidRepository, Message: "identify repository", Detail: err.Error(), Path: path, Blocking: true})
				continue
			}
			snapshot, err := adapter.Inspect(ctx, target)
			if err != nil {
				issues = append(issues, Issue{Code: ErrInvalidGitConfig, Message: "inspect repository configuration", Detail: err.Error(), Path: path, Blocking: true})
				continue
			}
			issues = append(issues, snapshot.Issues...)
			issues = append(issues, adapter.Doctor(ctx, target, resolved)...)
		}
	}
	return issues, nil
}

func applyPresetRequest(preset *Preset, request SavePresetRequest) {
	if request.Name != nil {
		preset.Name = *request.Name
	}
	if request.Description != nil {
		preset.Description = *request.Description
	}
	preset.Git = mergeFieldMap(preset.Git, request.Git)
}

func applyRuleRequest(rule *Rule, request SaveRuleRequest) error {
	if request.Name != nil {
		rule.Name = *request.Name
	}
	if request.Path != nil {
		path, err := CanonicalRulePath(*request.Path)
		if err != nil {
			return err
		}
		rule.Path = path
	}
	if request.Recursive != nil {
		rule.Recursive = *request.Recursive
	}
	if request.AutoDiscover != nil {
		rule.AutoDiscover = *request.AutoDiscover
	}
	if request.PresetID != nil {
		rule.PresetID = *request.PresetID
	}
	rule.Git = mergeFieldMap(rule.Git, request.Git)
	return nil
}

func mergeFieldMap(current, updates FieldMap) FieldMap {
	result := cloneFieldMap(current)
	if result == nil {
		result = FieldMap{}
	}
	for key, value := range updates {
		result[key] = value
	}
	return result
}

func cloneFieldMap(fields FieldMap) FieldMap {
	if fields == nil {
		return nil
	}
	result := make(FieldMap, len(fields))
	for key, value := range fields {
		result[key] = value
	}
	return result
}

func generateOpaqueID(prefix string) (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(bytes), nil
}

func selectedRules(rules []Rule, ids []string, autoOnly bool) []Rule {
	selected := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		selected[id] = struct{}{}
	}
	var result []Rule
	for _, rule := range rules {
		if autoOnly && !rule.AutoDiscover {
			continue
		}
		if len(selected) > 0 {
			if _, ok := selected[rule.ID]; !ok {
				continue
			}
		}
		result = append(result, rule)
	}
	return result
}

func warningsForRule(warnings []Issue, rule Rule) []Issue {
	var result []Issue
	for _, warning := range warnings {
		if PathContains(rule.Path, warning.Path) || warning.Path == rule.Path {
			result = append(result, warning)
		}
	}
	return result
}

func repositoryRecordsFromCandidates(candidates []RepositoryCandidate) []RepositoryRecord {
	records := make([]RepositoryRecord, 0, len(candidates))
	for _, candidate := range candidates {
		records = append(records, RepositoryRecord{
			ID:              candidate.Path,
			Aliases:         []RepositoryAlias{{Path: candidate.Path, RuleIDs: append([]string(nil), candidate.RuleIDs...)}},
			MatchingRuleIDs: append([]string(nil), candidate.RuleIDs...),
			Status:          "pending",
		})
	}
	return records
}

func (s *Service) planPaths(ctx context.Context, explicit, ruleIDs []string) ([]string, error) {
	if len(explicit) > 0 {
		return dedupeStrings(explicit), nil
	}
	state, err := s.state.Load(ctx)
	if err != nil {
		return nil, err
	}
	wantedRules := make(map[string]struct{}, len(ruleIDs))
	for _, id := range ruleIDs {
		wantedRules[id] = struct{}{}
	}
	var paths []string
	for _, repository := range state.Repositories {
		for _, alias := range repository.Aliases {
			if len(wantedRules) == 0 || intersectsRules(alias.RuleIDs, wantedRules) {
				paths = append(paths, alias.Path)
			}
		}
	}
	return dedupeStrings(paths), nil
}

func intersectsRules(ids []string, wanted map[string]struct{}) bool {
	for _, id := range ids {
		if _, ok := wanted[id]; ok {
			return true
		}
	}
	return false
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func desiredHash(desired map[string]FieldIntent) string {
	data, _ := json.Marshal(desired)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func hasIssueCode(issues []Issue, code ErrorCode) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func (s *Service) persistApplyResults(ctx context.Context, plans []ChangePlan, results []ApplyResult) {
	state, err := s.state.Load(ctx)
	if err != nil {
		return
	}
	byID := make(map[string]int, len(state.Repositories))
	for index, repository := range state.Repositories {
		key := repository.TargetID
		if key == "" {
			key = repository.ID
		}
		byID[key] = index
	}
	for index, result := range results {
		recordIndex, exists := byID[result.Target.ID]
		if !exists {
			recordIndex = len(state.Repositories)
			byID[result.Target.ID] = recordIndex
			aliases := []RepositoryAlias{}
			if index < len(plans) {
				aliases = append(aliases, plans[index].Aliases...)
				if len(aliases) == 0 && plans[index].RepoPath != "" {
					aliases = append(aliases, RepositoryAlias{Path: plans[index].RepoPath})
				}
			}
			state.Repositories = append(state.Repositories, RepositoryRecord{ID: result.Target.ID, TargetID: result.Target.ID, Aliases: aliases})
		}
		record := &state.Repositories[recordIndex]
		record.TargetID = result.Target.ID
		record.Fingerprint = result.Fingerprint
		record.LastApplyAt = s.now().UTC().Format(time.RFC3339Nano)
		record.LastApplyResult = string(result.Status)
		record.Status = statusForApply(result.Status)
		record.Blockers = append([]Issue(nil), result.Issues...)
	}
	_ = s.state.Save(ctx, state)
}

func statusForApply(status ApplyStatus) string {
	switch status {
	case ApplySuccess, ApplyUnchanged:
		return "consistent"
	case ApplyBlocked, ApplyStale:
		return "blocked"
	case ApplyFailed:
		return "failed"
	default:
		return "pending"
	}
}
