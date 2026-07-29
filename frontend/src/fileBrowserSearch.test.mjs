import assert from 'node:assert/strict'
import fs from 'node:fs'
import test from 'node:test'

const fileBrowserSource = fs.readFileSync(new URL('./components/FileBrowser.vue', import.meta.url), 'utf8')
const fsBridgeSource = fs.readFileSync(new URL('./components/fileExplorer/fsBridge.ts', import.meta.url), 'utf8')
const codeEditorSource = fs.readFileSync(new URL('./components/fileExplorer/CodeEditor.vue', import.meta.url), 'utf8')

test('file browser exposes a project filename search input backed by fsBridge', () => {
  assert.match(fileBrowserSource, /searchQuery/)
  assert.match(fileBrowserSource, /searchPaths/)
  assert.match(fileBrowserSource, /data-test=["']file-search-input["']/)
  assert.match(fileBrowserSource, /data-test=["']file-search-results["']/)
})

test('fsBridge maps project filename search to the Wails binding', () => {
  assert.match(fsBridgeSource, /SearchProjectFiles/)
  assert.match(fsBridgeSource, /searchPaths\(query:\s*string,\s*limit:\s*number\)/)
})

test('code editor enables current-file search through CodeMirror search extension', () => {
  assert.match(codeEditorSource, /@codemirror\/search/)
  assert.match(codeEditorSource, /search\(/)
  assert.match(codeEditorSource, /searchKeymap/)
  assert.match(codeEditorSource, /highlightSelectionMatches/)
})
