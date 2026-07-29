//go:build darwin

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mkAppBundle 造一个假 .app 目录树,内容里塞点标记文件方便断言"当前 bundle 是
// 老 / 新哪个版本"。真 bundle 有 Contents/MacOS 等结构,测试里只在乎目录能被
// rename、能被 stat 出来。
func mkAppBundle(t *testing.T, path, marker string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "marker"), []byte(marker), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readMarker(t *testing.T, appDir string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(appDir, "marker"))
	if err != nil {
		t.Fatalf("read marker in %s: %v", appDir, err)
	}
	return string(b)
}

// TestRenameSwapReplacesInPlace 覆盖正常路径:老 bundle → .old,新 bundle → 目标,
// 事后目标位置是新版内容。这是升级流程的关键正确性属性 —— 保证目标路径始终有
// 一个有效 bundle(Dock/Alfred 图标指向不失效),中间也不留下 .new/.old 残骸。
func TestRenameSwapReplacesInPlace(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "AT Starter.app")
	staging := dest + ".new"
	old := dest + ".old"

	mkAppBundle(t, dest, "v1")
	mkAppBundle(t, staging, "v2")

	if err := renameSwap(dest, old, staging); err != nil {
		t.Fatalf("renameSwap: %v", err)
	}
	if got := readMarker(t, dest); got != "v2" {
		t.Errorf("dest marker = %q, want v2 (new bundle should be in place)", got)
	}
	// oldApp 在 renameSwap 后应仍存在(调用方负责后续 cleanup);staging 已被 rename 走
	// 到 destApp,不能再存在。
	if _, err := os.Stat(staging); !os.IsNotExist(err) {
		t.Errorf("staging still exists after swap: err = %v", err)
	}
	if _, err := os.Stat(old); err != nil {
		t.Errorf("old bundle should be preserved for caller cleanup: %v", err)
	}
}

// TestRenameSwapFirstInstallNoDest 覆盖"目标路径本来就没有 bundle"的路径
// (首次装到 ~/Applications 的场景):不应报错、staging 应就位。
func TestRenameSwapFirstInstallNoDest(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "AT Starter.app")
	staging := dest + ".new"
	old := dest + ".old"

	mkAppBundle(t, staging, "v2")

	if err := renameSwap(dest, old, staging); err != nil {
		t.Fatalf("renameSwap: %v", err)
	}
	if got := readMarker(t, dest); got != "v2" {
		t.Errorf("dest marker = %q, want v2", got)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Errorf(".old should not exist when there was no old bundle: err = %v", err)
	}
}

// TestRenameSwapRollsBackWhenNewRenameFails 覆盖"第二步 rename 失败"的分支:
// 老 bundle 必须复位到目标路径,而不是留下"目标路径消失"的窗口 —— 否则用户
// Dock 图标会变问号,下次点开报"app 不存在"。
// 触发失败的方式:把 staging 提前占成一个 file 而不是 dir,再让 dest 是一个
// 已经不存在的路径? 不,更靠谱的做法是把 dest 的父目录设成只读,os.Rename
// 就会失败;但改父目录 mode 在 CI 里不稳。改用"删除 staging"制造 rename 找
// 不到源:renameSwap 内 os.Rename(stagingApp, destApp) 直接失败。
func TestRenameSwapRollsBackWhenNewRenameFails(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "AT Starter.app")
	staging := dest + ".new"
	old := dest + ".old"

	mkAppBundle(t, dest, "v1")
	// staging 不创建 —— 第二步 rename 立刻失败。

	err := renameSwap(dest, old, staging)
	if err == nil {
		t.Fatal("renameSwap should have failed with no staging present")
	}
	// 关键断言:老 bundle 必须复位到 dest,而不是卡在 .old。
	if got := readMarker(t, dest); got != "v1" {
		t.Errorf("dest marker = %q, want v1 (old bundle should be rolled back)", got)
	}
	if _, err := os.Stat(old); !os.IsNotExist(err) {
		t.Errorf(".old should have been rolled back to dest, but still exists: %v", err)
	}
}

// TestChooseDestParentPrefersOriginalLocation 原位置可写就用原位置 —— 升级"就
// 地覆盖",Dock/Alfred/spotlight 索引都不需要重建。
func TestChooseDestParentPrefersOriginalLocation(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "AT Starter.app")
	got, err := chooseDestParent(target)
	if err != nil {
		t.Fatal(err)
	}
	if got != dir {
		t.Errorf("chooseDestParent = %q, want original parent %q", got, dir)
	}
}

// TestChooseDestParentFallsBackWhenOriginalReadOnly 原位置不可写时退到
// ~/Applications。为了不真去动 /Applications 或家目录,把原位置伪造成一个只
// 读目录来触发 fallback。fallback 一定不能等于原目录。
func TestChooseDestParentFallsBackWhenOriginalReadOnly(t *testing.T) {
	dir := t.TempDir()
	readonly := filepath.Join(dir, "ro")
	if err := os.MkdirAll(readonly, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(readonly, 0o755) }) // 让 TempDir 能清

	target := filepath.Join(readonly, "AT Starter.app")
	got, err := chooseDestParent(target)
	if err != nil {
		t.Fatal(err)
	}
	if got == readonly {
		t.Errorf("chooseDestParent should have fallen back away from read-only %q", readonly)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("chooseDestParent should return absolute path, got %q", got)
	}
}

// TestFirstAppInFindsDirect 覆盖最常见 DMG 结构:mount 根目录下直接一个 *.app。
func TestFirstAppInFindsDirect(t *testing.T) {
	dir := t.TempDir()
	appPath := filepath.Join(dir, "AT Starter.app")
	mkAppBundle(t, appPath, "v1")
	// 制造几个干扰项:Applications 符号链接、一个 .txt、一个非 .app 目录
	if err := os.Symlink("/Applications", filepath.Join(dir, "Applications")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.txt"), []byte("read me"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := firstAppIn(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != appPath {
		t.Errorf("firstAppIn = %q, want %q", got, appPath)
	}
}

// TestFirstAppInNoAppErrors DMG 里没 .app 时应报错,不能返回空字符串让后续
// ditto 拿空 src 干出意外结果。
func TestFirstAppInNoAppErrors(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("nothing here"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := firstAppIn(dir); err == nil {
		t.Error("firstAppIn should fail when no .app is present")
	}
}

func TestDarwinRelaunchAfterExitScriptWaitsThenOpensNewInstance(t *testing.T) {
	script := darwinRelaunchAfterExitScript(12345, "/Applications/AT Starter.app")

	if !strings.Contains(script, "kill -0 12345") {
		t.Fatalf("relaunch script must wait for the old app process to exit, got:\n%s", script)
	}
	if !strings.Contains(script, "/usr/bin/open -n '/Applications/AT Starter.app'") {
		t.Fatalf("relaunch script must open a new instance after exit, got:\n%s", script)
	}
}

func TestDarwinShellQuoteHandlesSingleQuotes(t *testing.T) {
	got := darwinShellQuote("/tmp/Bob's Apps/AT Starter.app")
	want := `'/tmp/Bob'\''s Apps/AT Starter.app'`
	if got != want {
		t.Errorf("darwinShellQuote = %q, want %q", got, want)
	}
}
