import { ref, computed, watchEffect } from 'vue'

const STORAGE_KEY = 'atstarter.theme'
const VALID = new Set(['system', 'dark', 'light'])
const CYCLE = { system: 'dark', dark: 'light', light: 'system' }

export function resolveTheme(theme, prefersDark) {
  if (theme === 'dark' || theme === 'light') return theme
  if (theme === 'system') return prefersDark ? 'dark' : 'light'
  return prefersDark ? 'dark' : 'light'
}

export function nextTheme(theme) {
  return CYCLE[theme] || 'dark'
}

function loadTheme() {
  try {
    const value = localStorage.getItem(STORAGE_KEY)
    return VALID.has(value) ? value : 'system'
  } catch {
    return 'system'
  }
}

function saveTheme(value) {
  try { localStorage.setItem(STORAGE_KEY, value) } catch {}
}

const theme = ref(loadTheme())
const media = typeof window !== 'undefined' && window.matchMedia
  ? window.matchMedia('(prefers-color-scheme: dark)')
  : null
const prefersDark = ref(media ? media.matches : true)

if (media) {
  const listener = (event) => { prefersDark.value = event.matches }
  if (media.addEventListener) media.addEventListener('change', listener)
  else media.addListener(listener)
}

const resolvedTheme = computed(() => resolveTheme(theme.value, prefersDark.value))

function applyTheme(value) {
  if (typeof document === 'undefined') return
  document.documentElement.setAttribute('data-theme', value)
}

function cycleTheme() {
  const value = nextTheme(theme.value)
  theme.value = value
  saveTheme(value)
}

let initialized = false
function init() {
  if (initialized) return
  initialized = true
  watchEffect(() => applyTheme(resolvedTheme.value))
}

export function useTheme() {
  return { theme, resolvedTheme, cycleTheme, init }
}
