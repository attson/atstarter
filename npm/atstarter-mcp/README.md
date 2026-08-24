# @attson/atstarter-mcp

一行式启动 [AT Starter](https://github.com/attson/atstarter) 的 MCP server,适用于任何支持
`mcpServers` 的客户端,无需走 plugin marketplace。

这个包**不下载任何东西**。它假定你已经安装了 AT Starter 桌面端(正式安装包会提供 `atstarter`
命令),启动时定位本机已装的 `atstarter` 二进制并执行 `atstarter mcp`,透传 stdio。

## 前提

先安装 AT Starter 桌面端:<https://github.com/attson/atstarter/releases/latest>

## 配置

任意 MCP 客户端加入:

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

Claude Code 一行搞定:

```bash
claude mcp add atstarter -- npx -y @attson/atstarter-mcp
```

## 二进制定位顺序

1. `ATSTARTER_BIN` 环境变量(指向 `atstarter` 可执行文件时最高优先)。
2. `PATH` 中的 `atstarter`(Windows 为 `atstarter.exe`)。
3. 平台常规安装位置(便携包 / PATH 未刷新时的兜底):
   - macOS:`/usr/local/bin/atstarter`、`/Applications/AT Starter.app/Contents/MacOS/atstarter`
   - Linux:`/usr/bin/atstarter`、`~/.local/bin/atstarter`
   - Windows:`%ProgramFiles%\AT Starter\atstarter.exe` 等

都找不到时向 stderr 打印安装提示并以非 0 退出。

## 说明

- MCP 工具需要 AT Starter 桌面端处于运行状态才有数据;可用 `atstarter_app_start` 工具拉起它。
- 工具清单与用途见:<https://attson.github.io/atstarter/guide/ai-cli>
