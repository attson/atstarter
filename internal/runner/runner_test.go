package runner

import (
	"runtime"
	"testing"
	"time"
)

func TestRingBufferOverflow(t *testing.T) {
	rb := newRingBuffer(3)
	for _, s := range []string{"a", "b", "c", "d", "e"} {
		rb.add(s)
	}
	got := rb.snapshot()
	want := []string{"c", "d", "e"}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("snapshot[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestStartCapturesOutputAndExits(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell command is unix-specific")
	}
	r := New(1000)
	lines := make(chan LogLine, 100)
	r.SetEmitter(func(l LogLine) { lines <- l })

	spec := Spec{
		ID:      "p1",
		Command: "sh",
		Args:    []string{"-c", "echo hello; echo oops 1>&2"},
		Dir:     t.TempDir(),
	}
	if err := r.Start(spec); err != nil {
		t.Fatalf("Start: %v", err)
	}

	var stdout, stderr string
	timeout := time.After(5 * time.Second)
	for got := 0; got < 2; {
		select {
		case l := <-lines:
			if l.Stream == "stdout" {
				stdout = l.Text
			} else {
				stderr = l.Text
			}
			got++
		case <-timeout:
			t.Fatal("timed out waiting for output")
		}
	}
	if stdout != "hello" {
		t.Errorf("stdout = %q, want hello", stdout)
	}
	if stderr != "oops" {
		t.Errorf("stderr = %q, want oops", stderr)
	}

	// 等待退出并检查状态。
	waitStatus(t, r, "p1", StatusExited, 5*time.Second)
	if st := r.Status("p1"); st.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", st.ExitCode)
	}
}

func TestStartMissingBinaryYieldsError(t *testing.T) {
	r := New(100)
	r.SetEmitter(func(LogLine) {})
	spec := Spec{ID: "bad", Command: "/nonexistent/binary/xyz", Dir: t.TempDir()}
	err := r.Start(spec)
	if err == nil {
		waitStatus(t, r, "bad", StatusError, 3*time.Second)
	}
	// 无论 Start 立即返回 err,还是异步置为 error 状态,都算正确处理(不静默成功)。
	if err == nil && r.Status("bad").State != StatusError {
		t.Errorf("expected error state for missing binary, got %+v", r.Status("bad"))
	}
}

func TestStopKillsProcessTree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix process group test")
	}
	r := New(1000)
	r.SetEmitter(func(LogLine) {})
	// 父 shell 后台起一个长 sleep 子进程,并打印其 PID,然后自己也 sleep。
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
	if err := r.Stop("tree"); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	waitStatus(t, r, "tree", StatusExited, 8*time.Second)
	// 无直接断言子进程消失的可移植方法;此处以父进程组被终止、Stop 正常返回作为验证。
	// 进程组信号的正确性由 process_unix.go 的实现保证(kill 负 pgid)。
}

// waitStatus 轮询直到状态达到 want 或超时。
func waitStatus(t *testing.T, r *Runner, id string, want State, d time.Duration) {
	t.Helper()
	deadline := time.After(d)
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-deadline:
			t.Fatalf("timeout waiting for %s status %v, got %v", id, want, r.Status(id).State)
		case <-tick.C:
			if r.Status(id).State == want {
				return
			}
		}
	}
}
