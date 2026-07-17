<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import ProjectList from './components/ProjectList.vue'
import ProjectDetail from './components/ProjectDetail.vue'
import EditProjectDialog from './components/EditProjectDialog.vue'
import ScanDialog from './components/ScanDialog.vue'
import {
  ListProjects, AddProject, StartProject, StopProject,
  GetStatus, UpdateProjectCommand, UpdateProject,
} from '../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime'

const projects = ref([])
const selectedId = ref('')
const statuses = ref({})
const showEdit = ref(false)
const showScan = ref(false)

const selected = computed(() => projects.value.find((p) => p.id === selectedId.value))
const selectedStatus = computed(() => statuses.value[selectedId.value])

let statusSubs = []

function resubscribeStatus() {
  // 先取消旧订阅
  statusSubs.forEach((ev) => EventsOff(ev))
  statusSubs = []
  for (const p of projects.value) {
    const ev = 'status:' + p.id
    EventsOn(ev, (payload) => {
      statuses.value = {
        ...statuses.value,
        [p.id]: { State: payload.state, PID: payload.pid, ExitCode: payload.exitCode },
      }
    })
    statusSubs.push(ev)
  }
}

async function refresh() {
  projects.value = (await ListProjects()) || []
  if (!selectedId.value && projects.value.length) selectedId.value = projects.value[0].id
  resubscribeStatus()
}

async function pollStatuses() {
  const next = {}
  for (const p of projects.value) next[p.id] = await GetStatus(p.id)
  statuses.value = next
}

async function onAdd() {
  const dir = prompt('输入项目目录绝对路径')
  if (!dir) return
  await AddProject(dir)
  await refresh()
}

async function onStart() { await StartProject(selectedId.value); await pollStatuses() }
async function onStop() { await StopProject(selectedId.value); await pollStatuses() }

async function onSaveEdit(payload) {
  const updated = await UpdateProjectCommand(selectedId.value, payload.commandLine)
  updated.name = payload.name
  updated.cwd = payload.cwd
  await UpdateProject(updated)
  showEdit.value = false
  await refresh()
}

let timer
onMounted(async () => {
  await refresh()
  await pollStatuses()
  timer = setInterval(pollStatuses, 1500)
})
onUnmounted(() => {
  clearInterval(timer)
  statusSubs.forEach((ev) => EventsOff(ev))
})
</script>

<template>
  <div class="app">
    <ProjectList :projects="projects" :selectedId="selectedId" :statuses="statuses"
      @select="selectedId = $event" @add="onAdd" @scan="showScan = true" />
    <ProjectDetail :project="selected" :status="selectedStatus"
      @start="onStart" @stop="onStop" @edit="showEdit = true" />
    <EditProjectDialog :show="showEdit" :project="selected"
      @close="showEdit = false" @save="onSaveEdit" />
    <ScanDialog :show="showScan" @close="showScan = false" @added="refresh" />
  </div>
</template>

<style>
html, body, #app { height: 100%; margin: 0; }
.app { display: flex; height: 100vh; font-family: system-ui, sans-serif; background: #fff; }
</style>
