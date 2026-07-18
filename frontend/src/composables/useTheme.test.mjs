import assert from 'node:assert/strict'
import { test } from 'node:test'
import { resolveTheme, nextTheme } from './useTheme.js'

test('resolveTheme: system + prefers dark → dark', () => {
  assert.equal(resolveTheme('system', true), 'dark')
})

test('resolveTheme: system + prefers light → light', () => {
  assert.equal(resolveTheme('system', false), 'light')
})

test('resolveTheme: explicit dark ignores system', () => {
  assert.equal(resolveTheme('dark', false), 'dark')
})

test('resolveTheme: explicit light ignores system', () => {
  assert.equal(resolveTheme('light', true), 'light')
})

test('resolveTheme: falls back to dark when input invalid', () => {
  assert.equal(resolveTheme('garbage', true), 'dark')
  assert.equal(resolveTheme(undefined, false), 'light')
})

test('nextTheme cycles system → dark → light → system', () => {
  assert.equal(nextTheme('system'), 'dark')
  assert.equal(nextTheme('dark'), 'light')
  assert.equal(nextTheme('light'), 'system')
})

test('nextTheme: unknown input starts a fresh cycle at dark', () => {
  assert.equal(nextTheme('unknown'), 'dark')
})
