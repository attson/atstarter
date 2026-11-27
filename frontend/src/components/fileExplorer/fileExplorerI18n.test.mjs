import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'
import * as vue from 'vue'
import { createI18n, useI18n } from 'vue-i18n'
import * as icons from 'lucide-vue-next'
import * as tabs from './editorTabs.js'
import * as prefs from './editorPrefs.js'
import * as paths from './pathOps.js'
import * as clipboard from './fileClipboard.js'
import * as keyboard from './treeKeyboard.js'
import * as menuPlacement from './menuPlacement.js'
import * as cmState from '@codemirror/state'
import * as cmView from '@codemirror/view'
import * as cmCommands from '@codemirror/commands'
import * as cmSearch from '@codemirror/search'
import * as codeMirrorI18n from './codeMirrorI18n.js'
import en from '../../i18n/messages/en.js'
import zhCN from '../../i18n/messages/zh-CN.js'
import { compileComponent, mountVueComponent } from '../../testUtils/vueComponentHarness.mjs'

const read = (file) => readFileSync(new URL(file, import.meta.url), 'utf8')
const flush = async () => { await new Promise((resolve) => setImmediate(resolve)); await vue.nextTick() }
const noopComponent = { render: () => null }

function mountFileComponent(file, { props = {}, bridge = {}, dependencies: extra = {} } = {}) {
  const globals = new Map(['window', 'MutationObserver', 'getComputedStyle'].map((key) => [key, Object.getOwnPropertyDescriptor(globalThis, key)]))
  globalThis.window = new EventTarget()
  globalThis.MutationObserver = class { observe() {} disconnect() {} }
  globalThis.getComputedStyle = () => ({ getPropertyValue: () => '' })
  let mounts = 0
  let unmounts = 0
  let saves = 0
  let saveResult = false
  const editor = {
    props: ['path', 'fs', 'prefs', 'theme', 'viewMode'],
    setup(props, { expose }) {
      vue.onMounted(() => { mounts++ })
      vue.onBeforeUnmount(() => { unmounts++ })
      expose({ save: async () => { saves++; return saveResult }, focus() {} })
      return () => vue.h('div', { 'data-editor': props.path })
    },
  }
  const fs = { projectId: 'raw-project', listDir: async () => [], watchDir: async () => 1,
    unwatchDir: async () => {}, onDirChanged: () => () => {}, ...bridge }
  const dependencies = {
    vue, 'vue-i18n': { useI18n }, 'lucide-vue-next': icons,
    '../composables/useTheme': { useTheme: () => ({ resolvedTheme: vue.ref('dark') }) },
    './fileExplorer/fsBridge': { createProjectFSBridge: () => fs },
    './fileExplorer/editorTabs.js': tabs, './fileExplorer/editorPrefs.js': prefs, './editorPrefs.js': prefs,
    './fileExplorer/pathOps.js': paths, './pathOps.js': paths,
    './fileClipboard.js': clipboard, './treeKeyboard.js': keyboard, './menuPlacement.js': menuPlacement,
    './fileExplorer/FileTree.vue': noopComponent, './fileExplorer/FileEditor.vue': editor,
    './FileTreeNode.vue': noopComponent,
    './previewKind': { previewKind: () => 'code' },
    '@codemirror/state': cmState, '@codemirror/view': cmView,
    '@codemirror/commands': cmCommands, '@codemirror/search': cmSearch,
    './languageMap': { languageForPath: async () => null },
    './highlight': { highlightExtensionFor: () => [] },
    './codeMirrorI18n.js': codeMirrorI18n,
    ...extra,
  }
  for (const name of ['FileConfirmDialog', 'EditorTabs', 'EditorToolbar', 'EditorStatusBar', 'EditorPrefsPanel', 'BinaryBanner', 'InlineEditRow']) {
    const component = compileComponent(read(`./${name}.vue`), dependencies)
    dependencies[`./${name}.vue`] = component
    dependencies[`./fileExplorer/${name}.vue`] = component
  }
  const i18n = createI18n({ legacy: false, locale: 'en', messages: { en, 'zh-CN': zhCN } })
  let host
  function restore() {
    for (const [key, descriptor] of globals) {
      if (descriptor) Object.defineProperty(globalThis, key, descriptor)
      else delete globalThis[key]
    }
  }
  try {
    host = mountVueComponent(compileComponent(read(file), dependencies), {
      props, i18n, beforeMount({ document }) { document.documentElement = { getAttribute: () => 'dark' } },
    })
  } catch (error) { restore(); throw error }
  return {
    ...host, i18n, fs,
    counts: () => ({ mounts, unmounts, saves }),
    setSaveResult: (value) => { saveResult = value },
    unmount() { try { host.unmount() } finally { restore() } },
  }
}

test('open tab close guard follows locale and preserves raw filenames, actions, and editor lifetime', async () => {
  const host = mountFileComponent('../FileBrowser.vue', { props: { projectId: 'raw-project' } })
  const path = 'dir/原始 @ {name}|raw.md'
  const button = (id) => host.find((element) => element.props['data-test'] === `btn-${id}`)
  try {
    host.state.openPath(path)
    host.state.openPath('other.txt')
    await vue.nextTick()
    host.state.onDirtyChange(path, true)
    host.state.requestClose(path)
    await vue.nextTick()
    assert.ok(host.visibleText().includes('“原始 @ {name}|raw.md” has unsaved changes'))
    assert.equal(button('save').text, 'Save and close')
    assert.equal(button('discard').text, 'Discard')
    host.i18n.global.locale.value = 'zh-CN'
    await vue.nextTick()
    assert.ok(host.visibleText().includes('「原始 @ {name}|raw.md」有未保存的改动'))
    assert.equal(button('save').text, '保存并关闭')
    button('cancel').props.onClick()
    await vue.nextTick()
    assert.equal(host.state.closeGuard, null)
    assert.equal(host.state.tabs.tabs.length, 2)
    host.state.activate(path)
    host.i18n.global.locale.value = 'en'
    await vue.nextTick()
    assert.deepEqual(host.counts(), { mounts: 2, unmounts: 0, saves: 0 })
    host.state.requestClose(path)
    await vue.nextTick()
    button('save').props.onClick()
    await flush()
    assert.equal(host.state.tabs.tabs.length, 2, 'failed save keeps the editor open')
    host.setSaveResult(true)
    host.state.requestClose(path)
    await vue.nextTick()
    button('save').props.onClick()
    await flush()
    assert.equal(host.state.tabs.tabs.length, 1)
    assert.deepEqual(host.counts(), { mounts: 2, unmounts: 1, saves: 2 })
    host.state.onDirtyChange('other.txt', true)
    host.state.requestClose('other.txt')
    await vue.nextTick()
    button('discard').props.onClick()
    await vue.nextTick()
    assert.equal(host.state.tabs.tabs.length, 0)
    assert.deepEqual(host.counts(), { mounts: 2, unmounts: 2, saves: 2 })
  } finally { host.unmount() }
})

test('editor metadata failures preserve native detail and render a reactive fallback when absent', async () => {
  const raw = '  EACCES: /原始 @ {name}|raw\n native detail\n'
  let rejection
  for (const component of ['CodeEditor', 'FileEditor']) {
    const host = mountFileComponent(`./${component}.vue`, { props: {
      path: '原始 @ {name}|raw.ts', theme: 'light', viewMode: 'code',
      prefs: { lineNumbers: true, wrap: false, tabSize: 4, fontSize: 13 },
      fs: { fileMeta: async () => { throw rejection }, onDirChanged: () => () => {} },
    } })
    try {
      for (const detail of [raw, new Error(raw), { message: raw }, undefined, null, {}, new Error('')]) {
        rejection = detail
        host.i18n.global.locale.value = 'en'
        await host.state[component === 'CodeEditor' ? 'load' : 'resolveKind']()
        await vue.nextTick()
        const hasDetail = detail === raw || detail?.message === raw
        assert.ok(host.visibleText().includes(hasDetail ? raw : 'Unable to load file'))
        host.i18n.global.locale.value = 'zh-CN'
        await vue.nextTick()
        assert.ok(host.visibleText().includes(hasDetail ? raw : '无法加载文件'))
      }
    } finally { host.unmount() }
  }
})

test('editor language loading uses the latest locale without reading or rewriting the document again', async () => {
  let resolveLanguage
  const language = new Promise((resolve) => { resolveLanguage = resolve })
  let reads = 0
  let editor
  const raw = '原始 @ {name}|raw\n\tconst answer = 42;\n'
  class EditorViewBoundary extends cmView.EditorView {
    constructor({ state }) {
      // Browser rendering is the boundary; real CodeMirror state/extensions run.
      editor = { state, destroy() {}, focus() {} }
      return editor
    }
  }
  const host = mountFileComponent('./CodeEditor.vue', {
    props: { path: 'raw.ts', theme: 'light', prefs: { lineNumbers: true, wrap: false, tabSize: 4, fontSize: 13 }, fs: {
      fileMeta: async () => ({ size: 42, modTime: 7, isBinary: false }),
      readBytes: async () => { reads++; return { data: new TextEncoder().encode(raw) } },
      onDirChanged: () => () => {},
    } },
    dependencies: { '@codemirror/view': { ...cmView, EditorView: EditorViewBoundary }, './languageMap': { languageForPath: () => language } },
  })
  try {
    await flush()
    host.i18n.global.locale.value = 'zh-CN'
    await vue.nextTick()
    resolveLanguage(null)
    await flush()
    assert.equal(editor.state.phrase('Find'), '查找')
    assert.equal(editor.state.doc.toString(), raw)
    assert.equal(reads, 1)
  } finally { host.unmount() }
})

test('save errors preserve native details and use a reactive fallback without changing document bytes', async () => {
  let rejection
  const raw = '  write denied: /原始 @ {name}|raw\n'
  const content = '原始 @ {name}|raw\n\tconst answer = 42;\n'
  const writes = []
  class EditorViewBoundary extends cmView.EditorView {
    constructor({ state }) {
      return { state, dom: { querySelector: () => null, ownerDocument: { activeElement: null } },
        dispatch(transaction) { this.state = this.state.update(transaction).state }, destroy() {}, focus() {} }
    }
  }
  const host = mountFileComponent('./CodeEditor.vue', {
    props: { path: '原始 @ {name}|raw.ts', theme: 'light', prefs: { lineNumbers: true, wrap: false, tabSize: 4, fontSize: 13 }, fs: {
      fileMeta: async () => ({ size: 42, modTime: 7, isBinary: false }),
      readBytes: async () => ({ data: new TextEncoder().encode(content) }),
      writeBytes: async (...args) => { writes.push(args); throw rejection }, onDirChanged: () => () => {},
    } },
    dependencies: { '@codemirror/view': { ...cmView, EditorView: EditorViewBoundary } },
  })
  try {
    await flush()
    for (const detail of [raw, new Error(raw), { message: raw }, undefined, null, {}, new Error('')]) {
      rejection = detail
      host.i18n.global.locale.value = 'en'
      await vue.nextTick()
      assert.equal(await host.state.save(), false)
      await vue.nextTick()
      const hasDetail = detail === raw || detail?.message === raw
      const banner = host.find((element) => element.props['data-test'] === 'save-error')
      assert.ok(banner, 'a failed save must display its error even when native detail is absent')
      assert.equal(banner.text,
        hasDetail ? `Save failed: ${raw}` : 'Save failed: Unable to save file')
      host.i18n.global.locale.value = 'zh-CN'
      await vue.nextTick()
      assert.equal(host.find((element) => element.props['data-test'] === 'save-error').text,
        hasDetail ? `保存失败:${raw}` : '保存失败:无法保存文件')
    }
    for (const [path, bytes, expectedModTime, createIfMissing] of writes) {
      assert.equal(path, '原始 @ {name}|raw.ts')
      assert.deepEqual(bytes, new TextEncoder().encode(content))
      assert.equal(expectedModTime, 7)
      assert.equal(createIfMissing, false)
    }
  } finally { host.unmount() }
})

test('asset preview failures preserve native detail and absent details retain localized binary fallback', async () => {
  const raw = '  Native asset error: /原始 @ {name}|raw\n'
  for (const component of ['ImagePreview', 'MediaPreview', 'PdfPreview']) {
    for (const detail of [new Error(raw), raw, undefined]) {
      const path = '原始 @ {name}|raw.png'
      const revoked = []
      const host = mountFileComponent(`./${component}.vue`, { props: {
        path, kind: 'audio', theme: 'light',
        fs: { identity: 'raw-project', assetUrlFor: async () => { throw detail }, revokeAssetUrl: (value) => revoked.push(value) },
      } })
      try {
        await flush()
        assert.ok(host.visibleText().includes(detail ? raw : 'Preview is not supported for this file type.'))
        host.i18n.global.locale.value = 'zh-CN'
        await vue.nextTick()
        assert.ok(host.visibleText().includes(detail ? raw : '不支持预览此文件类型。'))
      } finally { host.unmount() }
      assert.deepEqual(revoked, [path], 'unmount revokes the original asset path exactly once')
    }
  }
})

test('browser error fallbacks stay reactive while native detail and search data stay verbatim', async () => {
  let rejection
  const raw = '  EACCES: /原始 @ {name}|raw\n native detail\n'
  const query = '  原始 @ {name}|raw  '
  const requested = []
  const host = mountFileComponent('../FileBrowser.vue', {
    props: { projectId: 'raw-project' },
    bridge: {
      reveal: async (path) => { requested.push(path); throw rejection },
      searchPaths: async (value, limit) => { requested.push([value, limit]); throw rejection },
    },
  })
  try {
    host.state.openPath('原始 @ {name}|raw.txt')
    for (const detail of [new Error(raw), { message: raw }, raw, undefined, null, {}, new Error('')]) {
      rejection = detail
      host.i18n.global.locale.value = 'en'
      await host.state.revealActive()
      await vue.nextTick()
      const hasDetail = detail === raw || detail?.message === raw
      assert.equal(host.find((element) => element.props['data-test'] === 'file-browser-error').text,
        hasDetail ? raw : 'Unable to show in file manager')
      host.i18n.global.locale.value = 'zh-CN'
      await vue.nextTick()
      assert.equal(host.find((element) => element.props['data-test'] === 'file-browser-error').text,
        hasDetail ? raw : '无法在文件管理器中显示')
      host.state.searchQuery = query
      await vue.nextTick()
      await host.state.runSearch(host.state.fs, query.trim(), host.state.searchGeneration)
      await vue.nextTick()
      assert.equal(host.find((element) => element.props.class === 'search-state search-error').text,
        hasDetail ? raw : '搜索失败')
      host.i18n.global.locale.value = 'en'
      await vue.nextTick()
      assert.equal(host.find((element) => element.props.class === 'search-state search-error').text,
        hasDetail ? raw : 'Search failed')
      assert.equal(host.state.searchQuery, query)
    }
    assert.ok(requested.every((value) => Array.isArray(value)
      ? value[0] === query.trim() && value[1] === 100 : value === '原始 @ {name}|raw.txt'))
  } finally { host.unmount() }
})

test('rendered editor tooltips and status follow locale while paths and encoding remain raw', async () => {
  const path = '原始 @ {name}|raw.md'
  const host = mountFileComponent('../FileBrowser.vue', { props: { projectId: 'raw-project' } })
  const find = (id) => host.find((element) => element.props['data-test'] === id)
  try {
    host.state.openPath(path)
    host.state.onKindChange(path, 'markdown')
    host.state.onDirtyChange(path, true)
    host.state.onCursorChange(path, { line: 12, column: 3, selected: 7, lines: 40 })
    await vue.nextTick()
    assert.equal(find('toolbar-save').props.title, 'Save (Ctrl/Cmd+S)')
    host.state.toggleViewMode()
    await vue.nextTick()
    assert.equal(find('toolbar-toggle-view').props.title, 'View source')
    host.state.toggleViewMode()
    await vue.nextTick()
    assert.equal(find('toolbar-toggle-view').props.title, 'Preview rendered content')
    assert.match(host.visibleText(), /7 characters selected.*Ln 12:3 \/ 40.*UTF-8/)
    assert.equal(host.state.kindLabel, 'Markdown')
    host.i18n.global.locale.value = 'zh-CN'
    await vue.nextTick()
    assert.equal(find('toolbar-save').props.title, '保存 (Ctrl/Cmd+S)')
    assert.equal(find('toolbar-toggle-view').props.title, '预览渲染结果')
    assert.match(host.visibleText(), /已选 7 字符.*行 12:3 \/ 40.*UTF-8/)
    assert.ok(host.visibleText().includes(path))
    assert.equal(host.state.tabs.tabs[0].path, path)
    host.state.onKindChange(path, 'image')
    await vue.nextTick()
    assert.equal(host.state.kindLabel, '图片')
    host.i18n.global.locale.value = 'en'
    await vue.nextTick()
    assert.equal(host.state.kindLabel, 'Image')
    assert.equal(find('tree-new-file').props.title, 'New file in project root')
  } finally { host.unmount() }
})

test('tree confirmations and operation errors translate at render time with original names and details', async () => {
  const raw = '  native failure: /原始 @ {name}|raw\n'
  const name = '原始 @ {name}|raw'
  const host = mountFileComponent('./FileTree.vue', { props: {
    fs: { projectId: 'raw', listDir: async () => [], onDirChanged: () => () => {}, watchDir: async () => 1, unwatchDir: async () => {} },
    root: '', showHidden: false,
  } })
  try {
    await flush()
    host.state.requestDelete([{ path: `dir/${name}`, isDir: true }], false)
    await vue.nextTick()
    assert.ok(host.visibleText().includes(`Move “${name}” to Trash?`))
    host.i18n.global.locale.value = 'zh-CN'
    await vue.nextTick()
    assert.ok(host.visibleText().includes(`确定把「${name}」移到废纸篓?`))
    await host.state.runFsOp(async () => { throw new Error(raw) })
    await vue.nextTick()
    assert.equal(host.find((element) => element.props['data-test'] === 'file-tree-error').text, raw)
    await host.state.runFsOp(async () => { throw {} })
    await vue.nextTick()
    assert.equal(host.find((element) => element.props['data-test'] === 'file-tree-error').text, '操作失败')
    host.i18n.global.locale.value = 'en'
    await vue.nextTick()
    assert.equal(host.find((element) => element.props['data-test'] === 'file-tree-error').text, 'Operation failed')
    clipboard.fileClipboard.copy('raw', [{ path: `dir/${name}`, isDir: true }])
    await host.state.doPaste(`dir/${name}/child`)
    await vue.nextTick()
    assert.ok(host.visibleText().includes(`Cannot paste “${name}” inside itself`))
    host.i18n.global.locale.value = 'zh-CN'
    await vue.nextTick()
    assert.ok(host.visibleText().includes(`不能把「${name}」粘贴到它自己内部`))
  } finally { clipboard.fileClipboard.clear(); host.unmount() }
})
