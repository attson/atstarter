# atstarter-control 插件

`atstarter-control` 插件包同时支持 Codex 和 Claude Code。它注册 `atstarter mcp`
MCP server,并额外附带一个 `use-atstarter` skill 引导 AI 使用 `atstarter_*` 工具。

> 多数客户端直接用 npx 一行式接入即可,无需插件:
> `claude mcp add atstarter -- npx -y @attson/atstarter-mcp`
> 详见 [AI / CLI 控制](https://attson.github.io/atstarter/guide/ai-cli)。
> 只有想要附带引导 skill 时才需要下面的插件安装。

## 前提

先安装 AT Starter 桌面端(正式安装包提供 `atstarter` 命令):
<https://github.com/attson/atstarter/releases/latest>

## Codex

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

## Claude Code

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
