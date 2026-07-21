<script setup>
import { computed, ref, watch } from 'vue'
import { buildProjectTree } from '../projectTree'
import GroupTreeNode from './GroupTreeNode.vue'
import AppButton from './ui/AppButton.vue'

const props = defineProps({ show: Boolean, group: Object, projects: Array })
const emit = defineEmits(['save', 'close'])

const name = ref('')
const checked = ref({})
const expandedDirs = ref({})
const query = ref('')

// 搜索时强制展开,避免匹配结果被折叠状态藏住。
const forceExpanded = computed(() => query.value.trim().length > 0)

// 目录默认展开;记录为 false 表示用户手动折叠。
function toggleDir(id) {
  expandedDirs.value = {
    ...expandedDirs.value,
    [id]: expandedDirs.value[id] === false,
  }
}

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
const projectTree = computed(() => buildProjectTree(props.projects || [], {}, query.value))

function reset() {
  name.value = props.group ? props.group.name : ''
  const next = {}
  for (const item of (props.group && props.group.items) || []) {
    next[keyFor(item.projectId, item.commandId)] = true
  }
  checked.value = next
  expandedDirs.value = {}
  query.value = ''
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
  <Transition name="dlg-fade">
    <div class="mask" v-if="show" @click.self="emit('close')">
      <Transition name="dlg-pop" appear>
        <div class="dialog">
          <h3>{{ group ? '编辑分组' : '新建分组' }}</h3>
          <label>名称<input v-model="name" placeholder="Local dev stack" /></label>

          <div class="commands-head">选择要启动的项目命令</div>
          <input
            v-model="query"
            class="option-search"
            placeholder="搜索项目名称、路径或命令…"
          />
          <div class="options">
            <GroupTreeNode
              v-for="node in projectTree"
              :key="node.id"
              :node="node"
              :level="0"
              :checked="checked"
              :expandedDirs="expandedDirs"
              :forceExpanded="forceExpanded"
              @toggle="toggleOption"
              @toggle-dir="toggleDir"
            />
            <div v-if="projectTree.length === 0" class="options-empty">
              没有匹配的项目
            </div>
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
  width: min(820px, calc(100vw - 36px));
  max-height: calc(100vh - 56px);
  /* No outer scroll: the project list (.options) scrolls internally and
     flex-shrinks to fit. A second scroll layer here made WebKitGTK paint a
     stray vertical scrollbar track through the header region. */
  overflow: hidden;
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

.dialog > label input {
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

.dialog > label input:focus {
  border-color: var(--text-subtle);
  box-shadow: 0 0 0 3px var(--focus-ring);
}

.commands-head {
  color: var(--text-secondary);
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
}

.option-search {
  height: 32px;
  box-sizing: border-box;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  color: var(--text);
  background: var(--bg);
  padding: 0 var(--space-5);
  font: inherit;
  font-size: var(--fs-sm);
  outline: none;
  transition: border-color var(--dur-fast) var(--ease), box-shadow var(--dur-fast) var(--ease);
}

.option-search:focus {
  border-color: var(--text-subtle);
  box-shadow: 0 0 0 3px var(--focus-ring);
}

.options {
  flex: 1 1 auto;
  min-height: 120px;
  max-height: 400px;
  overflow-y: auto;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg);
}

.options-empty {
  padding: var(--space-7) var(--space-5);
  color: var(--text-muted);
  font-size: var(--fs-sm);
  text-align: center;
}

.btns {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
}
</style>
