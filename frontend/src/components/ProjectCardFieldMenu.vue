<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = defineProps({
  x: { type: Number, default: 0 },
  y: { type: Number, default: 0 },
  fields: { type: Object, required: true },
})
const emit = defineEmits(['toggle', 'close'])
const menuRef = ref(null)

const options = computed(() => [
  { key: 'type', label: t('projects.card.fieldType') },
  { key: 'branch', label: t('projects.card.fieldBranch') },
  { key: 'path', label: t('projects.card.fieldPath') },
  { key: 'command', label: t('projects.card.fieldCommand') },
])

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
    role="group"
    :aria-label="t('projects.card.fieldsLabel')"
    data-test="project-card-field-menu"
    @contextmenu.prevent
  >
    <div class="menu-title">{{ t('projects.card.displayFields') }}</div>
    <label class="field-option fixed-option">
      <input type="checkbox" checked disabled />
      <span>{{ t('projects.card.nameAndStatus') }}</span>
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
