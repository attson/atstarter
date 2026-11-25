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
		Name           string `json:"name"`
		OutputFilename string `json:"outputfilename"`
		Info           struct {
			ProductName string `json:"productName"`
		} `json:"info"`
	}
	if err := json.Unmarshal(raw, &config); err != nil {
		t.Fatal(err)
	}
	if config.Name != "rootyard" {
		t.Errorf("name = %q, want rootyard", config.Name)
	}
	if config.OutputFilename != "rootyard" {
		t.Errorf("outputfilename = %q, want rootyard", config.OutputFilename)
	}
	if config.Info.ProductName != "Rootyard" {
		t.Errorf("productName = %q, want Rootyard", config.Info.ProductName)
	}
}

func TestLinuxPackagesExposeCanonicalCommand(t *testing.T) {
	packageScript := readReleaseFile(t, ".github/scripts/package-linux-deb.sh")
	for _, want := range []string{
		`bin="build/bin/rootyard"`,
		`install -Dm755 "$bin" "$root/usr/bin/rootyard"`,
		`Exec=rootyard`,
		`Package: rootyard`,
	} {
		if !strings.Contains(packageScript, want) {
			t.Errorf("package-linux-deb.sh does not contain %q", want)
		}
	}
	legacySlug := "at" + "starter"
	legacyAssetPrefix := "AT" + "-Starter"
	if strings.Contains(strings.ToLower(packageScript), legacySlug) || strings.Contains(packageScript, legacyAssetPrefix) {
		t.Error("package-linux-deb.sh must not contain legacy product names")
	}
	installerScript := readReleaseFile(t, "scripts/install-linux.sh")
	if !strings.Contains(installerScript, `-name "rootyard"`) {
		t.Error("install-linux.sh must locate the canonical rootyard archive member")
	}
	legacyDisplayName := "AT" + " Starter"
	if strings.Contains(strings.ToLower(installerScript), legacySlug) || strings.Contains(installerScript, legacyDisplayName) {
		t.Error("install-linux.sh must not accept legacy archive members")
	}
}

func TestWindowsInstallerExposesCanonicalCommandOnPath(t *testing.T) {
	installer := readReleaseFile(t, "build/windows/installer/project.nsi")
	for _, want := range []string{
		`-Action add -Directory "$INSTDIR"`,
		`-Action remove -Directory "$INSTDIR"`,
		`SendMessage ${HWND_BROADCAST} ${WM_SETTINGCHANGE} 0 "STR:Environment"`,
		`Rootyard`,
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
		`/Applications/Rootyard.app/Contents/MacOS/rootyard`,
		`cp "$component_pkg" "$staging/Rootyard.pkg"`,
		`--identifier "com.attson.rootyard"`,
	} {
		if !strings.Contains(packageScript, want) {
			t.Errorf("package-macos-dmg.sh does not contain %q", want)
		}
	}
	for _, forbidden := range []string{
		`cp -R "$app" "$staging/Rootyard.app"`,
		`ln -s /Applications "$staging/Applications"`,
	} {
		if strings.Contains(packageScript, forbidden) {
			t.Errorf("package-macos-dmg.sh still contains drag-copy layout %q", forbidden)
		}
	}
}

func TestMacOSBundleIdentifiersUseRootyard(t *testing.T) {
	production := readReleaseFile(t, "build/darwin/Info.plist")
	if !strings.Contains(production, "com.attson.rootyard") {
		t.Fatal("production bundle ID must be com.attson.rootyard")
	}
	development := readReleaseFile(t, "build/darwin/Info.dev.plist")
	if !strings.Contains(development, "com.attson.rootyard.dev") {
		t.Fatal("development bundle ID must be com.attson.rootyard.dev")
	}
}

func TestReleaseWorkflowUsesRootyardIdentity(t *testing.T) {
	workflow := readReleaseFile(t, ".github/workflows/build.yml")
	for _, want := range []string{
		`APP_ARTIFACT_NAME: 'Rootyard'`,
		`secrets.ROOTYARD_UPDATE_VERIFY_PUBLIC_KEY`,
		`ROOTYARD_UPDATE_SIGNING_PRIVATE_KEY: ${{ secrets.ROOTYARD_UPDATE_SIGNING_PRIVATE_KEY }}`,
		`working-directory: npm/rootyard-mcp`,
		`tar -czf "Rootyard-linux-${{ matrix.arch }}.tar.gz" rootyard`,
		`7z a "Rootyard-windows-amd64.zip" "rootyard.exe"`,
	} {
		if !strings.Contains(workflow, want) {
			t.Errorf("build.yml does not contain %q", want)
		}
	}

	signingScript := readReleaseFile(t, ".github/scripts/sign-release-checksums.go")
	if !strings.Contains(signingScript, `os.Getenv("ROOTYARD_UPDATE_SIGNING_PRIVATE_KEY")`) {
		t.Error("signing script must read ROOTYARD_UPDATE_SIGNING_PRIVATE_KEY")
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
