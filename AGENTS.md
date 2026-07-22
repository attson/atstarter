# AGENTS.md — atstarter

本地项目快速启动器(Wails v2 + Vue3 桌面 App)。给 AI 助手 / 新贡献者的项目速览与硬约束。

## 这个项目是什么

读取本地目录,自动识别项目类型并建议启动命令(pnpm/go/cargo/python…),在一处托管启动/停止多个项目、查看实时日志。核心能力:

- 批量扫描工作区、原生文件夹选择、每项目自定义命令(含指定运行时路径,如 `~/sdk/go1.24.13/bin/go run main.go serve`);
- 每个项目支持**多套启动命令**(default / debug / …);
- **启动分组**:把多个「项目+命令」编成一组,一键批量启停(如「前端+后端」);
- **登录 shell 启动**:子进程经 `$SHELL -l -i -c` 包裹,拿到用户完整 PATH(修复 GUI 启动时 `pnpm/nvm not found`);
- **进程树清理**:setsid 进程组,Stop 杀掉整棵子进程树(shell→pnpm→node→esbuild),不留孤儿占端口;
- **系统托盘**:关闭窗口即隐藏到托盘,托盘显示运行数、可一键停全部;
- **自更新**:轮询 GitHub Release,签名校验后自安装,内置国内下载加速镜像;
- **Docker 管理**:compose 项目(`docker-compose.yml` 等)融入项目树,支持整体与单 service 启停/日志;顶部 `Containers` Tab 管理宿主机独立容器(start/stop/restart/remove/logs)。detached + 2s 轮询,Docker 不可用时优雅降级。

## 硬约束(必须遵守)

| 约束 | 说明 |
|---|---|
| **Go 版本** | 系统默认 `go` 是 1.19(太旧)。**必须**用 `/home/attson/sdk/go1.24.13/bin/go`。所有 go 命令前 `export GO=/home/attson/sdk/go1.24.13/bin/go` 并用 `$GO`。 |
| **webkit tag** | Ubuntu 24.04 只有 `libwebkit2gtk-4.1-dev`,Wails 2.12 默认链 4.0。所有 wails 构建命令加 `-tags webkit2_41`。纯 `go test`/业务逻辑包不需要。 |
| **系统托盘依赖** | Linux 构建托盘需 `libayatana-appindicator3-dev`(CI 已装)。 |
| **commit 署名** | commit message **不要**加 `Co-Authored-By` 尾注。 |
| **runner 并发** | 改 `internal/runner` 的并发路径后**必须**跑 `$GO test -race ./internal/runner/`。所有 `m.status` 读写在 `r.mu` 锁内;回调(emit/onStatus)在锁外调用并传值拷贝;慢操作(killTree)锁外做。 |
| **main 分支保护** | 禁止直接 `git push` main。走 GitHub PR:push feature 分支 → `gh pr create` → CI 绿 → `gh pr merge --merge`。 |
| **TDD** | 业务逻辑改动先写失败测试再实现。runner/store/detector/cmdparse/scanner 都是纯 `go test`,不依赖 GUI。 |

## 架构

```
main.go                 Wails 入口 + -ldflags 注入 Version / UpdateVerifyPublicKey
app.go                  绑定层:组装模块,暴露 28 个方法给前端(含 updater 的 5 个)+ 推送事件
tray.go                 系统托盘:菜单、关闭到托盘、运行数展示、退出放行
updater.go              自更新:检查/下载/校验(Ed25519+SHA256)/安装/取消 + 下载加速镜像
scripts/install-*.{sh,ps1}  各平台安装脚本(自更新 handoff 用,embed 进二进制)
internal/
  cmdparse/             单行命令 ↔ command+args 拆分(github.com/google/shlex)
  detector/             按文件特征识别项目类型 + 建议命令(纯函数,规则表)
  scanner/              遍历工作区直接子目录调 detector,产出候选(含 .worktrees)
  store/                配置 JSON 持久化 + 路径去重(sha1(path) 生成 ID)
  runner/               子进程启停 + 环形缓冲日志 + 进程组清理(build tag 分平台)
  docker/               docker/compose CLI 封装(可注入 exec + 纯 parser + Client),logs -f 复用 runner
frontend/src/           Vue3(App.vue + 业务组件 + 4 个自写 ui/ 组件 + 主题系统)
```

**数据流**:`detector.Detect(dir) → Result{Type,Command}` → `cmdparse.Parse` 拆成 command+args → `store.Project`(可含多条 `Commands`)→ `runner.Spec` 启动。

**关键契约**:
- `store.IDForPath(path)` = sha1(path) hex,是去重依据。`Store.Add` 幂等(同路径不重复)。路径入库前经 `normalizePath`(展开 `~` + `filepath.Abs`)。
- **多命令模型**:`Project.Commands []LaunchCommand`,每条有 `id/name/command/args/cwd/env/isDefault`。默认命令规范化为 `id="default"`;旧配置(无 Commands)由 `NormalizeProjectCommands` 隐式升级为单条 default。
- **启动分组**:`LaunchGroup{id,name,items:[{projectId,commandId}]}`,`StartGroup` 批量拉起各成员。
- `runner` 通过 `SetEmitter`(日志)和状态回调把事件推给 app 层,app 转成 Wails 事件 `log:<id>` / `status:<id>`,前端订阅。运行数变化同时推给托盘。
- 进程退出时 runner 向日志追加 `[process exited with code N]` 尾行。
- **登录 shell**:`runner.buildCmd` 在 unix 用 `$SHELL -l -i -c '<shellJoin 的命令>'`(拿完整 PATH),windows 直接 exec。`isShellNoise` 过滤无 TTY 时 shell 打到 stderr 的 job-control 噪声。进程组用 `Setsid`,`killTree` 对 `-pgid` 发 SIGTERM→5s→SIGKILL。

## 后端绑定方法(44 个:`app.go` 39 + `updater.go` 5)

- **项目**:`ListProjects` `AddProject` `RemoveProject` `UpdateProject` `UpdateProjectCommand` `UpdateProjectCommands`
- **扫描/选择**:`ScanWorkspaces` `AddScanned` `PickDirectory` `GetWorkspaces` `SetWorkspaces`
- **启停/状态**:`StartProject` `StartProjectCommand` `StopProject` `StopProjectCommand` `GetStatus` `GetLogs` `ClearLogs`
- **分组**:`ListGroups` `SaveGroup` `RemoveGroup` `StartGroup` `StopGroup`
- **Docker 探测/容器**:`DockerAvailable` `ListContainers` `StartContainer` `StopContainer` `RestartContainer` `RemoveContainer` `FollowContainerLogs` `StopFollowContainerLogs`
- **compose**:`ListComposeServices` `ComposeUp` `ComposeStop` `ComposeRestart` `ComposeDown` `FollowComposeLogs` `StopFollowComposeLogs`
- **其它**:`GetProjectBranch`(git 分支,纯展示)
- **自更新**:`UpdateGetState` `UpdateCheck` `UpdateStartDownload` `UpdateInstall` `UpdateCancel`

> 改了 app.go 方法签名后,需重新生成绑定:`$($GO env GOPATH)/bin/wails generate module`,前端才能 import 到新方法。

## 前端 ↔ 后端

- **技术栈**:Vue3 Composition API + Vite 3,**无第三方 UI 库** —— 自写 `components/ui/`(AppButton/AppIcon/AppPill/ThemeToggle)+ 原生 CSS 设计令牌 + 明暗主题(`styles/tokens.css` `theme.light.css` `theme.dark.css`),图标用 `lucide-vue-next`。
- 前端调用:`import { ListProjects, StartProject, ... } from '../wailsjs/go/main/App'`。
- 事件:`import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'`;`log:<id>` payload `{stream,text}`;`status:<id>` payload `{state,pid,exitCode}`(小写);自更新走 `update:state` 事件。
- 前端 statuses map 内部用大写 `State/PID/ExitCode`(与轮询 `GetStatus` 返回一致);事件 payload 小写,回调里做映射。

## 常用命令

```bash
export GO=/home/attson/sdk/go1.24.13/bin/go
$GO test ./...                      # 全量后端测试
$GO test -race ./internal/runner/   # runner 并发检查(改并发必跑)
make dev                            # wails dev(自动带 -tags webkit2_41)
make build                          # 本平台打包
make test                           # go test + 前端 node --test
cd frontend && npm run build        # 仅前端构建(不需 GUI)
```

## 发布

打 `v*` tag → `.github/workflows/build.yml` 构建 5 组产物(linux/darwin/windows × arch)、签名 SHA256SUMS、上传 GitHub Release。**从 main 发版**:PR 合并后在 main HEAD 打 tag。

```bash
git tag -a v0.3.2 -m "…"           # 语义化版本
git push origin v0.3.2             # → CI 自动构建 + 发布
```

## 调试 tips(踩过的坑)

- **"点启动后没日志" 多半不是 bug**:①`go run` 有静默编译期(依赖多的项目要几秒~几十秒,期间无输出,status=running 但日志空是正常的);②框架项目(gamesh)需要子命令,如 ad-ai-toolkit 要 `go run main.go serve`,不带 serve 只打印帮助后 exit 0 秒退。detector 无从知道业务子命令,靠用户手动编辑命令兜底。
- **pnpm/nvm not found 已修复**:子进程经登录交互式 shell(`$SHELL -l -i -c`)启动,加载用户 `.zshrc/.bashrc` 拿完整 PATH。若仍报 not found,检查该命令是否真在用户 shell 的 rc 里配了 PATH。
- **子服务停不掉已修复**:setsid 进程组 + `kill -pgid` 杀整树。极少数自行 `setsid`/`disown` 脱离会话的孙进程是已知局限。
- **排查后端状态**:`wails dev` 会起 devserver 在 `http://localhost:34115`,可用浏览器直接调 `window.go.main.App.GetStatus(id)` / `GetLogs(id)` 看真实后端数据。

## 配置文件

各平台标准配置目录下的 `atstarter/config.json`(Linux `~/.config/atstarter/config.json`)。结构:`{version, workspaces[], projects[], groups[]}`,project 含 `id/name/path/command/args/cwd/env/detectedType/autoDetected/commands[]`。写入用「临时文件 + rename」保证原子性。
