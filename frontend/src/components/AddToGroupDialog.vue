<script setup>
import { computed, ref, watch } from 'vue'
import AppButton from './ui/AppButton.vue'

const props = defineProps({
  show: Boolean,
  groups: Array,
  project: Object,
  command: Object,
})
const emit = defineEmits(['close', 'save'])

const mode = ref('existing')
const selectedGroupId = ref('')
const newGroupName = ref('')

const commandLine = computed(() => props.command
  ? [props.command.command, ...(props.command.args || [])].filter(Boolean).join(' ')
  : ''
)

watch(() => props.show, (show) => {
  if (!show) return
  mode.value = (props.groups || []).length ? 'existing' : 'new'
  selectedGroupId.value = (props.groups || [])[0]?.id || ''
  newGroupName.value = props.project ? `${props.project.name} group` : ''
})

function save() {
  if (mode.value === 'existing' && !selectedGroupId.value) return
  if (mode.value === 'new' && !newGroupName.value.trim()) return
  emit('save', {
    mode: mode.value,
    groupId: selectedGroupId.value,
    groupName: newGroupName.value.trim(),
  })
}
</script>

<template>
  <Transition name="dlg-fade">
    <div class="mask" v-if="show" @click.self="emit('close')">
      <Transition name="dlg-pop" appear>
        <div class="dialog">
          <h3>添加到组</h3>
          <div class="target">
            <strong>{{ project && project.name }}</strong>
            <span>{{ command && command.name }}</span>
            <code>{{ commandLine }}</code>
          </div>

          <div class="mode-tabs">
            <button
              :class="['mode-tab', { active: mode === 'existing' }]"
              :disabled="!(groups || []).length"
              @click="mode = 'existing'"
            >已有组</button>
            <button
              :class="['mode-tab', { active: mode === 'new' }]"
              @click="mode = 'new'"
            >新建组</button>
          </div>

          <div v-if="mode === 'existing'" class="group-list">
            <button
              v-for="group in groups"
              :key="group.id"
              :class="['group-option', { selected: selectedGroupId === group.id }]"
              @click="selectedGroupId = group.id"
            >
              <span>{{ group.name }}</span>
              <small>{{ (group.items || []).length }} commands</small>
            </button>
          </div>
          <label v-else>组名<input v-model="newGroupName" placeholder="Local dev stack" /></label>

          <div class="btns">
            <AppButton variant="secondary" @click="emit('close')">取消</AppButton>
            <AppButton variant="primary" @click="save">添加</AppButton>
          </div>
        </div>
      </Transition>
    </div>
  </Transition>
</template>

<style scoped>
.mask {
  position: fixed;
  inset: 0;
  z-index: var(--z-modal);
  background: var(--overlay);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
}

.dialog {
  width: min(560px, calc(100vw - 36px));
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
  padding: var(--space-9);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--surface);
  box-shadow: var(--shadow-lg);
}

h3 {
  margin: 0;
  color: var(--text);
  font-size: var(--fs-md);
  font-weight: var(--fw-semibold);
  letter-spacing: -0.005em;
}

.target {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: var(--space-2) var(--space-5);
  padding: var(--space-5);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg);
}

.target strong,
.target code {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.target strong { color: var(--text); font-size: var(--fs-sm); font-weight: var(--fw-semibold); }

.target span {
  color: var(--text-muted);
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
}

.target code {
  grid-column: 1 / -1;
  color: var(--text-muted);
  font-family: var(--font-mono);
  font-size: var(--fs-mono);
}

.mode-tabs {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-3);
}

.mode-tab {
  height: 32px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: var(--bg);
  color: var(--text-secondary);
  padding: 0 var(--space-5);
  font: inherit;
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
  cursor: pointer;
  transition: background var(--dur-fast) var(--ease), color var(--dur-fast) var(--ease);
}

.mode-tab:disabled { cursor: not-allowed; opacity: .5; }

.mode-tab.active {
  background: var(--elevated);
  color: var(--text);
  border-color: var(--border-strong);
  box-shadow: inset 0 0 0 1px var(--border-strong);
}

.group-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.group-option {
  min-height: 40px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 var(--space-5);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg);
  color: var(--text-secondary);
  font: inherit;
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
  text-align: left;
  cursor: pointer;
  transition: background var(--dur-fast) var(--ease);
}

.group-option:hover { background: var(--elevated); }

.group-option.selected {
  background: var(--elevated);
  color: var(--text);
  border-color: var(--border-strong);
  box-shadow: inset 0 0 0 1px var(--border-strong);
}

.group-option small {
  color: var(--text-muted);
  font-weight: var(--fw-regular);
}

label {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  color: var(--text-secondary);
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
}

input {
  height: 32px;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  color: var(--text);
  background: var(--bg);
  padding: 0 var(--space-5);
  font: inherit;
  outline: none;
  transition: border-color var(--dur-fast) var(--ease), box-shadow var(--dur-fast) var(--ease);
}

input:focus {
  border-color: var(--text-subtle);
  box-shadow: 0 0 0 3px var(--focus-ring);
}

.btns {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
}
</style>
