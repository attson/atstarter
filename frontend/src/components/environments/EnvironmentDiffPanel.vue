<script setup>
import { computed, ref, watch } from 'vue'
import { AlertTriangle, CheckCircle2, RefreshCw, ShieldAlert } from 'lucide-vue-next'
import AppButton from '../ui/AppButton.vue'
import AppIcon from '../ui/AppIcon.vue'
import { planChangeCount, sharedTargetConflict, summarizeApplyResults } from '../../projectEnvironmentUi.js'

const props = defineProps({
  plans: { type: Array, required: true },
  results: { type: Array, default: () => [] },
  applying: { type: Boolean, default: false },
})
const emit = defineEmits(['apply', 'close', 'repreview'])
const confirmed = ref(false)
const changes = computed(() => planChangeCount(props.plans))
const blockers = computed(() => props.plans.flatMap((plan) => plan.blockers || plan.Blockers || []))
const summary = computed(() => summarizeApplyResults(props.results))

function valueFor(value, lower, upper, fallback) {
  return value?.[lower] ?? value?.[upper] ?? fallback
}

function targetLabel(plan) {
  const target = valueFor(plan, 'target', 'Target', {})
  return valueFor(target, 'worktreePath', 'WorktreePath', '') || valueFor(plan, 'repoPath', 'RepoPath', '')
}

function currentValues(change) {
  return (valueFor(change, 'current', 'Current', []) || []).map((item) => valueFor(item, 'value', 'Value', '')).join(', ') || '(absent)'
}

watch(() => props.plans, () => { confirmed.value = false }, { deep: true })
</script>

<template>
  <aside class="diff-panel">
    <div class="diff-heading">
      <div><strong>Native Git changes</strong><span>{{ changes }} changes</span></div>
      <button type="button" aria-label="Close preview" @click="$emit('close')">×</button>
    </div>

    <div v-if="blockers.length" class="diff-blockers">
      <AppIcon :icon="ShieldAlert" :size="16" />
      <div>
        <span v-for="(blocker, index) in blockers" :key="index">{{ blocker.message || blocker.Message }}</span>
      </div>
    </div>

    <div class="diff-plans">
      <section v-for="plan in plans" :key="(plan.target || plan.Target).id || (plan.target || plan.Target).ID" class="diff-target">
        <code :title="targetLabel(plan)">{{ targetLabel(plan) }}</code>
        <div v-if="sharedTargetConflict(plan)" class="shared-conflict">
          <AppIcon :icon="AlertTriangle" :size="14" />
          <span v-for="path in sharedTargetConflict(plan).paths" :key="path" :title="path">{{ path }}</span>
        </div>
        <div v-for="change in (plan.changes || plan.Changes || [])" :key="change.key || change.Key" class="diff-change">
          <code>{{ change.key || change.Key }}</code>
          <span>{{ currentValues(change) }}</span>
          <strong>{{ change.operation || change.Operation }}</strong>
          <span>{{ (change.desired || change.Desired).value || (change.desired || change.Desired).Value || '(clear)' }}</span>
          <small>{{ (change.source || change.Source).name || (change.source || change.Source).Name }}</small>
        </div>
      </section>
    </div>

    <div v-if="results.length" class="apply-results">
      <div><AppIcon :icon="CheckCircle2" :size="15" /><span>{{ summary.success }} applied</span></div>
      <div v-for="result in results" :key="(result.target || result.Target).id || (result.target || result.Target).ID" class="result-row">
        <code>{{ (result.target || result.Target).id || (result.target || result.Target).ID }}</code>
        <strong :class="result.status || result.Status">{{ result.status || result.Status }}</strong>
      </div>
      <AppButton v-if="summary.stale" variant="secondary" size="sm" @click="$emit('repreview')">
        <template #icon><AppIcon :icon="RefreshCw" :size="14" /></template>
        Re-preview stale targets
      </AppButton>
    </div>

    <div class="diff-confirmation">
      <label>
        <input v-model="confirmed" type="checkbox">
        <span>I reviewed these exact native Git changes</span>
      </label>
      <AppButton variant="primary" size="sm" :disabled="!confirmed || applying || blockers.length > 0" @click="$emit('apply', plans)">
        Apply changes
      </AppButton>
    </div>
  </aside>
</template>

<style scoped>
.diff-panel { min-width: 0; display: grid; grid-template-rows: auto auto minmax(0, 1fr) auto auto; border-left: 1px solid var(--border); background: var(--surface); }
.diff-heading { height: 48px; display: flex; align-items: center; justify-content: space-between; padding: 0 var(--space-7); border-bottom: 1px solid var(--border); }
.diff-heading div { display: flex; gap: var(--space-4); align-items: baseline; }
.diff-heading span { color: var(--text-muted); font-size: var(--fs-xs); }
.diff-heading button { border: 0; background: transparent; color: var(--text-muted); font-size: 20px; cursor: pointer; }
.diff-blockers { display: grid; grid-template-columns: 20px 1fr; gap: var(--space-4); padding: var(--space-5) var(--space-7); border-bottom: 1px solid var(--border); color: var(--danger); }
.diff-blockers div { display: grid; }
.diff-plans { min-height: 0; overflow: auto; }
.diff-target { padding: var(--space-6) var(--space-7); border-bottom: 1px solid var(--border); }
.diff-target > code { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--text-secondary); }
.diff-change { display: grid; grid-template-columns: minmax(110px, .9fr) minmax(100px, 1fr) 70px minmax(100px, 1fr); gap: var(--space-4); padding: var(--space-4) 0; border-top: 1px solid var(--border); font-size: var(--fs-xs); }
.diff-change small { grid-column: 1 / -1; color: var(--text-muted); }
.diff-change span { overflow-wrap: anywhere; }
.shared-conflict { display: grid; grid-template-columns: 20px minmax(0, 1fr); gap: var(--space-2); margin-top: var(--space-4); color: var(--danger); }
.shared-conflict span { grid-column: 2; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.apply-results { padding: var(--space-5) var(--space-7); border-top: 1px solid var(--border); }
.apply-results > div:first-child { display: flex; align-items: center; gap: var(--space-3); }
.result-row { display: grid; grid-template-columns: minmax(0, 1fr) auto; padding: var(--space-3) 0; }
.result-row code { overflow: hidden; text-overflow: ellipsis; }
.result-row .failed, .result-row .stale, .result-row .blocked { color: var(--danger); }
.diff-confirmation { display: flex; align-items: center; justify-content: space-between; gap: var(--space-6); padding: var(--space-6) var(--space-7); border-top: 1px solid var(--border); }
.diff-confirmation label { display: flex; align-items: center; gap: var(--space-4); color: var(--text-secondary); font-size: var(--fs-sm); }
@media (max-width: 1100px) { .diff-panel { border-top: 1px solid var(--border); border-left: 0; } }
</style>
