<script setup>
import { computed, ref, watch } from 'vue'
import { LayoutGrid, List as ListIcon, RefreshCw, Search, Trash2 } from 'lucide-vue-next'
import { buildProjectTree, buildPinnedList } from '../projectTree'
import { GetProjectBranch } from '../../wailsjs/go/main/App'
import {
  advanceBranchRevision,
  applyBranchEntry,
  loadVisibleProjectBranches,
  readProjectCardFields,
  readProjectListView,
  writeProjectCardFields,
  writeProjectListView,
} from '../projectNavigation.js'
import ProjectTreeNode from './ProjectTreeNode.vue'
import GroupTreeItem from './GroupTreeItem.vue'
import ProjectCardFieldMenu from './ProjectCardFieldMenu.vue'
import AppIcon from './ui/AppIcon.vue'
import AppButton from './ui/AppButton.vue'

const emit = defineEmits(['select', 'select-group', 'select-command', 'add', 'scan', 'rescan', 'reset', 'context-menu', 'reorder-pinned'])

const props = defineProps({
  projects: Array,
  groups: Array,
  selectedId: String,
  selectedGroupId: String,
  statuses: Object,
  statusFilter: { type: String, default: null },
  rescanning: { type: Boolean, default: false },
  missingIds: { type: Set, default: () => new Set() },
  branchUpdate: { type: Object, default: null },
})
const storage = typeof localStorage === 'undefined' ? null : localStorage
const query = ref('')
const expandedDirs = ref({})
const expandedGroups = ref({})
const viewMode = ref(readProjectListView(storage))
const cardFields = ref(readProjectCardFields(storage))
const cardFieldMenu = ref(null)
const branches = ref({})
let branchLoadToken = 0
let branchRevisions = {}

const filteredProjects = computed(() => {
  const list = props.projects || []
  if (!props.statusFilter) return list
  const statuses = props.statuses || {}
  return list.filter((p) => {
    const state = (statuses[p.id] || {}).State
    if (props.statusFilter === 'running') return state === 'running'
    if (props.statusFilter === 'exited') return state === 'exited' || state === 'error'
    return true
  })
})

const tree = computed(() => buildProjectTree(filteredProjects.value, props.statuses || {}, query.value))
const pinned = computed(() => buildPinnedList(filteredProjects.value, props.statuses || {}, query.value))
const forceExpanded = computed(() => query.value.trim().length > 0 || !!props.statusFilter)

const dragIndex = ref(-1)

function onDragStart(i) { dragIndex.value = i }
function onDrop(i) {
  const from = dragIndex.value
  dragIndex.value = -1
  if (from === -1 || from === i) return
  const ids = pinned.value.map((n) => n.project.id)
  const [moved] = ids.splice(from, 1)
  ids.splice(i, 0, moved)
  emit('reorder-pinned', ids)
}

function toggleDir(id) {
  expandedDirs.value = {
    ...expandedDirs.value,
    [id]: expandedDirs.value[id] === false,
  }
}

function toggleGroup(id) {
  expandedGroups.value = {
    ...expandedGroups.value,
    [id]: expandedGroups.value[id] === false,
  }
}

function setViewMode(mode) {
  cardFieldMenu.value = null
  viewMode.value = writeProjectListView(storage, mode)
}

function openCardFieldMenu(event) {
  cardFieldMenu.value = { x: event.clientX, y: event.clientY }
}

function toggleCardField(field) {
  cardFields.value = writeProjectCardFields(storage, {
    ...cardFields.value,
    [field]: !cardFields.value[field],
  })
  if (field !== 'branch') return
  branchLoadToken += 1
  if (!cardFields.value.branch) {
    branches.value = {}
    return
  }
  if (viewMode.value === 'card') void refreshBranches(props.projects || [])
}

async function refreshBranches(projects) {
  if (viewMode.value !== 'card' || !cardFields.value.branch) return
  const token = ++branchLoadToken
  const loading = {}
  for (const project of projects || []) {
    if (project?.id) loading[project.id] = { state: 'loading', branch: '' }
  }
  const revisionsAtStart = Object.fromEntries(
    (projects || []).filter((project) => project?.id).map((project) => [project.id, branchRevisions[project.id] || 0]),
  )
  branches.value = loading
  await loadVisibleProjectBranches(
    projects,
    viewMode.value,
    cardFields.value,
    GetProjectBranch,
    (id, entry) => {
      branches.value = applyBranchEntry(branches.value, branchRevisions, id, entry, revisionsAtStart[id])
    },
    () => token === branchLoadToken && viewMode.value === 'card' && cardFields.value.branch,
  )
}

watch(() => props.projects, () => {
  expandedDirs.value = {}
})

watch([() => props.projects, viewMode], ([projects]) => {
  if (viewMode.value === 'card') void refreshBranches(projects || [])
}, { immediate: true })

watch(() => props.branchUpdate, (update) => {
  if (!update?.projectId || viewMode.value !== 'card' || !cardFields.value.branch) return
  branchRevisions = advanceBranchRevision(branchRevisions, update.projectId)
  branches.value = applyBranchEntry(
    branches.value,
    branchRevisions,
    update.projectId,
    { state: 'ready', branch: String(update.branch || '').trim() },
  )
})
</script>

<template>
  <aside class="project-list">
    <div class="search-wrap">
      <div class="search-field">
        <AppIcon :icon="Search" :size="14" class="search-icon" />
        <input v-model="query" class="search" placeholder="Search projects, path, command…" />
        <AppButton
          variant="secondary"
          size="sm"
          icon-only
          class="rescan-btn"
          :class="{ spinning: rescanning }"
          title="重新扫描工作区"
          :disabled="rescanning"
          @click="emit('rescan')"
        >
          <template #icon><AppIcon :icon="RefreshCw" :size="14" /></template>
        </AppButton>
        <AppButton
          variant="danger"
          size="sm"
          icon-only
          class="reset-btn"
          title="重置项目列表"
          :disabled="!(projects || []).length"
          @click="emit('reset')"
        >
          <template #icon><AppIcon :icon="Trash2" :size="14" /></template>
        </AppButton>
      </div>
      <div class="view-switch" role="group" aria-label="项目导航显示方式">
        <button
          class="view-button"
          :class="{ active: viewMode === 'list' }"
          type="button"
          title="列表视图"
          aria-label="列表视图"
          :aria-pressed="viewMode === 'list'"
          data-test="project-view-list"
          @click="setViewMode('list')"
        >
          <AppIcon :icon="ListIcon" :size="13" />
        </button>
        <button
          class="view-button"
          :class="{ active: viewMode === 'card' }"
          type="button"
          title="卡片视图；右键配置字段"
          aria-label="卡片视图；右键配置字段"
          :aria-pressed="viewMode === 'card'"
          data-test="project-view-card"
          @click="setViewMode('card')"
          @contextmenu.prevent.stop="openCardFieldMenu"
        >
          <AppIcon :icon="LayoutGrid" :size="13" />
        </button>
      </div>
      <ProjectCardFieldMenu
        v-if="cardFieldMenu"
        :x="cardFieldMenu.x"
        :y="cardFieldMenu.y"
        :fields="cardFields"
        @toggle="toggleCardField"
        @close="cardFieldMenu = null"
      />
    </div>
    <div class="tree-scroll">
      <div v-if="pinned.length" class="pinned-section">
        <div class="section-title">Pinned</div>
        <div
          v-for="(node, i) in pinned"
          :key="node.id"
          class="pinned-item"
          draggable="true"
          @dragstart="onDragStart(i)"
          @dragover.prevent
          @drop="onDrop(i)"
        >
          <ProjectTreeNode
            :node="node"
            :selectedId="selectedId"
            :level="0"
            :expandedDirs="expandedDirs"
            :forceExpanded="false"
            :missingIds="missingIds"
            :viewMode="viewMode"
            :branches="branches"
            :card-fields="cardFields"
            @select="emit('select', $event)"
            @toggle="toggleDir"
            @context-menu="emit('context-menu', $event)"
          />
        </div>
      </div>
      <div v-if="(groups || []).length" class="group-section">
        <div class="section-title">Groups</div>
        <GroupTreeItem
          v-for="g in groups"
          :key="g.id"
          :group="g"
          :projects="projects"
          :statuses="statuses"
          :selected="g.id === selectedGroupId"
          :expanded="expandedGroups[`group:${g.id}`] !== false"
          :viewMode="viewMode"
          :branches="branches"
          :card-fields="cardFields"
          @select="emit('select-group', $event)"
          @toggle="toggleGroup"
          @select-command="emit('select-command', $event)"
        />
      </div>
      <ProjectTreeNode
        v-for="node in tree"
        :key="node.id"
        :node="node"
        :selectedId="selectedId"
        :level="0"
        :expandedDirs="expandedDirs"
        :forceExpanded="forceExpanded"
        :missingIds="missingIds"
        :viewMode="viewMode"
        :branches="branches"
        :card-fields="cardFields"
        @select="emit('select', $event)"
        @toggle="toggleDir"
        @context-menu="emit('context-menu', $event)"
      />
      <div v-if="tree.length === 0" class="empty">
        <span v-if="query">没有匹配的项目</span>
        <span v-else-if="statusFilter === 'running'">当前没有运行中的项目</span>
        <span v-else-if="statusFilter === 'exited'">当前没有已退出/错误的项目</span>
        <span v-else>还没有项目。点击 Add 或 Scan 开始。</span>
      </div>
    </div>
  </aside>
</template>

<style scoped>
.project-list {
  width: 300px;
  min-width: 280px;
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  background: linear-gradient(180deg, rgba(255, 255, 255, .015), transparent), var(--surface);
  box-shadow: var(--surface-highlight);
  min-height: 0;
}

.search-wrap {
  padding: var(--space-4) var(--space-5);
  border-bottom: 1px solid var(--border);
}

.search-field {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto;
  gap: var(--space-3);
  align-items: center;
  position: relative;
}

.view-switch {
  width: fit-content;
  display: grid;
  grid-template-columns: 28px 28px;
  margin: var(--space-3) 0 0 auto;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  overflow: hidden;
  background: var(--surface);
  box-shadow: var(--surface-highlight);
}

.view-button {
  height: 26px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-right: 1px solid var(--border-strong);
  background: transparent;
  color: var(--text-muted);
  cursor: pointer;
}
.view-button:last-child { border-right: 0; }
.view-button:hover { color: var(--text); background: var(--elevated); }
.view-button.active { color: var(--accent); background: var(--success-soft); }
.view-button:focus-visible { outline: 0; box-shadow: inset 0 0 0 2px var(--focus-ring); }

.search-icon {
  position: absolute;
  left: var(--space-4);
  color: var(--text-muted);
  pointer-events: none;
}

.section-title {
  color: var(--text-subtle);
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  letter-spacing: 0.03em;
  text-transform: uppercase;
  margin: var(--space-2) var(--space-2) var(--space-3);
}

.search {
  width: 100%;
  box-sizing: border-box;
  height: 30px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: var(--elevated-gradient);
  color: var(--text);
  padding: 0 var(--space-4) 0 32px;
  font: inherit;
  font-size: var(--fs-sm);
  outline: none;
  box-shadow: var(--surface-highlight);
  transition: border-color var(--dur-fast) var(--ease), box-shadow var(--dur-fast) var(--ease);
}

.rescan-btn,
.reset-btn { flex: 0 0 auto; }
.rescan-btn.spinning :deep(.app-icon) { animation: project-rescan-spin .9s linear infinite; }
@keyframes project-rescan-spin {
  to { transform: rotate(360deg); }
}

.search:focus {
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--focus-ring), var(--surface-highlight);
}

.tree-scroll {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: var(--space-3) var(--space-4);
}

.pinned-section {
  margin-bottom: var(--space-4);
  padding-bottom: var(--space-3);
  border-bottom: 1px solid var(--border);
}
.pinned-item[draggable="true"] { cursor: grab; }
.pinned-item:active { cursor: grabbing; }

.group-section {
  margin-bottom: var(--space-4);
  padding-bottom: var(--space-3);
  border-bottom: 1px solid var(--border);
}

.empty {
  color: var(--text-muted);
  font-size: var(--fs-sm);
  padding: var(--space-7) var(--space-5);
}
</style>
