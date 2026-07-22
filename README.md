# atstarter

本地项目快速启动器(Wails v2 + Vue3 桌面 App)。读取本地目录代码,自动识别项目类型并建议启动命令,一处托管启动/停止多个项目并查看实时日志。

支持自定义每个项目的启动命令,包括指定运行时路径,如 `~/sdk/go1.24.13/bin/go run main.go serve`。

## 功能

- **批量扫描工作区**:指定工作区根目录(支持 `~`,如 `~/GolandProjects`),扫描其下直接子目录(含 `.worktrees/`、`.claude/worktrees/`),识别项目类型并勾选批量加入。支持「📁 选择文件夹」调起系统原生目录选择器,选中后自动扫描。
- **自动识别 + 手动兜底**:内置规则识别项目类型并给出建议启动命令;识别结果可在编辑弹窗用单行输入框自由修改(自动拆分成 command + args 存储)。
- **多套启动命令**:每个项目可保存多条命令(default / debug / …),按需切换启动。
- **启动分组**:把多个「项目 + 命令」编成一组(如「前端 + 后端」),一键批量启停。
- **进程托管**:App 内启动/停止项目子进程,实时展示 stdout/stderr 日志。日志面板顶部有生命周期状态横幅(运行中 / 已退出+退出码 / 错误),进程退出时日志尾部追加 `[process exited with code N]` 标记。
- **Docker / docker compose 管理**:
  - **compose 项目**融入项目树:识别 `docker-compose.yml` / `compose.yaml` 等目录,支持整体 Up / Stop / Down 与**单 service** 的启停/重启/日志,聚合状态显示(如 `2/3 running`)。
  - **独立容器面板**:顶部 `Containers` Tab 展示宿主机 `docker ps -a` 快照,支持 start / stop / restart / remove / 实时日志,按 compose 归属分组。
  - detached + 2s 轮询(容器独立于 App 存活);Docker 不可用时优雅降级(给出原因 + 重试);删除类操作(compose `down` / 容器 `remove`)二次确认。
- **登录 shell 启动**:子进程经用户登录交互式 shell(`$SHELL -l -i -c`)启动,拿到完整 PATH —— 修复从桌面/IDE 启动 GUI 时 `pnpm`、`nvm`、`go` 等 `command not found`。
- **子进程树清理**:停止时用 setsid 进程组信号杀掉整棵子进程树(如 `pnpm dev` → node → vite → esbuild),不留孤儿占端口(Linux/macOS)。
- **系统托盘**:关闭窗口即隐藏到托盘(不退出),托盘菜单显示运行数、可显示/隐藏窗口、一键停全部、退出。
- **自更新**:轮询 GitHub Release 检查新版,下载后经 Ed25519 签名 + SHA256 校验再自安装。内置**下载加速镜像**(ghfast.top / gh-proxy.com / ghproxy.net),逐个尝试并自动回退到 github.com 原始地址,解决国内直连下载卡在 0% 的问题。
- **明暗主题**:内置浅色/深色主题切换。

## 支持识别的项目类型

docker compose(`docker-compose.yml` / `compose.yaml` 等,优先识别)、pnpm / yarn / bun / npm(node 项目,自动探测 dev/serve/start 脚本)、Go(根 `main.go` 及 `cmd/*/main.go`)、Rust(cargo)、Python(Django / poetry / main.py)。识别结果为建议,可手动修改(compose 项目走专用的 service 级控制,不用单行命令)。

## 使用说明

1. **添加项目**:点「扫描」输入工作区根目录(或用「📁 选择文件夹」),勾选识别到的项目加入;或点「+ 添加」输入单个项目路径。
2. **启动**:选中项目 → 点「▶ 启动」。注意 `go run` 有编译期(依赖多的项目需等待,此时日志面板显示"编译/启动中")。
3. **自定义命令**:点「编辑」,在单行输入框改成需要的命令。例如框架项目常需子命令:`go run main.go serve`;或指定 go 版本:`~/sdk/go1.24.13/bin/go run main.go serve`。
4. **分组**:把常一起启动的项目加入一个分组,在分组详情里一键启停全组。
5. **Docker**:含 compose 文件的目录会识别为 compose 项目,在详情里整体 Up/Down 或单独启停某个 service。切到顶部「Containers」标签管理宿主机上的独立容器(需本机装 Docker 且 daemon 运行;不可用时面板会提示原因)。

## 开发

**要求:** Go ≥ 1.23、Node ≥ 20、Wails CLI v2.12。

> **Ubuntu 24.04 注意:** 系统只提供 `libwebkit2gtk-4.1-dev`,而 Wails 2.12 默认链接 4.0。所有 wails 构建命令需加 `-tags webkit2_41`。系统托盘还需 `libayatana-appindicator3-dev`。

常用命令通过 `Makefile` 暴露(自动带上 `-tags webkit2_41`):

```bash
make dev          # 热重载
make build        # 本平台打包
make test         # go test + 前端 node --test
make test-race    # runner 并发检查

# 手动等价命令
wails dev -tags webkit2_41
wails build -tags webkit2_41
go test ./...
go test -race ./internal/runner/
```

## 多平台发布

`.github/workflows/build.yml` 在打 tag(`v*`)时构建并发布产物到 GitHub Release:

| 平台 | 产物 |
|---|---|
| linux/amd64、arm64 | `.deb`(deb 包)+ `.tar.gz`(裸二进制) |
| darwin/arm64、amd64 | `.dmg`(带 /Applications 拖拽安装)+ `.zip`(app 打包) |
| windows/amd64 | NSIS `.exe` 安装器 + `.zip`(裸 exe) |

每个 Release 附带 `SHA256SUMS` 及其 Ed25519 签名 `SHA256SUMS.sig`,供 App 自更新校验。

发布流程(**从 main 发版**,main 有直接 push 保护,先走 PR 合并):

```bash
git tag -a v0.4.0 -m "…"
git push origin v0.4.0
# → CI 自动构建 5 组产物、签名 checksums、生成 Release、附件上传
```

版本号通过 `-ldflags "-X main.Version=$TAG"` 打到二进制;自更新校验公钥通过 `-X main.UpdateVerifyPublicKey=<base64>` 注入(dev 构建无公钥,只能检查不能自安装)。

本地跨平台打包:

```bash
make build-linux
make build-darwin-arm64
make build-darwin-amd64
make build-windows        # 需 NSIS,推荐直接在 Windows 上跑
```

更多面向贡献者/AI 的架构说明与硬约束见 [AGENTS.md](AGENTS.md);架构/规范/样式总览见 [docs/specs/ARCHITECTURE.md](docs/specs/ARCHITECTURE.md)。

## 架构

Go 后端分单一职责模块:

- `internal/detector` — 按文件特征识别项目类型 + 建议命令(纯函数)
- `internal/scanner` — 遍历工作区调用 detector,产出候选项目
- `internal/store` — 配置持久化(JSON,增删改查 + 路径去重 + 多命令/分组模型)
- `internal/runner` — 子进程启停、登录 shell 包裹、日志环形缓冲、进程树清理、退出标记
- `internal/cmdparse` — 单行命令 ↔ command+args 的转换
- `internal/docker` — docker / docker compose CLI 封装(可注入执行器 + 纯 parser + Client),`logs -f` 复用 runner 托管

顶层 `app.go` 是 Wails 绑定层,组装以上模块并暴露 44 个方法给 Vue3 前端(含 Docker 的 15 个);`tray.go` 管系统托盘;`updater.go` 管自更新与下载加速(另暴露 5 个更新方法)。日志/状态/更新/Docker 状态通过 Wails 事件(`log:<id>` / `status:<id>` / `docker:state` / `docker:available` / `update:state`)推送给前端。

前端 Vue3 + Vite,自写 UI 组件(`components/ui/`)+ 原生 CSS 设计令牌 + 明暗主题,图标用 lucide。顶部 Tab 切换「Projects / Containers」两个视图;compose 项目渲染专用的 `ComposeDetail`,独立容器走 `ContainerPanel`。

## 配置

配置存于各平台标准配置目录下的 `atstarter/config.json`:

- Linux:`~/.config/atstarter/config.json`
- macOS:`~/Library/Application Support/atstarter/config.json`
- Windows:`%AppData%\atstarter\config.json`

结构:`{version, workspaces[], projects[], groups[]}`。写入用「临时文件 + rename」保证原子性。

## 已知限制

- **Windows 进程树终止**:当前用 `cmd.Process.Kill()` 兜底,完整的 Job Object 支持(确保子孙进程一并终止)待后续。Linux/macOS 用进程组信号已完整支持。
- **进程组脱离**:极少数子进程自行 `setsid`/`disown` 脱离会话组的,不受 Stop 的进程组信号覆盖(已知局限)。
- **Docker 依赖**:容器/compose 管理需本机安装 Docker 且 daemon 运行;不可用时相关面板降级(仅提示,不影响其它功能)。compose service 暂不支持加入「启动分组」;`ComposeFile` 支持单文件 `-f`,多文件 override 合并交给 docker 默认发现;`down -v`(删数据卷)不提供。
