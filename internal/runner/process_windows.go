//go:build windows

package runner

import "os/exec"

// setupProcAttr 在 Windows 上暂不做进程组设置(后续可接 Job Object)。
func setupProcAttr(cmd *exec.Cmd) {}

// killTree 在 Windows 上暂用简化实现:直接 Kill 主进程。
// TODO(后续): 接入 Job Object 保证子孙进程一并终止。
func killTree(cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

// buildCmd 在 Windows 上直接执行命令(不包 shell)。
func buildCmd(spec Spec) *exec.Cmd {
	return exec.Command(spec.Command, spec.Args...)
}

// isShellNoise 在 Windows 上恒为 false:命令不经登录交互式 shell 包裹,
// 无 job-control 启动噪声可过滤。
func isShellNoise(line string) bool { return false }
