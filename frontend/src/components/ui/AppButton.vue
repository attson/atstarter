<script setup>
defineProps({
  variant: { type: String, default: 'secondary' },
  size: { type: String, default: 'md' },
  disabled: { type: Boolean, default: false },
  iconOnly: { type: Boolean, default: false },
  type: { type: String, default: 'button' },
})
const emit = defineEmits(['click'])
</script>

<template>
  <button
    :type="type"
    :disabled="disabled"
    :class="['app-btn', `variant-${variant}`, `size-${size}`, { 'icon-only': iconOnly }]"
    @click="emit('click', $event)"
  >
    <span v-if="$slots.icon" class="icon-slot"><slot name="icon" /></span>
    <span v-if="!iconOnly" class="label"><slot /></span>
  </button>
</template>

<style scoped>
.app-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-3);
  height: 30px;
  padding: 0 var(--space-6);
  white-space: nowrap;
  flex: 0 0 auto;
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--text);
  font: inherit;
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
  line-height: 1;
  cursor: pointer;
  transition:
    background var(--dur-fast) var(--ease-spring),
    border-color var(--dur-fast) var(--ease-spring),
    color var(--dur-fast) var(--ease-spring),
    transform var(--dur-fast) var(--ease-spring),
    filter var(--dur-fast) var(--ease-spring);
}

.app-btn:active:not(:disabled) {
  transform: translateY(0.5px);
}

.app-btn:disabled {
  cursor: not-allowed;
  opacity: .45;
}

.app-btn.size-sm { height: 28px; padding: 0 var(--space-5); }
.app-btn.icon-only { width: 30px; padding: 0; }
.app-btn.size-sm.icon-only { width: 28px; }

.app-btn .icon-slot { display: inline-flex; align-items: center; opacity: .9; }

/* ==== Primary ==== */
.app-btn.variant-primary {
  background: var(--primary-gradient);
  color: var(--primary-fg);
  border-color: var(--accent-line);
  box-shadow: var(--primary-shadow);
}
.app-btn.variant-primary:hover:not(:disabled) { filter: brightness(1.06); }
.app-btn.variant-primary .icon-slot { opacity: 1; }

/* ==== Secondary ==== */
.app-btn.variant-secondary {
  background: var(--elevated-gradient);
  color: var(--text-secondary);
  border-color: var(--border-strong);
  box-shadow: var(--surface-highlight-strong), 0 1px 2px rgba(0, 0, 0, .18);
}
.app-btn.variant-secondary:hover:not(:disabled) {
  color: var(--text);
  background: color-mix(in srgb, var(--elevated) 60%, var(--text) 6%);
}

/* ==== Success — mirrors primary (green CTA family) ==== */
.app-btn.variant-success {
  background: var(--primary-gradient);
  color: var(--primary-fg);
  border-color: var(--accent-line);
  box-shadow: var(--primary-shadow);
}
.app-btn.variant-success:hover:not(:disabled) { filter: brightness(1.06); }
.app-btn.variant-success .icon-slot { opacity: 1; }

/* ==== Danger ==== */
.app-btn.variant-danger {
  background: var(--danger-gradient);
  color: var(--danger-fg);
  border-color: var(--danger-line);
  box-shadow: var(--surface-highlight);
}
.app-btn.variant-danger:hover:not(:disabled) {
  filter: brightness(1.08);
}

.app-btn:focus-visible {
  outline: 0;
  box-shadow: 0 0 0 3px var(--focus-ring), var(--surface-highlight);
}
.app-btn.variant-primary:focus-visible,
.app-btn.variant-success:focus-visible {
  box-shadow: 0 0 0 3px var(--focus-ring), var(--primary-shadow);
}
</style>
