package main

import (
	"os/exec"
	"path/filepath"
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

// revealInSystem 在系统文件管理器里定位 path(选中该条目),而不是打开它。
// Linux 没有统一的 "reveal" 协议,退化为打开父目录。
func revealInSystem(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", "-R", path)
	case "windows":
		cmd = exec.Command("explorer", "/select,"+path)
	default:
		cmd = exec.Command("xdg-open", filepath.Dir(path))
	}
	return cmd.Start()
}
