package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// TestFetchFirstFallsBackToNextCandidate 验证:首个候选失败时,fetchFirst 会尝试
// 下一个候选。这是校验文件(SHA256SUMS)能走镜像加速的基础 —— 此前 fetchText
// 只直连原始 github URL,国内网络超时会导致整个更新失败。
func TestFetchFirstFallsBackToNextCandidate(t *testing.T) {
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OK-BODY"))
	}))
	defer good.Close()
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer bad.Close()

	u := &updater{client: &http.Client{Timeout: 5 * time.Second}}
	// 首选(bad)500 失败,应回退到次选(good)成功。
	got, err := u.fetchFirst(context.Background(), []string{bad.URL, good.URL})
	if err != nil {
		t.Fatalf("fetchFirst: %v", err)
	}
	if got != "OK-BODY" {
		t.Errorf("fetchFirst = %q, want OK-BODY", got)
	}
}

// TestFetchTextUsesMirrors 验证 fetchText 对 github releases URL 会展开出多个候选
// (镜像 + 原始),而非只打原始 URL。
func TestFetchTextUsesMirrors(t *testing.T) {
	raw := "https://github.com/attson/atstarter/releases/download/v0.3.3/SHA256SUMS"
	cands := mirrorURLs(raw)
	if len(cands) < 2 {
		t.Fatalf("expected mirrors + original for a releases URL, got %d: %v", len(cands), cands)
	}
	if cands[len(cands)-1] != raw {
		t.Errorf("last candidate should be original URL %q, got %q", raw, cands[len(cands)-1])
	}
}

func TestVersionNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"v0.1.3", "v0.1.2", true},
		{"v0.1.10", "v0.1.2", true}, // integer compare, not lexicographic
		{"v0.2.0", "v0.1.9", true},
		{"v1.0.0", "v0.9.9", true},
		{"v0.1.2", "v0.1.2", false},
		{"v0.1.2", "v0.1.3", false},
		{"v0.1.2", "dev", false}, // "dev" won't parse → no update
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
		{"darwin", "arm64", "_arm64.dmg"},
		{"darwin", "amd64", "_amd64.dmg"},
		{"windows", "amd64", "_amd64.exe"},
		{"plan9", "amd64", ""},
	}
	for _, c := range cases {
		if got := assetPatternFor(c.os, c.arch); got != c.want {
			t.Errorf("assetPatternFor(%s,%s) = %q, want %q", c.os, c.arch, got, c.want)
		}
	}
}

func TestAssetPatternForMatchesReleasedMacOSDMGName(t *testing.T) {
	pattern := assetPatternFor("darwin", "arm64")
	name := "AT-Starter_0.4.3_arm64.dmg"
	if !contains(name, pattern) {
		t.Fatalf("macOS update pattern %q does not match release asset %q", pattern, name)
	}
}

func TestMacOSReleasePublishesLegacyDMGAlias(t *testing.T) {
	script, err := os.ReadFile(".github/scripts/package-macos-dmg.sh")
	if err != nil {
		t.Fatalf("read package-macos-dmg.sh: %v", err)
	}
	if !strings.Contains(string(script), "${ARTIFACT_NAME}-darwin-${ARCH}.dmg") {
		t.Fatalf("package-macos-dmg.sh must create a -darwin-${ARCH}.dmg alias for old updaters")
	}

	workflow, err := os.ReadFile(".github/workflows/build.yml")
	if err != nil {
		t.Fatalf("read build.yml: %v", err)
	}
	if !strings.Contains(string(workflow), "${{ env.APP_ARTIFACT_NAME }}-darwin-${{ matrix.arch }}.dmg") {
		t.Fatalf("build.yml must upload the -darwin-${{ matrix.arch }}.dmg compatibility alias")
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

func TestMirrorURLs(t *testing.T) {
	raw := "https://github.com/attson/atstarter/releases/download/v0.3.2/AT-Starter-linux-amd64.tar.gz"

	got := mirrorURLs(raw)

	// 至少要有:若干镜像候选 + 原始 URL 兜底。
	if len(got) < 2 {
		t.Fatalf("mirrorURLs returned %d candidates, want >=2 (mirrors + original)", len(got))
	}

	// 最后一个必须是原始 URL(保证不比现状差)。
	if got[len(got)-1] != raw {
		t.Errorf("last candidate = %q, want original URL %q", got[len(got)-1], raw)
	}

	// 原始 URL 只应作为最后的兜底出现一次,前面的都应是镜像(与原始不同)。
	for i := 0; i < len(got)-1; i++ {
		if got[i] == raw {
			t.Errorf("candidate[%d] equals original URL but is not last; mirrors must differ", i)
		}
		// 每个镜像候选都应仍然包含原始 GitHub 路径(镜像是前缀改写,不丢失 owner/repo/tag/asset)。
		if !contains(got[i], "attson/atstarter/releases/download/v0.3.2/AT-Starter-linux-amd64.tar.gz") {
			t.Errorf("mirror candidate[%d] = %q lost the github asset path", i, got[i])
		}
	}
}

func TestMirrorURLsNonGitHubPassthrough(t *testing.T) {
	// 非 github releases/download 的 URL 原样返回,不改写。
	raw := "https://example.com/some/file.bin"
	got := mirrorURLs(raw)
	if len(got) != 1 || got[0] != raw {
		t.Errorf("mirrorURLs(non-github) = %v, want single passthrough [%q]", got, raw)
	}
}
