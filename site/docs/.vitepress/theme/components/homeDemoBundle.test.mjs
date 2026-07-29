import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const distAssetsDir = fileURLToPath(new URL('../../dist/assets/', import.meta.url))
const codeMirrorStateError = 'multiple instances of @codemirror/state'

function readHomePageChunks() {
  assert.ok(fs.existsSync(distAssetsDir), 'run npm run docs:build before this bundle test')
  const names = fs.readdirSync(distAssetsDir).filter((name) => /^index\.md\..*\.js$/.test(name))
  const fullChunks = names.filter((name) => !name.endsWith('.lean.js'))
  const leanChunks = names.filter((name) => name.endsWith('.lean.js'))
  assert.ok(fullChunks.length > 0, 'expected built homepage chunk')
  assert.ok(leanChunks.length > 0, 'expected built homepage lean chunk')
  return [...fullChunks, ...leanChunks].map((name) => ({
    name,
    source: fs.readFileSync(path.join(distAssetsDir, name), 'utf8'),
  }))
}

test('built homepage chunks do not inline CodeMirror state', () => {
  for (const chunk of readHomePageChunks()) {
    assert.equal(
      chunk.source.includes(codeMirrorStateError),
      false,
      `${chunk.name} should not inline CodeMirror state; keep CodeEditor in an async chunk`,
    )
  }
})
