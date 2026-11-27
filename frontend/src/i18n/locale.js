export const LANGUAGE_PREFERENCES = Object.freeze(['system', 'zh-CN', 'en'])
export const LANGUAGE_CACHE_KEY = 'rootyard.settings.language'

export function normalizeLanguagePreference(value) {
  return LANGUAGE_PREFERENCES.includes(value) ? value : 'system'
}

export function resolveSystemLocale(language = '') {
  return String(language).toLowerCase().startsWith('zh') ? 'zh-CN' : 'en'
}

export function resolveEffectiveLocale(preference, language = '') {
  const normalized = normalizeLanguagePreference(preference)
  return normalized === 'system' ? resolveSystemLocale(language) : normalized
}

export function readCachedLanguagePreference(storage) {
  try {
    return normalizeLanguagePreference((storage === undefined ? globalThis.localStorage : storage)?.getItem(LANGUAGE_CACHE_KEY))
  } catch {
    return 'system'
  }
}

export function writeCachedLanguagePreference(value, storage) {
  try {
    (storage === undefined ? globalThis.localStorage : storage)?.setItem(LANGUAGE_CACHE_KEY, normalizeLanguagePreference(value))
  } catch {
    // Language changes remain usable when browser storage is blocked or full.
  }
}
