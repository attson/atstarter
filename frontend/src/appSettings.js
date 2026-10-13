export const DEFAULT_APP_SETTINGS = Object.freeze({
  shell: '',
  autoShell: '',
  availableShells: [],
  logFontFamily: 'monospace',
  logFontSize: 11,
  logLetterSpacing: 0,
})

function clampInteger(value, min, max, fallback) {
  const n = Number(value)
  if (!Number.isFinite(n)) return fallback
  return Math.min(max, Math.max(min, Math.round(n)))
}

export function normalizeAppSettings(value = {}) {
  return {
    shell: typeof value.shell === 'string' ? value.shell.trim() : '',
    autoShell: typeof value.autoShell === 'string' ? value.autoShell.trim() : '',
    availableShells: Array.isArray(value.availableShells)
      ? [...new Set(value.availableShells.filter((item) => typeof item === 'string' && item.trim()).map((item) => item.trim()))]
      : [],
    logFontFamily: typeof value.logFontFamily === 'string' && value.logFontFamily.trim()
      ? value.logFontFamily.trim()
      : DEFAULT_APP_SETTINGS.logFontFamily,
    logFontSize: clampInteger(value.logFontSize, 9, 20, DEFAULT_APP_SETTINGS.logFontSize),
    logLetterSpacing: clampInteger(value.logLetterSpacing, 0, 4, DEFAULT_APP_SETTINGS.logLetterSpacing),
  }
}

export function terminalFontFamily(family) {
  const name = typeof family === 'string' ? family.trim() : ''
  if (!name || name.toLowerCase() === 'monospace') return 'monospace'
  const escaped = name.replaceAll('\\', '\\\\').replaceAll('"', '\\"')
  return `"${escaped}", monospace`
}
