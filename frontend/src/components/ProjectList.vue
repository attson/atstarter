<script setup>
import { computed, ref, watch } from 'vue'
import { Search } from 'lucide-vue-next'
import { buildProjectTree } from '../projectTree'
import ProjectTreeNode from './ProjectTreeNode.vue'
import GroupTreeItem from './GroupTreeItem.vue'
import AppIcon from './ui/AppIcon.vue'

const emit = defineEmits(['select', 'select-group', 'select-command', 'add', 'scan'])

const props = defineProps({ projects: Array, groups: Array, selectedId: String, selectedGroupId: String, statuses: Object })
const query = ref('')
const expandedDirs = ref({})
const expandedGroups = ref({})

const tree = computed(() => buildProjectTree(props.projects || [], props.statuses || {}, query.value))
const forceExpanded = computed(() => query.value.trim().length > 0)

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

watch(() => props.projects, () => {
  expandedDirs.value = {}
})
</script>

<template>
  <aside class="project-list">
    <div class="search-wrap">
      <div class="search-field">
        <AppIcon :icon="Search" :size="14" class="search-icon" />
        <input v-model="query" class="search" placeholder="Search projects, path, command…" />
      </div>
    </div>
    <div class="tree-scroll">
      <div v-if="(groups || []).length" class="group-section">
        <div class="section-title">Groups</div>
        <GroupTreeItem
          v-for="g in groups"
          :key="g.id"
          :group="g"
          :projects="projects"
          :selected="g.id === selectedGroupId"
          :expanded="expandedGroups[`group:${g.id}`] !== false"
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
        @select="emit('select', $event)"
        @toggle="toggleDir"
      />
      <div v-if="tree.length === 0" class="empty">
        <span v-if="query">没有匹配的项目</span>
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
  background: var(--surface);
  min-height: 0;
}

.search-wrap {
  padding: var(--space-4) var(--space-5);
  border-bottom: 1px solid var(--border);
}

.search-field {
  position: relative;
  display: flex;
  align-items: center;
}

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
  height: 28px;
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  background: var(--elevated);
  color: var(--text);
  padding: 0 var(--space-4) 0 30px;
  font: inherit;
  font-size: var(--fs-sm);
  outline: none;
  transition: border-color var(--dur-fast) var(--ease), box-shadow var(--dur-fast) var(--ease);
}

.search:focus {
  border-color: var(--border-strong);
  box-shadow: 0 0 0 3px var(--focus-ring);
}

.tree-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: var(--space-3) var(--space-4);
}

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
