package docker

import "strings"

// classifyReason 把 docker 命令的 stderr / 启动错误归类成人类可读原因。
func classifyReason(stderr string, startErr bool) string {
	if startErr {
		return "docker 未安装或不在 PATH"
	}
	low := strings.ToLower(stderr)
	switch {
	case strings.Contains(low, "cannot connect to the docker daemon"),
		strings.Contains(low, "is the docker daemon running"):
		return "docker daemon 未运行"
	case strings.Contains(low, "permission denied") && strings.Contains(low, "docker daemon"):
		return "权限不足(当前用户可能不在 docker 组)"
	}
	return strings.TrimSpace(stderr)
}
