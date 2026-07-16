<script setup>
import { ref, watch } from 'vue'
const props = defineProps({ project: Object, show: Boolean })
const emit = defineEmits(['save', 'close'])

const name = ref('')
const commandLine = ref('')
const cwd = ref('')

watch(() => props.project, (p) => {
  if (!p) return
  name.value = p.name
  commandLine.value = [p.command, ...(p.args || [])].join(' ')
  cwd.value = p.cwd || ''
}, { immediate: true })

function save() {
  emit('save', { name: name.value, commandLine: commandLine.value, cwd: cwd.value })
}
</script>

<template>
  <div class="mask" v-if="show" @click.self="emit('close')">
    <div class="dialog">
      <h3>编辑项目</h3>
      <label>名称<input v-model="name" /></label>
      <label>启动命令<input v-model="commandLine" placeholder="如 pnpm run dev" /></label>
      <label>工作目录 (可选)<input v-model="cwd" :placeholder="project && project.path" /></label>
      <div class="btns">
        <button @click="emit('close')">取消</button>
        <button @click="save">保存</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.mask { position: fixed; inset: 0; background: rgba(0,0,0,.4);
  display: flex; align-items: center; justify-content: center; }
.dialog { background: #fff; padding: 20px; border-radius: 8px; width: 480px;
  display: flex; flex-direction: column; gap: 12px; }
.dialog label { display: flex; flex-direction: column; gap: 4px; font-size: 13px; }
.dialog input { padding: 6px 8px; }
.btns { display: flex; justify-content: flex-end; gap: 8px; }
</style>
