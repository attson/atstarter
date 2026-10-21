import assert from 'node:assert/strict'
import { test } from 'node:test'

import { logLinesForRun, terminalChunk } from './logBatches.js'

test('logLinesForRun keeps only the current RunID in receive order', () => {
  const batch = { lines: [
    { id: 'one', stream: 'stdout', text: 'first' },
    { id: 'two', stream: 'stderr', text: 'ignored' },
    { id: 'one', stream: 'stderr', text: '\u001b[31mthird\u001b[0m' },
  ] }

  assert.deepEqual(logLinesForRun(batch, 'one').map((line) => line.text), [
    'first',
    '\u001b[31mthird\u001b[0m',
  ])
})

test('terminalChunk preserves ANSI text and writeln newline semantics in one string', () => {
  const chunk = terminalChunk([
    { text: 'first' },
    { text: '\u001b[31msecond\u001b[0m' },
  ])

  assert.equal(chunk, 'first\r\n\u001b[31msecond\u001b[0m\r\n')
})
