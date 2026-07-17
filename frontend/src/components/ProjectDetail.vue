<script setup>
import LogPanel from './LogPanel.vue'
const props = defineProps({ project: Object, status: Object })
const emit = defineEmits(['start', 'stop', 'edit'])
</script>

<template>
  <div class="detail" v-if="project">
    <div class="bar">
      <div class="info">
        <strong>{{ project.name }}</strong>
        <span class="type">{{ project.detectedType }}</span>
        <code>{{ project.command }} {{ (project.args || []).join(' ') }}</code>
      </div>
      <div class="btns">
        <button :disabled="(status || {}).State === 'running'" @click="emit('start')">▶ 启动</button>
        <button :disabled="(status || {}).State !== 'running'" @click="emit('stop')">■ 停止</button>
        <button @click="emit('edit')">编辑</button>
      </div>
    </div>
    <LogPanel :projectId="project.id" :status="status" />
  </div>
  <div class="detail empty" v-else>选择一个项目</div>
</template>

<style scoped>
.detail { flex: 1; display: flex; flex-direction: column; background: #fff; }
.detail.empty { align-items: center; justify-content: center; color: #888; }
.bar { display: flex; justify-content: space-between; align-items: center;
  padding: 10px 14px; border-bottom: 1px solid #ddd; background: #fff; }
.info { display: flex; flex-direction: column; gap: 4px; }
.type { color: #666; font-size: 12px; }
.btns { display: flex; gap: 8px; }
</style>
