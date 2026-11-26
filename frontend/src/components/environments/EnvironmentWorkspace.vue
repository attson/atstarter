<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { FolderTree, GitBranch, LayoutDashboard, SlidersHorizontal } from 'lucide-vue-next'
import AppIcon from '../ui/AppIcon.vue'
import EnvironmentOverview from './EnvironmentOverview.vue'
import {
  CancelEnvironmentScan,
  GetEnvironmentDocument,
  GetEnvironmentState,
  StartEnvironmentScan,
} from '../../../wailsjs/go/main/App'
import { EventsOff, EventsOn } from '../../../wailsjs/runtime/runtime'
import { normalizeEnvironmentDocument } from '../../projectEnvironmentUi.js'

const document = ref(normalizeEnvironmentDocument())
const state = ref({ version: 1, ruleScans: [], repositories: [] })
const activeView = ref('overview')
const loading = ref(true)
const error = ref('')
const scanID = ref('')
const scanProgress = ref({})
const plans = ref([])

const views = [
  { id: 'overview', label: 'Overview', icon: LayoutDashboard },
  { id: 'presets', label: 'Presets', icon: SlidersHorizontal },
  { id: 'rules', label: 'Directory Rules', icon: FolderTree },
  { id: 'repositories', label: 'Repositories', icon: GitBranch },
]

const currentTitle = computed(() => views.find((view) => view.id === activeView.value)?.label || '')

async function refreshEnvironmentData() {
  error.value = ''
  try {
    const [nextDocument, nextState] = await Promise.all([GetEnvironmentDocument(), GetEnvironmentState()])
    document.value = normalizeEnvironmentDocument(nextDocument)
    state.value = nextState || { version: 1, ruleScans: [], repositories: [] }
  } catch (cause) {
    error.value = String(cause)
  } finally {
    loading.value = false
  }
}

async function startScan() {
  if (scanID.value) return
  error.value = ''
  scanProgress.value = {}
  try {
    scanID.value = await StartEnvironmentScan([])
  } catch (cause) {
    error.value = String(cause)
  }
}

function onScanProgress(payload) {
  if (!payload || payload.scanId !== scanID.value) return
  scanProgress.value = payload.progress || {}
}

async function onScanComplete(payload) {
  if (!payload || payload.scanId !== scanID.value) return
  scanID.value = ''
  if (payload.error && !payload.result?.canceled) error.value = payload.error
  await refreshEnvironmentData()
}

onMounted(async () => {
  EventsOn('projectenv:scan-progress', onScanProgress)
  EventsOn('projectenv:scan-complete', onScanComplete)
  await refreshEnvironmentData()
})

onUnmounted(() => {
  EventsOff('projectenv:scan-progress')
  EventsOff('projectenv:scan-complete')
  if (scanID.value) CancelEnvironmentScan(scanID.value)
})
</script>

<template>
  <div class="environment-workspace">
    <aside class="environment-sidebar" aria-label="Environment views">
      <button
        v-for="view in views"
        :key="view.id"
        type="button"
        class="environment-nav-item"
        :class="{ active: activeView === view.id }"
        @click="activeView = view.id"
      >
        <AppIcon :icon="view.icon" :size="15" />
        <span>{{ view.label }}</span>
      </button>
    </aside>

    <div class="environment-content">
      <div v-if="loading" class="environment-state-message">Loading environments...</div>
      <div v-else-if="error" class="environment-state-message error" role="alert">{{ error }}</div>
      <EnvironmentOverview
        v-else-if="activeView === 'overview'"
        :document="document"
        :state="state"
        :plans="plans"
        :scanning="Boolean(scanID)"
        :progress="scanProgress"
        @scan="startScan"
        @open-repositories="activeView = 'repositories'"
      />
      <section v-else class="environment-placeholder">
        <h1>{{ currentTitle }}</h1>
      </section>
    </div>
  </div>
</template>

<style scoped>
.environment-workspace { width: 100%; min-width: 0; min-height: 0; display: grid; grid-template-columns: 188px minmax(0, 1fr); background: var(--bg); }
.environment-sidebar { padding: var(--space-6); border-right: 1px solid var(--border); background: var(--surface); }
.environment-nav-item { width: 100%; height: 34px; display: grid; grid-template-columns: 20px minmax(0, 1fr); align-items: center; gap: var(--space-4); padding: 0 var(--space-5); border: 1px solid transparent; border-radius: var(--radius-sm); background: transparent; color: var(--text-muted); font: inherit; font-size: var(--fs-sm); text-align: left; cursor: pointer; }
.environment-nav-item:hover { color: var(--text); background: var(--elevated); }
.environment-nav-item.active { color: var(--text); background: var(--elevated); border-color: var(--border); }
.environment-content { min-width: 0; min-height: 0; display: flex; overflow: hidden; }
.environment-state-message { margin: auto; color: var(--text-muted); }
.environment-state-message.error { color: var(--danger); }
.environment-placeholder { width: 100%; padding: var(--space-9); }
.environment-placeholder h1 { margin: 0; font-size: var(--fs-lg); letter-spacing: 0; }
@media (max-width: 760px) { .environment-workspace { grid-template-columns: 52px minmax(0, 1fr); } .environment-nav-item { grid-template-columns: 20px; justify-content: center; padding: 0; } .environment-nav-item span { display: none; } }
</style>
