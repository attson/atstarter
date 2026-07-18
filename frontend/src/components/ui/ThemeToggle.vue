<script setup>
import { computed } from 'vue'
import { Sun, Moon, Monitor } from 'lucide-vue-next'
import { useTheme } from '../../composables/useTheme.js'
import AppIcon from './AppIcon.vue'
import AppButton from './AppButton.vue'

const { theme, cycleTheme } = useTheme()
const icon = computed(() => {
  if (theme.value === 'dark') return Moon
  if (theme.value === 'light') return Sun
  return Monitor
})
const label = computed(() => `Theme: ${theme.value}`)
</script>

<template>
  <AppButton
    variant="secondary"
    size="sm"
    icon-only
    :aria-label="label"
    :title="label"
    @click="cycleTheme"
  >
    <template #icon><AppIcon :icon="icon" :size="14" /></template>
  </AppButton>
</template>
