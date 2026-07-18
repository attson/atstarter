<script setup>
import { computed, ref, watch } from 'vue'
import { buildProjectTree } from '../projectTree'
import GroupTreeNode from './GroupTreeNode.vue'

const props = defineProps({ show: Boolean, group: Object, projects: Array })
const emit = defineEmits(['save', 'close'])

const name = ref('')
const checked = ref({})

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

const commandOptions = computed(() => (props.projects || []).flatMap((project) =>
  commandsFor(project).map((command) => ({ project, command, key: keyFor(project.id, command.id) }))
))
const projectTree = computed(() => buildProjectTree(props.projects || [], {}, ''))

function reset() {
  name.value = props.group ? props.group.name : ''
  const next = {}
  for (const item of (props.group && props.group.items) || []) {
    next[keyFor(item.projectId, item.commandId)] = true
  }
  checked.value = next
}

watch(() => props.show, (show) => {
  if (show) reset()
})
watch(() => props.group, reset, { immediate: true })

function save() {
  const items = commandOptions.value
    .filter((option) => checked.value[option.key])
    .map((option) => ({ projectId: option.project.id, commandId: option.command.id || 'default' }))
  emit('save', {
    id: props.group ? props.group.id : '',
    name: name.value.trim() || 'New group',
    items,
  })
}

function toggleOption(key) {
  checked.value = { ...checked.value, [key]: !checked.value[key] }
}
</script>

<template>
  <div class="mask" v-if="show" @click.self="emit('close')">
    <div class="dialog">
      <h3>{{ group ? '编辑分组' : '新建分组' }}</h3>
      <label>名称<input v-model="name" placeholder="Local dev stack" /></label>

      <div class="commands-head">选择要启动的项目命令</div>
      <div class="options">
        <GroupTreeNode
          v-for="node in projectTree"
          :key="node.id"
          :node="node"
          :level="0"
          :checked="checked"
          @toggle="toggleOption"
        />
      </div>

      <div class="btns">
        <button @click="emit('close')">取消</button>
        <button @click="save">保存</button>
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
  width: min(820px, calc(100vw - 36px));
  max-height: calc(100vh - 56px);
  overflow-y: auto;
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
  margin: 0 0 2px;
  color: #111827;
  font-size: 18px;
}

.dialog label {
  display: flex;
  flex-direction: column;
  gap: 6px;
  color: #334155;
  font-size: 13px;
  font-weight: 700;
}

.dialog > label input {
  height: 34px;
  border: 1px solid #cbd5e1;
  border-radius: 7px;
  color: #0f172a;
  padding: 0 10px;
  font: inherit;
  outline: none;
}

.commands-head {
  color: #334155;
  font-size: 13px;
  font-weight: 800;
}

.options {
  max-height: 400px;
  overflow-y: auto;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
}

.btns {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.btns button {
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

.btns button:last-child {
  color: #ffffff;
  background: #2563eb;
  border-color: #2563eb;
}
</style>
