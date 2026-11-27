import assert from 'node:assert/strict'
import test from 'node:test'
import { Compartment, EditorState } from '@codemirror/state'
import { SearchQuery, getSearchQuery, search, setSearchQuery } from '@codemirror/search'
import { createI18n } from 'vue-i18n'
import en from '../../i18n/messages/en.js'
import zhCN from '../../i18n/messages/zh-CN.js'

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
    state, dom: { ownerDocument: document, querySelector: () => currentPanel },
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
