# FAQ / 故障排查

## 启动项目提示 `command not found`(pnpm / nvm / go 等)

AT Starter 的子进程经用户登录交互式 shell(`$SHELL -l -i -c`)启动,以拿到完整 PATH。
若仍报错,确认对应工具在你的登录 shell(`.zshrc` / `.bash_profile` 等)里配置了 PATH。

## `go run` 启动后长时间无日志

`go run` 有编译期,依赖多的项目需等待。此时日志面板会显示「编译/启动中」,属正常现象,
并非卡死。编译完成后才会输出运行日志。

## Docker 面板提示不可用

需本机安装 Docker 且 daemon 正在运行。不可用时面板会给出原因并支持重试。

## CLI 提示 `app_not_running`

CLI/MCP 调用的是桌面 App 内的本地控制服务。先启动桌面 App,或直接执行:

```bash
atstarter cli app start --wait
```

如果使用的是开发配置或自定义配置,同时传入 `ATSTARTER_CONFIG` 或 `--config`。

## MCP 工具没有出现

确认 `atstarter` 二进制在 PATH 中,并重新安装本地插件:

```bash
codex plugin add atstarter-control@personal
```

重新安装后开新线程。Codex 需要新会话才能加载更新后的 skill 和 MCP 工具。

## 安装后提示 `atstarter: command not found`

请确认使用的是正式安装包:macOS 打开 DMG 后需要运行其中的 `AT Starter.pkg`,不能只
拖出 App;Linux 使用 Deb;Windows 使用安装器并新开一个终端。tar/zip 便携包不会自动
加入 PATH。

## 自更新下载卡在 0%

内置下载加速镜像(ghfast.top / gh-proxy.com / ghproxy.net),会逐个尝试并自动回退到
github.com 原始地址,用于解决国内直连 GitHub 下载慢的问题。

## Ubuntu 24.04 从源码构建报 webkit 链接错误(面向开发者)

系统只提供 `libwebkit2gtk-4.1-dev`,而 Wails 2.12 默认链接 4.0。所有 wails 构建命令需加
`-tags webkit2_41`(项目 Makefile 已自动带上)。系统托盘还需 `libayatana-appindicator3-dev`。
