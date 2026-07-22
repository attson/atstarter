<script setup>
import AppButton from './ui/AppButton.vue'
defineProps({ show: Boolean, title: String, message: String, confirmText: { type: String, default: '确认' }, danger: Boolean })
const emit = defineEmits(['close', 'confirm'])
</script>

<template>
  <Transition name="dlg-fade">
    <div v-if="show" class="mask" @click.self="emit('close')">
      <div class="dialog">
        <h3>{{ title }}</h3>
        <p>{{ message }}</p>
        <div class="actions">
          <AppButton variant="secondary" size="sm" @click="emit('close')">取消</AppButton>
          <AppButton :variant="danger ? 'danger' : 'primary'" size="sm" @click="emit('confirm')">{{ confirmText }}</AppButton>
        </div>
      </div>
    </div>
  </Transition>
</template>

<style scoped>
.mask {
  position: fixed; inset: 0; z-index: var(--z-modal);
  display: flex; align-items: center; justify-content: center;
  background: rgba(0, 0, 0, .45);
}
.dialog {
  width: min(420px, 92vw);
  padding: var(--space-8);
  border: 1px solid var(--border-strong);
  border-radius: var(--radius-lg);
  background: var(--surface);
  box-shadow: var(--shadow-lg);
}
.dialog h3 { margin: 0 0 var(--space-4); font-size: var(--fs-md); font-weight: var(--fw-semibold); color: var(--text); }
.dialog p { margin: 0 0 var(--space-8); color: var(--text-secondary); font-size: var(--fs-base); line-height: 1.6; }
.actions { display: flex; justify-content: flex-end; gap: var(--space-4); }
</style>
