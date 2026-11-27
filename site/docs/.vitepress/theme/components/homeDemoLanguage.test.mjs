import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'
import * as vue from '../../../../../frontend/node_modules/vue/index.mjs'
import { createI18n } from '../../../../../frontend/node_modules/vue-i18n/dist/vue-i18n.mjs'
import { createLanguageController } from '../../../../../frontend/src/i18n/index.js'
import en from '../../../../../frontend/src/i18n/messages/en.js'
import zhCN from '../../../../../frontend/src/i18n/messages/zh-CN.js'
import { compileComponent, mountVueComponent } from '../../../../../frontend/src/testUtils/vueComponentHarness.mjs'

// Compile the real theme entry as a Vue host. Only VitePress routing and browser
// DOM/observer boundaries are substituted; its setup and language controller run.
test('the mounted site host follows system language, preserves explicit choices, and cleans up', async () => {
  const originals = new Map(['window', 'MutationObserver'].map((key) => [key, Object.getOwnPropertyDescriptor(globalThis, key)]))
  const window = new EventTarget()
  globalThis.window = window
  globalThis.MutationObserver = class { observe() {} disconnect() {} }
  const navigator = { language: 'en-US' }
  const documentElement = { lang: '', dataset: {}, classList: { contains: () => false } }
  const i18n = createI18n({ legacy: false, locale: 'en', messages: { en, 'zh-CN': zhCN } })
  const controller = createLanguageController({
    locale: i18n.global.locale, window, navigator, document: { documentElement },
    storage: { getItem: () => 'system', setItem() {} },
  })
  let host
  try {
    const source = readFileSync(new URL('../index.js', import.meta.url), 'utf8')
    const theme = compileComponent(`<script>${source}</script><template><span>{{ $t('status.unknown') }}</span></template>`, {
      vue, 'vitepress/theme': {}, vitepress: { useRoute: () => vue.reactive({ path: '/' }) },
      '../../../../frontend/src/i18n/index.js': { i18n, startSystemLanguageSync: controller.startSystemLanguageSync },
    })
    host = mountVueComponent(theme, { props: {}, i18n, beforeMount({ document }) {
      document.documentElement = documentElement
      document.querySelector = () => null
      document.querySelectorAll = () => []
      document.body = { classList: { toggle() {}, remove() {} } }
    } })
    await vue.nextTick()
    assert.equal(host.visibleText(), 'Unknown')
    assert.equal(documentElement.lang, 'en')
    navigator.language = 'zh-TW'
    window.dispatchEvent(new Event('languagechange'))
    await vue.nextTick()
    assert.equal(host.visibleText(), '未知')
    assert.equal(documentElement.lang, 'zh-CN')
    controller.setPreference('en')
    window.dispatchEvent(new Event('languagechange'))
    await vue.nextTick()
    assert.equal(host.visibleText(), 'Unknown')
    assert.equal(documentElement.lang, 'en')
    controller.setPreference('system')
    await vue.nextTick()
    assert.equal(host.visibleText(), '未知')
    host.unmount()
    host = null
    navigator.language = 'en-US'
    window.dispatchEvent(new Event('languagechange'))
    assert.equal(i18n.global.locale.value, 'zh-CN', 'the unmounted host must remove its listener')
    assert.equal(documentElement.lang, 'zh-CN')
  } finally {
    host?.unmount()
    for (const [key, descriptor] of originals) {
      if (descriptor) Object.defineProperty(globalThis, key, descriptor)
      else delete globalThis[key]
    }
  }
})
