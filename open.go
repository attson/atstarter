package main

import (
	"os/exec"
	"runtime"
)

// openInSystem 用操作系统默认程序打开 path(文件或目录)。
func openInSystem(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	default: // linux 及其它
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}
