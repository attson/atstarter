# atstarter UI 现代化改造 · 设计文档

> 日期：2026-07-18
> 状态：设计已确认，待实现
> 背景 spec：[2026-07-16-atstarter-design.md](2026-07-16-atstarter-design.md)

## 1. 背景与目标

当前前端（Wails + Vue3）视觉是一次性堆出来的功能实现，配色/字重/间距都硬编码在各组件的 `<style scoped>` 里，主要问题：

- 字重全线 `700 / 800`，视觉过重，缺乏现代工具应有的呼吸感
- Tailwind slate 系颜色（`#d7dce5` / `#cbd5e1` / `#2563eb` ...）散落在 10+ 组件里，没有 token 化，主题/风格调整成本高
- 没有暗色模式（本项目使用者=开发者，暗色是刚需）
- 图标混用 emoji + 三角字符（`📁 ▸ ▾ ▴ ● ○`），观感老旧且大小/对齐不一致
- 一些密度参数偏松（顶栏 50px、侧栏 348px、按钮 32-34px），使可视密度低于同类开发者工具（Linear/Vercel/Raycast）
- 动效为零，交互反馈完全硬切

**目标：** 建立一套设计 token 体系，落地 **Dark（Linear 风）+ Light（shadcn 风）** 双主题、默认跟随系统；同时统一按钮/pill/图标 primitive；升级 typography 与密度；引入基础微交互。日志面板保持深色（IDE/终端风）。

## 2. 非目标 / Out of scope

- **不改后端**，不改 Wails 绑定层，不改事件/命令 API
- **不改三栏大结构**（顶栏 · 侧栏 · 详情/日志）
- 不引入 UI 框架（不上 Element Plus / Naive UI / shadcn-vue），只做设计 token + 少量 primitive
- 不引入完整 CSS 预处理器（原生 CSS 变量足够）
- 不做 Raycast 风格（玻璃/渐变）作为可切主题；仅保留 A + C 两个方向
- 不做多语言切换、不改现有中英文文案
- 不做 Wails 窗口 chrome（透明窗口、frameless）改造

## 3. 设计 tokens

以 CSS 自定义属性形式定义在 `frontend/src/styles/tokens.css`，主题变体在 `theme.dark.css` / `theme.light.css` 里 override 需要变的 token。主题挂载点：`<html data-theme="dark|light">`。

### 3.1 颜色 · Dark

```
--bg:             #0a0b0f     应用底
--surface:        #0c0e13     侧栏/次层面板
--elevated:       #171921     hover/active 面/输入框/pill 底
--border:         #1a1c22     panel 分隔线
--border-strong:  #23262f     elevated 元素边线 / ring
--text:           #f4f5f7     主标题
--text-secondary: #d5d8de     正文
--text-muted:     #a6acb9     次要
--text-subtle:    #5e6371     section label
--primary:        #f4f5f7     主按钮底（白）
--primary-fg:     #0a0b0f     主按钮字
--success:        #4ade80
--success-soft:   rgba(74,222,128,.09)
--success-line:   rgba(74,222,128,.18)
--warning:        #f2c56b
--warning-soft:   rgba(251,191,36,.08)
--warning-line:   rgba(251,191,36,.18)
--danger:         #ef4444
--danger-soft:    rgba(239,68,68,.10)
--danger-line:    rgba(239,68,68,.22)
--danger-fg:      #fca5a5
```

### 3.2 颜色 · Light

```
--bg:             #ffffff
--surface:        #fafaf9
--elevated:       #f4f4f5
--border:         #e7e5e4
--border-strong:  #e4e4e7
--text:           #18181b
--text-secondary: #3f3f46
--text-muted:     #52525b
--text-subtle:    #a1a1aa
--primary:        #18181b
--primary-fg:     #fafaf9
--success:        #16a34a
--success-soft:   #dcfce7
--success-line:   #bbf7d0
--warning:        #a16207
--warning-soft:   #fef9c3
--warning-line:   #fde68a
--danger:         #b91c1c
--danger-soft:    #fef2f2
--danger-line:    #fecaca
--danger-fg:      #b91c1c
```

### 3.3 尺寸 tokens

```
/* 间距 */
--space-1: 2px   --space-2: 4px   --space-3: 6px
--space-4: 8px   --space-5: 10px  --space-6: 12px
--space-7: 16px  --space-8: 20px  --space-9: 24px  --space-10: 32px

/* 圆角 */
--radius-sm: 4px    --radius-md: 7px    --radius-lg: 10px    --radius-full: 999px

/* 阴影（dark / light 共用相对透明度） */
--shadow-sm: 0 1px 2px rgba(0,0,0,.04)
--shadow-md: 0 8px 24px rgba(0,0,0,.10)
--shadow-lg: 0 20px 40px rgba(0,0,0,.16)     /* 弹窗 */

/* 动效 */
--dur-fast: 120ms   --dur-base: 200ms   --dur-slow: 320ms
--ease: cubic-bezier(.2,0,0,1)

/* z-index */
--z-menu: 20   --z-modal: 40   --z-toast: 60
```

## 4. Typography

字号 scale：

| Token | 用途 | 值 |
|---|---|---|
| `--fs-lg` | 页面标题 h1/h2 | `22px / 600 / -0.015em` |
| `--fs-md` | 分组标题 h3/h4 | `16px / 600` |
| `--fs-base` | 正文 | `13px / 400` |
| `--fs-sm` | meta / 次要标签 | `12px / 500` |
| `--fs-xs` | section label / 徽章 | `11px / 600 / +0.03em / uppercase` |
| `--fs-mono` | 命令 / 路径 / 日志 | `12px` |

字体栈：

```css
--font-sans:
  -apple-system, BlinkMacSystemFont, "Segoe UI", "Roboto",
  "PingFang SC", "Microsoft YaHei", "Helvetica Neue", sans-serif;
--font-mono:
  "SFMono-Regular", ui-monospace, Consolas, "Liberation Mono", monospace;
```

**Nunito 移除** —— 只覆盖拉丁字符、缺少 CJK，且现代观感不如系统字体。留 `frontend/src/assets/fonts/` 目录里的 woff2 可以删。

## 5. Primitives 组件

新增 3 个 primitive，放在 `frontend/src/components/ui/`：

### 5.1 `AppButton.vue`

Props: `variant`（`primary` | `secondary` | `success` | `danger`，默认 `secondary`）、`size`（`sm 26px` | `md 30px`，默认 `md`）、`disabled`、`iconOnly`（bool，宽=高）。

Slot: 默认 slot 放文本，`#icon` slot 放图标（放在文本左侧）。

规范：
- 高度 26 / 30，`padding: 0 12/14`，`border-radius: var(--radius-md)`
- 字重 500，字号 12
- `transition: background var(--dur-fast) var(--ease), border-color var(--dur-fast) var(--ease)`
- 各变体从 tokens 取色：
  - primary → `--primary` bg / `--primary-fg` text
  - secondary → `--elevated` bg / `--text-secondary` text / `--border-strong` border
  - success → `--success` bg（dark: 深绿字, light: 白字）
  - danger → `--danger-soft` bg / `--danger-fg` text / `--danger-line` border
- disabled: `opacity: .45; cursor: not-allowed;`

### 5.2 `AppPill.vue`

Props: `variant`（`running` | `exited` | `error` | `stopped` | `neutral`）、`dot`（bool，前置 `●`）。

规范：
- `padding: 2px 9px`，`border-radius: var(--radius-full)`
- 字号 11 / 字重 500
- 变体色映射到 `--success-* / --warning-* / --danger-* / --elevated / border` 组合
- neutral 变体用于 `type-pill` 之类的中性徽章

### 5.3 `AppIcon.vue`

薄封装：`<component :is="icon" />` + 统一 stroke-width 1.75，size prop（默认 14）。使用 `lucide-vue-next` 的按需导入。

例：
```vue
<AppIcon :icon="Play" :size="12" /> Start
```

## 6. 图标系统

引入依赖：**`lucide-vue-next`**（tree-shakeable，按需 import，无运行时开销）。

替换映射：

| 现在 | 换成（lucide） | 用在哪 |
|---|---|---|
| `📁 选择文件夹` | `FolderOpen` | ScanDialog |
| `+ 添加` | `Plus` | 顶栏 Add |
| `▶ 启动` | `Play` | 详情/组 Start |
| Stop（无 icon） | `Square` | 详情/组 Stop |
| Edit（无 icon） | `Pencil` | 详情/组 Edit |
| Scan（无 icon） | `Radar` | 顶栏 Scan |
| `▸ / ▾` chevron | `ChevronRight` / `ChevronDown` | 目录树 / group |
| `● / ○` 状态点 | **保留 CSS 圆点** | 树、pill 内点 |
| `▴ / ▾` command menu 三角 | `ChevronDown` | ProjectDetail |
| Search（当前无） | `Search` | 侧栏搜索框 |
| Group badge `G` | `FolderKanban` | GroupTreeItem |
| 主题切换（新） | `Sun` / `Moon` / `Monitor` | 顶栏切换按钮 |

## 7. 主题系统

### 7.1 组件

- `frontend/src/composables/useTheme.js`：暴露 `theme`（`ref<'system' | 'dark' | 'light'>`）、`resolvedTheme`（`computed<'dark' | 'light'>`）、`cycleTheme()`（system → dark → light → system）
- 顶栏新按钮 `ThemeToggle.vue`：图标 `Monitor | Moon | Sun`（对应 theme 值），点击调 `cycleTheme()`

### 7.2 行为

- 初始化：读 `localStorage.getItem('atstarter.theme')`；无值时 fallback `'system'`
- 应用主题：在 `<html>` 上写 `data-theme`
  - `system` → 读 `window.matchMedia('(prefers-color-scheme: dark)').matches` 决定
  - `dark` / `light` → 直接写
- 监听：只在 `theme === 'system'` 时监听 `matchMedia` change，其他状态解绑
- 持久化：cycleTheme 后 `localStorage.setItem('atstarter.theme', theme.value)`
- 过渡：在 `<html>` 上加 `transition: background-color var(--dur-base) var(--ease), color var(--dur-base) var(--ease);`

## 8. 布局 / 密度调整

**顶栏 `App.vue` `.topbar`**
- 高度 `50 → 48px`
- 左右 padding `18 → 16px`
- 主按钮从 32px 高降到 26px（顶栏节奏）
- 顺序：`brand · summary · (auto) · ThemeToggle · New Group · Scan · Add`

**侧栏 `ProjectList.vue` `.project-list`**
- 宽度 `348 → 300px`（min 280，max 420 —— 加 CSS resize hint 可选，本次不做）
- 搜索框：加 `Search` 图标（`padding-left: 30px` 容纳）
- 高度 `30 → 28px`
- section title 颜色改 `--text-subtle`（更弱）

**项目树 / group 树 `ProjectTreeNode.vue` / `GroupTreeItem.vue`**
- 项目行高 `31 → 28px`
- 目录行高 `26px` 保持
- chevron 换 lucide，size 12
- 选中态从 `background + inset ring` 保持形态但换 tokens
- type-pill 从蓝色改中性（`--elevated / --text-muted`），更收敛

**详情头 `ProjectDetail.vue` / `GroupDetail.vue`**
- padding `18 20 → 16 20`
- 标题 h1 `22px` 保持，加 `letter-spacing: -0.015em`
- 命令框高度 `≈42 → 36px`（padding `8 10 → 6 10`），字号 mono 12
- 命令 picker button 从 26px 保持
- 操作按钮 `.action` 34 → 30，改用 `AppButton`

**日志面板 `LogPanel.vue`**
- 底色 `#0f172a → #06070a`（更深、更 IDE 感）
- Banner 用 sticky 定位
- 字号 12 保持
- **不响应主题切换**，始终深色

**弹窗（4 个）**
- 遮罩：`rgba(0,0,0,.45)` + `backdrop-filter: blur(4px)`
- 面板：`background: var(--surface)`、`border-radius: var(--radius-lg)`、`box-shadow: var(--shadow-lg)`
- 出场：`opacity 0→1 + translateY(4px→0)` `200ms var(--ease)`
- 关闭态：`opacity 1→0 + translateY(0→4px)` `120ms ease`

## 9. 动效

除上文提到的过渡外，全局：

- 所有 `.tree-row`、`.member-row`、`AppButton` 的 hover：`background var(--dur-fast) var(--ease)`
- 状态 dot running 呼吸：`@keyframes pulse-ring { 0%, 100% { box-shadow: 0 0 0 2.5px var(--success-soft) } 50% { box-shadow: 0 0 0 4px var(--success-soft) } }` `animation: pulse-ring 2s ease-in-out infinite`
- 主题切换：`<html>` 的 `background/color/border-color transition var(--dur-base) var(--ease)`
- 日志新增行：**不加动画**（每秒可能多行，性能优先）
- 弹窗出场：见上

## 10. 组件迁移映射

| 文件 | 迁移动作 |
|---|---|
| `frontend/src/style.css` | 去掉 Nunito @font-face + `text-align: left`（保留），加 `<html>` transition |
| `frontend/src/styles/tokens.css` | **新增**：全部 size / motion / z-index token |
| `frontend/src/styles/theme.dark.css` | **新增**：dark 色 token override |
| `frontend/src/styles/theme.light.css` | **新增**：light 色 token override |
| `frontend/src/main.js` | 引入 3 个 css，挂载 `useTheme` 初始化 |
| `frontend/src/composables/useTheme.js` | **新增** |
| `frontend/src/components/ui/AppButton.vue` | **新增** |
| `frontend/src/components/ui/AppPill.vue` | **新增** |
| `frontend/src/components/ui/AppIcon.vue` | **新增** |
| `frontend/src/components/ui/ThemeToggle.vue` | **新增** |
| `frontend/src/App.vue` | topbar 迁移 + 引 ThemeToggle |
| `frontend/src/components/ProjectList.vue` | tokens + Search 图标 + 宽度调整 |
| `frontend/src/components/ProjectTreeNode.vue` | tokens + lucide chevron + 密度 |
| `frontend/src/components/GroupTreeItem.vue` | tokens + lucide + FolderKanban 替换 `G` 徽章 |
| `frontend/src/components/ProjectDetail.vue` | tokens + AppButton + AppPill + lucide 图标 |
| `frontend/src/components/GroupDetail.vue` | tokens + AppButton + AppPill + lucide |
| `frontend/src/components/LogPanel.vue` | 底色/边框 token 化，banner sticky |
| `frontend/src/components/EditProjectDialog.vue` | tokens + AppButton + 出场动效 |
| `frontend/src/components/ScanDialog.vue` | tokens + AppButton + FolderOpen 图标 + 动效 |
| `frontend/src/components/GroupDialog.vue` | 同上 |
| `frontend/src/components/AddProjectDialog.vue` | 同上 |
| `frontend/src/components/AddToGroupDialog.vue` | 同上 |

## 11. 依赖变更

**新增：**
- `lucide-vue-next` — MIT，按需 tree-shake

**移除：**
- `frontend/src/assets/fonts/nunito-v16-latin-regular.woff2`
- `frontend/src/style.css` 里的 `@font-face { font-family: "Nunito" }`

## 12. Rollout 顺序（写入 plan 的任务列表雏形）

分 9 步，每步应独立可运行 & 可 review：

1. **tokens 基础层** —— 新增 3 个 css，`main.js` 引入，`<html>` 挂 `data-theme="dark"` 硬编码看效果（未接主题切换）
2. **useTheme composable + ThemeToggle 组件** —— 顶栏加切换按钮，能三态循环 & localStorage 持久化 & matchMedia 联动
3. **lucide 引入 + 3 个 primitive**（AppButton / AppPill / AppIcon）—— 与旧组件并存
4. **App.vue 顶栏迁移** —— 用新 primitive + tokens 替换硬编码
5. **ProjectList / ProjectTreeNode / GroupTreeItem 迁移**
6. **ProjectDetail / GroupDetail 迁移**
7. **LogPanel 微调** —— 底色/边框 token 化 + banner sticky（保持深色）
8. **4 个 Dialog 迁移** —— 遮罩 + 动效
9. **清理 + 验证** —— 删 Nunito、扫描剩余硬编码 `#hex` 色、双主题手动验收（Wails dev 里切换 macOS System Preferences 深浅色）

## 13. 验证 / Acceptance

- `wails dev -tags webkit2_41` 起来后：
  - 默认跟随 macOS 系统深浅色
  - 顶栏 Toggle 三态循环 `system → dark → light → system` 有效
  - 关掉 App 再开，主题选择保留
  - 系统 System Preferences 切浅深色时（`system` 模式下）实时跟随
- 视觉抽查：
  - 无残留 `#d7dce5 / #cbd5e1 / #2563eb / #16a34a` 等硬编码色（`grep -R "#[0-9a-fA-F]\{6\}" frontend/src/components/` 只应出现在 tokens 相关文件里；SVG stroke `currentColor` 例外）
  - 无残留 emoji（`grep -RP "[\x{1F300}-\x{1F9FF}]" frontend/src/`）
- 交互抽查：
  - 项目 running 状态 dot 有 pulse 呼吸
  - 弹窗打开/关闭有动画
  - hover 有背景过渡
- 现有测试：`go test ./...`、`frontend/src/projectTree.test.mjs` 全部通过（本次不改逻辑）

## 14. 后续可扩展（out of scope but noted）

- Raycast/Warp 风格「Vibrant」第三主题
- 顶栏可折叠侧栏
- 中文文案 i18n / 英文界面
- 键盘快捷键（`⌘K` 快速切项目）
- 全局命令面板（Raycast 风）
