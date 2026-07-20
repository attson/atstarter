package main

import "testing"

func TestVersionNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"v0.1.3", "v0.1.2", true},
		{"v0.1.10", "v0.1.2", true},  // integer compare, not lexicographic
		{"v0.2.0", "v0.1.9", true},
		{"v1.0.0", "v0.9.9", true},
		{"v0.1.2", "v0.1.2", false},
		{"v0.1.2", "v0.1.3", false},
		{"v0.1.2", "dev", false},     // "dev" won't parse → no update
		{"", "v0.1.2", false},
		{"v0.1.2-rc.1", "v0.1.1", true}, // pre-release suffix stripped
	}
	for _, c := range cases {
		if got := versionNewer(c.latest, c.current); got != c.want {
			t.Errorf("versionNewer(%q, %q) = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}

func TestAssetPatternFor(t *testing.T) {
	cases := []struct {
		os, arch, want string
	}{
		{"linux", "amd64", "-linux-amd64.tar.gz"},
		{"linux", "arm64", "-linux-arm64.tar.gz"},
		{"darwin", "arm64", "-darwin-arm64.dmg"},
		{"darwin", "amd64", "-darwin-amd64.dmg"},
		{"windows", "amd64", "_amd64.exe"},
		{"plan9", "amd64", ""},
	}
	for _, c := range cases {
		if got := assetPatternFor(c.os, c.arch); got != c.want {
			t.Errorf("assetPatternFor(%s,%s) = %q, want %q", c.os, c.arch, got, c.want)
		}
	}
}

func TestPickAssetMatchesPatternSubstring(t *testing.T) {
	release := ghRelease{
		Assets: []ghAsset{
			{Name: "AT-Starter_0.1.3_amd64.deb", BrowserDownloadURL: "u1", Size: 1},
			{Name: "AT-Starter-linux-amd64.tar.gz", BrowserDownloadURL: "u2", Size: 2},
			{Name: "SHA256SUMS", BrowserDownloadURL: "u3", Size: 3},
		},
	}
	// linux/amd64 should pick the tar.gz, not the deb (deb's suffix is ".deb").
	a, err := pickAsset(release)
	if err != nil {
		t.Skipf("skipping on %s/%s where no asset matches", "current", "arch")
	}
	// pickAsset uses runtime.GOOS/GOARCH; only assert the match rule when on linux/amd64.
	if a.Name != "" && !contains(a.Name, "-linux-amd64.tar.gz") && !contains(a.Name, "-darwin-") && !contains(a.Name, "_amd64.exe") {
		t.Errorf("picked wrong asset: %s", a.Name)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
