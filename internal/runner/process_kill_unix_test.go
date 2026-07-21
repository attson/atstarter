//go:build !windows

package runner

import (
	"fmt"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestStopKillsProcessTree(t *testing.T) {
	r := New(1000)
	childPID := make(chan int, 1)
	r.SetEmitter(func(l LogLine) {
		if strings.HasPrefix(l.Text, "CHILD ") {
			var pid int
			_, _ = fmt.Sscanf(l.Text, "CHILD %d", &pid)
			select {
			case childPID <- pid:
			default:
			}
		}
	})
	// 后台起一个长 sleep 子进程,打印其 PID,父自身也 sleep 保持运行。
	spec := Spec{
		ID:      "tree",
		Command: "sh",
		Args:    []string{"-c", "sleep 300 & echo CHILD $!; sleep 300"},
		Dir:     t.TempDir(),
	}
	if err := r.Start(spec); err != nil {
		t.Fatal(err)
	}
	waitStatus(t, r, "tree", StatusRunning, 3*time.Second)

	var pid int
	select {
	case pid = <-childPID:
	case <-time.After(3 * time.Second):
		t.Fatal("did not receive child PID")
	}
	if !processAlive(pid) {
		t.Fatalf("child %d not alive before Stop", pid)
	}

	if err := r.Stop("tree"); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	waitStatus(t, r, "tree", StatusExited, 8*time.Second)

	// 轮询等待子进程被进程组信号清理(SIGTERM→SIGKILL)。
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if !processAlive(pid) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("child process %d still alive after Stop — process tree not killed", pid)
}

// processAlive 用信号 0 探测进程是否存在(不实际发信号)。
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}
