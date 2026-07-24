//go:build !windows

package runner

// collectDescendants 返回 root(含)及其所有子孙进程的 pid,按"子孙在前、root 在后"
// 排序(叶子优先),便于杀进程时先杀子再杀父。
//
// 关键前提:调用时进程树尚未因根进程退出而被 reparent —— setsid 只改子进程的
// sid/pgid,不改 ppid,所以只要根进程还活着,像 dev.sh 那样另开进程组/会话的
// 子进程仍能经 ppid 链被找到。这正是进程组信号覆盖不到、却必须清理的孤儿来源。
//
// 进程列表的采集是平台相关的:Linux 读 /proc(process_tree_linux.go),
// darwin 走 sysctl(process_tree_darwin.go)。二者都归约为 ppid->children 映射,
// 再由 buildDescendantOrder 这一平台无关内核建树。
func collectDescendants(root int) []int {
	return buildDescendantOrder(root, readProcChildren())
}

// buildDescendantOrder 对 ppid->children 映射做后序遍历,返回 root(含)及全部
// 子孙的 pid,叶子优先、root 最后。平台无关,便于单测。
func buildDescendantOrder(root int, children map[int][]int) []int {
	var order []int
	var walk func(pid int)
	walk = func(pid int) {
		for _, c := range children[pid] {
			walk(c)
		}
		order = append(order, pid) // 后序:子孙先入,自身后入
	}
	walk(root)
	return order
}
