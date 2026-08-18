package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const macOSInstalledAppPath = "/Applications/AT Starter.app"

type macOSPackageInstallOps struct {
	attach   func(string) (string, error)
	detach   func(string) error
	install  func(string) error
	relaunch func(string) error
}

func installMacOSPackageDMG(dmgPath string, ops macOSPackageInstallOps) error {
	if _, err := os.Stat(dmgPath); err != nil {
		return fmt.Errorf("asset missing: %w", err)
	}
	mountPoint, err := ops.attach(dmgPath)
	if err != nil {
		return fmt.Errorf("hdiutil attach: %w", err)
	}

	var installErr error
	pkgPath, err := firstPackageIn(mountPoint)
	if err != nil {
		installErr = fmt.Errorf("find .pkg in dmg: %w", err)
	} else if err := ops.install(pkgPath); err != nil {
		installErr = fmt.Errorf("install package: %w", err)
	}
	detachErr := ops.detach(mountPoint)
	if detachErr != nil {
		detachErr = fmt.Errorf("hdiutil detach: %w", detachErr)
	}
	if installErr != nil || detachErr != nil {
		return errors.Join(installErr, detachErr)
	}

	if err := ops.relaunch(macOSInstalledAppPath); err != nil {
		return fmt.Errorf("schedule relaunch: %w", err)
	}
	return nil
}

func firstPackageIn(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".pkg") {
			return filepath.Join(dir, entry.Name()), nil
		}
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subEntries, _ := os.ReadDir(filepath.Join(dir, entry.Name()))
		for _, subEntry := range subEntries {
			if !subEntry.IsDir() && strings.HasSuffix(subEntry.Name(), ".pkg") {
				return filepath.Join(dir, entry.Name(), subEntry.Name()), nil
			}
		}
	}
	return "", errors.New("no .pkg found in mount")
}

func macOSPackageInstallerArgs(pkgPath string) []string {
	return []string{
		"-e", "on run argv",
		"-e", `do shell script "/usr/sbin/installer -pkg " & quoted form of (item 1 of argv) & " -target /" with administrator privileges`,
		"-e", "end run",
		pkgPath,
	}
}
