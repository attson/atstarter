# AI / CLI 控制

AT Starter 除了桌面界面,还提供面向脚本和 AI agent 的本地控制入口:

- `atstarter cli ...`: JSON 输出的命令行客户端。
- `atstarter mcp`: stdio MCP server,把同一套能力暴露成 `atstarter_*` 工具。

桌面 App 仍然是运行态来源。CLI/MCP 不会另起一套 runner,也不会绕开 App 直接改运行态。

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

运行 MCP server:

```bash
atstarter mcp
```

主要工具:

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

## AI 插件

仓库内置 `atstarter-control` 插件包,同时支持 Codex 和 Claude Code:

```text
plugins/atstarter-control
```

插件会注册 `atstarter mcp`,并附带使用说明 skill。AI 会优先通过 MCP
调用桌面 App 的本地控制服务;如果桌面 App 未启动,可先调用启动工具。

### Codex

安装:

```bash
codex plugin marketplace add attson/atstarter --ref main --sparse .agents --sparse plugins
codex plugin add atstarter-control@atstarter
```

更新:

```bash
codex plugin marketplace upgrade atstarter
codex plugin add atstarter-control@atstarter
```

安装或更新后开新线程,让 Codex 重新加载新的 skill 和 MCP 工具。

### Claude Code

安装:

```bash
claude plugin marketplace add attson/atstarter --sparse .claude-plugin plugins
claude plugin install atstarter-control@atstarter
```

更新:

```bash
claude plugin marketplace update atstarter
claude plugin update atstarter-control
```

安装或更新后执行 `/reload-plugins`,或开启新的 Claude Code 会话。

### MCP 兜底

不支持插件的客户端可以直接注册 MCP:

```bash
claude mcp add atstarter -- bash -lc 'if command -v atstarter >/dev/null 2>&1; then exec atstarter mcp; fi; echo "atstarter binary not found in PATH; install atstarter or add it to PATH, then retry" >&2; exit 1'
```
