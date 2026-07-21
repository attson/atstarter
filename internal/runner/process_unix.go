//go:build !windows

package runner

import (
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// setupProcAttr 让子进程自成进程组,便于整组信号。
func setupProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killTree 先给整个进程组发 SIGTERM,超时后 SIGKILL。
// 负 PID 表示"发给该进程组"。
func killTree(pid int) {
	pgid := pid // 因 Setpgid,子进程 pid == pgid
	_ = syscall.Kill(-pgid, syscall.SIGTERM)
	go func() {
		time.Sleep(5 * time.Second)
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
	}()
}

// shellQuote 用单引号包裹一个 token,内部单引号转义为 '\”。
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// shellJoin 把 command 与各 arg 拼成可安全交给 shell 的单行命令。
func shellJoin(command string, args []string) string {
	parts := make([]string, 0, 1+len(args))
	parts = append(parts, shellQuote(command))
	for _, a := range args {
		parts = append(parts, shellQuote(a))
	}
	return strings.Join(parts, " ")
}

// userShell 返回用户登录 shell($SHELL),为空则回退 /bin/sh。
func userShell() string {
	if sh := os.Getenv("SHELL"); sh != "" {
		return sh
	}
	return "/bin/sh"
}

// buildCmd 用登录交互式 shell 包裹命令,让子进程拿到用户 shell 的完整 PATH
// (pnpm / nvm / go 等)。-l 加载 login rc,-i 加载交互 rc(PATH 通常在这),
// -c 执行拼好的命令行。
func buildCmd(spec Spec) *exec.Cmd {
	line := shellJoin(spec.Command, spec.Args)
	return exec.Command(userShell(), "-l", "-i", "-c", line)
}
