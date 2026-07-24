# FAQ / 故障排查

## 启动项目提示 `command not found`(pnpm / nvm / go 等)

AT Starter 的子进程经用户登录交互式 shell(`$SHELL -l -i -c`)启动,以拿到完整 PATH。
若仍报错,确认对应工具在你的登录 shell(`.zshrc` / `.bash_profile` 等)里配置了 PATH。

## `go run` 启动后长时间无日志

`go run` 有编译期,依赖多的项目需等待。此时日志面板会显示「编译/启动中」,属正常现象,
并非卡死。编译完成后才会输出运行日志。

## Docker 面板提示不可用

需本机安装 Docker 且 daemon 正在运行。不可用时面板会给出原因并支持重试。

## 自更新下载卡在 0%

内置下载加速镜像(ghfast.top / gh-proxy.com / ghproxy.net),会逐个尝试并自动回退到
github.com 原始地址,用于解决国内直连 GitHub 下载慢的问题。

## Ubuntu 24.04 从源码构建报 webkit 链接错误(面向开发者)

系统只提供 `libwebkit2gtk-4.1-dev`,而 Wails 2.12 默认链接 4.0。所有 wails 构建命令需加
`-tags webkit2_41`(项目 Makefile 已自动带上)。系统托盘还需 `libayatana-appindicator3-dev`。
