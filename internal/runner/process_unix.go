//go:build !windows

package runner

import (
	"os"
	"os/exec"
	"sort"
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

func shellLine(spec Spec) string {
	line := shellJoin(spec.Command, spec.Args)
	exports := shellExports(spec.Env)
	if exports == "" {
		return line
	}
	return exports + "; " + line
}

// shellExports 把 env 覆盖拼成在 login shell rc 执行“之后”生效的 export 前缀。
// 传入的是原始未展开的值,对两类变量引用有意保留、交给 shell 在 rc 之后展开:
//   - $PATH / ${PATH}:那时 nvm/pnpm 等已把各自的 bin 注入 PATH,使 PATH=x:$PATH
//     表现为“在当前 PATH 前追加 x”而非用进程贫瘠 PATH 覆盖;
//   - 用户在本 env 里定义的其它变量(如 JAVA_HOME):使 PATH=$JAVA_HOME/bin:$PATH
//     这类同份 env 内的互相引用能被 shell 解析到本 env 刚赋的值。
//
// 其余 $VAR 仍由 Go 用进程环境展开。多个赋值逐条 export(而非单条并行赋值),并按
// 依赖排序——被引用的变量排在引用它的变量之前,shell 顺序执行才能读到新值。
func shellExports(env map[string]string) string {
	userVars := make(map[string]bool, len(env))
	for k := range env {
		if isShellEnvName(k) {
			userVars[k] = true
		}
	}
	if len(userVars) == 0 {
		return ""
	}
	ordered := orderByDependency(env, userVars)
	exports := make([]string, 0, len(ordered))
	for _, k := range ordered {
		exports = append(exports, "export "+shellQuote(k)+"="+shellExportValue(env[k], userVars))
	}
	return strings.Join(exports, "; ")
}

// orderByDependency 对 userVars 里的变量名排序:若 A 的值引用了 userVars 中的 B,
// 则 B 排在 A 之前。用稳定的贪心分层(每轮取出“不再依赖未输出变量”者,字母序),
// 存在环时把剩余变量按字母序兜底输出,保证终止。
func orderByDependency(env map[string]string, userVars map[string]bool) []string {
	remaining := make([]string, 0, len(userVars))
	for k := range userVars {
		remaining = append(remaining, k)
	}
	sort.Strings(remaining)

	done := make(map[string]bool, len(remaining))
	ordered := make([]string, 0, len(remaining))
	for len(remaining) > 0 {
		progressed := false
		next := remaining[:0]
		for _, k := range remaining {
			ready := true
			for dep := range referencedVars(env[k], userVars) {
				if dep != k && !done[dep] {
					ready = false
					break
				}
			}
			if ready {
				ordered = append(ordered, k)
				done[k] = true
				progressed = true
			} else {
				next = append(next, k)
			}
		}
		remaining = next
		if !progressed { // 依赖成环:剩余按字母序兜底,避免死循环
			ordered = append(ordered, remaining...)
			break
		}
	}
	return ordered
}

// keptVars 返回本次编码中“引用应保留给 shell 展开”的变量名集合:PATH 恒在其中,
// 再并入用户在本 env 里定义的变量名。
func keptVars(userVars map[string]bool) map[string]bool {
	kept := make(map[string]bool, len(userVars)+1)
	for k := range userVars {
		kept[k] = true
	}
	kept["PATH"] = true
	return kept
}

// referencedVars 返回 v 中引用到的、属于 userVars 的变量名集合(供依赖排序用)。
func referencedVars(v string, userVars map[string]bool) map[string]bool {
	out := map[string]bool{}
	rest := v
	for {
		idx, n, name := indexVarRef(rest, userVars)
		if idx < 0 {
			return out
		}
		out[name] = true
		rest = rest[idx+n:]
	}
}

// shellExportValue 把一个 env 值编码为 export 右侧的 shell 片段。对 keptVars 内变量
// 的 $VAR / ${VAR} 引用输出为未被单引号包裹的 "$VAR",让 shell 在 rc 之后展开
// (见 shellExports);其余内容(含其他 $VAR)先由 Go 用进程环境展开,再单引号包裹
// 避免二次展开与断词。
func shellExportValue(v string, userVars map[string]bool) string {
	kept := keptVars(userVars)
	var b strings.Builder
	rest := v
	for {
		idx, n, name := indexVarRef(rest, kept)
		if idx < 0 {
			if rest != "" {
				b.WriteString(shellQuote(os.Expand(rest, os.Getenv)))
			}
			if b.Len() == 0 {
				return "''" // 整个值为空,保留一个空引号占位
			}
			return b.String()
		}
		if idx > 0 {
			b.WriteString(shellQuote(os.Expand(rest[:idx], os.Getenv)))
		}
		b.WriteString(`"$` + name + `"`)
		rest = rest[idx+n:]
	}
}

// indexVarRef 返回 s 中第一个 $NAME 或 ${NAME}(NAME ∈ names)引用的起始下标、
// 匹配长度、变量名;没有则返回 (-1, 0, "")。$NAME 形式要求其后不紧跟标识符字符,
// 避免把 $PATHEXTRA 误当成 $PATH。
func indexVarRef(s string, names map[string]bool) (int, int, string) {
	for i := 0; i+1 < len(s); i++ {
		if s[i] != '$' {
			continue
		}
		if s[i+1] == '{' {
			if end := strings.IndexByte(s[i+2:], '}'); end >= 0 {
				name := s[i+2 : i+2+end]
				if names[name] {
					return i, 2 + end + 1, name // ${ + name + }
				}
			}
			continue
		}
		j := i + 1
		for j < len(s) && isIdentByte(s[j]) {
			j++
		}
		name := s[i+1 : j]
		if names[name] {
			return i, j - i, name
		}
	}
	return -1, 0, ""
}

func isIdentByte(b byte) bool {
	return b == '_' || b >= 'A' && b <= 'Z' || b >= 'a' && b <= 'z' || b >= '0' && b <= '9'
}

func isShellEnvName(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if r == '_' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' {
			continue
		}
		if i > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
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
	line := shellLine(spec)
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
