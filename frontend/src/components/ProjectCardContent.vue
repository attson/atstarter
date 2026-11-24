<script setup>
import { computed } from 'vue'
import { Folder, GitBranch, Terminal } from 'lucide-vue-next'
import AppIcon from './ui/AppIcon.vue'
import { displayPath } from '../projectTree.js'
import { typeLabel } from '../typeLabel.js'

const props = defineProps({
  project: { type: Object, required: true },
  status: { type: Object, default: () => ({}) },
  branchEntry: { type: Object, default: null },
  commandName: { type: String, default: '' },
  count: { type: Number, default: 0 },
  missing: { type: Boolean, default: false },
})

const statusClass = computed(() => {
  const state = props.status?.State
  if (state === 'running') return 'running'
  if (state === 'error' || state === 'exited') return 'bad'
  return 'stopped'
})

const branchLabel = computed(() => {
  if (!props.branchEntry || props.branchEntry.state === 'loading') return '读取分支...'
  if (props.branchEntry.state === 'error') return '分支读取失败'
  return props.branchEntry.branch || '无分支信息'
})

const branchTitle = computed(() => props.branchEntry?.branch || branchLabel.value)
const shortPath = computed(() => displayPath(props.project.path))
</script>

<template>
  <span class="project-card-content" data-test="project-card-content">
    <span class="card-title-line">
      <span :class="['status-dot', statusClass]" />
      <span class="project-name" :title="project.name">{{ project.name }}</span>
      <span v-if="missing" class="meta-pill missing-pill">missing</span>
      <span v-else class="meta-pill">{{ typeLabel(project.detectedType) }}</span>
      <span v-if="count" class="meta-pill">{{ count }}</span>
    </span>
    <span
      class="card-meta branch-meta"
      :class="{ loading: !branchEntry || branchEntry.state === 'loading', error: branchEntry?.state === 'error' }"
      :title="branchTitle"
      data-test="project-card-branch"
    >
      <AppIcon :icon="GitBranch" :size="11" />
      <span>{{ branchLabel }}</span>
    </span>
    <span v-if="commandName" class="card-meta" :title="commandName">
      <AppIcon :icon="Terminal" :size="11" />
      <span>{{ commandName }}</span>
    </span>
    <span class="card-meta" :title="project.path" data-test="project-card-path">
      <AppIcon :icon="Folder" :size="11" />
      <span>{{ shortPath }}</span>
    </span>
  </span>
</template>

<style scoped>
.project-card-content {
  flex: 1 1 auto;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.card-title-line,
.card-meta {
  min-width: 0;
  display: flex;
  align-items: center;
}

.card-title-line { gap: var(--space-3); }
.card-meta {
  gap: var(--space-3);
  color: var(--text-muted);
  font-family: var(--font-mono);
  font-size: var(--fs-xs);
  line-height: 1.3;
}
.card-meta > :deep(.app-icon) { flex: 0 0 auto; }
.card-meta > span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.branch-meta { color: var(--text-secondary); }
.branch-meta.loading { color: var(--text-subtle); }
.branch-meta.error { color: var(--danger-fg); }

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex: 0 0 auto;
  background: var(--text-subtle);
}
.status-dot.running {
  background: var(--accent-strong);
  box-shadow: 0 0 0 2.5px var(--success-soft), 0 0 6px var(--success-glow-a);
}
.status-dot.bad {
  background: var(--danger);
  box-shadow: 0 0 0 2.5px var(--danger-soft), 0 0 6px rgba(239, 68, 68, .4);
}

.project-name {
  flex: 1 1 auto;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--text);
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
}

.meta-pill {
  flex: 0 0 auto;
  max-width: 72px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-full);
  color: var(--text-muted);
  padding: 1px 6px;
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
}
.missing-pill {
  color: var(--danger-fg);
  border-color: var(--danger-line);
  background: var(--danger-soft);
}

</style>
