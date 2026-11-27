import test from 'node:test'
import assert from 'node:assert/strict'
import { applyLoadedSettings, saveSettingsAndApply } from './settingsFlow.js'

test('loaded backend settings replace the cached preference', () => {
  let preference = 'zh-CN'
  const settings = applyLoadedSettings(
    { language: 'en', logFontFamily: ' JetBrains Mono ', logFontSize: 99 },
    (value) => { preference = value },
  )
  assert.equal(preference, 'en')
  assert.equal(settings.language, 'en')
  assert.equal(settings.logFontFamily, 'JetBrains Mono')
  assert.equal(settings.logFontSize, 20)
})

test('a backend language that is missing or invalid replaces cache with system', () => {
  for (const raw of [{}, { language: 'fr' }]) {
    let preference = 'zh-CN'
    const settings = applyLoadedSettings(raw, (value) => { preference = value })
    assert.equal(preference, 'system')
    assert.equal(settings.language, 'system')
  }
})

test('successful save applies the backend-normalized language', async () => {
  const applied = []
  const draft = { language: 'zh-CN' }
  const saved = await saveSettingsAndApply(
    draft,
    async (raw) => {
      assert.equal(raw, draft)
      assert.deepEqual(applied, [])
      return { language: 'en', logFontFamily: 'monospace', logFontSize: 11 }
    },
    (value) => applied.push(value),
  )
  assert.equal(saved.language, 'en')
  assert.deepEqual(applied, ['en'])
})

test('a pending save keeps the active preference until the backend responds', async () => {
  let resolveSave
  let preference = 'en'
  const response = new Promise((resolve) => { resolveSave = resolve })
  const saving = saveSettingsAndApply(
    { language: 'zh-CN' },
    () => response,
    (value) => { preference = value },
  )
  assert.equal(preference, 'en')
  resolveSave({ language: 'zh-CN' })
  await saving
  assert.equal(preference, 'zh-CN')
})

test('failed save preserves language and the original error details', async () => {
  const applied = []
  const failure = new Error('disk full: /Users/me/config.json\n原始错误')
  await assert.rejects(() => saveSettingsAndApply(
    { language: 'zh-CN' },
    async () => { throw failure },
    (value) => applied.push(value),
  ), (error) => error === failure)
  assert.deepEqual(applied, [])
})

test('saved settings normalize unsupported backend language before applying it', async () => {
  const applied = []
  const saved = await saveSettingsAndApply(
    { language: 'zh-CN' },
    async () => ({ language: 'unsupported', logFontSize: 99 }),
    (value) => applied.push(value),
  )
  assert.equal(saved.language, 'system')
  assert.equal(saved.logFontSize, 20)
  assert.deepEqual(applied, ['system'])
})
