<script setup>
import { ref, watch, computed, onMounted, onUnmounted } from 'vue'
import { GetLogs } from '../../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

const MAX_VISIBLE = 1000

const props = defineProps({ projectId: String, status: Object })
const lines = ref([]) // { stream, text }
const box = ref(null)

const banner = computed(() => {
  const st = props.status || {}
  switch (st.State) {
    case 'running':
      return { cls: 'running', text: '● 运行中' + (lines.value.length === 0 ? '(go run 可能正在编译,请稍候…)' : '') }
    case 'exited':
      return { cls: st.ExitCode === 0 ? 'exited-ok' : 'exited-bad',
        text: `● 已退出(exit code ${st.ExitCode}）` }
    case 'error':
      return { cls: 'exited-bad', text: '● 启动错误' }
    default:
      return { cls: 'stopped', text: '○ 未运行' }
  }
})

let currentEvent = ''
let loadToken = 0
let pending = []
let rafHandle = 0
let stickToBottom = true

function scheduleFlush() {
  if (rafHandle) return
  rafHandle = requestAnimationFrame(() => {
    rafHandle = 0
    if (!pending.length) return
    const batch = pending
    pending = []
    const next = lines.value.concat(batch)
    lines.value = next.length > MAX_VISIBLE ? next.slice(-MAX_VISIBLE) : next
    if (stickToBottom && box.value) {
      // Wait one more frame for Vue's DOM update.
      requestAnimationFrame(() => {
        if (box.value) box.value.scrollTop = box.value.scrollHeight
      })
    }
  })
}

function updateStickiness() {
  const el = box.value
  if (!el) return
  const gap = el.scrollHeight - el.scrollTop - el.clientHeight
  stickToBottom = gap < 40
}

async function load(id) {
  const token = ++loadToken
  pending = []
  if (rafHandle) { cancelAnimationFrame(rafHandle); rafHandle = 0 }
  lines.value = []
  stickToBottom = true
  if (!id) return
  const hist = await GetLogs(id)
  if (token !== loadToken) return // superseded by a newer switch
  const mapped = (hist || []).map((t) => ({ stream: 'stdout', text: t }))
  lines.value = mapped.length > MAX_VISIBLE ? mapped.slice(-MAX_VISIBLE) : mapped
  requestAnimationFrame(() => {
    if (box.value) box.value.scrollTop = box.value.scrollHeight
  })
}

function subscribe(id) {
  if (currentEvent) EventsOff(currentEvent)
  currentEvent = ''
  if (!id) return
  currentEvent = 'log:' + id
  EventsOn(currentEvent, (p) => {
    pending.push({ stream: p.stream, text: p.text })
    scheduleFlush()
  })
}

function onScroll() { updateStickiness() }

watch(() => props.projectId, async (id) => {
  await load(id)
  subscribe(id)
})

onMounted(() => {
  load(props.projectId)
  subscribe(props.projectId)
})
onUnmounted(() => {
  if (currentEvent) EventsOff(currentEvent)
  if (rafHandle) cancelAnimationFrame(rafHandle)
})
</script>

<template>
  <div class="log-wrap">
    <div :class="['banner', banner.cls]">{{ banner.text }}</div>
    <div ref="box" class="log-panel" @scroll.passive="onScroll">
      <div v-if="lines.length === 0" class="empty-hint">
        <template v-if="(status || {}).State === 'running'">编译/启动中,暂无输出…</template>
        <template v-else>暂无日志。点击「▶ 启动」运行该项目。</template>
      </div>
      <div v-for="(l, i) in lines" :key="i" :class="['log-line', l.stream]">{{ l.text }}</div>
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

.log-panel {
  flex: 1;
  overflow-y: auto;
  background: transparent;
  color: var(--log-text);
  font-family: var(--font-mono);
  font-size: var(--fs-mono);
  line-height: 1.6;
  padding: var(--space-6) var(--space-7);
  white-space: pre-wrap;
}

.log-line.stderr { color: var(--log-text-stderr); }

.empty-hint {
  color: var(--log-empty);
  font-style: italic;
  padding: var(--space-2) 0;
}
</style>
