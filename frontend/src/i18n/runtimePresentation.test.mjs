import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'
import * as vue from 'vue'
import { createI18n, useI18n } from 'vue-i18n'
import * as icons from 'lucide-vue-next'
import * as dockerState from '../dockerState.js'
import * as updateSchedule from '../updateSchedule.js'
import * as appSettings from '../appSettings.js'
import * as logBatches from '../logBatches.js'
import * as projectDetection from '../projectDetection.js'
import * as shellPresentation from './shellPresentation.js'
import en from './messages/en.js'
import zhCN from './messages/zh-CN.js'
import { compileComponent, mountVueComponent } from '../testUtils/vueComponentHarness.mjs'

const read = (file) => readFileSync(new URL(`../components/${file}`, import.meta.url), 'utf8')
const flush = async () => { await new Promise((resolve) => setImmediate(resolve)); await vue.nextTick() }
const updateState = (overrides = {}) => ({
  current: 'v1.0.0+原始', latest: 'v2.0.0@{release}|raw', available: false,
  notes: ' # 原始 release | @ {name}\n  no translation\n', checking: false,
  downloading: false, downloadPct: 0, ready: false, error: '',
  assetUrl: 'https://example.test/releases/v2.0.0/rootyard.pkg', assetSize: 1572864,
  canInstall: true, lastCheckAt: 1795734000000, ...overrides,
})

// Only native/browser boundaries are doubled. The real component, Vue reactivity,
// i18n, child controls, log filtering, and terminal chunk formatting remain active.
function mountRuntime(file, { props = {}, native = {}, runtime = {} } = {}) {
  const subscriptions = new Map()
  const writes = []
  let resets = 0
  let selection = ''
  const terminal = {
    options: {}, write: (chunk) => writes.push(chunk), reset: () => { resets++ },
    clear() {}, loadAddon() {}, open() {}, dispose() {}, getSelection: () => selection,
  }
  const globals = new Map(['window', 'MutationObserver', 'ResizeObserver', 'getComputedStyle'].map((key) => [key, Object.getOwnPropertyDescriptor(globalThis, key)]))
  globalThis.window = new EventTarget()
  globalThis.MutationObserver = class { observe() {} disconnect() {} }
  globalThis.ResizeObserver = class { observe() {} disconnect() {} }
  globalThis.getComputedStyle = () => ({ getPropertyValue: () => '' })
  const dependencies = {
    vue: { ...vue, Transition: vue.BaseTransition }, 'vue-i18n': { useI18n },
    'lucide-vue-next': icons,
    '../dockerState.js': dockerState, '../updateSchedule': updateSchedule,
    '../appSettings.js': appSettings, '../logBatches.js': logBatches,
    '../projectDetection.js': projectDetection,
    '../i18n/shellPresentation.js': shellPresentation,
    '../composables/useTheme': { useTheme: () => ({ resolvedTheme: vue.ref('dark') }) },
    '@xterm/xterm': { Terminal: class { constructor() { return terminal } } },
    '@xterm/addon-fit': { FitAddon: class { fit() {} } },
    '@xterm/addon-web-links': { WebLinksAddon: class {} },
    '../../wailsjs/go/main/App': {
      GetLogs: async () => [], ClearLogs: async () => {},
      DockerAvailable: async () => ({ available: true, reason: '' }), ListContainers: async () => [],
      ListComposeServices: async () => [], StopFollowComposeLogs: async () => {},
      UpdateGetState: async () => updateState(), UpdateCheck: async () => updateState(),
      ...native,
    },
    '../../wailsjs/runtime/runtime': {
      EventsOn: (name, callback) => { subscriptions.set(name, callback); return () => subscriptions.delete(name) },
      EventsOff: (name) => subscriptions.delete(name),
      ClipboardSetText: async () => true, BrowserOpenURL() {}, ...runtime,
    },
  }
  for (const name of ['AppButton', 'AppPill', 'AppIcon']) {
    dependencies[`./ui/${name}.vue`] = compileComponent(read(`ui/${name}.vue`), dependencies)
  }
  dependencies['./LogPanel.vue'] = compileComponent(read('LogPanel.vue'), dependencies)
  dependencies['./FileBrowser.vue'] = { render: () => null }
  dependencies['./DetectionSwitch.vue'] = compileComponent(read('DetectionSwitch.vue'), dependencies)
  const i18n = createI18n({ legacy: false, locale: 'en', messages: { en, 'zh-CN': zhCN } })
  let host
  try {
    host = mountVueComponent(compileComponent(read(file), dependencies), {
      props, i18n,
      beforeMount({ document }) {
        document.hidden = false
        document.documentElement = { getAttribute: () => 'dark' }
      },
    })
  } catch (error) { restoreGlobals(); throw error }
  function restoreGlobals() {
    for (const [key, descriptor] of globals) {
      if (descriptor) Object.defineProperty(globalThis, key, descriptor)
      else delete globalThis[key]
    }
  }
  return {
    ...host, i18n, subscriptions, writes, resetCount: () => resets,
    selectText: (raw) => { selection = raw },
    unmount() { try { host.unmount() } finally { restoreGlobals() } },
  }
}

test('mounted compose aggregate, service states, and accessible actions follow locale without rewriting service data', async () => {
  const project = { id: 'raw-project', name: '原始项目 @ {name}', path: '/srv/原始 project', detectedType: 'compose', detections: [] }
  const services = [
    { name: 'api @ {name}|原始', state: 'running', image: 'registry/镜像:latest', ports: ['127.0.0.1:8080->80/tcp'] },
    { name: 'worker', state: 'exited', image: 'worker:v2', ports: [] },
  ]
  const calls = []
  const host = mountRuntime('ComposeDetail.vue', { props: { project, dockerAvailable: true }, native: {
    ListComposeServices: async (id) => { assert.equal(id, project.id); return services },
    ComposeUp: async (...args) => calls.push(args),
  } })
  try {
    await flush()
    assert.match(host.visibleText(), /1\/2 running/)
    assert.match(host.visibleText(), /Exited/)
    assert.ok(host.find((element) => element.props.title === 'Start' && element.props['aria-label'] === 'Start'))
    host.i18n.global.locale.value = 'zh-CN'
    await vue.nextTick()
    assert.match(host.visibleText(), /1\/2 个运行中/)
    assert.match(host.visibleText(), /已退出/)
    assert.ok(host.find((element) => element.props.title === '启动' && element.props['aria-label'] === '启动'))
    for (const raw of [project.name, project.path, services[0].name, services[0].image, services[0].ports[0]]) assert.ok(host.visibleText().includes(raw))
    await host.state.upService(services[0])
    assert.deepEqual(calls, [[project.id, services[0].name]])
    host.state.services = [{ ...services[0], state: 'future-native-code' }]
    await vue.nextTick()
    assert.match(host.visibleText(), /future-native-code/)
    assert.match(host.visibleText(), /已停止/)
    host.state.services = []
    await vue.nextTick()
    assert.match(host.visibleText(), /未知/)
    host.i18n.global.locale.value = 'en'
    await vue.nextTick()
    assert.match(host.visibleText(), /Unknown/)
    assert.equal(services[0].state, 'running')
  } finally { host.unmount() }
})

test('mounted container summaries, state labels, search, and controls follow locale while native names and status details stay raw', async () => {
  const rawStatus = 'Up 2 hours (healthy) 原始'
  const containers = [
    { id: 'raw-1', name: 'api 原始 @ {name}', compose: 'stack 原始', service: 'api 原始', state: 'running', status: rawStatus, image: 'registry/镜像:v1' },
    { id: 'raw-2', name: 'worker', compose: 'stack 原始', service: '', state: 'exited', status: 'Exited (137) 原始', image: 'worker:v2' },
    { id: 'raw-3', name: 'standalone 原始', compose: '', service: '', state: 'paused', status: '', image: 'redis:7' },
  ]
  const host = mountRuntime('ContainerPanel.vue', { native: { ListContainers: async () => containers, FollowContainerLogs: async () => {}, StopFollowContainerLogs: async () => {} } })
  try {
    await flush()
    assert.ok(host.find((element) => element.props.placeholder === 'Filter containers…'))
    host.state.selectCompose('stack 原始')
    await vue.nextTick()
    assert.match(host.visibleText(), /1 running/)
    assert.match(host.visibleText(), /2 containers/)
    assert.match(host.visibleText(), /Images/)
    assert.match(host.visibleText(), /Exited/)
    host.i18n.global.locale.value = 'zh-CN'
    await vue.nextTick()
    assert.match(host.visibleText(), /1 个运行中/)
    assert.match(host.visibleText(), /2 个容器/)
    assert.match(host.visibleText(), /镜像/)
    assert.ok(host.find((element) => element.props.title === '启动' && element.props['aria-label'] === '启动'))
    assert.ok(host.visibleText().includes(containers[0].service))
    assert.ok(host.visibleText().includes(containers[0].image))
    host.state.select(containers[0])
    await flush()
    assert.ok(host.visibleText().includes(rawStatus))
    assert.ok(host.visibleText().includes(containers[0].name))
    assert.match(host.visibleText(), /运行中/)
    host.state.select(containers[2])
    await flush()
    assert.match(host.visibleText(), /已暂停/)
    host.state.dockerInfo = { available: false, reason: '  Cannot connect: 原始 /var/run/docker.sock\n' }
    await vue.nextTick()
    assert.ok(host.visibleText().includes('  Cannot connect: 原始 /var/run/docker.sock\n'))
    assert.match(host.visibleText(), /Docker 不可用/)
    host.i18n.global.locale.value = 'en'
    await vue.nextTick()
    assert.match(host.visibleText(), /Docker is unavailable/)
    assert.ok(host.visibleText().includes('  Cannot connect: 原始 /var/run/docker.sock\n'))
    assert.equal(containers[0].state, 'running')
  } finally { host.unmount() }
})

test('mounted log status, menu and active toast follow locale without rewriting or replaying terminal bytes', async () => {
  const copied = []
  let clipboardWorks = true
  const raw = '\u001b[31m日志 @ {name} | stdout\u001b[0m'
  const host = mountRuntime('LogPanel.vue', {
    props: { projectId: 'raw-run', status: { State: 'running', ExitCode: 137 } },
    runtime: { ClipboardSetText: async (text) => { copied.push(text); return clipboardWorks } },
  })
  try {
    await flush()
    assert.match(host.visibleText(), /Running/)
    assert.match(host.visibleText(), /waiting for output/)
    host.i18n.global.locale.value = 'zh-CN'
    await vue.nextTick()
    assert.match(host.visibleText(), /运行中/)
    assert.match(host.visibleText(), /等待输出/)
    host.subscriptions.get('logs:batch')({ lines: [{ id: 'other', text: 'ignored' }, { id: 'raw-run', stream: 'stdout', text: raw }] })
    await vue.nextTick()
    assert.deepEqual(host.writes, [raw + '\r\n'])
    const resets = host.resetCount()
    host.selectText(raw)
    host.state.openMenu({ clientX: 10, clientY: 20 })
    await vue.nextTick()
    assert.match(host.visibleText(), /复制选中/)
    host.i18n.global.locale.value = 'en'
    await vue.nextTick()
    assert.match(host.visibleText(), /Copy selection/)
    await host.state.copySelection()
    await vue.nextTick()
    assert.match(host.visibleText(), /Copied/)
    host.i18n.global.locale.value = 'zh-CN'
    await vue.nextTick()
    assert.match(host.visibleText(), /已复制/)
    assert.deepEqual(copied, [raw])
    host.props.status = { State: 'exited', ExitCode: 137 }
    await vue.nextTick()
    assert.match(host.visibleText(), /已退出.*137/)
    host.i18n.global.locale.value = 'en'
    await vue.nextTick()
    assert.match(host.visibleText(), /Exited.*137/)
    assert.deepEqual(host.writes, [raw + '\r\n'])
    assert.equal(host.resetCount(), resets)
    host.props.status = { State: 'error', ExitCode: -1 }
    await vue.nextTick()
    assert.match(host.visibleText(), /Start error/)
    host.props.status = { State: 'stopped' }
    await vue.nextTick()
    assert.match(host.visibleText(), /Not running/)
    host.selectText('')
    host.state.openMenu({ clientX: 0, clientY: 0 })
    await host.state.copySelection()
    await vue.nextTick()
    assert.match(host.visibleText(), /No selection/)
    host.i18n.global.locale.value = 'zh-CN'
    await vue.nextTick()
    assert.match(host.visibleText(), /无选中内容/)
    clipboardWorks = false
    host.selectText(raw)
    host.state.openMenu({ clientX: 0, clientY: 0 })
    await host.state.copySelection()
    await vue.nextTick()
    assert.match(host.visibleText(), /复制失败/)
    host.i18n.global.locale.value = 'en'
    await vue.nextTick()
    assert.match(host.visibleText(), /Copy failed/)
    assert.deepEqual(host.writes, [raw + '\r\n'])
    assert.equal(host.resetCount(), resets)
  } finally { host.unmount() }
})

test('mounted update stages and dismiss labels follow locale with raw versions, notes and backend errors', async () => {
  const host = mountRuntime('UpdateBanner.vue')
  try {
    await flush()
    const cases = [
      [{ checking: true }, /Checking for updates/, /正在检查更新/],
      [{ available: true }, /New version.*available/, /新版本.*可用/],
      [{ downloading: true, downloadPct: 42 }, /Downloading.*42%/, /下载.*42%/],
      [{ downloading: true, downloadPct: 100 }, /Verifying/, /正在验证/],
      [{ ready: true }, /downloaded/, /已下载完成/],
      [{}, /up to date/, /已是最新版本/],
    ]
    for (const [state, english, chinese] of cases) {
      host.state.manualNotice = true
      host.state.state = updateState(state)
      host.i18n.global.locale.value = 'en'
      await vue.nextTick()
      assert.match(host.visibleText(), english)
      const dismiss = host.find((element) => element.props['aria-label'] === 'Dismiss update notice')
      if (!state.downloading) assert.ok(dismiss)
      host.i18n.global.locale.value = 'zh-CN'
      await vue.nextTick()
      assert.match(host.visibleText(), chinese)
      if (!state.downloading) assert.ok(host.find((element) => element.props['aria-label'] === '关闭更新提示'))
      assert.equal(host.state.state.latest, 'v2.0.0@{release}|raw')
      assert.equal(host.state.state.notes, ' # 原始 release | @ {name}\n  no translation\n')
      if (state.available || state.ready) {
        assert.ok(host.visibleText().includes('v2.0.0@{release}|raw'))
      }
    }
    const raw = '  SHA256SUMS signature verification failed: 原始\n'
    host.state.state = updateState({ error: raw })
    await vue.nextTick()
    assert.ok(host.visibleText().includes(raw))
    host.i18n.global.locale.value = 'en'
    await vue.nextTick()
    assert.ok(host.visibleText().includes(raw))
  } finally { host.unmount() }
})

test('update native rejections preserve their detail and use a reactive fallback only when no detail exists', async () => {
  let rejection
  let returnedState
  let rejectInstall
  let pendingInstall = false
  const reject = async () => { if (returnedState) return returnedState; throw rejection }
  const host = mountRuntime('UpdateBanner.vue', { native: {
    UpdateCheck: reject, UpdateStartDownload: reject, UpdateCancel: reject,
    UpdateInstall: () => pendingInstall ? new Promise((resolve, reject) => { rejectInstall = reject }) : reject(),
  } })
  try {
    await flush()
    host.state.manualNotice = true
    const raw = '  permission denied: /Applications/原始.app\nNative detail @ {name}|raw\n'
    for (const action of ['check', 'download', 'cancel', 'install']) {
      for (const detail of [raw, new Error(raw), { message: raw }]) {
        rejection = detail
        await host.state[action]()
        await vue.nextTick()
        const error = host.find((element) => element.props.class === 'sub')
        assert.equal(error?.text, raw)
        host.i18n.global.locale.value = 'zh-CN'
        await vue.nextTick()
        assert.equal(host.find((element) => element.props.class === 'sub')?.text, raw)
        host.i18n.global.locale.value = 'en'
      }
      for (const detail of [undefined, null, '', new Error(''), {}]) {
        rejection = detail
        await host.state[action]()
        await vue.nextTick()
        assert.equal(host.find((element) => element.props.class === 'sub')?.text, 'Update failed')
        host.i18n.global.locale.value = 'zh-CN'
        await vue.nextTick()
        assert.equal(host.find((element) => element.props.class === 'sub')?.text, '更新失败')
        host.i18n.global.locale.value = 'en'
      }
      returnedState = updateState({ error: raw })
      await host.state[action]()
      await vue.nextTick()
      assert.equal(host.state.state.error, raw)
      assert.equal(host.find((element) => element.props.class === 'sub')?.text, raw)
      host.i18n.global.locale.value = 'zh-CN'
      await vue.nextTick()
      assert.equal(host.find((element) => element.props.class === 'sub')?.text, raw)
      host.i18n.global.locale.value = 'en'
      returnedState = undefined
    }
    host.state.state = updateState({ ready: true })
    host.subscriptions.get('update:state')(updateState({ ready: true }))
    pendingInstall = true
    const installing = host.state.install()
    await vue.nextTick()
    assert.match(host.visibleText(), /Installing/)
    // A concurrent native check emits checking=true while the install Promise
    // is still pending. Installation must retain its non-dismissible surface.
    host.subscriptions.get('update:state')(updateState({ ready: true, checking: true }))
    await vue.nextTick()
    assert.equal(host.state.installing, true)
    assert.match(host.visibleText(), /Installing/)
    assert.doesNotMatch(host.visibleText(), /Checking for updates/)
    assert.equal(host.find((element) => element.props['aria-label'] === 'Dismiss update notice'), undefined)
    assert.ok(host.find((element) => element.type === 'button' && element.props.disabled === true))
    host.i18n.global.locale.value = 'zh-CN'
    await vue.nextTick()
    assert.match(host.visibleText(), /正在安装/)
    assert.doesNotMatch(host.visibleText(), /正在检查更新/)
    assert.equal(host.find((element) => element.props['aria-label'] === '关闭更新提示'), undefined)
    returnedState = updateState({ ready: true, checking: false })
    await host.state.check()
    await vue.nextTick()
    assert.match(host.visibleText(), /正在安装/)
    assert.equal(host.find((element) => element.props['aria-label'] === '关闭更新提示'), undefined)
    returnedState = undefined
    rejectInstall(new Error(raw))
    await installing
    await vue.nextTick()
    assert.equal(host.find((element) => element.props.class === 'sub')?.text, raw)
    assert.doesNotMatch(host.visibleText(), /正在安装/)
    assert.equal(host.state.installing, false)
    host.i18n.global.locale.value = 'en'
    await vue.nextTick()
    assert.equal(host.find((element) => element.props.class === 'sub')?.text, raw)
  } finally { host.unmount() }
})
