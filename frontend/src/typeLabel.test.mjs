import { test } from 'node:test'
import assert from 'node:assert/strict'
import { typeLabel } from './typeLabel.js'

test('strips node- prefix', () => {
  assert.equal(typeLabel('node-pnpm'), 'pnpm')
  assert.equal(typeLabel('node-npm'), 'npm')
})

test('strips java- prefix', () => {
  assert.equal(typeLabel('java-maven'), 'maven')
  assert.equal(typeLabel('java-gradle'), 'gradle')
})

test('leaves other types unchanged', () => {
  assert.equal(typeLabel('go'), 'go')
  assert.equal(typeLabel('rust'), 'rust')
  assert.equal(typeLabel('python-django'), 'python-django')
  assert.equal(typeLabel('compose'), 'compose')
})

test('falls back to unknown for empty', () => {
  assert.equal(typeLabel(''), 'unknown')
  assert.equal(typeLabel(undefined), 'unknown')
})
