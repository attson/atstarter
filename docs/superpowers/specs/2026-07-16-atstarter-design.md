# atstarter — 本地项目快速启动器 设计文档

> 日期:2026-07-16
> 状态:设计已确认,待实现

## 1. 背景与目标

在本地有多个工作区(`~/GolandProjects`、`~/WebstormProjects` 等),每个项目的启动方式各不相同:

- `~/WebstormProjects/ad-ai-platform-front` → `pnpm run dev`
- `~/GolandProjects/ad-ai-toolkit` → `go run main.go`
- 也可能需要自定义,如 `~/sdk/go1.23.12/bin/go run main.go serve`(指定 go 版本 + 子命令)

目标:做一个**带 GUI 的全平台桌面 App**,读取本地目录代码,**自动识别项目类型并建议启动命令**,用户可确认或自定义命令,一处**托管启动/停止所有项目**并查看实时日志。

## 2. 技术栈

- **Wails**(Go 后端 + Web 前端),打包成原生桌面 App(Linux / macOS / Windows)。
- 前端:**Vue3**。
- 选型理由:核心逻辑(项目识别、子进程管理、文件系统交互)是 Go 的强项;前端用现有熟悉的 Vue3;Wails 体积小、性能好,前后端通过方法绑定 + 事件推送通信。

## 3. 整体架构

```
┌─────────────────────────────────────────────┐
│  Wails App (atstarter)                        │
│                                               │
│  ┌──────────────┐      ┌───────────────────┐ │
│  │  前端 (Vue3)  │◄────►│  Go 后端           │ │
│  │              │ 绑定  │                   │ │
│  │ - 项目列表    │ 方法  │ - Detector 识别器  │ │
│  │ - 日志面板    │      │ - Runner 进程管理   │ │
│  │ - 配置编辑    │◄────►│ - Store 配置存储   │ │
│  │              │ 事件  │ - Scanner 目录扫描  │ │
│  └──────────────┘ 推送  └───────────────────┘ │
└─────────────────────────────────────────────┘
                              │
                    ┌─────────┴─────────┐
                    ▼                   ▼
            ~/.config/atstarter/   子进程 (pnpm/go...)
              config.json
```

四个核心 Go 模块,各自单一职责、可独立测试:

- **Detector(识别器)**:输入项目目录路径,输出识别结果(项目类型 + 建议启动命令)。内置固定规则表。纯函数式,只读文件系统、无副作用 → 最易测。
- **Scanner(扫描器)**:输入若干工作区根目录,遍历直接子目录,对每个调用 Detector,输出候选项目列表。
- **Store(配置存储)**:读写 `~/.config/atstarter/config.json`,管理项目列表(增删改查)。
- **Runner(进程管理)**:启动/停止子进程,捕获 stdout/stderr 流,管理进程状态和子进程树清理。唯一持有运行时状态的模块。

前后端通信:Wails **方法绑定**(前端调 Go 方法)+ **事件推送**(Go 主动推日志/状态变化)。

## 4. 数据模型 / 配置文件

配置文件路径遵循各平台标准配置目录:

- Linux:`~/.config/atstarter/config.json`
- macOS:`~/Library/Application Support/atstarter/config.json`
- Windows:`%AppData%\atstarter\config.json`

```json
{
  "version": 1,
  "workspaces": [
    "/home/attson/GolandProjects",
    "/home/attson/WebstormProjects"
  ],
  "projects": [
    {
      "id": "路径哈希",
      "name": "ad-ai-toolkit",
      "path": "/home/attson/GolandProjects/ad-ai-toolkit",
      "command": "go",
      "args": ["run", "main.go", "serve"],
      "cwd": "",
      "env": { "GO_ENV": "local" },
      "detectedType": "go",
      "autoDetected": true
    }
  ]
}
```

### 字段设计要点

- **`command` + `args` 分开存**(关键设计):不存整串 `"go run main.go serve"`,而是拆成命令 + 参数数组。原因:
  1. 跨平台 spawn 子进程时避免 shell 解析歧义与注入问题;
  2. 自定义运行时(如 `~/sdk/go1.23.12/bin/go`)直接放 `command` 字段最干净。
- **`cwd`**:工作目录,空则默认用 `path`。
- **`env`**:项目专属环境变量,叠加在系统环境之上。
- **`detectedType` / `autoDetected`**:记录识别类型与"是否自动识别"(用户改过后置 `false`,UI 区分自动/手动)。
- **`id`**:稳定标识,由项目路径哈希生成,重复扫描不会重复添加(去重依据)。

### 命令编辑体验

UI 上是**单行输入框**(如填 `go run main.go serve`),后端用成熟的 shell 词法解析拆成 `command` + `args` 后存储。用户心智负担最小,存储层保持结构化。

## 5. Detector 识别规则表

按优先级从上到下匹配,命中即止。每条:匹配条件 → 建议命令。

| 优先级 | 匹配条件 | 建议命令 | 类型 |
|---|---|---|---|
| 1 | 有 `package.json` + `pnpm-lock.yaml` | `pnpm run dev` | node-pnpm |
| 2 | 有 `package.json` + `yarn.lock` | `yarn dev` | node-yarn |
| 3 | 有 `package.json` + `bun.lockb` | `bun run dev` | node-bun |
| 4 | 有 `package.json`(+`package-lock.json` 或无锁文件) | `npm run dev` | node-npm |
| 5 | 有 `go.mod` + `main.go`(根目录) | `go run main.go` | go |
| 6 | 有 `go.mod` + `cmd/*/main.go` | `go run ./cmd/<第一个>` | go |
| 7 | 有 `Cargo.toml` | `cargo run` | rust |
| 8 | 有 `pyproject.toml` + poetry | `poetry run python main.py` | python-poetry |
| 9 | 有 `manage.py` (Django) | `python manage.py runserver` | python-django |
| 10 | 有 `requirements.txt` / `main.py` / `app.py` | `python <存在的那个>` | python |
| 11 | 有 `Makefile` 且含 `dev:`/`run:` target | `make dev` / `make run` | make |
| 12 | 都不匹配 | 空命令,标记 `unknown`,提示手填 | unknown |

### 增强细节

- **node 的 dev 脚本探测**:命中 node 规则后,读 `package.json` 的 `scripts`。若无 `dev` 但有 `serve`/`start`,则用存在的那个,避免建议一个不存在的脚本。
- **识别永远是"建议"**:进列表时用户可见并可一键改。`unknown` 也能正常添加,命令留空等用户填。
- Detector 保持纯粹:只读文件系统 + 轻量读 JSON,无副作用,给定目录输出恒定 → 单测直接喂假目录结构。

## 6. Runner 进程管理与错误处理

### 状态机

```
stopped ──启动──► running ──进程正常退出──► exited(退出码)
   ▲                 │
   └──停止/杀掉───────┘ (或 running ──启动失败──► error)
```

运行时状态(`stopped`/`running`/`exited`/`error` + 退出码 + PID)存 Runner 内存,**不落配置文件**(配置只存"怎么启动",不存"当前在不在跑")。

### 日志流

- 启动子进程时分别接管 stdout / stderr 管道,逐行读取。
- 每行通过 Wails 事件推给前端(带 `projectId` + 流类型 + 文本)。前端按项目渲染,stderr 标红。
- 后端为每个项目维护**环形缓冲区**(如最近 5000 行),新启动/切换项目时前端可拉历史,避免内存无限增长。

### 子进程树清理(关键)

- **Linux/macOS**:启动时设 `Setpgid` 让子进程自成进程组;停止时对**整个进程组**发信号(`kill -TERM -pgid`),先 SIGTERM,超时(如 5s)再 SIGKILL。确保 `pnpm run dev` fork 出的 node/vite 一起清掉,不留孤儿。
- **Windows**:用 Job Object 绑定子进程及其后代,关闭 Job 时整棵树一起终止。
- **App 退出时**:统一停掉所有 running 项目,不留后台孤儿进程。

### 其它错误场景

- 可执行文件不存在(如手填路径错)→ 启动即失败,状态置 `error`,系统报错显示在日志区,不静默吞掉。
- 项目目录被删除 → 启动前校验路径存在,给明确提示。
- 重复启动已 running 的项目 → 前端禁用启动按钮 + 后端幂等拒绝。

### 跨平台隔离

平台差异收敛在 Runner 内部,对外只暴露 `Start(projectId)` / `Stop(projectId)` / 状态查询。平台相关代码用 Go build tag 隔离(`runner_unix.go` / `runner_windows.go`),上层无感知。

## 7. 前端界面

布局参考 "左列表 + 右详情":

```
┌───────────────┬─────────────────────────────────┐
│ 项目列表        │  ad-ai-toolkit    [▶启动][■停止] │
│               │  go · go run main.go serve  [编辑]│
│ ● ad-toolkit  ├─────────────────────────────────┤
│ ○ ad-front    │  日志 (stdout/stderr 实时流)      │
│ ○ my-rust-app │  > serving on :8080              │
│               │  > compiled successfully         │
│ [+添加][扫描]  │  ...                             │
└───────────────┴─────────────────────────────────┘
```

- **左侧**:项目列表 + 状态灯(灰=stopped / 绿=running / 红=error/异常退出)。底部"+添加单个" + "扫描工作区"。
- **右上**:选中项目信息条 + 启动/停止/编辑按钮。
- **右下**:该项目实时日志面板。
- **编辑弹窗**:名字、单行命令输入框、cwd、env 键值对、工作区根目录管理。
- **扫描弹窗**:选/输入根目录 → 列出候选(带识别结果)→ 勾选批量加入。

### 前后端接口(Wails 绑定,示意)

- 方法:`ListProjects` / `AddProject(path)` / `ScanWorkspaces(dirs)` / `UpdateProject(p)` / `RemoveProject(id)` / `StartProject(id)` / `StopProject(id)` / `GetLogs(id)`
- 事件:`log:{id}`、`status:{id}`

## 8. 测试策略

- **Detector(重点)**:构造临时目录塞不同文件组合(pnpm-lock / go.mod+main.go / cmd 结构 / unknown……),断言识别结果。规则表每条都覆盖。
- **Store**:配置读写、去重(同路径不重复)、损坏 JSON 容错。
- **命令拆分**:shell 词法拆分(带引号、绝对路径、多参数,如 `~/sdk/go1.23.12/bin/go run main.go serve`)。
- **Runner**:启动简单命令(`sleep`/`echo`)、捕获输出、停止、进程树清理(启一个 fork 子进程的脚本,验证停止后子进程消失)。偏集成测试。
- **前端**:以手动验证交互为主,不强求单测。

推进方式:TDD,每个模块先写测试再写实现,Detector 与 Store 尤其适合。

## 9. 非目标(YAGNI)

- 用户自定义识别规则文件(当前用"手动改单个项目命令"兜底即可)。
- 解析并展示 `package.json` 全部 scripts / Go 全部 main 入口供选择(后续可迭代增强)。
- 调起外部系统终端执行(统一走 App 内托管)。
