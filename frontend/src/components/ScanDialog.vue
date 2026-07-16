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
        <label v-for="c in candidates" :key="c.id" class="row">
          <input type="checkbox" v-model="checked[c.id]" />
          <span class="nm">{{ c.name }}</span>
          <span class="ty">{{ c.detectedType }}</span>
          <code>{{ c.command }} {{ (c.args || []).join(' ') }}</code>
        </label>
      </div>
      <div class="btns">
        <button @click="emit('close')">取消</button>
        <button @click="add">加入选中</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.mask { position: fixed; inset: 0; background: rgba(0,0,0,.4);
  display: flex; align-items: center; justify-content: center; }
.dialog { background: #fff; padding: 20px; border-radius: 8px; width: 620px;
  display: flex; flex-direction: column; gap: 10px; }
.root-actions { display: flex; gap: 8px; }
.root-actions button { flex: 1; padding: 6px; }
.root-actions .primary { background: #e8f0fe; }
.results { max-height: 320px; overflow-y: auto; border: 1px solid #eee; }
.row { display: flex; align-items: center; gap: 10px; padding: 6px 8px; font-size: 13px; }
.ty { color: #666; }
.btns { display: flex; justify-content: flex-end; gap: 8px; }
</style>
