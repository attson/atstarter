<script setup>
import { computed } from 'vue'

const props = defineProps({ group: Object, projects: Array, selected: Boolean, expanded: Boolean })
const emit = defineEmits(['select', 'toggle', 'select-command'])

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

function lineFor(command) {
  return [command.command, ...(command.args || [])].filter(Boolean).join(' ')
}

const projectById = computed(() => {
  const out = {}
  for (const project of props.projects || []) out[project.id] = project
  return out
})

const members = computed(() => {
  const out = []
  for (const item of props.group.items || []) {
    const project = projectById.value[item.projectId]
    if (!project) continue
    const command = commandsFor(project).find((c) => c.id === (item.commandId || 'default'))
    if (!command) continue
    out.push({
      key: `${props.group.id}:${project.id}:${command.id || 'default'}`,
      project,
      command,
    })
  }
  return out
})
</script>

<template>
  <div class="group-wrap">
    <div :class="['group-item', { active: selected }]">
      <button class="toggle" @click.stop="emit('toggle', `group:${group.id}`)">
        {{ expanded ? '▾' : '▸' }}
      </button>
      <button class="group-main" @click="emit('select', group.id)">
        <span class="group-badge">G</span>
        <span class="group-copy">
          <span class="group-name">{{ group.name }}</span>
          <span class="group-count">{{ (group.items || []).length }} commands</span>
        </span>
      </button>
      <span class="count">{{ (group.items || []).length }}</span>
    </div>
    <div v-if="expanded" class="members">
      <button
        v-for="member in members"
        :key="member.key"
        class="member-row"
        @click="emit('select-command', { projectId: member.project.id, commandId: member.command.id || 'default' })"
      >
        <span class="member-dot"></span>
        <span class="member-project">{{ member.project.name }}</span>
        <span class="member-command">{{ member.command.name }}</span>
        <code>{{ lineFor(member.command) }}</code>
      </button>
    </div>
  </div>
</template>

<style scoped>
.group-wrap {
  min-width: 0;
}

.group-item {
  width: 100%;
  display: grid;
  grid-template-columns: 14px minmax(0, 1fr) auto;
  align-items: center;
  gap: 6px;
  min-height: 31px;
  margin: 1px 0;
  padding: 3px 8px 3px 10px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: #1f2937;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.group-item:hover {
  background: #f8fafc;
}

.group-item.active {
  background: #eef4ff;
  box-shadow: inset 0 0 0 1px #bfdbfe;
}

.toggle,
.group-main {
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  cursor: pointer;
}

.toggle {
  width: 14px;
  padding: 0;
  color: #64748b;
  font-size: 11px;
}

.group-main {
  min-width: 0;
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr);
  align-items: center;
  gap: 8px;
  padding: 0;
  text-align: left;
}

.group-badge {
  width: 17px;
  height: 17px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 5px;
  color: #1d4ed8;
  background: #dbeafe;
  font-size: 10px;
  font-weight: 900;
}

.group-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.group-name,
.group-count {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.group-name {
  color: #111827;
  font-size: 13px;
  font-weight: 800;
}

.group-count {
  color: #64748b;
  font-size: 11px;
}

.chevron,
.count {
  color: #64748b;
  font-size: 11px;
}

.members {
  padding: 1px 0 5px;
}

.member-row {
  width: 100%;
  min-height: 29px;
  display: grid;
  grid-template-columns: 9px minmax(92px, 1fr) minmax(72px, 96px) minmax(0, 1fr);
  align-items: center;
  gap: 7px;
  padding: 3px 8px 3px 38px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: #334155;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.member-row:hover {
  background: #eff6ff;
}

.member-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #94a3b8;
}

.member-project,
.member-command,
.member-row code {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.member-project {
  color: #111827;
  font-size: 12px;
  font-weight: 800;
}

.member-command {
  color: #1d4ed8;
  font-size: 12px;
  font-weight: 800;
}

.member-row code {
  color: #64748b;
  font-size: 11px;
}
</style>
