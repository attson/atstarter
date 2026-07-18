import { createApp } from 'vue'
import App from './App.vue'
import './styles/tokens.css'
import './styles/theme.dark.css'
import './styles/theme.light.css'
import './style.css'
import { useTheme } from './composables/useTheme.js'

useTheme().init()
createApp(App).mount('#app')
