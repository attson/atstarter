# AI / CLI 控制

AT Starter 除了桌面界面,还提供面向脚本和 AI agent 的本地控制入口:

- `atstarter cli ...`: JSON 输出的命令行客户端。
- `atstarter mcp`: stdio MCP server,把同一套能力暴露成 `atstarter_*` 工具。

桌面 App 仍然是运行态来源。CLI/MCP 不会另起一套 runner,也不会绕开 App 直接改运行态。

## 安装与命令名

正式安装包在 macOS、Linux 和 Windows 上统一提供 `atstarter` 命令。macOS 需打开
DMG 并运行其中的 `AT Starter.pkg`;Linux 使用 Deb;Windows 使用安装器并在安装后
新开终端。tar/zip 便携包不会自动加入 PATH。

## 启动模型

桌面 App 启动后会监听 localhost 控制服务,并在配置文件旁写入运行时状态文件:

```text
<config path>.control.json
```

CLI/MCP 读取这个文件并用 bearer token 调用桌面进程。如果桌面 App 没启动:

```bash
atstarter cli app start --wait
```

开发或多配置场景可显式指定配置:

```bash
ATSTARTER_CONFIG=/path/to/config.json atstarter cli app ping
atstarter cli --config /path/to/config.json app ping
```

## 常用 CLI

所有 CLI 输出都是 JSON:

```json
{"ok":true,"data":{}}
{"ok":false,"error":{"code":"app_not_running","message":"atstarter desktop app is not running","hint":"run: atstarter cli app start --wait"}}
```

常见工作流:

```bash
atstarter cli app ping
atstarter cli app start --wait

atstarter cli scan ~/GolandProjects --add
atstarter cli project add /path/to/project
atstarter cli project list
atstarter cli project commands <project>
atstarter cli project detection-options <project>
atstarter cli project switch-type <project> --type go
atstarter cli project switch-type <project> --type compose

atstarter cli project start <project> --command default
atstarter cli project status <project> --command default
atstarter cli project logs <project> --command default --tail 200
atstarter cli project logs <project> --command default --tail 200 --follow
atstarter cli project stop <project> --command default

atstarter cli group create dev --item api:default --item web:serve
atstarter cli group add-item dev --item worker:default
atstarter cli group remove-item dev --item worker:default
atstarter cli group start dev
atstarter cli group stop dev

atstarter cli docker info
atstarter cli container list
atstarter cli compose services <project>
atstarter cli compose up <project> --service web
atstarter cli compose logs <project> --service web --tail 200
```

项目、命令、分组参数可以传 ID 或名称。名称匹配到多个对象时会报错,请改用 ID。

## MCP 工具

接入方式见下方「[接入 AI](#接入-ai)」。MCP server 暴露的主要工具:

| 工具 | 用途 |
|------|------|
| `atstarter_app_ping` / `atstarter_app_start` | 检查或启动桌面 App |
| `atstarter_scan` | 扫描 workspace,可直接加入检测到的项目 |
| `atstarter_project_add` / `atstarter_project_list` | 添加和列出项目 |
| `atstarter_project_commands` | 查看项目启动命令 |
| `atstarter_project_detection_options` / `atstarter_project_switch_type` | 在 compose 与普通命令模式间切换 |
| `atstarter_project_start` / `atstarter_project_stop` / `atstarter_project_restart` | 管理项目命令 |
| `atstarter_project_status` / `atstarter_project_logs` | 读取状态和日志 |
| `atstarter_group_create` / `atstarter_group_update` / `atstarter_group_remove` | 管理启动分组 |
| `atstarter_group_add_item` / `atstarter_group_remove_item` | 增删分组成员 |
| `atstarter_group_start` / `atstarter_group_stop` | 启停分组 |
| `atstarter_docker_info` / `atstarter_container_list` | 查看 Docker 和容器 |
| `atstarter_compose_services` / `atstarter_compose_up` / `atstarter_compose_logs` | 管理 compose 服务 |

MCP tool result 是文本内容,里面包含和 CLI 相同的 JSON envelope。

## 接入 AI

装好 AT Starter 桌面端后,任意支持 `mcpServers` 的客户端加一段 npx 配置即可,
无需 plugin marketplace:

```json
{
  "mcpServers": {
    "atstarter": {
      "command": "npx",
      "args": ["-y", "@attson/atstarter-mcp"]
    }
  }
}
```

Claude Code 也可一行:

```bash
claude mcp add atstarter -- npx -y @attson/atstarter-mcp
```

`@attson/atstarter-mcp` 只是启动器:它不下载任何东西,而是定位本机已装的
`atstarter` 二进制并执行 `atstarter mcp`。二进制不在 PATH(便携包 / 新终端未刷新
PATH)时,它会探测各平台常规安装位置;非标准安装可用 `ATSTARTER_BIN` 指定路径。

AI 会优先通过 MCP 调用桌面 App 的本地控制服务。如果桌面 App 未启动,先调
`atstarter_app_ping` 检查,返回 `app_not_running` 时用 `atstarter_app_start` 拉起。

### 直连已装二进制

已装桌面端、不想经过 npm 的客户端,可以直接注册本机二进制:

```bash
claude mcp add atstarter -- atstarter mcp
```

### 插件包(可选,附带引导 skill)

仓库还内置 `plugins/atstarter-control` 插件包,支持 Codex 和 Claude Code。它在 MCP
之外额外附带一个 `use-atstarter` skill 引导 AI 使用工具。安装步骤见
[插件 README](https://github.com/attson/atstarter/tree/main/plugins/atstarter-control)。
多数客户端直接用上面的 npx 配置即可,无需插件。
