<script setup>
import { computed, ref, watch } from 'vue'
import LogPanel from './LogPanel.vue'
const props = defineProps({ project: Object, status: Object, selectedCommandId: String })
const emit = defineEmits(['start', 'stop', 'edit', 'command-change', 'add-to-group'])
const commandMenuOpen = ref(false)

const commands = computed(() => {
  if (!props.project) return []
  if (props.project.commands && props.project.commands.length) return props.project.commands
  return [{
    id: 'default',
    name: 'Default',
    command: props.project.command,
    args: props.project.args || [],
    cwd: props.project.cwd || '',
    isDefault: true,
  }]
})
const selectedCommand = computed(() =>
  commands.value.find((c) => c.id === props.selectedCommandId) ||
  commands.value.find((c) => c.isDefault) ||
  commands.value[0]
)
const selectedRunId = computed(() => props.project && selectedCommand.value
  ? `${props.project.id}:${selectedCommand.value.id || 'default'}`
  : ''
)
const commandLine = computed(() => selectedCommand.value
  ? [selectedCommand.value.command, ...(selectedCommand.value.args || [])].join(' ')
  : ''
)

watch(() => props.selectedCommandId, () => {
  commandMenuOpen.value = false
})

function chooseCommand(command) {
  emit('command-change', command.id)
  commandMenuOpen.value = false
}
</script>

<template>
  <section class="detail" v-if="project">
    <div class="project-header">
      <div class="info">
        <div class="title-line">
          <h1>{{ project.name }}</h1>
          <span :class="['state-pill', (status || {}).State || 'stopped']">
            {{ (status || {}).State || 'stopped' }}
          </span>
          <span class="type-pill">{{ project.detectedType || 'unknown' }}</span>
        </div>
        <div class="path">{{ project.path }}</div>
        <div class="command-box">
          <span>CMD</span>
          <div class="command-picker">
            <button class="command-trigger" @click="commandMenuOpen = !commandMenuOpen">
              {{ selectedCommand && selectedCommand.name }}
              <span>{{ commandMenuOpen ? '▴' : '▾' }}</span>
            </button>
            <div v-if="commandMenuOpen" class="command-menu">
              <button
                v-for="cmd in commands"
                :key="cmd.id"
                :class="{ active: selectedCommand && selectedCommand.id === cmd.id }"
                @click="chooseCommand(cmd)"
              >
                {{ cmd.name }}
              </button>
            </div>
          </div>
          <code>{{ commandLine }}</code>
        </div>
      </div>
      <div class="btns">
        <button class="action secondary" @click="emit('add-to-group')">Add Group</button>
        <button class="action secondary" @click="emit('edit')">Edit</button>
        <button class="action danger" :disabled="(status || {}).State !== 'running'" @click="emit('stop', selectedCommand.id)">Stop</button>
        <button class="action primary" :disabled="(status || {}).State === 'running'" @click="emit('start', selectedCommand.id)">Start</button>
      </div>
    </div>
    <LogPanel :projectId="selectedRunId" :status="status" />
  </section>
  <section class="detail empty" v-else>
    <div>
      <h2>选择一个项目</h2>
      <p>从左侧目录树选择项目后查看命令、状态和实时日志。</p>
    </div>
  </section>
</template>

<style scoped>
.detail {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: #ffffff;
}

.detail.empty {
  align-items: center;
  justify-content: center;
  color: #64748b;
  text-align: center;
}

.detail.empty h2 {
  margin: 0 0 8px;
  color: #111827;
  font-size: 22px;
}

.detail.empty p {
  margin: 0;
  font-size: 14px;
}

.project-header {
  display: grid;
  grid-template-columns: minmax(280px, 1fr) auto;
  align-items: start;
  gap: 20px;
  padding: 18px 20px;
  border-bottom: 1px solid #d7dce5;
  background: #ffffff;
}

.info {
  min-width: 0;
  max-width: 100%;
  display: flex;
  flex-direction: column;
  gap: 9px;
}

.title-line {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  min-width: 0;
}

h1 {
  max-width: min(560px, 100%);
  margin: 0;
  color: #111827;
  font-size: 22px;
  line-height: 1.1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.path {
  max-width: 100%;
  color: #64748b;
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.command-box {
  display: flex;
  align-items: center;
  gap: 8px;
  width: min(100%, 760px);
  box-sizing: border-box;
  min-width: 0;
  padding: 8px 10px;
  border: 1px solid #e2e8f0;
  border-radius: 7px;
  background: #f8fafc;
}

.command-picker {
  position: relative;
  flex-shrink: 0;
}

.command-trigger {
  height: 26px;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  border: 1px solid #cbd5e1;
  border-radius: 6px;
  background: #ffffff;
  color: #0f172a;
  padding: 0 9px;
  font: inherit;
  font-size: 12px;
  font-weight: 800;
  cursor: pointer;
}

.command-trigger span {
  color: #64748b;
  font-size: 10px;
}

.command-menu {
  position: absolute;
  z-index: 10;
  top: 31px;
  left: 0;
  min-width: 150px;
  padding: 5px;
  border: 1px solid #cbd5e1;
  border-radius: 7px;
  background: #ffffff;
  box-shadow: 0 12px 28px rgba(15, 23, 42, .14);
}

.command-menu button {
  width: 100%;
  height: 30px;
  border: 0;
  border-radius: 5px;
  background: transparent;
  color: #334155;
  padding: 0 8px;
  font: inherit;
  font-size: 12px;
  font-weight: 800;
  text-align: left;
  cursor: pointer;
}

.command-menu button:hover,
.command-menu button.active {
  color: #1d4ed8;
  background: #eff6ff;
}

.command-box span {
  color: #64748b;
  font-size: 11px;
  font-weight: 800;
}

.command-box code {
  min-width: 0;
  color: #0f172a;
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.state-pill,
.type-pill {
  border-radius: 999px;
  padding: 2px 8px;
  font-size: 12px;
  font-weight: 800;
  flex-shrink: 0;
}

.state-pill.running {
  color: #047857;
  background: #ecfdf5;
  border: 1px solid #a7f3d0;
}

.state-pill.exited,
.state-pill.error {
  color: #b91c1c;
  background: #fef2f2;
  border: 1px solid #fecaca;
}

.state-pill.stopped,
.state-pill.undefined {
  color: #475569;
  background: #f1f5f9;
  border: 1px solid #cbd5e1;
}

.type-pill {
  color: #2563eb;
  background: #ffffff;
  border: 1px solid #dbeafe;
}

.btns {
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  align-items: flex-start;
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

@media (max-width: 980px) {
  .project-header {
    grid-template-columns: 1fr;
    gap: 14px;
  }

  .btns {
    justify-content: flex-start;
    max-width: none;
  }
}

@media (max-width: 760px) {
  .project-header {
    padding: 14px 16px;
  }

  .command-box {
    flex-wrap: wrap;
  }

  .command-box code {
    flex-basis: 100%;
  }
}

.action:disabled {
  cursor: not-allowed;
  opacity: .45;
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
</style>
