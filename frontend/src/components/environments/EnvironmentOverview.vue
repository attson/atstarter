<script setup>
import { computed } from 'vue'
import { AlertTriangle, FileDiff, GitBranch, RefreshCw, Route, ScanSearch } from 'lucide-vue-next'
import AppButton from '../ui/AppButton.vue'
import AppIcon from '../ui/AppIcon.vue'
import { environmentSummary } from '../../projectEnvironmentUi.js'

const props = defineProps({
  document: { type: Object, required: true },
  state: { type: Object, required: true },
  plans: { type: Array, default: () => [] },
  scanning: { type: Boolean, default: false },
  progress: { type: Object, default: () => ({}) },
})
defineEmits(['scan', 'open-repositories'])

const summary = computed(() => environmentSummary(props.document, props.state, props.plans))
const metrics = computed(() => [
  { label: 'Directory rules', value: summary.value.rules, icon: Route },
  { label: 'Repositories', value: summary.value.repositories, icon: GitBranch },
  { label: 'Pending', value: summary.value.pending, icon: RefreshCw },
  { label: 'Native changes', value: summary.value.changes, icon: FileDiff },
  { label: 'Blockers', value: summary.value.blockers, icon: AlertTriangle },
])
</script>

<template>
  <section class="environment-overview">
    <div class="environment-view-heading">
      <div>
        <h1>Environment overview</h1>
      </div>
      <AppButton variant="primary" size="sm" :disabled="scanning" @click="$emit('scan')">
        <template #icon><AppIcon :icon="scanning ? RefreshCw : ScanSearch" :size="14" :class="{ spinning: scanning }" /></template>
        {{ scanning ? 'Scanning' : 'Scan repositories' }}
      </AppButton>
    </div>

    <div class="environment-metrics">
      <div v-for="metric in metrics" :key="metric.label" class="environment-metric">
        <AppIcon :icon="metric.icon" :size="16" />
        <span>{{ metric.label }}</span>
        <strong>{{ metric.value }}</strong>
      </div>
    </div>

    <div v-if="scanning" class="scan-progress" aria-live="polite">
      <span class="scan-progress-path" :title="progress.path || ''">{{ progress.path || 'Preparing scan' }}</span>
      <span>{{ progress.repositories || 0 }} repositories</span>
      <span>{{ progress.warnings || 0 }} warnings</span>
    </div>

    <button class="environment-pending-row" type="button" @click="$emit('open-repositories')">
      <span>Repositories awaiting review</span>
      <strong>{{ summary.pending }}</strong>
    </button>
  </section>
</template>

<style scoped>
.environment-overview { width: 100%; box-sizing: border-box; padding: var(--space-9); overflow: auto; }
.environment-view-heading { display: flex; align-items: center; justify-content: space-between; gap: var(--space-8); padding-bottom: var(--space-8); border-bottom: 1px solid var(--border); }
h1 { margin: 0; font-size: var(--fs-lg); font-weight: var(--fw-semibold); letter-spacing: 0; }
.environment-metrics { display: grid; grid-template-columns: repeat(5, minmax(120px, 1fr)); border-bottom: 1px solid var(--border); }
.environment-metric { min-width: 0; display: grid; grid-template-columns: 20px 1fr auto; align-items: center; gap: var(--space-4); padding: var(--space-8) var(--space-7); border-right: 1px solid var(--border); color: var(--text-secondary); }
.environment-metric:last-child { border-right: 0; }
.environment-metric strong { color: var(--text); font-size: var(--fs-md); }
.scan-progress { display: grid; grid-template-columns: minmax(0, 1fr) auto auto; gap: var(--space-7); align-items: center; padding: var(--space-6) var(--space-7); border-bottom: 1px solid var(--border); color: var(--text-muted); font-size: var(--fs-sm); }
.scan-progress-path { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-family: var(--font-mono); }
.environment-pending-row { width: 100%; display: flex; align-items: center; justify-content: space-between; margin-top: var(--space-8); padding: var(--space-7) 0; border: 0; border-bottom: 1px solid var(--border); background: transparent; color: var(--text); font: inherit; text-align: left; cursor: pointer; }
.environment-pending-row:hover { color: var(--accent); }
.spinning { animation: environment-spin 1s linear infinite; }
@keyframes environment-spin { to { transform: rotate(360deg); } }
@media (max-width: 900px) { .environment-metrics { grid-template-columns: repeat(2, minmax(130px, 1fr)); } .environment-metric:nth-child(2) { border-right: 0; } }
</style>
