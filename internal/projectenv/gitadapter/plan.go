package gitadapter

import "rootyard/internal/projectenv"

func (a *Adapter) Plan(resolved projectenv.ResolvedEnvironment, snapshot projectenv.Snapshot) projectenv.ChangePlan {
	plan := projectenv.ChangePlan{
		Adapter:         a.Name(),
		Target:          snapshot.Target,
		RepoPath:        resolved.RepoPath,
		MatchingRuleIDs: append([]string(nil), resolved.MatchingRuleIDs...),
		Snapshot:        snapshot,
		Fingerprint:     snapshot.Fingerprint,
		Desired:         make(map[string]projectenv.FieldIntent, len(projectenv.SupportedGitKeys)),
		Changes:         make([]projectenv.Change, 0, len(projectenv.SupportedGitKeys)),
		Warnings:        []projectenv.Issue{},
		Blockers:        []projectenv.Issue{},
	}
	for _, issue := range snapshot.Issues {
		if issue.Blocking {
			plan.Blockers = append(plan.Blockers, issue)
		} else {
			plan.Warnings = append(plan.Warnings, issue)
		}
	}
	for _, key := range projectenv.SupportedGitKeys {
		resolvedField, managed := resolved.Git[key]
		intent := projectenv.FieldIntent{Mode: projectenv.ModeInherit}
		if managed {
			intent = resolvedField.Intent
		}
		plan.Desired[key] = intent
		current := append([]projectenv.OriginValue(nil), snapshot.Values[key]...)
		keyIssues := blockingIssuesForKey(snapshot.Issues, key)
		operation := planOperation(intent, current, len(keyIssues) > 0)
		plan.Changes = append(plan.Changes, projectenv.Change{
			Key:       key,
			Operation: operation,
			Current:   current,
			Desired:   intent,
			Source:    resolvedField.Source,
			Issues:    keyIssues,
		})
	}
	return plan
}

func blockingIssuesForKey(issues []projectenv.Issue, key string) []projectenv.Issue {
	var result []projectenv.Issue
	for _, issue := range issues {
		if issue.Blocking && (issue.Key == "" || issue.Key == key) {
			result = append(result, issue)
		}
	}
	return result
}

func planOperation(intent projectenv.FieldIntent, current []projectenv.OriginValue, blocked bool) projectenv.Operation {
	if blocked {
		return projectenv.OperationBlocked
	}
	switch intent.Mode {
	case projectenv.ModeSet:
		if len(current) == 0 {
			return projectenv.OperationSet
		}
		if len(current) == 1 && current[0].MainConfig && current[0].Value == intent.Value {
			return projectenv.OperationUnchanged
		}
		return projectenv.OperationReplace
	case projectenv.ModeClear:
		if len(current) == 0 {
			return projectenv.OperationUnchanged
		}
		return projectenv.OperationClear
	default:
		return projectenv.OperationUnmanaged
	}
}
