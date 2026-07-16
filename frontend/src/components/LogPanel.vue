<script setup>
import { ref, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { GetLogs } from '../../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

const props = defineProps({ projectId: String })
const lines = ref([]) // { stream, text }
const box = ref(null)

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
  <div ref="box" class="log-panel">
    <div v-for="(l, i) in lines" :key="i" :class="['log-line', l.stream]">{{ l.text }}</div>
  </div>
</template>

<style scoped>
.log-panel { flex: 1; overflow-y: auto; background: #1e1e1e; color: #ddd;
  font-family: monospace; font-size: 12px; padding: 8px; white-space: pre-wrap; }
.log-line.stderr { color: #ff6b6b; }
</style>
