# Rootyard

Your local project workspace.

Rootyard 是面向 AI 开发时代的本地项目工作台。它在一处管理本地项目、启动命令、
运行状态、日志、Docker、文件、Git 状态和项目级 Git/Codex 环境，AI Agent 也可通过 CLI 或 MCP
操作同一套桌面运行态。

---

- **体验** — <https://attson.github.io/rootyard/>
- **下载** — [GitHub Releases](https://github.com/attson/rootyard/releases/latest)
- **文档** — <https://attson.github.io/rootyard/guide/>
- **协议** — MIT

---

## 跑起来

```bash
# 只用桌面端:到 Releases 下载对应平台的正式安装包。
# macOS 打开 DMG 后运行 Rootyard.pkg;Linux 使用 Deb;Windows 使用安装器。

# 源码调试:
make dev              # Wails 桌面端(Ubuntu 24.04 自带 -tags webkit2_41)
```

依赖 Go 1.24+ / Node 20+ / Wails v2。Linux 需 `libwebkit2gtk-4.1-dev libayatana-appindicator3-dev`。
正式安装完成后,三个平台都通过 `rootyard` 调用 CLI/MCP;Windows 需新开终端。
tar/zip 便携包不会自动加入 PATH。

AI Agent / 脚本控制:

```bash
rootyard cli project list
rootyard cli project start <project> --command default
rootyard mcp                                             # stdio MCP server
```

项目环境使用显式工作流，保存规则不会修改任何仓库：

```text
创建 Environment Preset -> 配置 Directory Rule -> 扫描仓库
-> 预览按 Git/Codex 分组的逐字段变更 -> 手动确认 -> 应用
```

桌面未启动时也可使用独立的离线命令：

```bash
rootyard env preset list
rootyard env scan
rootyard env diff /path/to/repository
rootyard env apply /path/to/repository          # 交互确认
rootyard env apply /path/to/repository --yes    # 自动化中的显式确认
rootyard env preset set --name Company --codex-set model=custom-model
rootyard env diff /path/to/repository --codex-set sandbox_mode=workspace-write
rootyard env apply /path/to/repository --codex-clear model --yes
rootyard env apply /path/to/repository --codex-set sandbox_mode=danger-full-access \
  --codex-set approval_policy=never --yes --accept-dangerous
```

Git adapter 只管理仓库本地的 `user.name`、`user.email`、`user.signingKey`、
`commit.gpgSign`、`tag.gpgSign` 和 `gpg.format`。扫描、解析、诊断和预览均不会写 Git 配置；
linked worktree 会按共享的 Git common directory 合并为一个应用目标。

Codex adapter 只管理各 worktree 的 `.codex/config.toml` 中五个项目级字段，不修改全局
Codex 配置，也不验证账号是否可用某个模型。`danger-full-access` 与 `never` 同时生效时，
非交互应用除 `--yes` 外还必须显式传入 `--accept-dangerous`。

MCP 最省事的接法(任意客户端,无需 plugin marketplace):

```bash
claude mcp add rootyard -- npx -y @attson/rootyard-mcp
```

---

## 文档

[快速上手](https://attson.github.io/rootyard/guide/) ·
[AI CLI / MCP](https://attson.github.io/rootyard/guide/ai-cli) ·
[FAQ](https://attson.github.io/rootyard/guide/faq.html)

贡献者从 [CLAUDE.md](./CLAUDE.md) 开始;架构 / 契约规范在 [docs/specs/](./docs/specs/)。
