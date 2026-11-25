# rootyard-control 插件

`rootyard-control` 插件包同时支持 Codex 和 Claude Code。它注册 `rootyard mcp`
MCP server,并额外附带一个 `use-rootyard` skill 引导 AI 使用 `rootyard_*` 工具。

> 多数客户端直接用 npx 一行式接入即可,无需插件:
> `claude mcp add rootyard -- npx -y @attson/rootyard-mcp`
> 详见 [AI / CLI 控制](https://attson.github.io/rootyard/guide/ai-cli)。
> 只有想要附带引导 skill 时才需要下面的插件安装。

## 前提

先安装 Rootyard 桌面端(正式安装包提供 `rootyard` 命令):
<https://github.com/attson/rootyard/releases/latest>

## Codex

安装:

```bash
codex plugin marketplace add attson/rootyard --ref main --sparse .agents --sparse plugins
codex plugin add rootyard-control@rootyard
```

更新:

```bash
codex plugin marketplace upgrade rootyard
codex plugin add rootyard-control@rootyard
```

安装或更新后开新线程,让 Codex 重新加载新的 skill 和 MCP 工具。

## Claude Code

安装:

```bash
claude plugin marketplace add attson/rootyard --sparse .claude-plugin plugins
claude plugin install rootyard-control@rootyard
```

更新:

```bash
claude plugin marketplace update rootyard
claude plugin update rootyard-control
```

安装或更新后执行 `/reload-plugins`,或开启新的 Claude Code 会话。
