<script setup>
import { useI18n } from 'vue-i18n'
import { computed, ref, watch } from 'vue'
import { AlertTriangle, CheckCircle2, RefreshCw, ShieldAlert } from 'lucide-vue-next'
import AppButton from '../ui/AppButton.vue'
import AppIcon from '../ui/AppIcon.vue'
import { planChangeCount, plansHaveIssueCode, sharedTargetConflict, summarizeApplyResults } from '../../projectEnvironmentUi.js'

const { t } = useI18n()

const props = defineProps({
  plans: { type: Array, required: true },
  results: { type: Array, default: () => [] },
  applying: { type: Boolean, default: false },
})
const emit = defineEmits(['apply', 'close', 'repreview'])
const confirmed = ref(false)
const dangerousConfirmed = ref(false)
const changes = computed(() => planChangeCount(props.plans))
const highRisk = computed(() => plansHaveIssueCode(props.plans, 'dangerous_codex_permissions'))
const blockers = computed(() => props.plans.flatMap((plan) => plan.blockers || plan.Blockers || []))
const summary = computed(() => summarizeApplyResults(props.results))
const groups = computed(() => {
  const grouped = new Map()
  for (const plan of props.plans) {
    const worktree = targetLabel(plan)
    if (!grouped.has(worktree)) grouped.set(worktree, { worktree, plans: [] })
    grouped.get(worktree).plans.push(plan)
  }
  return [...grouped.values()]
})

const statusKeys = {
  all: 'environments.status.all', pending: 'environments.status.pending',
  consistent: 'environments.status.consistent', drifted: 'environments.status.drifted',
  blocked: 'environments.status.blocked', failed: 'environments.status.failed',
  success: 'environments.status.success', unchanged: 'environments.status.unchanged', stale: 'environments.status.stale',
}
function statusLabel(status) {
  return Object.hasOwn(statusKeys, status) ? t(statusKeys[status]) : status
}

const operationKeys = {
  set: 'environments.operation.set', replace: 'environments.operation.replace',
  clear: 'environments.operation.clear', unchanged: 'environments.operation.unchanged',
  unmanaged: 'environments.operation.unmanaged', blocked: 'environments.operation.blocked',
}
function operationLabel(operation) {
  return Object.hasOwn(operationKeys, operation) ? t(operationKeys[operation]) : operation
}

function valueFor(value, lower, upper, fallback) {
  return value?.[lower] ?? value?.[upper] ?? fallback
}

function targetLabel(plan) {
  const target = valueFor(plan, 'target', 'Target', {})
  return valueFor(target, 'worktreePath', 'WorktreePath', '') || valueFor(plan, 'repoPath', 'RepoPath', '')
}

function currentValues(change) {
  return (valueFor(change, 'current', 'Current', []) || []).map((item) => valueFor(item, 'value', 'Value', '')).join(', ') || t('environments.diff.absent')
}

function planAdapter(plan) {
  return valueFor(plan, 'adapter', 'Adapter', 'unknown')
}

function displayChanges(plan) {
  return (valueFor(plan, 'changes', 'Changes', []) || []).filter((change) =>
    !['unchanged', 'unmanaged'].includes(valueFor(change, 'operation', 'Operation', '')))
}

function configPath(plan) {
  const target = valueFor(plan, 'target', 'Target', {})
  return target.configPath || target.ConfigPath || ''
}

watch(() => props.plans, () => {
  confirmed.value = false
  dangerousConfirmed.value = false
}, { deep: true })
</script>

<template>
  <aside class="diff-panel">
    <div class="diff-heading">
      <div><strong>{{ t('environments.diff.title') }}</strong><span>{{ t('environments.diff.changes', { count: changes }) }}</span></div>
      <button type="button" :aria-label="t('environments.diff.close')" @click="$emit('close')">×</button>
    </div>

    <div v-if="blockers.length" class="diff-blockers" :aria-label="t('environments.diff.blockers')">
      <AppIcon :icon="ShieldAlert" :size="16" />
      <div>
        <span v-for="(blocker, index) in blockers" :key="index">{{ blocker.message || blocker.Message }}</span>
      </div>
    </div>

    <div v-if="highRisk" class="diff-risk" role="alert" :aria-label="t('environments.diff.warnings')">
      <AppIcon :icon="AlertTriangle" :size="16" />
      <span>{{ t('environments.diff.risk') }}</span>
    </div>

    <div class="diff-plans">
      <section v-for="group in groups" :key="group.worktree" class="diff-target">
        <code :title="group.worktree">{{ group.worktree }}</code>
        <div v-for="plan in group.plans" :key="`${planAdapter(plan)}-${(plan.target || plan.Target).id || (plan.target || plan.Target).ID}`" class="adapter-plan">
          <div class="adapter-heading">
            <strong>{{ planAdapter(plan) }}</strong>
            <span>{{ t('environments.diff.changes', { count: planChangeCount([plan]) }) }}</span>
          </div>
          <code class="config-path" :title="configPath(plan)">{{ configPath(plan) }}</code>
          <div v-if="sharedTargetConflict(plan)" class="shared-conflict">
            <AppIcon :icon="AlertTriangle" :size="14" />
            <span v-for="path in sharedTargetConflict(plan).paths" :key="path" :title="path">{{ path }}</span>
          </div>
          <div v-for="change in displayChanges(plan)" :key="change.key || change.Key" class="diff-change">
            <code>{{ change.key || change.Key }}</code>
            <span>{{ currentValues(change) }}</span>
            <strong>{{ operationLabel(change.operation || change.Operation) }}</strong>
            <span>{{ (change.desired || change.Desired).value || (change.desired || change.Desired).Value || t('environments.diff.clear') }}</span>
            <small>{{ (change.source || change.Source).name || (change.source || change.Source).Name }}</small>
          </div>
          <pre v-if="(plan.nativeDiff || plan.NativeDiff || []).length" class="native-diff"><code>{{ (plan.nativeDiff || plan.NativeDiff).join('\n') }}</code></pre>
        </div>
      </section>
    </div>

    <div v-if="results.length" class="apply-results">
      <div><AppIcon :icon="CheckCircle2" :size="15" /><span>{{ t('environments.diff.applied', { count: summary.success }) }}</span></div>
      <div v-for="result in results" :key="(result.target || result.Target).id || (result.target || result.Target).ID" class="result-row">
        <code>{{ (result.target || result.Target).id || (result.target || result.Target).ID }}</code>
        <strong :class="result.status || result.Status">{{ statusLabel(result.status || result.Status) }}</strong>
      </div>
      <AppButton v-if="summary.stale" variant="secondary" size="sm" @click="$emit('repreview')">
        <template #icon><AppIcon :icon="RefreshCw" :size="14" /></template>
        {{ t('environments.diff.repreview') }}
      </AppButton>
    </div>

    <div class="diff-confirmation">
      <div class="confirmation-checks">
        <label>
          <input v-model="confirmed" type="checkbox">
          <span>{{ t('environments.diff.reviewed') }}</span>
        </label>
        <label v-if="highRisk" class="dangerous-confirmation">
          <input v-model="dangerousConfirmed" type="checkbox">
          <span>{{ t('environments.diff.dangerousConfirmed') }}</span>
        </label>
      </div>
      <AppButton variant="primary" size="sm" :disabled="!confirmed || (highRisk && !dangerousConfirmed) || applying || blockers.length > 0" @click="$emit('apply', plans)">
        {{ t('environments.diff.apply') }}
      </AppButton>
    </div>
  </aside>
</template>

<style scoped>
.diff-panel { min-width: 0; min-height: 0; display: flex; flex-direction: column; border-left: 1px solid var(--border); background: var(--surface); }
.diff-heading { height: 48px; display: flex; align-items: center; justify-content: space-between; padding: 0 var(--space-7); border-bottom: 1px solid var(--border); }
.diff-heading div { display: flex; gap: var(--space-4); align-items: baseline; }
.diff-heading span { color: var(--text-muted); font-size: var(--fs-xs); }
.diff-heading button { border: 0; background: transparent; color: var(--text-muted); font-size: 20px; cursor: pointer; }
.diff-blockers { display: grid; grid-template-columns: 20px 1fr; gap: var(--space-4); padding: var(--space-5) var(--space-7); border-bottom: 1px solid var(--border); color: var(--danger); }
.diff-blockers div { display: grid; }
.diff-risk { display: grid; grid-template-columns: 20px minmax(0, 1fr); gap: var(--space-4); padding: var(--space-5) var(--space-7); border-bottom: 1px solid var(--border); color: var(--warning); }
.diff-plans { flex: 1; min-height: 0; overflow: auto; }
.diff-target { padding: var(--space-6) var(--space-7); border-bottom: 1px solid var(--border); }
.diff-target > code { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--text-secondary); }
.adapter-plan { min-width: 0; margin-top: var(--space-5); }
.adapter-heading { display: flex; align-items: center; justify-content: space-between; gap: var(--space-4); text-transform: capitalize; }
.adapter-heading span { color: var(--text-muted); font-size: var(--fs-xs); }
.config-path { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--text-muted); }
.diff-change { display: grid; grid-template-columns: minmax(110px, .9fr) minmax(100px, 1fr) 70px minmax(100px, 1fr); gap: var(--space-4); padding: var(--space-4) 0; border-top: 1px solid var(--border); font-size: var(--fs-xs); }
.diff-change small { grid-column: 1 / -1; color: var(--text-muted); }
.diff-change span { overflow-wrap: anywhere; }
.shared-conflict { display: grid; grid-template-columns: 20px minmax(0, 1fr); gap: var(--space-2); margin-top: var(--space-4); color: var(--danger); }
.shared-conflict span { grid-column: 2; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.native-diff { max-width: 100%; margin: var(--space-4) 0 0; overflow: auto; color: var(--text-secondary); font-size: var(--fs-xs); }
.apply-results { padding: var(--space-5) var(--space-7); border-top: 1px solid var(--border); }
.apply-results > div:first-child { display: flex; align-items: center; gap: var(--space-3); }
.result-row { display: grid; grid-template-columns: minmax(0, 1fr) auto; padding: var(--space-3) 0; }
.result-row code { overflow: hidden; text-overflow: ellipsis; }
.result-row .failed, .result-row .stale, .result-row .blocked { color: var(--danger); }
.diff-confirmation { display: flex; align-items: center; justify-content: space-between; gap: var(--space-6); padding: var(--space-6) var(--space-7); border-top: 1px solid var(--border); }
.confirmation-checks { min-width: 0; display: grid; gap: var(--space-3); }
.diff-confirmation label { display: flex; align-items: center; gap: var(--space-4); color: var(--text-secondary); font-size: var(--fs-sm); }
.dangerous-confirmation { color: var(--warning) !important; }
@media (max-width: 1100px) { .diff-panel { border-top: 1px solid var(--border); border-left: 0; } }
</style>
