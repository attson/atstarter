// Package docker 封装 docker / docker compose CLI 调用。
// CLI 执行走可注入的 execFunc(测试注 fake);输出解析在 parse.go 是纯函数。
package docker

import (
	"context"
	"strings"
)

// ContainerState 是一个容器的运行时快照(不落库)。
type ContainerState struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Image             string   `json:"image"`
	State             string   `json:"state"`             // running / exited / created / paused ...
	Status            string   `json:"status"`            // "Up 3 minutes" 人类可读串
	Compose           string   `json:"compose"`           // 所属 compose project 名;独立容器为空
	Service           string   `json:"service"`           // compose service 名;独立容器为空
	ComposeWorkingDir string   `json:"composeWorkingDir"` // compose project 工作目录;独立容器为空
	Ports             []string `json:"ports"`
}

// ComposeService 是 compose 项目下一个 service 的聚合视图。
type ComposeService struct {
	Name  string   `json:"name"`
	State string   `json:"state"` // 由该 service 的容器聚合:running/partial/stopped
	Image string   `json:"image"`
	Ports []string `json:"ports"`
}

// Info 是 Docker 可用性探测结果。
type Info struct {
	Available bool   `json:"available"`
	Version   string `json:"version"`
	Reason    string `json:"reason"` // 不可用时的人类可读原因
}

// Detect 探测 Docker 可用性。跑 `docker version --format '{{.Server.Version}}'`。
func (c *Client) Detect(ctx context.Context) Info {
	res := c.exec(ctx, "docker", "version", "--format", "{{.Server.Version}}")
	if res.Err == nil && res.ExitCode == 0 {
		return Info{Available: true, Version: strings.TrimSpace(res.Stdout)}
	}
	return Info{Available: false, Reason: classifyReason(res.Stderr, res.Err != nil)}
}

// ListContainers 返回 `docker ps -a` 快照。
func (c *Client) ListContainers(ctx context.Context) ([]ContainerState, error) {
	res := c.exec(ctx, "docker", "ps", "-a", "--format", "{{json .}}")
	if res.Err != nil {
		return nil, res.Err
	}
	if res.ExitCode != 0 {
		return nil, errFromResult(res)
	}
	return parsePs(res.Stdout)
}

// runVoid 执行一条命令,非零退出返回归类后的错误。
func (c *Client) runVoid(ctx context.Context, args ...string) error {
	res := c.exec(ctx, "docker", args...)
	if res.Err != nil {
		return res.Err
	}
	if res.ExitCode != 0 {
		return errFromResult(res)
	}
	return nil
}

func (c *Client) StartContainer(ctx context.Context, id string) error {
	return c.runVoid(ctx, "start", id)
}
func (c *Client) StopContainer(ctx context.Context, id string) error {
	return c.runVoid(ctx, "stop", id)
}
func (c *Client) RestartContainer(ctx context.Context, id string) error {
	return c.runVoid(ctx, "restart", id)
}
func (c *Client) RemoveContainer(ctx context.Context, id string, force bool) error {
	if force {
		return c.runVoid(ctx, "rm", "-f", id)
	}
	return c.runVoid(ctx, "rm", id)
}

// composeBase 构造 `compose --project-directory <dir> [-f <file>]` 前缀。
// file 非空时追加 -f;其相对 --project-directory 解析,原样透传。
func (c *Client) composeBase(dir, file string) []string {
	base := []string{"compose", "--project-directory", dir}
	if file != "" {
		base = append(base, "-f", file)
	}
	return base
}

func (c *Client) ComposeUp(ctx context.Context, dir, file, service string) error {
	args := append(c.composeBase(dir, file), "up", "-d")
	if service != "" {
		args = append(args, service)
	}
	return c.runVoid(ctx, args...)
}
func (c *Client) ComposeStop(ctx context.Context, dir, file, service string) error {
	args := append(c.composeBase(dir, file), "stop")
	if service != "" {
		args = append(args, service)
	}
	return c.runVoid(ctx, args...)
}
func (c *Client) ComposeRestart(ctx context.Context, dir, file, service string) error {
	args := append(c.composeBase(dir, file), "restart")
	if service != "" {
		args = append(args, service)
	}
	return c.runVoid(ctx, args...)
}
func (c *Client) ComposeDown(ctx context.Context, dir, file string) error {
	return c.runVoid(ctx, append(c.composeBase(dir, file), "down")...)
}

// ListServiceNames 返回 compose 项目的 service 名列表。
func (c *Client) ListServiceNames(ctx context.Context, dir, file string) ([]string, error) {
	args := append(c.composeBase(dir, file), "config", "--services")
	res := c.exec(ctx, "docker", args...)
	if res.Err != nil {
		return nil, res.Err
	}
	if res.ExitCode != 0 {
		return nil, errFromResult(res)
	}
	return parseServiceNames(res.Stdout), nil
}

// ExecResult 是 CLI 调用结果(导出以便上层注入测试)。
type ExecResult = execResult

// NewWithExecForTest 构造注入自定义执行器的 Client,供上层集成测试用。
func NewWithExecForTest(fn func(ctx context.Context, name string, args ...string) execResult) *Client {
	return &Client{exec: fn}
}

// ComposeServices 聚合 service 名与容器快照(导出封装,供上层用)。
func ComposeServices(project string, names []string, containers []ContainerState) []ComposeService {
	return aggregateServices(project, names, containers)
}
