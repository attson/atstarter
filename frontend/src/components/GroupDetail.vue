<script setup>
import { computed } from 'vue'
import { Play, Square, Pencil, Trash2 } from 'lucide-vue-next'
import AppButton from './ui/AppButton.vue'
import AppPill from './ui/AppPill.vue'
import AppIcon from './ui/AppIcon.vue'

const props = defineProps({ group: Object, projects: Array })
const emit = defineEmits(['start', 'stop', 'edit', 'remove', 'select-command'])

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
  for (const item of (props.group && props.group.items) || []) {
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
  <section class="group-detail" v-if="group">
    <header class="group-header">
      <div class="info">
        <div class="title-line">
          <h1>{{ group.name }}</h1>
          <AppPill variant="neutral">group</AppPill>
          <AppPill variant="stopped">{{ (group.items || []).length }} commands</AppPill>
        </div>
        <p>启动、停止或检查这组项目命令。</p>
      </div>
      <div class="btns">
        <AppButton variant="secondary" @click="emit('edit', group)">
          <template #icon><AppIcon :icon="Pencil" :size="14" /></template>
          Edit
        </AppButton>
        <AppButton variant="secondary" @click="emit('remove', group.id)">
          <template #icon><AppIcon :icon="Trash2" :size="14" /></template>
          Remove
        </AppButton>
        <AppButton variant="danger" @click="emit('stop', group.id)">
          <template #icon><AppIcon :icon="Square" :size="14" /></template>
          Stop
        </AppButton>
        <AppButton variant="success" @click="emit('start', group.id)">
          <template #icon><AppIcon :icon="Play" :size="14" /></template>
          Start
        </AppButton>
      </div>
    </header>

    <div class="members">
      <button
        v-for="member in members"
        :key="member.key"
        class="member-row"
        @click="emit('select-command', { projectId: member.project.id, commandId: member.command.id || 'default' })"
      >
        <span class="dot" />
        <span class="project-name">{{ member.project.name }}</span>
        <span class="command-name">{{ member.command.name }}</span>
        <code>{{ lineFor(member.command) }}</code>
      </button>
      <div v-if="members.length === 0" class="empty">这个分组还没有项目命令。</div>
    </div>
  </section>
</template>

<style scoped>
.group-detail {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: transparent;
}

.group-header {
  display: grid;
  grid-template-columns: minmax(280px, 1fr) auto;
  align-items: start;
  gap: var(--space-8);
  padding: var(--space-7) var(--space-8);
  border-bottom: 1px solid var(--border);
}

.info { min-width: 0; }

.title-line {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--space-4);
}

h1 {
  margin: 0;
  color: var(--text);
  font-size: var(--fs-lg);
  font-weight: var(--fw-semibold);
  letter-spacing: -0.022em;
  line-height: 1.15;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

p {
  margin: var(--space-4) 0 0;
  color: var(--text-muted);
  font-size: var(--fs-base);
}

.btns {
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: var(--space-3);
  max-width: 420px;
}

.members {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: var(--space-7) var(--space-8);
  background: linear-gradient(180deg, rgba(255, 255, 255, .015), transparent), var(--surface);
}

.member-row {
  width: 100%;
  min-height: 38px;
  display: grid;
  grid-template-columns: 10px minmax(160px, 240px) minmax(90px, 140px) minmax(0, 1fr);
  align-items: center;
  gap: var(--space-5);
  margin-bottom: var(--space-3);
  padding: var(--space-3) var(--space-5);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--elevated-gradient);
  color: var(--text-secondary);
  font: inherit;
  text-align: left;
  cursor: pointer;
  box-shadow: var(--surface-highlight);
  transition: background var(--dur-fast) var(--ease-spring), border-color var(--dur-fast) var(--ease-spring);
}

.member-row:hover {
  border-color: var(--border-strong);
  background: color-mix(in srgb, var(--elevated) 50%, var(--text) 6%);
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--text-subtle);
}

.project-name, .command-name, .member-row code {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-name {
  color: var(--text);
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
}

.command-name {
  color: var(--text-muted);
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
}

.member-row code {
  color: var(--text-muted);
  font-family: var(--font-mono);
  font-size: var(--fs-mono);
}

.empty {
  color: var(--text-muted);
  font-size: var(--fs-sm);
  padding: var(--space-8) var(--space-2);
}

@media (max-width: 980px) {
  .group-header {
    grid-template-columns: 1fr;
    gap: var(--space-6);
  }

  .btns {
    justify-content: flex-start;
    max-width: none;
  }
}
</style>
