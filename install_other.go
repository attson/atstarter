//go:build !darwin

package main

// platformInstall 非 darwin(linux / windows)沿用旧的外部脚本方案。darwin
// 走 install_darwin.go 里的内联实现,那边"派 bash 后立即退出、赌孤儿存活"的
// 模型在 macOS 上已知会踩 Launch Services 竞态,先在 darwin 修掉。
func platformInstall(asset, target, execPath string) error {
	return runInstallScript(asset, target, execPath)
}
