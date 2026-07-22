package docker

import "testing"

func TestClassifyReason(t *testing.T) {
	cases := []struct {
		name     string
		stderr   string
		startErr bool // 命令没跑起来
		want     string
	}{
		{"not installed", "", true, "docker 未安装或不在 PATH"},
		{"daemon down", "Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?", false, "docker daemon 未运行"},
		{"permission", "permission denied while trying to connect to the Docker daemon socket", false, "权限不足(当前用户可能不在 docker 组)"},
		{"other", "some other error", false, "some other error"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := classifyReason(c.stderr, c.startErr)
			if got != c.want {
				t.Errorf("classifyReason = %q, want %q", got, c.want)
			}
		})
	}
}
