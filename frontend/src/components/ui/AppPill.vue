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
  box-shadow: var(--surface-highlight);
}

.pill-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
}

/* ==== Running — accent + ambient glow on dot ==== */
.app-pill.variant-running {
  color: var(--accent-strong);
  background: var(--success-gradient);
  border-color: var(--success-line);
}
.app-pill.variant-running .pill-dot {
  background: var(--accent-strong);
  box-shadow: 0 0 6px var(--success-glow-a);
}

/* ==== Exited ==== */
.app-pill.variant-exited {
  color: var(--warning);
  background: var(--warning-gradient);
  border-color: var(--warning-line);
}

/* ==== Error ==== */
.app-pill.variant-error {
  color: var(--danger-fg);
  background: var(--danger-gradient);
  border-color: var(--danger-line);
}

/* ==== Stopped ==== */
.app-pill.variant-stopped {
  color: var(--text-muted);
  background: var(--elevated-gradient);
  border-color: var(--border-strong);
}

/* ==== Neutral ==== */
.app-pill.variant-neutral {
  color: var(--text-muted);
  background: transparent;
  border-color: var(--border-strong);
  box-shadow: none;
}
</style>
