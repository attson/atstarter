<script setup>
import { ref } from 'vue'
import { ScanWorkspaces, AddScanned } from '../../wailsjs/go/main/App'
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
        placeholder="每行一个根目录,如&#10;/home/attson/GolandProjects&#10;/home/attson/WebstormProjects"></textarea>
      <button @click="scan">扫描</button>
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
.results { max-height: 320px; overflow-y: auto; border: 1px solid #eee; }
.row { display: flex; align-items: center; gap: 10px; padding: 6px 8px; font-size: 13px; }
.ty { color: #666; }
.btns { display: flex; justify-content: flex-end; gap: 8px; }
</style>
