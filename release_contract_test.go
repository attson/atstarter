package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestWailsNamingSeparatesProductAndCommand(t *testing.T) {
	raw, err := os.ReadFile("wails.json")
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		OutputFilename string `json:"outputfilename"`
		Info           struct {
			ProductName string `json:"productName"`
		} `json:"info"`
	}
	if err := json.Unmarshal(raw, &config); err != nil {
		t.Fatal(err)
	}
	if config.OutputFilename != "atstarter" {
		t.Errorf("outputfilename = %q, want canonical command name atstarter", config.OutputFilename)
	}
	if config.Info.ProductName != "AT Starter" {
		t.Errorf("productName = %q, want desktop display name AT Starter", config.Info.ProductName)
	}
}

func TestLinuxPackagesExposeCanonicalCommand(t *testing.T) {
	packageScript := readReleaseFile(t, ".github/scripts/package-linux-deb.sh")
	for _, want := range []string{
		`bin="build/bin/atstarter"`,
		`install -Dm755 "$bin" "$root/usr/bin/atstarter"`,
		`Exec=atstarter`,
	} {
		if !strings.Contains(packageScript, want) {
			t.Errorf("package-linux-deb.sh does not contain %q", want)
		}
	}
	if strings.Contains(packageScript, `/usr/bin/AT-Starter`) {
		t.Error("package-linux-deb.sh must not install a legacy AT-Starter command")
	}
	installerScript := readReleaseFile(t, "scripts/install-linux.sh")
	if !strings.Contains(installerScript, `-name "atstarter"`) {
		t.Error("install-linux.sh must locate the canonical atstarter archive member")
	}
	if strings.Contains(installerScript, `-name "AT Starter"`) {
		t.Error("install-linux.sh must not accept the legacy AT Starter archive member")
	}

	workflow := readReleaseFile(t, ".github/workflows/build.yml")
	if !strings.Contains(workflow, `tar -czf "AT-Starter-linux-${{ matrix.arch }}.tar.gz" atstarter`) {
		t.Error("Linux tarball must contain the canonical atstarter executable")
	}
}

func TestWindowsInstallerExposesCanonicalCommandOnPath(t *testing.T) {
	workflow := readReleaseFile(t, ".github/workflows/build.yml")
	for _, want := range []string{
		`test -f "atstarter.exe"`,
		`7z a "AT-Starter-windows-amd64.zip" "atstarter.exe"`,
		`powershell.exe -NoProfile -ExecutionPolicy Bypass -File build/windows/installer/update-path.ps1 -SelfTest`,
	} {
		if !strings.Contains(workflow, want) {
			t.Errorf("build.yml does not contain %q", want)
		}
	}

	installer := readReleaseFile(t, "build/windows/installer/project.nsi")
	for _, want := range []string{
		`-Action add -Directory "$INSTDIR"`,
		`-Action remove -Directory "$INSTDIR"`,
		`SendMessage ${HWND_BROADCAST} ${WM_SETTINGCHANGE} 0 "STR:Environment"`,
	} {
		if !strings.Contains(installer, want) {
			t.Errorf("project.nsi does not contain %q", want)
		}
	}

	pathScript := readReleaseFile(t, "build/windows/installer/update-path.ps1")
	for _, want := range []string{"function Update-PathValue", "OrdinalIgnoreCase", "SetEnvironmentVariable"} {
		if !strings.Contains(pathScript, want) {
			t.Errorf("update-path.ps1 does not contain %q", want)
		}
	}
}

func TestMacOSDMGWrapsInstallerPackage(t *testing.T) {
	packageScript := readReleaseFile(t, ".github/scripts/package-macos-dmg.sh")
	for _, want := range []string{
		`pkgbuild --root "$pkgroot"`,
		`/Applications/AT Starter.app/Contents/MacOS/atstarter`,
		`cp "$component_pkg" "$staging/AT Starter.pkg"`,
	} {
		if !strings.Contains(packageScript, want) {
			t.Errorf("package-macos-dmg.sh does not contain %q", want)
		}
	}
	for _, forbidden := range []string{
		`cp -R "$app" "$staging/AT Starter.app"`,
		`ln -s /Applications "$staging/Applications"`,
	} {
		if strings.Contains(packageScript, forbidden) {
			t.Errorf("package-macos-dmg.sh still contains drag-copy layout %q", forbidden)
		}
	}
}

func readReleaseFile(t *testing.T, name string) string {
	t.Helper()
	raw, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
