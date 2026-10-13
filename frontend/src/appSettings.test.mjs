import test from 'node:test'
import assert from 'node:assert/strict'
import { normalizeAppSettings, terminalFontFamily } from './appSettings.js'

test('normalizeAppSettings supplies stable defaults', () => {
  assert.deepEqual(normalizeAppSettings(), {
    shell: '', autoShell: '', availableShells: [], logFontFamily: 'monospace', logFontSize: 11, logLetterSpacing: 0,
  })
})

test('normalizeAppSettings trims values, deduplicates shells, and clamps numbers', () => {
  assert.deepEqual(normalizeAppSettings({
    shell: ' /usr/bin/zsh ', autoShell: ' /usr/bin/zsh ',
    availableShells: ['/usr/bin/zsh', ' /usr/bin/zsh ', '', null],
    logFontFamily: ' JetBrains Mono ', logFontSize: 99, logLetterSpacing: -2,
  }), {
    shell: '/usr/bin/zsh', autoShell: '/usr/bin/zsh', availableShells: ['/usr/bin/zsh'],
    logFontFamily: 'JetBrains Mono', logFontSize: 20, logLetterSpacing: 0,
  })
})

test('terminalFontFamily keeps a guaranteed monospace fallback', () => {
  assert.equal(terminalFontFamily('monospace'), 'monospace')
  assert.equal(terminalFontFamily('JetBrains Mono'), '"JetBrains Mono", monospace')
  assert.equal(terminalFontFamily('A"B'), '"A\\"B", monospace')
})
