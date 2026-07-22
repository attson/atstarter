//go:build darwin

package main

/*
#cgo darwin CFLAGS: -x objective-c -fobjc-arc
#cgo darwin LDFLAGS: -framework Cocoa

#include <stdbool.h>

void atstarter_start_tray(const char *iconBytes, int length);
void atstarter_update_running(int n);
void atstarter_update_visible(bool visible);
*/
import "C"

import (
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var quitRequested atomic.Bool

var (
	trayApp     *App
	trayReady   bool
	trayMu      sync.Mutex
	trayVisible = true
)

func startTray(a *App) {
	trayApp = a
	if len(trayIcon) == 0 {
		C.atstarter_start_tray((*C.char)(nil), 0)
		return
	}
	C.atstarter_start_tray((*C.char)(unsafe.Pointer(&trayIcon[0])), C.int(len(trayIcon)))
}

func traySupported() bool {
	return true
}

//export darwinTrayReady
func darwinTrayReady() {
	trayMu.Lock()
	wasReady := trayReady
	trayReady = true
	trayMu.Unlock()

	if !wasReady && trayApp != nil && trayApp.runner != nil {
		updateTrayRunning(trayApp.runner.RunningCount())
	}
}

//export darwinTrayToggle
func darwinTrayToggle() {
	go toggleWindow()
}

//export darwinTrayStopAll
func darwinTrayStopAll() {
	if trayApp != nil && trayApp.runner != nil {
		go trayApp.runner.StopAll()
	}
}

//export darwinTrayQuit
func darwinTrayQuit() {
	quitRequested.Store(true)
	if trayApp != nil && trayApp.ctx != nil {
		go runtime.Quit(trayApp.ctx)
	}
}

func toggleWindow() {
	trayMu.Lock()
	defer trayMu.Unlock()
	if trayVisible {
		runtime.WindowHide(trayApp.ctx)
		trayVisible = false
	} else {
		runtime.WindowShow(trayApp.ctx)
		runtime.WindowUnminimise(trayApp.ctx)
		trayVisible = true
	}
	refreshToggleLabelLocked()
}

func setTrayWindowVisible(v bool) {
	trayMu.Lock()
	defer trayMu.Unlock()
	trayVisible = v
	refreshToggleLabelLocked()
}

func refreshToggleLabelLocked() {
	C.atstarter_update_visible(C.bool(trayVisible))
}

func updateTrayRunning(n int) {
	trayMu.Lock()
	defer trayMu.Unlock()
	C.atstarter_update_running(C.int(n))
}

func isTrayReady() bool {
	trayMu.Lock()
	defer trayMu.Unlock()
	return trayReady
}
