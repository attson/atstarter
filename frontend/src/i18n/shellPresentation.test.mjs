import assert from 'node:assert/strict'
import test from 'node:test'
import { computed, ref } from 'vue'
import { createI18n } from 'vue-i18n'
import en from './messages/en.js'
import zhCN from './messages/zh-CN.js'
import { confirmationPresentation, detectionChoiceLabel, summaryTotalLabel } from './shellPresentation.js'

function translator(locale = 'en') {
  return createI18n({ legacy: false, locale, messages: { en, 'zh-CN': zhCN } }).global
}

test('open removal confirmation follows locale and live container state while preserving the raw name', () => {
  const { t, locale } = translator()
  const name = '服务 @ {count} | api'
  const confirmation = ref({ show: true, kind: 'remove', payload: { id: 'c1', name, state: 'running' }, danger: true })
  const presentation = computed(() => confirmationPresentation(confirmation.value, t))

  assert.deepEqual(presentation.value, {
    title: 'Remove container',
    message: `Container “${name}” is running and will be forcibly removed (rm -f). Continue?`,
    confirmText: 'Remove',
  })
  locale.value = 'zh-CN'
  assert.deepEqual(presentation.value, {
    title: '删除容器',
    message: `容器「${name}」正在运行，将强制删除（rm -f）。确认继续？`,
    confirmText: '删除',
  })
  confirmation.value.payload.state = 'exited'
  assert.equal(presentation.value.message, `删除容器「${name}」？`)
  assert.equal(confirmation.value.show, true)
  assert.equal(confirmation.value.payload.name, name)
})

test('compose down and reset confirmations follow the active locale without storing display copy', () => {
  const { t, locale } = translator()
  const state = ref({ show: true, kind: 'down', payload: 'p1', danger: true })
  const presentation = computed(() => confirmationPresentation(state.value, t))
  assert.deepEqual(presentation.value, {
    title: 'Compose Down',
    message: 'Stop and remove all containers and networks for this compose project. Volumes will be kept. Continue?',
    confirmText: 'Down',
  })
  locale.value = 'zh-CN'
  assert.equal(presentation.value.message, '将停止并删除该 compose 项目的所有容器与网络（数据卷保留）。确认继续？')
  state.value = { show: true, kind: 'reset-projects', payload: null, danger: true }
  assert.deepEqual(presentation.value, {
    title: '重置项目列表',
    message: '将清空所有项目和启动分组，保留扫描工作区。正在运行的普通项目会先停止。确认继续？',
    confirmText: '重置',
  })
  locale.value = 'en'
  assert.deepEqual(presentation.value, {
    title: 'Reset project list',
    message: 'Clear all projects and launch groups while keeping scanned workspaces. Running non-compose projects will be stopped first. Continue?',
    confirmText: 'Reset',
  })
})

test('shell totals use the selected domain count and plural grammar for both locales', () => {
  const { t, locale } = translator()
  for (const [tab, projects, containers, want] of [
    ['projects', 0, 8, 'no projects'],
    ['projects', 1, 8, '1 project'],
    ['projects', 2, 8, '2 projects'],
    ['containers', 8, 0, 'no containers'],
    ['containers', 8, 1, '1 container'],
    ['containers', 8, 2, '2 containers'],
  ]) assert.equal(summaryTotalLabel(tab, projects, containers, t), want)
  locale.value = 'zh-CN'
  for (const count of [0, 1, 2]) {
    assert.equal(summaryTotalLabel('projects', count, 8, t), `${count} 个项目`)
    assert.equal(summaryTotalLabel('containers', 8, count, t), `${count} 个容器`)
  }
})

test('detection choices translate the unknown fallback while retaining tool labels and future types', () => {
  const { t, locale } = translator()
  assert.equal(detectionChoiceLabel({ type: 'node-pnpm' }, t), 'pnpm')
  assert.equal(detectionChoiceLabel({ type: 'custom-tool' }, t), 'custom-tool')
  assert.equal(detectionChoiceLabel({ type: 'unknown' }, t), 'Unknown')
  locale.value = 'zh-CN'
  assert.equal(detectionChoiceLabel({ type: 'unknown' }, t), '未知')
  assert.equal(detectionChoiceLabel(null, t), '未知')
  assert.equal(detectionChoiceLabel({ type: 'compose' }, t), 'compose')
})
