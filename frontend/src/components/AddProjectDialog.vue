<script setup>
import { ref, watch } from 'vue'
import { PickDirectory } from '../../wailsjs/go/main/App'

const props = defineProps({ show: Boolean })
const emit = defineEmits(['close', 'save'])

const path = ref('')

watch(() => props.show, (show) => {
  if (show) path.value = ''
})

async function pickDir() {
  const dir = await PickDirectory()
  if (dir) path.value = dir
}

function save() {
  const value = path.value.trim()
  if (!value) return
  emit('save', value)
}
</script>

<template>
  <div class="mask" v-if="show" @click.self="emit('close')">
    <div class="dialog">
      <h3>添加项目</h3>
      <label>项目目录<input v-model="path" placeholder="/home/attson/GolandProjects/atstarter" /></label>
      <div class="inline-actions">
        <button @click="pickDir">选择文件夹</button>
      </div>
      <div class="btns">
        <button @click="emit('close')">取消</button>
        <button :disabled="!path.trim()" @click="save">添加</button>
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
  width: min(520px, calc(100vw - 36px));
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

label {
  display: flex;
  flex-direction: column;
  gap: 6px;
  color: #334155;
  font-size: 13px;
  font-weight: 700;
}

input {
  height: 34px;
  border: 1px solid #cbd5e1;
  border-radius: 7px;
  color: #0f172a;
  padding: 0 10px;
  font: inherit;
  outline: none;
}

input:focus {
  border-color: #2563eb;
  box-shadow: 0 0 0 3px #dbeafe;
}

.inline-actions,
.btns {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

button {
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

button:disabled {
  cursor: not-allowed;
  opacity: .48;
}
</style>
