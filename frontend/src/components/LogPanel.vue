<script setup>
import { ref, watch, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { GetLogs } from '../../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

const props = defineProps({ projectId: String, status: Object })
const lines = ref([]) // { stream, text }
const box = ref(null)

// 生命周期状态横幅文本 + 样式。
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

async function load(id) {
  lines.value = []
  if (!id) return
  const hist = await GetLogs(id)
  lines.value = (hist || []).map((t) => ({ stream: 'stdout', text: t }))
  await scrollBottom()
}

function subscribe(id) {
  if (currentEvent) EventsOff(currentEvent)
  if (!id) return
  currentEvent = 'log:' + id
  EventsOn(currentEvent, async (p) => {
    lines.value.push({ stream: p.stream, text: p.text })
    await scrollBottom()
  })
}

async function scrollBottom() {
  await nextTick()
  if (box.value) box.value.scrollTop = box.value.scrollHeight
}

watch(() => props.projectId, async (id) => {
  await load(id)
  subscribe(id)
})

onMounted(() => {
  load(props.projectId)
  subscribe(props.projectId)
})
onUnmounted(() => { if (currentEvent) EventsOff(currentEvent) })
</script>

<template>
  <div class="log-wrap">
    <div :class="['banner', banner.cls]">{{ banner.text }}</div>
    <div ref="box" class="log-panel">
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
  background: linear-gradient(180deg, #05060a 0%, #050609 100%);
  position: relative;
}

.log-wrap::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 1px;
  background: linear-gradient(90deg, transparent, rgba(52, 211, 153, .4), transparent);
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
  border-bottom: 1px solid rgba(255, 255, 255, .05);
  background: linear-gradient(180deg, rgba(16, 185, 129, .06), transparent);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, .03);
  position: sticky;
  top: 0;
  z-index: 1;
}

.banner.running { color: #86efac; }
.banner.exited-ok { color: #bef264; }
.banner.exited-bad,
.banner.error {
  color: #fca5a5;
  background: linear-gradient(180deg, rgba(244, 63, 94, .08), transparent);
}
.banner.stopped {
  color: #94a3b8;
  background: linear-gradient(180deg, rgba(255, 255, 255, .02), transparent);
}

.log-panel {
  flex: 1;
  overflow-y: auto;
  background: transparent;
  color: #d1d5db;
  font-family: var(--font-mono);
  font-size: var(--fs-mono);
  line-height: 1.6;
  padding: var(--space-6) var(--space-7);
  white-space: pre-wrap;
}

.log-line.stderr { color: #fca5a5; }

.empty-hint {
  color: #64748b;
  font-style: italic;
  padding: var(--space-2) 0;
}
</style>
