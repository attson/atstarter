import { createApp } from 'vue'
import App from './App.vue'
import './styles/tokens.css'
import './styles/theme.dark.css'
import './styles/theme.light.css'
import './style.css'
import { useTheme } from './composables/useTheme.js'
import { i18n, startSystemLanguageSync } from './i18n/index.js'

useTheme().init()
startSystemLanguageSync()
createApp(App).use(i18n).mount('#app')
