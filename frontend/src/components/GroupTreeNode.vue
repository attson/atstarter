<script setup>
import { ChevronDown, ChevronRight } from 'lucide-vue-next'
import AppIcon from './ui/AppIcon.vue'
import { typeLabel } from '../typeLabel.js'

const props = defineProps({
  node: Object,
  level: Number,
  checked: Object,
  expandedDirs: Object,
  forceExpanded: Boolean,
})
const emit = defineEmits(['toggle', 'toggle-dir'])

// 目录默认展开;expandedDirs[id] === false 表示用户手动折叠。
// 搜索态(forceExpanded)下一律展开。
function isExpanded(node) {
  return props.forceExpanded || props.expandedDirs[node.id] !== false
}

function commandsFor(project) {
  if (project.commands && project.commands.length) return project.commands
  return [{
    id: 'default',
    name: 'Default',
    command: project.command,
    args: project.args || [],
    isDefault: true,
  }]
}

function keyFor(projectId, commandId) {
  return `${projectId}:${commandId || 'default'}`
}

function lineFor(command) {
  return [command.command, ...(command.args || [])].filter(Boolean).join(' ')
}
</script>

<template>
  <div v-if="node.type === 'directory'" class="group-node">
    <button
      type="button"
      class="dir-row"
      :style="{ paddingLeft: `${10 + level * 16}px` }"
      @click="emit('toggle-dir', node.id)"
    >
      <span class="chevron">
        <AppIcon :icon="isExpanded(node) ? ChevronDown : ChevronRight" :size="12" />
      </span>
      <span class="dir-name">{{ node.label }}</span>
      <span class="count">{{ node.count }}</span>
    </button>
    <template v-if="isExpanded(node)">
      <GroupTreeNode
        v-for="child in node.children"
        :key="child.id"
        :node="child"
        :level="level + 1"
        :checked="checked"
        :expandedDirs="expandedDirs"
        :forceExpanded="forceExpanded"
        @toggle="emit('toggle', $event)"
        @toggle-dir="emit('toggle-dir', $event)"
      />
    </template>
  </div>

  <div v-else class="group-node">
    <div class="project-row" :style="{ paddingLeft: `${12 + level * 16}px` }">
      <span class="project-dot"></span>
      <span class="project-name">{{ node.project.name }}</span>
      <span v-if="node.children && node.children.length" class="count">{{ node.count }}</span>
      <span v-else class="type-pill">{{ typeLabel(node.project.detectedType) }}</span>
    </div>
    <div class="command-rows">
      <button
        v-for="command in commandsFor(node.project)"
        :key="keyFor(node.project.id, command.id)"
        :class="['command-row', { selected: checked[keyFor(node.project.id, command.id)] }]"
        :style="{ paddingLeft: `${34 + level * 16}px` }"
        @click="emit('toggle', keyFor(node.project.id, command.id))"
      >
        <span class="check-mark">{{ checked[keyFor(node.project.id, command.id)] ? '✓' : '' }}</span>
        <span class="command-name">{{ command.name }}</span>
        <code>{{ lineFor(command) }}</code>
      </button>
    </div>
    <GroupTreeNode
      v-for="child in node.children"
      :key="child.id"
      :node="child"
      :level="level + 1"
      :checked="checked"
      :expandedDirs="expandedDirs"
      :forceExpanded="forceExpanded"
      @toggle="emit('toggle', $event)"
      @toggle-dir="emit('toggle-dir', $event)"
    />
  </div>
</template>

<style scoped>
.group-node {
  min-width: 0;
}

.dir-row,
.project-row {
  display: grid;
  align-items: center;
  min-height: 28px;
  gap: var(--space-3);
}

.dir-row {
  width: 100%;
  grid-template-columns: 16px minmax(0, 1fr) auto;
  color: var(--text-secondary);
  background: var(--elevated);
  border: 0;
  border-bottom: 1px solid var(--border);
  font: inherit;
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
  text-align: left;
  cursor: pointer;
  transition: background var(--dur-fast) var(--ease);
}

.dir-row:hover { background: var(--border); }

.project-row {
  grid-template-columns: 10px minmax(0, 1fr) auto;
  color: var(--text);
  border-bottom: 1px solid var(--border);
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
}

.chevron,
.count {
  color: var(--text-muted);
}

.project-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--text-subtle);
}

.dir-name,
.project-name,
.command-name,
.command-row code {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.type-pill {
  margin-right: var(--space-4);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-full);
  color: var(--text-muted);
  background: transparent;
  padding: 1px 7px;
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
}

.command-row {
  width: 100%;
  display: grid;
  grid-template-columns: 18px minmax(90px, 140px) minmax(0, 1fr);
  align-items: center;
  gap: var(--space-4);
  min-height: 30px;
  border: 0;
  border-bottom: 1px solid var(--border);
  background: transparent;
  color: var(--text-secondary);
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: background var(--dur-fast) var(--ease);
}

.command-row:hover { background: var(--elevated); }

.command-row.selected {
  background: var(--elevated);
}

.check-mark {
  width: 16px;
  height: 16px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  color: var(--primary-fg);
  background: transparent;
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
}

.command-row.selected .check-mark {
  border-color: var(--primary);
  background: var(--primary);
}

.command-name {
  color: var(--text);
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
}

.command-row code {
  color: var(--text-muted);
  font-family: var(--font-mono);
  font-size: var(--fs-mono);
}
</style>
