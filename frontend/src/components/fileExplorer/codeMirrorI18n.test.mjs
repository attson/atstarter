import assert from 'node:assert/strict'
import test from 'node:test'
import { Compartment, EditorState } from '@codemirror/state'
import { SearchQuery, getSearchQuery, search, setSearchQuery } from '@codemirror/search'
import { createI18n } from 'vue-i18n'
import en from '../../i18n/messages/en.js'
import zhCN from '../../i18n/messages/zh-CN.js'

function panelRedrawBoundary(kind, { externalFocus = false, connected = true } = {}) {
  const document = { activeElement: null }
  const locale = new Compartment()
  const i18n = createI18n({ legacy: false, locale: 'en', messages: { en, 'zh-CN': zhCN } })
  const control = (name, value = '') => ({
    name, value, checked: false, selectionStart: 1, selectionEnd: 4, selectionDirection: 'backward', isConnected: true,
    getAttribute(key) { return this[key] },
    focus() { document.activeElement = this },
    setSelectionRange(start, end, direction) { Object.assign(this, { selectionStart: start, selectionEnd: end, selectionDirection: direction }) },
  })
  const input = (value) => control(kind === 'search' ? 'search' : 'line', value)
  const panel = (value) => {
    const field = input(value)
    const submit = control('')
    submit.textContent = 'go'
    const labelText = { nodeType: 3, nodeValue: 'Go to line: ' }
    field.parentElement = { childNodes: [labelText, field] }
    return {
      field, submit, labelText,
      classList: { contains: (name) => name === (kind === 'search' ? 'cm-search' : 'cm-dialog') },
      contains: (element) => element === field || element === submit,
      querySelectorAll: (selector) => selector.includes('button') ? [field, submit] : [field],
      querySelector(selector) {
        if (selector.includes('button')) return submit
        return selector.includes(`name="${field.name}"`) ? field : null
      },
    }
  }
  let currentPanel = panel(kind === 'search' ? '@ {name}|原始' : '+12:3')
  const originalInput = currentPanel.field
  const outside = control('outside', '原始 @ {name}|external')
  outside.isConnected = connected
  document.activeElement = externalFocus ? outside : originalInput
  const view = {
    state: EditorState.create({ doc: '原始 @ {name}|raw\n'.repeat(20), extensions: [locale.of(EditorState.phrases.of({ 'Go to line': 'Go to line', go: 'go' }))] }),
    dom: { ownerDocument: document,
      querySelector: (selector) => kind === 'search' && selector === '.cm-search' ? currentPanel : null,
      querySelectorAll: () => [currentPanel],
    },
    dispatch(transaction) {
      this.state = this.state.update(transaction).state
      // CodeMirror's panel constructors select the first input when redrawn.
      // The built-in line dialog also reuses its captured opening-time label.
      currentPanel = panel(kind === 'search' ? '' : '1')
      document.activeElement = currentPanel.field
    },
  }
  return { view, locale, i18n, document, outside, originalInput, panel: () => currentPanel }
}

test('phrase changes restore a connected external control after an open search panel redraw', async () => {
  const { editorPhrases, reconfigureEditorPhrases } = await import('./codeMirrorI18n.js')
  const host = panelRedrawBoundary('search', { externalFocus: true })
  host.i18n.global.locale.value = 'zh-CN'
  reconfigureEditorPhrases(host.view, host.locale.reconfigure(EditorState.phrases.of(editorPhrases(host.i18n.global.t))))
  assert.equal(host.document.activeElement, host.outside, 'an open search panel must not steal focus from an external control')
  assert.equal(host.outside.value, '原始 @ {name}|external')
  assert.deepEqual([host.outside.selectionStart, host.outside.selectionEnd, host.outside.selectionDirection], [1, 4, 'backward'])
})

test('phrase changes preserve Go to line input, focus and selection while refreshing its captured labels', async () => {
  const { editorPhrases, reconfigureEditorPhrases } = await import('./codeMirrorI18n.js')
  const host = panelRedrawBoundary('line')
  const originalDoc = host.view.state.doc
  for (const [locale, label, submit] of [['zh-CN', '跳转到行: ', '跳转'], ['en', 'Go to line: ', 'go']]) {
    host.i18n.global.locale.value = locale
    reconfigureEditorPhrases(host.view, host.locale.reconfigure(EditorState.phrases.of(editorPhrases(host.i18n.global.t))))
    const panel = host.panel()
    assert.equal(panel.field.value, '+12:3', 'the open line dialog must retain the typed position')
    assert.equal(host.document.activeElement, panel.field)
    assert.deepEqual([panel.field.selectionStart, panel.field.selectionEnd, panel.field.selectionDirection], [1, 4, 'backward'])
    assert.equal(panel.labelText.nodeValue, label)
    assert.equal(panel.submit.textContent, submit)
    assert.equal(host.view.state.doc, originalDoc)
  }
})

test('CodeMirror phrase reconfiguration restores open search values, focus and selection after its panel redraw', async () => {
  const { editorPhrases, reconfigureEditorPhrases } = await import('./codeMirrorI18n.js').catch(() => ({}))
  assert.equal(typeof editorPhrases, 'function', 'editor phrases must be translated at render time')
  assert.equal(typeof reconfigureEditorPhrases, 'function', 'search focus must survive CodeMirror panel redraw')
  const i18n = createI18n({ legacy: false, locale: 'en', messages: { en, 'zh-CN': zhCN } })
  const locale = new Compartment()
  const raw = '原始 @ {name}|raw\n\tconst answer = 42;\n'
  const query = new SearchQuery({ search: '@ {name}|原始', replace: '替换 @ {name}|raw', caseSensitive: true, wholeWord: true })
  let state = EditorState.create({ doc: raw, selection: { anchor: 3 }, extensions: [search(), locale.of(EditorState.phrases.of(editorPhrases(i18n.global.t)))] })
  state = state.update({ effects: setSearchQuery.of(query) }).state
  const document = { activeElement: null }
  const input = (name, value = '', checked = false) => ({
    name, value, checked, selectionStart: name === 'search' || name === 'replace' ? 2 : null,
    selectionEnd: 4, selectionDirection: 'backward',
    getAttribute(key) { return this[key] },
    focus() { document.activeElement = this },
    setSelectionRange(start, end, direction) { Object.assign(this, { selectionStart: start, selectionEnd: end, selectionDirection: direction }) },
  })
  const fields = () => [input('search', query.search), input('replace', query.replace), input('case', '', true), input('re'), input('word', '', true)]
  const panel = (inputs) => ({
    contains: (element) => inputs.includes(element),
    querySelectorAll: () => inputs,
    querySelector: (selector) => inputs.find((field) => selector.includes(`name="${field.name}"`)),
  })
  const originalInputs = fields()
  originalInputs[1].value += ' uncommitted'
  let currentPanel = panel(originalInputs)
  document.activeElement = originalInputs[1]
  const view = {
    state, dom: { ownerDocument: document, querySelector: () => currentPanel, querySelectorAll: () => [currentPanel] },
    dispatch(transaction) {
      this.state = this.state.update(transaction).state
      // CodeMirror retains its state but recreates all panels on phrase changes.
      const recreated = fields()
      for (const field of recreated) { field.value = ''; field.checked = false; field.selectionStart = 0; field.selectionEnd = 0 }
      currentPanel = panel(recreated)
      document.activeElement = recreated[0]
    },
  }
  i18n.global.locale.value = 'zh-CN'
  reconfigureEditorPhrases(view, locale.reconfigure(EditorState.phrases.of(editorPhrases(i18n.global.t))))
  const replacement = currentPanel.querySelector('input[name="replace"]')
  assert.equal(view.state.phrase('Find'), '查找')
  assert.equal(view.state.phrase('replaced $ matches', 7), '已替换 7 处匹配')
  assert.equal(view.state.doc.toString(), raw)
  assert.equal(view.state.selection.main.anchor, 3)
  assert.equal(getSearchQuery(view.state), query)
  assert.equal(document.activeElement, replacement)
  assert.equal(replacement.value, '替换 @ {name}|raw uncommitted')
  assert.deepEqual([replacement.selectionStart, replacement.selectionEnd, replacement.selectionDirection], [2, 4, 'backward'])
  assert.equal(currentPanel.querySelector('input[name="search"]').value, query.search)
  assert.equal(currentPanel.querySelector('input[name="case"]').checked, true)
  assert.equal(currentPanel.querySelector('input[name="re"]').checked, false)
  assert.equal(currentPanel.querySelector('input[name="word"]').checked, true)
})
