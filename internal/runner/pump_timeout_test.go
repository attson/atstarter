//go:build !windows

package runner

import (
	"testing"
	"time"
)

// TestWaitDoesNotHangWhenPipeHeldByOrphan 是防御纵深测试:即便进程树清理漏杀了
// 某个仍持有 stdout/stderr 写端的子孙(任何平台都可能因竞态或新派生而发生),
// wait 也必须在有限时间内把状态推进到终态,而不是永久卡在 running。
//
// 复现:顶层 shell setsid 出一个继承了 stdout 的后台 sleep(另开会话,进程组
// 信号覆盖不到),随后顶层自身退出。顶层退出后 cmd 的 stdout 管道写端仍被该
// 孤儿持有 → pump 读不到 EOF → 若 wait 无条件 m.pumps.Wait(),则永久阻塞,
// cmd.Wait 永不返回,状态卡死在 running(正是 macOS 上的退出无响应症状)。
//
// 期望:wait 对 pump 收尾设超时兜底,超时后照常 cmd.Wait 并落终态。
func TestWaitDoesNotHangWhenPipeHeldByOrphan(t *testing.T) {
	r := New(1000)
	// setsid 让 sleep 另开会话(不继承为顶层进程组成员);sleep 继承顶层 stdout。
	// 顶层 echo 后立即退出,但 sleep 仍持有 stdout 写端。
	spec := Spec{
		ID:      "orphan-pipe",
		Command: "sh",
		Args:    []string{"-c", "setsid sleep 60 & echo started"},
		Dir:     t.TempDir(),
	}
	if err := r.Start(spec); err != nil {
		t.Fatal(err)
	}
	// 顶层进程很快退出,但因孤儿持有管道,若无超时兜底,状态永远到不了 Exited。
	// 给足够时间覆盖兜底超时窗口。
	waitStatus(t, r, "orphan-pipe", StatusExited, 10*time.Second)

	// 清理:杀掉残留的 setsid sleep(尽力而为,不影响断言)。
	_ = r.Stop("orphan-pipe")
}
