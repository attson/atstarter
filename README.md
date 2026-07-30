# atstarter

atstarter 是一个本地项目快速启动器:扫描工作区、识别项目类型、保存多套启动命令、启动/停止项目、查看实时日志,并把 Docker/compose、文件浏览、系统托盘和自更新放进同一个 Wails 桌面应用。

技术栈:Wails v2 + Go 后端 + Vue3/Vite 前端。配置保存在本机 JSON,不依赖服务端。

## 功能

- **工作区扫描**:扫描工作区直接子目录,包含 `.worktrees/`、`.claude/worktrees/`,自动识别项目类型并批量加入。
- **项目识别与切换**:支持 docker compose、pnpm/yarn/bun/npm、Go、Rust、Python 等;compose 项目可切回普通命令模式。
- **多套启动命令**:每个项目保存多条 `LaunchCommand`(`default` / `debug` / 自定义),每条命令包含 `command + args + cwd + env`。
- **环境变量编辑**:编辑项目时按行填写 `KEY=value`,保存到当前命令的 `env`,启动时与项目级 env 合并。
- **启动分组**:把多个「项目 + 命令」组成分组,一键批量启动/停止。
- **进程托管**:登录 shell 启动、实时 stdout/stderr、退出码日志、Stop 清理整棵进程树。
- **Docker 管理**:compose 项目融入项目树,支持整体与 service 级 Up/Stop/Restart/Down/logs;顶部 `Containers` 面板管理宿主机独立容器。
- **文件浏览器**:项目详情内置「文件」Tab,支持目录树、代码高亮、Markdown/PDF/图片/媒体预览、文本编辑保存、创建/重命名/删除/移入废纸篓。
- **系统托盘**:关闭窗口隐藏到托盘,托盘显示运行数,支持显示/隐藏、停全部、退出。
- **AI/CLI 控制**:桌面 App 启动本地控制服务,`atstarter cli` 和 `atstarter mcp` 可直接管理项目、分组、Docker/compose 与日志。
- **自更新**:检查 GitHub Release,下载产物后用 Ed25519 签名 + SHA256 校验再安装;内置国内下载镜像并回退原始 GitHub URL。
- **明暗主题**:自写设计令牌与浅色/深色主题。
- **官网演示**:VitePress 首页直接嵌入真实前端 App,通过合成 mock 数据展示可交互的项目启动器。

## 使用

1. 点 `Scan` 扫描工作区,或点 `Add` 添加单个项目目录。
2. 选中项目后点右上角 `Start` 启动。`go run` 等命令可能有编译静默期,状态为 running 但短时间无日志是正常的。
3. 在命令条点 `Edit` 修改项目名、命令名、命令行、工作目录和环境变量。工作目录默认填入项目路径,可直接编辑。
4. 需要联动启动时,用 `Add Group` 把当前项目命令加入分组。
5. 切到 `Files` 查看或编辑项目文件;切到顶部 `Containers` 管理独立 Docker 容器。

AI 或脚本可使用 CLI:

```bash
atstarter cli app start --wait
atstarter cli scan ~/GolandProjects --add
atstarter cli project list
atstarter cli project switch-type <project> --type go
atstarter cli group create dev --item api:default
atstarter cli project start <project> --command default
atstarter cli project logs <project> --command default --tail 200
atstarter mcp
```

Codex 和 Claude Code 也可以直接安装 AI 插件。插件会注册 `atstarter mcp`,
并附带使用说明,让 agent 优先通过桌面 App 的本地控制服务操作项目:

```bash
codex plugin marketplace add attson/atstarter --ref main --sparse .agents --sparse plugins
codex plugin add atstarter-control@atstarter

claude plugin marketplace add attson/atstarter --sparse .claude-plugin plugins
claude plugin install atstarter-control@atstarter
```

CLI 输出固定为 JSON。详细命令、MCP 工具和 AI 插件安装见 [docs/ai/atstarter-cli.md](docs/ai/atstarter-cli.md)。

## 支持识别的项目类型

| 类型 | 识别依据 | 默认行为 |
|------|----------|----------|
| docker compose | `docker-compose.yml`、`compose.yaml` 等 | compose 专用详情页,不使用单行命令启动 |
| Node | `package.json` + pnpm/yarn/bun/npm lock/scripts | 优先 dev/serve/start |
| Go | 根 `main.go` 或 `cmd/*/main.go` | `go run ...` |
| Rust | `Cargo.toml` | `cargo run` |
| Python | Django、poetry、`main.py` 等 | 对应框架/入口建议命令 |

识别结果只是建议。框架项目如果需要业务子命令,例如 `go run main.go serve`,请在编辑弹窗手动改。

## 开发

本机硬约束:系统默认 `go` 太旧,请显式使用 `/home/attson/sdk/go1.24.13/bin/go`。

```bash
export GO=/home/attson/sdk/go1.24.13/bin/go
$GO test ./...
$GO test -race ./internal/runner/

cd frontend && npm run build
node --test frontend/src/projectTree.test.mjs frontend/src/commandForms.test.mjs frontend/src/composables/useTheme.test.mjs frontend/src/dockerState.test.mjs frontend/src/envVars.test.mjs frontend/src/projectDetection.test.mjs frontend/src/updateSchedule.test.mjs frontend/src/workspaceRoots.test.mjs
```

Wails 构建:

```bash
make dev
make build
make build-linux
make build-darwin-arm64
make build-darwin-amd64
make build-windows
```

Ubuntu 24.04 只有 `libwebkit2gtk-4.1-dev`,Wails 命令必须带 `-tags webkit2_41`;Makefile 已默认带该 tag。Linux 托盘还需要 `libayatana-appindicator3-dev`。

官网 / 文档站:

```bash
cd site
npm run docs:dev
npm run docs:build
npm run docs:preview
node --test docs/.vitepress/theme/components/homeDemoSource.test.mjs docs/.vitepress/theme/components/mockWails.test.mjs docs/.vitepress/theme/components/homeDemoBundle.test.mjs
```

站点发布到 GitHub Pages 的 `/atstarter/` 路径。首页 demo 复用 `frontend/src/App.vue`,Wails 绑定通过站点 mock 模块替换,fixture 必须使用合成数据。

## 架构

```text
main.go                 Wails 入口 + frontend embed + ldflags
app.go                  Wails 绑定层,导出 64 个 App 方法
cli.go                  JSON CLI,通过桌面 App 本地控制服务操作运行态
control_server.go       桌面 App 内的 localhost 控制服务
mcp.go                  stdio MCP server,复用 CLI/控制协议
tray.go                 系统托盘
updater.go              自更新,另导出 5 个更新方法
internal/
  cmdparse/             单行命令解析/拼接
  detector/             项目类型识别
  scanner/              工作区扫描
  store/                config.json 持久化、命令/分组模型
  runner/               子进程托管、日志、进程树清理
  docker/               docker/compose CLI 封装
  filetree/             项目文件浏览、预览、写入、监听
  control/              控制服务状态文件与 RPC 客户端协议
frontend/src/           Vue3 业务组件、UI 基础组件、主题系统
```

详细规范见:

- [docs/specs/ARCHITECTURE.md](docs/specs/ARCHITECTURE.md)
- [docs/specs/DOMAIN_MODEL.md](docs/specs/DOMAIN_MODEL.md)
- [docs/specs/RUNTIME_CONTRACTS.md](docs/specs/RUNTIME_CONTRACTS.md)
- [docs/specs/FRONTEND_STYLE.md](docs/specs/FRONTEND_STYLE.md)
- [docs/specs/FILE_BROWSER.md](docs/specs/FILE_BROWSER.md)
- [docs/specs/SITE.md](docs/specs/SITE.md)

AI/贡献者硬约束见 [CLAUDE.md](CLAUDE.md)。`AGENTS.md` 仅作为跨工具入口。

## 配置

配置文件位于各平台标准配置目录下的 `atstarter/config.json`:

- Linux: `~/.config/atstarter/config.json`
- macOS: `~/Library/Application Support/atstarter/config.json`
- Windows: `%AppData%\atstarter\config.json`

顶层结构:`{version, workspaces[], projects[], groups[]}`。写入采用临时文件 + rename 保证原子性。

桌面 App 运行时还会写入相邻的 `<config>.control.json`,供 CLI/MCP 发现 localhost 控制服务。该文件只保存运行时 URL/token/PID/version,退出时删除,不会进入持久配置。

## 发布

主分支受保护:功能改动走 GitHub PR,CI 绿后 merge 到 `main`,再从 `main` HEAD 打 `v*` tag。

```bash
git tag -a v0.5.12 -m "v0.5.12"
git push origin v0.5.12
```

`.github/workflows/build.yml` 会构建并发布:

| 平台 | 产物 |
|------|------|
| Linux amd64/arm64 | `.deb` + `.tar.gz` |
| macOS arm64/amd64 | `.dmg` + `.zip` |
| Windows amd64 | NSIS `.exe` + `.zip` |

Release 附带 `SHA256SUMS` 和 `SHA256SUMS.sig`,供自更新校验。

## 已知限制

- Windows 进程树终止目前用 `cmd.Process.Kill()` 兜底,完整 Job Object 支持待后续补齐。
- 极少数自行 `setsid`/`disown` 脱离会话组的孙进程不受 Stop 的进程组信号覆盖。
- compose service 暂不支持加入启动分组。
- `ComposeFile` 只保存单个 compose 文件相对路径;多文件 override 依赖 docker 默认发现。
