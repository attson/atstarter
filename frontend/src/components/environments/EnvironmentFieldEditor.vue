<script setup>
import { computed } from 'vue'
import { editFieldMode, editFieldValue, fieldRows } from '../../projectEnvironmentUi.js'

const props = defineProps({
  modelValue: { type: Object, default: () => ({}) },
})
const emit = defineEmits(['update:modelValue'])
const rows = computed(() => fieldRows(props.modelValue))

function setMode(key, mode) {
  emit('update:modelValue', editFieldMode(props.modelValue, key, mode))
}

function setValue(key, value) {
  emit('update:modelValue', editFieldValue(props.modelValue, key, value))
}
</script>

<template>
  <div class="field-editor">
    <div v-for="row in rows" :key="row.key" class="field-row">
      <label :for="`mode-${row.key}`" class="field-key">{{ row.key }}</label>
      <select :id="`mode-${row.key}`" :value="row.intent.mode" @change="setMode(row.key, $event.target.value)">
        <option value="inherit">Not managed</option>
        <option value="set">Set</option>
        <option value="clear">Clear</option>
      </select>

      <label v-if="row.intent.mode === 'set' && (row.key === 'commit.gpgSign' || row.key === 'tag.gpgSign')" class="boolean-control">
        <input type="checkbox" :checked="row.intent.value === 'true'" @change="setValue(row.key, $event.target.checked ? 'true' : 'false')">
        <span>{{ row.intent.value === 'true' ? 'Enabled' : 'Disabled' }}</span>
      </label>
      <select v-else-if="row.intent.mode === 'set' && row.key === 'gpg.format'" :value="row.intent.value" @change="setValue(row.key, $event.target.value)">
        <option value="">Select format</option>
        <option value="openpgp">OpenPGP</option>
        <option value="ssh">SSH</option>
        <option value="x509">X.509</option>
      </select>
      <input
        v-else
        class="field-value"
        :disabled="row.intent.mode !== 'set'"
        :value="row.intent.value"
        :placeholder="row.intent.mode === 'set' ? 'Value' : ''"
        @input="setValue(row.key, $event.target.value)"
      >
    </div>
  </div>
</template>

<style scoped>
.field-editor { border-top: 1px solid var(--border); }
.field-row { min-height: 44px; display: grid; grid-template-columns: minmax(150px, .8fr) 128px minmax(180px, 1.3fr); align-items: center; gap: var(--space-6); border-bottom: 1px solid var(--border); }
.field-key { min-width: 0; overflow: hidden; text-overflow: ellipsis; font-family: var(--font-mono); font-size: var(--fs-mono); color: var(--text-secondary); }
select, input { width: 100%; min-width: 0; height: 30px; box-sizing: border-box; border: 1px solid var(--border-strong); border-radius: var(--radius-sm); background: var(--input-bg, var(--elevated)); color: var(--text); font: inherit; padding: 0 var(--space-4); }
input:disabled { opacity: .45; }
.boolean-control { width: fit-content; display: flex; align-items: center; gap: var(--space-4); color: var(--text-secondary); }
.boolean-control input { width: 16px; height: 16px; }
@media (max-width: 800px) { .field-row { grid-template-columns: 1fr 112px; padding: var(--space-4) 0; } .field-row > :last-child { grid-column: 1 / -1; } }
</style>
