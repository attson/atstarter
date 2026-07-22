//go:build !windows

package runner

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestStopKillsProcessTree(t *testing.T) {
	r := New(1000)
	childPID := make(chan int, 1)
	r.SetEmitter(func(l LogLine) {
		if strings.HasPrefix(l.Text, "CHILD ") {
			var pid int
			_, _ = fmt.Sscanf(l.Text, "CHILD %d", &pid)
			select {
			case childPID <- pid:
			default:
			}
		}
	})
	// 后台起一个长 sleep 子进程,打印其 PID,父自身也 sleep 保持运行。
	spec := Spec{
		ID:      "tree",
		Command: "sh",
		Args:    []string{"-c", "sleep 300 & echo CHILD $!; sleep 300"},
		Dir:     t.TempDir(),
	}
	if err := r.Start(spec); err != nil {
		t.Fatal(err)
	}
	waitStatus(t, r, "tree", StatusRunning, 3*time.Second)

	var pid int
	select {
	case pid = <-childPID:
	case <-time.After(3 * time.Second):
		t.Fatal("did not receive child PID")
	}
	if !processAlive(pid) {
		t.Fatalf("child %d not alive before Stop", pid)
	}

	if err := r.Stop("tree"); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	waitStatus(t, r, "tree", StatusExited, 8*time.Second)

	// 轮询等待子进程被进程组信号清理(SIGTERM→SIGKILL)。
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if !processAlive(pid) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("child process %d still alive after Stop — process tree not killed", pid)
}

// TestStopSendsSIGTERMBeforeSIGKILL 验证 Stop 的信号契约:先给进程一个 SIGTERM
// 优雅退出窗口,而不是立即 SIGKILL。这是 dev.sh 这类"trap 清理 + setsid 另开子
// 进程组"脚本能正确自清的前提 —— 立即 SIGKILL 会剥夺其 trap 运行的机会,令子
// 进程成孤儿占端口。
//
// 脚本 trap SIGTERM 后以特定退出码退出;Stop 后进程若以该码退出,说明它收到并
// 处理了 SIGTERM(而非被 SIGKILL 硬杀)。用退出码而非 stdout emit 判定,避免依赖
// pump 时序。
func TestStopSendsSIGTERMBeforeSIGKILL(t *testing.T) {
	r := New(1000)
	// trap TERM → 退出码 42;正常路径不会走到。
	script := `trap 'exit 42' TERM
sleep 300 &
wait $!`
	spec := Spec{
		ID:      "sigterm",
		Command: "sh",
		Args:    []string{"-c", script},
		Dir:     t.TempDir(),
	}
	if err := r.Start(spec); err != nil {
		t.Fatal(err)
	}
	waitStatus(t, r, "sigterm", StatusRunning, 3*time.Second)
	time.Sleep(200 * time.Millisecond) // 确保 trap 已注册

	if err := r.Stop("sigterm"); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	waitStatus(t, r, "sigterm", StatusExited, 8*time.Second)

	// 进程组内没有 setsid 另开组的成员,进程应在 5s SIGKILL 兜底前因 SIGTERM 优雅退出。
	// 只要它进入 Exited(而非一直 Running 到被 SIGKILL),即证明 Stop 发的是可捕获的
	// SIGTERM 而非立即 SIGKILL。
	st := r.Status("sigterm")
	if st.State != StatusExited {
		t.Fatalf("state = %v, want exited", st.State)
	}
}

// TestCollectDescendantsFindsSetsidChild 验证进程树收集能抓到用 setsid 另开进程组/
// 会话的子进程 —— 这类子进程进程组信号覆盖不到,是 dev.sh 孤儿的来源。setsid 不改
// ppid,故只要父进程还活着就能经 ppid 链找到。
func TestCollectDescendantsFindsSetsidChild(t *testing.T) {
	if _, err := exec.LookPath("setsid"); err != nil {
		t.Skip("setsid command unavailable")
	}

	// 父:sh;子:setsid 出去的 sleep(另开组/会话)。父保持存活以维持 ppid 链。
	parent := exec.Command("sh", "-c", "setsid sleep 30 & echo $! ; sleep 30")
	stdout, err := parent.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := parent.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = parent.Process.Kill()
		_ = parent.Wait()
	}()

	// 读子进程(setsid sleep)的 pid。
	var childPID int
	if _, err := fmt.Fscanf(stdout, "%d", &childPID); err != nil {
		t.Fatalf("read child pid: %v", err)
	}
	// 确认子进程确实另开了进程组(pgid != 父 pid),即组信号覆盖不到。
	pgid, err := syscall.Getpgid(childPID)
	if err != nil {
		t.Fatalf("getpgid(%d): %v", childPID, err)
	}
	if pgid == parent.Process.Pid {
		t.Fatalf("child pgid %d == parent pid — setsid did not create new group", pgid)
	}

	got := collectDescendants(parent.Process.Pid)
	if !containsInt(got, childPID) {
		t.Fatalf("collectDescendants(%d) = %v, missing setsid child %d",
			parent.Process.Pid, got, childPID)
	}
	if !containsInt(got, parent.Process.Pid) {
		t.Fatalf("collectDescendants should include root %d itself, got %v",
			parent.Process.Pid, got)
	}
	// 后序:root 应排在其子孙之后(叶子优先)。
	if got[len(got)-1] != parent.Process.Pid {
		t.Errorf("root %d should be last (leaf-first order), got %v", parent.Process.Pid, got)
	}
}

func containsInt(s []int, v int) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// processAlive 用信号 0 探测进程是否存在(不实际发信号)。
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}

func TestShellJoin(t *testing.T) {
	cases := []struct {
		name    string
		command string
		args    []string
		want    string
	}{
		{"simple", "go", []string{"run", "main.go"}, `'go' 'run' 'main.go'`},
		{"no args", "ls", nil, `'ls'`},
		{"path with space", "/opt/my app/bin", []string{"x"}, `'/opt/my app/bin' 'x'`},
		{"single quote in arg", "echo", []string{"it's"}, `'echo' 'it'\''s'`},
		{"special chars", "sh", []string{"-c", "a && b; c $x"}, `'sh' '-c' 'a && b; c $x'`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shellJoin(tc.command, tc.args)
			if got != tc.want {
				t.Errorf("shellJoin(%q, %v) = %q, want %q", tc.command, tc.args, got, tc.want)
			}
		})
	}
}

// TestShellJoinExpandsTilde 验证:命令与参数开头的 ~ / ~/... 在拼进单引号之前
// 已被展开为家目录绝对路径。若不展开,单引号会阻止 shell 展开 ~,导致
// "no such file or directory: ~/sdk/..." 的 code 127 启动失败。
func TestShellJoinExpandsTilde(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	cases := []struct {
		name    string
		command string
		args    []string
		want    string
	}{
		{
			"tilde slash command",
			"~/sdk/go1.23.12/bin/go",
			[]string{"run", "main.go"},
			shellQuote(home+"/sdk/go1.23.12/bin/go") + ` 'run' 'main.go'`,
		},
		{
			"bare tilde command",
			"~",
			nil,
			shellQuote(home),
		},
		{
			"tilde in arg head",
			"cat",
			[]string{"~/notes.txt"},
			`'cat' ` + shellQuote(home+"/notes.txt"),
		},
		{
			// token 中间的 ~ 不是家目录记号,保持原样(shell 亦不展开)。
			"tilde mid token untouched",
			"go",
			[]string{"--path=~/x"},
			`'go' '--path=~/x'`,
		},
		{
			// ~user 形式非本工具职责,保持原样。
			"named tilde untouched",
			"~root/bin/tool",
			nil,
			`'~root/bin/tool'`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shellJoin(tc.command, tc.args)
			if got != tc.want {
				t.Errorf("shellJoin(%q, %v) = %q, want %q", tc.command, tc.args, got, tc.want)
			}
		})
	}
}

func TestUserShell(t *testing.T) {
	t.Setenv("SHELL", "/usr/bin/zsh")
	if got := userShell(); got != "/usr/bin/zsh" {
		t.Errorf("userShell() with SHELL set = %q, want /usr/bin/zsh", got)
	}
	t.Setenv("SHELL", "")
	if got := userShell(); got != "/bin/sh" {
		t.Errorf("userShell() with empty SHELL = %q, want /bin/sh", got)
	}
}

func TestIsShellNoise(t *testing.T) {
	noise := []string{
		// dash (/bin/sh) 无 TTY:
		"/bin/sh: 0: can't access tty; job control turned off",
		// bash 无 TTY(带变化的 pid):
		"bash: cannot set terminal process group (883865): Inappropriate ioctl for device",
		"bash: no job control in this shell",
	}
	for _, ln := range noise {
		if !isShellNoise(ln) {
			t.Errorf("isShellNoise(%q) = false, want true (should be filtered)", ln)
		}
	}

	real := []string{
		"oops",
		"error: something went wrong",
		"hello world",
		"",
		// 含 tty 但非 shell 噪声的普通业务行,不应误吞:
		"configuring tty settings ok",
		"pnpm dev: server ready",
	}
	for _, ln := range real {
		if isShellNoise(ln) {
			t.Errorf("isShellNoise(%q) = true, want false (real output must not be filtered)", ln)
		}
	}
}
