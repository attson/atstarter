<script setup>
const props = defineProps({
  node: Object,
  selectedId: String,
  level: Number,
  expandedDirs: Object,
  forceExpanded: Boolean,
})
const emit = defineEmits(['select', 'toggle'])

function stateClass(state) {
  if (state === 'running') return 'running'
  if (state === 'error' || state === 'exited') return 'bad'
  return 'stopped'
}

function isExpanded(node) {
  return props.forceExpanded || props.expandedDirs[node.id] !== false
}

function hasChildren(node) {
  return node.children && node.children.length > 0
}
</script>

<template>
  <div v-if="node.type === 'directory'" class="tree-group">
    <button
      class="tree-row dir-row"
      :style="{ paddingLeft: `${10 + level * 16}px` }"
      @click="emit('toggle', node.id)"
    >
      <span class="chevron">{{ isExpanded(node) ? '▾' : '▸' }}</span>
      <span class="dir-name">{{ node.label }}</span>
      <span class="count">{{ node.count }}</span>
    </button>
    <div v-if="isExpanded(node)" class="children">
      <ProjectTreeNode
        v-for="child in node.children"
        :key="child.id"
        :node="child"
        :selectedId="selectedId"
        :level="level + 1"
        :expandedDirs="expandedDirs"
        :forceExpanded="forceExpanded"
        @select="emit('select', $event)"
        @toggle="emit('toggle', $event)"
      />
    </div>
  </div>

  <div v-else class="tree-group">
    <div
      :class="['tree-row', 'project-row', { active: node.project.id === selectedId }]"
      :style="{ paddingLeft: `${12 + level * 16}px` }"
    >
      <button
        v-if="hasChildren(node)"
        class="project-toggle"
        @click.stop="emit('toggle', node.id)"
      >
        {{ isExpanded(node) ? '▾' : '▸' }}
      </button>
      <span v-else class="project-spacer"></span>
      <button class="project-main" @click="emit('select', node.project.id)">
        <span :class="['status-dot', stateClass((node.status || {}).State)]"></span>
        <span class="project-copy">
          <span class="project-name">{{ node.project.name }}</span>
        </span>
      </button>
      <span v-if="hasChildren(node)" class="count">{{ node.count }}</span>
      <span v-else class="type-pill">{{ node.project.detectedType || 'unknown' }}</span>
    </div>
    <div v-if="hasChildren(node) && isExpanded(node)" class="children">
      <ProjectTreeNode
        v-for="child in node.children"
        :key="child.id"
        :node="child"
        :selectedId="selectedId"
        :level="level + 1"
        :expandedDirs="expandedDirs"
        :forceExpanded="forceExpanded"
        @select="emit('select', $event)"
        @toggle="emit('toggle', $event)"
      />
    </div>
  </div>
</template>

<style scoped>
.tree-group {
  min-width: 0;
}

.tree-row {
  width: 100%;
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.dir-row {
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr) auto;
  align-items: center;
  height: 26px;
  gap: 4px;
  color: #334155;
  font-size: 12px;
  font-weight: 700;
  border-radius: 6px;
}

.dir-row:hover {
  background: #f1f5f9;
}

.chevron,
.count {
  color: #64748b;
}

.count {
  font-weight: 600;
  font-size: 11px;
  padding-right: 7px;
}

.dir-name,
.project-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.children {
  position: relative;
}

.project-row {
  display: grid;
  grid-template-columns: 14px minmax(0, 1fr) auto;
  align-items: center;
  gap: 6px;
  min-height: 31px;
  margin: 1px 0;
  padding-top: 2px;
  padding-bottom: 2px;
  padding-right: 8px;
  border-radius: 6px;
  color: #1f2937;
}

.project-row:hover {
  background: #f8fafc;
}

.project-row.active {
  background: #eef4ff;
  box-shadow: inset 0 0 0 1px #bfdbfe;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

.status-dot.running {
  background: #16a34a;
  box-shadow: 0 0 0 3px #dcfce7;
}

.status-dot.bad {
  background: #ef4444;
  box-shadow: 0 0 0 3px #fee2e2;
}

.status-dot.stopped {
  background: #94a3b8;
}

.project-toggle,
.project-main {
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  cursor: pointer;
}

.project-toggle {
  width: 14px;
  padding: 0;
  color: #64748b;
  font-size: 11px;
}

.project-spacer {
  width: 14px;
}

.project-main {
  min-width: 0;
  display: grid;
  grid-template-columns: 12px minmax(0, 1fr);
  align-items: center;
  gap: 8px;
  padding: 0;
  text-align: left;
}

.project-copy {
  display: flex;
  align-items: center;
  min-width: 0;
}

.project-name {
  font-size: 13px;
  font-weight: 700;
}

.type-pill {
  max-width: 72px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  border: 1px solid #dbeafe;
  border-radius: 999px;
  color: #2563eb;
  background: #ffffff;
  padding: 1px 7px;
  font-size: 11px;
  font-weight: 700;
}
</style>
