<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { Play, Square, RotateCcw, Trash2, ScrollText, RefreshCw } from 'lucide-vue-next'
import LogPanel from './LogPanel.vue'
import AppButton from './ui/AppButton.vue'
import AppIcon from './ui/AppIcon.vue'
import { groupContainers, filterContainers } from '../dockerState.js'
import {
  ListContainers, StartContainer, StopContainer, RestartContainer,
  FollowContainerLogs, StopFollowContainerLogs, DockerAvailable,
} from '../../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

const emit = defineEmits(['confirm-remove'])
const containers = ref([])
const keyword = ref('')
const selectedId = ref('')
const dockerInfo = ref({ available: true, reason: '' })

const filtered = computed(() => filterContainers(containers.value, keyword.value))
const grouped = computed(() => groupContainers(filtered.value))
const selected = computed(() => containers.value.find((c) => c.id === selectedId.value))
const followRunId = computed(() => selectedId.value ? `container:${selectedId.value}` : '')

async function refresh() {
  dockerInfo.value = await DockerAvailable()
  if (!dockerInfo.value.available) { containers.value = []; return }
  try { containers.value = (await ListContainers()) || [] } catch (e) { containers.value = [] }
}

async function retry() { await refresh() }
function select(c) {
  if (selectedId.value === c.id) return
  if (selectedId.value) StopFollowContainerLogs(selectedId.value)
  selectedId.value = c.id
  FollowContainerLogs(c.id)
}
async function start(c) { await StartContainer(c.id); await refresh() }
async function stop(c) { await StopContainer(c.id); await refresh() }
async function restart(c) { await RestartContainer(c.id); await refresh() }
function requestRemove(c) { emit('confirm-remove', c) }

function onDockerState(list) { containers.value = list || [] }
function onDockerAvailable(info) { dockerInfo.value = info }

onMounted(() => {
  refresh()
  EventsOn('docker:state', onDockerState)
  EventsOn('docker:available', onDockerAvailable)
})
onUnmounted(() => {
  EventsOff('docker:state')
  EventsOff('docker:available')
  if (selectedId.value) StopFollowContainerLogs(selectedId.value)
})

// 供父组件在 remove 确认后调用刷新。
defineExpose({ refresh })
</script>

<template>
  <section class="panel">
    <div v-if="!dockerInfo.available" class="banner">
      ⚠ Docker 不可用:{{ dockerInfo.reason }}
      <AppButton variant="secondary" size="sm" @click="retry">重试</AppButton>
    </div>
    <template v-else>
      <div class="toolbar">
        <input class="search" v-model="keyword" placeholder="筛选容器…" />
        <AppButton variant="secondary" size="sm" iconOnly title="刷新" @click="refresh">
          <template #icon><AppIcon :icon="RefreshCw" :size="14" /></template>
        </AppButton>
      </div>
      <div class="list">
        <template v-for="(list, proj) in grouped.compose" :key="proj">
          <div class="group-label">Compose · {{ proj }}</div>
          <div v-for="c in list" :key="c.id" class="row" :class="{ active: c.id === selectedId }" @click="select(c)">
            <span class="dot" :class="c.state"></span>
            <span class="name">{{ c.name }}</span>
            <span class="status">{{ c.status }}</span>
            <span class="image">{{ c.image }}</span>
            <span class="acts" @click.stop>
              <AppButton v-if="c.state !== 'running'" variant="success" size="sm" iconOnly title="start" @click="start(c)"><template #icon><AppIcon :icon="Play" :size="13" /></template></AppButton>
              <AppButton v-else variant="danger" size="sm" iconOnly title="stop" @click="stop(c)"><template #icon><AppIcon :icon="Square" :size="13" /></template></AppButton>
              <AppButton variant="secondary" size="sm" iconOnly title="restart" @click="restart(c)"><template #icon><AppIcon :icon="RotateCcw" :size="13" /></template></AppButton>
              <AppButton variant="secondary" size="sm" iconOnly title="remove" @click="requestRemove(c)"><template #icon><AppIcon :icon="Trash2" :size="13" /></template></AppButton>
            </span>
          </div>
        </template>
        <div v-if="grouped.standalone.length" class="group-label">Standalone</div>
        <div v-for="c in grouped.standalone" :key="c.id" class="row" :class="{ active: c.id === selectedId }" @click="select(c)">
          <span class="dot" :class="c.state"></span>
          <span class="name">{{ c.name }}</span>
          <span class="status">{{ c.status }}</span>
          <span class="image">{{ c.image }}</span>
          <span class="acts" @click.stop>
            <AppButton v-if="c.state !== 'running'" variant="success" size="sm" iconOnly title="start" @click="start(c)"><template #icon><AppIcon :icon="Play" :size="13" /></template></AppButton>
            <AppButton v-else variant="danger" size="sm" iconOnly title="stop" @click="stop(c)"><template #icon><AppIcon :icon="Square" :size="13" /></template></AppButton>
            <AppButton variant="secondary" size="sm" iconOnly title="restart" @click="restart(c)"><template #icon><AppIcon :icon="RotateCcw" :size="13" /></template></AppButton>
            <AppButton variant="secondary" size="sm" iconOnly title="remove" @click="requestRemove(c)"><template #icon><AppIcon :icon="Trash2" :size="13" /></template></AppButton>
          </span>
        </div>
      </div>
      <LogPanel v-if="followRunId" :projectId="followRunId" :status="{}" />
    </template>
  </section>
</template>

<style scoped>
.panel { flex: 1; min-width: 0; display: flex; flex-direction: column; }
.banner { margin: var(--space-7); padding: var(--space-5) var(--space-6); border: 1px solid var(--warn, #6b5a2c); background: rgba(217,164,65,.1); border-radius: var(--radius-md); color: var(--warn, #d9a441); font-size: var(--fs-sm); display: flex; align-items: center; gap: var(--space-5); }
.toolbar { display: flex; align-items: center; gap: var(--space-4); padding: var(--space-5) var(--space-7); border-bottom: 1px solid var(--border); }
.search { height: 28px; background: var(--surface); border: 1px solid var(--border); border-radius: var(--radius-sm); color: var(--text); padding: 0 var(--space-4); font-size: var(--fs-sm); width: 220px; }
.list { padding: var(--space-4) var(--space-7); overflow: auto; flex: 1; }
.group-label { color: var(--text-subtle); font-size: var(--fs-xs); text-transform: uppercase; letter-spacing: .06em; font-weight: var(--fw-semibold); padding: var(--space-5) 0 var(--space-3); }
.row { display: flex; align-items: center; gap: var(--space-5); padding: var(--space-4) var(--space-5); border: 1px solid var(--border); border-radius: var(--radius-md); margin-bottom: var(--space-3); background: var(--elevated-gradient); cursor: pointer; }
.row.active { outline: 1px solid var(--accent, #4f8cff); }
.name { font-weight: var(--fw-medium); min-width: 120px; }
.status { font-size: var(--fs-xs); color: var(--text-muted); }
.image { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-secondary); margin-left: auto; }
.acts { display: flex; gap: var(--space-2); }
.dot { width: 8px; height: 8px; border-radius: 50%; flex: none; background: var(--text-subtle); }
.dot.running { background: var(--success); }
</style>
