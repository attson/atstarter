//go:build !windows

package runner

import (
	"fmt"
	"os"
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
