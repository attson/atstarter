<script setup>
import { ref, watch } from 'vue'
import AppButton from './ui/AppButton.vue'
import CommandRow from './CommandRow.vue'
import { commandFormsForProject, blankCommandForm } from '../commandForms.js'
import { envTextToMap } from '../envVars.js'

const props = defineProps({ project: Object, show: Boolean })
const emit = defineEmits(['save', 'close'])

const name = ref('')
const commands = ref([])

function reset(p) {
  if (!p) return
  name.value = p.name
  commands.value = commandFormsForProject(p)
}

watch(() => props.project, (p) => {
  reset(p)
}, { immediate: true })

function save() {
  emit('save', {
    name: name.value,
    commands: commands.value.map((cmd) => ({
      ...cmd,
      env: envTextToMap(cmd.envText),
    })),
  })
}

function addCommand() {
  commands.value.push(blankCommandForm(props.project, commands.value.length))
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
  <Transition name="dlg-fade">
    <div class="mask" v-if="show" @click.self="emit('close')">
      <Transition name="dlg-pop" appear>
        <div class="dialog">
          <h3>编辑项目</h3>
          <label>名称<input v-model="name" /></label>
          <div class="commands-head">
            <span>启动命令</span>
            <AppButton variant="secondary" size="sm" @click="addCommand">Add command</AppButton>
          </div>
          <div class="command-list">
            <CommandRow
              v-for="(cmd, index) in commands"
              :key="cmd.id || index"
              :cmd="cmd"
              :project-path="(project && project.path) || ''"
              :can-remove="commands.length > 1"
              @set-default="setDefault(index)"
              @remove="removeCommand(index)"
            />
          </div>
          <div class="btns">
            <AppButton variant="secondary" @click="emit('close')">取消</AppButton>
            <AppButton variant="primary" @click="save">保存</AppButton>
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
  width: min(720px, calc(100vw - 36px));
  max-height: calc(100vh - 56px);
  overflow-y: auto;
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
  margin: 0 0 var(--space-2);
  color: var(--text);
  font-size: var(--fs-md);
  font-weight: var(--fw-semibold);
  letter-spacing: -0.005em;
}

.dialog label {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  color: var(--text-secondary);
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
}

.dialog input {
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

.dialog input:focus {
  border-color: var(--text-subtle);
  box-shadow: 0 0 0 3px var(--focus-ring);
}

.commands-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: var(--text-secondary);
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
}

.command-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.btns {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
  margin-top: var(--space-2);
}
</style>
