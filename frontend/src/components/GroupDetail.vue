<script setup>
import { computed } from 'vue'

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
          <span class="group-pill">group</span>
          <span class="count-pill">{{ (group.items || []).length }} commands</span>
        </div>
        <p>启动、停止或检查这组项目命令。</p>
      </div>
      <div class="btns">
        <button class="action secondary" @click="emit('edit', group)">Edit</button>
        <button class="action danger" @click="emit('stop', group.id)">Stop</button>
        <button class="action primary" @click="emit('start', group.id)">Start</button>
        <button class="action secondary" @click="emit('remove', group.id)">Remove</button>
      </div>
    </header>

    <div class="members">
      <button
        v-for="member in members"
        :key="member.key"
        class="member-row"
        @click="emit('select-command', { projectId: member.project.id, commandId: member.command.id || 'default' })"
      >
        <span class="dot"></span>
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
  background: #ffffff;
}

.group-header {
  display: grid;
  grid-template-columns: minmax(280px, 1fr) auto;
  align-items: start;
  gap: 20px;
  padding: 18px 20px;
  border-bottom: 1px solid #d7dce5;
}

.info {
  min-width: 0;
}

.title-line {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}

h1 {
  margin: 0;
  color: #111827;
  font-size: 22px;
  line-height: 1.1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

p {
  margin: 8px 0 0;
  color: #64748b;
  font-size: 13px;
}

.group-pill,
.count-pill {
  border-radius: 999px;
  padding: 2px 8px;
  font-size: 12px;
  font-weight: 800;
}

.group-pill {
  color: #1d4ed8;
  background: #eff6ff;
  border: 1px solid #bfdbfe;
}

.count-pill {
  color: #475569;
  background: #f8fafc;
  border: 1px solid #cbd5e1;
}

.btns {
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 8px;
  max-width: 340px;
}

.action {
  height: 34px;
  border-radius: 7px;
  padding: 0 12px;
  font: inherit;
  font-size: 13px;
  font-weight: 800;
  cursor: pointer;
}

.action.primary {
  color: #ffffff;
  background: #16a34a;
  border: 1px solid #16a34a;
}

.action.secondary {
  color: #334155;
  background: #f1f5f9;
  border: 1px solid #cbd5e1;
}

.action.danger {
  color: #991b1b;
  background: #fee2e2;
  border: 1px solid #fecaca;
}

.members {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 14px 18px;
  background: #f8fafc;
}

.member-row {
  width: 100%;
  min-height: 38px;
  display: grid;
  grid-template-columns: 10px minmax(160px, 240px) minmax(90px, 140px) minmax(0, 1fr);
  align-items: center;
  gap: 10px;
  margin-bottom: 7px;
  padding: 7px 10px;
  border: 1px solid #e2e8f0;
  border-radius: 7px;
  background: #ffffff;
  color: #334155;
  font: inherit;
  text-align: left;
  cursor: pointer;
}

.member-row:hover {
  border-color: #93c5fd;
  background: #eff6ff;
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #94a3b8;
}

.project-name,
.command-name,
.member-row code {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.project-name {
  color: #111827;
  font-size: 13px;
  font-weight: 800;
}

.command-name {
  color: #1d4ed8;
  font-size: 12px;
  font-weight: 800;
}

.member-row code {
  color: #64748b;
  font-size: 12px;
}

.empty {
  color: #64748b;
  font-size: 13px;
  padding: 18px 4px;
}

@media (max-width: 980px) {
  .group-header {
    grid-template-columns: 1fr;
    gap: 14px;
  }

  .btns {
    justify-content: flex-start;
    max-width: none;
  }
}
</style>
