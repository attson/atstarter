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

// defaultExec 用真实 os/exec 执行,带 10s 超时兜底。
func defaultExec(ctx context.Context, name string, args ...string) execResult {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
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
