<script setup>
import { computed } from 'vue'
import { ChevronDown, ChevronRight, FolderKanban } from 'lucide-vue-next'
import AppIcon from './ui/AppIcon.vue'

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
        <AppIcon :icon="expanded ? ChevronDown : ChevronRight" :size="12" />
      </button>
      <button class="group-main" @click="emit('select', group.id)">
        <span class="group-badge"><AppIcon :icon="FolderKanban" :size="12" /></span>
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
        <span class="member-dot" />
        <span class="member-project">{{ member.project.name }}</span>
        <span class="member-command">{{ member.command.name }}</span>
        <code>{{ lineFor(member.command) }}</code>
      </button>
    </div>
  </div>
</template>

<style scoped>
.group-wrap { min-width: 0; }

.group-item {
  width: 100%;
  display: grid;
  grid-template-columns: 14px minmax(0, 1fr) auto;
  align-items: center;
  gap: var(--space-3);
  min-height: 28px;
  margin: 1px 0;
  padding: 3px var(--space-4) 3px var(--space-5);
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-secondary);
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: background var(--dur-fast) var(--ease), box-shadow var(--dur-fast) var(--ease);
}

.group-item:hover { background: var(--elevated-gradient); }

.group-item.active {
  background: var(--elevated-gradient);
  color: var(--text);
  box-shadow: inset 0 0 0 1px var(--border-strong), var(--surface-highlight);
}

.toggle, .group-main {
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  cursor: pointer;
}

.toggle {
  width: 14px;
  padding: 0;
  color: var(--text-muted);
}

.group-main {
  min-width: 0;
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr);
  align-items: center;
  gap: var(--space-4);
  padding: 0;
  text-align: left;
}

.group-badge {
  width: 17px;
  height: 17px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-sm);
  color: var(--text);
  background: var(--elevated-gradient);
  border: 1px solid var(--border-strong);
}

.group-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.group-name, .group-count {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.group-name {
  color: var(--text);
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
}

.group-count {
  color: var(--text-muted);
  font-size: var(--fs-xs);
}

.count {
  color: var(--text-muted);
  font-size: var(--fs-xs);
}

.members { padding: 1px 0 var(--space-3); }

.member-row {
  width: 100%;
  min-height: 26px;
  display: grid;
  grid-template-columns: 9px minmax(92px, 1fr) minmax(72px, 96px) minmax(0, 1fr);
  align-items: center;
  gap: var(--space-3);
  padding: 3px var(--space-4) 3px 38px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-secondary);
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: background var(--dur-fast) var(--ease);
}

.member-row:hover { background: var(--elevated-gradient); }

.member-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--text-subtle);
}

.member-project, .member-command, .member-row code {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.member-project {
  color: var(--text);
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
}

.member-command {
  color: var(--text-muted);
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
}

.member-row code {
  color: var(--text-muted);
  font-size: var(--fs-xs);
  font-family: var(--font-mono);
}
</style>
