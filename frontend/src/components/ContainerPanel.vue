<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { ChevronDown, Play, Square, RotateCcw, Trash2, ScrollText, RefreshCw } from 'lucide-vue-next'
import LogPanel from './LogPanel.vue'
import AppButton from './ui/AppButton.vue'
import AppIcon from './ui/AppIcon.vue'
import {
  findComposeProject,
  groupContainers,
  filterContainers,
  summarizeContainers,
} from '../dockerState.js'
import {
  ListContainers, StartContainer, StopContainer, RestartContainer,
  ComposeRestart, ComposeStop, ComposeUp,
  FollowContainerLogs, StopFollowContainerLogs, DockerAvailable,
} from '../../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

const emit = defineEmits(['confirm-remove', 'summary'])
const props = defineProps({
  projects: { type: Array, default: () => [] },
})
const containers = ref([])
const keyword = ref('')
const selectedId = ref('')
const selectedCompose = ref('')
const dockerInfo = ref({ available: true, reason: '' })

const filtered = computed(() => filterContainers(containers.value, keyword.value))
const grouped = computed(() => groupContainers(filtered.value))
const selected = computed(() => containers.value.find((c) => c.id === selectedId.value))
const followRunId = computed(() => selectedId.value ? `container:${selectedId.value}` : '')
const selectedComposeContainers = computed(() => selectedCompose.value ? (grouped.value.compose[selectedCompose.value] || []) : [])
const selectedComposeSummary = computed(() => summarizeContainers(selectedComposeContainers.value))
const selectedComposeProject = computed(() => findComposeProject(props.projects, selectedCompose.value))
const selectedComposePath = computed(() => selectedComposeSummary.value.workingDir || (selectedComposeProject.value || {}).path || '')

watch(containers, (list) => {
  emit('summary', summarizeContainers(list))
}, { immediate: true })

async function refresh() {
  dockerInfo.value = await DockerAvailable()
  if (!dockerInfo.value.available) { containers.value = []; return }
  try { containers.value = (await ListContainers()) || [] } catch (e) { containers.value = [] }
}

async function retry() { await refresh() }
function stopFollow() {
  if (selectedId.value) StopFollowContainerLogs(selectedId.value)
}
function select(c) {
  if (selectedId.value === c.id) return
  stopFollow()
  selectedCompose.value = ''
  selectedId.value = c.id
  FollowContainerLogs(c.id)
}
function selectCompose(name) {
  if (selectedCompose.value === name && !selectedId.value) return
  stopFollow()
  selectedId.value = ''
  selectedCompose.value = name
}
async function start(c) { await StartContainer(c.id); await refresh() }
async function stop(c) { await StopContainer(c.id); await refresh() }
async function restart(c) { await RestartContainer(c.id); await refresh() }
async function runEachComposeContainer(fn) {
  for (const c of selectedComposeContainers.value) {
    await fn(c)
  }
}
async function composeStart() {
  if (selectedComposeProject.value) {
    await ComposeUp(selectedComposeProject.value.id, '')
  } else {
    await runEachComposeContainer((c) => c.state === 'running' ? Promise.resolve() : StartContainer(c.id))
  }
  await refresh()
}
async function composeStop() {
  if (selectedComposeProject.value) {
    await ComposeStop(selectedComposeProject.value.id, '')
  } else {
    await runEachComposeContainer((c) => c.state === 'running' ? StopContainer(c.id) : Promise.resolve())
  }
  await refresh()
}
async function composeRestart() {
  if (selectedComposeProject.value) {
    await ComposeRestart(selectedComposeProject.value.id, '')
  } else {
    await runEachComposeContainer((c) => RestartContainer(c.id))
  }
  await refresh()
}
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
  stopFollow()
})

// 供父组件在 remove 确认后调用刷新。
defineExpose({ refresh })
</script>

<template>
  <section class="panel">
    <div v-if="!dockerInfo.available" class="banner">
      Docker 不可用: {{ dockerInfo.reason }}
      <AppButton variant="secondary" size="sm" @click="retry">重试</AppButton>
    </div>
    <template v-else>
      <div class="container-layout">
        <aside class="container-list">
          <div class="toolbar">
            <input class="search" v-model="keyword" placeholder="筛选容器…" />
            <AppButton variant="secondary" size="sm" iconOnly title="刷新" @click="refresh">
              <template #icon><AppIcon :icon="RefreshCw" :size="14" /></template>
            </AppButton>
          </div>
          <div class="list">
            <template v-for="(list, proj) in grouped.compose" :key="proj">
              <button class="compose-row" :class="{ active: selectedCompose === proj && !selectedId }" @click="selectCompose(proj)">
                <span class="chev"><AppIcon :icon="ChevronDown" :size="12" /></span>
                <span class="name">{{ proj }}</span>
                <span class="count">{{ list.length }}</span>
              </button>
              <div v-for="c in list" :key="c.id" class="row child" :class="{ active: c.id === selectedId }" @click="select(c)">
                <span class="dot" :class="c.state"></span>
                <span class="name">{{ c.name }}</span>
              </div>
            </template>
            <div v-if="grouped.standalone.length" class="group-label">Standalone</div>
            <div v-for="c in grouped.standalone" :key="c.id" class="row" :class="{ active: c.id === selectedId }" @click="select(c)">
              <span class="dot" :class="c.state"></span>
              <span class="name">{{ c.name }}</span>
            </div>
            <div v-if="filtered.length === 0" class="empty-list">没有匹配的容器</div>
          </div>
        </aside>

        <main class="container-detail">
          <template v-if="selectedCompose">
            <div class="detail-header">
              <div class="detail-copy">
                <div class="detail-title">
                  <span class="folder-dot running"></span>
                  <span>{{ selectedCompose }}</span>
                </div>
                <div class="detail-meta">
                  <span>{{ selectedComposeSummary.running }} running</span>
                  <span>{{ selectedComposeSummary.total }} containers</span>
                  <span v-if="!selectedComposeProject && !selectedComposePath">按现有容器批量操作</span>
                </div>
                <div v-if="selectedComposePath" class="path-line">{{ selectedComposePath }}</div>
              </div>
              <div class="detail-actions">
                <AppButton variant="success" size="sm" @click="composeStart">
                  <template #icon><AppIcon :icon="Play" :size="13" /></template>
                  启动全部
                </AppButton>
                <AppButton variant="danger" size="sm" @click="composeStop">
                  <template #icon><AppIcon :icon="Square" :size="13" /></template>
                  停止全部
                </AppButton>
                <AppButton variant="secondary" size="sm" @click="composeRestart">
                  <template #icon><AppIcon :icon="RotateCcw" :size="13" /></template>
                  重启全部
                </AppButton>
              </div>
            </div>
            <div class="compose-detail-list">
              <div v-for="c in selectedComposeContainers" :key="c.id" class="detail-row" @click="select(c)">
                <span class="dot" :class="c.state"></span>
                <span class="detail-row-main">
                  <span class="name">{{ c.name }}</span>
                  <span class="status">{{ c.service || c.status }}</span>
                </span>
                <span class="image">{{ c.image }}</span>
                <span class="detail-actions compact" @click.stop>
                  <AppButton v-if="c.state !== 'running'" variant="success" size="sm" iconOnly title="start" @click="start(c)"><template #icon><AppIcon :icon="Play" :size="13" /></template></AppButton>
                  <AppButton v-else variant="danger" size="sm" iconOnly title="stop" @click="stop(c)"><template #icon><AppIcon :icon="Square" :size="13" /></template></AppButton>
                  <AppButton variant="secondary" size="sm" iconOnly title="restart" @click="restart(c)"><template #icon><AppIcon :icon="RotateCcw" :size="13" /></template></AppButton>
                </span>
              </div>
              <div v-if="selectedComposeSummary.images.length" class="images-panel">
                <div class="section-title">Images</div>
                <div v-for="image in selectedComposeSummary.images" :key="image" class="image-line">{{ image }}</div>
              </div>
            </div>
          </template>
          <template v-else-if="selected">
            <div class="detail-header">
              <div class="detail-copy">
                <div class="detail-title">
                  <span class="dot" :class="selected.state"></span>
                  <span>{{ selected.name }}</span>
                </div>
                <div class="detail-meta">
                  <span>{{ selected.status || selected.state }}</span>
                  <span>{{ selected.image }}</span>
                  <span v-if="selected.compose">Compose · {{ selected.compose }}</span>
                </div>
              </div>
              <div class="detail-actions">
                <AppButton v-if="selected.state !== 'running'" variant="success" size="sm" iconOnly title="start" @click="start(selected)"><template #icon><AppIcon :icon="Play" :size="13" /></template></AppButton>
                <AppButton v-else variant="danger" size="sm" iconOnly title="stop" @click="stop(selected)"><template #icon><AppIcon :icon="Square" :size="13" /></template></AppButton>
                <AppButton variant="secondary" size="sm" iconOnly title="restart" @click="restart(selected)"><template #icon><AppIcon :icon="RotateCcw" :size="13" /></template></AppButton>
                <AppButton variant="secondary" size="sm" iconOnly title="remove" @click="requestRemove(selected)"><template #icon><AppIcon :icon="Trash2" :size="13" /></template></AppButton>
              </div>
            </div>
            <LogPanel :projectId="followRunId" :status="{ State: selected.state === 'running' ? 'running' : 'stopped' }" />
          </template>
          <div v-else class="empty-detail">
            <AppIcon :icon="ScrollText" :size="22" />
            <span>选择一个容器查看日志</span>
          </div>
        </main>
      </div>
    </template>
  </section>
</template>

<style scoped>
.panel { flex: 1; min-width: 0; display: flex; flex-direction: column; }
.banner { margin: var(--space-7); padding: var(--space-5) var(--space-6); border: 1px solid var(--warn, #6b5a2c); background: rgba(217,164,65,.1); border-radius: var(--radius-md); color: var(--warn, #d9a441); font-size: var(--fs-sm); display: flex; align-items: center; gap: var(--space-5); }
.container-layout { flex: 1; min-height: 0; display: grid; grid-template-columns: minmax(280px, 340px) minmax(0, 1fr); }
.container-list { min-height: 0; display: flex; flex-direction: column; border-right: 1px solid var(--border); background: linear-gradient(180deg, rgba(255, 255, 255, .015), transparent), var(--surface); box-shadow: var(--surface-highlight); }
.toolbar { display: flex; align-items: center; gap: var(--space-4); padding: var(--space-4) var(--space-5); border-bottom: 1px solid var(--border); }
.search { min-width: 0; flex: 1; height: 30px; background: var(--elevated-gradient); border: 1px solid var(--border-strong); border-radius: var(--radius-md); color: var(--text); padding: 0 var(--space-4); font: inherit; font-size: var(--fs-sm); outline: none; box-shadow: var(--surface-highlight); }
.search:focus { border-color: var(--accent); box-shadow: 0 0 0 3px var(--focus-ring), var(--surface-highlight); }
.list { padding: var(--space-3) var(--space-4); overflow: auto; flex: 1; min-height: 0; }
.group-label { color: var(--text-subtle); font-size: var(--fs-xs); text-transform: uppercase; letter-spacing: .06em; font-weight: var(--fw-semibold); padding: var(--space-5) var(--space-2) var(--space-3); }
.compose-row {
  width: 100%;
  min-height: 26px;
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: 2px var(--space-3) 2px 6px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-secondary);
  font: inherit;
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
  text-align: left;
  cursor: pointer;
  transition: background var(--dur-fast) var(--ease), box-shadow var(--dur-fast) var(--ease);
}
.compose-row:hover { background: var(--elevated-gradient); }
.compose-row.active { background: var(--elevated-gradient); color: var(--text); box-shadow: inset 0 0 0 1px var(--border-strong), var(--surface-highlight); }
.chev { flex: 0 0 14px; color: var(--text-muted); }
.count {
  flex: 0 0 auto;
  white-space: nowrap;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-full);
  color: var(--text-muted);
  background: transparent;
  padding: 1px 7px;
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
}
.row {
  min-height: 28px;
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: 2px var(--space-4) 2px 18px;
  border: 0;
  border-radius: var(--radius-sm);
  margin: 1px 0;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  transition: background var(--dur-fast) var(--ease), box-shadow var(--dur-fast) var(--ease);
}
.row.child { padding-left: 34px; }
.row:hover { background: var(--elevated-gradient); }
.row.active { background: var(--elevated-gradient); color: var(--text); box-shadow: inset 0 0 0 1px var(--border-strong), var(--surface-highlight); }
.name { flex: 0 1 auto; font-weight: var(--fw-medium); min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.compose-row .name { flex: 0 1 auto; }
.status { font-size: var(--fs-xs); color: var(--text-muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.image { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-secondary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dot { width: 8px; height: 8px; border-radius: 50%; flex: none; background: var(--text-subtle); }
.dot.running { background: var(--success); }
.folder-dot { width: 8px; height: 8px; border-radius: 3px; flex: none; background: var(--text-subtle); }
.folder-dot.running { background: var(--success); }
.empty-list { color: var(--text-muted); font-size: var(--fs-sm); padding: var(--space-7) var(--space-4); }
.container-detail { min-width: 0; min-height: 0; display: flex; flex-direction: column; background: var(--bg); }
.detail-header { min-height: 72px; display: flex; align-items: center; gap: var(--space-5); padding: var(--space-5) var(--space-7); border-bottom: 1px solid var(--border); background: linear-gradient(180deg, rgba(255, 255, 255, .02), transparent); }
.detail-copy { min-width: 0; flex: 1; display: flex; flex-direction: column; gap: var(--space-3); }
.detail-title { min-width: 0; display: flex; align-items: center; gap: var(--space-4); font-size: var(--fs-xl); font-weight: var(--fw-semibold); }
.detail-title span:last-child { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.detail-meta { min-width: 0; display: flex; flex-wrap: wrap; gap: var(--space-4); color: var(--text-muted); font-size: var(--fs-xs); }
.detail-meta span { max-width: 360px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.path-line { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--text-muted); font-family: var(--font-mono); font-size: var(--fs-xs); }
.detail-actions { display: flex; align-items: center; gap: var(--space-3); flex: 0 0 auto; }
.detail-actions.compact { gap: var(--space-2); }
.compose-detail-list { flex: 1; min-height: 0; overflow: auto; padding: var(--space-5) var(--space-7); }
.detail-row { display: grid; grid-template-columns: 10px minmax(180px, 1fr) minmax(180px, 1fr) auto; align-items: center; gap: var(--space-4); padding: var(--space-4) 0; border-bottom: 1px solid var(--border); cursor: pointer; }
.detail-row-main { min-width: 0; display: flex; flex-direction: column; gap: var(--space-2); }
.images-panel { margin-top: var(--space-7); }
.section-title { color: var(--text-subtle); font-size: var(--fs-xs); text-transform: uppercase; letter-spacing: .06em; font-weight: var(--fw-semibold); margin-bottom: var(--space-3); }
.image-line { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-secondary); padding: var(--space-2) 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.empty-detail { flex: 1; min-height: 0; display: flex; align-items: center; justify-content: center; gap: var(--space-4); color: var(--text-muted); font-size: var(--fs-sm); }

@media (max-width: 900px) {
  .container-layout { grid-template-columns: minmax(240px, 300px) minmax(0, 1fr); }
  .detail-header { align-items: flex-start; flex-direction: column; }
  .detail-row { grid-template-columns: 10px minmax(0, 1fr); }
  .detail-row > .image,
  .detail-actions.compact { display: none; }
}
</style>
