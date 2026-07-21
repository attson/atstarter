//go:build !windows

package runner

import (
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

// setupProcAttr 让子进程成为新会话(setsid)的首进程,自成进程组。
// Stop 时对该进程组发信号可覆盖绝大多数子孙;极少数自行 setpgid 另开
// 进程组的孙进程不在同组内,是已知局限(常见场景不触发)。
func setupProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

// killTree 先给整个进程组发 SIGTERM,超时后 SIGKILL。
// 负 PID 表示"发给该进程组"。
func killTree(pid int) {
	pgid := pid // 因 Setsid,shell 成为会话首进程,pid == sid == pgid
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

// expandTilde 把开头的 ~ 或 ~/... 展开为家目录绝对路径,其它形式(绝对/相对
// 路径、~user、token 中间的 ~)原样返回。必须在 shellQuote 之前调用:命令与
// 参数最终被单引号包裹交给 shell,而单引号内 shell 不会展开 ~,不预先展开则
// 用户填的 ~/sdk/go 之类路径会以字面量查找而 code 127 失败。
func expandTilde(s string) string {
	if s != "~" && !strings.HasPrefix(s, "~/") {
		return s
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return s
	}
	if s == "~" {
		return home
	}
	return home + s[1:]
}

// shellJoin 把 command 与各 arg 拼成可安全交给 shell 的单行命令。
// 每个 token 先展开开头的 ~ 再单引号包裹(见 expandTilde)。
func shellJoin(command string, args []string) string {
	parts := make([]string, 0, 1+len(args))
	parts = append(parts, shellQuote(expandTilde(command)))
	for _, a := range args {
		parts = append(parts, shellQuote(expandTilde(a)))
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

// shellNoiseMarkers 是交互式 shell 在无控制 TTY 时(CI、部分 GUI 启动场景)
// 向 stderr 打印的 job-control 诊断噪声特征子串。用子串而非整行匹配,因为 bash
// 的 "cannot set terminal process group (<pid>)" 含变化的 pid。这几条是 shell
// 诊断专用语,业务命令原样打印的概率可忽略。
var shellNoiseMarkers = []string{
	"can't access tty",                  // dash: "...: 0: can't access tty; job control turned off"
	"no job control",                    // bash: "bash: no job control in this shell"
	"cannot set terminal process group", // bash: "...Inappropriate ioctl for device"
}

// isShellNoise 判断一行 stderr 是否为交互式 shell 无 TTY 启动噪声,应从日志过滤。
func isShellNoise(line string) bool {
	for _, m := range shellNoiseMarkers {
		if strings.Contains(line, m) {
			return true
		}
	}
	return false
}
