package docker

import (
	"encoding/json"
	"strings"
)

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

type psLine struct {
	ID     string `json:"ID"`
	Names  string `json:"Names"`
	Image  string `json:"Image"`
	State  string `json:"State"`
	Status string `json:"Status"`
	Ports  string `json:"Ports"`
	Labels string `json:"Labels"`
}

// parsePs 解析 `docker ps -a --format '{{json .}}'`(每行一个 JSON)。
func parsePs(out string) ([]ContainerState, error) {
	var res []ContainerState
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var pl psLine
		if err := json.Unmarshal([]byte(line), &pl); err != nil {
			return nil, err
		}
		labels := parseLabels(pl.Labels)
		res = append(res, ContainerState{
			ID:      pl.ID,
			Name:    pl.Names,
			Image:   pl.Image,
			State:   pl.State,
			Status:  pl.Status,
			Compose: labels["com.docker.compose.project"],
			Service: labels["com.docker.compose.service"],
			Ports:   splitPorts(pl.Ports),
		})
	}
	return res, nil
}

// parseLabels 解析 "k1=v1,k2=v2" 形式的 label 串。
func parseLabels(s string) map[string]string {
	m := map[string]string{}
	for _, kv := range strings.Split(s, ",") {
		if i := strings.IndexByte(kv, '='); i > 0 {
			m[strings.TrimSpace(kv[:i])] = strings.TrimSpace(kv[i+1:])
		}
	}
	return m
}

// splitPorts 把逗号分隔的端口串拆成切片;空串返回 nil。
func splitPorts(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
