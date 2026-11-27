import assert from 'node:assert/strict'
import test from 'node:test'
import { computed, ref } from 'vue'
import { createI18n } from 'vue-i18n'
import en from './messages/en.js'
import zhCN from './messages/zh-CN.js'
import { i18n } from './index.js'
import { confirmationPresentation, summaryTotalLabel } from './shellPresentation.js'

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

test('both catalogs render interpolation, plural counts, status, tooltips, and accessible announcements', () => {
  const { t, locale } = createI18n({ legacy: false, locale: 'en', messages: { en, 'zh-CN': zhCN } }).global
  const raw = '服务 @ {count}|api'
  for (const [language, expected] of [
    ['en', {
      removal: `Remove container “${raw}”?`, counts: ['no projects', '1 project', '2 projects'],
      status: 'Running', tooltip: 'Working tree has uncommitted changes', accessibility: 'Use pnpm',
      deletion: 'Selection deleted',
    }],
    ['zh-CN', {
      removal: `删除容器「${raw}」？`, counts: ['0 个项目', '1 个项目', '2 个项目'],
      status: '运行中', tooltip: '工作区有未提交改动', accessibility: '使用 pnpm',
      deletion: '已删除选中内容',
    }],
  ]) {
    locale.value = language
    assert.equal(t('dialogs.removeContainer.message', { name: raw }), expected.removal)
    assert.deepEqual([0, 1, 2].map((count) => summaryTotalLabel('projects', count, 9, t)), expected.counts)
    assert.equal(t('status.running'), expected.status)
    assert.equal(t('branches.dirty'), expected.tooltip)
    assert.equal(t('accessibility.useDetection', { type: 'pnpm' }), expected.accessibility)
    assert.equal(t('files.editorSearch.selectionDeleted'), expected.deletion)
  }
})

test('an open confirmation follows locale and live payload changes without interpreting container names', () => {
  const { t, locale } = createI18n({ legacy: false, locale: 'en', messages: { en, 'zh-CN': zhCN } }).global
  const name = 'api @ {name}|原始'
  const confirmation = ref({ show: true, kind: 'remove', payload: { name, state: 'running' } })
  const presentation = computed(() => confirmationPresentation(confirmation.value, t))
  assert.equal(presentation.value.message, `Container “${name}” is running and will be forcibly removed (rm -f). Continue?`)
  locale.value = 'zh-CN'
  assert.equal(presentation.value.message, `容器「${name}」正在运行，将强制删除（rm -f）。确认继续？`)
  confirmation.value.payload.state = 'exited'
  assert.equal(presentation.value.message, `删除容器「${name}」？`)
  locale.value = 'en'
  assert.equal(presentation.value.message, `Remove container “${name}”?`)
  assert.equal(confirmation.value.show, true)
  assert.equal(confirmation.value.payload.name, name)
})

test('the runtime English fallback renders English instead of keys for an absent test-only locale', () => {
  assert.ok(!i18n.global.availableLocales.includes('test-only'))
  for (const [key, values, expected] of [
    ['common.cancel', {}, 'Cancel'],
    ['dialogs.removeContainer.message', { name: 'api @ {name}|' }, 'Remove container “api @ {name}|”?'],
    ['status.running', {}, 'Running'],
    ['branches.dirty', {}, 'Working tree has uncommitted changes'],
    ['accessibility.useDetection', { type: 'pnpm' }, 'Use pnpm'],
    ['files.editorSearch.selectionDeleted', {}, 'Selection deleted'],
  ]) assert.equal(i18n.global.t(key, values, { locale: 'test-only' }), expected)
})

test('every catalog leaf renders without leaking its key in either locale', () => {
  const composer = createI18n({ legacy: false, locale: 'en', messages: { en, 'zh-CN': zhCN } }).global
  const values = {
    name: 'raw @ {name}|', count: 2, type: 'pnpm', field: 'user.name', theme: 'Dark', id: 'repository',
    version: '1.2.3', size: 4, line: 7, column: 2, head: 'abc123', code: 1, detail: 'raw detail',
    lines: 3, message: 'raw error', percent: 75, running: 1, state: 'future-state', total: 2,
  }
  for (const locale of ['en', 'zh-CN']) {
    composer.locale.value = locale
    for (const key of flattenMessages(en)) {
      const rendered = composer.t(key, values)
      assert.ok(rendered.trim(), `${locale}:${key} rendered empty`)
      assert.notEqual(rendered, key, `${locale}:${key} leaked its key`)
    }
  }
})
