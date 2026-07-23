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
import ComposeDetail from './components/ComposeDetail.vue'
import ContainerPanel from './components/ContainerPanel.vue'
import ConfirmDialog from './components/ConfirmDialog.vue'
import { FolderPlus, Radar, Plus, RefreshCw } from 'lucide-vue-next'
import {
  ListProjects, AddProject, StartProjectCommand, StopProjectCommand,
  GetStatus, UpdateProjectCommands, ListGroups, SaveGroup, RemoveGroup,
  StartGroup, StopGroup, GetWorkspaces, SetWorkspaces, ScanWorkspaces, AddScanned,
  UpdateProject, ResetProjects,
} from '../wailsjs/go/main/App'
import { ComposeDown, RemoveContainer, DockerAvailable } from '../wailsjs/go/main/App'
import { EventsOn, EventsOff } from '../wailsjs/runtime/runtime'
import { inferWorkspaceRoots } from './workspaceRoots'
import { applyDetectionOption } from './projectDetection.js'

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
const rescanning = ref(false)

const activeTab = ref('projects') // 'projects' | 'containers'
const dockerAvailable = ref(true)
const containerSummary = ref({ total: 0, running: 0, exited: 0 })
const confirm = ref({ show: false, kind: '', payload: null, title: '', message: '', confirmText: '', danger: true })
const containerPanelRef = ref(null)
const updateBannerRef = ref(null)

const isComposeSelected = computed(() => selected.value && selected.value.detectedType === 'compose')

async function checkDocker() {
  const info = await DockerAvailable()
  dockerAvailable.value = info.available
}

function onConfirmDown(projectId) {
  confirm.value = { show: true, kind: 'down', payload: projectId, title: 'Compose Down', message: '将停止并删除该 compose 项目的所有容器与网络(数据卷保留)。确认继续?', confirmText: 'Down', danger: true }
}
function onConfirmRemove(container) {
  const running = container.state === 'running'
  confirm.value = { show: true, kind: 'remove', payload: container, title: '删除容器', message: running ? `容器「${container.name}」正在运行,将强制删除(rm -f)。确认?` : `删除容器「${container.name}」?`, confirmText: '删除', danger: true }
}
function onConfirmResetProjects() {
  confirm.value = { show: true, kind: 'reset-projects', payload: null, title: '重置项目列表', message: '将清空所有项目和启动分组,保留扫描工作区。正在运行的普通项目会先停止。确认继续?', confirmText: '重置', danger: true }
}
function onContainerSummary(summary) {
  containerSummary.value = summary || { total: 0, running: 0, exited: 0 }
}
async function onConfirmAccept() {
  const { kind, payload } = confirm.value
  try {
    if (kind === 'down') await ComposeDown(payload)
    else if (kind === 'remove') await RemoveContainer(payload.id, payload.state === 'running')
    else if (kind === 'reset-projects') {
      await ResetProjects()
      selectedId.value = ''
      selectedGroupId.value = ''
      selectedCommandIds.value = {}
      statuses.value = {}
      await refresh()
    }
  } catch (e) {
    console.error('操作失败:', e)
  } finally {
    confirm.value = { ...confirm.value, show: false }
  }
  if (kind === 'remove' && containerPanelRef.value) await containerPanelRef.value.refresh()
}

function toggleStatusFilter(kind) {
  if (activeTab.value !== 'projects') return
  statusFilter.value = statusFilter.value === kind ? null : kind
}

async function onCheckUpdates() {
  await updateBannerRef.value?.check({ notify: true })
}

const selected = computed(() => projects.value.find((p) => p.id === selectedId.value))
const selectedGroup = computed(() => groups.value.find((g) => g.id === selectedGroupId.value))
const selectedCommandId = computed(() => selectedCommandIds.value[selectedId.value] || defaultCommandId(selected.value))
const selectedRunId = computed(() => selected.value ? runIdForCommand(selected.value.id, selectedCommandId.value) : '')
const selectedCommand = computed(() => commandsFor(selected.value).find((c) => c.id === selectedCommandId.value) || commandsFor(selected.value)[0])
const selectedStatus = computed(() => statuses.value[selectedRunId.value])
const runningCount = computed(() => Object.values(statuses.value).filter((s) => s && s.State === 'running').length)
const exitedCount = computed(() => Object.values(statuses.value).filter((s) => s && (s.State === 'exited' || s.State === 'error')).length)
const summaryTotalLabel = computed(() => activeTab.value === 'containers'
  ? `${containerSummary.value.total} containers`
  : `${projects.value.length} projects`)
const summaryRunningCount = computed(() => activeTab.value === 'containers' ? containerSummary.value.running : runningCount.value)
const summaryExitedCount = computed(() => activeTab.value === 'containers' ? containerSummary.value.exited : exitedCount.value)
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
  if (selectedId.value && !projects.value.some((p) => p.id === selectedId.value)) selectedId.value = ''
  if (selectedGroupId.value && !groups.value.some((g) => g.id === selectedGroupId.value)) selectedGroupId.value = ''
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

async function onRescanProjects() {
  if (rescanning.value) return
  rescanning.value = true
  try {
    let roots = (await GetWorkspaces()) || []
    if (!roots.length) roots = inferWorkspaceRoots(projects.value)
    if (!roots.length) {
      showScan.value = true
      return
    }
    await SetWorkspaces(roots)
    const candidates = (await ScanWorkspaces(roots)) || []
    const detected = candidates.filter((p) => p.detectedType !== 'unknown')
    if (detected.length) await AddScanned(detected)
    await refresh()
    await pollStatuses()
  } catch (e) {
    console.error('重新扫描失败:', e)
  } finally {
    rescanning.value = false
  }
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

async function onSwitchDetection(option) {
  if (!selected.value) return
  await UpdateProject(applyDetectionOption(selected.value, option))
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
  await checkDocker()
  timer = setInterval(pollStatuses, 1500)
})
onUnmounted(() => {
  clearInterval(timer)
  statusSubs.forEach((ev) => EventsOff(ev))
})
</script>

<template>
  <div class="app-shell">
    <UpdateBanner ref="updateBannerRef" />
    <header class="topbar">
      <div class="brand">atstarter</div>
      <div class="summary">
        <span class="summary-count">{{ summaryTotalLabel }}</span>
        <AppPill
          variant="running"
          dot
          :clickable="activeTab === 'projects'"
          :active="activeTab === 'projects' && statusFilter === 'running'"
          @click="toggleStatusFilter('running')"
        >{{ summaryRunningCount }} running</AppPill>
        <AppPill
          variant="exited"
          :clickable="activeTab === 'projects'"
          :active="activeTab === 'projects' && statusFilter === 'exited'"
          @click="toggleStatusFilter('exited')"
        >{{ summaryExitedCount }} exited</AppPill>
      </div>
      <div class="tabs">
        <button class="tab" :class="{ active: activeTab === 'projects' }" @click="activeTab = 'projects'">Projects</button>
        <button class="tab" :class="{ active: activeTab === 'containers' }" @click="activeTab = 'containers'">Containers</button>
      </div>
      <div class="top-actions">
        <ThemeToggle />
        <AppButton variant="secondary" size="sm" icon-only title="检查更新" aria-label="检查更新" @click="onCheckUpdates">
          <template #icon><AppIcon :icon="RefreshCw" :size="14" /></template>
        </AppButton>
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
      <template v-if="activeTab === 'projects'">
        <ProjectList :projects="projects" :groups="groups" :selectedId="selectedId" :selectedGroupId="selectedGroupId"
          :statuses="projectStatuses" :statusFilter="statusFilter" :rescanning="rescanning"
          @select="selectProject" @select-group="selectGroup"
          @select-command="selectCommand" @add="showAddProject = true" @scan="showScan = true"
          @rescan="onRescanProjects" @reset="onConfirmResetProjects" />
        <GroupDetail v-if="selectedGroup" :group="selectedGroup" :projects="projects"
          @start="onStartGroup" @stop="onStopGroup" @edit="onEditGroup" @remove="onRemoveGroup"
          @select-command="selectCommand" />
        <ComposeDetail v-else-if="isComposeSelected" :project="selected" :dockerAvailable="dockerAvailable"
          @confirm-down="onConfirmDown" @switch-type="onSwitchDetection" />
        <ProjectDetail v-else :project="selected" :status="selectedStatus"
          :selectedCommandId="selectedCommandId" @command-change="setSelectedCommand"
          @start="onStart" @stop="onStop" @restart="onRestart" @edit="showEdit = true"
          @add-to-group="showAddToGroup = true" @switch-type="onSwitchDetection" />
      </template>
      <ContainerPanel v-else ref="containerPanelRef" :projects="projects" @confirm-remove="onConfirmRemove" @summary="onContainerSummary" />
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
    <ScanDialog :show="showScan" :projects="projects" @close="showScan = false" @added="refresh" />
    <ConfirmDialog :show="confirm.show" :title="confirm.title" :message="confirm.message"
      :confirmText="confirm.confirmText" :danger="confirm.danger"
      @close="confirm = { ...confirm, show: false }" @confirm="onConfirmAccept" />
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

.tabs { display: flex; gap: var(--space-2); }
.tab { height: 28px; padding: 0 var(--space-5); border: 1px solid transparent; border-radius: var(--radius-sm); background: transparent; color: var(--text-muted); font: inherit; font-size: var(--fs-sm); font-weight: var(--fw-medium); cursor: pointer; }
.tab.active { background: var(--elevated); color: var(--text); border-color: var(--border-strong); }

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
