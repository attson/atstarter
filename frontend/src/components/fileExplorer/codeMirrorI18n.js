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
    'Selection deleted': t('files.editorSearch.selectionDeleted'),
    'current match': t('files.editorSearch.currentMatch'),
    'on line': t('files.editorSearch.onLine'),
    'replaced match on line $': t('files.editorSearch.replacedMatch'),
    'replaced $ matches': t('files.editorSearch.replacedMatches'),
  }
}

// EditorView redraws every panel when its phrases facet changes. Preserve the
// current controls by panel/control order, including uncommitted input and focus
// outside the editor. The phrase transaction retains document/history state.
export function reconfigureEditorPhrases(view, effect) {
  const active = view.dom.ownerDocument.activeElement
  const panels = Array.from(view.dom.querySelectorAll('.cm-panel'))
  const controls = (panel) => Array.from(panel.querySelectorAll('input, textarea, select, button, [tabindex]'))
  const snapshots = panels.map((panel) => controls(panel).map((control) => ({
    value: control.value, checked: control.checked, focused: control === active,
    start: control.selectionStart, end: control.selectionEnd, direction: control.selectionDirection,
  })))
  const external = panels.some((panel) => panel.contains(active)) ? null : active
  view.dispatch({ effects: effect })
  const nextPanels = Array.from(view.dom.querySelectorAll('.cm-panel'))
  nextPanels.forEach((panel, index) => {
    // gotoLine passes opening-time translated strings to showDialog. Unlike
    // search controls, these two labels are captured in the panel constructor.
    const line = panel.querySelector('input[name="line"]')
    if (line && panel.classList.contains('cm-dialog')) {
      const label = Array.from(line.parentElement.childNodes).find((node) => node.nodeType === 3)
      if (label) label.nodeValue = `${view.state.phrase('Go to line')}: `
      const submit = panel.querySelector('button[type="submit"]')
      if (submit) submit.textContent = view.state.phrase('go')
    }
    const nextControls = controls(panel)
    for (const [controlIndex, field] of (snapshots[index] || []).entries()) {
      const control = nextControls[controlIndex]
      if (!control) continue
      if (field.value !== undefined) control.value = field.value
      if (field.checked !== undefined) control.checked = field.checked
      if (field.focused) control.focus({ preventScroll: true })
      if (typeof field.start === 'number' && typeof field.end === 'number') {
        control.setSelectionRange(field.start, field.end, field.direction)
      }
    }
  })
  if (external?.isConnected) external.focus({ preventScroll: true })
}
