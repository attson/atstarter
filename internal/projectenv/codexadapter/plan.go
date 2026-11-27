package codexadapter

import "rootyard/internal/projectenv"

func (a *Adapter) Plan(resolved projectenv.ResolvedEnvironment, snapshot projectenv.Snapshot) projectenv.ChangePlan {
	plan := projectenv.ChangePlan{
		Adapter: a.Name(), Target: snapshot.Target, RepoPath: resolved.RepoPath,
		Snapshot: snapshot, Fingerprint: snapshot.Fingerprint,
		Desired: map[string]projectenv.FieldIntent{}, Changes: []projectenv.Change{},
		Warnings: []projectenv.Issue{}, Blockers: []projectenv.Issue{},
		NativeEdits: []projectenv.TextEdit{}, NativeDiff: []string{},
	}
	for _, issue := range snapshot.Issues {
		if issue.Blocking {
			plan.Blockers = append(plan.Blockers, issue)
		} else {
			plan.Warnings = append(plan.Warnings, issue)
		}
	}
	doc, parseErr := parseDocument(snapshot.NativeContent)
	if parseErr != nil {
		plan.Blockers = append(plan.Blockers, projectenv.Issue{
			Code: projectenv.ErrInvalidCodexConfig, Message: "invalid Codex project configuration",
			Path: snapshot.Target.ConfigPath, Blocking: true,
		})
	}
	missing := []string{}
	for _, key := range projectenv.SupportedCodexKeys {
		field, managed := resolved.Codex[key]
		intent := projectenv.FieldIntent{Mode: projectenv.ModeInherit}
		source := projectenv.Source{}
		if managed {
			intent = field.Intent
			source = field.Source
		}
		plan.Desired[key] = intent
		currentValues := append([]projectenv.OriginValue(nil), snapshot.Values[key]...)
		operation := codexOperation(intent, currentValues, len(plan.Blockers) > 0)
		plan.Changes = append(plan.Changes, projectenv.Change{
			Key: key, Operation: operation, Current: currentValues, Desired: intent,
			Source: source, Issues: []projectenv.Issue{},
		})
		if parseErr != nil {
			continue
		}
		switch operation {
		case projectenv.OperationSet:
			missing = append(missing, key+" = "+tomlBasicString(intent.Value))
		case projectenv.OperationReplace:
			current := doc.Assignments[key]
			plan.NativeEdits = append(plan.NativeEdits, projectenv.TextEdit{
				Start: current.ValueStart, End: current.ValueEnd, Replacement: tomlBasicString(intent.Value),
			})
		case projectenv.OperationClear:
			plan.NativeEdits = append(plan.NativeEdits, clearEdit(doc, doc.Assignments[key]))
		}
	}
	if len(missing) > 0 && parseErr == nil {
		plan.NativeEdits = append(plan.NativeEdits, insertionEdit(doc, missing))
	}
	plan.NativeDiff = nativeDiff(snapshot.NativeContent, plan.NativeEdits)
	if dangerousCodexPermissions(resolved.Codex) {
		plan.Warnings = append(plan.Warnings, projectenv.Issue{
			Code:    projectenv.ErrDangerousCodexPermissions,
			Message: "Codex will run without sandboxing or approval prompts",
			Path:    snapshot.Target.ConfigPath,
		})
	}
	return plan
}

func codexOperation(intent projectenv.FieldIntent, current []projectenv.OriginValue, blocked bool) projectenv.Operation {
	if blocked {
		return projectenv.OperationBlocked
	}
	switch intent.Mode {
	case projectenv.ModeSet:
		if len(current) == 0 {
			return projectenv.OperationSet
		}
		if len(current) == 1 && current[0].Value == intent.Value {
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

func dangerousCodexPermissions(fields map[string]projectenv.ResolvedField) bool {
	sandbox, sandboxOK := fields["sandbox_mode"]
	approval, approvalOK := fields["approval_policy"]
	return sandboxOK && approvalOK &&
		sandbox.Intent.Mode == projectenv.ModeSet && sandbox.Intent.Value == "danger-full-access" &&
		approval.Intent.Mode == projectenv.ModeSet && approval.Intent.Value == "never"
}
