package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"rootyard/internal/projectenv"
)

func (a *App) GetEnvironmentDocument() (projectenv.Document, error) {
	return a.envService.GetDocument(a.environmentContext())
}

func (a *App) SaveEnvironmentDefaults(request projectenv.SaveDefaultsRequest) (projectenv.Document, error) {
	return a.envService.SaveDefaults(a.environmentContext(), request)
}

func (a *App) SaveEnvironmentPreset(request projectenv.SavePresetRequest) (projectenv.Document, error) {
	return a.envService.SavePreset(a.environmentContext(), request)
}

func (a *App) CloneEnvironmentPreset(request projectenv.ClonePresetRequest) (projectenv.Document, error) {
	return a.envService.ClonePreset(a.environmentContext(), request)
}

func (a *App) DeleteEnvironmentPreset(request projectenv.DeletePresetRequest) (projectenv.Document, error) {
	return a.envService.DeletePreset(a.environmentContext(), request)
}

func (a *App) SaveEnvironmentRule(request projectenv.SaveRuleRequest) (projectenv.Document, error) {
	return a.envService.SaveRule(a.environmentContext(), request)
}

func (a *App) DeleteEnvironmentRule(request projectenv.DeleteRuleRequest) (projectenv.Document, error) {
	return a.envService.DeleteRule(a.environmentContext(), request)
}

func (a *App) GetEnvironmentState() (projectenv.StateDocument, error) {
	return a.envService.State(a.environmentContext())
}

func (a *App) ResolveEnvironment(request projectenv.ResolveRequest) (projectenv.ResolvedEnvironment, error) {
	return a.envService.Resolve(a.environmentContext(), request)
}

func (a *App) PlanEnvironmentRepositories(request projectenv.PlanRequest) ([]projectenv.ChangePlan, error) {
	return a.envService.Plan(a.environmentContext(), request)
}

func (a *App) ApplyEnvironmentPlans(plans []projectenv.ChangePlan) []projectenv.ApplyResult {
	return a.envService.Apply(a.environmentContext(), plans)
}

func (a *App) DoctorEnvironments(request projectenv.DoctorRequest) ([]projectenv.Issue, error) {
	return a.envService.Doctor(a.environmentContext(), request)
}

func (a *App) StartEnvironmentScan(ruleIDs []string) (string, error) {
	return a.startEnvironmentScan(projectenv.ScanRequest{RuleIDs: ruleIDs})
}

func (a *App) CancelEnvironmentScan(scanID string) bool {
	a.environmentScanMu.Lock()
	cancel := a.environmentScans[scanID]
	a.environmentScanMu.Unlock()
	if cancel == nil {
		return false
	}
	cancel()
	return true
}

func (a *App) startAutomaticEnvironmentScan() {
	_, _ = a.startEnvironmentScan(projectenv.ScanRequest{AutoOnly: true})
}

func (a *App) startEnvironmentScan(request projectenv.ScanRequest) (string, error) {
	scanID, err := a.environmentScanID()
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithCancel(a.environmentContext())
	a.environmentScanMu.Lock()
	a.environmentScans[scanID] = cancel
	a.environmentScanMu.Unlock()

	go func() {
		result, scanErr := a.environmentScan(ctx, request, func(progress projectenv.ScanProgress) {
			a.environmentEmit("projectenv:scan-progress", map[string]any{
				"scanId":   scanID,
				"progress": progress,
			})
		})
		a.environmentScanMu.Lock()
		delete(a.environmentScans, scanID)
		a.environmentScanMu.Unlock()
		payload := map[string]any{"scanId": scanID, "result": result}
		if scanErr != nil {
			payload["error"] = scanErr.Error()
		}
		a.environmentEmit("projectenv:scan-complete", payload)
	}()
	return scanID, nil
}

func (a *App) cancelEnvironmentScans() {
	a.environmentScanMu.Lock()
	cancels := make([]context.CancelFunc, 0, len(a.environmentScans))
	for _, cancel := range a.environmentScans {
		cancels = append(cancels, cancel)
	}
	a.environmentScans = make(map[string]context.CancelFunc)
	a.environmentScanMu.Unlock()
	for _, cancel := range cancels {
		cancel()
	}
}

func (a *App) environmentContext() context.Context {
	if a.ctx != nil {
		return a.ctx
	}
	return context.Background()
}

func newEnvironmentScanID() (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return "scan_" + hex.EncodeToString(value), nil
}
