# AGENTS.md — atstarter

本地项目快速启动器(Wails v2 + Vue3 桌面 App)。给 AI 助手 / 新贡献者的项目速览与硬约束。

## 这个项目是什么

读取本地目录,自动识别项目类型并建议启动命令(pnpm/go/cargo/python…),在一处托管启动/停止多个项目、查看实时日志。支持批量扫描工作区、原生文件夹选择、每项目自定义命令(含指定运行时路径,如 `~/sdk/go1.24.13/bin/go run main.go serve`)。

## 硬约束(必须遵守)

| 约束 | 说明 |
|---|---|
| **Go 版本** | 系统默认 `go` 是 1.19(太旧)。**必须**用 `/home/attson/sdk/go1.24.13/bin/go`。所有 go 命令前 `export GO=/home/attson/sdk/go1.24.13/bin/go` 并用 `$GO`。 |
| **webkit tag** | Ubuntu 24.04 只有 `libwebkit2gtk-4.1-dev`,Wails 2.12 默认链 4.0。所有 wails 构建命令加 `-tags webkit2_41`(如 `wails dev -tags webkit2_41`)。纯 `go test`/业务逻辑包不需要。 |
| **commit 署名** | commit message **不要**加 `Co-Authored-By` 尾注。 |
| **runner 并发** | 改 `internal/runner` 的并发路径后**必须**跑 `$GO test -race ./internal/runner/`。所有 `m.status` 读写在 `r.mu` 锁内;回调(emit/onStatus)在锁外调用并传值拷贝;慢操作(killTree)锁外做。 |
| **TDD** | 业务逻辑改动先写失败测试再实现。runner/store/detector/cmdparse/scanner 都是纯 `go test`,不依赖 GUI。 |

## 架构

```
main.go / app.go        Wails 入口 + 绑定层(组装模块,暴露 14 个方法给前端)
internal/
  cmdparse/             单行命令 ↔ command+args 拆分(github.com/google/shlex)
  detector/             按文件特征识别项目类型 + 建议命令(纯函数,规则表)
  scanner/              遍历工作区直接子目录调 detector,产出候选
  store/                配置 JSON 持久化 + 路径去重(sha1(path) 生成 ID)
  runner/               子进程启停 + 环形缓冲日志 + 进程组清理(build tag 分平台)
frontend/src/           Vue3(App.vue + 5 个组件)
```

**数据流**:`detector.Detect(dir) → Result{Type,Command}` → `cmdparse.Parse` 拆成 command+args → `store.Project` → `runner.Spec` 启动。

**关键契约**:
- `store.IDForPath(path)` = sha1(path) hex,是去重依据。`Store.Add` 幂等(同路径不重复)。路径入库前经 `normalizePath`(展开 `~` + `filepath.Abs`)。
- `runner` 通过 `SetEmitter`(日志)和 `SetStatusListener`(状态)两个回调把事件推给 app 层,app 转成 Wails 事件 `log:<id>` / `status:<id>`,前端订阅。
- 进程退出时 runner 向日志追加 `[process exited with code N]` 尾行。

## 前端 ↔ 后端

- 前端调用:`import { ListProjects, StartProject, ... } from '../wailsjs/go/main/App'`(组件里是 `../../wailsjs/...`)。
- 事件:`import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'`;`log:<id>` payload `{stream,text}`;`status:<id>` payload `{state,pid,exitCode}`(小写)。
- **改了 app.go 的方法签名后**,需重新生成绑定:`$($GO env GOPATH)/bin/wails generate module`,前端才能 import 到新方法。
- 前端 statuses map 内部用大写 `State/PID/ExitCode`(与轮询 `GetStatus` 返回一致);事件 payload 小写,回调里做映射。

## 常用命令

```bash
export GO=/home/attson/sdk/go1.24.13/bin/go
$GO test ./...                      # 全量后端测试
$GO test -race ./internal/runner/   # runner 并发检查(改并发必跑)
$($GO env GOPATH)/bin/wails dev -tags webkit2_41     # 开发(热重载)
$($GO env GOPATH)/bin/wails build -tags webkit2_41   # 打包
cd frontend && npm run build        # 仅前端构建(不需 GUI)
```

## 调试 tips(踩过的坑)

- **"点启动后没日志" 多半不是 bug**:①`go run` 有静默编译期(依赖多的项目要几秒~几十秒,期间无输出,status=running 但日志空是正常的);②框架项目(gamesh)需要子命令,如 ad-ai-toolkit 要 `go run main.go serve`,不带 serve 只打印帮助后 exit 0 秒退。detector 无从知道业务子命令,靠用户手动编辑命令兜底。
- **排查后端状态**:`wails dev` 会起 devserver 在 `http://localhost:34115`,可用浏览器直接调 `window.go.main.App.GetStatus(id)` / `GetLogs(id)` 看真实后端数据。
- **App 继承启动 shell 的 PATH**:从 go1.19 的 shell 启动 App,跑 go1.23+ 项目会编译失败。确保启动 App 的 shell 里 `go` 是 1.23+。

## 配置文件

各平台标准配置目录下的 `atstarter/config.json`(Linux `~/.config/atstarter/config.json`)。结构:`{version, workspaces[], projects[]}`,project 含 `id/name/path/command/args/cwd/env/detectedType/autoDetected`。
