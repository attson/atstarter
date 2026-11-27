import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import test from 'node:test'
import {
  createLanguageController,
  i18n,
  getLanguagePreference,
  setLanguagePreference,
  refreshSystemLocale,
  startSystemLanguageSync,
} from './index.js'

function fixture(cached = null, language = 'zh-TW') {
  const values = new Map([['rootyard.settings.language', cached]])
  const storage = {
    getItem: (key) => values.get(key) ?? null,
    setItem: (key, value) => values.set(key, value),
  }
  const locale = { value: 'en' }
  const navigator = { language }
  const document = { documentElement: { lang: '' } }
  const window = new EventTarget()
  const controller = createLanguageController({ locale, navigator, document, window, storage })
  return { controller, locale, navigator, document, window, values }
}

test('controller applies cached preference and html language before mount', () => {
  const { controller, locale, document } = fixture('en', 'zh-CN')
  assert.equal(controller.preference(), 'en')
  assert.equal(locale.value, 'en')
  assert.equal(document.documentElement.lang, 'en')
  const system = fixture('invalid', 'ZH-Hant-TW')
  assert.equal(system.controller.preference(), 'system')
  assert.equal(system.locale.value, 'zh-CN')
  assert.equal(system.document.documentElement.lang, 'zh-CN')
})

test('setting language immediately updates locale, html language, and normalized cache', () => {
  const { controller, locale, document, values } = fixture(null, 'en-US')
  assert.equal(controller.preference(), 'system')
  controller.setPreference('zh-CN')
  assert.equal(controller.preference(), 'zh-CN')
  assert.equal(locale.value, 'zh-CN')
  assert.equal(document.documentElement.lang, 'zh-CN')
  assert.equal(values.get('rootyard.settings.language'), 'zh-CN')
  controller.setPreference('unsupported')
  assert.equal(controller.preference(), 'system')
  assert.equal(locale.value, 'en')
  assert.equal(document.documentElement.lang, 'en')
  assert.equal(values.get('rootyard.settings.language'), 'system')
})

test('system language events update system preference and preserve explicit choices', () => {
  const { controller, locale, navigator, document, window } = fixture()
  const stop = controller.startSystemLanguageSync()
  controller.setPreference('zh-CN')
  assert.equal(locale.value, 'zh-CN')
  assert.equal(document.documentElement.lang, 'zh-CN')
  controller.setPreference('system')
  navigator.language = 'en-GB'
  window.dispatchEvent(new Event('languagechange'))
  assert.equal(locale.value, 'en')
  assert.equal(document.documentElement.lang, 'en')
  controller.setPreference('zh-CN')
  navigator.language = 'en-US'
  window.dispatchEvent(new Event('languagechange'))
  assert.equal(locale.value, 'zh-CN')
  assert.equal(document.documentElement.lang, 'zh-CN')
  stop()
})

test('system listener is registered once and can unsubscribe and restart', () => {
  const { controller, locale, navigator, window } = fixture()
  const firstStop = controller.startSystemLanguageSync()
  const secondStop = controller.startSystemLanguageSync()
  assert.equal(secondStop, firstStop)
  firstStop()
  navigator.language = 'en-US'
  window.dispatchEvent(new Event('languagechange'))
  assert.equal(locale.value, 'zh-CN')
  controller.refreshSystemLocale()
  assert.equal(locale.value, 'en')
  const stop = controller.startSystemLanguageSync()
  navigator.language = 'zh-SG'
  window.dispatchEvent(new Event('languagechange'))
  assert.equal(locale.value, 'zh-CN')
  firstStop() // An old unsubscribe must not remove a newly registered listener.
  navigator.language = 'fr-FR'
  window.dispatchEvent(new Event('languagechange'))
  assert.equal(locale.value, 'en')
  stop()
})

test('manual system refresh is a no-op for an explicit preference', () => {
  const { controller, locale, navigator } = fixture('en')
  navigator.language = 'zh-CN'
  controller.refreshSystemLocale()
  assert.equal(locale.value, 'en')
})

test('controller works without browser globals or available storage', () => {
  const locale = { value: 'zh-CN' }
  const controller = createLanguageController({ locale, storage: null })
  assert.equal(controller.preference(), 'system')
  assert.equal(locale.value, 'en')
  assert.doesNotThrow(() => controller.startSystemLanguageSync()())
  assert.doesNotThrow(() => controller.setPreference('zh-CN'))
  assert.equal(locale.value, 'zh-CN')
})

test('singleton exposes composition-mode catalogs with English fallback and language controls', () => {
  assert.equal(i18n.mode, 'composition')
  assert.equal(i18n.global.fallbackLocale.value, 'en')
  assert.equal(i18n.global.t('common.cancel', {}, { locale: 'en' }), 'Cancel')
  assert.equal(i18n.global.t('common.cancel', {}, { locale: 'zh-CN' }), '取消')
  setLanguagePreference('zh-CN')
  assert.equal(getLanguagePreference(), 'zh-CN')
  assert.equal(i18n.global.locale.value, 'zh-CN')
  refreshSystemLocale()
  assert.equal(i18n.global.locale.value, 'zh-CN')
  assert.doesNotThrow(() => startSystemLanguageSync()())
  setLanguagePreference('system')
})

test('singleton import is SSR-safe with no browser globals and blocked storage', () => {
  const moduleUrl = new URL('./index.js', import.meta.url).href
  const script = `
    delete globalThis.window
    delete globalThis.document
    delete globalThis.navigator
    Object.defineProperty(globalThis, 'localStorage', {
      get() { throw new Error('blocked storage') },
    })
    const { i18n, getLanguagePreference } = await import(${JSON.stringify(moduleUrl)})
    if (i18n.global.locale.value !== 'en' || getLanguagePreference() !== 'system') process.exit(1)
  `
  assert.doesNotThrow(() => execFileSync(process.execPath, ['--input-type=module', '-e', script]))
})
