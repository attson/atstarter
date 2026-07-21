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
import AppButton from './components/ui/AppButton.vue'
import AppPill from './components/ui/AppPill.vue'
import AppIcon from './components/ui/AppIcon.vue'
import ThemeToggle from './components/ui/ThemeToggle.vue'
import UpdateBanner from './components/UpdateBanner.vue'
import { FolderPlus, Radar, Plus } from 'lucide-vue-next'
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
const statusFilter = ref(null) // null | 'running' | 'exited'

function toggleStatusFilter(kind) {
  statusFilter.value = statusFilter.value === kind ? null : kind
}

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

// 重启:先停,等旧进程真正退出(避免与旧进程抢端口),再启。Stop 是异步的
// (SIGTERM→退出,或 5s SIGKILL 兜底),故需轮询等状态离开 running。
async function onRestart(commandId) {
  const projectId = selectedId.value
  const runId = runIdForCommand(projectId, commandId)
  await StopProjectCommand(projectId, commandId)
  const deadline = Date.now() + 8000
  while (Date.now() < deadline) {
    const st = await GetStatus(runId)
    if (!st || st.State !== 'running') break
    await new Promise((r) => setTimeout(r, 200))
  }
  await StartProjectCommand(projectId, commandId)
  await pollStatuses()
}

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
    <UpdateBanner />
    <header class="topbar">
      <div class="brand">atstarter</div>
      <div class="summary">
        <span class="summary-count">{{ projects.length }} projects</span>
        <AppPill
          variant="running"
          dot
          clickable
          :active="statusFilter === 'running'"
          @click="toggleStatusFilter('running')"
        >{{ runningCount }} running</AppPill>
        <AppPill
          variant="exited"
          clickable
          :active="statusFilter === 'exited'"
          @click="toggleStatusFilter('exited')"
        >{{ exitedCount }} exited</AppPill>
      </div>
      <div class="top-actions">
        <ThemeToggle />
        <AppButton variant="secondary" size="sm" @click="editingGroup = null; showGroup = true">
          <template #icon><AppIcon :icon="FolderPlus" :size="14" /></template>
          New Group
        </AppButton>
        <AppButton variant="secondary" size="sm" @click="showScan = true">
          <template #icon><AppIcon :icon="Radar" :size="14" /></template>
          Scan
        </AppButton>
        <AppButton variant="primary" size="sm" @click="showAddProject = true">
          <template #icon><AppIcon :icon="Plus" :size="14" /></template>
          Add
        </AppButton>
      </div>
    </header>
    <main class="workspace">
      <ProjectList :projects="projects" :groups="groups" :selectedId="selectedId" :selectedGroupId="selectedGroupId"
        :statuses="projectStatuses" :statusFilter="statusFilter"
        @select="selectProject" @select-group="selectGroup"
        @select-command="selectCommand" @add="showAddProject = true" @scan="showScan = true" />
      <GroupDetail v-if="selectedGroup" :group="selectedGroup" :projects="projects"
        @start="onStartGroup" @stop="onStopGroup" @edit="onEditGroup" @remove="onRemoveGroup"
        @select-command="selectCommand" />
      <ProjectDetail v-else :project="selected" :status="selectedStatus"
        :selectedCommandId="selectedCommandId" @command-change="setSelectedCommand"
        @start="onStart" @stop="onStop" @restart="onRestart" @edit="showEdit = true" @add-to-group="showAddToGroup = true" />
    </main>
  </div>

  <!--
    Dialogs are teleported to <body> so they never become grid items of
    .app-shell. Their collapsed (v-if=false) comment placeholders would
    otherwise shift the grid track assignment under WebKitGTK, squashing
    .workspace into the 48px header row and blanking the project list.
  -->
  <Teleport to="body">
    <EditProjectDialog :show="showEdit" :project="selected"
      @close="showEdit = false" @save="onSaveEdit" />
    <GroupDialog :show="showGroup" :group="editingGroup" :projects="projects"
      @close="showGroup = false; editingGroup = null" @save="onSaveGroup" />
    <AddProjectDialog :show="showAddProject" @close="showAddProject = false" @save="onAddProject" />
    <AddToGroupDialog :show="showAddToGroup" :groups="groups" :project="selected" :command="selectedCommand"
      @close="showAddToGroup = false" @save="onAddToGroup" />
    <ScanDialog :show="showScan" @close="showScan = false" @added="refresh" />
  </Teleport>
</template>

<style>
html, body, #app { height: 100%; margin: 0; }

.app-shell {
  display: grid;
  grid-template-rows: auto 48px 1fr;
  height: 100vh;
  font-family: var(--font-sans);
  background: var(--bg-gradient);
  color: var(--text);
}

/*
 * Pin each region to an explicit grid track. Without this, a collapsed
 * UpdateBanner (rendered as a v-if comment node) can shift track
 * assignment under WebKitGTK, dropping .workspace into the 48px header
 * row. Explicit grid-row makes the layout independent of child order.
 */
.app-shell > .update-banner { grid-row: 1; }
.app-shell > .topbar { grid-row: 2; }
.app-shell > .workspace { grid-row: 3; }

.topbar {
  display: flex;
  align-items: center;
  min-width: 0;
  gap: var(--space-7);
  padding: 0 var(--space-7);
  background: linear-gradient(180deg, rgba(255, 255, 255, .02), transparent);
  border-bottom: 1px solid var(--border);
  box-shadow: var(--surface-highlight);
}

.brand {
  font-size: 17px;
  font-weight: var(--fw-semibold);
  letter-spacing: -0.02em;
  background: var(--brand-gradient);
  -webkit-background-clip: text;
  background-clip: text;
  -webkit-text-fill-color: transparent;
  color: var(--text);
}

.summary {
  display: flex;
  align-items: center;
  min-width: 0;
  flex-wrap: wrap;
  gap: var(--space-4);
  color: var(--text-muted);
  font-size: var(--fs-sm);
}

.summary-count {
  color: var(--text-secondary);
  font-weight: var(--fw-medium);
}

.top-actions {
  margin-left: auto;
  display: flex;
  align-items: center;
  flex-shrink: 0;
  gap: var(--space-4);
}

.workspace {
  min-height: 0;
  display: flex;
}

@media (max-width: 820px) {
  .topbar {
    gap: var(--space-5);
    padding: 0 var(--space-6);
  }
}
</style>
