import assert from 'node:assert/strict'
import test from 'node:test'
import {
  LANGUAGE_PREFERENCES,
  LANGUAGE_CACHE_KEY,
  normalizeLanguagePreference,
  resolveSystemLocale,
  resolveEffectiveLocale,
  readCachedLanguagePreference,
  writeCachedLanguagePreference,
} from './locale.js'

test('language preferences normalize unsupported and legacy values to system', () => {
  assert.deepEqual(LANGUAGE_PREFERENCES, ['system', 'zh-CN', 'en'])
  assert.equal(Object.isFrozen(LANGUAGE_PREFERENCES), true)
  for (const [value, expected] of [
    ['system', 'system'], ['zh-CN', 'zh-CN'], ['en', 'en'],
    [undefined, 'system'], [null, 'system'], ['', 'system'],
    ['zh', 'system'], ['EN', 'system'], ['fr', 'system'], [42, 'system'],
  ]) {
    assert.equal(normalizeLanguagePreference(value), expected)
  }
})

test('system locale maps every zh variant to zh-CN and everything else to en', () => {
  for (const [language, expected] of [
    ['zh-CN', 'zh-CN'], ['ZH-hant-TW', 'zh-CN'], ['zh', 'zh-CN'],
    ['zh-HK', 'zh-CN'], ['en-US', 'en'], ['fr-FR', 'en'], ['', 'en'],
    [undefined, 'en'], [null, 'en'],
  ]) {
    assert.equal(resolveSystemLocale(language), expected)
  }
})

test('effective locale follows the system only for system or invalid preferences', () => {
  for (const [preference, language, expected] of [
    ['system', 'zh-TW', 'zh-CN'], ['system', 'en-GB', 'en'],
    ['en', 'zh-CN', 'en'], ['zh-CN', 'en-US', 'zh-CN'],
    ['invalid', 'zh-SG', 'zh-CN'], [undefined, '', 'en'],
  ]) {
    assert.equal(resolveEffectiveLocale(preference, language), expected)
  }
})

test('cache helpers use the Rootyard namespace and normalize reads and writes', () => {
  const values = new Map([['language', 'en'], ['rootyard.settings.language', 'zh-CN']])
  const storage = {
    getItem: (key) => values.get(key) ?? null,
    setItem: (key, value) => values.set(key, value),
  }
  assert.equal(LANGUAGE_CACHE_KEY, 'rootyard.settings.language')
  assert.equal(readCachedLanguagePreference(storage), 'zh-CN')
  writeCachedLanguagePreference('en', storage)
  assert.equal(values.get('rootyard.settings.language'), 'en')
  assert.equal(values.get('language'), 'en')
  values.set('rootyard.settings.language', 'fr')
  assert.equal(readCachedLanguagePreference(storage), 'system')
  writeCachedLanguagePreference('fr', storage)
  assert.equal(values.get('rootyard.settings.language'), 'system')
  values.delete('rootyard.settings.language')
  assert.equal(readCachedLanguagePreference(storage), 'system')
})

test('cache helpers survive unavailable storage and storage method failures', () => {
  const broken = {
    getItem() { throw new Error('disabled') },
    setItem() { throw new Error('quota') },
  }
  for (const storage of [null, {}, broken]) {
    assert.equal(readCachedLanguagePreference(storage), 'system')
    assert.doesNotThrow(() => writeCachedLanguagePreference('zh-CN', storage))
  }
})

test('cache helpers catch a throwing browser storage property lookup', () => {
  const original = Object.getOwnPropertyDescriptor(globalThis, 'localStorage')
  Object.defineProperty(globalThis, 'localStorage', {
    configurable: true,
    get() { throw new Error('storage blocked') },
  })
  try {
    assert.equal(readCachedLanguagePreference(), 'system')
    assert.doesNotThrow(() => writeCachedLanguagePreference('en'))
  } finally {
    if (original) Object.defineProperty(globalThis, 'localStorage', original)
    else delete globalThis.localStorage
  }
})
