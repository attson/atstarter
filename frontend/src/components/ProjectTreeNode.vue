<script setup>
import { ChevronRight, ChevronDown } from 'lucide-vue-next'
import AppIcon from './ui/AppIcon.vue'

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
      :style="{ paddingLeft: `${6 + level * 12}px` }"
      @click="emit('toggle', node.id)"
    >
      <span class="chev">
        <AppIcon :icon="isExpanded(node) ? ChevronDown : ChevronRight" :size="12" />
      </span>
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
      :style="{ paddingLeft: `${4 + level * 12}px` }"
      role="button"
      tabindex="0"
      @click="emit('select', node.project.id)"
      @keydown.enter.prevent="emit('select', node.project.id)"
      @keydown.space.prevent="emit('select', node.project.id)"
    >
      <button
        v-if="hasChildren(node)"
        class="project-toggle"
        @click.stop="emit('toggle', node.id)"
      >
        <AppIcon :icon="isExpanded(node) ? ChevronDown : ChevronRight" :size="12" />
      </button>
      <span v-else class="project-spacer" />
      <span class="project-main">
        <span :class="['status-dot', stateClass((node.status || {}).State)]" />
        <span class="project-name">{{ node.project.name }}</span>
        <span v-if="!hasChildren(node)" class="type-pill">{{ node.project.detectedType || 'unknown' }}</span>
      </span>
      <span v-if="hasChildren(node)" class="count">{{ node.count }}</span>
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
.tree-group { min-width: 0; }

.tree-row {
  width: 100%;
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: background var(--dur-fast) var(--ease);
}

.dir-row {
  display: grid;
  grid-template-columns: 14px minmax(0, 1fr) auto;
  align-items: center;
  height: 26px;
  gap: var(--space-2);
  color: var(--text-secondary);
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
  border-radius: var(--radius-sm);
  padding-right: var(--space-3);
}

.dir-row:hover { background: var(--elevated-gradient); }

.chev, .count { color: var(--text-muted); }

.count {
  font-weight: var(--fw-regular);
  font-size: var(--fs-xs);
  padding-right: var(--space-3);
}

.dir-name, .project-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.children { position: relative; }

.project-row {
  display: grid;
  grid-template-columns: 12px minmax(0, 1fr) auto;
  align-items: center;
  gap: var(--space-2);
  min-height: 28px;
  margin: 1px 0;
  padding-top: 2px;
  padding-bottom: 2px;
  padding-right: var(--space-4);
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  cursor: pointer;
  transition: background var(--dur-fast) var(--ease), box-shadow var(--dur-fast) var(--ease);
}

.project-row:focus-visible {
  outline: 0;
  box-shadow: 0 0 0 2px var(--focus-ring);
}

.project-row:hover { background: var(--elevated-gradient); }

.project-row.active {
  background: var(--elevated-gradient);
  color: var(--text);
  box-shadow: inset 0 0 0 1px var(--border-strong), var(--surface-highlight);
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.status-dot.running {
  background: var(--accent-strong);
  animation: pulse-ring 2s ease-in-out infinite;
}

.status-dot.bad {
  background: var(--danger);
  box-shadow: 0 0 0 2.5px var(--danger-soft), 0 0 6px rgba(239, 68, 68, .4);
}
.status-dot.stopped { background: var(--text-subtle); }

.project-toggle {
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  cursor: pointer;
}

.project-main {
  min-width: 0;
  display: flex;
  align-items: center;
  gap: var(--space-3);
  color: inherit;
  font: inherit;
}

.project-main .status-dot { flex: 0 0 auto; }
.project-main .project-name { flex: 0 1 auto; }
.project-main .type-pill { flex: 0 0 auto; }

.project-toggle {
  width: 12px;
  padding: 0;
  color: var(--text-muted);
  font-size: var(--fs-xs);
}

.project-spacer { width: 12px; }

.project-name {
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}

.type-pill {
  flex-shrink: 0;
  white-space: nowrap;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-full);
  color: var(--text-muted);
  background: transparent;
  padding: 1px 7px;
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
}
</style>
