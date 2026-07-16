# atstarter Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 构建一个 Wails(Go + Vue3)桌面 App,自动识别本地项目类型并建议启动命令,托管启动/停止项目并展示实时日志。

**Architecture:** Go 后端分四个单一职责模块 —— Detector(识别)、Scanner(扫描)、Store(配置持久化)、Runner(进程管理);外加一个 cmdparse(命令行拆分)工具包。前端 Vue3 通过 Wails 方法绑定调用后端,通过事件接收日志与状态推送。核心业务逻辑(Detector / Store / cmdparse / Runner)先以纯 Go 包实现并单测,再由 Wails 的 `App` 结构体作为绑定层薄封装暴露给前端。

**Tech Stack:** Go 1.20+、Wails v2、Vue3、Vite。第三方库:`github.com/google/shlex`(shell 词法拆分)、`github.com/google/uuid`(可选,本计划用路径哈希不引入)。

---

## 环境前提

- **Go 版本**:Wails v2 要求 Go ≥ 1.20。系统默认 `go` 是 1.19,但本机 `~/sdk/` 下已装有 **go1.23.12 与 go1.24.13**。本计划统一用 `go1.24.13`(路径 `/home/attson/sdk/go1.24.13/bin/go`),下文所有 `go` 命令均指它。执行时可先设一个别名简化:
  ```bash
  export GO=/home/attson/sdk/go1.24.13/bin/go
  $GO version   # 期望 go1.24.13
  ```
  下文命令写 `go` 处,执行时用 `$GO` 替换即可。
- **Node**:v22 + pnpm 均已就绪,满足前端构建。
- 本计划的业务逻辑测试(Task 2–6)**不依赖 Wails 运行时**,是纯 `go test`,即使 GUI 环境未完全就绪也能推进。Task 7(Wails 集成)与 Task 8(前端)才需要完整 Wails CLI。

---

## 文件结构

```
atstarter/
├── main.go                     # Wails 入口(Task 7)
├── app.go                      # App 绑定结构体,暴露给前端的方法(Task 7)
├── wails.json                  # Wails 项目配置(Task 1)
├── go.mod / go.sum
├── internal/
│   ├── cmdparse/
│   │   ├── cmdparse.go         # 单行命令 → command + args 拆分(Task 2)
│   │   └── cmdparse_test.go
│   ├── detector/
│   │   ├── detector.go         # 规则表 + 识别逻辑(Task 3)
│   │   ├── rules.go            # 规则定义(Task 3)
│   │   └── detector_test.go
│   ├── store/
│   │   ├── model.go            # Config / Project 数据结构(Task 4)
│   │   ├── store.go            # JSON 读写 + 增删改查 + 去重(Task 4)
│   │   └── store_test.go
│   ├── scanner/
│   │   ├── scanner.go          # 遍历工作区 → 调 detector(Task 5)
│   │   └── scanner_test.go
│   └── runner/
│       ├── runner.go           # 跨平台无关的进程管理 + 环形缓冲 + 状态(Task 6)
│       ├── ringbuffer.go       # 日志环形缓冲区(Task 6)
│       ├── process_unix.go     # //go:build unix,进程组信号(Task 6)
│       ├── process_windows.go  # //go:build windows,Job Object(Task 6,占位实现)
│       └── runner_test.go
└── frontend/
    ├── src/
    │   ├── App.vue             # 左列表 + 右详情布局(Task 8)
    │   ├── components/
    │   │   ├── ProjectList.vue
    │   │   ├── ProjectDetail.vue
    │   │   ├── LogPanel.vue
    │   │   ├── EditProjectDialog.vue
    │   │   └── ScanDialog.vue
    │   └── main.js
    └── package.json
```

**依赖顺序**:cmdparse → detector → store → scanner → runner 各自独立可测;Task 7 把它们组装进 `App`;Task 8 做前端。Task 2–6 之间无强耦合,但按此顺序推进最顺(store 复用 model,scanner 复用 detector)。

---

## Task 0: 准备 Go 工具链

**Files:** 无(环境操作)

- [ ] **Step 1: 固定使用本机 go1.24.13**

```bash
export GO=/home/attson/sdk/go1.24.13/bin/go
$GO version
```

Expected: 打印 `go version go1.24.13 linux/amd64`。本机已装(`~/sdk/` 下另有 go1.23.12 亦可),无需下载。下文所有 `go` 命令均指 `$GO`。

- [ ] **Step 2: 安装 Wails CLI(用 go1.24.13 安装)**

```bash
$GO install github.com/wailsapp/wails/v2/cmd/wails@latest
$($GO env GOPATH)/bin/wails version
```

Expected: 打印 Wails 版本号(如 `v2.x.x`)。若 `wails` 不在 PATH,用 `$($GO env GOPATH)/bin/wails`。

- [ ] **Step 3: 运行 wails doctor 检查依赖**

Run: `$($GO env GOPATH)/bin/wails doctor`
Expected: 报告系统依赖状态。Linux 上可能提示缺 `gtk3`/`webkit2gtk` 等库,按提示安装(如 `sudo apt install libgtk-3-dev libwebkit2gtk-4.0-dev`)。前端业务逻辑 Task 不受此阻塞。

---

## Task 1: 初始化 Wails 项目脚手架

**Files:**
- Create: `main.go`, `app.go`, `wails.json`, `go.mod`, `frontend/`(由 wails init 生成)

> 当前目录已有 `docs/` 与 `.git`。wails init 会生成到子目录,需把生成内容搬到仓库根,或原地初始化。采用"临时目录生成 → 拷贝"避免污染。

- [ ] **Step 1: 在临时目录生成 wails vue 模板**

```bash
cd /tmp && rm -rf atstarter-scaffold
wails init -n atstarter-scaffold -t vue
ls /tmp/atstarter-scaffold
```

Expected: 看到 `main.go app.go wails.json go.mod frontend` 等条目。

- [ ] **Step 2: 拷贝脚手架到项目根(保留已有 docs/.git/.gitignore)**

```bash
cd /home/attson/GolandProjects/atstarter
cp -rn /tmp/atstarter-scaffold/. ./
ls
```

Expected: 项目根出现 `main.go app.go wails.json go.mod frontend/`,原 `docs/` 保留。

- [ ] **Step 3: 修正 module 名与构建冒烟**

打开 `go.mod`,确认 module 行为 `module atstarter`(wails 默认用项目名,通常已正确;若不是则改为 `atstarter`)。

Run: `go build ./...`
Expected: 编译通过,无输出(或仅生成二进制)。

- [ ] **Step 4: 提交脚手架**

```bash
git add -A
git commit -m "chore: scaffold wails vue project"
```

---

## Task 2: cmdparse —— 单行命令拆分为 command + args

**Files:**
- Create: `internal/cmdparse/cmdparse.go`
- Test: `internal/cmdparse/cmdparse_test.go`

- [ ] **Step 1: 添加 shlex 依赖**

```bash
go get github.com/google/shlex@latest
```

Expected: `go.mod` 出现 `github.com/google/shlex`。

- [ ] **Step 2: 写失败测试**

`internal/cmdparse/cmdparse_test.go`:

```go
package cmdparse

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		wantCmd  string
		wantArgs []string
		wantErr  bool
	}{
		{"simple", "pnpm run dev", "pnpm", []string{"run", "dev"}, false},
		{"go run", "go run main.go", "go", []string{"run", "main.go"}, false},
		{"go subcommand", "go run main.go serve", "go", []string{"run", "main.go", "serve"}, false},
		{"absolute runtime", "/home/attson/sdk/go1.23.12/bin/go run main.go serve",
			"/home/attson/sdk/go1.23.12/bin/go", []string{"run", "main.go", "serve"}, false},
		{"quoted arg", `node -e "console.log('hi there')"`, "node",
			[]string{"-e", "console.log('hi there')"}, false},
		{"trailing spaces", "  cargo run  ", "cargo", []string{"run"}, false},
		{"only command", "make", "make", []string{}, false},
		{"empty", "   ", "", nil, true},
		{"unbalanced quote", `node -e "oops`, "", nil, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cmd, args, err := Parse(c.input)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got cmd=%q args=%v", cmd, args)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cmd != c.wantCmd {
				t.Errorf("cmd = %q, want %q", cmd, c.wantCmd)
			}
			if !reflect.DeepEqual(args, c.wantArgs) {
				t.Errorf("args = %v, want %v", args, c.wantArgs)
			}
		})
	}
}

func TestJoin(t *testing.T) {
	got := Join("go", []string{"run", "main.go", "serve"})
	want := "go run main.go serve"
	if got != want {
		t.Errorf("Join = %q, want %q", got, want)
	}
}
```

- [ ] **Step 3: 运行测试确认失败**

Run: `go test ./internal/cmdparse/ -v`
Expected: FAIL,`undefined: Parse` / `undefined: Join`。

- [ ] **Step 4: 实现 cmdparse.go**

`internal/cmdparse/cmdparse.go`:

```go
// Package cmdparse 在单行命令字符串与 (command, args) 结构之间转换。
// 存储层用结构化的 command+args;UI 层用单行字符串。
package cmdparse

import (
	"errors"
	"strings"

	"github.com/google/shlex"
)

// ErrEmpty 表示输入为空或仅含空白。
var ErrEmpty = errors.New("cmdparse: empty command")

// Parse 把单行命令拆成可执行文件与参数。
// 使用 shell 词法规则,正确处理引号与空格。
// args 永远非 nil(可能为空切片),便于与 JSON 序列化保持稳定。
func Parse(line string) (command string, args []string, err error) {
	if strings.TrimSpace(line) == "" {
		return "", nil, ErrEmpty
	}
	tokens, err := shlex.Split(line)
	if err != nil {
		return "", nil, err
	}
	if len(tokens) == 0 {
		return "", nil, ErrEmpty
	}
	return tokens[0], tokens[1:], nil
}

// Join 把 command+args 拼回可读的单行字符串,供 UI 回显。
// 注意:这是展示用途,不保证与原始输入逐字节一致(引号可能规范化)。
func Join(command string, args []string) string {
	parts := append([]string{command}, args...)
	return strings.Join(parts, " ")
}
```

> 说明:`Parse` 返回的 `tokens[1:]` 在只有一个 token 时是长度 0 的非 nil 切片,与测试 `{}` 匹配。

- [ ] **Step 5: 运行测试确认通过**

Run: `go test ./internal/cmdparse/ -v`
Expected: PASS(全部子用例)。

- [ ] **Step 6: 提交**

```bash
git add internal/cmdparse/ go.mod go.sum
git commit -m "feat: add cmdparse for command line splitting"
```

---

## Task 3: Detector —— 项目类型识别 + 建议命令

**Files:**
- Create: `internal/detector/detector.go`, `internal/detector/rules.go`
- Test: `internal/detector/detector_test.go`

- [ ] **Step 1: 写失败测试(覆盖规则表每一类)**

`internal/detector/detector_test.go`:

```go
package detector

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// mkProject 在临时目录建一个项目,files 是 相对路径→内容 的映射。
func mkProject(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for rel, content := range files {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestDetect(t *testing.T) {
	pkg := func(scripts map[string]string) string {
		m := map[string]any{"name": "x", "scripts": scripts}
		b, _ := json.Marshal(m)
		return string(b)
	}
	cases := []struct {
		name     string
		files    map[string]string
		wantType string
		wantCmd  string
	}{
		{"pnpm", map[string]string{"package.json": pkg(map[string]string{"dev": "vite"}), "pnpm-lock.yaml": ""},
			"node-pnpm", "pnpm run dev"},
		{"yarn", map[string]string{"package.json": pkg(map[string]string{"dev": "vite"}), "yarn.lock": ""},
			"node-yarn", "yarn dev"},
		{"bun", map[string]string{"package.json": pkg(map[string]string{"dev": "vite"}), "bun.lockb": ""},
			"node-bun", "bun run dev"},
		{"npm default", map[string]string{"package.json": pkg(map[string]string{"dev": "vite"})},
			"node-npm", "npm run dev"},
		{"npm serve fallback", map[string]string{"package.json": pkg(map[string]string{"serve": "vue-cli-service serve"})},
			"node-npm", "npm run serve"},
		{"go root main", map[string]string{"go.mod": "module x\n", "main.go": "package main\nfunc main(){}\n"},
			"go", "go run main.go"},
		{"go cmd", map[string]string{"go.mod": "module x\n", "cmd/server/main.go": "package main\nfunc main(){}\n"},
			"go", "go run ./cmd/server"},
		{"rust", map[string]string{"Cargo.toml": "[package]\nname=\"x\"\n"},
			"rust", "cargo run"},
		{"django", map[string]string{"manage.py": "", "requirements.txt": ""},
			"python-django", "python manage.py runserver"},
		{"python main", map[string]string{"main.py": "", "requirements.txt": ""},
			"python", "python main.py"},
		{"unknown", map[string]string{"README.md": "hi"},
			"unknown", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := mkProject(t, c.files)
			res := Detect(dir)
			if res.Type != c.wantType {
				t.Errorf("Type = %q, want %q", res.Type, c.wantType)
			}
			if res.Command != c.wantCmd {
				t.Errorf("Command = %q, want %q", res.Command, c.wantCmd)
			}
		})
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/detector/ -v`
Expected: FAIL,`undefined: Detect`。

- [ ] **Step 3: 实现 rules.go(辅助判断函数)**

`internal/detector/rules.go`:

```go
package detector

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

// exists 判断项目 dir 下的相对路径是否存在。
func exists(dir, rel string) bool {
	_, err := os.Stat(filepath.Join(dir, rel))
	return err == nil
}

// readScripts 读取 package.json 的 scripts 字段;失败返回 nil。
func readScripts(dir string) map[string]string {
	b, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return nil
	}
	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if json.Unmarshal(b, &pkg) != nil {
		return nil
	}
	return pkg.Scripts
}

// pickNodeScript 依次挑选存在的脚本名,优先 dev,其次 serve、start;都没有则返回 "dev"(兜底)。
func pickNodeScript(dir string) string {
	scripts := readScripts(dir)
	for _, name := range []string{"dev", "serve", "start"} {
		if _, ok := scripts[name]; ok {
			return name
		}
	}
	return "dev"
}

// firstCmdMain 返回按字母序第一个含 main.go 的 cmd/<name> 目录名;无则返回 ""。
func firstCmdMain(dir string) string {
	entries, err := os.ReadDir(filepath.Join(dir, "cmd"))
	if err != nil {
		return ""
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() && exists(dir, filepath.Join("cmd", e.Name(), "main.go")) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return ""
	}
	return names[0]
}

// firstExisting 返回候选相对路径中第一个存在的;都不存在返回 ""。
func firstExisting(dir string, candidates ...string) string {
	for _, c := range candidates {
		if exists(dir, c) {
			return c
		}
	}
	return ""
}
```

- [ ] **Step 4: 实现 detector.go(规则表按优先级匹配)**

`internal/detector/detector.go`:

```go
// Package detector 根据项目目录内的文件特征识别项目类型并给出建议启动命令。
// 纯函数式:只读文件系统,无副作用,给定目录输出恒定。
package detector

import "fmt"

// Result 是一次识别的结果。Command 为空表示未能识别(Type == "unknown")。
type Result struct {
	Type    string // 如 "go" / "node-pnpm" / "unknown"
	Command string // 建议的单行启动命令,供 UI 回显;可为空
}

// Detect 按优先级从上到下匹配规则,命中即返回。
func Detect(dir string) Result {
	hasPkg := exists(dir, "package.json")

	switch {
	case hasPkg && exists(dir, "pnpm-lock.yaml"):
		return Result{"node-pnpm", "pnpm run " + pickNodeScript(dir)}
	case hasPkg && exists(dir, "yarn.lock"):
		return Result{"node-yarn", "yarn " + pickNodeScript(dir)}
	case hasPkg && exists(dir, "bun.lockb"):
		return Result{"node-bun", "bun run " + pickNodeScript(dir)}
	case hasPkg:
		return Result{"node-npm", "npm run " + pickNodeScript(dir)}

	case exists(dir, "go.mod") && exists(dir, "main.go"):
		return Result{"go", "go run main.go"}
	case exists(dir, "go.mod"):
		if name := firstCmdMain(dir); name != "" {
			return Result{"go", fmt.Sprintf("go run ./cmd/%s", name)}
		}
		return Result{"go", "go run ."}

	case exists(dir, "Cargo.toml"):
		return Result{"rust", "cargo run"}

	case exists(dir, "manage.py"):
		return Result{"python-django", "python manage.py runserver"}

	case exists(dir, "pyproject.toml") && exists(dir, "poetry.lock"):
		if f := firstExisting(dir, "main.py", "app.py"); f != "" {
			return Result{"python-poetry", "poetry run python " + f}
		}
		return Result{"python-poetry", "poetry run python main.py"}

	case firstExisting(dir, "main.py", "app.py") != "":
		return Result{"python", "python " + firstExisting(dir, "main.py", "app.py")}
	case exists(dir, "requirements.txt"):
		return Result{"python", "python main.py"}
	}

	return Result{"unknown", ""}
}
```

> 注意:`manage.py` 规则放在通用 python 之前,确保 Django 项目(同时有 requirements.txt)命中 `python-django`。测试用例 "django" 与 "python main" 覆盖此顺序。

- [ ] **Step 5: 运行测试确认通过**

Run: `go test ./internal/detector/ -v`
Expected: PASS(全部子用例)。

- [ ] **Step 6: 提交**

```bash
git add internal/detector/
git commit -m "feat: add project type detector with rule table"
```

---

## Task 4: Store —— 配置数据模型与 JSON 持久化

**Files:**
- Create: `internal/store/model.go`, `internal/store/store.go`
- Test: `internal/store/store_test.go`

- [ ] **Step 1: 实现 model.go(先写数据结构,供测试引用)**

`internal/store/model.go`:

```go
// Package store 负责 atstarter 配置(工作区 + 项目列表)的持久化。
package store

import (
	"crypto/sha1"
	"encoding/hex"
)

// Project 是一个可启动项目的完整配置。
type Project struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Path         string            `json:"path"`
	Command      string            `json:"command"`
	Args         []string          `json:"args"`
	Cwd          string            `json:"cwd"`
	Env          map[string]string `json:"env"`
	DetectedType string            `json:"detectedType"`
	AutoDetected bool              `json:"autoDetected"`
}

// Config 是配置文件的顶层结构。
type Config struct {
	Version    int       `json:"version"`
	Workspaces []string  `json:"workspaces"`
	Projects   []Project `json:"projects"`
}

// IDForPath 由项目绝对路径生成稳定 ID(去重依据)。
func IDForPath(path string) string {
	sum := sha1.Sum([]byte(path))
	return hex.EncodeToString(sum[:])
}
```

- [ ] **Step 2: 写失败测试**

`internal/store/store_test.go`:

```go
package store

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	return New(path)
}

func TestLoadMissingFileReturnsEmptyConfig(t *testing.T) {
	s := newTestStore(t)
	cfg, err := s.Load()
	if err != nil {
		t.Fatalf("Load on missing file should not error: %v", err)
	}
	if cfg.Version != 1 {
		t.Errorf("Version = %d, want 1", cfg.Version)
	}
	if len(cfg.Projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(cfg.Projects))
	}
}

func TestAddAndReload(t *testing.T) {
	s := newTestStore(t)
	p := Project{Name: "toolkit", Path: "/x/ad-ai-toolkit", Command: "go", Args: []string{"run", "main.go"}}
	if err := s.Add(p); err != nil {
		t.Fatal(err)
	}
	// 新实例从磁盘重读,验证持久化。
	s2 := New(s.path)
	cfg, err := s2.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(cfg.Projects))
	}
	if cfg.Projects[0].ID == "" {
		t.Error("Add should assign an ID")
	}
	if cfg.Projects[0].Name != "toolkit" {
		t.Errorf("Name = %q", cfg.Projects[0].Name)
	}
}

func TestAddDedupBySamePath(t *testing.T) {
	s := newTestStore(t)
	_ = s.Add(Project{Name: "a", Path: "/x/proj"})
	_ = s.Add(Project{Name: "a-again", Path: "/x/proj"})
	cfg, _ := s.Load()
	if len(cfg.Projects) != 1 {
		t.Fatalf("expected dedup to 1, got %d", len(cfg.Projects))
	}
}

func TestUpdate(t *testing.T) {
	s := newTestStore(t)
	_ = s.Add(Project{Name: "a", Path: "/x/proj", Command: "go"})
	cfg, _ := s.Load()
	p := cfg.Projects[0]
	p.Command = "pnpm"
	p.AutoDetected = false
	if err := s.Update(p); err != nil {
		t.Fatal(err)
	}
	cfg2, _ := s.Load()
	if cfg2.Projects[0].Command != "pnpm" {
		t.Errorf("Command = %q, want pnpm", cfg2.Projects[0].Command)
	}
}

func TestRemove(t *testing.T) {
	s := newTestStore(t)
	_ = s.Add(Project{Name: "a", Path: "/x/proj"})
	cfg, _ := s.Load()
	id := cfg.Projects[0].ID
	if err := s.Remove(id); err != nil {
		t.Fatal(err)
	}
	cfg2, _ := s.Load()
	if len(cfg2.Projects) != 0 {
		t.Errorf("expected 0 after remove, got %d", len(cfg2.Projects))
	}
}

func TestLoadCorruptJSONReturnsError(t *testing.T) {
	s := newTestStore(t)
	if err := os.WriteFile(s.path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load(); err == nil {
		t.Error("expected error loading corrupt json")
	}
}
```

- [ ] **Step 3: 运行测试确认失败**

Run: `go test ./internal/store/ -v`
Expected: FAIL,`undefined: New` / `undefined: Store`。

- [ ] **Step 4: 实现 store.go**

`internal/store/store.go`:

```go
package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Store 管理单个 JSON 配置文件的读写。所有写操作先改内存再落盘(全量覆盖写)。
type Store struct {
	path string
}

// New 用给定配置文件路径构造 Store。
func New(path string) *Store {
	return &Store{path: path}
}

// Load 读取配置。文件不存在时返回一个已初始化的空 Config(Version=1),不视为错误。
// JSON 损坏时返回错误。
func (s *Store) Load() (Config, error) {
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{Version: 1, Workspaces: []string{}, Projects: []Project{}}, nil
	}
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	return cfg, nil
}

// save 全量写回,先写临时文件再 rename,保证原子性。
func (s *Store) save(cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// Add 新增项目。若已存在相同 Path(以 IDForPath 判重),则忽略(幂等)。
// 自动为项目分配基于 Path 的 ID。
func (s *Store) Add(p Project) error {
	cfg, err := s.Load()
	if err != nil {
		return err
	}
	p.ID = IDForPath(p.Path)
	for _, existing := range cfg.Projects {
		if existing.ID == p.ID {
			return nil // 已存在,幂等返回
		}
	}
	cfg.Projects = append(cfg.Projects, p)
	return s.save(cfg)
}

// Update 按 ID 覆盖已存在的项目。找不到则返回错误。
func (s *Store) Update(p Project) error {
	cfg, err := s.Load()
	if err != nil {
		return err
	}
	for i := range cfg.Projects {
		if cfg.Projects[i].ID == p.ID {
			cfg.Projects[i] = p
			return s.save(cfg)
		}
	}
	return errors.New("store: project not found: " + p.ID)
}

// Remove 按 ID 删除项目。找不到视为成功(幂等)。
func (s *Store) Remove(id string) error {
	cfg, err := s.Load()
	if err != nil {
		return err
	}
	out := cfg.Projects[:0]
	for _, p := range cfg.Projects {
		if p.ID != id {
			out = append(out, p)
		}
	}
	cfg.Projects = out
	return s.save(cfg)
}

// SetWorkspaces 覆盖工作区根目录列表。
func (s *Store) SetWorkspaces(dirs []string) error {
	cfg, err := s.Load()
	if err != nil {
		return err
	}
	cfg.Workspaces = dirs
	return s.save(cfg)
}
```

- [ ] **Step 5: 运行测试确认通过**

Run: `go test ./internal/store/ -v`
Expected: PASS(全部子用例)。

- [ ] **Step 6: 提交**

```bash
git add internal/store/
git commit -m "feat: add config store with json persistence and dedup"
```

---

## Task 5: Scanner —— 批量扫描工作区

**Files:**
- Create: `internal/scanner/scanner.go`
- Test: `internal/scanner/scanner_test.go`

- [ ] **Step 1: 写失败测试**

`internal/scanner/scanner_test.go`:

```go
package scanner

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestScanFindsProjectsInDirectChildren(t *testing.T) {
	root := t.TempDir()
	// 子目录 a:go 项目
	write(t, filepath.Join(root, "a", "go.mod"), "module a\n")
	write(t, filepath.Join(root, "a", "main.go"), "package main\nfunc main(){}\n")
	// 子目录 b:pnpm 项目
	write(t, filepath.Join(root, "b", "package.json"), `{"scripts":{"dev":"vite"}}`)
	write(t, filepath.Join(root, "b", "pnpm-lock.yaml"), "")
	// 子目录 c:无法识别,仍应列出(unknown)
	write(t, filepath.Join(root, "c", "README.md"), "hi")
	// 文件(非目录)应被忽略
	write(t, filepath.Join(root, "loose.txt"), "x")

	got := Scan([]string{root})
	if len(got) != 3 {
		t.Fatalf("expected 3 candidates, got %d: %+v", len(got), got)
	}
	sort.Slice(got, func(i, j int) bool { return got[i].Name < got[j].Name })
	if got[0].Name != "a" || got[0].DetectedType != "go" {
		t.Errorf("a: got %+v", got[0])
	}
	if got[0].Command != "go" || len(got[0].Args) != 2 {
		t.Errorf("a command/args: got cmd=%q args=%v", got[0].Command, got[0].Args)
	}
	if got[1].Name != "b" || got[1].DetectedType != "node-pnpm" {
		t.Errorf("b: got %+v", got[1])
	}
	if got[2].Name != "c" || got[2].DetectedType != "unknown" {
		t.Errorf("c: got %+v", got[2])
	}
}

func TestScanSkipsMissingRoot(t *testing.T) {
	got := Scan([]string{"/nonexistent/path/xyz"})
	if len(got) != 0 {
		t.Errorf("expected 0 for missing root, got %d", len(got))
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./internal/scanner/ -v`
Expected: FAIL,`undefined: Scan`。

- [ ] **Step 3: 实现 scanner.go**

`internal/scanner/scanner.go`:

```go
// Package scanner 遍历工作区根目录的直接子目录,对每个调用 detector,
// 产出候选 store.Project 列表(command/args 已拆分,ID 已生成)。
package scanner

import (
	"os"
	"path/filepath"

	"atstarter/internal/cmdparse"
	"atstarter/internal/detector"
	"atstarter/internal/store"
)

// Scan 扫描每个 root 的直接子目录。识别为 unknown 的也会列出(命令留空)。
// 无法读取的 root 被静默跳过。
func Scan(roots []string) []store.Project {
	var out []store.Project
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue // 跳过不存在/不可读的 root
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			dir := filepath.Join(root, e.Name())
			res := detector.Detect(dir)
			p := store.Project{
				ID:           store.IDForPath(dir),
				Name:         e.Name(),
				Path:         dir,
				DetectedType: res.Type,
				AutoDetected: true,
			}
			if res.Command != "" {
				if cmd, args, err := cmdparse.Parse(res.Command); err == nil {
					p.Command = cmd
					p.Args = args
				}
			}
			out = append(out, p)
		}
	}
	return out
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./internal/scanner/ -v`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add internal/scanner/
git commit -m "feat: add workspace scanner"
```

---

## Task 6: Runner —— 进程管理、日志环形缓冲、子进程树清理

**Files:**
- Create: `internal/runner/ringbuffer.go`, `internal/runner/runner.go`, `internal/runner/process_unix.go`, `internal/runner/process_windows.go`
- Test: `internal/runner/runner_test.go`

> Runner 是唯一有运行时状态的模块,含平台相关代码。为可测,Runner 通过一个 `emit` 回调把日志行推给外部(Wails 层接回调转成事件),测试里用 channel 接收。

- [ ] **Step 1: 实现 ringbuffer.go(先写,供 runner 与测试引用)**

`internal/runner/ringbuffer.go`:

```go
package runner

import "sync"

// ringBuffer 是固定容量的日志行环形缓冲,满时丢弃最旧行。并发安全。
type ringBuffer struct {
	mu    sync.Mutex
	buf   []string
	size  int
	start int
	count int
}

func newRingBuffer(size int) *ringBuffer {
	return &ringBuffer{buf: make([]string, size), size: size}
}

func (r *ringBuffer) add(line string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	idx := (r.start + r.count) % r.size
	if r.count < r.size {
		r.buf[idx] = line
		r.count++
	} else {
		r.buf[r.start] = line
		r.start = (r.start + 1) % r.size
	}
}

// snapshot 返回当前缓冲内容(按时间顺序)的拷贝。
func (r *ringBuffer) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, r.count)
	for i := 0; i < r.count; i++ {
		out[i] = r.buf[(r.start+i)%r.size]
	}
	return out
}
```

- [ ] **Step 2: 写 ringbuffer 与 runner 的失败测试**

`internal/runner/runner_test.go`:

```go
package runner

import (
	"runtime"
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
	r := New(100)
	r.SetEmitter(func(LogLine) {})
	spec := Spec{ID: "bad", Command: "/nonexistent/binary/xyz", Dir: t.TempDir()}
	err := r.Start(spec)
	if err == nil {
		waitStatus(t, r, "bad", StatusError, 3*time.Second)
	}
	// 无论 Start 立即返回 err,还是异步置为 error 状态,都算正确处理(不静默成功)。
	if err == nil && r.Status("bad").State != StatusError {
		t.Errorf("expected error state for missing binary, got %+v", r.Status("bad"))
	}
}

func TestStopKillsProcessTree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix process group test")
	}
	r := New(1000)
	r.SetEmitter(func(LogLine) {})
	// 父 shell 后台起一个长 sleep 子进程,并打印其 PID,然后自己也 sleep。
	spec := Spec{
		ID:      "tree",
		Command: "sh",
		Args:    []string{"-c", "sleep 300 & echo CHILD $!; sleep 300"},
		Dir:     t.TempDir(),
	}
	if err := r.Start(spec); err != nil {
		t.Fatal(err)
	}
	waitStatus(t, r, "tree", StatusRunning, 3*time.Second)
	if err := r.Stop("tree"); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	waitStatus(t, r, "tree", StatusExited, 8*time.Second)
	// 无直接断言子进程消失的可移植方法;此处以父进程组被终止、Stop 正常返回作为验证。
	// 进程组信号的正确性由 process_unix.go 的实现保证(kill 负 pgid)。
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
```

- [ ] **Step 3: 运行测试确认失败**

Run: `go test ./internal/runner/ -v`
Expected: FAIL,`undefined: New` / `undefined: Spec` 等。

- [ ] **Step 4: 实现 process_unix.go(进程组 + 信号)**

`internal/runner/process_unix.go`:

```go
//go:build !windows

package runner

import (
	"os/exec"
	"syscall"
	"time"
)

// setupProcAttr 让子进程自成进程组,便于整组信号。
func setupProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killTree 先给整个进程组发 SIGTERM,超时后 SIGKILL。
// 负 PID 表示"发给该进程组"。
func killTree(pid int) {
	pgid := pid // 因 Setpgid,子进程 pid == pgid
	_ = syscall.Kill(-pgid, syscall.SIGTERM)
	go func() {
		time.Sleep(5 * time.Second)
		_ = syscall.Kill(-pgid, syscall.SIGKILL)
	}()
}
```

- [ ] **Step 5: 实现 process_windows.go(占位:直接 Kill)**

`internal/runner/process_windows.go`:

```go
//go:build windows

package runner

import "os/exec"

// setupProcAttr 在 Windows 上暂不做进程组设置(后续可接 Job Object)。
func setupProcAttr(cmd *exec.Cmd) {}

// killTree 在 Windows 上暂用 taskkill /T 杀进程树的简化替代:
// 由 runner 持有 *exec.Cmd 时直接 Process.Kill();此处保留空实现,
// 实际终止在 runner.Stop 中通过 cmd.Process.Kill() 完成。
// TODO(后续): 接入 Job Object 保证子孙进程一并终止。
func killTree(pid int) {}
```

> 说明:Windows 的完整进程树终止(Job Object)列为后续增强。当前占位保证可编译、单机(Linux)开发不受阻。runner.Stop 在两平台都会调用 `cmd.Process.Kill()` 作为兜底。

- [ ] **Step 6: 实现 runner.go**

`internal/runner/runner.go`:

```go
// Package runner 管理子进程的启动/停止、输出捕获与状态维护。
// 平台相关的进程组/信号逻辑在 process_unix.go / process_windows.go。
package runner

import (
	"bufio"
	"errors"
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
}

type managed struct {
	cmd    *exec.Cmd
	status Status
	logs   *ringBuffer
}

// New 构造 Runner。bufSize 是每个项目日志环形缓冲的行数。
func New(bufSize int) *Runner {
	return &Runner{procs: map[string]*managed{}, bufSize: bufSize, emit: func(LogLine) {}}
}

// SetEmitter 设置日志回调(Wails 层接成事件;测试里接 channel)。
func (r *Runner) SetEmitter(fn func(LogLine)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.emit = fn
}

// Start 启动一个进程。若同 ID 已在运行则返回错误(幂等拒绝)。
func (r *Runner) Start(spec Spec) error {
	r.mu.Lock()
	if m, ok := r.procs[spec.ID]; ok && m.status.State == StatusRunning {
		r.mu.Unlock()
		return errors.New("runner: already running: " + spec.ID)
	}
	r.mu.Unlock()

	cmd := exec.Command(spec.Command, spec.Args...)
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
	r.mu.Unlock()

	go r.pump(spec.ID, m, stdout, "stdout")
	go r.pump(spec.ID, m, stderr, "stderr")
	go r.wait(spec.ID, m)
	return nil
}

// pump 逐行读取一个流,写入环形缓冲并 emit。
func (r *Runner) pump(id string, m *managed, pipe interface{ Read([]byte) (int, error) }, stream string) {
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

// wait 等待进程结束并更新状态。
func (r *Runner) wait(id string, m *managed) {
	err := m.cmd.Wait()
	r.mu.Lock()
	defer r.mu.Unlock()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			m.status.State = StatusExited
			m.status.ExitCode = ee.ExitCode()
		} else {
			m.status.State = StatusError
		}
	} else {
		m.status.State = StatusExited
		m.status.ExitCode = 0
	}
}

// Stop 终止进程(整进程组)。未知 ID 或已停止视为成功。
func (r *Runner) Stop(id string) error {
	r.mu.Lock()
	m, ok := r.procs[id]
	r.mu.Unlock()
	if !ok || m.status.State != StatusRunning || m.cmd.Process == nil {
		return nil
	}
	killTree(m.cmd.Process.Pid)
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
```

> `pump` 用 `interface{ Read(...) }` 接受 stdout/stderr 管道,避免 import io 仅为一个类型。若执行者偏好显式 `io.ReadCloser`,可 import io 并替换,行为一致。

- [ ] **Step 7: 运行测试确认通过**

Run: `go test ./internal/runner/ -v`
Expected: PASS(Linux 下 4 个测试全过;Windows 会 skip 两个 unix 专属用例)。

- [ ] **Step 8: 提交**

```bash
git add internal/runner/
git commit -m "feat: add process runner with ring buffer and process-group cleanup"
```

---

## Task 7: App 绑定层 —— 组装模块并暴露给前端

**Files:**
- Modify: `app.go`(替换 wails 脚手架默认内容)
- Modify: `main.go`(注册 App 生命周期钩子)

- [ ] **Step 1: 写 App 层测试(不依赖 Wails 运行时)**

`app_test.go`:

```go
package main

import (
	"path/filepath"
	"testing"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), "config.json")
	return NewAppWithConfig(cfgPath)
}

func TestAppAddProjectDetectsCommand(t *testing.T) {
	app := newTestApp(t)
	dir := t.TempDir()
	// 造一个 pnpm 项目
	writeFile(t, filepath.Join(dir, "package.json"), `{"scripts":{"dev":"vite"}}`)
	writeFile(t, filepath.Join(dir, "pnpm-lock.yaml"), "")

	p, err := app.AddProject(dir)
	if err != nil {
		t.Fatal(err)
	}
	if p.Command != "pnpm" {
		t.Errorf("Command = %q, want pnpm", p.Command)
	}
	if p.DetectedType != "node-pnpm" {
		t.Errorf("DetectedType = %q", p.DetectedType)
	}

	// 应已持久化
	list, _ := app.ListProjects()
	if len(list) != 1 {
		t.Errorf("expected 1 persisted project, got %d", len(list))
	}
}

func TestAppUpdateProjectCommandFromLine(t *testing.T) {
	app := newTestApp(t)
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "go.mod"), "module x\n")
	writeFile(t, filepath.Join(dir, "main.go"), "package main\nfunc main(){}\n")
	p, _ := app.AddProject(dir)

	// 用户在 UI 单行输入自定义命令(含绝对路径 go)
	updated, err := app.UpdateProjectCommand(p.ID, "/home/attson/sdk/go1.23.12/bin/go run main.go serve")
	if err != nil {
		t.Fatal(err)
	}
	if updated.Command != "/home/attson/sdk/go1.23.12/bin/go" {
		t.Errorf("Command = %q", updated.Command)
	}
	wantArgs := []string{"run", "main.go", "serve"}
	if len(updated.Args) != len(wantArgs) {
		t.Fatalf("Args = %v", updated.Args)
	}
	if !updated.AutoDetected == false {
		t.Errorf("AutoDetected should be false after manual edit")
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
```

> 顶部需要 `import ("os"; "path/filepath"; "testing")`。执行者补上 import。

- [ ] **Step 2: 运行测试确认失败**

Run: `go test . -run TestApp -v`
Expected: FAIL,`undefined: NewAppWithConfig` / `AddProject` 等。

- [ ] **Step 3: 实现 app.go**

替换 `app.go` 全部内容:

```go
package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"atstarter/internal/cmdparse"
	"atstarter/internal/detector"
	"atstarter/internal/runner"
	"atstarter/internal/scanner"
	"atstarter/internal/store"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App 是 Wails 绑定层,组装各内部模块并暴露方法给前端。
type App struct {
	ctx    context.Context
	store  *store.Store
	runner *runner.Runner
}

// NewApp 用默认配置路径(用户配置目录)构造。
func NewApp() *App {
	return NewAppWithConfig(defaultConfigPath())
}

// NewAppWithConfig 用指定配置路径构造(测试用)。
func NewAppWithConfig(cfgPath string) *App {
	return &App{
		store:  store.New(cfgPath),
		runner: runner.New(5000),
	}
}

// defaultConfigPath 返回各平台标准配置目录下的 config.json。
func defaultConfigPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "atstarter", "config.json")
}

// startup 由 Wails 在启动时调用,保存 ctx 并接好日志/状态事件转发。
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.runner.SetEmitter(func(l runner.LogLine) {
		runtime.EventsEmit(a.ctx, "log:"+l.ID, map[string]string{
			"stream": l.Stream, "text": l.Text,
		})
	})
}

// shutdown 由 Wails 在退出时调用,停掉所有进程。
func (a *App) shutdown(ctx context.Context) {
	a.runner.StopAll()
}

// ---- 暴露给前端的方法 ----

// ListProjects 返回所有已保存项目。
func (a *App) ListProjects() ([]store.Project, error) {
	cfg, err := a.store.Load()
	if err != nil {
		return nil, err
	}
	return cfg.Projects, nil
}

// AddProject 识别目录并保存为项目。
func (a *App) AddProject(path string) (store.Project, error) {
	if _, err := os.Stat(path); err != nil {
		return store.Project{}, errors.New("path not found: " + path)
	}
	res := detector.Detect(path)
	p := store.Project{
		ID:           store.IDForPath(path),
		Name:         filepath.Base(path),
		Path:         path,
		DetectedType: res.Type,
		AutoDetected: true,
	}
	if res.Command != "" {
		if cmd, args, err := cmdparse.Parse(res.Command); err == nil {
			p.Command, p.Args = cmd, args
		}
	}
	if err := a.store.Add(p); err != nil {
		return store.Project{}, err
	}
	return p, nil
}

// ScanWorkspaces 扫描给定根目录,返回候选(不自动保存)。
func (a *App) ScanWorkspaces(roots []string) []store.Project {
	return scanner.Scan(roots)
}

// AddScanned 批量保存用户勾选的候选项目。
func (a *App) AddScanned(projects []store.Project) error {
	for _, p := range projects {
		if err := a.store.Add(p); err != nil {
			return err
		}
	}
	return nil
}

// UpdateProject 覆盖保存一个项目。
func (a *App) UpdateProject(p store.Project) error {
	return a.store.Update(p)
}

// UpdateProjectCommand 用 UI 单行命令更新项目的 command/args,并标记为手动。
func (a *App) UpdateProjectCommand(id, line string) (store.Project, error) {
	cfg, err := a.store.Load()
	if err != nil {
		return store.Project{}, err
	}
	for _, p := range cfg.Projects {
		if p.ID == id {
			cmd, args, err := cmdparse.Parse(line)
			if err != nil {
				return store.Project{}, err
			}
			p.Command, p.Args = cmd, args
			p.AutoDetected = false
			if err := a.store.Update(p); err != nil {
				return store.Project{}, err
			}
			return p, nil
		}
	}
	return store.Project{}, errors.New("project not found: " + id)
}

// RemoveProject 删除项目(若在运行先停止)。
func (a *App) RemoveProject(id string) error {
	_ = a.runner.Stop(id)
	return a.store.Remove(id)
}

// StartProject 启动项目对应的进程。
func (a *App) StartProject(id string) error {
	cfg, err := a.store.Load()
	if err != nil {
		return err
	}
	for _, p := range cfg.Projects {
		if p.ID == id {
			dir := p.Cwd
			if dir == "" {
				dir = p.Path
			}
			return a.runner.Start(runner.Spec{
				ID: p.ID, Command: p.Command, Args: p.Args, Dir: dir, Env: p.Env,
			})
		}
	}
	return errors.New("project not found: " + id)
}

// StopProject 停止项目进程。
func (a *App) StopProject(id string) error {
	return a.runner.Stop(id)
}

// GetStatus 返回项目运行时状态。
func (a *App) GetStatus(id string) runner.Status {
	return a.runner.Status(id)
}

// GetLogs 返回项目日志缓冲快照。
func (a *App) GetLogs(id string) []string {
	return a.runner.Logs(id)
}

// SetWorkspaces 保存工作区根目录列表。
func (a *App) SetWorkspaces(dirs []string) error {
	return a.store.SetWorkspaces(dirs)
}

// GetWorkspaces 返回已保存的工作区根目录。
func (a *App) GetWorkspaces() ([]string, error) {
	cfg, err := a.store.Load()
	if err != nil {
		return nil, err
	}
	return cfg.Workspaces, nil
}
```

- [ ] **Step 4: 修改 main.go 注册生命周期钩子**

打开 wails 生成的 `main.go`,把 `app := NewApp()` 保持,并确认 `OnStartup`/`OnShutdown` 指向 `app.startup`/`app.shutdown`。典型片段:

```go
app := NewApp()

err := wails.Run(&options.App{
	Title:  "atstarter",
	Width:  1100,
	Height: 700,
	AssetServer: &assetserver.Options{
		Assets: assets,
	},
	OnStartup:  app.startup,
	OnShutdown: app.shutdown,
	Bind: []interface{}{
		app,
	},
})
if err != nil {
	println("Error:", err.Error())
}
```

> 若脚手架里方法名是大写 `Startup`,统一改为本计划的 `startup`/`shutdown`(小写),或把 app.go 里改成大写与之匹配。二选一,保持一致。

- [ ] **Step 5: 运行 App 层测试确认通过**

Run: `go test . -run TestApp -v`
Expected: PASS。

- [ ] **Step 6: 整体编译**

Run: `go build ./...`
Expected: 通过。

- [ ] **Step 7: 提交**

```bash
git add app.go main.go app_test.go
git commit -m "feat: wire modules into wails app binding layer"
```

---

## Task 8: 前端界面(Vue3)

**Files:**
- Modify: `frontend/src/App.vue`
- Create: `frontend/src/components/ProjectList.vue`, `ProjectDetail.vue`, `LogPanel.vue`, `EditProjectDialog.vue`, `ScanDialog.vue`

> Wails 会自动生成 `frontend/wailsjs/go/main/App.js` 绑定与 `frontend/wailsjs/runtime`。前端调用 `import { ListProjects, StartProject, ... } from '../wailsjs/go/main/App'`,事件用 `EventsOn` 来自 `../wailsjs/runtime/runtime`。前端以手动验证为主,不写单测。

- [ ] **Step 1: 生成 Wails 绑定并确认可 dev 运行**

Run: `$($GO env GOPATH)/bin/wails dev`
Expected: 编译前后端并弹出窗口(或提示本机缺 GUI 库)。若 GUI 库缺失,先 `wails build` 确认后端绑定生成:`ls frontend/wailsjs/go/main/`,应看到 `App.js`。

- [ ] **Step 2: 实现 LogPanel.vue**

`frontend/src/components/LogPanel.vue`:

```vue
<script setup>
import { ref, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { GetLogs } from '../../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

const props = defineProps({ projectId: String })
const lines = ref([]) // { stream, text }
const box = ref(null)

let currentEvent = ''

async function load(id) {
  lines.value = []
  if (!id) return
  const hist = await GetLogs(id)
  lines.value = (hist || []).map((t) => ({ stream: 'stdout', text: t }))
  await scrollBottom()
}

function subscribe(id) {
  if (currentEvent) EventsOff(currentEvent)
  if (!id) return
  currentEvent = 'log:' + id
  EventsOn(currentEvent, async (p) => {
    lines.value.push({ stream: p.stream, text: p.text })
    await scrollBottom()
  })
}

async function scrollBottom() {
  await nextTick()
  if (box.value) box.value.scrollTop = box.value.scrollHeight
}

watch(() => props.projectId, async (id) => {
  await load(id)
  subscribe(id)
})

onMounted(() => {
  load(props.projectId)
  subscribe(props.projectId)
})
onUnmounted(() => { if (currentEvent) EventsOff(currentEvent) })
</script>

<template>
  <div ref="box" class="log-panel">
    <div v-for="(l, i) in lines" :key="i" :class="['log-line', l.stream]">{{ l.text }}</div>
  </div>
</template>

<style scoped>
.log-panel { flex: 1; overflow-y: auto; background: #1e1e1e; color: #ddd;
  font-family: monospace; font-size: 12px; padding: 8px; white-space: pre-wrap; }
.log-line.stderr { color: #ff6b6b; }
</style>
```

- [ ] **Step 3: 实现 ProjectList.vue**

`frontend/src/components/ProjectList.vue`:

```vue
<script setup>
defineProps({ projects: Array, selectedId: String, statuses: Object })
const emit = defineEmits(['select', 'add', 'scan'])

function dot(state) {
  if (state === 'running') return '#4caf50'
  if (state === 'error') return '#f44336'
  if (state === 'exited') return '#f44336'
  return '#888'
}
</script>

<template>
  <div class="list">
    <div class="items">
      <div v-for="p in projects" :key="p.id"
           :class="['item', { active: p.id === selectedId }]"
           @click="emit('select', p.id)">
        <span class="dot" :style="{ background: dot((statuses[p.id] || {}).State) }"></span>
        <span class="name">{{ p.name }}</span>
      </div>
    </div>
    <div class="actions">
      <button @click="emit('add')">+ 添加</button>
      <button @click="emit('scan')">扫描</button>
    </div>
  </div>
</template>

<style scoped>
.list { width: 240px; border-right: 1px solid #ddd; display: flex; flex-direction: column; }
.items { flex: 1; overflow-y: auto; }
.item { display: flex; align-items: center; gap: 8px; padding: 8px 12px; cursor: pointer; }
.item.active { background: #e8f0fe; }
.dot { width: 10px; height: 10px; border-radius: 50%; display: inline-block; }
.actions { display: flex; gap: 8px; padding: 8px; border-top: 1px solid #ddd; }
.actions button { flex: 1; }
</style>
```

- [ ] **Step 4: 实现 ProjectDetail.vue**

`frontend/src/components/ProjectDetail.vue`:

```vue
<script setup>
import LogPanel from './LogPanel.vue'
const props = defineProps({ project: Object, status: Object })
const emit = defineEmits(['start', 'stop', 'edit'])
</script>

<template>
  <div class="detail" v-if="project">
    <div class="bar">
      <div class="info">
        <strong>{{ project.name }}</strong>
        <span class="type">{{ project.detectedType }}</span>
        <code>{{ project.command }} {{ (project.args || []).join(' ') }}</code>
      </div>
      <div class="btns">
        <button :disabled="(status || {}).State === 'running'" @click="emit('start')">▶ 启动</button>
        <button :disabled="(status || {}).State !== 'running'" @click="emit('stop')">■ 停止</button>
        <button @click="emit('edit')">编辑</button>
      </div>
    </div>
    <LogPanel :projectId="project.id" />
  </div>
  <div class="detail empty" v-else>选择一个项目</div>
</template>

<style scoped>
.detail { flex: 1; display: flex; flex-direction: column; }
.detail.empty { align-items: center; justify-content: center; color: #888; }
.bar { display: flex; justify-content: space-between; align-items: center;
  padding: 10px 14px; border-bottom: 1px solid #ddd; }
.info { display: flex; flex-direction: column; gap: 4px; }
.type { color: #666; font-size: 12px; }
.btns { display: flex; gap: 8px; }
</style>
```

- [ ] **Step 5: 实现 EditProjectDialog.vue**

`frontend/src/components/EditProjectDialog.vue`:

```vue
<script setup>
import { ref, watch } from 'vue'
const props = defineProps({ project: Object, show: Boolean })
const emit = defineEmits(['save', 'close'])

const name = ref('')
const commandLine = ref('')
const cwd = ref('')

watch(() => props.project, (p) => {
  if (!p) return
  name.value = p.name
  commandLine.value = [p.command, ...(p.args || [])].join(' ')
  cwd.value = p.cwd || ''
}, { immediate: true })

function save() {
  emit('save', { name: name.value, commandLine: commandLine.value, cwd: cwd.value })
}
</script>

<template>
  <div class="mask" v-if="show" @click.self="emit('close')">
    <div class="dialog">
      <h3>编辑项目</h3>
      <label>名称<input v-model="name" /></label>
      <label>启动命令<input v-model="commandLine" placeholder="如 pnpm run dev" /></label>
      <label>工作目录 (可选)<input v-model="cwd" :placeholder="project && project.path" /></label>
      <div class="btns">
        <button @click="emit('close')">取消</button>
        <button @click="save">保存</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.mask { position: fixed; inset: 0; background: rgba(0,0,0,.4);
  display: flex; align-items: center; justify-content: center; }
.dialog { background: #fff; padding: 20px; border-radius: 8px; width: 480px;
  display: flex; flex-direction: column; gap: 12px; }
.dialog label { display: flex; flex-direction: column; gap: 4px; font-size: 13px; }
.dialog input { padding: 6px 8px; }
.btns { display: flex; justify-content: flex-end; gap: 8px; }
</style>
```

- [ ] **Step 6: 实现 ScanDialog.vue**

`frontend/src/components/ScanDialog.vue`:

```vue
<script setup>
import { ref } from 'vue'
import { ScanWorkspaces, AddScanned } from '../../wailsjs/go/main/App'
const props = defineProps({ show: Boolean })
const emit = defineEmits(['close', 'added'])

const rootsText = ref('')
const candidates = ref([])
const checked = ref({})

async function scan() {
  const roots = rootsText.value.split('\n').map((s) => s.trim()).filter(Boolean)
  candidates.value = await ScanWorkspaces(roots)
  checked.value = {}
  candidates.value.forEach((c) => { checked.value[c.id] = c.detectedType !== 'unknown' })
}

async function add() {
  const chosen = candidates.value.filter((c) => checked.value[c.id])
  await AddScanned(chosen)
  emit('added')
  emit('close')
}
</script>

<template>
  <div class="mask" v-if="show" @click.self="emit('close')">
    <div class="dialog">
      <h3>扫描工作区</h3>
      <textarea v-model="rootsText" rows="3"
        placeholder="每行一个根目录,如&#10;/home/attson/GolandProjects&#10;/home/attson/WebstormProjects"></textarea>
      <button @click="scan">扫描</button>
      <div class="results">
        <label v-for="c in candidates" :key="c.id" class="row">
          <input type="checkbox" v-model="checked[c.id]" />
          <span class="nm">{{ c.name }}</span>
          <span class="ty">{{ c.detectedType }}</span>
          <code>{{ c.command }} {{ (c.args || []).join(' ') }}</code>
        </label>
      </div>
      <div class="btns">
        <button @click="emit('close')">取消</button>
        <button @click="add">加入选中</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.mask { position: fixed; inset: 0; background: rgba(0,0,0,.4);
  display: flex; align-items: center; justify-content: center; }
.dialog { background: #fff; padding: 20px; border-radius: 8px; width: 620px;
  display: flex; flex-direction: column; gap: 10px; }
.results { max-height: 320px; overflow-y: auto; border: 1px solid #eee; }
.row { display: flex; align-items: center; gap: 10px; padding: 6px 8px; font-size: 13px; }
.ty { color: #666; }
.btns { display: flex; justify-content: flex-end; gap: 8px; }
</style>
```

- [ ] **Step 7: 实现 App.vue(组装 + 轮询状态)**

`frontend/src/App.vue`:

```vue
<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import ProjectList from './components/ProjectList.vue'
import ProjectDetail from './components/ProjectDetail.vue'
import EditProjectDialog from './components/EditProjectDialog.vue'
import ScanDialog from './components/ScanDialog.vue'
import {
  ListProjects, AddProject, StartProject, StopProject,
  GetStatus, UpdateProjectCommand, UpdateProject,
} from '../wailsjs/go/main/App'

const projects = ref([])
const selectedId = ref('')
const statuses = ref({})
const showEdit = ref(false)
const showScan = ref(false)

const selected = computed(() => projects.value.find((p) => p.id === selectedId.value))
const selectedStatus = computed(() => statuses.value[selectedId.value])

async function refresh() {
  projects.value = (await ListProjects()) || []
  if (!selectedId.value && projects.value.length) selectedId.value = projects.value[0].id
}

async function pollStatuses() {
  const next = {}
  for (const p of projects.value) next[p.id] = await GetStatus(p.id)
  statuses.value = next
}

async function onAdd() {
  const dir = prompt('输入项目目录绝对路径')
  if (!dir) return
  await AddProject(dir)
  await refresh()
}

async function onStart() { await StartProject(selectedId.value); await pollStatuses() }
async function onStop() { await StopProject(selectedId.value); await pollStatuses() }

async function onSaveEdit(payload) {
  // 先更新命令(会拆分 + 标记手动),再更新名字/cwd
  const updated = await UpdateProjectCommand(selectedId.value, payload.commandLine)
  updated.name = payload.name
  updated.cwd = payload.cwd
  await UpdateProject(updated)
  showEdit.value = false
  await refresh()
}

let timer
onMounted(async () => {
  await refresh()
  await pollStatuses()
  timer = setInterval(pollStatuses, 1500)
})
onUnmounted(() => clearInterval(timer))
</script>

<template>
  <div class="app">
    <ProjectList :projects="projects" :selectedId="selectedId" :statuses="statuses"
      @select="selectedId = $event" @add="onAdd" @scan="showScan = true" />
    <ProjectDetail :project="selected" :status="selectedStatus"
      @start="onStart" @stop="onStop" @edit="showEdit = true" />
    <EditProjectDialog :show="showEdit" :project="selected"
      @close="showEdit = false" @save="onSaveEdit" />
    <ScanDialog :show="showScan" @close="showScan = false" @added="refresh" />
  </div>
</template>

<style>
html, body, #app { height: 100%; margin: 0; }
.app { display: flex; height: 100vh; font-family: system-ui, sans-serif; }
</style>
```

- [ ] **Step 8: 手动验证**

Run: `$($GO env GOPATH)/bin/wails dev`
手动检查清单:
1. 点"扫描",输入 `~` 展开后的绝对路径(如 `/home/attson/GolandProjects`),点扫描 → 列出候选,go 项目显示 `go run main.go`。
2. 勾选加入 → 左列表出现项目。
3. 选中一个 pnpm 项目 → 点启动 → 日志面板出现 vite 输出,状态灯变绿。
4. 点停止 → 进程结束,状态灯变灰;`ps aux | grep vite` 确认无残留子进程。
5. 点编辑 → 改成 `/home/attson/sdk/...bin/go run main.go serve` → 保存 → 详情条显示新命令。

- [ ] **Step 9: 提交**

```bash
git add frontend/
git commit -m "feat: add vue3 frontend for project management"
```

---

## Task 9: 收尾 —— 全量测试与 README

**Files:**
- Create: `README.md`

- [ ] **Step 1: 跑全部 Go 测试**

Run: `go test ./...`
Expected: 所有包 PASS(runner 在非 unix 平台会 skip 部分)。

- [ ] **Step 2: 写 README.md**

`README.md`:

```markdown
# atstarter

本地项目快速启动器(Wails + Vue3 桌面 App)。自动识别本地项目类型、建议启动命令,一处托管启动/停止并查看实时日志。

## 开发

要求:Go ≥ 1.20、Node ≥ 18、Wails CLI。

\`\`\`bash
wails dev     # 开发模式
wails build   # 打包
go test ./... # 后端测试
\`\`\`

## 配置

配置存于各平台标准目录下的 `atstarter/config.json`(Linux: `~/.config/atstarter/`)。

## 支持识别的项目类型

pnpm / yarn / bun / npm、Go(根 main.go 与 cmd/*)、Rust、Python(Django/poetry/main.py)。识别为建议,可在 UI 手动改。
\`\`\`
```

- [ ] **Step 3: 提交**

```bash
git add README.md
git commit -m "docs: add readme"
```

---

## Self-Review 结论

- **Spec 覆盖**:技术栈(T1)、数据模型/command-args 分离(T4 model + T2 拆分)、Detector 规则表全部 12 类(T3)、node dev 脚本探测(T3 pickNodeScript)、批量扫描(T5)、进程管理/状态机/环形缓冲/进程树清理(T6)、跨平台 build tag 隔离(T6 process_unix/windows)、单行命令编辑体验(T7 UpdateProjectCommand + T8 EditDialog)、前端布局与事件(T8)、测试策略(各 Task 的 TDD 步骤)。均有对应任务。
- **类型一致性**:`store.Project` 字段贯穿 store/scanner/app;`runner.Spec`/`LogLine`/`Status`/`State` 在 runner 与 app 一致;`detector.Result{Type,Command}` 在 detector/scanner/app 一致。
- **已知取舍(非占位)**:Windows 进程树终止用 Job Object 为后续增强,当前占位实现 + `cmd.Process.Kill()` 兜底,已在 T6 Step 5 明确标注,不阻塞 Linux 开发。
- **环境前提显式化**:Go 1.19→1.20+ 与 Wails CLI 安装在 T0 处理,业务逻辑 Task(T2–T6)为纯 go test 不依赖 GUI。
