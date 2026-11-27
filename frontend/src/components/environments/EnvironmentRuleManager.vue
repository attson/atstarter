<script setup>
import { useI18n } from 'vue-i18n'
import { computed, ref, watch } from 'vue'
import { FolderOpen, Plus, Radar, Save, Trash2 } from 'lucide-vue-next'
import AppButton from '../ui/AppButton.vue'
import AppIcon from '../ui/AppIcon.vue'
import CodexEnvironmentFieldEditor from './CodexEnvironmentFieldEditor.vue'
import EnvironmentFieldEditor from './EnvironmentFieldEditor.vue'
import { isEnvironmentStaleError, mergeCodexFields, validatePresetDraft, validateRuleDraft } from '../../projectEnvironmentUi.js'

const { t } = useI18n()

const props = defineProps({
  document: { type: Object, required: true },
  state: { type: Object, required: true },
  saveDefaults: { type: Function, required: true },
  saveRule: { type: Function, required: true },
  deleteRule: { type: Function, required: true },
  pickDirectory: { type: Function, required: true },
  scanRule: { type: Function, required: true },
  modelSuggestions: { type: Array, default: () => [] },
})

const selectedID = ref('defaults')
const draft = ref(null)
const saving = ref(false)
const conflict = ref(false)
const message = ref('')
const saveSucceeded = ref(false)
const confirmDelete = ref(false)
const fieldTab = ref('git')
const rules = computed(() => props.document.config.rules || [])
const presets = computed(() => props.document.config.presets || [])
const isDefaults = computed(() => selectedID.value === 'defaults')
const draftName = computed({
  get: () => isDefaults.value ? t('environments.globalDefaults') : draft.value?.name || '',
  set: (name) => { if (draft.value && !isDefaults.value) draft.value.name = name },
})
const presetCodex = (id) => presets.value.find((preset) => preset.id === id)?.codex || {}
const effectiveDefaultsCodex = computed(() => mergeCodexFields(
  presetCodex(props.document.config.defaults.presetId),
  props.document.config.defaults.codex,
))
const effectiveCodex = computed(() => {
  if (!draft.value) return {}
  if (isDefaults.value) return mergeCodexFields(presetCodex(draft.value.presetId), draft.value.codex)
  return mergeCodexFields(effectiveDefaultsCodex.value, presetCodex(draft.value.presetId), draft.value.codex)
})
const validation = computed(() => isDefaults.value
  ? validatePresetDraft({ name: 'Global Defaults', git: draft.value?.git || {}, codex: draft.value?.codex || {} })
  : validateRuleDraft(draft.value || {}))
const matchedCount = computed(() => (props.state.repositories || props.state.Repositories || []).filter((repository) => {
  const ids = repository.matchingRuleIds || repository.MatchingRuleIDs || []
  return ids.includes(selectedID.value)
}).length)

function selectDefaults() {
  selectedID.value = 'defaults'
  draft.value = {
    id: 'defaults', name: 'Global Defaults', presetId: props.document.config.defaults.presetId,
    git: JSON.parse(JSON.stringify(props.document.config.defaults.git || {})),
    codex: JSON.parse(JSON.stringify(props.document.config.defaults.codex || {})),
  }
  conflict.value = false
  saveSucceeded.value = false
  message.value = ''
}

function selectRule(rule) {
  selectedID.value = rule.id
  draft.value = JSON.parse(JSON.stringify(rule))
  conflict.value = false
  saveSucceeded.value = false
  message.value = ''
  confirmDelete.value = false
}

function createRule() {
  selectedID.value = ''
  draft.value = { id: '', name: '', path: '', recursive: false, autoDiscover: false, presetId: '', git: {}, codex: {} }
}

async function chooseDirectory() {
  const path = await props.pickDirectory(draft.value?.path || '')
  if (path) draft.value.path = path
}

async function save() {
  if (!validation.value.valid || saving.value) return
  saving.value = true
  conflict.value = false
  saveSucceeded.value = false
  message.value = ''
  try {
    const saved = isDefaults.value ? await props.saveDefaults(draft.value) : await props.saveRule(draft.value)
    if (isDefaults.value) {
      draft.value = {
        id: 'defaults', name: 'Global Defaults',
        presetId: saved.config.defaults.presetId,
        git: JSON.parse(JSON.stringify(saved.config.defaults.git || {})),
        codex: JSON.parse(JSON.stringify(saved.config.defaults.codex || {})),
      }
    }
    else {
      const current = saved.config.rules.find((rule) => rule.id === draft.value.id) || saved.config.rules.at(-1)
      if (current) selectRule(current)
    }
    saveSucceeded.value = true
  } catch (error) {
    conflict.value = isEnvironmentStaleError(error)
    saveSucceeded.value = false
    message.value = conflict.value ? '' : String(error)
  } finally {
    saving.value = false
  }
}

async function remove() {
  if (!confirmDelete.value) {
    confirmDelete.value = true
    return
  }
  try {
    await props.deleteRule(draft.value.id)
    selectDefaults()
  } catch (error) {
    conflict.value = isEnvironmentStaleError(error)
    saveSucceeded.value = false
    message.value = String(error)
  }
}

watch(() => props.document.revision, () => {
  if (!draft.value) selectDefaults()
})
selectDefaults()
</script>

<template>
  <section class="rule-layout">
    <aside class="rule-list">
      <button type="button" class="rule-row protected" :class="{ active: selectedID === 'defaults' }" @click="selectDefaults">
        <span>{{ t('environments.globalDefaults') }}</span><small>{{ t('environments.rules.protected') }}</small>
      </button>
      <div class="rule-list-heading">
        <span>{{ t('environments.directoryRules') }}</span>
        <AppButton variant="secondary" size="sm" icon-only :title="t('environments.rules.new')" :aria-label="t('environments.rules.new')" @click="createRule">
          <template #icon><AppIcon :icon="Plus" :size="14" /></template>
        </AppButton>
      </div>
      <button v-for="rule in rules" :key="rule.id" type="button" class="rule-row" :class="{ active: selectedID === rule.id }" @click="selectRule(rule)">
        <span>{{ rule.name }}</span>
        <code :title="rule.path">{{ rule.path }}</code>
      </button>
    </aside>

    <div v-if="draft" class="rule-editor">
      <div class="rule-heading">
        <input v-model="draftName" class="title-input" :disabled="isDefaults" :placeholder="t('environments.rules.name')">
        <div class="rule-actions">
          <AppButton v-if="draft.id && !isDefaults" variant="secondary" size="sm" @click="scanRule(draft.id)">
            <template #icon><AppIcon :icon="Radar" :size="14" /></template>
            {{ t('environments.rules.scan') }}
          </AppButton>
          <AppButton v-if="draft.id && !isDefaults" variant="secondary" size="sm" @click="remove">
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
      <div v-else-if="message || saveSucceeded" class="rule-message">{{ saveSucceeded ? t('environments.saved') : message }}</div>

      <template v-if="!isDefaults">
        <div class="path-row">
          <input v-model="draft.path" class="path-input" :title="t('environments.rules.directory')" :placeholder="t('environments.rules.directory')">
          <AppButton variant="secondary" size="sm" icon-only :title="t('environments.rules.chooseDirectory')" :aria-label="t('environments.rules.chooseDirectory')" @click="chooseDirectory">
            <template #icon><AppIcon :icon="FolderOpen" :size="14" /></template>
          </AppButton>
        </div>
        <div class="rule-options">
          <div class="segmented" :aria-label="t('environments.rules.scope')">
            <button type="button" :class="{ active: !draft.recursive }" @click="draft.recursive = false">{{ t('environments.rules.exact') }}</button>
            <button type="button" :class="{ active: draft.recursive }" @click="draft.recursive = true">{{ t('environments.rules.recursive') }}</button>
          </div>
          <label><input v-model="draft.autoDiscover" type="checkbox"> {{ t('environments.rules.autoDiscover') }}</label>
          <span>{{ t('environments.rules.matched', { count: matchedCount }) }}</span>
        </div>
      </template>

      <label class="preset-row">
        <span>{{ t('environments.rules.preset') }}</span>
        <select v-model="draft.presetId">
          <option value="">{{ t('environments.rules.noPreset') }}</option>
          <option v-for="preset in presets" :key="preset.id" :value="preset.id">{{ preset.name }}</option>
        </select>
      </label>
      <div class="field-tabs" role="tablist" :aria-label="t('environments.fields.label')">
        <button type="button" role="tab" :aria-selected="fieldTab === 'git'" :class="{ active: fieldTab === 'git' }" @click="fieldTab = 'git'">{{ t('environments.fields.git') }}</button>
        <button type="button" role="tab" :aria-selected="fieldTab === 'codex'" :class="{ active: fieldTab === 'codex' }" @click="fieldTab = 'codex'">{{ t('environments.fields.codex') }}</button>
      </div>
      <EnvironmentFieldEditor v-if="fieldTab === 'git'" v-model="draft.git" />
      <CodexEnvironmentFieldEditor
        v-else
        v-model="draft.codex"
        :effective-fields="effectiveCodex"
        :model-suggestions="modelSuggestions"
      />
    </div>
  </section>
</template>

<style scoped>
.rule-layout { width: 100%; min-width: 0; display: grid; grid-template-columns: 260px minmax(0, 1fr); }
.rule-list { padding: var(--space-6); border-right: 1px solid var(--border); overflow: auto; }
.rule-list-heading { display: flex; align-items: center; justify-content: space-between; margin: var(--space-7) var(--space-3) var(--space-3); color: var(--text-muted); font-size: var(--fs-xs); text-transform: uppercase; }
.rule-row { width: 100%; min-width: 0; display: grid; gap: 2px; padding: var(--space-5); border: 1px solid transparent; border-radius: var(--radius-sm); background: transparent; color: var(--text-secondary); font: inherit; text-align: left; cursor: pointer; }
.rule-row.active { background: var(--elevated); border-color: var(--border); color: var(--text); }
.rule-row span, .rule-row code { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.rule-row code { color: var(--text-muted); font-size: var(--fs-xs); }
.rule-row.protected { grid-template-columns: 1fr auto; align-items: center; }
.rule-row small { color: var(--text-muted); }
.rule-editor { min-width: 0; padding: var(--space-8) var(--space-9); overflow: auto; }
.rule-heading { display: flex; align-items: center; gap: var(--space-7); }
.title-input { min-width: 0; flex: 1; border: 0; border-bottom: 1px solid var(--border); outline: 0; background: transparent; color: var(--text); font: inherit; font-size: var(--fs-lg); font-weight: var(--fw-semibold); letter-spacing: 0; padding: var(--space-3) 0; }
.title-input:disabled { opacity: 1; }
.rule-actions { display: flex; gap: var(--space-4); }
.path-row { display: grid; grid-template-columns: minmax(0, 1fr) 32px; gap: var(--space-4); margin-top: var(--space-7); }
.path-input, select { height: 32px; box-sizing: border-box; border: 1px solid var(--border); border-radius: var(--radius-sm); background: var(--elevated); color: var(--text); padding: 0 var(--space-5); font: inherit; }
.path-input { min-width: 0; font-family: var(--font-mono); }
.rule-options { display: flex; align-items: center; gap: var(--space-7); padding: var(--space-6) 0; color: var(--text-secondary); }
.segmented { display: flex; border: 1px solid var(--border); border-radius: var(--radius-sm); overflow: hidden; }
.segmented button { height: 28px; border: 0; border-right: 1px solid var(--border); background: transparent; color: var(--text-muted); cursor: pointer; }
.segmented button:last-child { border-right: 0; }
.segmented button.active { background: var(--elevated); color: var(--text); }
.preset-row { display: grid; grid-template-columns: 130px minmax(180px, 320px); align-items: center; gap: var(--space-6); padding: var(--space-6) 0; border-top: 1px solid var(--border); }
.field-tabs { width: fit-content; display: flex; margin: var(--space-5) 0; border: 1px solid var(--border); border-radius: var(--radius-sm); overflow: hidden; }
.field-tabs button { min-width: 72px; height: 28px; border: 0; border-right: 1px solid var(--border); background: transparent; color: var(--text-muted); cursor: pointer; }
.field-tabs button:last-child { border-right: 0; }
.field-tabs button.active { background: var(--elevated); color: var(--text); }
.conflict-banner, .rule-message { margin-top: var(--space-5); padding: var(--space-4) var(--space-5); border-left: 2px solid var(--warning); color: var(--text-secondary); }
@media (max-width: 820px) { .rule-layout { grid-template-columns: 200px minmax(0, 1fr); } .rule-heading { align-items: flex-start; flex-direction: column; } }
</style>
