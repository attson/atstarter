<script setup>
import { computed, ref, watch } from 'vue'
import { buildProjectTree } from '../projectTree'
import ProjectTreeNode from './ProjectTreeNode.vue'
import GroupTreeItem from './GroupTreeItem.vue'

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
      <input v-model="query" class="search" placeholder="Search projects, path, command..." />
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
  width: 348px;
  min-width: 300px;
  border-right: 1px solid #d7dce5;
  display: flex;
  flex-direction: column;
  background: #ffffff;
  min-height: 0;
}

.search-wrap {
  padding: 9px 10px;
  border-bottom: 1px solid #e5e7eb;
}

.section-title {
  color: #64748b;
  font-size: 11px;
  font-weight: 800;
  text-transform: uppercase;
  margin: 2px 2px 5px;
}

.search {
  width: 100%;
  box-sizing: border-box;
  height: 30px;
  border: 1px solid #cbd5e1;
  border-radius: 7px;
  background: #f8fafc;
  color: #0f172a;
  padding: 0 11px;
  font-size: 13px;
  outline: none;
}

.search:focus {
  border-color: #2563eb;
  box-shadow: 0 0 0 3px #dbeafe;
  background: #ffffff;
}

.tree-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 6px 8px;
}

.group-section {
  margin-bottom: 8px;
  padding-bottom: 7px;
  border-bottom: 1px solid #e5e7eb;
}

.empty {
  color: #64748b;
  font-size: 13px;
  padding: 16px 10px;
}
</style>
