<script setup>
import { ref, computed, watch, onUnmounted } from 'vue'
import { Play, Square, RotateCcw, Trash2, ScrollText } from 'lucide-vue-next'
import LogPanel from './LogPanel.vue'
import AppButton from './ui/AppButton.vue'
import AppPill from './ui/AppPill.vue'
import AppIcon from './ui/AppIcon.vue'
import DetectionSwitch from './DetectionSwitch.vue'
import { hasDetectionSwitch } from '../projectDetection.js'
import {
  ListComposeServices, ComposeUp, ComposeStop, ComposeRestart, ComposeDown,
  FollowComposeLogs, StopFollowComposeLogs,
} from '../../wailsjs/go/main/App'

const props = defineProps({ project: Object, dockerAvailable: Boolean })
const emit = defineEmits(['confirm-down', 'switch-type'])
const services = ref([])
const followRunId = ref('')
const followServiceName = ref('')
// 记录当前正在 follow 的 {projectId, service},切项目/卸载时用它停掉后端 logs -f 进程。
// 不能只依赖 props.project.id,切换后 id 已变,故在启动时快照下来。
let activeFollow = { projectId: '', service: '' }

// stopActiveFollow 停掉当前正在 follow 的 compose logs 进程并清空标记。
async function stopActiveFollow() {
  if (!activeFollow.projectId) return
  const { projectId, service } = activeFollow
  activeFollow = { projectId: '', service: '' }
  try { await StopFollowComposeLogs(projectId, service) } catch (e) { /* 进程可能已退出 */ }
}

async function refresh() {
  if (!props.project || !props.dockerAvailable) { services.value = []; return }
  try { services.value = (await ListComposeServices(props.project.id)) || [] }
  catch (e) { services.value = [] }
}

const aggregate = computed(() => {
  const list = services.value
  if (!list.length) return { label: 'unknown', variant: 'stopped' }
  const running = list.filter((s) => s.state === 'running').length
  if (running === list.length) return { label: `${running}/${list.length} running`, variant: 'running' }
  if (running === 0) return { label: 'stopped', variant: 'stopped' }
  return { label: `${running}/${list.length} running`, variant: 'exited' }
})

async function upAll() { await ComposeUp(props.project.id, ''); await refresh() }
async function stopAll() { await ComposeStop(props.project.id, ''); await refresh() }
async function upService(s) { await ComposeUp(props.project.id, s.name); await refresh() }
async function stopService(s) { await ComposeStop(props.project.id, s.name); await refresh() }
async function restartService(s) { await ComposeRestart(props.project.id, s.name); await refresh() }

function requestDown() { emit('confirm-down', props.project.id) }

async function followLogs(service) {
  const id = service || ''
  await stopActiveFollow()
  followServiceName.value = id
  await FollowComposeLogs(props.project.id, id)
  activeFollow = { projectId: props.project.id, service: id }
  followRunId.value = id ? `compose:${props.project.id}:${id}` : `compose:${props.project.id}`
}

watch(() => props.project?.id, () => { stopActiveFollow(); followRunId.value = ''; refresh() }, { immediate: true })
let timer = setInterval(refresh, 2500)
onUnmounted(() => { clearInterval(timer); stopActiveFollow() })
</script>

<template>
  <section class="detail" v-if="project">
    <div class="project-header">
      <div class="info">
        <div class="title-line">
          <h1>{{ project.name }}</h1>
          <AppPill :variant="aggregate.variant" :dot="aggregate.variant === 'running'">{{ aggregate.label }}</AppPill>
          <DetectionSwitch v-if="hasDetectionSwitch(project)" :project="project" @switch="emit('switch-type', $event)" />
          <AppPill v-else variant="neutral">compose</AppPill>
        </div>
        <div class="path">{{ project.path }}</div>
        <div class="ops">
          <AppButton variant="success" size="sm" :disabled="!dockerAvailable" @click="upAll">
            <template #icon><AppIcon :icon="Play" :size="13" /></template> Up (all)
          </AppButton>
          <AppButton variant="secondary" size="sm" :disabled="!dockerAvailable" @click="stopAll">
            <template #icon><AppIcon :icon="Square" :size="13" /></template> Stop (all)
          </AppButton>
          <AppButton variant="danger" size="sm" :disabled="!dockerAvailable" @click="requestDown">
            <template #icon><AppIcon :icon="Trash2" :size="13" /></template> Down…
          </AppButton>
        </div>
      </div>
    </div>

    <div class="services" v-if="dockerAvailable">
      <div class="svc-head">SERVICES <span class="count">{{ services.length }}</span></div>
      <div v-for="s in services" :key="s.name" class="svc-row">
        <span class="dot" :class="s.state"></span>
        <span class="svc-name">{{ s.name }}</span>
        <span class="svc-state">{{ s.state }}</span>
        <span class="svc-image">{{ s.image }}</span>
        <span class="svc-ports">{{ (s.ports || []).join(', ') }}</span>
        <span class="svc-acts">
          <AppButton v-if="s.state !== 'running'" variant="success" size="sm" iconOnly title="start" @click="upService(s)">
            <template #icon><AppIcon :icon="Play" :size="13" /></template>
          </AppButton>
          <AppButton v-else variant="danger" size="sm" iconOnly title="stop" @click="stopService(s)">
            <template #icon><AppIcon :icon="Square" :size="13" /></template>
          </AppButton>
          <AppButton variant="secondary" size="sm" iconOnly title="restart" @click="restartService(s)">
            <template #icon><AppIcon :icon="RotateCcw" :size="13" /></template>
          </AppButton>
          <AppButton variant="secondary" size="sm" iconOnly title="logs" @click="followLogs(s.name)">
            <template #icon><AppIcon :icon="ScrollText" :size="13" /></template>
          </AppButton>
        </span>
      </div>
    </div>
    <div v-else class="docker-off">Docker 不可用,无法管理 compose 服务。</div>

    <LogPanel v-if="followRunId" :projectId="followRunId" :status="{}" />
  </section>
</template>

<style scoped>
.detail { flex: 1; min-width: 0; display: flex; flex-direction: column; }
.project-header { padding: var(--space-7) var(--space-8); border-bottom: 1px solid var(--border); }
.info { display: flex; flex-direction: column; gap: var(--space-4); }
.title-line { display: flex; align-items: center; gap: var(--space-4); flex-wrap: wrap; }
h1 { margin: 0; font-size: var(--fs-lg); font-weight: var(--fw-semibold); letter-spacing: -.022em; }
.path { color: var(--text-muted); font-family: var(--font-mono); font-size: var(--fs-xs); }
.ops { display: flex; gap: var(--space-3); margin-top: var(--space-3); }
.services { padding: var(--space-5) var(--space-7); overflow: auto; }
.svc-head { color: var(--text-muted); font-size: var(--fs-xs); font-weight: var(--fw-semibold); letter-spacing: .04em; margin-bottom: var(--space-4); }
.svc-head .count { margin-left: var(--space-3); }
.svc-row { display: flex; align-items: center; gap: var(--space-5); padding: var(--space-4) var(--space-5); border: 1px solid var(--border); border-radius: var(--radius-md); margin-bottom: var(--space-3); background: var(--elevated-gradient); }
.svc-name { font-weight: var(--fw-medium); min-width: 100px; }
.svc-state { font-size: var(--fs-xs); color: var(--text-muted); }
.svc-image { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-secondary); }
.svc-ports { font-family: var(--font-mono); font-size: var(--fs-xs); color: var(--text-subtle); margin-left: auto; }
.svc-acts { display: flex; gap: var(--space-2); }
.dot { width: 8px; height: 8px; border-radius: 50%; flex: none; background: var(--text-subtle); }
.dot.running { background: var(--success); }
.dot.partial { background: var(--warn, #d9a441); }
.docker-off { padding: var(--space-8); color: var(--text-muted); }
</style>
