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
    background var(--dur-fast) var(--ease),
    border-color var(--dur-fast) var(--ease),
    color var(--dur-fast) var(--ease);
}

.app-btn:disabled {
  cursor: not-allowed;
  opacity: .45;
}

.app-btn.size-sm { height: 26px; padding: 0 var(--space-5); }
.app-btn.icon-only { width: 30px; padding: 0; }
.app-btn.size-sm.icon-only { width: 26px; }

.app-btn .icon-slot { display: inline-flex; align-items: center; }

.app-btn.variant-primary {
  background: var(--primary);
  color: var(--primary-fg);
}
.app-btn.variant-primary:hover:not(:disabled) { filter: brightness(.94); }

.app-btn.variant-secondary {
  background: var(--elevated);
  color: var(--text-secondary);
  border-color: var(--border-strong);
}
.app-btn.variant-secondary:hover:not(:disabled) {
  background: color-mix(in srgb, var(--elevated) 82%, var(--text) 10%);
}

.app-btn.variant-success {
  background: var(--success);
  color: var(--success-fg);
}
.app-btn.variant-success:hover:not(:disabled) { filter: brightness(1.05); }

.app-btn.variant-danger {
  background: var(--danger-soft);
  color: var(--danger-fg);
  border-color: var(--danger-line);
}
.app-btn.variant-danger:hover:not(:disabled) {
  background: color-mix(in srgb, var(--danger-soft) 70%, var(--danger) 15%);
}

.app-btn:focus-visible {
  outline: 0;
  box-shadow: 0 0 0 3px var(--focus-ring);
}
</style>
