// editorPrefs:文件浏览器/编辑器的本地偏好。
// 全部走 localStorage,和后端无关;纯函数便于 node:test 覆盖(注入一个假 storage)。

export const PREFS_KEY = 'fileBrowser.prefs'
// v1 只存了行号一项,单独一个 key。读取时迁移过来,之后不再写回旧 key。
export const LEGACY_LINE_NUMBERS_KEY = 'fileBrowser.lineNumbers'

export const DEFAULT_PREFS = Object.freeze({
  lineNumbers: false,
  wrap: false,
  tabSize: 2,
  fontSize: 13,
  showHidden: true,
})

export const TAB_SIZE_RANGE = Object.freeze({ min: 1, max: 8 })
export const FONT_SIZE_RANGE = Object.freeze({ min: 10, max: 24 })

function clampInt(value, fallback, range) {
  const n = Math.round(Number(value))
  if (!Number.isFinite(n)) return fallback
  if (n < range.min) return range.min
  if (n > range.max) return range.max
  return n
}

function bool(value, fallback) {
  return typeof value === 'boolean' ? value : fallback
}

export function normalizePrefs(raw) {
  const src = raw && typeof raw === 'object' ? raw : {}
  return {
    lineNumbers: bool(src.lineNumbers, DEFAULT_PREFS.lineNumbers),
    wrap: bool(src.wrap, DEFAULT_PREFS.wrap),
    tabSize: clampInt(src.tabSize, DEFAULT_PREFS.tabSize, TAB_SIZE_RANGE),
    fontSize: clampInt(src.fontSize, DEFAULT_PREFS.fontSize, FONT_SIZE_RANGE),
    showHidden: bool(src.showHidden, DEFAULT_PREFS.showHidden),
  }
}

export function readPrefs(storage) {
  if (!storage) return { ...DEFAULT_PREFS }
  let stored = null
  try {
    stored = JSON.parse(storage.getItem(PREFS_KEY) || 'null')
  } catch {
    stored = null // 手改坏了就当没有,不要让整个文件面板打不开
  }
  if (stored) return normalizePrefs(stored)
  const legacy = storage.getItem(LEGACY_LINE_NUMBERS_KEY)
  if (legacy !== null) return normalizePrefs({ lineNumbers: legacy === '1' })
  return { ...DEFAULT_PREFS }
}

export function writePrefs(storage, prefs) {
  const next = normalizePrefs(prefs)
  if (storage) {
    try {
      storage.setItem(PREFS_KEY, JSON.stringify(next))
    } catch {
      // 隐私模式下 setItem 会抛;偏好丢失可以接受,不该冒泡成界面报错。
    }
  }
  return next
}
