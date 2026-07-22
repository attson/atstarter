// Package docker 封装 docker / docker compose CLI 调用。
// CLI 执行走可注入的 execFunc(测试注 fake);输出解析在 parse.go 是纯函数。
package docker

import (
	"context"
	"strings"
)

// ContainerState 是一个容器的运行时快照(不落库)。
type ContainerState struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Image   string   `json:"image"`
	State   string   `json:"state"`   // running / exited / created / paused ...
	Status  string   `json:"status"`  // "Up 3 minutes" 人类可读串
	Compose string   `json:"compose"` // 所属 compose project 名;独立容器为空
	Service string   `json:"service"` // compose service 名;独立容器为空
	Ports   []string `json:"ports"`
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
