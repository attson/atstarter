package main

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestInstallMacOSPackageDMGRunsInstallerBeforeRelaunch(t *testing.T) {
	dmg := filepath.Join(t.TempDir(), "AT-Starter.dmg")
	if err := os.WriteFile(dmg, []byte("dmg"), 0o644); err != nil {
		t.Fatal(err)
	}
	mount := t.TempDir()
	pkg := filepath.Join(mount, "AT Starter.pkg")
	if err := os.WriteFile(pkg, []byte("pkg"), 0o644); err != nil {
		t.Fatal(err)
	}

	var calls []string
	ops := macOSPackageInstallOps{
		attach: func(path string) (string, error) {
			calls = append(calls, "attach:"+path)
			return mount, nil
		},
		detach: func(path string) error {
			calls = append(calls, "detach:"+path)
			return nil
		},
		install: func(path string) error {
			calls = append(calls, "install:"+path)
			return nil
		},
		relaunch: func(path string) error {
			calls = append(calls, "relaunch:"+path)
			return nil
		},
	}

	if err := installMacOSPackageDMG(dmg, ops); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"attach:" + dmg,
		"install:" + pkg,
		"detach:" + mount,
		"relaunch:/Applications/AT Starter.app",
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %#v, want %#v", calls, want)
	}
}

func TestInstallMacOSPackageDMGDetachesAndDoesNotRelaunchAfterInstallFailure(t *testing.T) {
	dmg := filepath.Join(t.TempDir(), "AT-Starter.dmg")
	if err := os.WriteFile(dmg, []byte("dmg"), 0o644); err != nil {
		t.Fatal(err)
	}
	mount := t.TempDir()
	if err := os.WriteFile(filepath.Join(mount, "AT Starter.pkg"), []byte("pkg"), 0o644); err != nil {
		t.Fatal(err)
	}

	installErr := errors.New("authorization denied")
	detached := false
	relaunched := false
	err := installMacOSPackageDMG(dmg, macOSPackageInstallOps{
		attach:  func(string) (string, error) { return mount, nil },
		detach:  func(string) error { detached = true; return nil },
		install: func(string) error { return installErr },
		relaunch: func(string) error {
			relaunched = true
			return nil
		},
	})

	if !errors.Is(err, installErr) {
		t.Fatalf("error = %v, want install error", err)
	}
	if !detached {
		t.Error("DMG was not detached after installer failure")
	}
	if relaunched {
		t.Error("app must not relaunch after installer failure")
	}
}

func TestMacOSPackageInstallerArgsPassPackagePathAsAppleScriptArgument(t *testing.T) {
	pkg := "/Volumes/AT Starter/AT Starter.pkg"
	got := macOSPackageInstallerArgs(pkg)
	want := []string{
		"-e", "on run argv",
		"-e", `do shell script "/usr/sbin/installer -pkg " & quoted form of (item 1 of argv) & " -target /" with administrator privileges`,
		"-e", "end run",
		pkg,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("args = %#v, want %#v", got, want)
	}
}

func TestInstallMacOSPackageDMGReportsDetachFailure(t *testing.T) {
	dmg := filepath.Join(t.TempDir(), "AT-Starter.dmg")
	if err := os.WriteFile(dmg, []byte("dmg"), 0o644); err != nil {
		t.Fatal(err)
	}
	mount := t.TempDir()
	if err := os.WriteFile(filepath.Join(mount, "AT Starter.pkg"), []byte("pkg"), 0o644); err != nil {
		t.Fatal(err)
	}

	detachErr := errors.New("device busy")
	relaunched := false
	err := installMacOSPackageDMG(dmg, macOSPackageInstallOps{
		attach:  func(string) (string, error) { return mount, nil },
		detach:  func(string) error { return detachErr },
		install: func(string) error { return nil },
		relaunch: func(string) error {
			relaunched = true
			return nil
		},
	})

	if !errors.Is(err, detachErr) {
		t.Fatalf("error = %v, want detach error", err)
	}
	if relaunched {
		t.Error("app must not relaunch while the DMG is still mounted")
	}
}
