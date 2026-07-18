<script setup>
import { ref } from 'vue'
import { ScanWorkspaces, AddScanned, PickDirectory } from '../../wailsjs/go/main/App'
const props = defineProps({ show: Boolean })
const emit = defineEmits(['close', 'added'])

const rootsText = ref('')
const candidates = ref([])
const checked = ref({})

async function scan() {
  const roots = rootsText.value.split('\n').map((s) => s.trim()).filter(Boolean)
  candidates.value = await ScanWorkspaces(roots)
  checked.value = {}
  candidates.value.forEach((c) => { checked.value[c.id] = c.detectedType !== 'unknown' })
}

async function pickDir() {
  const dir = await PickDirectory()
  if (!dir) return // 用户取消
  // 追加到文本框(去重,避免重复行)
  const lines = rootsText.value.split('\n').map((s) => s.trim()).filter(Boolean)
  if (!lines.includes(dir)) lines.push(dir)
  rootsText.value = lines.join('\n')
  await scan() // 选中后立即扫描
}

async function add() {
  const chosen = candidates.value.filter((c) => checked.value[c.id])
  await AddScanned(chosen)
  emit('added')
  emit('close')
}

function toggle(id) {
  checked.value = { ...checked.value, [id]: !checked.value[id] }
}
</script>

<template>
  <div class="mask" v-if="show" @click.self="emit('close')">
    <div class="dialog">
      <h3>扫描工作区</h3>
      <textarea v-model="rootsText" rows="3"
        placeholder="每行一个根目录,支持 ~,如&#10;~/GolandProjects&#10;~/WebstormProjects"></textarea>
      <div class="root-actions">
        <button @click="pickDir">📁 选择文件夹</button>
        <button class="primary" @click="scan">扫描</button>
      </div>
      <div class="results">
        <button v-for="c in candidates" :key="c.id" :class="['row', { selected: checked[c.id] }]" @click="toggle(c.id)">
          <span class="check-mark">{{ checked[c.id] ? '✓' : '' }}</span>
          <span class="nm">{{ c.name }}</span>
          <span class="ty" :class="{ unknown: c.detectedType === 'unknown' }">{{ c.detectedType }}</span>
          <code>{{ [c.command, ...(c.args || [])].join(' ').trim() || '—' }}</code>
        </button>
      </div>
      <div class="btns">
        <button @click="emit('close')">取消</button>
        <button @click="add">加入选中</button>
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
  width: min(760px, calc(100vw - 36px));
  display: flex;
  flex-direction: column;
  gap: 12px;
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

textarea {
  box-sizing: border-box;
  width: 100%;
  border: 1px solid #cbd5e1;
  border-radius: 7px;
  padding: 10px;
  color: #0f172a;
  font: inherit;
  font-size: 13px;
  resize: vertical;
  outline: none;
}

textarea:focus {
  border-color: #2563eb;
  box-shadow: 0 0 0 3px #dbeafe;
}

.root-actions {
  display: flex;
  gap: 8px;
}

.root-actions button,
.btns button {
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

.root-actions button {
  flex: 1;
}

.root-actions .primary,
.btns button:last-child {
  color: #ffffff;
  background: #2563eb;
  border-color: #2563eb;
}

.results {
  max-height: 340px;
  overflow-y: auto;
  border: 1px solid #e2e8f0;
  border-radius: 7px;
}

.row {
  width: 100%;
  display: grid;
  grid-template-columns: 18px minmax(160px, 220px) 92px minmax(0, 1fr);
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border: 0;
  font-size: 13px;
  font: inherit;
  text-align: left;
  background: #ffffff;
  border-bottom: 1px solid #f1f5f9;
  cursor: pointer;
}

.row:last-child {
  border-bottom: 0;
}

.row.selected {
  background: #eff6ff;
}

.check-mark {
  width: 16px;
  height: 16px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid #cbd5e1;
  border-radius: 5px;
  color: #ffffff;
  background: #ffffff;
  font-size: 11px;
  font-weight: 900;
}

.row.selected .check-mark {
  border-color: #2563eb;
  background: #2563eb;
}

.nm {
  color: #111827;
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ty {
  color: #2563eb;
  font-size: 12px;
  font-weight: 700;
}

.ty.unknown {
  color: #94a3b8;
}

.row code {
  color: #475569;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
}

.btns {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
