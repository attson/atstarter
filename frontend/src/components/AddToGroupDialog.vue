<script setup>
import { computed, ref, watch } from 'vue'

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
  <div class="mask" v-if="show" @click.self="emit('close')">
    <div class="dialog">
      <h3>添加到组</h3>
      <div class="target">
        <strong>{{ project && project.name }}</strong>
        <span>{{ command && command.name }}</span>
        <code>{{ commandLine }}</code>
      </div>

      <div class="mode-tabs">
        <button :class="{ active: mode === 'existing' }" :disabled="!(groups || []).length" @click="mode = 'existing'">已有组</button>
        <button :class="{ active: mode === 'new' }" @click="mode = 'new'">新建组</button>
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
        <button @click="emit('close')">取消</button>
        <button @click="save">添加</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.mask {
  position: fixed;
  inset: 0;
  background: rgba(15, 23, 42, .46);
  display: flex;
  align-items: center;
  justify-content: center;
}

.dialog {
  width: min(560px, calc(100vw - 36px));
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 22px;
  border: 1px solid #d7dce5;
  border-radius: 8px;
  background: #ffffff;
  box-shadow: 0 24px 70px rgba(15, 23, 42, .22);
}

h3 {
  margin: 0;
  color: #111827;
  font-size: 18px;
}

.target {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 4px 10px;
  padding: 10px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: #f8fafc;
}

.target strong,
.target code {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.target span {
  color: #2563eb;
  font-size: 12px;
  font-weight: 800;
}

.target code {
  grid-column: 1 / -1;
  color: #64748b;
  font-size: 12px;
}

.mode-tabs {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 6px;
}

button {
  height: 32px;
  border: 1px solid #cbd5e1;
  border-radius: 7px;
  background: #f8fafc;
  color: #334155;
  padding: 0 12px;
  font: inherit;
  font-size: 13px;
  font-weight: 800;
  cursor: pointer;
}

button:disabled {
  cursor: not-allowed;
  opacity: .46;
}

.mode-tabs button.active,
.group-option.selected {
  color: #1d4ed8;
  background: #eff6ff;
  border-color: #93c5fd;
}

.group-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.group-option {
  height: auto;
  min-height: 40px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  text-align: left;
}

.group-option small {
  color: #64748b;
}

label {
  display: flex;
  flex-direction: column;
  gap: 6px;
  color: #334155;
  font-size: 13px;
  font-weight: 700;
}

input {
  height: 34px;
  border: 1px solid #cbd5e1;
  border-radius: 7px;
  color: #0f172a;
  padding: 0 10px;
  font: inherit;
  outline: none;
}

input:focus {
  border-color: #2563eb;
  box-shadow: 0 0 0 3px #dbeafe;
}

.btns {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.btns button:last-child {
  color: #ffffff;
  background: #2563eb;
  border-color: #2563eb;
}
</style>
