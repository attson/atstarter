<script setup>
import { useI18n } from 'vue-i18n'
import { computed, getCurrentInstance } from 'vue'
import { TriangleAlert } from 'lucide-vue-next'
import AppIcon from '../ui/AppIcon.vue'
import {
  codexFieldRows,
  dangerousCodexCombination,
  editFieldMode,
  editFieldValue,
} from '../../projectEnvironmentUi.js'

const { t } = useI18n()

const props = defineProps({
  modelValue: { type: Object, default: () => ({}) },
  modelSuggestions: { type: Array, default: () => [] },
  effectiveFields: { type: Object, default: null },
})
const emit = defineEmits(['update:modelValue'])
const rows = computed(() => codexFieldRows(props.modelValue))
const dangerous = computed(() => dangerousCodexCombination(props.effectiveFields || props.modelValue))
const instanceID = getCurrentInstance()?.uid || 0

function suggestions(row) {
  return row.key === 'model'
    ? props.modelSuggestions
    : row.suggestions
}

function setMode(key, mode) {
  emit('update:modelValue', editFieldMode(props.modelValue, key, mode))
}

function setValue(key, value) {
  emit('update:modelValue', editFieldValue(props.modelValue, key, value))
}
</script>

<template>
  <div class="codex-field-editor">
    <div v-if="dangerous" class="dangerous-warning" role="alert">
      <AppIcon :icon="TriangleAlert" :size="15" />
      <span>{{ t('environments.fields.dangerous') }}</span>
    </div>
    <div v-for="row in rows" :key="row.key" class="codex-field-row">
      <label :for="`codex-mode-${instanceID}-${row.key}`" class="codex-field-key" :title="t(row.descriptionKey)">
        <span>{{ t(row.labelKey) }}</span>
        <code>{{ row.key }}</code>
      </label>
      <select
        :id="`codex-mode-${instanceID}-${row.key}`"
        :aria-label="t('environments.fields.managementMode', { field: t(row.labelKey) })"
        :value="row.intent.mode"
        @change="setMode(row.key, $event.target.value)"
      >
        <option value="inherit">{{ t('environments.fields.notManaged') }}</option>
        <option value="set">{{ t('environments.fields.set') }}</option>
        <option value="clear">{{ t('environments.fields.clear') }}</option>
      </select>
      <select
        v-if="row.control === 'select'"
        :aria-label="t('environments.fields.valueLabel', { field: t(row.labelKey) })"
        :disabled="row.intent.mode !== 'set'"
        :value="row.intent.value"
        @change="setValue(row.key, $event.target.value)"
      >
        <option value="">{{ t('environments.fields.selectValue') }}</option>
        <option v-for="option in row.options" :key="option.value" :value="option.value">{{ t(option.labelKey) }}</option>
      </select>
      <div v-else class="codex-value-control">
        <input
          :aria-label="t('environments.fields.valueLabel', { field: t(row.labelKey) })"
          :disabled="row.intent.mode !== 'set'"
          :value="row.intent.value"
          :list="`codex-suggestions-${instanceID}-${row.key}`"
          @input="setValue(row.key, $event.target.value)"
        >
        <datalist :id="`codex-suggestions-${instanceID}-${row.key}`">
          <option v-for="value in suggestions(row)" :key="value" :value="value" />
        </datalist>
      </div>
    </div>
  </div>
</template>

<style scoped>
.codex-field-editor { border-top: 1px solid var(--border); }
.dangerous-warning { min-height: 38px; display: flex; align-items: center; gap: var(--space-4); padding: 0 var(--space-5); border-bottom: 1px solid var(--border); border-left: 2px solid var(--warning); color: var(--text-secondary); }
.codex-field-row { min-height: 52px; display: grid; grid-template-columns: minmax(180px, .9fr) 128px minmax(180px, 1.3fr); align-items: center; gap: var(--space-6); border-bottom: 1px solid var(--border); }
.codex-field-key { min-width: 0; display: grid; gap: 2px; color: var(--text-secondary); }
.codex-field-key span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.codex-field-key code { overflow: hidden; color: var(--text-muted); font-family: var(--font-mono); font-size: var(--fs-xs); text-overflow: ellipsis; white-space: nowrap; }
select, input { width: 100%; min-width: 0; height: 30px; box-sizing: border-box; border: 1px solid var(--border-strong); border-radius: var(--radius-sm); background: var(--input-bg, var(--elevated)); color: var(--text); font: inherit; padding: 0 var(--space-4); }
select:disabled, input:disabled { opacity: .45; }
.codex-value-control { min-width: 0; }
@media (max-width: 800px) {
  .codex-field-row { grid-template-columns: minmax(0, 1fr) 112px; padding: var(--space-4) 0; }
  .codex-field-row > :last-child { grid-column: 1 / -1; }
}
</style>
