<script setup>
const props = defineProps({
  node: Object,
  level: Number,
  checked: Object,
})
const emit = defineEmits(['toggle'])

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
    <div class="dir-row" :style="{ paddingLeft: `${10 + level * 16}px` }">
      <span class="chevron">▾</span>
      <span class="dir-name">{{ node.label }}</span>
      <span class="count">{{ node.count }}</span>
    </div>
    <GroupTreeNode
      v-for="child in node.children"
      :key="child.id"
      :node="child"
      :level="level + 1"
      :checked="checked"
      @toggle="emit('toggle', $event)"
    />
  </div>

  <div v-else class="group-node">
    <div class="project-row" :style="{ paddingLeft: `${12 + level * 16}px` }">
      <span class="project-dot"></span>
      <span class="project-name">{{ node.project.name }}</span>
      <span v-if="node.children && node.children.length" class="count">{{ node.count }}</span>
      <span v-else class="type-pill">{{ node.project.detectedType || 'unknown' }}</span>
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
      @toggle="emit('toggle', $event)"
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
  gap: 7px;
}

.dir-row {
  grid-template-columns: 16px minmax(0, 1fr) auto;
  color: #334155;
  background: #f8fafc;
  border-bottom: 1px solid #edf2f7;
  font-size: 12px;
  font-weight: 800;
}

.project-row {
  grid-template-columns: 10px minmax(0, 1fr) auto;
  color: #111827;
  border-bottom: 1px solid #f1f5f9;
  font-size: 13px;
  font-weight: 800;
}

.chevron,
.count {
  color: #64748b;
}

.project-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #94a3b8;
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
  margin-right: 8px;
  border: 1px solid #dbeafe;
  border-radius: 999px;
  color: #2563eb;
  background: #ffffff;
  padding: 1px 7px;
  font-size: 11px;
  font-weight: 800;
}

.command-row {
  width: 100%;
  display: grid;
  grid-template-columns: 18px minmax(90px, 140px) minmax(0, 1fr);
  align-items: center;
  gap: 9px;
  min-height: 30px;
  border: 0;
  border-bottom: 1px solid #f8fafc;
  background: #ffffff;
  color: #334155;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.command-row.selected {
  background: #eff6ff;
}

.check-mark {
  width: 16px;
  height: 16px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid #cbd5e1;
  border-radius: 5px;
  color: #ffffff;
  background: #ffffff;
  font-size: 11px;
  font-weight: 900;
}

.command-row.selected .check-mark {
  border-color: #2563eb;
  background: #2563eb;
}

.command-name {
  color: #2563eb;
  font-size: 12px;
  font-weight: 800;
}

.command-row code {
  color: #64748b;
  font-size: 12px;
}
</style>
