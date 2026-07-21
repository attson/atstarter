<script setup>
import { computed, ref, watch } from 'vue'
import { Play, Square, RotateCcw, Pencil, FolderPlus, ChevronDown, ChevronUp, GitBranch } from 'lucide-vue-next'
import LogPanel from './LogPanel.vue'
import AppButton from './ui/AppButton.vue'
import AppPill from './ui/AppPill.vue'
import AppIcon from './ui/AppIcon.vue'
import { typeLabel } from '../typeLabel.js'
import { GetProjectBranch } from '../../wailsjs/go/main/App'

const props = defineProps({ project: Object, status: Object, selectedCommandId: String })
const emit = defineEmits(['start', 'stop', 'restart', 'edit', 'command-change', 'add-to-group'])
const commandMenuOpen = ref(false)
const branch = ref('')
let branchToken = 0

async function refreshBranch(path) {
  const token = ++branchToken
  branch.value = ''
  if (!path) return
  const value = await GetProjectBranch(path)
  if (token !== branchToken) return
  branch.value = value || ''
}

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

const state = computed(() => (props.status || {}).State || 'stopped')
const pillVariant = computed(() => {
  if (state.value === 'running') return 'running'
  if (state.value === 'exited') return 'exited'
  if (state.value === 'error') return 'error'
  return 'stopped'
})

watch(() => props.selectedCommandId, () => {
  commandMenuOpen.value = false
})

watch(() => props.project?.path, (path) => { refreshBranch(path) }, { immediate: true })

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
          <AppPill :variant="pillVariant" :dot="state === 'running'">{{ state }}</AppPill>
          <AppPill variant="neutral">{{ typeLabel(project.detectedType) }}</AppPill>
          <AppPill v-if="branch" variant="neutral" class="branch-pill">
            <AppIcon :icon="GitBranch" :size="11" />
            {{ branch }}
          </AppPill>
        </div>
        <div class="path">{{ project.path }}</div>
        <div class="command-box">
          <span class="cmd-label">CMD</span>
          <div class="command-picker">
            <button class="command-trigger" @click="commandMenuOpen = !commandMenuOpen">
              <span>{{ selectedCommand && selectedCommand.name }}</span>
              <AppIcon :icon="commandMenuOpen ? ChevronUp : ChevronDown" :size="12" />
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
        <AppButton variant="secondary" size="sm" @click="emit('add-to-group')">
          <template #icon><AppIcon :icon="FolderPlus" :size="13" /></template>
          Add Group
        </AppButton>
        <AppButton variant="secondary" size="sm" @click="emit('edit')">
          <template #icon><AppIcon :icon="Pencil" :size="13" /></template>
          Edit
        </AppButton>
        <div class="run-controls">
          <AppButton
            variant="danger"
            size="sm"
            iconOnly
            title="Stop"
            :disabled="(status || {}).State !== 'running'"
            @click="emit('stop', selectedCommand.id)"
          >
            <template #icon><AppIcon :icon="Square" :size="14" /></template>
          </AppButton>
          <AppButton
            variant="secondary"
            size="sm"
            iconOnly
            title="Restart"
            :disabled="(status || {}).State !== 'running'"
            @click="emit('restart', selectedCommand.id)"
          >
            <template #icon><AppIcon :icon="RotateCcw" :size="14" /></template>
          </AppButton>
          <AppButton
            variant="success"
            size="sm"
            iconOnly
            title="Start"
            :disabled="(status || {}).State === 'running'"
            @click="emit('start', selectedCommand.id)"
          >
            <template #icon><AppIcon :icon="Play" :size="14" /></template>
          </AppButton>
        </div>
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
  background: transparent;
}

.detail.empty {
  align-items: center;
  justify-content: center;
  color: var(--text-muted);
  text-align: center;
}

.detail.empty h2 {
  margin: 0 0 var(--space-4);
  color: var(--text);
  font-size: var(--fs-lg);
  font-weight: var(--fw-semibold);
  letter-spacing: -0.015em;
}

.detail.empty p {
  margin: 0;
  font-size: var(--fs-base);
}

.project-header {
  display: grid;
  grid-template-columns: minmax(280px, 1fr) auto;
  align-items: start;
  gap: var(--space-8);
  padding: var(--space-7) var(--space-8);
  border-bottom: 1px solid var(--border);
  background: transparent;
}

.info {
  min-width: 0;
  max-width: 100%;
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.branch-pill {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--accent-strong);
  border-color: var(--success-line);
  background: var(--success-gradient);
}

.title-line {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--space-4);
  min-width: 0;
}

h1 {
  max-width: min(560px, 100%);
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

.path {
  max-width: 100%;
  color: var(--text-muted);
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.command-box {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  width: min(100%, 760px);
  box-sizing: border-box;
  min-width: 0;
  padding: var(--space-4) var(--space-5);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--elevated-gradient);
  box-shadow: var(--surface-highlight);
}

.cmd-label {
  color: var(--text-subtle);
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  letter-spacing: 0.03em;
}

.command-picker {
  position: relative;
  flex-shrink: 0;
}

.command-trigger {
  height: 24px;
  display: inline-flex;
  align-items: center;
  gap: var(--space-3);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  background: var(--elevated-gradient);
  color: var(--text);
  padding: 0 var(--space-3);
  font: inherit;
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
  cursor: pointer;
  box-shadow: var(--surface-highlight);
  transition: background var(--dur-fast) var(--ease), filter var(--dur-fast) var(--ease);
}

.command-trigger:hover { filter: brightness(1.08); }

.command-menu {
  position: absolute;
  z-index: var(--z-menu);
  top: 28px;
  left: 0;
  min-width: 150px;
  padding: var(--space-2);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: var(--surface);
  box-shadow: var(--shadow-md);
}

.command-menu button {
  width: 100%;
  height: 28px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-secondary);
  padding: 0 var(--space-3);
  font: inherit;
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
  text-align: left;
  cursor: pointer;
  transition: background var(--dur-fast) var(--ease);
}

.command-menu button:hover,
.command-menu button.active {
  color: var(--text);
  background: var(--elevated);
}

.command-box code {
  min-width: 0;
  color: var(--text);
  font-family: var(--font-mono);
  font-size: var(--fs-mono);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.btns {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: var(--space-3);
  flex-wrap: wrap;
  row-gap: var(--space-3);
}

/* Stop / Restart / Start 作为一组运行控制,紧凑相邻,与左侧 Add Group/Edit 分隔。 */
.run-controls {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  margin-left: var(--space-3);
  padding-left: var(--space-4);
  border-left: 1px solid var(--border);
}

@media (max-width: 980px) {
  .project-header {
    grid-template-columns: 1fr;
    gap: var(--space-6);
  }

  .btns {
    justify-content: flex-start;
    flex-wrap: wrap;
  }
}

@media (max-width: 760px) {
  .project-header {
    padding: var(--space-6) var(--space-7);
  }

  .command-box {
    flex-wrap: wrap;
  }

  .command-box code {
    flex-basis: 100%;
  }
}
</style>
