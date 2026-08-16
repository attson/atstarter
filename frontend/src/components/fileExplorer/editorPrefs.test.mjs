import assert from 'node:assert/strict'
import test from 'node:test'
import {
  DEFAULT_PREFS,
  LEGACY_LINE_NUMBERS_KEY,
  PREFS_KEY,
  normalizePrefs,
  readPrefs,
  writePrefs,
} from './editorPrefs.js'

function fakeStorage(initial = {}) {
  const map = new Map(Object.entries(initial))
  return {
    getItem: (k) => (map.has(k) ? map.get(k) : null),
    setItem: (k, v) => map.set(k, String(v)),
    dump: () => Object.fromEntries(map),
  }
}

test('没有任何存储时用默认偏好', () => {
  assert.deepEqual(readPrefs(fakeStorage()), { ...DEFAULT_PREFS })
  assert.deepEqual(readPrefs(null), { ...DEFAULT_PREFS })
})

test('迁移旧的单键行号偏好', () => {
  const storage = fakeStorage({ [LEGACY_LINE_NUMBERS_KEY]: '1' })
  assert.equal(readPrefs(storage).lineNumbers, true)
  const off = fakeStorage({ [LEGACY_LINE_NUMBERS_KEY]: '0' })
  assert.equal(readPrefs(off).lineNumbers, false)
})

test('新键存在时优先于旧键', () => {
  const storage = fakeStorage({
    [LEGACY_LINE_NUMBERS_KEY]: '1',
    [PREFS_KEY]: JSON.stringify({ lineNumbers: false, tabSize: 4 }),
  })
  const prefs = readPrefs(storage)
  assert.equal(prefs.lineNumbers, false)
  assert.equal(prefs.tabSize, 4)
})

test('损坏的 JSON 退回默认值而不是抛异常', () => {
  const storage = fakeStorage({ [PREFS_KEY]: '{not json' })
  assert.deepEqual(readPrefs(storage), { ...DEFAULT_PREFS })
})

test('normalizePrefs 夹紧越界数值并过滤错误类型', () => {
  assert.equal(normalizePrefs({ tabSize: 0 }).tabSize, 1)
  assert.equal(normalizePrefs({ tabSize: 99 }).tabSize, 8)
  assert.equal(normalizePrefs({ tabSize: '4' }).tabSize, 4)
  assert.equal(normalizePrefs({ tabSize: 'x' }).tabSize, DEFAULT_PREFS.tabSize)
  assert.equal(normalizePrefs({ fontSize: 2 }).fontSize, 10)
  assert.equal(normalizePrefs({ fontSize: 200 }).fontSize, 24)
  assert.equal(normalizePrefs({ wrap: 'yes' }).wrap, DEFAULT_PREFS.wrap)
})

test('writePrefs 落盘的是归一化之后的值', () => {
  const storage = fakeStorage()
  const written = writePrefs(storage, { tabSize: 99, lineNumbers: true })
  assert.equal(written.tabSize, 8)
  assert.deepEqual(JSON.parse(storage.dump()[PREFS_KEY]), written)
})

test('writePrefs 在 storage 抛异常时不冒泡', () => {
  const hostile = { getItem: () => null, setItem: () => { throw new Error('quota') } }
  assert.doesNotThrow(() => writePrefs(hostile, DEFAULT_PREFS))
})
