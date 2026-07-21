// Package runner 管理子进程的启动/停止、输出捕获与状态维护。
// 平台相关的进程组/信号逻辑在 process_unix.go / process_windows.go。
package runner

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
)

// State 是一个受管进程的运行时状态。
type State string

const (
	StatusStopped State = "stopped"
	StatusRunning State = "running"
	StatusExited  State = "exited"
	StatusError   State = "error"
)

// Spec 描述一次启动请求。
type Spec struct {
	ID      string
	Command string
	Args    []string
	Dir     string            // 工作目录;为空则用当前目录
	Env     map[string]string // 叠加到 os.Environ() 之上
}

// LogLine 是一行输出,带来源项目与流类型。
type LogLine struct {
	ID     string
	Stream string // "stdout" 或 "stderr"
	Text   string
}

// Status 是对外暴露的状态快照。
type Status struct {
	State    State
	PID      int
	ExitCode int
}

// Runner 管理多个受管进程。并发安全。
type Runner struct {
	mu       sync.Mutex
	procs    map[string]*managed
	bufSize  int
	emit     func(LogLine)
	onStatus func(id string, st Status)
}

type managed struct {
	cmd    *exec.Cmd
	status Status
	logs   *ringBuffer
	pumps  sync.WaitGroup // stdout + stderr pumps drain before the exit marker is emitted
}

// New 构造 Runner。bufSize 是每个项目日志环形缓冲的行数。
func New(bufSize int) *Runner {
	return &Runner{
		procs:    map[string]*managed{},
		bufSize:  bufSize,
		emit:     func(LogLine) {},
		onStatus: func(string, Status) {},
	}
}

// SetEmitter 设置日志回调(Wails 层接成事件;测试里接 channel)。
func (r *Runner) SetEmitter(fn func(LogLine)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.emit = fn
}

// SetStatusListener 设置状态变化回调(Wails 层接成事件;测试里接 channel)。
func (r *Runner) SetStatusListener(fn func(id string, st Status)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.onStatus = fn
}

// Start 启动一个进程。若同 ID 已在运行则返回错误(幂等拒绝)。
func (r *Runner) Start(spec Spec) error {
	r.mu.Lock()
	if m, ok := r.procs[spec.ID]; ok && m.status.State == StatusRunning {
		r.mu.Unlock()
		return errors.New("runner: already running: " + spec.ID)
	}
	r.mu.Unlock()

	cmd := buildCmd(spec)
	if spec.Dir != "" {
		cmd.Dir = spec.Dir
	}
	cmd.Env = os.Environ()
	for k, v := range spec.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	setupProcAttr(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	m := &managed{cmd: cmd, logs: newRingBuffer(r.bufSize)}
	if err := cmd.Start(); err != nil {
		m.status = Status{State: StatusError}
		r.mu.Lock()
		r.procs[spec.ID] = m
		r.mu.Unlock()
		return err
	}
	m.status = Status{State: StatusRunning, PID: cmd.Process.Pid}
	r.mu.Lock()
	r.procs[spec.ID] = m
	runningSt := m.status // 值拷贝
	fn := r.onStatus
	r.mu.Unlock()
	fn(spec.ID, runningSt) // 锁外调用

	m.pumps.Add(2)
	go r.pump(spec.ID, m, stdout, "stdout")
	go r.pump(spec.ID, m, stderr, "stderr")
	go r.wait(spec.ID, m)
	return nil
}

// pump 逐行读取一个流,写入环形缓冲并 emit。
func (r *Runner) pump(id string, m *managed, pipe interface{ Read([]byte) (int, error) }, stream string) {
	defer m.pumps.Done()
	sc := bufio.NewScanner(pipe)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		m.logs.add(line)
		r.mu.Lock()
		emit := r.emit
		r.mu.Unlock()
		emit(LogLine{ID: id, Stream: stream, Text: line})
	}
}

// wait 等待进程结束并更新状态,并向日志追加一行退出标记。
func (r *Runner) wait(id string, m *managed) {
	// CRUCIAL: pumps must finish reading BEFORE cmd.Wait — the Go exec
	// docs explicitly say "it is incorrect to call Wait before all reads
	// from the pipe have completed" when StdoutPipe/StderrPipe are in
	// use, because Wait closes the parent side of the pipes and can
	// truncate pending reads. Pumps naturally exit on EOF once the child
	// closes its write ends (normal exit or kill), so waiting on them
	// blocks only until the process is actually done.
	m.pumps.Wait()
	err := m.cmd.Wait()
	r.mu.Lock()
	var marker string
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			m.status.State = StatusExited
			m.status.ExitCode = ee.ExitCode()
			marker = fmt.Sprintf("[process exited with code %d]", ee.ExitCode())
		} else {
			m.status.State = StatusError
			marker = fmt.Sprintf("[process error: %v]", err)
		}
	} else {
		m.status.State = StatusExited
		m.status.ExitCode = 0
		marker = "[process exited with code 0]"
	}
	st := m.status // 值拷贝
	fn := r.onStatus
	emit := r.emit
	r.mu.Unlock()

	// 追加退出标记到日志缓冲并推送(让日志区自证结局)。
	m.logs.add(marker)
	emit(LogLine{ID: id, Stream: "stderr", Text: marker})
	fn(id, st) // 锁外调用
}

// Stop 终止进程(整进程组)。未知 ID 或已停止视为成功。
func (r *Runner) Stop(id string) error {
	r.mu.Lock()
	m, ok := r.procs[id]
	if !ok || m.status.State != StatusRunning || m.cmd.Process == nil {
		r.mu.Unlock()
		return nil
	}
	pid := m.cmd.Process.Pid
	r.mu.Unlock()
	killTree(pid)
	_ = m.cmd.Process.Kill() // 兜底(Windows 占位路径依赖此行)
	return nil
}

// Status 返回某项目的状态快照;未知 ID 返回 stopped。
func (r *Runner) Status(id string) Status {
	r.mu.Lock()
	defer r.mu.Unlock()
	if m, ok := r.procs[id]; ok {
		return m.status
	}
	return Status{State: StatusStopped}
}

// RunningCount 返回当前处于 running 状态的受管进程数。并发安全。
func (r *Runner) RunningCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, m := range r.procs {
		if m.status.State == StatusRunning {
			n++
		}
	}
	return n
}

// Logs 返回某项目日志缓冲快照;未知 ID 返回空。
func (r *Runner) Logs(id string) []string {
	r.mu.Lock()
	m, ok := r.procs[id]
	r.mu.Unlock()
	if !ok {
		return nil
	}
	return m.logs.snapshot()
}

// StopAll 停止所有运行中的进程(App 退出时调用)。
func (r *Runner) StopAll() {
	r.mu.Lock()
	ids := make([]string, 0, len(r.procs))
	for id, m := range r.procs {
		if m.status.State == StatusRunning {
			ids = append(ids, id)
		}
	}
	r.mu.Unlock()
	for _, id := range ids {
		_ = r.Stop(id)
	}
}
