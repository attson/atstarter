//go:build linux

package main

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFile 在 root 下创建 rel 指向的文件(含父目录),内容为 content。
func writeGPUFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", rel, err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

// nvidiaVersionFile 是 /proc/driver/nvidia/version 的真实格式样本,内核模块版本 595.71.05。
const nvidiaVersionFile = "NVRM version: NVIDIA UNIX Open Kernel Module for x86_64  595.71.05  Release Build  (dvs-builder@host)  Fri Apr 24 06:42:30 UTC 2026\nGCC version:  gcc version 13.3.0\n"

func TestShouldDisableDMABUF_NvidiaVersionMismatch(t *testing.T) {
	root := t.TempDir()
	// 内核模块报告 595.71.05
	writeGPUFile(t, root, "proc/driver/nvidia/version", nvidiaVersionFile)
	// 用户态库却是 595.84 —— 经典 mismatch,正是本次崩溃的成因
	writeGPUFile(t, root, "usr/lib/x86_64-linux-gnu/libGLX_nvidia.so.595.84", "")
	// 补一个 DRI 设备,排除"无设备"信号干扰,确保命中的是 mismatch
	writeGPUFile(t, root, "dev/dri/renderD128", "")

	disable, reason := shouldDisableDMABUF(root)
	if !disable {
		t.Fatalf("mismatch 应触发禁用 DMABUF,却返回 false (reason=%q)", reason)
	}
	if reason == "" {
		t.Fatal("触发禁用时应给出非空 reason")
	}
}

func TestShouldDisableDMABUF_NvidiaVersionMatches(t *testing.T) {
	root := t.TempDir()
	writeGPUFile(t, root, "proc/driver/nvidia/version", nvidiaVersionFile)
	// 用户态库版本与内核模块一致 —— GPU 正常,不该禁用
	writeGPUFile(t, root, "usr/lib/x86_64-linux-gnu/libGLX_nvidia.so.595.71.05", "")
	writeGPUFile(t, root, "dev/dri/renderD128", "")

	disable, reason := shouldDisableDMABUF(root)
	if disable {
		t.Fatalf("版本一致时不应禁用 DMABUF (reason=%q)", reason)
	}
}

func TestShouldDisableDMABUF_NoDRIDevice(t *testing.T) {
	root := t.TempDir()
	// 无 NVIDIA(非 N 卡机器),但 /dev/dri 下没有任何 render 节点 —— 无可用 GPU 渲染路径
	writeGPUFile(t, root, "dev/dri/.keep", "")

	disable, reason := shouldDisableDMABUF(root)
	if !disable {
		t.Fatalf("无 renderD* 设备应触发禁用 (reason=%q)", reason)
	}
	if reason == "" {
		t.Fatal("触发禁用时应给出非空 reason")
	}
}

func TestShouldDisableDMABUF_HealthyNonNvidia(t *testing.T) {
	root := t.TempDir()
	// 无 NVIDIA,但有正常的 render 节点(Intel/AMD)—— 健康环境,不该禁用
	writeGPUFile(t, root, "dev/dri/renderD128", "")
	writeGPUFile(t, root, "dev/dri/card0", "")

	disable, reason := shouldDisableDMABUF(root)
	if disable {
		t.Fatalf("健康非 N 卡环境不应禁用 (reason=%q)", reason)
	}
}

func TestShouldDisableDMABUF_NvidiaVersionUnreadableIsConservative(t *testing.T) {
	root := t.TempDir()
	// 有 NVIDIA 用户态库,但读不到 /proc/driver/nvidia/version(模块未加载/权限问题)。
	// 无法确认版本是否匹配时,保守起见不禁用(有 render 节点说明有 GPU),避免误伤正常环境。
	writeGPUFile(t, root, "usr/lib/x86_64-linux-gnu/libGLX_nvidia.so.595.84", "")
	writeGPUFile(t, root, "dev/dri/renderD128", "")

	disable, _ := shouldDisableDMABUF(root)
	if disable {
		t.Fatal("无法读取内核模块版本且有 GPU 设备时应保守不禁用")
	}
}
