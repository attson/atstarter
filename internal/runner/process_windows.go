//go:build windows

package runner

import "os/exec"

// setupProcAttr 在 Windows 上暂不做进程组设置(后续可接 Job Object)。
func setupProcAttr(cmd *exec.Cmd) {}

// killTree 在 Windows 上暂用简化实现:实际终止在 runner.Stop 中通过
// cmd.Process.Kill() 完成。此处保留空实现。
// TODO(后续): 接入 Job Object 保证子孙进程一并终止。
func killTree(pid int) {}
