<script setup>
import { useI18n } from 'vue-i18n'
import { computed, ref, watch } from 'vue'
import { Copy, Plus, Save, Search, Trash2 } from 'lucide-vue-next'
import AppButton from '../ui/AppButton.vue'
import AppIcon from '../ui/AppIcon.vue'
import CodexEnvironmentFieldEditor from './CodexEnvironmentFieldEditor.vue'
import EnvironmentFieldEditor from './EnvironmentFieldEditor.vue'
import { isEnvironmentStaleError, presetUsage, validatePresetDraft } from '../../projectEnvironmentUi.js'

const { t } = useI18n()

const props = defineProps({
  document: { type: Object, required: true },
  state: { type: Object, required: true },
  savePreset: { type: Function, required: true },
  clonePreset: { type: Function, required: true },
  deletePreset: { type: Function, required: true },
  modelSuggestions: { type: Array, default: () => [] },
})

const search = ref('')
const selectedID = ref('')
const draft = ref(null)
const tab = ref('fields')
const fieldTab = ref('git')
const saving = ref(false)
const message = ref('')
const saveSucceeded = ref(false)
const conflict = ref(false)
const confirmDelete = ref(false)

const presets = computed(() => props.document.config.presets || [])
const filtered = computed(() => presets.value.filter((preset) => preset.name.toLowerCase().includes(search.value.toLowerCase())))
const usage = computed(() => draft.value ? presetUsage(props.document, props.state, draft.value.id) : { defaults: false, rules: [], repositories: [], total: 0 })
const validation = computed(() => validatePresetDraft(draft.value || {}))

function selectPreset(preset) {
  selectedID.value = preset.id
  draft.value = JSON.parse(JSON.stringify(preset))
  saveSucceeded.value = false
  message.value = ''
  conflict.value = false
  confirmDelete.value = false
}

function createPreset() {
  selectedID.value = ''
  draft.value = { id: '', name: '', description: '', git: {}, codex: {} }
  tab.value = 'fields'
  saveSucceeded.value = false
  message.value = ''
}

async function save() {
  if (!validation.value.valid || saving.value) return
  saving.value = true
  saveSucceeded.value = false
  message.value = ''
  conflict.value = false
  try {
    const saved = await props.savePreset(draft.value)
    const current = saved.config.presets.find((preset) => preset.id === draft.value.id) || saved.config.presets.at(-1)
    if (current) selectPreset(current)
    saveSucceeded.value = true
  } catch (error) {
    conflict.value = isEnvironmentStaleError(error)
    saveSucceeded.value = false
    message.value = conflict.value ? '' : String(error)
  } finally {
    saving.value = false
  }
}

async function clone() {
  if (!draft.value?.id) return
  try {
    const saved = await props.clonePreset(draft.value.id, t('environments.presets.cloneName', { name: draft.value.name }))
    const cloned = saved.config.presets.at(-1)
    if (cloned) selectPreset(cloned)
  } catch (error) {
    saveSucceeded.value = false
    message.value = String(error)
  }
}

async function remove() {
  if (!draft.value?.id || usage.value.total || !confirmDelete.value) {
    confirmDelete.value = true
    return
  }
  try {
    await props.deletePreset(draft.value.id)
    draft.value = null
    selectedID.value = ''
  } catch (error) {
    conflict.value = isEnvironmentStaleError(error)
    saveSucceeded.value = false
    message.value = String(error)
  }
}

watch(presets, (next) => {
  if (!draft.value && next.length) selectPreset(next[0])
}, { immediate: true })
</script>

<template>
  <section class="manager-layout">
    <aside class="manager-list">
      <div class="manager-list-tools">
        <label class="search-box">
          <AppIcon :icon="Search" :size="14" />
          <input v-model="search" :aria-label="t('environments.presets.search')" :placeholder="t('environments.presets.search')">
        </label>
        <AppButton variant="secondary" size="sm" icon-only :title="t('environments.presets.new')" :aria-label="t('environments.presets.new')" @click="createPreset">
          <template #icon><AppIcon :icon="Plus" :size="14" /></template>
        </AppButton>
      </div>
      <button v-for="preset in filtered" :key="preset.id" type="button" class="manager-list-row" :class="{ active: selectedID === preset.id }" @click="selectPreset(preset)">
        <span>{{ preset.name }}</span>
      </button>
    </aside>

    <div v-if="draft" class="manager-editor">
      <div class="manager-heading">
        <input v-model="draft.name" class="title-input" :placeholder="t('environments.presets.name')">
        <div class="manager-actions">
          <AppButton v-if="draft.id" variant="secondary" size="sm" icon-only :title="t('environments.presets.clone')" :aria-label="t('environments.presets.clone')" @click="clone">
            <template #icon><AppIcon :icon="Copy" :size="14" /></template>
          </AppButton>
          <AppButton v-if="draft.id" variant="secondary" size="sm" :disabled="usage.total > 0" @click="remove">
            <template #icon><AppIcon :icon="Trash2" :size="14" /></template>
            {{ confirmDelete ? t('environments.presets.confirmDelete') : t('environments.presets.delete') }}
          </AppButton>
          <AppButton variant="primary" size="sm" :disabled="!validation.valid || saving" @click="save">
            <template #icon><AppIcon :icon="Save" :size="14" /></template>
            {{ t('common.save') }}
          </AppButton>
        </div>
      </div>

      <div v-if="conflict" class="conflict-banner">{{ t('environments.conflict') }}</div>
      <div v-else-if="message || saveSucceeded" class="manager-message">{{ saveSucceeded ? t('environments.saved') : message }}</div>
      <textarea v-model="draft.description" rows="2" :placeholder="t('environments.presets.description')"></textarea>
      <div class="manager-tabs">
        <button type="button" :class="{ active: tab === 'fields' }" @click="tab = 'fields'">{{ t('environments.presets.fields') }}</button>
        <button type="button" :class="{ active: tab === 'usage' }" @click="tab = 'usage'">{{ t('environments.presets.usage') }} <span>{{ usage.total }}</span></button>
      </div>
      <div v-if="tab === 'fields'" class="field-section">
        <div class="field-tabs" role="tablist" :aria-label="t('environments.fields.label')">
          <button type="button" role="tab" :aria-selected="fieldTab === 'git'" :class="{ active: fieldTab === 'git' }" @click="fieldTab = 'git'">{{ t('environments.fields.git') }}</button>
          <button type="button" role="tab" :aria-selected="fieldTab === 'codex'" :class="{ active: fieldTab === 'codex' }" @click="fieldTab = 'codex'">{{ t('environments.fields.codex') }}</button>
        </div>
        <EnvironmentFieldEditor v-if="fieldTab === 'git'" v-model="draft.git" />
        <CodexEnvironmentFieldEditor v-else v-model="draft.codex" :model-suggestions="modelSuggestions" />
      </div>
      <div v-else class="usage-list">
        <div><span>{{ t('environments.globalDefaults') }}</span><strong>{{ usage.defaults ? t('environments.presets.referenced') : t('environments.presets.notReferenced') }}</strong></div>
        <div><span>{{ t('environments.directoryRules') }}</span><strong>{{ usage.rules.length }}</strong></div>
        <div><span>{{ t('environments.repositoriesLabel') }}</span><strong>{{ usage.repositories.length }}</strong></div>
      </div>
    </div>
    <div v-else class="manager-empty">{{ t('environments.presets.empty') }}</div>
  </section>
</template>

<style scoped>
.manager-layout { width: 100%; min-width: 0; display: grid; grid-template-columns: 240px minmax(0, 1fr); }
.manager-list { min-width: 0; padding: var(--space-6); border-right: 1px solid var(--border); overflow: auto; }
.manager-list-tools { display: grid; grid-template-columns: minmax(0, 1fr) 32px; gap: var(--space-4); margin-bottom: var(--space-5); }
.search-box { height: 30px; display: flex; align-items: center; gap: var(--space-3); padding: 0 var(--space-4); border: 1px solid var(--border); border-radius: var(--radius-sm); color: var(--text-muted); }
.search-box input { width: 100%; min-width: 0; border: 0; outline: 0; background: transparent; color: var(--text); }
.manager-list-row { width: 100%; height: 34px; padding: 0 var(--space-5); border: 1px solid transparent; border-radius: var(--radius-sm); background: transparent; color: var(--text-secondary); font: inherit; text-align: left; cursor: pointer; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.manager-list-row.active { background: var(--elevated); color: var(--text); border-color: var(--border); }
.manager-editor { min-width: 0; padding: var(--space-8) var(--space-9); overflow: auto; }
.manager-heading { display: flex; align-items: center; gap: var(--space-7); }
.title-input { min-width: 0; flex: 1; border: 0; border-bottom: 1px solid var(--border); outline: 0; background: transparent; color: var(--text); font: inherit; font-size: var(--fs-lg); font-weight: var(--fw-semibold); letter-spacing: 0; padding: var(--space-3) 0; }
.manager-actions { display: flex; align-items: center; gap: var(--space-4); }
textarea { width: 100%; box-sizing: border-box; margin-top: var(--space-6); padding: var(--space-5); resize: vertical; border: 1px solid var(--border); border-radius: var(--radius-sm); background: var(--elevated); color: var(--text); font: inherit; }
.manager-tabs { display: flex; gap: var(--space-2); margin-top: var(--space-7); border-bottom: 1px solid var(--border); }
.manager-tabs button { height: 32px; border: 0; border-bottom: 2px solid transparent; background: transparent; color: var(--text-muted); cursor: pointer; }
.manager-tabs button.active { color: var(--text); border-bottom-color: var(--accent); }
.manager-tabs span { margin-left: var(--space-2); color: var(--text-muted); }
.field-tabs { width: fit-content; display: flex; margin: var(--space-5) 0; border: 1px solid var(--border); border-radius: var(--radius-sm); overflow: hidden; }
.field-tabs button { min-width: 72px; height: 28px; border: 0; border-right: 1px solid var(--border); background: transparent; color: var(--text-muted); cursor: pointer; }
.field-tabs button:last-child { border-right: 0; }
.field-tabs button.active { background: var(--elevated); color: var(--text); }
.usage-list > div { display: flex; justify-content: space-between; padding: var(--space-6) 0; border-bottom: 1px solid var(--border); }
.usage-list strong { font-weight: var(--fw-medium); }
.conflict-banner, .manager-message { margin-top: var(--space-5); padding: var(--space-4) var(--space-5); border-left: 2px solid var(--warning); color: var(--text-secondary); }
.manager-empty { margin: auto; color: var(--text-muted); }
@media (max-width: 820px) { .manager-layout { grid-template-columns: 190px minmax(0, 1fr); } .manager-heading { align-items: flex-start; flex-direction: column; } }
</style>
