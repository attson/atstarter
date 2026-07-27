//go:build !linux

package main

// maybeDisableDMABUF 仅在 Linux 上有实际逻辑;macOS/Windows 的 WebView
// 不使用 WebKitGTK 的 DMABUF/GBM 渲染路径,这里是空操作。
func maybeDisableDMABUF() {}
