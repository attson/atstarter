<script setup>
import { computed } from 'vue'
import { commandLineForDetection, detectionLabel, detectionOptionsFor } from '../projectDetection.js'

const props = defineProps({ project: Object })
const emit = defineEmits(['switch'])

const options = computed(() => detectionOptionsFor(props.project))

function choose(option) {
  if (!props.project || option.type === props.project.detectedType) return
  emit('switch', option)
}
</script>

<template>
  <div v-if="options.length > 1" class="detection-switch" @click.stop>
    <button
      v-for="option in options"
      :key="option.type"
      type="button"
      :class="{ active: project && option.type === project.detectedType }"
      :title="commandLineForDetection(option) || detectionLabel(option)"
      @click.stop="choose(option)"
    >
      {{ detectionLabel(option) }}
    </button>
  </div>
</template>

<style scoped>
.detection-switch {
  display: inline-flex;
  align-items: center;
  flex: 0 0 auto;
  min-width: 0;
  padding: 2px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--bg);
}

.detection-switch button {
  min-width: 54px;
  height: 24px;
  padding: 0 var(--space-3);
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text-muted);
  font: inherit;
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
  cursor: pointer;
}

.detection-switch button:hover {
  color: var(--text);
  background: var(--elevated);
}

.detection-switch button.active {
  color: var(--primary-fg);
  background: var(--primary);
}
</style>
