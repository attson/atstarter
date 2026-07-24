//go:build darwin

package runner

import "golang.org/x/sys/unix"

// readProcChildren 返回 ppid -> 子 pid 列表 的映射。
//
// darwin 没有 Linux 的 /proc 文件系统,进程信息经 sysctl 的 kern.proc.all 暴露:
// 一次调用拿到全部进程的 KinfoProc,其中 Proc.P_pid 是进程自身 pid、Eproc.Ppid
// 是父 pid。据此在内存里构建父子映射,语义与 Linux 版完全一致,供平台无关的
// buildDescendantOrder 建树。
//
// 历史坑:早期 process_tree 只有 /proc 实现(build tag !windows),在 macOS 上
// os.ReadDir("/proc") 直接失败返回空映射 —— collectDescendants 只剩顶层进程,
// killTree 杀不到任何子孙,导致 dev server 变孤儿、Stop/退出时 pump 永久阻塞。
// 本文件是该回归的修复。
func readProcChildren() map[int][]int {
	out := map[int][]int{}
	procs, err := unix.SysctlKinfoProcSlice("kern.proc.all")
	if err != nil {
		return out // 拿不到进程表时退化为空,collectDescendants 至少能杀顶层
	}
	for i := range procs {
		pid := int(procs[i].Proc.P_pid)
		ppid := int(procs[i].Eproc.Ppid)
		if pid <= 0 {
			continue
		}
		out[ppid] = append(out[ppid], pid)
	}
	return out
}
