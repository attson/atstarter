// Package git 提供项目工作目录的只读 git 状态与分支切换。
// 全部通过 exec 调用系统 git,不引入 git 库:atstarter 只需要「看一眼当前分支」
// 和「切过去」这两件事,自带实现的成本远高于收益。
//
// 约定:
//   - 任何命令都带超时,git 卡住(网络钩子、锁文件)不能拖死界面。
//   - 失败时把 git 自己的 stderr 原样带回前端 —— "Your local changes would be
//     overwritten" 这类消息比我们能写的任何提示都准确。
//   - 不做 fetch/pull/push。切换分支是本地操作,联网操作是另一个话题。
package git

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// readTimeout 给查询类命令。这些都是本地读操作,慢就是有问题。
const readTimeout = 3 * time.Second

// checkoutTimeout 给切换分支。大仓库更新工作区可能要几秒到十几秒。
const checkoutTimeout = 60 * time.Second

// Status 是工作目录的 git 概况。非仓库时 Repo 为 false,其余字段无意义。
type Status struct {
	Repo     bool   `json:"repo"`
	Branch   string `json:"branch"`   // detached HEAD 时为空
	Detached bool   `json:"detached"`
	Dirty    bool   `json:"dirty"` // 有未提交改动(含未跟踪文件)
	Head     string `json:"head"`  // 短 SHA,detached 时给用户一个抓手
}

// Branches 是分支选择器需要的全部数据。
type Branches struct {
	Status
	Local []string `json:"local"`
	// Remote 是「只在远端存在」的分支短名(已去掉 origin/ 前缀,也去掉了
	// 已有同名本地分支的)。切过去时 git 的 DWIM 会自动建立跟踪分支。
	Remote []string `json:"remote"`
}

// ErrNotRepo 表示目录不是 git 工作区。
var ErrNotRepo = errors.New("not a git repository")

// IsRepo 快速判断:没有 .git 就一定不是仓库,免掉 exec 开销。
// 注意 worktree 的 .git 是文件不是目录,所以只看存在性。
func IsRepo(dir string) bool {
	if dir == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil
}

// run 执行 git 子命令,返回 trim 过的 stdout。失败时错误里带 stderr。
func run(dir string, timeout time.Duration, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if ctx.Err() == context.DeadlineExceeded {
			return "", errors.New("git 命令超时: git " + strings.Join(args, " "))
		}
		if msg == "" {
			return "", err
		}
		return "", errors.New(msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// CurrentBranch 返回当前分支名。非仓库、detached HEAD、git 不可用都返回空串
// (调用方据此隐藏分支标签)。
func CurrentBranch(dir string) string {
	if !IsRepo(dir) {
		return ""
	}
	// symbolic-ref --short HEAD 在 detached 时非零退出,正好落到空串。
	out, err := run(dir, readTimeout, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		return ""
	}
	return out
}

// GetStatus 返回工作目录概况。非仓库返回 Repo=false 而不是错误 ——
// 「这个项目不是 git 仓库」是常态,不该让界面弹错误。
func GetStatus(dir string) (Status, error) {
	if !IsRepo(dir) {
		return Status{}, nil
	}
	s := Status{Repo: true}
	if branch, err := run(dir, readTimeout, "symbolic-ref", "--short", "HEAD"); err == nil {
		s.Branch = branch
	} else {
		s.Detached = true
	}
	if head, err := run(dir, readTimeout, "rev-parse", "--short", "HEAD"); err == nil {
		s.Head = head
	}
	// --porcelain 有输出就是有改动;这里也把未跟踪文件算进去,因为切分支时
	// 未跟踪文件同样可能被覆盖。
	if out, err := run(dir, readTimeout, "status", "--porcelain"); err == nil {
		s.Dirty = out != ""
	}
	return s, nil
}

// ListBranches 列出本地分支与仅存在于远端的分支。
func ListBranches(dir string) (Branches, error) {
	status, err := GetStatus(dir)
	if err != nil {
		return Branches{}, err
	}
	if !status.Repo {
		return Branches{}, nil
	}
	b := Branches{Status: status}

	locals, err := run(dir, readTimeout, "for-each-ref", "--format=%(refname:short)", "refs/heads")
	if err != nil {
		return Branches{}, err
	}
	b.Local = splitLines(locals)

	remotes, err := run(dir, readTimeout, "for-each-ref", "--format=%(refname:short)", "refs/remotes")
	if err != nil {
		// 没有远端不是错误。
		return b, nil
	}
	localSet := make(map[string]bool, len(b.Local))
	for _, name := range b.Local {
		localSet[name] = true
	}
	seen := make(map[string]bool)
	for _, ref := range splitLines(remotes) {
		short := stripRemote(ref)
		// origin/HEAD 是符号引用,不是真分支。
		if short == "" || short == "HEAD" || localSet[short] || seen[short] {
			continue
		}
		seen[short] = true
		b.Remote = append(b.Remote, short)
	}
	return b, nil
}

func splitLines(out string) []string {
	if out == "" {
		return nil
	}
	lines := strings.Split(out, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

// stripRemote 把 origin/feature/x 削成 feature/x。只削第一段。
func stripRemote(ref string) string {
	i := strings.Index(ref, "/")
	if i < 0 {
		return ""
	}
	return ref[i+1:]
}

// validateName 挡住会被 git 拒绝或被当成命令行选项的分支名。
// 这不是 check-ref-format 的完整复刻,只覆盖用户手输时真会撞上的那些。
func validateName(name string) error {
	if name == "" {
		return errors.New("分支名不能为空")
	}
	if strings.HasPrefix(name, "-") {
		return errors.New("分支名不能以 - 开头")
	}
	if strings.ContainsAny(name, " \t~^:?*[\\") {
		return errors.New("分支名不能包含空格或 ~ ^ : ? * [ \\")
	}
	if strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") || strings.Contains(name, "//") {
		return errors.New("分支名不能以 / 开头或结尾,也不能包含 //")
	}
	if strings.HasPrefix(name, ".") || strings.Contains(name, "..") || strings.HasSuffix(name, ".lock") {
		return errors.New("分支名不能以 . 开头、包含 .. 或以 .lock 结尾")
	}
	if strings.Contains(name, "@{") {
		return errors.New("分支名不能包含 @{")
	}
	return nil
}

// Checkout 切换到已有分支 name。name 只在远端存在时,git 的 DWIM 会自动建立
// 同名本地跟踪分支。工作区有冲突改动时 git 会拒绝,错误原样返回给前端。
func Checkout(dir, name string) (Branches, error) {
	if !IsRepo(dir) {
		return Branches{}, ErrNotRepo
	}
	if err := validateName(name); err != nil {
		return Branches{}, err
	}
	if _, err := run(dir, checkoutTimeout, "checkout", name); err != nil {
		return Branches{}, err
	}
	return ListBranches(dir)
}

// CheckoutNew 以 startPoint 为起点新建分支并切过去。startPoint 为空时用当前 HEAD。
func CheckoutNew(dir, name, startPoint string) (Branches, error) {
	if !IsRepo(dir) {
		return Branches{}, ErrNotRepo
	}
	if err := validateName(name); err != nil {
		return Branches{}, err
	}
	args := []string{"checkout", "-b", name}
	if startPoint != "" {
		if err := validateName(startPoint); err != nil {
			return Branches{}, err
		}
		args = append(args, startPoint)
	}
	if _, err := run(dir, checkoutTimeout, args...); err != nil {
		return Branches{}, err
	}
	return ListBranches(dir)
}
