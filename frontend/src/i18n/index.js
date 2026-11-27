import { createI18n } from 'vue-i18n'
import en from './messages/en.js'
import zhCN from './messages/zh-CN.js'
import {
  normalizeLanguagePreference,
  readCachedLanguagePreference,
  resolveEffectiveLocale,
  writeCachedLanguagePreference,
} from './locale.js'

export function createLanguageController({ locale, navigator, document, window, storage }) {
  let languagePreference = readCachedLanguagePreference(storage)
  let unsubscribe

  function applyLocale() {
    const effectiveLocale = resolveEffectiveLocale(languagePreference, navigator?.language)
    locale.value = effectiveLocale
    if (document?.documentElement) document.documentElement.lang = effectiveLocale
  }

  function setPreference(value) {
    languagePreference = normalizeLanguagePreference(value)
    writeCachedLanguagePreference(languagePreference, storage)
    applyLocale()
  }

  function refreshSystemLocale() {
    if (languagePreference === 'system') applyLocale()
  }

  function startSystemLanguageSync() {
    if (unsubscribe) return unsubscribe
    if (!window?.addEventListener) return () => {}
    window.addEventListener('languagechange', refreshSystemLocale)
    const stop = () => {
      if (unsubscribe !== stop) return
      window.removeEventListener('languagechange', refreshSystemLocale)
      unsubscribe = undefined
    }
    unsubscribe = stop
    return stop
  }

  applyLocale()
  return {
    preference: () => languagePreference,
    setPreference,
    refreshSystemLocale,
    startSystemLanguageSync,
  }
}

const initialEffectiveLocale = resolveEffectiveLocale(readCachedLanguagePreference(), globalThis.navigator?.language)

export const i18n = createI18n({
  legacy: false,
  locale: initialEffectiveLocale,
  fallbackLocale: 'en',
  messages: { en, 'zh-CN': zhCN },
  missingWarn: import.meta.env?.DEV ?? false,
  fallbackWarn: import.meta.env?.DEV ?? false,
})

const controller = createLanguageController({
  locale: i18n.global.locale,
  navigator: globalThis.navigator,
  document: globalThis.document,
  window: globalThis.window,
})

export const getLanguagePreference = controller.preference
export const setLanguagePreference = controller.setPreference
export const refreshSystemLocale = controller.refreshSystemLocale
export const startSystemLanguageSync = controller.startSystemLanguageSync
