// Auto-update subsystem. Polls GitHub Releases, downloads the platform
// asset with progress reporting, verifies its SHA-256 against a
// SHA256SUMS file whose Ed25519 signature is checked with
// UpdateVerifyPublicKey (embedded via -ldflags at build time), then
// applies the update by handing off to a per-OS install script.
//
// State is held on the App and pushed to the frontend as "update:state"
// events. All exported methods return a fresh UpdateState snapshot so
// callers can reason about the outcome without waiting for the event.
package main

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed scripts/install-darwin.sh
//go:embed scripts/install-linux.sh
//go:embed scripts/install-windows.ps1
var installScripts embed.FS

// GitHub repo the updater polls. Kept in-code (not a build var) so the
// origin is auditable and does not depend on env at runtime.
const updateRepoOwner = "attson"
const updateRepoName = "atstarter"

// downloadMirrors 是一组 GitHub releases 下载加速镜像前缀(国内直连 github.com
// 常极慢甚至卡住)。每个前缀直接拼在完整 github.com URL 之前,形如
// "https://<mirror>/https://github.com/owner/repo/releases/download/...".
// 顺序即优先级;download 会逐个尝试,任一失败/超时就换下一个,全部失败最后回退
// 原始 github.com URL。镜像仅用于加速,内容安全由 Ed25519 签名 + SHA256 校验兜底
// (见 verify),即使镜像被污染也无法通过安装。
var downloadMirrors = []string{
	"https://ghfast.top/",
	"https://gh-proxy.com/",
	"https://ghproxy.net/",
}

// mirrorURLs 把一个 github.com releases 下载 URL 展开为按优先级排序的候选列表:
// 先各镜像前缀改写,最后是原始 URL 兜底。非 github releases/download 的 URL 原样
// 返回单元素列表(不改写 —— 例如已是镜像或本地路径)。
func mirrorURLs(raw string) []string {
	const marker = "https://github.com/"
	// 只加速标准的 github releases 下载直链。
	if !strings.HasPrefix(raw, marker) || !strings.Contains(raw, "/releases/download/") {
		return []string{raw}
	}
	out := make([]string, 0, len(downloadMirrors)+1)
	for _, m := range downloadMirrors {
		out = append(out, m+raw)
	}
	out = append(out, raw) // 原始 URL 永远作为最后兜底,保证不比现状差。
	return out
}

// UpdateState mirrors the Wails-marshalable view of the updater. Fields
// use JSON tags so the frontend can bind to the same shape.
type UpdateState struct {
	Current     string `json:"current"`
	Latest      string `json:"latest"`
	Available   bool   `json:"available"`
	Notes       string `json:"notes"`
	Checking    bool   `json:"checking"`
	LastCheckAt int64  `json:"lastCheckAt"`
	Downloading bool   `json:"downloading"`
	DownloadPct int    `json:"downloadPct"`
	Ready       bool   `json:"ready"`
	Error       string `json:"error"`
	AssetURL    string `json:"assetUrl"`
	AssetSize   int64  `json:"assetSize"`
	CanInstall  bool   `json:"canInstall"` // false when UpdateVerifyPublicKey is empty
}

// updater lives on the App. All mutation goes through mu.
type updater struct {
	mu          sync.Mutex
	state       UpdateState
	client      *http.Client
	assetPath   string // full path of the downloaded, verified asset
	cancel      context.CancelFunc
	downloading atomic.Bool
}

func newUpdater() *updater {
	return &updater{
		client: &http.Client{Timeout: 30 * time.Second},
		state: UpdateState{
			Current:    Version,
			CanInstall: UpdateVerifyPublicKey != "",
		},
	}
}

// emit pushes the current state to the frontend.
func (u *updater) emit(ctx context.Context) {
	if ctx == nil {
		return
	}
	u.mu.Lock()
	snapshot := u.state
	u.mu.Unlock()
	wailsruntime.EventsEmit(ctx, "update:state", snapshot)
}

// snapshot returns a copy under the lock.
func (u *updater) snapshot() UpdateState {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.state
}

// setError updates the error field and clears in-flight flags.
func (u *updater) setError(err error) {
	u.mu.Lock()
	u.state.Error = err.Error()
	u.state.Checking = false
	u.state.Downloading = false
	u.state.DownloadPct = 0
	u.mu.Unlock()
}

// -----------------------------------------------------------------
// GitHub release lookup
// -----------------------------------------------------------------

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}
type ghRelease struct {
	TagName string    `json:"tag_name"`
	Body    string    `json:"body"`
	Assets  []ghAsset `json:"assets"`
}

// assetPatternFor returns a substring match pattern for the current
// platform's release archive (e.g. "-linux-amd64.tar.gz"). Windows uses
// the NSIS installer .exe; macOS uses the drag-in DMG.
func assetPatternFor(goos, goarch string) string {
	switch goos {
	case "linux":
		return "-linux-" + goarch + ".tar.gz"
	case "darwin":
		return "-darwin-" + goarch + ".dmg"
	case "windows":
		return "_" + goarch + ".exe"
	}
	return ""
}

// pickAsset returns the release asset that matches the current platform.
func pickAsset(r ghRelease) (ghAsset, error) {
	pattern := assetPatternFor(runtime.GOOS, runtime.GOARCH)
	if pattern == "" {
		return ghAsset{}, fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	for _, a := range r.Assets {
		if strings.Contains(a.Name, pattern) {
			return a, nil
		}
	}
	return ghAsset{}, fmt.Errorf("no release asset matches %s", pattern)
}

// versionNewer compares tag-style version strings (v0.1.2 vs v0.1.10).
// Returns true when latest > current. Missing/malformed values yield false
// so we err on the side of "no update".
func versionNewer(latest, current string) bool {
	l := parseVer(latest)
	c := parseVer(current)
	if l == nil || c == nil {
		return false
	}
	for i := 0; i < 3; i++ {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

func parseVer(s string) []int {
	s = strings.TrimPrefix(s, "v")
	parts := strings.SplitN(s, "-", 2)[0] // drop any -rc.x suffix
	segs := strings.Split(parts, ".")
	if len(segs) != 3 {
		return nil
	}
	out := make([]int, 3)
	for i, seg := range segs {
		n, err := fmt.Sscanf(seg, "%d", &out[i])
		if err != nil || n != 1 {
			return nil
		}
	}
	return out
}

// -----------------------------------------------------------------
// Wails-exposed methods on App
// -----------------------------------------------------------------

// UpdateGetState returns the latest known state (event replay for a
// freshly-mounted frontend).
func (a *App) UpdateGetState() UpdateState {
	if a.updater == nil {
		return UpdateState{Current: Version, Error: "updater not initialized"}
	}
	return a.updater.snapshot()
}

// UpdateCheck polls the GitHub Releases API for the latest tag. Cheap;
// safe to call on startup and from a manual "check now" button.
func (a *App) UpdateCheck() UpdateState {
	if a.updater == nil {
		return UpdateState{Current: Version, Error: "updater not initialized"}
	}
	u := a.updater
	u.mu.Lock()
	if u.state.Checking {
		out := u.state
		u.mu.Unlock()
		return out
	}
	u.state.Checking = true
	u.state.Error = ""
	u.mu.Unlock()
	u.emit(a.ctx)

	release, err := u.fetchLatestRelease()
	u.mu.Lock()
	u.state.Checking = false
	u.state.LastCheckAt = time.Now().Unix()
	if err != nil {
		u.state.Error = err.Error()
	} else {
		u.state.Latest = release.TagName
		u.state.Notes = release.Body
		if versionNewer(release.TagName, u.state.Current) {
			asset, aerr := pickAsset(release)
			if aerr != nil {
				u.state.Error = aerr.Error()
			} else {
				u.state.Available = true
				u.state.AssetURL = asset.BrowserDownloadURL
				u.state.AssetSize = asset.Size
			}
		} else {
			u.state.Available = false
			u.state.AssetURL = ""
			u.state.AssetSize = 0
		}
	}
	out := u.state
	u.mu.Unlock()
	u.emit(a.ctx)
	return out
}

// UpdateStartDownload begins downloading the currently advertised asset.
// A pre-existing verified download short-circuits to Ready.
func (a *App) UpdateStartDownload() UpdateState {
	if a.updater == nil {
		return UpdateState{Current: Version, Error: "updater not initialized"}
	}
	u := a.updater
	u.mu.Lock()
	if u.downloading.Load() {
		out := u.state
		u.mu.Unlock()
		return out
	}
	if !u.state.Available || u.state.AssetURL == "" {
		u.state.Error = "no update to download"
		out := u.state
		u.mu.Unlock()
		u.emit(a.ctx)
		return out
	}
	u.state.Downloading = true
	u.state.DownloadPct = 0
	u.state.Ready = false
	u.state.Error = ""
	assetURL := u.state.AssetURL
	assetSize := u.state.AssetSize
	latest := u.state.Latest
	u.mu.Unlock()
	u.downloading.Store(true)
	u.emit(a.ctx)

	ctx, cancel := context.WithCancel(context.Background())
	u.mu.Lock()
	u.cancel = cancel
	u.mu.Unlock()

	go func() {
		defer u.downloading.Store(false)
		defer func() {
			u.mu.Lock()
			u.cancel = nil
			u.mu.Unlock()
		}()

		path, err := u.download(ctx, assetURL, assetSize, a.ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				u.mu.Lock()
				u.state.Downloading = false
				u.state.DownloadPct = 0
				u.mu.Unlock()
			} else {
				u.setError(err)
			}
			u.emit(a.ctx)
			return
		}
		// Verify signature + our asset's checksum.
		if err := u.verify(ctx, path, latest); err != nil {
			u.setError(err)
			u.emit(a.ctx)
			return
		}
		u.mu.Lock()
		u.state.Downloading = false
		u.state.DownloadPct = 100
		u.state.Ready = true
		u.assetPath = path
		u.mu.Unlock()
		u.emit(a.ctx)
	}()

	return u.snapshot()
}

// UpdateInstall hands off to the platform install script with the
// downloaded asset path and the current binary/app path. On success the
// app quits so the script can replace the running binary and relaunch.
func (a *App) UpdateInstall() UpdateState {
	if a.updater == nil {
		return UpdateState{}
	}
	u := a.updater
	u.mu.Lock()
	ready := u.state.Ready
	asset := u.assetPath
	canInstall := u.state.CanInstall
	u.mu.Unlock()

	if !ready || asset == "" {
		u.setError(errors.New("no downloaded update ready"))
		u.emit(a.ctx)
		return u.snapshot()
	}
	if !canInstall {
		u.setError(errors.New("this build cannot self-install (unsigned)"))
		u.emit(a.ctx)
		return u.snapshot()
	}

	exePath, err := os.Executable()
	if err != nil {
		u.setError(err)
		u.emit(a.ctx)
		return u.snapshot()
	}
	target := exePath
	if runtime.GOOS == "darwin" {
		// On macOS the running exec is inside <App>.app/Contents/MacOS/<name>.
		// Walk up to the .app bundle so the install script replaces the whole thing.
		app := exePath
		for i := 0; i < 4 && filepath.Ext(app) != ".app"; i++ {
			app = filepath.Dir(app)
		}
		if filepath.Ext(app) == ".app" {
			target = app
		}
	}

	if err := runInstallScript(asset, target, exePath); err != nil {
		u.setError(err)
		u.emit(a.ctx)
		return u.snapshot()
	}

	// Hand off cleanly — the script has already spawned the replacement.
	go func() {
		time.Sleep(300 * time.Millisecond)
		wailsruntime.Quit(a.ctx)
	}()
	return u.snapshot()
}

// runInstallScript extracts the platform install script from the
// embedded FS into a temp file and spawns it detached with the given
// arguments. The script MUST relaunch the app itself.
func runInstallScript(asset, target, execPath string) error {
	name := installScriptName()
	if name == "" {
		return fmt.Errorf("no install script for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	data, err := installScripts.ReadFile("scripts/" + name)
	if err != nil {
		return fmt.Errorf("embed missing %s: %w", name, err)
	}
	tmpDir, err := os.MkdirTemp("", "atstarter-install-*")
	if err != nil {
		return err
	}
	scriptPath := filepath.Join(tmpDir, name)
	if err := os.WriteFile(scriptPath, data, 0o755); err != nil {
		return err
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-File", scriptPath,
			"-Asset", asset, "-Target", target, "-Exec", execPath)
	default:
		cmd = exec.Command("/bin/bash", scriptPath, asset, target, execPath)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Start()
}

func installScriptName() string {
	switch runtime.GOOS {
	case "darwin":
		return "install-darwin.sh"
	case "linux":
		return "install-linux.sh"
	case "windows":
		return "install-windows.ps1"
	}
	return ""
}

// UpdateCancel aborts an in-flight download. Idempotent.
func (a *App) UpdateCancel() UpdateState {
	if a.updater == nil {
		return UpdateState{}
	}
	u := a.updater
	u.mu.Lock()
	if u.cancel != nil {
		u.cancel()
	}
	u.state.Downloading = false
	u.state.DownloadPct = 0
	u.mu.Unlock()
	u.emit(a.ctx)
	return u.snapshot()
}

// -----------------------------------------------------------------
// Internal: fetch, download, verify
// -----------------------------------------------------------------

func (u *updater) fetchLatestRelease() (ghRelease, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", updateRepoOwner, updateRepoName)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return ghRelease{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := u.client.Do(req)
	if err != nil {
		return ghRelease{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ghRelease{}, fmt.Errorf("github api: HTTP %d", resp.StatusCode)
	}
	var r ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return ghRelease{}, err
	}
	return r, nil
}

// download streams the asset into a per-version cache dir, reporting
// progress percent to the frontend. Returns the on-disk path.
//
// It tries each mirror candidate (see mirrorURLs) in order and falls back
// to the next on any failure, with the original github.com URL as the
// final candidate — so a broken/blocked mirror never makes things worse
// than a direct download. A user cancel (context.Canceled) aborts
// immediately without trying further candidates.
func (u *updater) download(ctx context.Context, assetURL string, expectedSize int64, appCtx context.Context) (string, error) {
	if _, err := url.Parse(assetURL); err != nil {
		return "", err
	}
	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(cacheRoot, "atstarter", "updates")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	// Derive the cache filename from the ORIGINAL url so mirror prefixes
	// don't change where we store (basename is identical, but be explicit).
	base := filepath.Base(assetURL)
	if base == "" || base == "." || strings.ContainsAny(base, "/\\") {
		return "", fmt.Errorf("suspicious asset name: %q", base)
	}
	out := filepath.Join(dir, base)

	candidates := mirrorURLs(assetURL)
	var lastErr error
	for i, candURL := range candidates {
		// Reset progress for each attempt so a failed mirror doesn't leave
		// a stale percentage on screen.
		u.mu.Lock()
		u.state.DownloadPct = 0
		u.mu.Unlock()
		u.emit(appCtx)

		err := u.downloadFrom(ctx, candURL, out, dir, expectedSize, appCtx)
		if err == nil {
			return out, nil
		}
		// User cancel: stop immediately, do not fall through to next mirror.
		if errors.Is(err, context.Canceled) {
			return "", err
		}
		lastErr = fmt.Errorf("candidate %d/%d (%s): %w", i+1, len(candidates), candURL, err)
	}
	if lastErr == nil {
		lastErr = errors.New("no download candidates")
	}
	return "", lastErr
}

// downloadFrom streams a single URL into out (via a temp file in dir),
// reporting progress. Any error leaves out untouched.
func (u *updater) downloadFrom(ctx context.Context, fromURL, out, dir string, expectedSize int64, appCtx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", fromURL, nil)
	if err != nil {
		return err
	}
	// Downloads can be big; override the default 30s client timeout.
	slowClient := &http.Client{Timeout: 15 * time.Minute}
	resp, err := slowClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download: HTTP %d", resp.StatusCode)
	}
	total := expectedSize
	if total == 0 {
		total = resp.ContentLength
	}

	tmp, err := os.CreateTemp(dir, ".part-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		tmp.Close()
		if cleanup {
			os.Remove(tmpName)
		}
	}()

	pr := &progressReader{r: resp.Body, total: total, cb: func(pct int) {
		u.mu.Lock()
		u.state.DownloadPct = pct
		u.mu.Unlock()
		u.emit(appCtx)
	}}
	if _, err := io.Copy(tmp, pr); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, out); err != nil {
		return err
	}
	cleanup = false
	return nil
}

type progressReader struct {
	r     io.Reader
	read  int64
	total int64
	last  int
	cb    func(pct int)
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.read += int64(n)
	if p.total > 0 {
		pct := int(p.read * 100 / p.total)
		if pct != p.last {
			p.last = pct
			p.cb(pct)
		}
	}
	return n, err
}

// verify fetches SHA256SUMS + SHA256SUMS.sig for the release, verifies
// the signature with the embedded public key, then checks our asset's
// SHA-256 against the manifest entry. Refuses to install without a
// public key in the binary (dev / unofficial builds).
func (u *updater) verify(ctx context.Context, assetPath string, tag string) error {
	if UpdateVerifyPublicKey == "" {
		return errors.New("this build is not signed for auto-update; download and install manually")
	}
	pub, err := base64.StdEncoding.DecodeString(UpdateVerifyPublicKey)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return fmt.Errorf("bad embedded pubkey")
	}
	base := fmt.Sprintf("https://github.com/%s/%s/releases/download/%s", updateRepoOwner, updateRepoName, tag)
	sums, err := u.fetchText(ctx, base+"/SHA256SUMS")
	if err != nil {
		return fmt.Errorf("fetch SHA256SUMS: %w", err)
	}
	sigB64, err := u.fetchText(ctx, base+"/SHA256SUMS.sig")
	if err != nil {
		return fmt.Errorf("fetch SHA256SUMS.sig: %w", err)
	}
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(sigB64))
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}
	if !ed25519.Verify(pub, []byte(sums), sig) {
		return errors.New("SHA256SUMS signature verification failed")
	}

	// Compute our asset's hash and look it up in the manifest.
	f, err := os.Open(assetPath)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	assetName := filepath.Base(assetPath)
	for _, line := range strings.Split(sums, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		if fields[1] == assetName {
			if fields[0] == got {
				return nil
			}
			return fmt.Errorf("checksum mismatch for %s", assetName)
		}
	}
	return fmt.Errorf("asset %s not listed in SHA256SUMS", assetName)
}

func (u *updater) fetchText(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	resp, err := u.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
