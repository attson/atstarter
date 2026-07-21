//go:build !windows

package runner

import (
	"os"
	"strconv"
)

// collectDescendants 返回 root(含)及其所有子孙进程的 pid,按"子孙在前、root 在后"
// 排序(叶子优先),便于杀进程时先杀子再杀父。
//
// 通过 /proc/<pid>/stat 读取每个进程的 ppid 构建父子关系。关键前提:调用时进程树
// 尚未因根进程退出而被 reparent —— setsid 只改子进程的 sid/pgid,不改 ppid,所以
// 只要根进程还活着,像 dev.sh 那样另开进程组/会话的子进程仍能经 ppid 链被找到。
// 这正是进程组信号覆盖不到、却必须清理的孤儿来源。
func collectDescendants(root int) []int {
	children := readProcChildren() // ppid -> []pid

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

// readProcChildren 扫描 /proc,返回 ppid -> 子 pid 列表 的映射。
func readProcChildren() map[int][]int {
	out := map[int][]int{}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return out
	}
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue // 非 pid 目录
		}
		ppid, ok := readPPID(pid)
		if !ok {
			continue
		}
		out[ppid] = append(out[ppid], pid)
	}
	return out
}

// readPPID 从 /proc/<pid>/stat 解析父 pid。stat 第 4 个字段是 ppid,但第 2 个字段
// comm 可能含空格与括号(如 "(a b)"),故从最后一个 ')' 之后开始按空格切分,
// 避开 comm 干扰:) 之后的字段依次是 state(1) ppid(2)...
func readPPID(pid int) (int, bool) {
	data, err := os.ReadFile("/proc/" + strconv.Itoa(pid) + "/stat")
	if err != nil {
		return 0, false
	}
	s := string(data)
	rparen := lastIndexByte(s, ')')
	if rparen < 0 || rparen+2 >= len(s) {
		return 0, false
	}
	rest := s[rparen+2:] // 跳过 ") "
	// rest = "<state> <ppid> ..."
	fields := splitFields(rest)
	if len(fields) < 2 {
		return 0, false
	}
	ppid, err := strconv.Atoi(fields[1])
	if err != nil {
		return 0, false
	}
	return ppid, true
}

func lastIndexByte(s string, b byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == b {
			return i
		}
	}
	return -1
}

// splitFields 按空白切分,返回前若干字段(够解析 ppid 即可,不必全切)。
func splitFields(s string) []string {
	var out []string
	i := 0
	for i < len(s) && len(out) < 3 {
		for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n') {
			i++
		}
		start := i
		for i < len(s) && s[i] != ' ' && s[i] != '\t' && s[i] != '\n' {
			i++
		}
		if start < i {
			out = append(out, s[start:i])
		}
	}
	return out
}
