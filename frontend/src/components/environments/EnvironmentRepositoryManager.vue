<script setup>
import { computed, ref } from 'vue'
import { CheckSquare, Search, ShieldCheck, Square, Workflow } from 'lucide-vue-next'
import AppButton from '../ui/AppButton.vue'
import AppIcon from '../ui/AppIcon.vue'
import {
  filterEnvironmentRepositories,
  repositoryStatus,
  selectAllRepositories,
  toggleRepositorySelection,
} from '../../projectEnvironmentUi.js'

const props = defineProps({
  state: { type: Object, required: true },
  rules: { type: Array, default: () => [] },
  selection: { type: Object, required: true },
  planning: { type: Boolean, default: false },
  doctorIssues: { type: Array, default: () => [] },
})
const emit = defineEmits(['update:selection', 'preview', 'doctor'])

const search = ref('')
const status = ref('all')
const ruleID = ref('')
const repositories = computed(() => props.state.repositories || props.state.Repositories || [])
const visible = computed(() => filterEnvironmentRepositories(repositories.value, { search: search.value, status: status.value, ruleId: ruleID.value }))
const statuses = ['all', 'pending', 'consistent', 'drifted', 'blocked', 'failed']

function valueFor(value, lower, upper, fallback) {
  return value?.[lower] ?? value?.[upper] ?? fallback
}

function repositoryID(repository) {
  return valueFor(repository, 'targetId', 'TargetID', '') || valueFor(repository, 'id', 'ID', '')
}

function aliases(repository) {
  return valueFor(repository, 'aliases', 'Aliases', []) || []
}

function blockers(repository) {
  return valueFor(repository, 'blockers', 'Blockers', []) || []
}

function toggle(repository) {
  emit('update:selection', toggleRepositorySelection(props.selection, repositoryID(repository), repositories.value))
}

function toggleAll() {
  const selectable = selectAllRepositories(visible.value)
  const allSelected = selectable.size > 0 && [...selectable].every((id) => props.selection.has(id))
  emit('update:selection', allSelected ? new Set() : selectable)
}

function preview() {
  const paths = []
  for (const repository of repositories.value) {
    if (!props.selection.has(repositoryID(repository))) continue
    for (const alias of aliases(repository)) {
      const path = valueFor(alias, 'path', 'Path', '')
      if (path && !paths.includes(path)) paths.push(path)
    }
  }
  emit('preview', paths)
}
</script>

<template>
  <section class="repository-manager">
    <div class="repository-toolbar">
      <label class="repository-search">
        <AppIcon :icon="Search" :size="14" />
        <input v-model="search" aria-label="Search repositories" placeholder="Search repositories">
      </label>
      <select v-model="ruleID" aria-label="Filter by rule">
        <option value="">All rules</option>
        <option v-for="rule in rules" :key="rule.id" :value="rule.id">{{ rule.name }}</option>
      </select>
      <select v-model="status" aria-label="Filter by status">
        <option v-for="item in statuses" :key="item" :value="item">{{ item }}</option>
      </select>
      <AppButton variant="secondary" size="sm" icon-only title="Select all eligible repositories" aria-label="Select all eligible repositories" @click="toggleAll">
        <template #icon><AppIcon :icon="selection.size ? CheckSquare : Square" :size="14" /></template>
      </AppButton>
      <AppButton variant="secondary" size="sm" @click="$emit('doctor')">
        <template #icon><AppIcon :icon="ShieldCheck" :size="14" /></template>
        Doctor
      </AppButton>
      <AppButton variant="primary" size="sm" :disabled="selection.size === 0 || planning" @click="preview">
        <template #icon><AppIcon :icon="Workflow" :size="14" /></template>
        Preview {{ selection.size }}
      </AppButton>
    </div>

    <div v-if="doctorIssues.length" class="doctor-issues">
      <div v-for="(issue, index) in doctorIssues" :key="`${issue.code || issue.Code}-${index}`">
        <strong>{{ issue.code || issue.Code }}</strong>
        <span>{{ issue.message || issue.Message }}</span>
      </div>
    </div>

    <div class="repository-table" role="table">
      <div class="repository-header" role="row">
        <span></span><span>Repository</span><span>Rules</span><span>Status</span>
      </div>
      <div v-for="repository in visible" :key="repositoryID(repository)" class="repository-row" role="row">
        <input
          type="checkbox"
          :checked="selection.has(repositoryID(repository))"
          :disabled="repositoryStatus(repository) === 'blocked'"
          :aria-label="`Select ${repositoryID(repository)}`"
          @change="toggle(repository)"
        >
        <div class="repository-paths">
          <code v-for="alias in aliases(repository)" :key="alias.path || alias.Path" :title="alias.path || alias.Path">{{ alias.path || alias.Path }}</code>
        </div>
        <span>{{ (repository.matchingRuleIds || repository.MatchingRuleIDs || []).length }}</span>
        <span class="repository-status" :class="repositoryStatus(repository)">{{ repositoryStatus(repository) }}</span>
        <div v-if="blockers(repository).length" class="repository-blockers">
          <span v-for="(blocker, index) in blockers(repository)" :key="index">{{ blocker.message || blocker.Message }}</span>
        </div>
      </div>
      <div v-if="visible.length === 0" class="repository-empty">No repositories</div>
    </div>
  </section>
</template>

<style scoped>
.repository-manager { width: 100%; min-width: 0; overflow: auto; }
.repository-toolbar { min-height: 48px; display: grid; grid-template-columns: minmax(170px, 1fr) 150px 120px 32px auto auto; align-items: center; gap: var(--space-4); padding: var(--space-5) var(--space-7); border-bottom: 1px solid var(--border); }
.repository-search { height: 30px; min-width: 0; display: flex; align-items: center; gap: var(--space-3); padding: 0 var(--space-4); border: 1px solid var(--border); border-radius: var(--radius-sm); color: var(--text-muted); }
.repository-search input { width: 100%; min-width: 0; border: 0; outline: 0; background: transparent; color: var(--text); }
select { height: 30px; min-width: 0; border: 1px solid var(--border); border-radius: var(--radius-sm); background: var(--elevated); color: var(--text); }
.doctor-issues { padding: var(--space-5) var(--space-7); border-bottom: 1px solid var(--border); }
.doctor-issues > div { display: flex; gap: var(--space-5); color: var(--text-secondary); }
.repository-header, .repository-row { display: grid; grid-template-columns: 28px minmax(260px, 1fr) 70px 90px; align-items: center; column-gap: var(--space-5); padding: 0 var(--space-7); }
.repository-header { height: 32px; color: var(--text-muted); font-size: var(--fs-xs); border-bottom: 1px solid var(--border); }
.repository-row { min-height: 48px; border-bottom: 1px solid var(--border); }
.repository-row:hover { background: var(--elevated); }
.repository-paths { min-width: 0; display: grid; gap: 2px; }
.repository-paths code { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--text-secondary); }
.repository-status { width: fit-content; padding: 2px var(--space-4); border-radius: var(--radius-sm); color: var(--text-secondary); background: var(--elevated); font-size: var(--fs-xs); }
.repository-status.blocked, .repository-status.failed { color: var(--danger); }
.repository-status.consistent { color: var(--success); }
.repository-status.pending, .repository-status.drifted { color: var(--warning); }
.repository-blockers { grid-column: 2 / -1; display: grid; padding-bottom: var(--space-4); color: var(--danger); font-size: var(--fs-xs); }
.repository-empty { padding: var(--space-9); text-align: center; color: var(--text-muted); }
@media (max-width: 1000px) { .repository-toolbar { grid-template-columns: minmax(160px, 1fr) 120px 100px 32px; } .repository-toolbar > :nth-last-child(-n+2) { grid-row: 2; } }
</style>
