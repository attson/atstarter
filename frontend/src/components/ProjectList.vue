<script setup>
defineProps({ projects: Array, selectedId: String, statuses: Object })
const emit = defineEmits(['select', 'add', 'scan'])

function dot(state) {
  if (state === 'running') return '#4caf50'
  if (state === 'error') return '#f44336'
  if (state === 'exited') return '#f44336'
  return '#888'
}
</script>

<template>
  <div class="list">
    <div class="items">
      <div v-for="p in projects" :key="p.id"
           :class="['item', { active: p.id === selectedId }]"
           @click="emit('select', p.id)">
        <span class="dot" :style="{ background: dot((statuses[p.id] || {}).State) }"></span>
        <span class="name">{{ p.name }}</span>
      </div>
    </div>
    <div class="actions">
      <button @click="emit('add')">+ 添加</button>
      <button @click="emit('scan')">扫描</button>
    </div>
  </div>
</template>

<style scoped>
.list { width: 240px; border-right: 1px solid #ddd; display: flex; flex-direction: column; background: #fff; }
.items { flex: 1; overflow-y: auto; background: #fff; }
.item { display: flex; align-items: center; gap: 8px; padding: 8px 12px; cursor: pointer; background: #fff; }
.item.active { background: #e8f0fe; }
.dot { width: 10px; height: 10px; border-radius: 50%; display: inline-block; }
.actions { display: flex; gap: 8px; padding: 8px; border-top: 1px solid #ddd; }
.actions button { flex: 1; }
</style>
