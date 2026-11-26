package projectenv

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

type StateDocument struct {
	Version      int                `json:"version"`
	RuleScans    []RuleScanState    `json:"ruleScans"`
	Repositories []RepositoryRecord `json:"repositories"`
}

type RuleScanState struct {
	RuleID     string  `json:"ruleId"`
	ScanID     string  `json:"scanId,omitempty"`
	LastScanAt string  `json:"lastScanAt,omitempty"`
	Warnings   []Issue `json:"warnings,omitempty"`
}

type RepositoryRecord struct {
	ID                 string            `json:"id"`
	TargetID           string            `json:"targetId,omitempty"`
	Aliases            []RepositoryAlias `json:"aliases"`
	MatchingRuleIDs    []string          `json:"matchingRuleIds,omitempty"`
	WinningSourceIDs   []string          `json:"winningSourceIds,omitempty"`
	Fingerprint        string            `json:"fingerprint,omitempty"`
	ResolvedTargetHash string            `json:"resolvedTargetHash,omitempty"`
	Status             string            `json:"status,omitempty"`
	LastApplyAt        string            `json:"lastApplyAt,omitempty"`
	LastApplyResult    string            `json:"lastApplyResult,omitempty"`
	Blockers           []Issue           `json:"blockers,omitempty"`
}

type RepositoryAlias struct {
	Path    string   `json:"path"`
	RuleIDs []string `json:"ruleIds,omitempty"`
}

type StateStore struct {
	paths Paths
	now   func() time.Time
}

func NewStateStore(configPath string) *StateStore {
	return &StateStore{paths: PathsForConfig(configPath), now: time.Now}
}

func (s *StateStore) Load(ctx context.Context) (StateDocument, error) {
	var state StateDocument
	err := withFileLock(ctx, s.paths.StateLock, func() error {
		data, err := os.ReadFile(s.paths.State)
		if errors.Is(err, os.ErrNotExist) {
			state = emptyState()
			return nil
		}
		if err != nil {
			return err
		}
		if err := decodeStrictJSON(data, &state); err != nil || state.Version != 1 {
			suffix := s.now().UTC().Format("20060102T150405.000000000Z")
			if renameErr := os.Rename(s.paths.State, s.paths.State+".corrupt-"+suffix); renameErr != nil {
				return fmt.Errorf("quarantine corrupt environment state: %w", renameErr)
			}
			state = emptyState()
			return nil
		}
		normalizeStateCollections(&state)
		return nil
	})
	return state, err
}

func (s *StateStore) Save(ctx context.Context, state StateDocument) error {
	return withFileLock(ctx, s.paths.StateLock, func() error {
		if state.Version == 0 {
			state.Version = 1
		}
		if state.Version != 1 {
			return newError(ErrUnsupportedVersion, "unsupported project environment state version", fmt.Sprintf("version %d", state.Version))
		}
		normalizeStateCollections(&state)
		data, err := json.MarshalIndent(state, "", "  ")
		if err != nil {
			return err
		}
		data = append(data, '\n')
		return atomicWriteFile(s.paths.State, data, 0o600)
	})
}

func emptyState() StateDocument {
	return StateDocument{Version: 1, RuleScans: []RuleScanState{}, Repositories: []RepositoryRecord{}}
}

func normalizeStateCollections(state *StateDocument) {
	if state.RuleScans == nil {
		state.RuleScans = []RuleScanState{}
	}
	if state.Repositories == nil {
		state.Repositories = []RepositoryRecord{}
	}
}
