<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'

const props = defineProps({
  x: { type: Number, default: 0 },
  y: { type: Number, default: 0 },
  fields: { type: Object, required: true },
})
const emit = defineEmits(['toggle', 'close'])
const menuRef = ref(null)

const options = [
  { key: 'type', label: '项目类型' },
  { key: 'branch', label: '分支' },
  { key: 'path', label: '路径' },
  { key: 'command', label: '启动命令' },
]

const style = computed(() => {
  const width = 176
  const height = 184
  const viewportWidth = typeof window === 'undefined' ? width : window.innerWidth
  const viewportHeight = typeof window === 'undefined' ? height : window.innerHeight
  return {
    left: `${Math.max(0, Math.min(props.x, viewportWidth - width))}px`,
    top: `${Math.max(0, Math.min(props.y, viewportHeight - height))}px`,
  }
})

function onDocumentMouseDown(event) {
  if (menuRef.value && !menuRef.value.contains(event.target)) emit('close')
}

function onDocumentKeyDown(event) {
  if (event.key === 'Escape') emit('close')
}

onMounted(() => {
  document.addEventListener('mousedown', onDocumentMouseDown, true)
  document.addEventListener('keydown', onDocumentKeyDown)
})

onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onDocumentMouseDown, true)
  document.removeEventListener('keydown', onDocumentKeyDown)
})
</script>

<template>
  <div
    ref="menuRef"
    class="field-menu"
    :style="style"
    role="menu"
    aria-label="卡片显示字段"
    data-test="project-card-field-menu"
    @contextmenu.prevent
  >
    <div class="menu-title">显示字段</div>
    <label class="field-option fixed-option">
      <input type="checkbox" checked disabled />
      <span>项目名与运行状态</span>
    </label>
    <label v-for="option in options" :key="option.key" class="field-option">
      <input
        type="checkbox"
        :checked="fields[option.key]"
        @change="emit('toggle', option.key)"
      />
      <span>{{ option.label }}</span>
    </label>
  </div>
</template>

<style scoped>
.field-menu {
  box-sizing: border-box;
  position: fixed;
  z-index: 1000;
  width: 176px;
  padding: var(--space-3);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  background: var(--surface);
  box-shadow: var(--shadow-lg), var(--surface-highlight);
}

.menu-title {
  padding: var(--space-2) var(--space-3) var(--space-3);
  color: var(--text-muted);
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
}

.field-option {
  box-sizing: border-box;
  width: 100%;
  min-height: 28px;
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  font-size: var(--fs-sm);
  cursor: pointer;
}

.field-option:hover { color: var(--text); background: var(--elevated-gradient); }
.field-option:focus-within { box-shadow: inset 0 0 0 2px var(--focus-ring); }
.field-option input { margin: 0; accent-color: var(--accent); }
.fixed-option { color: var(--text-muted); cursor: default; }
.fixed-option:hover { color: var(--text-muted); background: transparent; }
</style>
