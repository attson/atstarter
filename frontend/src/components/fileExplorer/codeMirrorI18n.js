// These English keys are CodeMirror's phrase identifiers, not display copy.
export function editorPhrases(t) {
  return {
    Find: t('files.editorSearch.find'),
    Replace: t('files.editorSearch.replace'),
    next: t('files.editorSearch.next'),
    previous: t('files.editorSearch.previous'),
    all: t('files.editorSearch.all'),
    'match case': t('files.editorSearch.matchCase'),
    regexp: t('files.editorSearch.regexp'),
    'by word': t('files.editorSearch.byWord'),
    replace: t('files.editorSearch.replaceAction'),
    'replace all': t('files.editorSearch.replaceAll'),
    close: t('files.close'),
    'Go to line': t('files.gotoLine'),
    go: t('files.editorSearch.go'),
    'Control character': t('files.editorSearch.controlCharacter'),
    'current match': t('files.editorSearch.currentMatch'),
    'on line': t('files.editorSearch.onLine'),
    'replaced match on line $': t('files.editorSearch.replacedMatch'),
    'replaced $ matches': t('files.editorSearch.replacedMatches'),
  }
}

// EditorView internally redraws panels when its phrases facet changes. Snapshot
// only the search panel's controls around the Compartment transaction so a live
// language change preserves even uncommitted field text, focus, and selection.
// CodeMirror retains the document, history, search state, and EditorView itself.
export function reconfigureEditorPhrases(view, effect) {
  const panel = view.dom.querySelector('.cm-search')
  const active = view.dom.ownerDocument.activeElement
  const focusedName = panel?.contains(active) ? active?.getAttribute('name') : null
  const fields = Array.from(panel?.querySelectorAll('input[name]') || []).map((input) => ({
    name: input.name, value: input.value, checked: input.checked,
    start: input.selectionStart, end: input.selectionEnd, direction: input.selectionDirection,
  }))
  view.dispatch({ effects: effect })
  const nextPanel = view.dom.querySelector('.cm-search')
  if (!nextPanel) return
  for (const field of fields) {
    const input = nextPanel.querySelector(`input[name="${field.name}"]`)
    if (!input) continue
    input.value = field.value
    input.checked = field.checked
  }
  if (focusedName) nextPanel.querySelector(`[name="${focusedName}"]`)?.focus({ preventScroll: true })
  for (const field of fields) {
    if (typeof field.start !== 'number' || typeof field.end !== 'number') continue
    nextPanel.querySelector(`input[name="${field.name}"]`)?.setSelectionRange(field.start, field.end, field.direction)
  }
}
