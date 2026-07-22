//go:build darwin

package main

import "testing"

func TestDarwinDoesNotUseExternalSystrayLoop(t *testing.T) {
	if !traySupported() {
		t.Fatal("darwin should support a native tray without starting an external AppKit loop")
	}
}
