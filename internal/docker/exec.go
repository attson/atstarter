package docker

import (
	"bytes"
	"context"
	"os/exec"
	"time"
)

// execResult 是一次 CLI 调用的结果。
type execResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Err      error // 启动失败(如 docker 不存在)时非 nil
}

// execFunc 执行一条命令并返回结果。可注入以便测试。
type execFunc func(ctx context.Context, name string, args ...string) execResult

// fallbackTimeout 是无 deadline ctx 的兜底超时。生命周期命令(compose up 首次拉镜像)
// 可能耗时数分钟,故兜底给到 5 分钟,由调用方通过传入更短 deadline 收紧快命令。
const fallbackTimeout = 5 * time.Minute

// execDeadline 返回本次执行应采用的 deadline:
// 调用者 ctx 已有 deadline 则原样沿用;否则给一个兜底 deadline。
func execDeadline(ctx context.Context) time.Time {
	if d, ok := ctx.Deadline(); ok {
		return d
	}
	return time.Now().Add(fallbackTimeout)
}

// defaultExec 用真实 os/exec 执行。若调用者的 ctx 已带 deadline 则尊重之,
// 否则套一个 5 分钟兜底超时(避免命令永久阻塞)。
func defaultExec(ctx context.Context, name string, args ...string) execResult {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, fallbackTimeout)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, name, args...)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	err := cmd.Run()
	res := execResult{Stdout: out.String(), Stderr: errBuf.String()}
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			res.ExitCode = ee.ExitCode()
		} else {
			res.Err = err // 命令没跑起来(docker 未安装等)
		}
	}
	return res
}

// Client 持有一个执行器,是所有 docker 操作的入口。
type Client struct {
	exec execFunc
}

// New 构造用真实 docker CLI 的 Client。
func New() *Client { return &Client{exec: defaultExec} }

// newWithExec 构造注入 fake 执行器的 Client(测试用)。
func newWithExec(fn execFunc) *Client { return &Client{exec: fn} }
