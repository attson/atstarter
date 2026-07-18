<script setup>
import { ref, watch } from 'vue'
const props = defineProps({ project: Object, show: Boolean })
const emit = defineEmits(['save', 'close'])

const name = ref('')
const commands = ref([])

function lineFor(command) {
  return [command.command, ...(command.args || [])].filter(Boolean).join(' ')
}

function reset(p) {
  if (!p) return
  name.value = p.name
  const source = p.commands && p.commands.length ? p.commands : [{
    id: 'default',
    name: 'Default',
    command: p.command,
    args: p.args || [],
    cwd: p.cwd || '',
    isDefault: true,
  }]
  commands.value = source.map((c, index) => ({
    id: c.id || '',
    name: c.name || (index === 0 ? 'Default' : `Command ${index + 1}`),
    line: lineFor(c),
    cwd: c.cwd || '',
    isDefault: !!c.isDefault || index === 0,
  }))
}

watch(() => props.project, (p) => {
  reset(p)
}, { immediate: true })

function save() {
  emit('save', { name: name.value, commands: commands.value })
}

function addCommand() {
  commands.value.push({
    id: '',
    name: `Command ${commands.value.length + 1}`,
    line: '',
    cwd: '',
    isDefault: commands.value.length === 0,
  })
}

function removeCommand(index) {
  if (commands.value.length <= 1) return
  const wasDefault = commands.value[index].isDefault
  commands.value.splice(index, 1)
  if (wasDefault && commands.value.length) commands.value[0].isDefault = true
}

function setDefault(index) {
  commands.value = commands.value.map((c, i) => ({ ...c, isDefault: i === index }))
}
</script>

<template>
  <div class="mask" v-if="show" @click.self="emit('close')">
    <div class="dialog">
      <h3>编辑项目</h3>
      <label>名称<input v-model="name" /></label>
      <div class="commands-head">
        <span>启动命令</span>
        <button @click="addCommand">Add command</button>
      </div>
      <div class="command-list">
        <div v-for="(cmd, index) in commands" :key="cmd.id || index" class="command-row">
          <div class="command-top">
            <input v-model="cmd.name" placeholder="Name" />
            <button :class="['default-toggle', { active: cmd.isDefault }]" @click="setDefault(index)">Default</button>
            <button :disabled="commands.length <= 1" @click="removeCommand(index)">Remove</button>
          </div>
          <input v-model="cmd.line" placeholder="如 pnpm run dev 或 go run main.go serve" />
          <input v-model="cmd.cwd" :placeholder="project && project.path" />
        </div>
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
  width: min(720px, calc(100vw - 36px));
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

.dialog input {
  height: 34px;
  border: 1px solid #cbd5e1;
  border-radius: 7px;
  color: #0f172a;
  padding: 0 10px;
  font: inherit;
  outline: none;
}

.commands-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: #334155;
  font-size: 13px;
  font-weight: 800;
}

.commands-head button,
.command-row button {
  height: 28px;
  border: 1px solid #cbd5e1;
  border-radius: 7px;
  background: #f8fafc;
  color: #334155;
  padding: 0 10px;
  font: inherit;
  font-size: 12px;
  font-weight: 800;
  cursor: pointer;
}

.command-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.command-row {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: #f8fafc;
}

.command-top {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto;
  align-items: center;
  gap: 8px;
}

.default-toggle.active {
  color: #1d4ed8;
  background: #eff6ff;
  border-color: #93c5fd;
}

.dialog input:focus {
  border-color: #2563eb;
  box-shadow: 0 0 0 3px #dbeafe;
}

.btns {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 4px;
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
