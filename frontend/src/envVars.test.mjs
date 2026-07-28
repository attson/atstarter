import test from 'node:test'
import assert from 'node:assert/strict'
import { envMapToText, envTextToMap } from './envVars.js'

test('envTextToMap parses key value lines and ignores blanks', () => {
  const env = envTextToMap(`
NODE_ENV=development

VITE_PORT=1420
TOKEN=a=b=c
`)

  assert.deepEqual(env, {
    NODE_ENV: 'development',
    VITE_PORT: '1420',
    TOKEN: 'a=b=c',
  })
})

test('envTextToMap trims keys and skips incomplete lines', () => {
  const env = envTextToMap(`
 PORT = 1420
NO_VALUE
=missing-key
EMPTY=
`)

  assert.deepEqual(env, {
    PORT: '1420',
    EMPTY: '',
  })
})

test('envMapToText serializes env keys in stable order', () => {
  const text = envMapToText({ VITE_PORT: '1420', NODE_ENV: 'development' })

  assert.equal(text, 'NODE_ENV=development\nVITE_PORT=1420')
})
