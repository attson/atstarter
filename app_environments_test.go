package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"rootyard/internal/projectenv"
)

func TestEnvironmentAppDelegatesDocumentCRUD(t *testing.T) {
	app := newTestApp(t)
	doc, err := app.GetEnvironmentDocument()
	if err != nil {
		t.Fatal(err)
	}
	name := "Company"
	doc, err = app.SaveEnvironmentPreset(projectenv.SavePresetRequest{Revision: doc.Revision, Name: &name})
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Config.Presets) != 1 || doc.Config.Presets[0].Name != "Company" {
		t.Fatalf("document = %+v", doc)
	}
	clone, err := app.CloneEnvironmentPreset(projectenv.ClonePresetRequest{Revision: doc.Revision, ID: doc.Config.Presets[0].ID, Name: "Clone"})
	if err != nil {
		t.Fatal(err)
	}
	if len(clone.Config.Presets) != 2 {
		t.Fatalf("clone = %+v", clone)
	}
}

func TestEnvironmentAppScanIDsProgressCompletionAndCancellation(t *testing.T) {
	app := newTestApp(t)
	app.ctx = context.Background()
	ids := []string{"scan_one", "scan_two"}
	app.environmentScanID = func() (string, error) {
		id := ids[0]
		ids = ids[1:]
		return id, nil
	}
	started := make(chan projectenv.ScanRequest, 2)
	released := make(chan struct{})
	app.environmentScan = func(ctx context.Context, request projectenv.ScanRequest, emit func(projectenv.ScanProgress)) (projectenv.ScanResult, error) {
		started <- request
		emit(projectenv.ScanProgress{RuleID: "rule", Repositories: 1})
		select {
		case <-ctx.Done():
			return projectenv.ScanResult{Canceled: true}, ctx.Err()
		case <-released:
			return projectenv.ScanResult{Repositories: []projectenv.RepositoryCandidate{{Path: "/repo"}}}, nil
		}
	}
	type event struct {
		name    string
		payload any
	}
	var mu sync.Mutex
	var events []event
	complete := make(chan struct{}, 2)
	app.environmentEmit = func(name string, payload any) {
		mu.Lock()
		events = append(events, event{name: name, payload: payload})
		mu.Unlock()
		if name == "projectenv:scan-complete" {
			complete <- struct{}{}
		}
	}

	first, err := app.StartEnvironmentScan([]string{"rule"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := app.StartEnvironmentScan(nil)
	if err != nil {
		t.Fatal(err)
	}
	if first == second || first != "scan_one" || second != "scan_two" {
		t.Fatalf("scan IDs = %q %q", first, second)
	}
	<-started
	<-started
	if !app.CancelEnvironmentScan(first) {
		t.Fatal("expected first scan cancellation")
	}
	close(released)
	<-complete
	<-complete
	mu.Lock()
	defer mu.Unlock()
	progressCount := 0
	completeCount := 0
	for _, event := range events {
		switch event.name {
		case "projectenv:scan-progress":
			progressCount++
		case "projectenv:scan-complete":
			completeCount++
		}
	}
	if progressCount != 2 || completeCount != 2 {
		t.Fatalf("events = %+v", events)
	}
}

func TestEnvironmentAppShutdownCancelsAllScans(t *testing.T) {
	app := newTestApp(t)
	app.ctx = context.Background()
	canceled := make(chan struct{})
	app.environmentScan = func(ctx context.Context, _ projectenv.ScanRequest, _ func(projectenv.ScanProgress)) (projectenv.ScanResult, error) {
		<-ctx.Done()
		close(canceled)
		return projectenv.ScanResult{Canceled: true}, ctx.Err()
	}
	app.environmentEmit = func(string, any) {}
	if _, err := app.StartEnvironmentScan(nil); err != nil {
		t.Fatal(err)
	}
	app.cancelEnvironmentScans()
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("scan was not canceled")
	}
}

func TestEnvironmentAppAutomaticDiscoveryUsesAutoOnly(t *testing.T) {
	app := newTestApp(t)
	app.ctx = context.Background()
	request := make(chan projectenv.ScanRequest, 1)
	app.environmentScan = func(_ context.Context, got projectenv.ScanRequest, _ func(projectenv.ScanProgress)) (projectenv.ScanResult, error) {
		request <- got
		return projectenv.ScanResult{}, nil
	}
	app.environmentEmit = func(string, any) {}
	app.startAutomaticEnvironmentScan()
	select {
	case got := <-request:
		if !got.AutoOnly || len(got.RuleIDs) != 0 {
			t.Fatalf("request = %+v", got)
		}
	case <-time.After(time.Second):
		t.Fatal("automatic scan did not start")
	}
}
