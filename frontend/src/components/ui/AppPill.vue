<script setup>
defineProps({
  variant: { type: String, default: 'neutral' },
  dot: { type: Boolean, default: false },
})
</script>

<template>
  <span :class="['app-pill', `variant-${variant}`]">
    <span v-if="dot" class="pill-dot" />
    <slot />
  </span>
</template>

<style scoped>
.app-pill {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  padding: 2px 9px;
  border: 1px solid transparent;
  border-radius: var(--radius-full);
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
  line-height: 1.4;
  white-space: nowrap;
}

.pill-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
}

.app-pill.variant-running {
  color: var(--success);
  background: var(--success-soft);
  border-color: var(--success-line);
}

.app-pill.variant-exited {
  color: var(--warning);
  background: var(--warning-soft);
  border-color: var(--warning-line);
}

.app-pill.variant-error {
  color: var(--danger-fg);
  background: var(--danger-soft);
  border-color: var(--danger-line);
}

.app-pill.variant-stopped {
  color: var(--text-muted);
  background: var(--elevated);
  border-color: var(--border-strong);
}

.app-pill.variant-neutral {
  color: var(--text-muted);
  background: transparent;
  border-color: var(--border-strong);
}
</style>
