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
.log-wrap { flex: 1; display: flex; flex-direction: column; min-height: 0; }
.banner { padding: 4px 10px; font-size: 12px; font-weight: 500; border-bottom: 1px solid #333; }
.banner.running { background: #1b3a1b; color: #7fd77f; }
.banner.exited-ok { background: #22331a; color: #9ccc65; }
.banner.exited-bad { background: #3a1b1b; color: #ff8a80; }
.banner.error { background: #3a1b1b; color: #ff8a80; }
.banner.stopped { background: #2a2a2a; color: #999; }
.log-panel { flex: 1; overflow-y: auto; background: #1e1e1e; color: #ddd;
  font-family: monospace; font-size: 12px; padding: 8px; white-space: pre-wrap; }
.log-line.stderr { color: #ff6b6b; }
.empty-hint { color: #777; font-style: italic; padding: 4px 0; }
</style>
