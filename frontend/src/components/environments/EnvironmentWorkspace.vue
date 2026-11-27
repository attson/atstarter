<script setup>
import { useI18n } from 'vue-i18n'
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { FolderTree, GitBranch, LayoutDashboard, SlidersHorizontal } from 'lucide-vue-next'
import AppIcon from '../ui/AppIcon.vue'
import EnvironmentOverview from './EnvironmentOverview.vue'
import EnvironmentPresetManager from './EnvironmentPresetManager.vue'
import EnvironmentRuleManager from './EnvironmentRuleManager.vue'
import EnvironmentRepositoryManager from './EnvironmentRepositoryManager.vue'
import EnvironmentDiffPanel from './EnvironmentDiffPanel.vue'
import {
  ApplyEnvironmentPlans,
  CancelEnvironmentScan,
  CloneEnvironmentPreset,
  DeleteEnvironmentPreset,
  DeleteEnvironmentRule,
  GetEnvironmentDocument,
  GetEnvironmentState,
  DoctorEnvironments,
  PickDirectoryFrom,
  PlanEnvironmentRepositories,
  SaveEnvironmentDefaults,
  SaveEnvironmentPreset,
  SaveEnvironmentRule,
  StartEnvironmentScan,
} from '../../../wailsjs/go/main/App'
import { EventsOff, EventsOn } from '../../../wailsjs/runtime/runtime'
import { codexModelSuggestions, normalizeEnvironmentDocument } from '../../projectEnvironmentUi.js'

const { t } = useI18n()

const document = ref(normalizeEnvironmentDocument())
const state = ref({ version: 1, ruleScans: [], repositories: [] })
const activeView = ref('overview')
const loading = ref(true)
const error = ref('')
const scanID = ref('')
const scanStarting = ref(false)
const pendingScanCompletions = new Map()
const scanProgress = ref({})
const plans = ref([])
const selectedRepositories = ref(new Set())
const applyResults = ref([])
const planning = ref(false)
const applying = ref(false)
const doctorIssues = ref([])
const modelSuggestions = computed(() => codexModelSuggestions(document.value, plans.value))

const views = [
  { id: 'overview', labelKey: 'environments.views.overview', icon: LayoutDashboard },
  { id: 'presets', labelKey: 'environments.views.presets', icon: SlidersHorizontal },
  { id: 'rules', labelKey: 'environments.views.rules', icon: FolderTree },
  { id: 'repositories', labelKey: 'environments.views.repositories', icon: GitBranch },
]

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

async function startScan(ruleIDs = []) {
  if (scanStarting.value || scanID.value) return
  scanStarting.value = true
  error.value = ''
  scanProgress.value = {}
  try {
    scanID.value = await StartEnvironmentScan(ruleIDs)
    const completion = pendingScanCompletions.get(scanID.value)
    if (completion) await onScanComplete(completion)
  } catch (cause) {
    error.value = String(cause)
  } finally {
    scanStarting.value = false
    pendingScanCompletions.clear()
  }
}

async function acceptDocumentMutation(operation) {
  try {
    const next = await operation()
    document.value = normalizeEnvironmentDocument(next)
    return document.value
  } catch (cause) {
    if (String(cause).includes('environment_stale') || String(cause).includes('configuration changed')) await refreshEnvironmentData()
    throw cause
  }
}

function savePreset(draft) {
  return acceptDocumentMutation(() => SaveEnvironmentPreset({
    revision: document.value.revision,
    id: draft.id || '',
    name: draft.name,
    description: draft.description || '',
    git: draft.git || {},
    codex: draft.codex || {},
  }))
}

function clonePreset(id, name) {
  return acceptDocumentMutation(() => CloneEnvironmentPreset({ revision: document.value.revision, id, name }))
}

function deletePreset(id) {
  return acceptDocumentMutation(() => DeleteEnvironmentPreset({ revision: document.value.revision, id }))
}

function saveDefaults(draft) {
  return acceptDocumentMutation(() => SaveEnvironmentDefaults({
    revision: document.value.revision,
    presetId: draft.presetId || '',
    git: draft.git || {},
    codex: draft.codex || {},
  }))
}

async function saveRule(draft) {
  const saved = await acceptDocumentMutation(() => SaveEnvironmentRule({
    revision: document.value.revision,
    id: draft.id || '',
    name: draft.name,
    path: draft.path,
    recursive: Boolean(draft.recursive),
    autoDiscover: Boolean(draft.autoDiscover),
    presetId: draft.presetId || '',
    git: draft.git || {},
    codex: draft.codex || {},
  }))
  const savedRule = draft.id
    ? saved.config.rules.find((rule) => rule.id === draft.id)
    : saved.config.rules.at(-1)
  if (savedRule?.autoDiscover) await startScan([savedRule.id])
  return saved
}

function deleteRule(id) {
  return acceptDocumentMutation(() => DeleteEnvironmentRule({ revision: document.value.revision, id }))
}

function pickDirectory(path) {
  return PickDirectoryFrom(path || '')
}

async function previewRepositories(paths) {
  planning.value = true
  error.value = ''
  applyResults.value = []
  try {
    plans.value = await PlanEnvironmentRepositories({ repositoryPaths: paths, ruleIds: [], session: {} }) || []
  } catch (cause) {
    error.value = String(cause)
  } finally {
    planning.value = false
  }
}

async function applyDisplayedPlans(displayedPlans) {
  applying.value = true
  try {
    applyResults.value = await ApplyEnvironmentPlans(displayedPlans) || []
    await refreshEnvironmentData()
  } catch (cause) {
    error.value = String(cause)
  } finally {
    applying.value = false
  }
}

async function doctorRepositories() {
  try {
    doctorIssues.value = await DoctorEnvironments({ repositoryPaths: [], ruleIds: [], session: {} }) || []
  } catch (cause) {
    error.value = String(cause)
  }
}

function repreviewStale() {
  const paths = plans.value.flatMap((plan) => (plan.aliases || plan.Aliases || []).map((alias) => alias.path || alias.Path)).filter(Boolean)
  previewRepositories(paths)
}

function onScanProgress(payload) {
  if (!payload || payload.scanId !== scanID.value) return
  scanProgress.value = payload.progress || {}
}

async function onScanComplete(payload) {
  if (!payload?.scanId) return
  if (scanStarting.value && !scanID.value) {
    pendingScanCompletions.set(payload.scanId, payload)
    return
  }
  if (payload.scanId !== scanID.value) return
  scanID.value = ''
  await refreshEnvironmentData()
  if (payload.error && !payload.result?.canceled) error.value = payload.error
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
    <aside class="environment-sidebar" :aria-label="t('environments.views.label')">
      <button
        v-for="view in views"
        :key="view.id"
        type="button"
        class="environment-nav-item"
        :title="t(view.labelKey)"
        :aria-label="t(view.labelKey)"
        :class="{ active: activeView === view.id }"
        @click="activeView = view.id"
      >
        <AppIcon :icon="view.icon" :size="15" />
        <span>{{ t(view.labelKey) }}</span>
      </button>
    </aside>

    <div class="environment-content">
      <div v-if="loading" class="environment-state-message">{{ t('environments.loading') }}</div>
      <div v-else-if="error" class="environment-state-message error" role="alert">{{ error }}</div>
      <EnvironmentOverview
        v-else-if="activeView === 'overview'"
        :document="document"
        :state="state"
        :plans="plans"
        :scanning="scanStarting || Boolean(scanID)"
        :progress="scanProgress"
        @scan="startScan"
        @open-repositories="activeView = 'repositories'"
      />
      <EnvironmentPresetManager
        v-else-if="activeView === 'presets'"
        :document="document"
        :state="state"
        :save-preset="savePreset"
        :clone-preset="clonePreset"
        :delete-preset="deletePreset"
        :model-suggestions="modelSuggestions"
      />
      <EnvironmentRuleManager
        v-else-if="activeView === 'rules'"
        :document="document"
        :state="state"
        :save-defaults="saveDefaults"
        :save-rule="saveRule"
        :delete-rule="deleteRule"
        :pick-directory="pickDirectory"
        :scan-rule="(id) => startScan([id])"
        :model-suggestions="modelSuggestions"
      />
      <div v-else-if="activeView === 'repositories'" class="repository-flow" :class="{ previewing: plans.length }">
        <EnvironmentRepositoryManager
          :state="state"
          :rules="document.config.rules"
          :selection="selectedRepositories"
          :planning="planning"
          :doctor-issues="doctorIssues"
          @update:selection="selectedRepositories = $event"
          @preview="previewRepositories"
          @doctor="doctorRepositories"
        />
        <EnvironmentDiffPanel
          v-if="plans.length"
          :plans="plans"
          :results="applyResults"
          :applying="applying"
          @apply="applyDisplayedPlans"
          @close="plans = []; applyResults = []"
          @repreview="repreviewStale"
        />
      </div>
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
.repository-flow { width: 100%; min-width: 0; min-height: 0; display: grid; grid-template-columns: minmax(0, 1fr); }
.repository-flow.previewing { grid-template-columns: minmax(420px, 1fr) minmax(400px, .9fr); }
@media (max-width: 1100px) { .repository-flow.previewing { grid-template-columns: minmax(0, 1fr); grid-template-rows: minmax(220px, .8fr) minmax(320px, 1.2fr); overflow: auto; } }
@media (max-width: 760px) { .environment-workspace { grid-template-columns: 52px minmax(0, 1fr); } .environment-nav-item { grid-template-columns: 20px; justify-content: center; padding: 0; } .environment-nav-item span { display: none; } }
</style>
