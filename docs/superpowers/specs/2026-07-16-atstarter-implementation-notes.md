# atstarter — 实现现状与规范

> 日期:2026-07-16
> 状态:已实现并可运行
> 配套:初始设计见 [2026-07-16-atstarter-design.md](2026-07-16-atstarter-design.md);本文档记录初始设计之后的演进、当前接口现状与代码/样式规范。

本文档反映**当前代码的真实状态**。初始设计文档是需求确认阶段的快照,部分接口在实现中扩展了。

## 1. 相对初始设计的演进

初始设计之后新增/变更的能力:

| 变更 | 说明 |
|---|---|
| **`~` 路径展开** | `ScanWorkspaces` 与 `normalizePath` 展开开头的 `~`/`~/…` 为家目录。修复了手输 `~/GolandProjects` 扫描无结果的问题。 |
| **原生文件夹选择** | 新增 `PickDirectory()` 方法(`runtime.OpenDirectoryDialog`),扫描弹窗「📁 选择文件夹」按钮,选中后自动扫描。 |
| **路径归一化去重** | `AddProject`/`AddScanned` 用 `normalizePath`(展开 `~` + `filepath.Abs`)后再算 ID,避免相对/非规范路径重复入库。 |
| **状态事件推送** | runner 新增 `SetStatusListener`,进程状态变化时主动推 `status:<id>` 事件,前端即时更新(保留 1.5s 轮询作兜底)。 |
| **进程退出标记** | 进程退出/出错时,runner 向日志追加一行 `[process exited with code N]` / `[process error: …]`,日志区自证结局。 |
| **生命周期状态横幅** | 日志面板顶部显示状态横幅(运行中 / 已退出+码 / 错误 / 未运行)+ 编译期空态提示,解决 `go run` 编译期"看着卡死"。 |

## 2. 当前 App 绑定接口(14 个方法)

| 方法 | 用途 |
|---|---|
| `ListProjects() ([]Project, error)` | 列出所有已保存项目 |
| `AddProject(path) (Project, error)` | 单个添加(识别 + 归一化 + 保存) |
| `ScanWorkspaces(roots) []Project` | 扫描候选(展开 `~`,不保存) |
| `PickDirectory() (string, error)` | 原生目录选择器 |
| `AddScanned(projects) error` | 批量保存勾选候选 |
| `UpdateProject(p) error` | 覆盖保存项目 |
| `UpdateProjectCommand(id, line) (Project, error)` | 单行命令拆分更新(置 autoDetected=false) |
| `RemoveProject(id) error` | 删除(先 Stop 再删) |
| `StartProject(id) / StopProject(id) error` | 启停进程 |
| `GetStatus(id) runner.Status` | 状态快照 {State,PID,ExitCode} |
| `GetLogs(id) []string` | 日志缓冲快照 |
| `SetWorkspaces(dirs) / GetWorkspaces()` | 工作区根目录读写 |

**事件**:`log:<id>` payload `{stream,text}`;`status:<id>` payload `{state,pid,exitCode}`(小写)。

## 3. 代码规范

- **分层**:业务逻辑在 `internal/*` 纯 Go 包(可独立 `go test`,不依赖 Wails/GUI);`app.go` 是薄绑定层,只做组装 + 事件转发,不含业务逻辑。
- **数据结构单一来源**:`store.Project` / `store.Config` 是配置的唯一模型,scanner/app 复用,不另立结构。
- **命令存储**:UI 单行字符串 ↔ 存储 command+args 分离,转换只经 `cmdparse`。绝不把整串命令塞进单个字段。
- **纯函数优先**:`detector` 只读文件系统、无副作用,给定目录输出恒定 → 直接喂假目录测试。
- **并发**:`runner` 是唯一有运行时状态的模块。所有 `m.status` 读写在 `r.mu` 锁内;回调在锁外调用并传值拷贝;慢操作(killTree)锁外做。改并发路径**必跑** `go test -race ./internal/runner/`。
- **平台隔离**:平台相关代码用 build tag(`process_unix.go` `!windows` / `process_windows.go` `windows`),上层无感知。
- **TDD**:业务改动先写失败测试再实现。前端以手动/浏览器验证为主。
- **commit**:message 不带 `Co-Authored-By` 尾注。

## 4. 前端样式规范

- **全局对齐**:去掉了 wails 脚手架 `html`/`#app` 的 `text-align:center`(它会污染全局,让日志/列表居中)。默认左对齐。
- **布局**:`.app` 用 flex,左列表(240px 固定)+ 右详情(flex:1)。详情区内 flex 纵向:信息栏 + 日志面板(`min-height:0` 保证可滚动)。
- **日志面板**:深色终端风(背景 `#1e1e1e`、等宽字体、`white-space:pre-wrap`)。stderr 行标红 `#ff6b6b`。顶部状态横幅按状态着色(运行中绿 / 异常退出红 / 停止灰)。
- **扫描候选行**:三列固定宽对齐(name 200px 深色加粗 / type 90px 色码 / command 占剩余),超长省略号;空命令显示 `—` 占位。
- **状态灯**:列表项圆点,绿=running、红=error/异常退出、灰=stopped。

## 5. 仍未做(YAGNI / 后续)

- Windows 完整进程树终止(Job Object)——当前 `cmd.Process.Kill()` 兜底。
- 用户自定义识别规则文件。
- 解析展示 `package.json` 全部 scripts / Go 全部 main 入口供选择。
- 调起外部系统终端(统一走 App 内托管)。
