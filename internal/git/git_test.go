package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// newRepo 造一个带一次提交的临时仓库。git 不可用时跳过整组测试。
func newRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	mustGit(t, dir, "init", "-b", "main")
	mustGit(t, dir, "config", "user.email", "test@example.com")
	mustGit(t, dir, "config", "user.name", "Test")
	// 全局 commit hook / gpg 签名会干扰测试。
	mustGit(t, dir, "config", "commit.gpgsign", "false")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mustGit(t, dir, "add", ".")
	mustGit(t, dir, "commit", "-m", "init")
	return dir
}

func mustGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

func TestIsRepoAndCurrentBranch(t *testing.T) {
	dir := newRepo(t)
	if !IsRepo(dir) {
		t.Fatal("want IsRepo true")
	}
	if got := CurrentBranch(dir); got != "main" {
		t.Errorf("want main, got %q", got)
	}

	plain := t.TempDir()
	if IsRepo(plain) {
		t.Error("want IsRepo false for a plain directory")
	}
	if got := CurrentBranch(plain); got != "" {
		t.Errorf("want empty branch for non-repo, got %q", got)
	}
}

func TestGetStatusCleanRepo(t *testing.T) {
	dir := newRepo(t)
	status, err := GetStatus(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Repo || status.Branch != "main" || status.Detached || status.Dirty {
		t.Fatalf("unexpected status: %+v", status)
	}
	if status.Head == "" {
		t.Error("want a short HEAD sha")
	}
}

func TestGetStatusNonRepoIsNotAnError(t *testing.T) {
	status, err := GetStatus(t.TempDir())
	if err != nil {
		t.Fatalf("非 git 目录不该报错: %v", err)
	}
	if status.Repo {
		t.Error("want Repo false")
	}
}

func TestGetStatusDirtyCountsUntracked(t *testing.T) {
	dir := newRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "scratch.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	status, err := GetStatus(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Dirty {
		t.Error("未跟踪文件也算脏:切分支时同样可能被覆盖")
	}
}

func TestGetStatusDetached(t *testing.T) {
	dir := newRepo(t)
	head := mustGit(t, dir, "rev-parse", "HEAD")
	mustGit(t, dir, "checkout", "--detach", head)
	status, err := GetStatus(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Detached || status.Branch != "" {
		t.Fatalf("want detached with empty branch, got %+v", status)
	}
}

func TestListBranchesLocalOnly(t *testing.T) {
	dir := newRepo(t)
	mustGit(t, dir, "branch", "feature/x")
	b, err := ListBranches(dir)
	if err != nil {
		t.Fatal(err)
	}
	if b.Branch != "main" {
		t.Errorf("want current main, got %q", b.Branch)
	}
	if len(b.Local) != 2 || b.Local[0] != "feature/x" || b.Local[1] != "main" {
		t.Fatalf("want [feature/x main], got %v", b.Local)
	}
	if len(b.Remote) != 0 {
		t.Errorf("want no remote-only branches, got %v", b.Remote)
	}
}

func TestListBranchesHidesRemoteWithLocalCounterpart(t *testing.T) {
	origin := newRepo(t)
	mustGit(t, origin, "branch", "shared")
	mustGit(t, origin, "branch", "remote-only")

	clone := t.TempDir()
	cloneInto(t, origin, clone)

	b, err := ListBranches(clone)
	if err != nil {
		t.Fatal(err)
	}
	// 克隆后只有 main 是本地分支,另外两个只在远端。
	if !contains(b.Remote, "remote-only") || !contains(b.Remote, "shared") {
		t.Fatalf("want remote-only branches listed, got %v", b.Remote)
	}
	if contains(b.Remote, "main") {
		t.Errorf("main 已有本地分支,不该重复出现在远端列表: %v", b.Remote)
	}
	if contains(b.Remote, "HEAD") {
		t.Errorf("origin/HEAD 不是真分支: %v", b.Remote)
	}
}

func cloneInto(t *testing.T, src, dst string) {
	t.Helper()
	cmd := exec.Command("git", "clone", src, dst)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git clone: %v\n%s", err, out)
	}
	mustGit(t, dst, "config", "user.email", "test@example.com")
	mustGit(t, dst, "config", "user.name", "Test")
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

func TestCheckoutExistingBranch(t *testing.T) {
	dir := newRepo(t)
	mustGit(t, dir, "branch", "feature/x")
	b, err := Checkout(dir, "feature/x")
	if err != nil {
		t.Fatal(err)
	}
	if b.Branch != "feature/x" {
		t.Fatalf("want feature/x, got %q", b.Branch)
	}
	if CurrentBranch(dir) != "feature/x" {
		t.Error("工作区没有真的切过去")
	}
}

func TestCheckoutRemoteOnlyBranchCreatesTracking(t *testing.T) {
	origin := newRepo(t)
	mustGit(t, origin, "branch", "remote-only")
	clone := t.TempDir()
	cloneInto(t, origin, clone)

	b, err := Checkout(clone, "remote-only")
	if err != nil {
		t.Fatal(err)
	}
	if b.Branch != "remote-only" {
		t.Fatalf("want remote-only, got %q", b.Branch)
	}
	if !contains(b.Local, "remote-only") {
		t.Errorf("切过去之后应该有本地分支了: %v", b.Local)
	}
}

func TestCheckoutNewBranch(t *testing.T) {
	dir := newRepo(t)
	b, err := CheckoutNew(dir, "feature/new", "")
	if err != nil {
		t.Fatal(err)
	}
	if b.Branch != "feature/new" {
		t.Fatalf("want feature/new, got %q", b.Branch)
	}
}

func TestCheckoutNewFromStartPoint(t *testing.T) {
	dir := newRepo(t)
	mustGit(t, dir, "branch", "base")
	b, err := CheckoutNew(dir, "derived", "base")
	if err != nil {
		t.Fatal(err)
	}
	if b.Branch != "derived" {
		t.Fatalf("want derived, got %q", b.Branch)
	}
}

func TestCheckoutSurfacesGitError(t *testing.T) {
	dir := newRepo(t)
	_, err := Checkout(dir, "does-not-exist")
	if err == nil {
		t.Fatal("want an error for a missing branch")
	}
	// git 自己的话比我们编的准。
	if !strings.Contains(strings.ToLower(err.Error()), "does-not-exist") {
		t.Errorf("want git's own message, got %q", err)
	}
}

func TestCheckoutRejectsHostileNames(t *testing.T) {
	dir := newRepo(t)
	for _, name := range []string{"", "-f", "has space", "a..b", "/lead", "trail/", "a//b", ".hidden", "x.lock", "a@{0}", "sta*r"} {
		if _, err := Checkout(dir, name); err == nil {
			t.Errorf("want %q rejected", name)
		}
		if _, err := CheckoutNew(dir, name, ""); err == nil {
			t.Errorf("want %q rejected for new branch", name)
		}
	}
}

func TestCheckoutOnNonRepo(t *testing.T) {
	if _, err := Checkout(t.TempDir(), "main"); err != ErrNotRepo {
		t.Fatalf("want ErrNotRepo, got %v", err)
	}
	if _, err := CheckoutNew(t.TempDir(), "main", ""); err != ErrNotRepo {
		t.Fatalf("want ErrNotRepo, got %v", err)
	}
}

func TestStripRemote(t *testing.T) {
	if got := stripRemote("origin/feature/x"); got != "feature/x" {
		t.Errorf("want feature/x, got %q", got)
	}
	if got := stripRemote("noslash"); got != "" {
		t.Errorf("want empty, got %q", got)
	}
}
