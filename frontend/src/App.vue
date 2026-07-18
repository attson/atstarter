<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import ProjectList from './components/ProjectList.vue'
import ProjectDetail from './components/ProjectDetail.vue'
import GroupDetail from './components/GroupDetail.vue'
import EditProjectDialog from './components/EditProjectDialog.vue'
import ScanDialog from './components/ScanDialog.vue'
import GroupDialog from './components/GroupDialog.vue'
import AddProjectDialog from './components/AddProjectDialog.vue'
import AddToGroupDialog from './components/AddToGroupDialog.vue'
import {
  ListProjects, AddProject, StartProjectCommand, StopProjectCommand,
  GetStatus, UpdateProjectCommands, ListGroups, SaveGroup, RemoveGroup,
  StartGroup, StopGroup,
} from '../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime'

const projects = ref([])
const groups = ref([])
const selectedId = ref('')
const selectedGroupId = ref('')
const statuses = ref({})
const selectedCommandIds = ref({})
const showEdit = ref(false)
const showScan = ref(false)
const showGroup = ref(false)
const showAddProject = ref(false)
const showAddToGroup = ref(false)
const editingGroup = ref(null)

const selected = computed(() => projects.value.find((p) => p.id === selectedId.value))
const selectedGroup = computed(() => groups.value.find((g) => g.id === selectedGroupId.value))
const selectedCommandId = computed(() => selectedCommandIds.value[selectedId.value] || defaultCommandId(selected.value))
const selectedRunId = computed(() => selected.value ? runIdForCommand(selected.value.id, selectedCommandId.value) : '')
const selectedCommand = computed(() => commandsFor(selected.value).find((c) => c.id === selectedCommandId.value) || commandsFor(selected.value)[0])
const selectedStatus = computed(() => statuses.value[selectedRunId.value])
const runningCount = computed(() => Object.values(statuses.value).filter((s) => s && s.State === 'running').length)
const exitedCount = computed(() => Object.values(statuses.value).filter((s) => s && (s.State === 'exited' || s.State === 'error')).length)
const projectStatuses = computed(() => {
  const out = {}
  for (const p of projects.value) {
    const commandStatuses = commandsFor(p).map((c) => statuses.value[runIdForCommand(p.id, c.id)]).filter(Boolean)
    out[p.id] = commandStatuses.find((s) => s.State === 'running') ||
      commandStatuses.find((s) => s.State === 'error' || s.State === 'exited') ||
      { State: 'stopped' }
  }
  return out
})

let statusSubs = []

function commandsFor(project) {
  if (!project) return []
  if (project.commands && project.commands.length) return project.commands
  return [{
    id: 'default',
    name: 'Default',
    command: project.command,
    args: project.args || [],
    cwd: project.cwd || '',
    env: project.env || {},
    isDefault: true,
  }]
}

function defaultCommandId(project) {
  const cmd = commandsFor(project).find((c) => c.isDefault) || commandsFor(project)[0]
  return cmd ? cmd.id : 'default'
}

function runIdForCommand(projectId, commandId) {
  return `${projectId}:${commandId || 'default'}`
}

function setSelectedCommand(commandId) {
  if (!selectedId.value) return
  selectedCommandIds.value = { ...selectedCommandIds.value, [selectedId.value]: commandId }
}

function selectCommand(payload) {
  selectedGroupId.value = ''
  selectedId.value = payload.projectId
  selectedCommandIds.value = { ...selectedCommandIds.value, [payload.projectId]: payload.commandId }
}

function selectProject(id) {
  selectedGroupId.value = ''
  selectedId.value = id
}

function selectGroup(id) {
  selectedGroupId.value = id
}

function resubscribeStatus() {
  // 先取消旧订阅
  statusSubs.forEach((ev) => EventsOff(ev))
  statusSubs = []
  for (const p of projects.value) {
    for (const cmd of commandsFor(p)) {
      const runId = runIdForCommand(p.id, cmd.id)
      const ev = 'status:' + runId
      EventsOn(ev, (payload) => {
        statuses.value = {
          ...statuses.value,
          [runId]: { State: payload.state, PID: payload.pid, ExitCode: payload.exitCode },
        }
      })
      statusSubs.push(ev)
    }
  }
}

async function refresh() {
  projects.value = (await ListProjects()) || []
  groups.value = (await ListGroups()) || []
  if (!selectedId.value && projects.value.length) selectedId.value = projects.value[0].id
  const nextSelected = { ...selectedCommandIds.value }
  for (const p of projects.value) {
    if (!commandsFor(p).some((c) => c.id === nextSelected[p.id])) nextSelected[p.id] = defaultCommandId(p)
  }
  selectedCommandIds.value = nextSelected
  resubscribeStatus()
}

async function pollStatuses() {
  const next = {}
  for (const p of projects.value) {
    for (const cmd of commandsFor(p)) {
      const runId = runIdForCommand(p.id, cmd.id)
      next[runId] = await GetStatus(runId)
    }
  }
  statuses.value = next
}

async function onAddProject(dir) {
  if (!dir) return
  await AddProject(dir)
  showAddProject.value = false
  await refresh()
}

async function onStart(commandId) { await StartProjectCommand(selectedId.value, commandId); await pollStatuses() }
async function onStop(commandId) { await StopProjectCommand(selectedId.value, commandId); await pollStatuses() }

async function onSaveEdit(payload) {
  await UpdateProjectCommands(selectedId.value, payload.name, payload.commands)
  showEdit.value = false
  await refresh()
  await pollStatuses()
}

async function onSaveGroup(group) {
  await SaveGroup(group)
  showGroup.value = false
  editingGroup.value = null
  await refresh()
}

function onEditGroup(group) {
  editingGroup.value = group
  showGroup.value = true
}

async function onRemoveGroup(id) {
  await RemoveGroup(id)
  if (selectedGroupId.value === id) selectedGroupId.value = ''
  await refresh()
}

async function onStartGroup(id) {
  await StartGroup(id)
  await pollStatuses()
}

async function onStopGroup(id) {
  await StopGroup(id)
  await pollStatuses()
}

async function onAddToGroup(payload) {
  if (!selected.value || !selectedCommand.value) return
  const item = { projectId: selected.value.id, commandId: selectedCommand.value.id || 'default' }
  let group
  if (payload.mode === 'existing') {
    const existing = groups.value.find((g) => g.id === payload.groupId)
    if (!existing) return
    const items = existing.items || []
    const alreadyExists = items.some((i) => i.projectId === item.projectId && i.commandId === item.commandId)
    group = { ...existing, items: alreadyExists ? items : [...items, item] }
  } else {
    group = { id: '', name: payload.groupName, items: [item] }
  }
  await SaveGroup(group)
  showAddToGroup.value = false
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
  <div class="app-shell">
    <header class="topbar">
      <div class="brand">atstarter</div>
      <div class="summary">
        <span>{{ projects.length }} projects</span>
        <span class="summary-pill running">{{ runningCount }} running</span>
        <span class="summary-pill exited">{{ exitedCount }} exited</span>
      </div>
      <div class="top-actions">
        <button class="btn secondary" @click="editingGroup = null; showGroup = true">New Group</button>
        <button class="btn secondary" @click="showScan = true">Scan</button>
        <button class="btn primary" @click="showAddProject = true">Add</button>
      </div>
    </header>
    <main class="workspace">
      <ProjectList :projects="projects" :groups="groups" :selectedId="selectedId" :selectedGroupId="selectedGroupId"
        :statuses="projectStatuses" @select="selectProject" @select-group="selectGroup"
        @select-command="selectCommand" @add="showAddProject = true" @scan="showScan = true" />
      <GroupDetail v-if="selectedGroup" :group="selectedGroup" :projects="projects"
        @start="onStartGroup" @stop="onStopGroup" @edit="onEditGroup" @remove="onRemoveGroup"
        @select-command="selectCommand" />
      <ProjectDetail v-else :project="selected" :status="selectedStatus"
        :selectedCommandId="selectedCommandId" @command-change="setSelectedCommand"
        @start="onStart" @stop="onStop" @edit="showEdit = true" @add-to-group="showAddToGroup = true" />
    </main>
    <EditProjectDialog :show="showEdit" :project="selected"
      @close="showEdit = false" @save="onSaveEdit" />
    <GroupDialog :show="showGroup" :group="editingGroup" :projects="projects"
      @close="showGroup = false; editingGroup = null" @save="onSaveGroup" />
    <AddProjectDialog :show="showAddProject" @close="showAddProject = false" @save="onAddProject" />
    <AddToGroupDialog :show="showAddToGroup" :groups="groups" :project="selected" :command="selectedCommand"
      @close="showAddToGroup = false" @save="onAddToGroup" />
    <ScanDialog :show="showScan" @close="showScan = false" @added="refresh" />
  </div>
</template>

<style>
html, body, #app { height: 100%; margin: 0; }
.app-shell {
  display: grid;
  grid-template-rows: 50px 1fr;
  height: 100vh;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", system-ui, sans-serif;
  background: #f5f6f8;
  color: #182033;
}

.topbar {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: 18px;
  padding: 0 18px;
  background: #ffffff;
  border-bottom: 1px solid #d7dce5;
}

.brand {
  color: #111827;
  font-size: 16px;
  font-weight: 800;
}

.summary {
  display: flex;
  align-items: center;
  min-width: 0;
  flex-wrap: wrap;
  gap: 10px;
  color: #4b5563;
  font-size: 12px;
}

.summary-pill {
  border-radius: 999px;
  padding: 3px 9px;
  font-weight: 700;
}

.summary-pill.running {
  color: #047857;
  background: #ecfdf5;
  border: 1px solid #a7f3d0;
}

.summary-pill.exited {
  color: #b45309;
  background: #fffbeb;
  border: 1px solid #fde68a;
}

.top-actions {
  margin-left: auto;
  display: flex;
  flex-shrink: 0;
  gap: 8px;
}

.btn {
  height: 32px;
  border-radius: 7px;
  padding: 0 12px;
  font: inherit;
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
}

.btn.primary {
  color: #ffffff;
  background: #2563eb;
  border: 1px solid #2563eb;
}

.btn.secondary {
  color: #334155;
  background: #f8fafc;
  border: 1px solid #cbd5e1;
}

.workspace {
  min-height: 0;
  display: flex;
}

@media (max-width: 820px) {
  .topbar {
    gap: 10px;
    padding: 0 12px;
  }

  .summary {
    gap: 6px;
  }

  .btn {
    padding: 0 10px;
  }
}
</style>
