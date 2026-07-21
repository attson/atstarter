# atstarter 架构 / 规范 / 样式总览

> 本文件是入库的**正式**设计与规范文档。AI 迭代过程中的 spec/plan 草稿在
> `docs/superpowers/`(已 gitignore,不入库)。面向贡献者的速览与硬约束见
> 根目录 [AGENTS.md](../../AGENTS.md)。

## 1. 系统概览

atstarter 是一个 Wails v2(Go 后端 + WebKit 前端)桌面应用,把「识别本地项目 →
选启动命令 → 托管子进程 → 看日志」串成一处操作。单二进制,配置存本地 JSON。

```
┌─────────────────────── Wails 桌面进程 ───────────────────────┐
│  前端 (Vue3 + Vite, WebKit 渲染)                              │
│    App.vue ── 业务组件 ×13 ── ui/ 自写组件 ×4 ── 主题系统      │
│        │  调用绑定方法 / 订阅 log·status·update 事件          │
│  ──────┼──────────────────────────────────────────────────  │
│  后端 (Go)                                                    │
│    app.go   绑定层(28 方法)+ 事件推送                        │
│    tray.go  系统托盘        updater.go  自更新 + 下载加速      │
│    internal/ cmdparse · detector · scanner · store · runner  │
└──────────────────────────────────────────────────────────────┘
         │ 子进程(登录 shell 包裹 + setsid 进程组)
         ▼  pnpm dev / go run / cargo run / python main.py …
```

## 2. 后端模块职责(单一职责边界)

| 模块 | 职责 | 依赖 | 纯度 |
|---|---|---|---|
| `internal/cmdparse` | 单行命令字符串 ↔ `(command, args)`(google/shlex) | 无 | 纯函数 |
| `internal/detector` | 按文件特征识别项目类型 + 建议命令 | 只读 FS | 纯函数 |
| `internal/scanner` | 遍历工作区子目录调 detector,产候选 | detector | 只读 FS |
| `internal/store` | config.json 读写、路径去重、多命令/分组模型 | 无 | 有状态(文件) |
| `internal/runner` | 子进程启停、登录 shell、日志缓冲、进程树清理 | os/exec、syscall | 并发 |
| `app.go` | 组装模块,暴露方法,转发事件 | 上述全部 | 绑定层 |
| `tray.go` | 系统托盘菜单、关闭到托盘、运行数 | wails runtime | 有状态 |
| `updater.go` | 检查/下载/校验/安装/取消 + 镜像加速 | net/http、crypto | 有状态 |

### 数据流

```
detector.Detect(dir) → Result{Type, Command}
  → cmdparse.Parse(line) → (command, args)
    → store.Project{ Commands: []LaunchCommand{...} }
      → runner.Spec{ Command, Args, Dir, Env } → 启动
```

## 3. 关键契约

- **去重 ID**:`store.IDForPath(path) = sha1(path)` hex。`Store.Add` 幂等。路径入库前
  `normalizePath`(展开 `~` + `filepath.Abs`)。
- **多命令模型**:`Project.Commands []LaunchCommand{id,name,command,args,cwd,env,isDefault}`。
  默认命令规范化为 `id="default"`;旧配置(无 Commands)由 `NormalizeProjectCommands`
  升级为单条 default。
- **启动分组**:`LaunchGroup{id,name,items:[{projectId,commandId}]}`;`StartGroup`
  批量拉起成员。
- **事件**:runner 经 `SetEmitter`(日志)+ 状态回调推给 app;app 转 Wails 事件
  `log:<id>` `{stream,text}` / `status:<id>` `{state,pid,exitCode}`;自更新走
  `update:state`。运行数变化同步推托盘。
- **退出标记**:进程结束时 runner 向日志追加 `[process exited with code N]`。
- **配置原子写**:临时文件 + `rename`。

## 4. 子进程启动规范(runner)

问题背景:GUI 从桌面/IDE 启动时 PATH 最小化,直接 exec 会 `pnpm/nvm not found`;
脚本内 fork 的子服务(node/vite/esbuild)按父 PID 杀不干净,残留占端口。

- **登录 shell 包裹**:`buildCmd` 在 unix 用 `$SHELL -l -i -c '<line>'`。`-l` 加载 login
  rc,`-i` 加载交互 rc(pnpm/nvm 的 PATH 通常在此)。`line` 由 `shellJoin` 单引号转义
  拼成,防注入。Windows 直接 exec,不包 shell。
- **进程组**:`SysProcAttr{Setsid: true}` 让子进程自成会话首进程,整棵 fork 树同
  `sid==pgid`。`killTree` 对 `-pgid` 发 `SIGTERM`,5s 后 `SIGKILL`。
- **噪声过滤**:无 TTY 时交互 shell 向 stderr 打 job-control 诊断
  (`can't access tty` / `no job control` / `cannot set terminal process group`),
  `isShellNoise` 在 pump 层丢弃这些行,不污染日志、不误伤业务输出。
- **并发规范**:`m.status` 读写在 `r.mu` 锁内;回调锁外调用并传值拷贝;`killTree` 锁外做。
  改并发路径必跑 `go test -race ./internal/runner/`。

## 5. 自更新规范(updater)

- **检查**:轮询 `api.github.com/.../releases/latest`,`versionNewer` 语义化比较
  (整数段比较,非字典序),按平台 `assetPatternFor` 选产物。
- **下载加速**:`mirrorURLs(raw)` 把 `github.com/.../releases/download/...` 展开为
  `[镜像1, 镜像2, 镜像3, 原始URL]`(镜像 = 前缀拼接)。`download` 逐个尝试,失败/超时
  换下一个,**原始 URL 永远兜底**;用户取消(`context.Canceled`)立即中止。仅改写标准
  releases 直链,其他 URL 透传。
- **安全**:下载后 Ed25519 验签 `SHA256SUMS.sig`(公钥 `main.UpdateVerifyPublicKey`,
  build 时 `-ldflags` 注入),再核对产物 SHA256。镜像仅加速,污染的镜像**无法**通过校验。
  dev 构建无公钥 → 只能检查、不能自安装。
- **安装**:抽取 embed 的平台脚本(`scripts/install-*.{sh,ps1}`)detached 运行,脚本
  负责替换二进制并重启,App 随后 `Quit`。

## 6. 前端规范与样式

- **技术栈**:Vue3 Composition API(`<script setup>`)+ Vite 3。**不引第三方 UI 库**。
- **UI 组件**:自写 `components/ui/`(`AppButton` `AppIcon` `AppPill` `ThemeToggle`)。
  图标用 `lucide-vue-next`。
- **样式方案**:原生 CSS + 设计令牌,不用 Tailwind/原子 CSS。
  - `styles/tokens.css` — 设计令牌(色板、间距、圆角、字号等 CSS 变量)。
  - `styles/theme.light.css` / `theme.dark.css` — 明暗主题变量覆盖。
  - 组件样式写在 `<style scoped>`,颜色/尺寸引用令牌变量,不硬编码。
- **组件结构**:左侧 `ProjectList`(树:`ProjectTreeNode` / `GroupTreeNode` /
  `GroupTreeItem`)+ 右侧 `ProjectDetail` / `GroupDetail` + 若干 Dialog +
  `LogPanel` + `UpdateBanner`。
- **前后端交互**:绑定方法从 `wailsjs/go/main/App` import;事件用 `EventsOn/EventsOff`。
  statuses map 内部大写 `State/PID/ExitCode`(与 `GetStatus` 一致),事件 payload 小写,
  回调里映射。改 app.go 方法签名后需 `wails generate module` 重生成绑定。

## 7. 工程规范

- **Go 版本**:开发用 `~/sdk/go1.24.13`,CI 用 1.23.12。系统默认 go 过旧,勿用。
- **构建 tag**:Ubuntu 24.04 一律 `-tags webkit2_41`;托盘需 `libayatana-appindicator3-dev`。
- **测试**:业务包纯 `go test`,TDD(先失败测试再实现);前端有 `node --test`
  (`projectTree.test.mjs` / `useTheme.test.mjs`)。runner 改并发跑 `-race`。
- **分支/发布**:main 禁直接 push,走 GitHub PR;从 main HEAD 打 `v*` tag 触发 CI 发布。
- **commit**:不加 `Co-Authored-By` 尾注;语义化 commit message(feat/fix/test/docs)。

## 8. 已知限制

- Windows 进程树终止用 `cmd.Process.Kill()` 兜底,完整 Job Object 待后续。
- 自行 `setsid`/`disown` 脱离会话组的孙进程不受 Stop 的进程组信号覆盖。
