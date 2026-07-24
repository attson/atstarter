# 介绍与使用

AT Starter 是本地项目快速启动器(Wails v2 + Vue3 桌面 App)。读取本地目录代码,
自动识别项目类型并建议启动命令,一处托管启动/停止多个项目并查看实时日志。

支持自定义每个项目的启动命令,包括指定运行时路径,如
`~/sdk/go1.24.13/bin/go run main.go serve`。

## 支持识别的项目类型

docker compose(`docker-compose.yml` / `compose.yaml` 等,优先识别)、
pnpm / yarn / bun / npm(node 项目,自动探测 dev/serve/start 脚本)、
Go(根 `main.go` 及 `cmd/*/main.go`)、Rust(cargo)、
Python(Django / poetry / main.py)。识别结果为建议,可手动修改。

## 使用说明

1. **添加项目**:点「扫描」输入工作区根目录(或用「📁 选择文件夹」),
   勾选识别到的项目加入;或点「+ 添加」输入单个项目路径。
2. **启动**:选中项目 → 点「▶ 启动」。注意 `go run` 有编译期(依赖多的项目需等待,
   此时日志面板显示「编译/启动中」)。
3. **自定义命令**:点「编辑」,在单行输入框改成需要的命令。例如框架项目常需子命令:
   `go run main.go serve`;或指定 go 版本:`~/sdk/go1.24.13/bin/go run main.go serve`。
4. **分组**:把常一起启动的项目加入一个分组,在分组详情里一键启停全组。
5. **文件浏览**:选中项目后,在右侧详情切到「文件」tab,可浏览项目目录并只读预览文本文件。
6. **Docker**:含 compose 文件的目录会识别为 compose 项目,在详情里整体 Up/Down 或
   单独启停某个 service。切到顶部「Containers」标签管理宿主机上的独立容器
   (需本机装 Docker 且 daemon 运行;不可用时面板会提示原因)。
