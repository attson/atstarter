<script setup>
import { onMounted, onBeforeUnmount, ref, computed } from 'vue'
import AppIcon from './AppIcon.vue'

const props = defineProps({
  x: { type: Number, default: 0 },
  y: { type: Number, default: 0 },
  items: { type: Array, default: () => [] }, // [{ key, label, icon, danger, disabled }]
})
const emit = defineEmits(['select', 'close'])

const menuRef = ref(null)

// 视口边界内收敛
const style = computed(() => {
  const vw = typeof window !== 'undefined' ? window.innerWidth : 0
  const vh = typeof window !== 'undefined' ? window.innerHeight : 0
  const left = Math.min(props.x, Math.max(0, vw - 200))
  const top = Math.min(props.y, Math.max(0, vh - 40 - props.items.length * 32))
  return { left: `${left}px`, top: `${top}px` }
})

function onSelect(item) {
  if (item.disabled) return
  emit('select', item.key)
  emit('close')
}

function onDocClick(e) {
  if (menuRef.value && !menuRef.value.contains(e.target)) emit('close')
}
function onKey(e) {
  if (e.key === 'Escape') emit('close')
}
function onScroll() {
  emit('close')
}

onMounted(() => {
  document.addEventListener('mousedown', onDocClick, true)
  document.addEventListener('keydown', onKey)
  window.addEventListener('scroll', onScroll, true)
})
onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onDocClick, true)
  document.removeEventListener('keydown', onKey)
  window.removeEventListener('scroll', onScroll, true)
})
</script>

<template>
  <div ref="menuRef" class="context-menu" :style="style" @contextmenu.prevent>
    <button
      v-for="item in items"
      :key="item.key"
      class="menu-item"
      :class="{ danger: item.danger, disabled: item.disabled }"
      :disabled="item.disabled"
      @click="onSelect(item)"
    >
      <AppIcon v-if="item.icon" :icon="item.icon" :size="14" class="menu-icon" />
      <span class="menu-label">{{ item.label }}</span>
    </button>
  </div>
</template>

<style scoped>
.context-menu {
  position: fixed;
  z-index: 1000;
  min-width: 160px;
  padding: var(--space-2);
  background: var(--surface);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg, 0 8px 24px rgba(0, 0, 0, .25));
}

.menu-item {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  width: 100%;
  border: 0;
  background: transparent;
  color: var(--text-secondary);
  font: inherit;
  font-size: var(--fs-sm);
  text-align: left;
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background var(--dur-fast) var(--ease);
}

.menu-item:hover:not(.disabled) { background: var(--elevated-gradient); color: var(--text); }
.menu-item.danger { color: var(--danger, #ef4444); }
.menu-item.disabled { opacity: .5; cursor: default; }
.menu-icon { flex: 0 0 auto; color: var(--text-muted); }
.menu-item.danger .menu-icon { color: var(--danger, #ef4444); }
.menu-label { flex: 1 1 auto; }
</style>
