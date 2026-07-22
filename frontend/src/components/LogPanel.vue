<script setup>
import { ref, watch, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import '@xterm/xterm/css/xterm.css'
import { GetLogs, ClearLogs } from '../../wailsjs/go/main/App'
import { EventsOn, EventsOff, ClipboardSetText, BrowserOpenURL } from '../../wailsjs/runtime/runtime'
import { useTheme } from '../composables/useTheme'

const SCROLLBACK = 1000

const props = defineProps({ projectId: String, status: Object })
const { resolvedTheme } = useTheme()

const termHost = ref(null)
const empty = ref(true)

const banner = computed(() => {
  const st = props.status || {}
  switch (st.State) {
    case 'running':
      return { cls: 'running', text: '● 运行中' + (empty.value ? '(启动中,等待输出…)' : '') }
    case 'exited':
      return { cls: st.ExitCode === 0 ? 'exited-ok' : 'exited-bad',
        text: `● 已退出(exit code ${st.ExitCode}）` }
    case 'error':
      return { cls: 'exited-bad', text: '● 启动错误' }
    default:
      return { cls: 'stopped', text: '○ 未运行' }
  }
})

let term = null
let fitAddon = null
let resizeObserver = null
let currentEvent = ''
let loadToken = 0

// xterm 主题只吃纯色。从当前 data-theme 对应的 CSS 变量取色,主题切换时重设。
function readThemeColors() {
  const cs = getComputedStyle(document.documentElement)
  const pick = (name, fallback) => (cs.getPropertyValue(name).trim() || fallback)
  const dark = resolvedTheme.value === 'dark'
  return {
    background: dark ? '#05060a' : '#f7f7f6',
    foreground: pick('--log-text', dark ? '#d1d5db' : '#27272a'),
    // 光标/选中沿用前景与半透明高亮,足够贴合。
    cursor: pick('--log-text', dark ? '#d1d5db' : '#27272a'),
    selectionBackground: dark ? 'rgba(52,211,153,.25)' : 'rgba(16,185,129,.22)',
  }
}

// 后端日志按行、末尾无换行;writeln 自动补 \r\n,ANSI 由 xterm 解析上色。
function writeLine(text) {
  if (!term) return
  term.writeln(text)
  if (empty.value) empty.value = false
}

async function load(id) {
  const token = ++loadToken
  if (term) term.reset()
  empty.value = true
  if (!id) return
  const hist = await GetLogs(id)
  if (token !== loadToken || !term) return // 被更晚的切换取代
  for (const line of hist || []) writeLine(line)
}

function subscribe(id) {
  if (currentEvent) EventsOff(currentEvent)
  currentEvent = ''
  if (!id) return
  currentEvent = 'log:' + id
  EventsOn(currentEvent, (p) => writeLine(p.text))
}

// ── 右键菜单 ─────────────────────────────────────────
const menu = ref({ show: false, x: 0, y: 0, hasSelection: false })
const toast = ref({ show: false, text: '', ok: true })
let toastTimer = 0

function flashToast(text, ok = true) {
  toast.value = { show: true, text, ok }
  if (toastTimer) clearTimeout(toastTimer)
  toastTimer = setTimeout(() => { toast.value.show = false }, 1600)
}

// 右键弹菜单时立即快照当前选中文本。webkit 下右键/点击过程会清除 xterm 选区,
// 等到点「复制选中」时 getSelection() 已为空,故必须在此刻取。
let selectionSnapshot = ''

function openMenu(e) {
  selectionSnapshot = term ? term.getSelection() : ''
  menu.value = {
    show: true,
    x: e.clientX,
    y: e.clientY,
    hasSelection: !!selectionSnapshot,
  }
}
function closeMenu() { menu.value.show = false }

// 复制到剪贴板:Wails 原生 ClipboardSetText 优先(webkit 下 navigator.clipboard 常被
// 安全策略拦截),失败再退回浏览器 API。任一成功即算复制成功。
async function writeClipboard(text) {
  try {
    if (await ClipboardSetText(text)) return true
  } catch (_) { /* 尝试下一种 */ }
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch (_) { /* 都失败 */ }
  return false
}

async function copySelection() {
  const sel = selectionSnapshot || (term ? term.getSelection() : '')
  closeMenu()
  if (!sel) {
    flashToast('无选中内容', false)
    return
  }
  const ok = await writeClipboard(sel)
  flashToast(ok ? '已复制' : '复制失败', ok)
}

async function clearLogs() {
  if (term) term.clear()
  empty.value = true
  if (props.projectId) {
    try { await ClearLogs(props.projectId) } catch (_) { /* 后端清空失败不阻塞 UI */ }
  }
  closeMenu()
}

// 点击菜单外部关闭菜单;点在菜单内(.ctx-menu)不关,否则会抢在按钮 click 前关掉它。
// 用 capture 阶段监听,故这里靠 closest 判断目标,而非依赖冒泡 .stop。
function onGlobalPointer(e) {
  if (!menu.value.show) return
  if (e.target && e.target.closest && e.target.closest('.ctx-menu')) return
  closeMenu()
}
function onKeydown(e) { if (e.key === 'Escape' && menu.value.show) closeMenu() }

watch(() => props.projectId, async (id) => {
  await load(id)
  subscribe(id)
  await nextTick()
  if (fitAddon) fitAddon.fit()
})

watch(resolvedTheme, () => {
  if (term) term.options.theme = readThemeColors()
})

onMounted(() => {
  term = new Terminal({
    scrollback: SCROLLBACK,
    convertEol: true,
    disableStdin: true,
    cursorBlink: false,
    fontFamily: '"SF Mono", SFMono-Regular, ui-monospace, Menlo, Consolas, "Liberation Mono", monospace',
    fontSize: 11,
    lineHeight: 1.25,
    theme: readThemeColors(),
  })
  fitAddon = new FitAddon()
  term.loadAddon(fitAddon)
  // 识别日志中的 URL 为可点击链接,点击用系统默认浏览器打开(而非 webview 内导航)。
  term.loadAddon(new WebLinksAddon((event, uri) => BrowserOpenURL(uri)))
  term.open(termHost.value)
  fitAddon.fit()

  resizeObserver = new ResizeObserver(() => { if (fitAddon) fitAddon.fit() })
  resizeObserver.observe(termHost.value)

  window.addEventListener('pointerdown', onGlobalPointer, true)
  window.addEventListener('keydown', onKeydown)

  load(props.projectId).then(() => { if (fitAddon) fitAddon.fit() })
  subscribe(props.projectId)
})

onUnmounted(() => {
  if (currentEvent) EventsOff(currentEvent)
  if (resizeObserver) resizeObserver.disconnect()
  window.removeEventListener('pointerdown', onGlobalPointer, true)
  window.removeEventListener('keydown', onKeydown)
  if (toastTimer) clearTimeout(toastTimer)
  if (term) term.dispose()
  term = null
  fitAddon = null
})
</script>

<template>
  <div class="log-wrap">
    <div :class="['banner', banner.cls]">{{ banner.text }}</div>
    <div class="term-area" @contextmenu.prevent="openMenu">
      <div ref="termHost" class="term-host" />
      <div v-if="empty" class="empty-hint">
        <template v-if="(status || {}).State === 'running'">编译/启动中,暂无输出…</template>
        <template v-else>暂无日志。点击「▶ 启动」运行该项目。</template>
      </div>
      <div
        v-if="menu.show"
        class="ctx-menu"
        :style="{ left: menu.x + 'px', top: menu.y + 'px' }"
        @contextmenu.prevent
        @pointerdown.stop
      >
        <button class="ctx-item" :disabled="!menu.hasSelection" @mousedown.prevent @click="copySelection">复制选中</button>
        <button class="ctx-item" @mousedown.prevent @click="clearLogs">清空日志</button>
      </div>
      <div v-if="toast.show" :class="['toast', { bad: !toast.ok }]">{{ toast.text }}</div>
    </div>
  </div>
</template>

<style scoped>
.log-wrap {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  background: var(--log-bg);
  position: relative;
}

.log-wrap::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 1px;
  background: var(--log-hairline);
  pointer-events: none;
  z-index: 2;
}

.banner {
  height: 34px;
  box-sizing: border-box;
  display: flex;
  align-items: center;
  padding: 0 var(--space-6);
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  letter-spacing: 0.03em;
  border-bottom: 1px solid var(--log-border);
  background: var(--log-banner-running-bg);
  box-shadow: var(--log-highlight);
  position: sticky;
  top: 0;
  z-index: 1;
}

.banner.running { color: var(--log-banner-running); }
.banner.exited-ok { color: var(--log-banner-exited-ok); }
.banner.exited-bad,
.banner.error {
  color: var(--log-banner-error);
  background: var(--log-banner-error-bg);
}
.banner.stopped {
  color: var(--log-banner-stopped);
  background: var(--log-banner-stopped-bg);
}

.term-area {
  flex: 1;
  min-height: 0;
  position: relative;
  padding: var(--space-4) var(--space-5);
  overflow: hidden;
}

.term-host {
  width: 100%;
  height: 100%;
}

/* xterm 视口/画布铺满,背景透明交给 .log-wrap 的渐变。 */
.term-host :deep(.xterm),
.term-host :deep(.xterm-viewport) {
  background: transparent !important;
}

.empty-hint {
  position: absolute;
  top: var(--space-6);
  left: var(--space-7);
  color: var(--log-empty);
  font-style: italic;
  font-size: var(--fs-mono);
  pointer-events: none;
}

.ctx-menu {
  position: fixed;
  z-index: 20;
  min-width: 120px;
  padding: var(--space-1);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  background: var(--elevated-gradient, var(--log-bg));
  background-color: var(--surface, #1a1a1f);
  box-shadow: 0 8px 24px rgba(0, 0, 0, .35);
}

.ctx-item {
  display: block;
  width: 100%;
  padding: 6px 10px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text);
  font: inherit;
  font-size: var(--fs-sm);
  text-align: left;
  cursor: pointer;
}

.ctx-item:hover:not(:disabled) { background: var(--elevated-gradient); }
.ctx-item:disabled { color: var(--text-subtle); cursor: default; }

.toast {
  position: absolute;
  bottom: var(--space-5);
  left: 50%;
  transform: translateX(-50%);
  z-index: 20;
  padding: 5px 14px;
  border-radius: var(--radius-full);
  background: var(--surface, #1a1a1f);
  border: 1px solid var(--border-strong);
  color: var(--text);
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
  box-shadow: 0 6px 18px rgba(0, 0, 0, .3);
  pointer-events: none;
}

.toast.bad { color: var(--danger); border-color: var(--danger); }
</style>
