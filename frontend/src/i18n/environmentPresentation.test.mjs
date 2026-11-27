import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'
import * as vue from 'vue'
import { createI18n, useI18n } from 'vue-i18n'
import * as icons from 'lucide-vue-next'
import * as environmentUi from '../projectEnvironmentUi.js'
import en from './messages/en.js'
import zhCN from './messages/zh-CN.js'
import { compileComponent, mountVueComponent } from '../testUtils/vueComponentHarness.mjs'

const read = (file) => readFileSync(new URL(`../components/${file}`, import.meta.url), 'utf8')
const flush = async () => { await new Promise((resolve) => setImmediate(resolve)); await vue.nextTick() }
function mountEnvironment(file, props, native = {}) {
  const dependencies = {
    vue, 'vue-i18n': { useI18n }, 'lucide-vue-next': icons,
    '../../projectEnvironmentUi.js': environmentUi,
    '../../../wailsjs/go/main/App': native,
    '../../../wailsjs/runtime/runtime': { EventsOn() {}, EventsOff() {} },
  }
  for (const name of ['AppButton', 'AppIcon']) {
    dependencies[`../ui/${name}.vue`] = compileComponent(read(`ui/${name}.vue`), dependencies)
  }
  for (const name of ['EnvironmentFieldEditor', 'CodexEnvironmentFieldEditor', 'EnvironmentDiffPanel', 'EnvironmentOverview', 'EnvironmentPresetManager', 'EnvironmentRepositoryManager', 'EnvironmentRuleManager']) {
    dependencies[`./${name}.vue`] = compileComponent(read(`environments/${name}.vue`), dependencies)
  }
  const i18n = createI18n({ legacy: false, locale: 'en', messages: { en, 'zh-CN': zhCN } })
  return { ...mountVueComponent(compileComponent(read(`environments/${file}`), dependencies), { props, i18n }), i18n }
}

const raw = '  原始 @ {name}|literal /srv/项目\n'
const plan = () => ({
  adapter: 'codex', target: { id: 'target-raw', worktreePath: '/srv/项目', configPath: '/srv/项目/.codex/config.toml' },
  changes: [{ key: 'sandbox_mode', operation: 'replace', current: [{ value: raw }], desired: { mode: 'set', value: 'danger-full-access' }, source: { name: raw } }],
  warnings: [{ code: 'dangerous_codex_permissions', message: raw, detail: 'warning detail @ {value}|原始' }],
  blockers: [], nativeDiff: ['-sandbox_mode = "read-only"', '+sandbox_mode = "danger-full-access" # 原始 @ {x}|'],
})

test('open preview and both confirmations follow locale while exact plans and native diff remain raw', async () => {
  const plans = [plan()]
  const snapshot = structuredClone(plans)
  const applied = []
  const host = mountEnvironment('EnvironmentDiffPanel.vue', { plans, results: [{ target: { id: 'target-raw' }, status: 'stale' }], onApply: (value) => applied.push(value) })
  try {
    // Use the real AppButton's rendered disabled state and the exposed reactive checks.
    const button = () => host.find((element) => element.type === 'button' && element.props.class?.includes('variant-primary') && element.props.disabled !== undefined)
    assert.match(host.visibleText(), /Native environment changes/)
    assert.match(host.visibleText(), /I reviewed these exact native environment changes/)
    assert.equal(button().props.disabled, true)
    host.state.confirmed = true
    await vue.nextTick()
    assert.equal(button().props.disabled, true)
    host.state.dangerousConfirmed = true
    await vue.nextTick()
    assert.equal(button().props.disabled, false)
    host.i18n.global.locale.value = 'zh-CN'
    await vue.nextTick()
    assert.match(host.visibleText(), /原生环境更改/)
    assert.match(host.visibleText(), /我已检查这些确切的原生环境更改/)
    assert.match(host.visibleText(), /我了解这会同时禁用沙箱和审批/)
    assert.match(host.visibleText(), /替换/)
    assert.match(host.visibleText(), /已过期/)
    assert.equal(host.find((element) => element.props['aria-label'] === '关闭预览')?.type, 'button')
    assert.equal(host.state.confirmed, true)
    assert.equal(host.state.dangerousConfirmed, true)
    assert.equal(button().props.disabled, false)
    for (const value of [raw, 'codex', 'sandbox_mode', 'danger-full-access', '/srv/项目/.codex/config.toml', ...plans[0].nativeDiff]) assert.ok(host.visibleText().includes(value), value)
    button().props.onClick({})
    assert.equal(applied[0], host.props.plans)
    assert.deepEqual(plans, snapshot)
    host.props.plans = [{ ...plan(), blockers: [{ message: raw }] }]
    await vue.nextTick()
    assert.equal(host.state.confirmed, false)
    assert.equal(host.state.dangerousConfirmed, false)
    host.state.confirmed = true
    host.state.dangerousConfirmed = true
    await vue.nextTick()
    assert.equal(button().props.disabled, true)
    assert.ok(host.visibleText().includes(raw))
    host.i18n.global.locale.value = 'en'
    await vue.nextTick()
    assert.ok(host.visibleText().includes(raw))
    host.props.plans = [{ ...plan(), changes: [{ ...plan().changes[0], operation: 'future-operation' }] }]
    host.props.results = [{ target: { id: 'raw' }, status: 'future-status' }]
    await vue.nextTick()
    assert.ok(host.visibleText().includes('future-operation'))
    assert.ok(host.visibleText().includes('future-status'))
  } finally { host.unmount() }
})

test('Codex controls translate labels and accessibility while enum changes round-trip native values', async () => {
  const updates = []
  const host = mountEnvironment('CodexEnvironmentFieldEditor.vue', {
    modelValue: { sandbox_mode: { mode: 'set', value: 'read-only' }, model: { mode: 'set', value: raw } },
    modelSuggestions: [raw], 'onUpdate:modelValue': (value) => updates.push(value),
  })
  try {
    assert.ok(host.find((element) => element.props['aria-label'] === 'Sandbox management mode'))
    host.i18n.global.locale.value = 'zh-CN'
    await vue.nextTick()
    assert.match(host.visibleText(), /沙箱/)
    assert.match(host.visibleText(), /仅限读取/)
    assert.ok(host.find((element) => element.props['aria-label'] === '沙箱管理方式'))
    assert.ok(host.find((element) => element.props['aria-label'] === '沙箱值'))
    assert.ok(host.find((element) => element.type === 'option' && element.props.value === 'danger-full-access'))
    host.state.setValue('sandbox_mode', 'danger-full-access')
    assert.equal(updates[0].sandbox_mode.value, 'danger-full-access')
    assert.equal(host.props.modelValue.model.value, raw)
    assert.ok(host.find((element) => element.type === 'input' && element.props.value === raw))
  } finally { host.unmount() }
})

test('open preset delete confirmation and save feedback follow locale while backend errors stay raw', async () => {
  const preset = { id: 'p', name: raw, description: raw, git: {}, codex: {} }
  let failure
  const document = { revision: 'rev', config: { defaults: {}, presets: [preset], rules: [] } }
  const host = mountEnvironment('EnvironmentPresetManager.vue', { document, state: {}, savePreset: async () => { if (failure) throw failure; return document }, clonePreset() {}, deletePreset() {} })
  try {
    await host.state.remove()
    await vue.nextTick()
    assert.match(host.visibleText(), /Confirm delete/)
    host.i18n.global.locale.value = 'zh-CN'
    await vue.nextTick()
    assert.match(host.visibleText(), /确认删除/)
    assert.equal(host.state.confirmDelete, true)
    await host.state.save()
    await vue.nextTick()
    assert.match(host.visibleText(), /已保存/)
    host.i18n.global.locale.value = 'en'
    await vue.nextTick()
    assert.match(host.visibleText(), /Saved/)
    failure = raw
    await host.state.save()
    await vue.nextTick()
    assert.ok(host.visibleText().includes(raw))
    host.i18n.global.locale.value = 'zh-CN'
    await vue.nextTick()
    assert.ok(host.visibleText().includes(raw))
    assert.equal(host.state.draft.name, raw)
    assert.equal(host.state.draft.description, raw)
  } finally { host.unmount() }
})

test('repository filters localize display statuses while selection and raw diagnostic messages stay unchanged', async () => {
  const repositories = [
    { targetId: 'eligible', aliases: [{ path: '/srv/原始 @ {x}|', ruleIds: ['rule'] }], adapters: { codex: { status: 'drifted', blockers: [] } } },
    { targetId: 'blocked', aliases: [{ path: '/srv/blocked' }], adapters: { git: { status: 'consistent', blockers: [{ message: raw }] } } },
  ]
  const selections = []
  const previews = []
  const host = mountEnvironment('EnvironmentRepositoryManager.vue', {
    state: { repositories }, rules: [{ id: 'rule', name: raw }], selection: new Set(['eligible']),
    doctorIssues: [{ code: 'native_issue', message: raw }],
    'onUpdate:selection': (value) => selections.push(value), onPreview: (paths) => previews.push(paths),
  })
  try {
    host.state.status = 'drifted'
    await vue.nextTick()
    assert.match(host.visibleText(), /Drifted/)
    host.i18n.global.locale.value = 'zh-CN'
    await vue.nextTick()
    assert.match(host.visibleText(), /有偏差/)
    assert.equal(host.state.status, 'drifted')
    assert.ok(host.find((element) => element.type === 'option' && element.props.value === 'blocked' && element.text === '已阻止'))
    assert.ok(host.find((element) => element.props.placeholder === '搜索仓库'))
    assert.ok(host.find((element) => element.text === raw))
    host.state.preview()
    assert.deepEqual(previews, [['/srv/原始 @ {x}|']])
    host.state.status = 'all'
    await vue.nextTick()
    assert.equal(host.find((element) => element.props['aria-label'] === '选择 blocked').props.disabled, true)
    host.state.toggleAll()
    assert.deepEqual([...selections[0]], [])
    host.state.toggle(repositories[1])
    assert.deepEqual([...selections[1]], ['eligible'])
    host.i18n.global.locale.value = 'en'
    await vue.nextTick()
    assert.ok(host.find((element) => element.text === raw))
    assert.deepEqual([...host.props.selection], ['eligible'])
  } finally { host.unmount() }
})

test('rule defaults, open delete confirmation and save feedback follow locale without rewriting rule data', async () => {
  const rule = { id: 'rule', name: raw, path: '/srv/原始 @ {x}|', git: {}, codex: {}, presetId: '', recursive: true, autoDiscover: false }
  const document = { revision: 'rev', config: { defaults: { git: {}, codex: {} }, presets: [], rules: [rule] } }
  let failure
  const host = mountEnvironment('EnvironmentRuleManager.vue', {
    document, state: {}, saveDefaults: async () => document,
    saveRule: async () => { if (failure) throw failure; return document }, deleteRule() {}, pickDirectory() {}, scanRule() {},
  })
  try {
    assert.ok(host.find((element) => element.type === 'input' && element.props.disabled === true && element.props.value === 'Global Defaults'))
    host.i18n.global.locale.value = 'zh-CN'
    await vue.nextTick()
    assert.ok(host.find((element) => element.type === 'input' && element.props.disabled === true && element.props.value === '全局默认值'))
    host.state.selectRule(rule)
    await host.state.remove()
    await vue.nextTick()
    assert.match(host.visibleText(), /确认删除/)
    host.i18n.global.locale.value = 'en'
    await vue.nextTick()
    assert.match(host.visibleText(), /Confirm delete/)
    assert.equal(host.state.confirmDelete, true)
    await host.state.save()
    await vue.nextTick()
    assert.match(host.visibleText(), /Saved/)
    host.i18n.global.locale.value = 'zh-CN'
    await vue.nextTick()
    assert.match(host.visibleText(), /已保存/)
    failure = raw
    await host.state.save()
    await vue.nextTick()
    assert.ok(host.find((element) => element.props.class === 'rule-message' && element.text === raw))
    host.i18n.global.locale.value = 'en'
    await vue.nextTick()
    assert.ok(host.find((element) => element.props.class === 'rule-message' && element.text === raw))
    assert.equal(host.state.draft.name, raw)
    assert.equal(host.state.draft.path, rule.path)
  } finally { host.unmount() }
})

test('overview scan progress translates counts and fallbacks while native scan paths remain raw', async () => {
  const host = mountEnvironment('EnvironmentOverview.vue', { document: {}, state: {}, scanning: true, progress: { repositories: 2, warnings: 1 } })
  try {
    assert.match(host.visibleText(), /Preparing scan/)
    host.i18n.global.locale.value = 'zh-CN'
    await vue.nextTick()
    assert.match(host.visibleText(), /正在准备扫描/)
    assert.match(host.visibleText(), /2 个仓库/)
    assert.match(host.visibleText(), /1 条警告/)
    host.props.progress = { path: raw, repositories: 2, warnings: 1 }
    await vue.nextTick()
    assert.ok(host.find((element) => element.props.title === raw && element.text === raw))
    host.i18n.global.locale.value = 'en'
    await vue.nextTick()
    assert.ok(host.find((element) => element.props.title === raw && element.text === raw))
  } finally { host.unmount() }
})

test('workspace navigation and native failure surface react to locale without translating the error', async () => {
  let failure = false
  const host = mountEnvironment('EnvironmentWorkspace.vue', {}, {
    GetEnvironmentDocument: async () => { if (failure) throw raw; return {} },
    GetEnvironmentState: async () => ({}),
  })
  try {
    await flush()
    assert.ok(host.find((element) => element.props.title === 'Directory Rules' && element.props['aria-label'] === 'Directory Rules'))
    host.i18n.global.locale.value = 'zh-CN'
    await vue.nextTick()
    assert.ok(host.find((element) => element.props.title === '目录规则' && element.props['aria-label'] === '目录规则'))
    failure = true
    await host.state.refreshEnvironmentData()
    await vue.nextTick()
    assert.equal(host.find((element) => element.props.role === 'alert').text, raw)
    host.i18n.global.locale.value = 'en'
    await vue.nextTick()
    assert.equal(host.find((element) => element.props.role === 'alert').text, raw)
  } finally { host.unmount() }
})
