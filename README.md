# atstarter

本地项目快速启动器(Wails + Vue3 桌面 App)。读取本地目录代码,自动识别项目类型并建议启动命令,一处托管启动/停止多个项目并查看实时日志。

支持自定义每个项目的启动命令,包括指定运行时路径,如 `~/sdk/go1.24.13/bin/go run main.go serve`。

## 功能

- **批量扫描工作区**:指定工作区根目录(支持 `~`,如 `~/GolandProjects`),扫描其下直接子目录,识别项目类型并勾选批量加入。支持「📁 选择文件夹」调起系统原生目录选择器,选中后自动扫描。
- **自动识别 + 手动兜底**:内置规则识别项目类型并给出建议启动命令;识别结果可在编辑弹窗用单行输入框自由修改(自动拆分成 command + args 存储)。
- **进程托管**:App 内启动/停止项目子进程,实时展示 stdout/stderr 日志。日志面板顶部有生命周期状态横幅(运行中 / 已退出+退出码 / 错误),进程退出时日志尾部追加 `[process exited with code N]` 标记。
- **子进程树清理**:停止时清理整个进程组(如 `pnpm run dev` fork 出的 node/vite),不留孤儿进程(Linux/macOS)。

## 支持识别的项目类型

pnpm / yarn / bun / npm(node 项目,自动探测 dev/serve/start 脚本)、Go(根 `main.go` 及 `cmd/*/main.go`)、Rust(cargo)、Python(Django / poetry / main.py)。识别结果为建议,可手动修改。

## 使用说明

1. **添加项目**:点「扫描」输入工作区根目录(或用「📁 选择文件夹」),勾选识别到的项目加入;或点「+ 添加」输入单个项目路径。
2. **启动**:选中项目 → 点「▶ 启动」。注意 `go run` 有编译期(依赖多的项目需等待,此时日志面板显示"编译/启动中")。
3. **自定义命令**:点「编辑」,在单行输入框改成需要的命令。例如框架项目常需子命令:`go run main.go serve`;或指定 go 版本:`~/sdk/go1.24.13/bin/go run main.go serve`。

## 开发

**要求:** Go ≥ 1.20、Node ≥ 18、Wails CLI v2。

> **Ubuntu 24.04 注意:** 系统只提供 `libwebkit2gtk-4.1-dev`,而 Wails 2.12 默认链接 4.0。所有 wails 构建命令需加 `-tags webkit2_41`。

```bash
# 开发模式(热重载)
wails dev -tags webkit2_41

# 打包
wails build -tags webkit2_41

# 后端测试
go test ./...

# runner 并发检查(改进程管理相关代码后)
go test -race ./internal/runner/
```

更多面向贡献者/AI 的架构说明与硬约束见 [AGENTS.md](AGENTS.md)。

## 架构

Go 后端分四个单一职责模块:

- `internal/detector` — 按文件特征识别项目类型 + 建议命令(纯函数)
- `internal/scanner` — 遍历工作区调用 detector,产出候选项目
- `internal/store` — 配置持久化(JSON,增删改查 + 路径去重)
- `internal/runner` — 子进程启停、日志环形缓冲、进程树清理、退出标记

`internal/cmdparse` 负责单行命令 ↔ command+args 的转换。`app.go` 是 Wails 绑定层,组装以上模块并暴露方法给 Vue3 前端;日志/状态通过 Wails 事件推送给前端。

设计文档见 [docs/superpowers/specs/](docs/superpowers/specs/)。

## 配置

配置存于各平台标准配置目录下的 `atstarter/config.json`:

- Linux:`~/.config/atstarter/config.json`
- macOS:`~/Library/Application Support/atstarter/config.json`
- Windows:`%AppData%\atstarter\config.json`

## 已知限制

- **Windows 进程树终止**:当前用 `cmd.Process.Kill()` 兜底,完整的 Job Object 支持(确保子孙进程一并终止)待后续。Linux/macOS 用进程组信号已完整支持。
