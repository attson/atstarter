// Branch switcher wiring checks and rendered behavior using the real Vue/i18n runtime.
import assert from 'node:assert/strict'
import fs from 'node:fs'
import test from 'node:test'
import * as vue from 'vue'
import { babelParse, compileScript, compileTemplate, parse } from 'vue/compiler-sfc'
import { createI18n, useI18n } from 'vue-i18n'
import * as icons from 'lucide-vue-next'
import * as gitBranches from './gitBranches.js'
import en from './i18n/messages/en.js'
import zhCN from './i18n/messages/zh-CN.js'

const read = (name) => fs.readFileSync(new URL(name, import.meta.url), 'utf8')

const switcher = process.env.ROOTYARD_BRANCH_SWITCHER_SOURCE
  ? fs.readFileSync(process.env.ROOTYARD_BRANCH_SWITCHER_SOURCE, 'utf8')
  : read('./components/BranchSwitcher.vue')
const projectDetail = read('./components/ProjectDetail.vue')

test('分支标签变成可点的切换入口', () => {
  assert.match(switcher, /data-test="branch-pill"/)
  assert.match(switcher, /data-test="branch-menu"/)
  assert.match(switcher, /clickable/)
})

test('切换器接的是三个 git 绑定', () => {
  assert.match(switcher, /ListProjectBranches/)
  assert.match(switcher, /CheckoutProjectBranch/)
  assert.match(switcher, /CreateProjectBranch/)
})

test('分支列表每次打开都重新拉,避免展示过期数据', () => {
  assert.match(switcher, /async function toggleOpen/)
  assert.match(switcher, /await load\(\)/)
})

// Compile the real components with Vue's existing compiler. The only dependency
// double is the native Wails boundary; Vue, i18n, and child components stay real.
function compileComponent(source, dependencies) {
  const { descriptor } = parse(source)
  const script = compileScript(descriptor, { id: 'branch-ui-test' })
  const template = compileTemplate({
    source: descriptor.template.content,
    filename: 'BranchUiTest.vue',
    id: 'branch-ui-test',
    compilerOptions: { bindingMetadata: script.bindings },
  })
  let code = script.content.replace('export default ', 'const component = ') + '\n' +
    template.code.replace('export function render', 'function render')
  const bindings = {}
  const imports = babelParse(code, { sourceType: 'module' }).program.body
    .filter((node) => node.type === 'ImportDeclaration')
  for (const node of imports.reverse()) {
    for (const specifier of node.specifiers) {
      const dependency = dependencies[node.source.value]
      bindings[specifier.local.name] = specifier.type === 'ImportDefaultSpecifier'
        ? dependency : dependency[specifier.imported.name]
    }
    code = code.slice(0, node.start) + code.slice(node.end)
  }
  return new Function(...Object.keys(bindings), code + '\ncomponent.render = render; return component')(...Object.values(bindings))
}

function mountSwitcher(native, running = false) {
  const dependencies = {
    vue,
    'vue-i18n': { useI18n },
    '../gitBranches.js': gitBranches,
    '../../wailsjs/go/main/App': native,
    'lucide-vue-next': icons,
    './ui/AppPill.vue': compileComponent(read('./components/ui/AppPill.vue'), { vue }),
    './ui/AppIcon.vue': compileComponent(read('./components/ui/AppIcon.vue'), { vue }),
  }
  const component = compileComponent(switcher, dependencies)
  const node = (type, text = '') => Object.assign(new EventTarget(), {
    type, text, props: {}, children: [], parent: null, focus() {},
    getRootNode: () => globalThis.document,
  })
  const detach = (child) => {
    if (child.parent) child.parent.children.splice(child.parent.children.indexOf(child), 1)
    child.parent = null
  }
  const renderer = vue.createRenderer({
    createElement: (type) => node(type), createComment: (text) => node('#comment', text), createText: (text) => node('#text', text),
    setText: (element, text) => { element.text = text },
    setElementText: (element, text) => { element.text = text; element.children = [] },
    patchProp: (element, key, previous, next) => { element.props[key] = next },
    insert: (child, parent, anchor) => {
      detach(child)
      const index = anchor ? parent.children.indexOf(anchor) : -1
      parent.children.splice(index < 0 ? parent.children.length : index, 0, child)
      child.parent = parent
    },
    remove: detach,
    parentNode: (element) => element.parent,
    nextSibling: (element) => element.parent?.children[element.parent.children.indexOf(element) + 1] || null,
  })
  const i18n = createI18n({ legacy: false, locale: 'en', messages: { en, 'zh-CN': zhCN } })
  const app = renderer.createApp(component, { project: { id: 'project-raw' }, running })
  app.use(i18n)
  const container = node('root')
  const originalDocument = globalThis.document
  const originalDocumentClass = globalThis.Document
  globalThis.Document = class extends EventTarget {}
  globalThis.document = new Document()
  const instance = app.mount(container)
  const all = (element = container) => [element, ...element.children.flatMap((child) => all(child))]
  const text = (element) => element.type === '#comment' ? '' : element.text + element.children.map(text).join('')
  const find = (key, value) => all().find((element) => element.props[key] === value)
  return {
    state: instance.$.setupState,
    i18n,
    visibleText: () => text(container),
    visibleError: async () => {
      // Drain the native Promise rejection and both resulting Vue render jobs.
      await new Promise((resolve) => setImmediate(resolve))
      await vue.nextTick()
      const element = find('data-test', 'branch-error')
      assert.ok(element, 'the error must be rendered in the branch menu')
      return text(element)
    },
    unmount: () => {
      try { app.unmount() } finally {
        if (originalDocument === undefined) delete globalThis.document
        else globalThis.document = originalDocument
        if (originalDocumentClass === undefined) delete globalThis.Document
        else globalThis.Document = originalDocumentClass
      }
    },
  }
}

test('branch read and checkout errors preserve native details and translate only absent details', async () => {
  let rejection
  const calls = []
  const native = {
    ListProjectBranches: async (projectId) => {
      calls.push(projectId)
      throw rejection
    },
    CheckoutProjectBranch: async (projectId, name) => {
      assert.equal(projectId, 'project-raw')
      assert.equal(name, 'feature/原始')
      throw rejection
    },
    CreateProjectBranch: async () => { throw rejection },
  }
  const { state, i18n, visibleError, unmount } = mountSwitcher(native)
  try {
    state.open = true
    const raw = '  fatal: 分支 feature/原始\nYour local changes would be overwritten.\n'
    for (const detail of [raw, new Error(raw), { message: raw }]) {
      rejection = detail
      await state.load()
      assert.equal(await visibleError(), raw)
      state.checkout('feature/原始')
      assert.equal(await visibleError(), raw)
      i18n.global.locale.value = 'zh-CN'
      assert.equal(await visibleError(), raw)
      i18n.global.locale.value = 'en'
    }
    for (const detail of [undefined, null, '', new Error(''), {}]) {
      rejection = detail
      await state.load()
      assert.equal(await visibleError(), 'Failed to read branches')
      i18n.global.locale.value = 'zh-CN'
      assert.equal(await visibleError(), '读取分支失败')
      i18n.global.locale.value = 'en'
      state.checkout('feature/原始')
      assert.equal(await visibleError(), 'Failed to switch branch')
      i18n.global.locale.value = 'zh-CN'
      assert.equal(await visibleError(), '切换分支失败')
      i18n.global.locale.value = 'en'
    }
    assert.ok(calls.every((id) => id === 'project-raw'))
  } finally { unmount() }
})

test('branch group, warning, and validation copy follows locale while branch names stay raw', async () => {
  const native = {
    ListProjectBranches: async () => ({
      repo: true, branch: 'feature/原始', detached: false, dirty: true, head: 'abc1234',
      local: ['feature/原始'], remote: ['remote/分支'],
    }),
    CheckoutProjectBranch: async () => { throw new Error('unexpected checkout') },
    CreateProjectBranch: async () => { throw new Error('unexpected create') },
  }
  const { state, i18n, visibleText, unmount } = mountSwitcher(native, true)
  try {
    await state.toggleOpen()
    assert.match(visibleText(), /Local branches/)
    assert.match(visibleText(), /Remote branches/)
    assert.match(visibleText(), /Switching branches will not restart its processes/)
    assert.match(visibleText(), /feature\/原始/)
    assert.match(visibleText(), /remote\/分支/)
    state.query = 'bad name'
    await vue.nextTick()
    assert.match(visibleText(), /Branch name cannot contain spaces/)
    i18n.global.locale.value = 'zh-CN'
    await vue.nextTick()
    assert.match(visibleText(), /分支名不能包含空格/)
    assert.match(visibleText(), /项目正在运行/)
    state.query = ''
    await vue.nextTick()
    assert.match(visibleText(), /本地分支/)
    assert.match(visibleText(), /远端分支/)
    assert.match(visibleText(), /feature\/原始/)
    assert.match(visibleText(), /remote\/分支/)
  } finally { unmount() }
})

test('脏工作区与运行中项目会给出提醒', () => {
  assert.match(switcher, /checkoutWarnings/)
  assert.match(switcher, /branch-warnings/)
})

test('搜索框里输入新名字可以直接建分支', () => {
  assert.match(switcher, /data-test="branch-create"/)
  assert.match(switcher, /canCreateFromQuery/)
  assert.match(switcher, /validateBranchName/)
})

test('ProjectDetail 用切换器替掉了只读的分支 pill', () => {
  assert.match(projectDetail, /<BranchSwitcher\s+:project="project"\s+:running="state === 'running'"/)
  assert.doesNotMatch(projectDetail, /GetProjectBranch/, '分支读取已经归切换器管')
  assert.doesNotMatch(projectDetail, /refreshBranch/)
})
