package projectenv

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

type DiscoverOptions struct {
	RuleIDs  []string `json:"ruleIds,omitempty"`
	AutoOnly bool     `json:"autoOnly,omitempty"`
}

type RepositoryCandidate struct {
	Path    string   `json:"path"`
	RuleIDs []string `json:"ruleIds"`
}

type ScanProgress struct {
	RuleID       string `json:"ruleId"`
	Path         string `json:"path"`
	Repositories int    `json:"repositories"`
	Warnings     int    `json:"warnings"`
	Done         bool   `json:"done"`
}

type ScanResult struct {
	Repositories []RepositoryCandidate `json:"repositories"`
	Warnings     []Issue               `json:"warnings"`
	Canceled     bool                  `json:"canceled"`
}

func Discover(ctx context.Context, rules []Rule, opts DiscoverOptions, emit func(ScanProgress)) ScanResult {
	selected := make(map[string]struct{}, len(opts.RuleIDs))
	for _, id := range opts.RuleIDs {
		selected[id] = struct{}{}
	}
	candidates := make(map[string]*RepositoryCandidate)
	result := ScanResult{Repositories: []RepositoryCandidate{}, Warnings: []Issue{}}

	for _, rule := range rules {
		if ctx.Err() != nil {
			result.Canceled = true
			break
		}
		if opts.AutoOnly && !rule.AutoDiscover {
			continue
		}
		if len(selected) > 0 {
			if _, ok := selected[rule.ID]; !ok {
				continue
			}
		}

		root, err := CanonicalRulePath(rule.Path)
		if err != nil {
			result.Warnings = append(result.Warnings, scanIssue(rule.Path, err))
			emitScanProgress(emit, rule.ID, rule.Path, len(candidates), len(result.Warnings), true)
			continue
		}
		if !rule.Recursive {
			if hasGitMarker(root) {
				mergeRepositoryCandidate(candidates, root, rule.ID)
			}
			emitScanProgress(emit, rule.ID, root, len(candidates), len(result.Warnings), true)
			continue
		}

		walkErr := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if walkErr != nil {
				result.Warnings = append(result.Warnings, scanIssue(path, walkErr))
				emitScanProgress(emit, rule.ID, path, len(candidates), len(result.Warnings), false)
				if entry != nil && entry.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if !entry.IsDir() {
				return nil
			}
			if path != root && ignoredDiscoveryDirectory(entry.Name()) {
				return filepath.SkipDir
			}
			if hasGitMarker(path) {
				canonical, err := CanonicalRulePath(path)
				if err != nil {
					result.Warnings = append(result.Warnings, scanIssue(path, err))
					emitScanProgress(emit, rule.ID, path, len(candidates), len(result.Warnings), false)
					return filepath.SkipDir
				}
				mergeRepositoryCandidate(candidates, canonical, rule.ID)
				emitScanProgress(emit, rule.ID, canonical, len(candidates), len(result.Warnings), false)
				return filepath.SkipDir
			}
			emitScanProgress(emit, rule.ID, path, len(candidates), len(result.Warnings), false)
			return nil
		})
		if errors.Is(walkErr, context.Canceled) || errors.Is(walkErr, context.DeadlineExceeded) || ctx.Err() != nil {
			result.Canceled = true
			emitScanProgress(emit, rule.ID, root, len(candidates), len(result.Warnings), true)
			break
		}
		if walkErr != nil {
			result.Warnings = append(result.Warnings, scanIssue(root, walkErr))
		}
		emitScanProgress(emit, rule.ID, root, len(candidates), len(result.Warnings), true)
	}

	paths := make([]string, 0, len(candidates))
	for path := range candidates {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		result.Repositories = append(result.Repositories, *candidates[path])
	}
	return result
}

func hasGitMarker(path string) bool {
	_, err := os.Lstat(filepath.Join(path, ".git"))
	return err == nil
}

func ignoredDiscoveryDirectory(name string) bool {
	switch name {
	case ".git", "node_modules", "vendor", ".cache":
		return true
	default:
		return false
	}
}

func mergeRepositoryCandidate(candidates map[string]*RepositoryCandidate, path, ruleID string) {
	candidate, ok := candidates[path]
	if !ok {
		candidates[path] = &RepositoryCandidate{Path: path, RuleIDs: []string{ruleID}}
		return
	}
	for _, existing := range candidate.RuleIDs {
		if existing == ruleID {
			return
		}
	}
	candidate.RuleIDs = append(candidate.RuleIDs, ruleID)
}

func scanIssue(path string, err error) Issue {
	return Issue{Code: ErrScanWarning, Message: "unable to scan directory", Detail: err.Error(), Path: path}
}

func emitScanProgress(emit func(ScanProgress), ruleID, path string, repositories, warnings int, done bool) {
	if emit == nil {
		return
	}
	emit(ScanProgress{RuleID: ruleID, Path: path, Repositories: repositories, Warnings: warnings, Done: done})
}
