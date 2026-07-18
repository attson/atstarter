<script setup>
defineProps({
  icon: { type: [Object, Function], required: true },
  size: { type: Number, default: 14 },
  strokeWidth: { type: Number, default: 1.75 },
})
</script>

<template>
  <component :is="icon" :size="size" :stroke-width="strokeWidth" class="app-icon" />
</template>

<style scoped>
.app-icon {
  display: inline-block;
  flex-shrink: 0;
  vertical-align: -2px;
}
</style>
