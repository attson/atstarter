import assert from 'node:assert/strict'
import test from 'node:test'
import en from './messages/en.js'
import zhCN from './messages/zh-CN.js'

function flattenMessages(messages, prefix = '') {
  assert.ok(messages && typeof messages === 'object' && !Array.isArray(messages), prefix || 'catalog')
  return Object.entries(messages).flatMap(([key, value]) => {
    const path = prefix ? `${prefix}.${key}` : key
    if (value && typeof value === 'object' && !Array.isArray(value)) {
      return flattenMessages(value, path)
    }
    assert.equal(typeof value, 'string', `${path} must be a string`)
    assert.ok(value.trim().length > 0, `${path} must not be empty`)
    return [path]
  }).sort()
}

test('English and Chinese catalogs have the same non-empty string leaves', () => {
  const englishKeys = flattenMessages(en)
  assert.ok(englishKeys.length > 0)
  assert.deepEqual(flattenMessages(zhCN), englishKeys)
})
