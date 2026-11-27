package projectenv

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"time"
)

type SaveDefaultsRequest struct {
	Revision string   `json:"revision"`
	PresetID *string  `json:"presetId,omitempty"`
	Git      FieldMap `json:"git,omitempty"`
	Codex    FieldMap `json:"codex,omitempty"`
}

type SavePresetRequest struct {
	Revision    string   `json:"revision"`
	ID          string   `json:"id,omitempty"`
	Name        *string  `json:"name,omitempty"`
	Description *string  `json:"description,omitempty"`
	Git         FieldMap `json:"git,omitempty"`
	Codex       FieldMap `json:"codex,omitempty"`
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
	Codex        FieldMap `json:"codex,omitempty"`
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
	RepoPath string            `json:"repoPath"`
	Session  EnvironmentFields `json:"session,omitempty"`
}

type PlanRequest struct {
	RepositoryPaths []string          `json:"repositoryPaths,omitempty"`
	RuleIDs         []string          `json:"ruleIds,omitempty"`
	Session         EnvironmentFields `json:"session,omitempty"`
}

type DoctorRequest struct {
	RepositoryPaths []string          `json:"repositoryPaths,omitempty"`
	RuleIDs         []string          `json:"ruleIds,omitempty"`
	Session         EnvironmentFields `json:"session,omitempty"`
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
		cfg.Defaults.Codex = mergeFieldMap(cfg.Defaults.Codex, request.Codex)
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
			clone.Codex = cloneFieldMap(preset.Codex)
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
	selectedRules := selectedRules(doc.Config.Rules, request.RuleIDs, request.AutoOnly)
	scanID, err := s.idGenerator("scan_")
	if err != nil {
		return result, err
	}
	when := s.now().UTC().Format(time.RFC3339Nano)
	selectedScanStates := make([]RuleScanState, 0, len(selectedRules))
	for _, rule := range selectedRules {
		selectedScanStates = append(selectedScanStates, RuleScanState{RuleID: rule.ID, ScanID: scanID, LastScanAt: when, Warnings: warningsForRule(result.Warnings, rule)})
	}
	fullScan := len(request.RuleIDs) == 0 && !request.AutoOnly
	if err := s.state.Update(ctx, func(state *StateDocument) error {
		reconcileScanState(state, result.Repositories, selectedRules, selectedScanStates, fullScan)
		return nil
	}); err != nil {
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
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return nil, err
				}
				target = Target{Adapter: adapter.Name(), ID: path, WorktreePath: path}
				plans = append(plans, blockedAdapterPlan(adapter.Name(), path, resolved.MatchingRuleIDs, target, ErrInvalidRepository, "identify repository", err))
				continue
			}
			snapshot, err := adapter.Inspect(ctx, target)
			if err != nil {
				if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
					return nil, err
				}
				code := ErrInvalidGitConfig
				if adapter.Name() == "codex" {
					code = ErrInvalidCodexConfig
				}
				plans = append(plans, blockedAdapterPlan(adapter.Name(), path, resolved.MatchingRuleIDs, target, code, "inspect repository environment configuration", err))
				continue
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
				code := ErrInvalidGitConfig
				message := "inspect repository environment configuration"
				detail := err.Error()
				if adapter.Name() == "codex" {
					code = ErrInvalidCodexConfig
				}
				var environmentErr *Error
				if errors.As(err, &environmentErr) {
					code = environmentErr.Code
					message = environmentErr.Message
					detail = environmentErr.Detail
				}
				issuePath := target.ConfigPath
				if issuePath == "" {
					issuePath = path
				}
				issues = append(issues, Issue{Code: code, Message: message, Detail: detail, Path: issuePath, Blocking: true})
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
	preset.Codex = mergeFieldMap(preset.Codex, request.Codex)
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
	rule.Codex = mergeFieldMap(rule.Codex, request.Codex)
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
			Adapters:        map[string]AdapterState{},
		})
	}
	return records
}

func reconcileScanState(state *StateDocument, candidates []RepositoryCandidate, selected []Rule, scans []RuleScanState, fullScan bool) {
	selectedIDs := make(map[string]struct{}, len(selected))
	for _, rule := range selected {
		selectedIDs[rule.ID] = struct{}{}
	}

	records := make(map[string]RepositoryRecord, len(state.Repositories))
	order := make([]string, 0, len(state.Repositories)+len(candidates))
	for _, record := range state.Repositories {
		filtered := make([]RepositoryAlias, 0, len(record.Aliases))
		for _, alias := range record.Aliases {
			remaining := make([]string, 0, len(alias.RuleIDs))
			removedMembership := false
			for _, ruleID := range alias.RuleIDs {
				_, replacing := selectedIDs[ruleID]
				if fullScan || replacing {
					removedMembership = true
					continue
				}
				remaining = append(remaining, ruleID)
			}
			if len(remaining) > 0 || !removedMembership {
				alias.RuleIDs = remaining
				filtered = append(filtered, alias)
			}
		}
		record.Aliases = filtered
		records[record.ID] = record
		order = append(order, record.ID)
	}

	for _, candidate := range candidates {
		record, exists := records[candidate.Path]
		if !exists {
			record = RepositoryRecord{ID: candidate.Path, Status: "pending", Adapters: map[string]AdapterState{}}
			order = append(order, candidate.Path)
		}
		aliasIndex := -1
		for index := range record.Aliases {
			if record.Aliases[index].Path == candidate.Path {
				aliasIndex = index
				break
			}
		}
		if aliasIndex < 0 {
			record.Aliases = append(record.Aliases, RepositoryAlias{Path: candidate.Path})
			aliasIndex = len(record.Aliases) - 1
		}
		record.Aliases[aliasIndex].RuleIDs = dedupeStrings(append(record.Aliases[aliasIndex].RuleIDs, candidate.RuleIDs...))
		record.MatchingRuleIDs = matchingRuleIDs(record.Aliases)
		records[candidate.Path] = record
	}

	state.Repositories = make([]RepositoryRecord, 0, len(records))
	seen := map[string]struct{}{}
	for _, id := range order {
		if _, duplicate := seen[id]; duplicate {
			continue
		}
		seen[id] = struct{}{}
		record := records[id]
		if len(record.Aliases) == 0 {
			continue
		}
		record.MatchingRuleIDs = matchingRuleIDs(record.Aliases)
		state.Repositories = append(state.Repositories, record)
	}

	keptScans := make([]RuleScanState, 0, len(state.RuleScans)+len(scans))
	for _, scan := range state.RuleScans {
		_, replacing := selectedIDs[scan.RuleID]
		if !fullScan && !replacing {
			keptScans = append(keptScans, scan)
		}
	}
	state.RuleScans = append(keptScans, scans...)
}

func matchingRuleIDs(aliases []RepositoryAlias) []string {
	var ids []string
	for _, alias := range aliases {
		ids = append(ids, alias.RuleIDs...)
	}
	return dedupeStrings(ids)
}

func blockedAdapterPlan(adapter, path string, ruleIDs []string, target Target, defaultCode ErrorCode, defaultMessage string, err error) ChangePlan {
	issue := Issue{Code: defaultCode, Message: defaultMessage, Detail: err.Error(), Path: target.ConfigPath, Blocking: true}
	var typed *Error
	if errors.As(err, &typed) {
		issue.Code = typed.Code
		issue.Message = typed.Message
		issue.Detail = typed.Detail
	}
	if issue.Path == "" {
		issue.Path = path
	}
	return ChangePlan{
		Adapter: adapter, Target: target, RepoPath: path,
		Aliases:         []RepositoryAlias{{Path: path, RuleIDs: append([]string(nil), ruleIDs...)}},
		MatchingRuleIDs: append([]string(nil), ruleIDs...), Desired: map[string]FieldIntent{}, Changes: []Change{},
		Warnings: []Issue{}, Blockers: []Issue{issue}, NativeEdits: []TextEdit{}, NativeDiff: []string{},
	}
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
	_ = s.state.Update(ctx, func(state *StateDocument) error {
		byTarget := make(map[string][]int, len(state.Repositories))
		byAlias := make(map[string]int, len(state.Repositories))
		for index, repository := range state.Repositories {
			for adapter, adapterState := range repository.Adapters {
				if adapterState.TargetID != "" {
					key := adapter + "\x00" + adapterState.TargetID
					byTarget[key] = append(byTarget[key], index)
				}
			}
			for _, alias := range repository.Aliases {
				byAlias[alias.Path] = index
			}
		}
		for index, result := range results {
			if index >= len(plans) {
				continue
			}
			plan := plans[index]
			recordIndexes := append([]int(nil), byTarget[plan.Adapter+"\x00"+result.Target.ID]...)
			for _, alias := range plan.Aliases {
				if recordIndex, exists := byAlias[alias.Path]; exists && !containsInt(recordIndexes, recordIndex) {
					recordIndexes = append(recordIndexes, recordIndex)
				}
			}
			if len(recordIndexes) == 0 && plan.RepoPath != "" {
				if recordIndex, exists := byAlias[plan.RepoPath]; exists {
					recordIndexes = append(recordIndexes, recordIndex)
				}
			}
			if len(recordIndexes) == 0 {
				aliases := append([]RepositoryAlias(nil), plan.Aliases...)
				if len(aliases) == 0 && plan.RepoPath != "" {
					aliases = append(aliases, RepositoryAlias{Path: plan.RepoPath})
				}
				recordID := plan.RepoPath
				if recordID == "" {
					recordID = result.Target.WorktreePath
				}
				recordIndexes = append(recordIndexes, len(state.Repositories))
				state.Repositories = append(state.Repositories, RepositoryRecord{ID: recordID, Aliases: aliases, Adapters: map[string]AdapterState{}})
			}
			for _, recordIndex := range recordIndexes {
				record := &state.Repositories[recordIndex]
				if record.Adapters == nil {
					record.Adapters = map[string]AdapterState{}
				}
				record.Adapters[plan.Adapter] = AdapterState{
					TargetID: result.Target.ID, Fingerprint: result.Fingerprint, Status: statusForApply(result.Status),
					LastApplyAt: s.now().UTC().Format(time.RFC3339Nano), LastApplyResult: string(result.Status),
					Blockers: append([]Issue(nil), result.Issues...),
				}
				record.Status = aggregateAdapterStatus(record.Adapters)
			}
		}
		return nil
	})
}

func containsInt(values []int, target int) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func aggregateAdapterStatus(adapters map[string]AdapterState) string {
	severity := map[string]int{"consistent": 1, "pending": 2, "drifted": 3, "blocked": 4, "failed": 5}
	best := ""
	bestSeverity := 0
	for _, adapter := range adapters {
		if severity[adapter.Status] > bestSeverity {
			best = adapter.Status
			bestSeverity = severity[adapter.Status]
		}
	}
	if best == "" {
		return "pending"
	}
	return best
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
