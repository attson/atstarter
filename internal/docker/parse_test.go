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

func TestParsePs(t *testing.T) {
	// 两行:一个 compose 容器 + 一个独立容器
	sample := `{"ID":"abc123","Names":"myapp-web-1","Image":"nginx:alpine","State":"running","Status":"Up 3 minutes","Ports":"0.0.0.0:8080->80/tcp","Labels":"com.docker.compose.project=myapp,com.docker.compose.service=web"}
{"ID":"def456","Names":"redis","Image":"redis:7.2","State":"exited","Status":"Exited (0) 2 hours ago","Ports":"","Labels":"foo=bar"}`
	got, err := parsePs(sample)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Name != "myapp-web-1" || got[0].Compose != "myapp" || got[0].Service != "web" {
		t.Errorf("compose container = %+v", got[0])
	}
	if got[0].Ports[0] != "0.0.0.0:8080->80/tcp" {
		t.Errorf("ports = %v", got[0].Ports)
	}
	if got[1].Name != "redis" || got[1].Compose != "" {
		t.Errorf("standalone container = %+v", got[1])
	}
}

func TestAggregateServices(t *testing.T) {
	names := []string{"web", "api", "db"}
	containers := []ContainerState{
		{Name: "myapp-web-1", Image: "nginx", State: "running", Compose: "myapp", Service: "web", Ports: []string{":8080->80"}},
		{Name: "myapp-api-1", Image: "api:dev", State: "running", Compose: "myapp", Service: "api"},
		// db 没有容器 → stopped
		{Name: "redis", State: "running", Compose: "", Service: ""}, // 独立容器,不该混入
	}
	got := aggregateServices("myapp", names, containers)
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	byName := map[string]ComposeService{}
	for _, s := range got {
		byName[s.Name] = s
	}
	if byName["web"].State != "running" || byName["web"].Image != "nginx" {
		t.Errorf("web = %+v", byName["web"])
	}
	if byName["db"].State != "stopped" {
		t.Errorf("db state = %q, want stopped", byName["db"].State)
	}
}
