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

// killTree 终止顶层进程及其所有子孙,分两阶段:子孙先 SIGTERM 给优雅退出窗口,
// 5s 后统一 SIGKILL 兜底。
//
// 为什么不能只靠进程组信号(syscall.Kill(-pgid, ...)):像 dev.sh 这类启动脚本会
// 用 setsid 把前后端子进程另开进程组/会话,组信号覆盖不到它们,只发组信号会留下
// 占端口的孤儿。改为遍历 ppid 进程树逐个发信号 —— setsid 不改 ppid,趁顶层进程
// 尚在(未被 reparent),经 ppid 链能抓到这些另开组的子孙。
//
// 为什么子孙用 SIGTERM 而非立即 SIGKILL:SIGKILL 不可捕获,会剥夺 dev.sh 的 trap
// 清理机会;先 SIGTERM 让脚本与其子进程有机会优雅退出(与 Ctrl-C 等价)。
//
// 为什么顶层直接 SIGKILL:顶层是 buildCmd 的 `zsh -l -i`(交互式)包装,交互式 shell
// 默认忽略 SIGTERM(杀不死),且它只是包装、没有需要保护的 trap。对它直接 SIGKILL
// 才能让 cmd.Wait 返回、状态转为 Exited;其内部脚本(如 dev.sh)是子孙,已先收到
// SIGTERM 有优雅窗口。
func killTree(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	top := cmd.Process.Pid // 因 Setsid,顶层 shell pid == sid == pgid

	// 阶段一:收集整棵树。子孙 SIGTERM,顶层 SIGKILL(它忽略 SIGTERM)。
	tree := collectDescendants(top) // 叶子优先,top 在最后
	for _, pid := range tree {
		if pid == top {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		} else {
			_ = syscall.Kill(pid, syscall.SIGTERM)
		}
	}

	go func() {
		time.Sleep(5 * time.Second)
		// 阶段二:重新收集(可能有新派生进程),全部 SIGKILL;再补一发组 SIGKILL 兜底。
		for _, pid := range collectDescendants(top) {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
		_ = syscall.Kill(-top, syscall.SIGKILL)
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
