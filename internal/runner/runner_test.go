package runner

import (
	"runtime"
	"strings"
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
	if runtime.GOOS == "windows" {
		t.Skip("shell wrapping is unix-specific")
	}
	r := New(100)
	r.SetEmitter(func(LogLine) {})
	spec := Spec{ID: "bad", Command: "definitely-not-a-real-binary-xyz", Dir: t.TempDir()}
	if err := r.Start(spec); err != nil {
		// 若 Start 直接失败也算处理正确(不静默成功)。
		return
	}
	// 否则:shell 启动成功,但子命令 not found → 非零退出。
	waitStatus(t, r, "bad", StatusExited, 5*time.Second)
	if st := r.Status("bad"); st.ExitCode == 0 {
		t.Errorf("missing binary should exit non-zero, got exit code 0 (state=%v)", st.State)
	}
}

func TestStatusCallbackFiresOnStateChange(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell command is unix-specific")
	}
	r := New(100)
	r.SetEmitter(func(LogLine) {})

	statuses := make(chan Status, 20)
	r.SetStatusListener(func(id string, st Status) {
		if id == "s1" {
			statuses <- st
		}
	})

	spec := Spec{ID: "s1", Command: "sh", Args: []string{"-c", "echo hi"}, Dir: t.TempDir()}
	if err := r.Start(spec); err != nil {
		t.Fatal(err)
	}

	// 应最终收到一个 exited 状态。
	deadline := time.After(5 * time.Second)
	sawExited := false
	for !sawExited {
		select {
		case st := <-statuses:
			if st.State == StatusExited {
				sawExited = true
			}
		case <-deadline:
			t.Fatal("did not receive exited status via callback")
		}
	}
}

// TestFastExitProcessLogsAreFullyCaptured 复现:进程快速退出时,
// wait() 不能在 pump 读完管道前关闭管道,否则日志丢失。
// 模拟真实场景——进程秒退后,前端才拉 Logs() 历史快照。
func TestFastExitProcessLogsAreFullyCaptured(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell command is unix-specific")
	}
	r := New(1000)
	r.SetEmitter(func(LogLine) {})

	// 立刻打印 10 行到 stdout 和 stderr 后马上退出(退出码非零,模拟编译失败)。
	spec := Spec{
		ID:      "fast",
		Command: "sh",
		Args:    []string{"-c", "for i in 1 2 3 4 5 6 7 8 9 10; do echo out$i; echo err$i 1>&2; done; exit 1"},
		Dir:     t.TempDir(),
	}
	if err := r.Start(spec); err != nil {
		t.Fatal(err)
	}

	// 等进程退出。
	waitStatus(t, r, "fast", StatusExited, 5*time.Second)

	// 进程退出后拉历史日志快照:必须包含全部 20 行输出(10 stdout + 10 stderr)
	// 加 1 行退出标记。轮询等标记落入缓冲。
	var logs []string
	deadline := time.After(2 * time.Second)
	for {
		logs = r.Logs("fast")
		if len(logs) >= 21 {
			break
		}
		select {
		case <-deadline:
			t.Fatalf("expected 21 lines (20 output + exit marker), got %d: %v", len(logs), logs)
		case <-time.After(20 * time.Millisecond):
		}
	}
	// 全部输出行 + 退出标记都应在。
	joined := strings.Join(logs, "\n")
	for _, want := range []string{"out1", "out10", "err1", "err10", "exited with code 1"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing log line %q; captured: %v", want, logs)
		}
	}
}

// TestExitAppendsMarkerToLogs 验证:进程退出后,日志尾部追加一行退出标记,
// 让日志区能自证结局(尤其是无输出后秒退的情况)。
func TestExitAppendsMarkerToLogs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell command is unix-specific")
	}
	r := New(1000)
	r.SetEmitter(func(LogLine) {})

	// 非零退出、无 stdout 输出(模拟缺子命令秒退)。
	spec := Spec{ID: "ex", Command: "sh", Args: []string{"-c", "exit 3"}, Dir: t.TempDir()}
	if err := r.Start(spec); err != nil {
		t.Fatal(err)
	}
	waitStatus(t, r, "ex", StatusExited, 5*time.Second)

	// 轮询等尾行落入缓冲(退出标记是在 wait 里 add 的,可能略晚于状态)。
	var logs []string
	deadline := time.After(2 * time.Second)
	for {
		logs = r.Logs("ex")
		if len(logs) > 0 && strings.Contains(logs[len(logs)-1], "exited with code 3") {
			return // 成功
		}
		select {
		case <-deadline:
			t.Fatalf("expected exit marker with code 3 in logs, got: %v", logs)
		case <-time.After(20 * time.Millisecond):
		}
	}
}

func TestShellJoin(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shellJoin is unix-only")
	}
	cases := []struct {
		name    string
		command string
		args    []string
		want    string
	}{
		{"simple", "go", []string{"run", "main.go"}, `'go' 'run' 'main.go'`},
		{"no args", "ls", nil, `'ls'`},
		{"path with space", "/opt/my app/bin", []string{"x"}, `'/opt/my app/bin' 'x'`},
		{"single quote in arg", "echo", []string{"it's"}, `'echo' 'it'\''s'`},
		{"special chars", "sh", []string{"-c", "a && b; c $x"}, `'sh' '-c' 'a && b; c $x'`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := shellJoin(tc.command, tc.args)
			if got != tc.want {
				t.Errorf("shellJoin(%q, %v) = %q, want %q", tc.command, tc.args, got, tc.want)
			}
		})
	}
}

func TestUserShell(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("userShell is unix-only")
	}
	t.Setenv("SHELL", "/usr/bin/zsh")
	if got := userShell(); got != "/usr/bin/zsh" {
		t.Errorf("userShell() with SHELL set = %q, want /usr/bin/zsh", got)
	}
	t.Setenv("SHELL", "")
	if got := userShell(); got != "/bin/sh" {
		t.Errorf("userShell() with empty SHELL = %q, want /bin/sh", got)
	}
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

func TestRunningCount(t *testing.T) {
	r := New(100)
	if got := r.RunningCount(); got != 0 {
		t.Fatalf("empty runner: want 0, got %d", got)
	}

	// 起两个长命令,应计数为 2
	if err := r.Start(Spec{ID: "a", Command: "sleep", Args: []string{"5"}}); err != nil {
		t.Fatalf("start a: %v", err)
	}
	if err := r.Start(Spec{ID: "b", Command: "sleep", Args: []string{"5"}}); err != nil {
		t.Fatalf("start b: %v", err)
	}
	// 给进程一点时间进入 running 状态
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if r.RunningCount() == 2 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if got := r.RunningCount(); got != 2 {
		t.Fatalf("two running: want 2, got %d", got)
	}

	// 停一个,应降到 1
	_ = r.Stop("a")
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if r.RunningCount() == 1 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if got := r.RunningCount(); got != 1 {
		t.Fatalf("after stop one: want 1, got %d", got)
	}
	r.StopAll()
}
