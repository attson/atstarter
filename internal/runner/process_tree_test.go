//go:build !windows

package runner

import (
	"reflect"
	"testing"
)

// TestBuildDescendantOrder 验证纯建树逻辑:给定 ppid->children 映射,返回
// root(含)及全部子孙的后序遍历(叶子优先、root 最后)。这是 collectDescendants
// 的平台无关内核 —— Linux 从 /proc、darwin 从 sysctl 各自采集 children 映射,
// 再共用本函数建树,故本测试在所有 unix 平台都能跑,是 darwin 修复正确性的唯一
// 可在非 macOS 环境验证的锚点。
func TestBuildDescendantOrder(t *testing.T) {
	// 树形:
	//   1
	//   ├── 2
	//   │   └── 4
	//   └── 3
	children := map[int][]int{
		1: {2, 3},
		2: {4},
	}
	got := buildDescendantOrder(1, children)

	// 后序:子孙必须排在各自父之前;root(1)必须最后。
	if got[len(got)-1] != 1 {
		t.Fatalf("root should be last, got %v", got)
	}
	if !before(got, 4, 2) {
		t.Errorf("child 4 should come before parent 2, got %v", got)
	}
	if !before(got, 2, 1) {
		t.Errorf("child 2 should come before root 1, got %v", got)
	}
	if !before(got, 3, 1) {
		t.Errorf("child 3 should come before root 1, got %v", got)
	}
	// 完整性:恰好包含 1,2,3,4。
	want := map[int]bool{1: true, 2: true, 3: true, 4: true}
	if len(got) != len(want) {
		t.Fatalf("got %v, want exactly %v", got, want)
	}
	for _, p := range got {
		if !want[p] {
			t.Errorf("unexpected pid %d in %v", p, got)
		}
	}
}

// TestBuildDescendantOrderLoneRoot 验证无子孙时只返回 root 自身 —— 保证在拿不到
// 子进程信息(如数据源失败)时至少能杀顶层,不 panic、不空返回。
func TestBuildDescendantOrderLoneRoot(t *testing.T) {
	got := buildDescendantOrder(42, map[int][]int{})
	if !reflect.DeepEqual(got, []int{42}) {
		t.Fatalf("buildDescendantOrder(42, empty) = %v, want [42]", got)
	}
}

// before 报告 a 是否在 order 中排在 b 之前。二者都必须存在。
func before(order []int, a, b int) bool {
	ia, ib := -1, -1
	for i, v := range order {
		if v == a {
			ia = i
		}
		if v == b {
			ib = i
		}
	}
	return ia >= 0 && ib >= 0 && ia < ib
}
