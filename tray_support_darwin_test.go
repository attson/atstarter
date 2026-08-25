//go:build darwin

package main

import "testing"

func TestDarwinDoesNotUseExternalSystrayLoop(t *testing.T) {
	if !traySupported() {
		t.Fatal("darwin should support a native tray without starting an external AppKit loop")
	}
}

func TestDarwinNativeWindowCloseDoesNotInterceptApplicationQuit(t *testing.T) {
	if !useNativeHideOnClose() {
		t.Fatal("darwin should let Wails hide the window natively so OnBeforeClose only handles application quit")
	}
	if shouldPreventClose(false, true, useNativeHideOnClose()) {
		t.Fatal("application quit must not be intercepted when native hide-on-close handles the window close button")
	}
}

func TestDarwinNativeVisibilityUpdatesTrayState(t *testing.T) {
	trayMu.Lock()
	original := trayVisible
	trayMu.Unlock()
	t.Cleanup(func() { setDarwinTrayVisibility(original) })

	setDarwinTrayVisibility(false)
	trayMu.Lock()
	visible := trayVisible
	trayMu.Unlock()
	if visible {
		t.Fatal("native application hide should mark the tray window state hidden")
	}

	setDarwinTrayVisibility(true)
	trayMu.Lock()
	visible = trayVisible
	trayMu.Unlock()
	if !visible {
		t.Fatal("native application unhide should mark the tray window state visible")
	}
}
