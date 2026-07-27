//go:build linux

package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// maybeDisableDMABUF 在 Linux 上启动前调用:当探测到已知的、会让 WebKitGTK
// 在 DMABUF/GBM 渲染路径上 abort 的环境信号时,兜底设置
// WEBKIT_DISABLE_DMABUF_RENDERER=1,把 WebKit 引到软件合成路径,避免整个进程
// SIGABRT。C 层的 abort 无法被 Go recover,所以只能"事前避让"而非"事后接住"。
//
// 只读文件、不触碰 EGL/GBM,因此探测本身绝不会崩。GPU 环境正常时不做任何改动,
// 保留硬件加速。用户已显式设置该变量时一律尊重,不覆盖。
func maybeDisableDMABUF() {
	const key = "WEBKIT_DISABLE_DMABUF_RENDERER"
	if _, set := os.LookupEnv(key); set {
		return // 用户已表态,不干预
	}
	if disable, reason := shouldDisableDMABUF("/"); disable {
		os.Setenv(key, "1")
		println("[gpu] 检测到 GPU 渲染异常,已禁用 WebKit DMABUF 渲染器:", reason)
	}
}

// nvidiaLibVersionRe 从 libGLX_nvidia.so.<ver> 之类的文件名里抽版本号。
var nvidiaLibVersionRe = regexp.MustCompile(`libGLX_nvidia\.so\.(\d+\.\d+(?:\.\d+)?)$`)

// nvrmVersionRe 从 /proc/driver/nvidia/version 的 NVRM 行里抽内核模块版本号。
var nvrmVersionRe = regexp.MustCompile(`NVRM version:.*?(\d+\.\d+\.\d+)`)

// shouldDisableDMABUF 在以 root 为根的文件树上判断是否应禁用 DMABUF 渲染。
// root 通常是 "/",测试时传临时目录。返回是否禁用及可读的原因。
func shouldDisableDMABUF(root string) (bool, string) {
	// 信号一:NVIDIA 驱动 Driver/library version mismatch。
	// 内核模块版本(/proc/driver/nvidia/version)与用户态库版本对不上时,
	// EGL 初始化会失败 —— 正是本次崩溃的根因。
	if kernelVer, ok := readNVRMVersion(root); ok {
		if libVer, ok := readNvidiaLibVersion(root); ok && libVer != kernelVer {
			return true, "NVIDIA 驱动版本不一致(内核模块 " + kernelVer + " ≠ 用户态库 " + libVer + "),多为升级后未重启"
		}
		// 读到了内核版本但读不到库版本,或两者一致:不因 NVIDIA 而禁用。
	}

	// 信号二:没有任何可用的 DRI render 节点,说明没有可走的 GPU 渲染路径。
	if !hasRenderNode(root) {
		return true, "未发现可用的 /dev/dri render 设备,无 GPU 渲染路径"
	}

	return false, ""
}

// readNVRMVersion 读取 /proc/driver/nvidia/version 中的内核模块版本。
func readNVRMVersion(root string) (string, bool) {
	data, err := os.ReadFile(filepath.Join(root, "proc/driver/nvidia/version"))
	if err != nil {
		return "", false
	}
	m := nvrmVersionRe.FindStringSubmatch(string(data))
	if m == nil {
		return "", false
	}
	return m[1], true
}

// readNvidiaLibVersion 在常见库目录下查找 libGLX_nvidia.so.<ver> 并抽出版本号。
func readNvidiaLibVersion(root string) (string, bool) {
	libDirs := []string{
		"usr/lib/x86_64-linux-gnu",
		"usr/lib64",
		"usr/lib",
	}
	for _, d := range libDirs {
		entries, err := os.ReadDir(filepath.Join(root, d))
		if err != nil {
			continue
		}
		for _, e := range entries {
			if m := nvidiaLibVersionRe.FindStringSubmatch(e.Name()); m != nil {
				return m[1], true
			}
		}
	}
	return "", false
}

// hasRenderNode 判断 /dev/dri 下是否存在 renderD* 节点。
func hasRenderNode(root string) bool {
	entries, err := os.ReadDir(filepath.Join(root, "dev/dri"))
	if err != nil {
		return false
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "renderD") {
			return true
		}
	}
	return false
}
