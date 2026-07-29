# 介绍与使用

AT Starter 是本地项目快速启动器(Wails v2 + Vue3 桌面 App)。读取本地目录代码,
自动识别项目类型并建议启动命令,一处托管启动/停止多个项目并查看实时日志。

支持自定义每个项目的多套启动命令,包括命令行、工作目录和环境变量。例如:
`~/sdk/go1.24.13/bin/go run main.go serve`。

## 支持识别的项目类型

docker compose(`docker-compose.yml` / `compose.yaml` 等,优先识别)、
pnpm / yarn / bun / npm(Node 项目,自动探测 dev/serve/start 脚本)、
Go(根 `main.go` 及 `cmd/*/main.go`)、Rust(cargo)、
Python(Django / poetry / main.py)。compose 项目可切回普通命令模式,识别结果均可手动修改。

## 使用说明

1. **添加项目**:点「Scan」输入工作区根目录,扫描直接子目录以及 `.worktrees/`、
   `.claude/worktrees/` 中的项目;或点「Add」输入单个项目路径。
2. **启动**:选中项目 → 点「▶ 启动」。注意 `go run` 有编译期(依赖多的项目需等待,
   此时日志面板显示「编译/启动中」)。
3. **自定义命令**:在命令条点「Edit」,修改项目名、命令名、命令行、工作目录和环境变量。
   环境变量按 `KEY=value` 每行一条填写;工作目录默认显示项目路径,可直接编辑。
4. **分组**:把常一起启动的「项目 + 命令」加入一个分组,在分组详情里一键启停全组。
5. **文件浏览**:选中项目后,在右侧详情切到「文件」tab,可浏览文件树、代码高亮预览、
   Markdown/PDF/图片/媒体预览,也可编辑保存文本文件和创建、重命名、删除文件。
6. **Docker**:含 compose 文件的目录会识别为 compose 项目,在详情里整体 Up/Down 或
   单独启停某个 service。切到顶部「Containers」标签管理宿主机上的独立容器
   (需本机装 Docker 且 daemon 运行;不可用时面板会提示原因)。
