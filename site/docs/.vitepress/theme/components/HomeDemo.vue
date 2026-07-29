<script setup>
import FrontendApp from '../../../../../frontend/src/App.vue'
import '../../../../../frontend/src/styles/tokens.css'
import '../../../../../frontend/src/styles/theme.dark.css'
import '../../../../../frontend/src/styles/theme.light.css'
import '../../../../../frontend/src/style.css'
import { useTheme } from '../../../../../frontend/src/composables/useTheme.js'

useTheme().init()
</script>

<template>
  <section class="home-demo reveal" aria-label="AT Starter mock workspace">
    <FrontendApp embedded />
  </section>
</template>

<style scoped>
.home-demo {
  --home-demo-width: min(1248px, calc(100vw - 24px));

  position: relative;
  z-index: 1;
  width: var(--home-demo-width);
  height: 620px;
  margin: 20px calc((100% - var(--home-demo-width)) / 2) 0;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--bg-gradient);
  box-shadow: var(--shadow-lg);
}

.home-demo,
.home-demo :deep(*) {
  box-sizing: border-box;
}

@media (max-width: 720px) {
  .home-demo {
    --home-demo-width: calc(100vw - 24px);

    width: calc(100vw - 24px);
    height: 620px;
    margin-top: 14px;
    overflow-x: auto;
    overflow-y: hidden;
  }

  .home-demo :deep(.app-shell-embedded) {
    min-width: 1120px;
  }
}
</style>
