<script setup>
import { ref } from 'vue'
import { FolderOpen } from 'lucide-vue-next'
import { ScanWorkspaces, AddScanned, PickDirectory } from '../../wailsjs/go/main/App'
import AppButton from './ui/AppButton.vue'
import AppIcon from './ui/AppIcon.vue'
import { typeLabel } from '../typeLabel.js'

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
  if (!dir) return
  const lines = rootsText.value.split('\n').map((s) => s.trim()).filter(Boolean)
  if (!lines.includes(dir)) lines.push(dir)
  rootsText.value = lines.join('\n')
  await scan()
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
  <Transition name="dlg-fade">
    <div class="mask" v-if="show" @click.self="emit('close')">
      <Transition name="dlg-pop" appear>
        <div class="dialog">
          <h3>扫描工作区</h3>
          <textarea v-model="rootsText" rows="3"
            placeholder="每行一个根目录，支持 ~，如&#10;~/GolandProjects&#10;~/WebstormProjects"></textarea>
          <div class="root-actions">
            <AppButton variant="secondary" @click="pickDir">
              <template #icon><AppIcon :icon="FolderOpen" :size="14" /></template>
              选择文件夹
            </AppButton>
            <AppButton variant="primary" @click="scan">扫描</AppButton>
          </div>
          <div class="results">
            <button v-for="c in candidates" :key="c.id" :class="['row', { selected: checked[c.id] }]" @click="toggle(c.id)">
              <span class="check-mark">{{ checked[c.id] ? '✓' : '' }}</span>
              <span class="nm">{{ c.name }}</span>
              <span class="ty" :class="{ unknown: c.detectedType === 'unknown' }">{{ typeLabel(c.detectedType) }}</span>
              <code>{{ [c.command, ...(c.args || [])].join(' ').trim() || '—' }}</code>
            </button>
          </div>
          <div class="btns">
            <AppButton variant="secondary" @click="emit('close')">取消</AppButton>
            <AppButton variant="primary" @click="add">加入选中</AppButton>
          </div>
        </div>
      </Transition>
    </div>
  </Transition>
</template>

<style scoped>
.mask {
  position: fixed;
  inset: 0;
  z-index: var(--z-modal);
  background: var(--overlay);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
}

.dialog {
  width: min(760px, calc(100vw - 36px));
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
  padding: var(--space-9);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--surface);
  box-shadow: var(--shadow-lg);
}

h3 {
  margin: 0 0 var(--space-2);
  color: var(--text);
  font-size: var(--fs-md);
  font-weight: var(--fw-semibold);
  letter-spacing: -0.005em;
}

textarea {
  box-sizing: border-box;
  width: 100%;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-md);
  padding: var(--space-5);
  color: var(--text);
  background: var(--bg);
  font: inherit;
  font-size: var(--fs-sm);
  resize: vertical;
  outline: none;
  transition: border-color var(--dur-fast) var(--ease), box-shadow var(--dur-fast) var(--ease);
}

textarea:focus {
  border-color: var(--text-subtle);
  box-shadow: 0 0 0 3px var(--focus-ring);
}

.root-actions {
  display: flex;
  gap: var(--space-3);
}

.root-actions > * { flex: 1; }

.results {
  max-height: 340px;
  overflow-y: auto;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg);
}

.row {
  width: 100%;
  display: grid;
  grid-template-columns: 18px minmax(160px, 220px) 92px minmax(0, 1fr);
  align-items: center;
  gap: var(--space-5);
  padding: var(--space-4) var(--space-5);
  border: 0;
  font-size: var(--fs-sm);
  font: inherit;
  text-align: left;
  background: transparent;
  color: var(--text-secondary);
  border-bottom: 1px solid var(--border);
  cursor: pointer;
  transition: background var(--dur-fast) var(--ease);
}

.row:last-child { border-bottom: 0; }
.row:hover { background: var(--elevated); }
.row.selected { background: var(--elevated); }

.check-mark {
  width: 16px;
  height: 16px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-sm);
  color: var(--primary-fg);
  background: transparent;
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
}

.row.selected .check-mark {
  border-color: var(--primary);
  background: var(--primary);
}

.nm {
  color: var(--text);
  font-weight: var(--fw-semibold);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ty {
  color: var(--text-secondary);
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
}

.ty.unknown { color: var(--text-subtle); }

.row code {
  color: var(--text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: var(--font-mono);
  font-size: var(--fs-mono);
}

.btns {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
}
</style>
