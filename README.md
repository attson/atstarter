```
   █████╗ ████████╗    ███████╗████████╗ █████╗ ██████╗ ████████╗███████╗██████╗
  ██╔══██╗╚══██╔══╝    ██╔════╝╚══██╔══╝██╔══██╗██╔══██╗╚══██╔══╝██╔════╝██╔══██╗
  ███████║   ██║       ███████╗   ██║   ███████║██████╔╝   ██║   █████╗  ██████╔╝
  ██╔══██║   ██║       ╚════██║   ██║   ██╔══██║██╔══██╗   ██║   ██╔══╝  ██╔══██╗
  ██║  ██║   ██║       ███████║   ██║   ██║  ██║██║  ██║   ██║   ███████╗██║  ██║
  ╚═╝  ╚═╝   ╚═╝       ╚══════╝   ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝   ╚═╝   ╚══════╝╚═╝  ╚═╝
```

**本地项目快速启动器**  ·  OPEN SOURCE [GO · VUE]

一处托管启动/停止多个项目,查看实时日志。扫描本地工作区,自动识别 docker compose / Node / Go / Rust / Python 等项目类型,保存多套启动命令。AI Agent 可通过 CLI 或 MCP 直接操作。

---

- **体验** — <https://attson.github.io/atstarter/>
- **下载** — [GitHub Releases](https://github.com/attson/atstarter/releases/latest)
- **文档** — <https://attson.github.io/atstarter/guide/>
- **协议** — MIT

---

## 跑起来

```bash
# 只用桌面端:到 Releases 下载对应平台的包,启动即用。

# 源码调试:
make dev              # Wails 桌面端(Ubuntu 24.04 自带 -tags webkit2_41)
```

依赖 Go 1.24+ / Node 20+ / Wails v2。Linux 需 `libwebkit2gtk-4.1-dev libayatana-appindicator3-dev`。

AI Agent / 脚本控制:

```bash
atstarter cli project list
atstarter cli project start <project> --command default
atstarter mcp                                             # stdio MCP server
```

---

## 文档

[快速上手](https://attson.github.io/atstarter/guide/) ·
[AI CLI / MCP](https://attson.github.io/atstarter/guide/ai-cli) ·
[FAQ](https://attson.github.io/atstarter/guide/faq.html)

贡献者从 [CLAUDE.md](./CLAUDE.md) 开始;架构 / 契约规范在 [docs/specs/](./docs/specs/)。
