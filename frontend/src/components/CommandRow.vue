<script setup>
import { computed, nextTick, ref, watch } from 'vue'
import { FolderOpen } from 'lucide-vue-next'
import { PickDirectoryFrom, ListPackageScripts } from '../../wailsjs/go/main/App'
import AppButton from './ui/AppButton.vue'
import AppIcon from './ui/AppIcon.vue'
import { scriptCompleteContext, filterScripts, applyScript } from '../scriptComplete.js'

// cmd 是 commandFormsForProject 产出的表单对象,直接就地修改(与 EditProjectDialog
// 里 v-model 到数组元素的既有写法一致),不走 update:modelValue。
const props = defineProps({
  cmd: { type: Object, required: true },
  projectPath: { type: String, default: '' },
  canRemove: { type: Boolean, default: true },
})
const emit = defineEmits(['set-default', 'remove'])

const suggestList = ref(null)
const suggestions = ref([])
const activeIndex = ref(0)
const open = ref(false)

// cwd 留空时命令实际跑在项目根目录,补全也应该读那里的 package.json。
const effectiveCwd = computed(() => (props.cmd.cwd || '').trim() || props.projectPath || '')

// 按目录缓存 scripts,避免每次按键都过一遍 IPC + 读盘。对话框关闭即随组件销毁,
// 下次打开重新读,不会拿到过期的 package.json。
const scriptCache = new Map()
// 每次请求带一个自增令牌,回来时如果输入已经变了就丢弃结果,避免竞态覆盖。
let requestToken = 0

async function scriptsFor(dir) {
  if (scriptCache.has(dir)) return scriptCache.get(dir)
  const pending = ListPackageScripts(dir)
    .then((list) => (Array.isArray(list) ? list : []))
    .catch(() => [])
  scriptCache.set(dir, pending)
  return pending
}

function close() {
  open.value = false
  suggestions.value = []
  activeIndex.value = 0
}

async function refresh() {
  const ctx = scriptCompleteContext(props.cmd.line)
  if (!ctx) {
    close()
    return
  }
  const token = ++requestToken
  const scripts = await scriptsFor(effectiveCwd.value)
  if (token !== requestToken) return
  const matched = filterScripts(scripts, ctx.prefix)
  suggestions.value = matched
  activeIndex.value = 0
  open.value = matched.length > 0
  if (!open.value) return
  // 对话框本身是 overflow-y: auto 的滚动容器,浮层会被它裁掉;
  // 展开后把浮层滚进可视区,靠近底部的命令也能看全候选。
  await nextTick()
  if (suggestList.value) suggestList.value.scrollIntoView({ block: 'nearest' })
}

// 改了工作目录就换了一份 package.json,已展开的候选立即失效。
watch(effectiveCwd, () => {
  if (open.value) refresh()
})

// 候选项用 @mousedown.prevent 拦掉了默认的失焦行为,键盘选中本来就没离开输入框,
// 所以这里不需要(也不该)再 focus 一次 —— 重新 focus 会触发 @focus="refresh",
// 把刚补全好的完整脚本名又当成前缀重新展开一遍浮层。
function choose(script) {
  props.cmd.line = applyScript(props.cmd.line, script.name)
  close()
}

async function move(delta) {
  if (!open.value || !suggestions.value.length) return
  const size = suggestions.value.length
  activeIndex.value = (activeIndex.value + delta + size) % size
  await nextTick()
  const el = suggestList.value && suggestList.value.children[activeIndex.value]
  if (el) el.scrollIntoView({ block: 'nearest' })
}

function onKeydown(event) {
  if (event.key === 'Escape' && open.value) {
    event.stopPropagation() // 别让 Esc 冒泡到对话框把整个弹窗关掉
    close()
    return
  }
  if (!open.value || !suggestions.value.length) return
  if (event.key === 'ArrowDown') {
    event.preventDefault()
    move(1)
  } else if (event.key === 'ArrowUp') {
    event.preventDefault()
    move(-1)
  } else if (event.key === 'Enter' || event.key === 'Tab') {
    event.preventDefault()
    choose(suggestions.value[activeIndex.value])
  }
}

async function pickDir() {
  const dir = await PickDirectoryFrom(effectiveCwd.value)
  if (dir) props.cmd.cwd = dir
}
</script>

<template>
  <div class="command-row">
    <div class="command-top">
      <input v-model="cmd.name" placeholder="Name" />
      <AppButton
        :variant="cmd.isDefault ? 'primary' : 'secondary'"
        size="sm"
        @click="emit('set-default')"
      >Default</AppButton>
      <AppButton
        variant="secondary"
        size="sm"
        :disabled="!canRemove"
        @click="emit('remove')"
      >Remove</AppButton>
    </div>

    <div class="field-with-action">
      <input v-model="cmd.cwd" :placeholder="projectPath" :title="cmd.cwd || projectPath" />
      <button type="button" class="action-btn" title="选择工作目录" @click="pickDir">
        <AppIcon :icon="FolderOpen" :size="14" />
      </button>
    </div>

    <div class="field-with-suggest">
      <input
        v-model="cmd.line"
        placeholder="如 pnpm run dev 或 go run main.go serve"
        autocomplete="off"
        spellcheck="false"
        @input="refresh"
        @focus="refresh"
        @blur="close"
        @keydown="onKeydown"
      />
      <ul v-if="open" ref="suggestList" class="suggest">
        <li
          v-for="(script, i) in suggestions"
          :key="script.name"
          :class="{ active: i === activeIndex }"
          @mousedown.prevent="choose(script)"
          @mouseenter="activeIndex = i"
        >
          <span class="suggest-name">{{ script.name }}</span>
          <span class="suggest-script" :title="script.script">{{ script.script }}</span>
        </li>
      </ul>
    </div>

    <textarea
      v-model="cmd.envText"
      spellcheck="false"
      placeholder="环境变量,每行一个: KEY=value"
    />
  </div>
</template>

<style scoped>
.command-row {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  padding: var(--space-5);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg);
}

.command-top {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto;
  align-items: center;
  gap: var(--space-4);
}

input,
textarea {
  width: 100%;
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

textarea {
  min-height: 76px;
  resize: vertical;
  padding: var(--space-4) var(--space-5);
  line-height: 1.45;
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
}

input:focus,
textarea:focus {
  border-color: var(--text-subtle);
  box-shadow: 0 0 0 3px var(--focus-ring);
}

.field-with-action {
  position: relative;
}

.field-with-action input {
  padding-right: 34px;
  text-overflow: ellipsis;
}

.action-btn {
  position: absolute;
  top: 50%;
  right: var(--space-2);
  transform: translateY(-50%);
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  padding: 0;
  border: 0;
  border-radius: var(--radius-sm);
  color: var(--text-muted);
  background: transparent;
  cursor: pointer;
  transition: color var(--dur-fast) var(--ease), background var(--dur-fast) var(--ease);
}

.action-btn:hover {
  color: var(--text);
  background: var(--elevated);
}

.action-btn:focus-visible {
  outline: none;
  box-shadow: 0 0 0 3px var(--focus-ring);
}

.field-with-suggest {
  position: relative;
}

.suggest {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  right: 0;
  z-index: var(--z-menu);
  max-height: 184px;
  overflow-y: auto;
  margin: 0;
  padding: var(--space-2);
  list-style: none;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: var(--surface);
  box-shadow: var(--shadow-lg);
}

.suggest li {
  display: grid;
  grid-template-columns: minmax(0, auto) minmax(0, 1fr);
  align-items: baseline;
  gap: var(--space-4);
  padding: var(--space-2) var(--space-4);
  border-radius: var(--radius-sm);
  cursor: pointer;
}

.suggest li.active {
  background: var(--elevated);
}

.suggest-name {
  color: var(--text);
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
  white-space: nowrap;
}

.suggest-script {
  overflow: hidden;
  color: var(--text-subtle);
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  white-space: nowrap;
  text-overflow: ellipsis;
}
</style>
