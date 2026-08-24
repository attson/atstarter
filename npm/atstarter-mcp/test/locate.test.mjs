import assert from 'node:assert/strict'
import { test } from 'node:test'

import { locateAtstarterBinary } from '../lib/locate.mjs'

// existsSet builds an existsSync stub that returns true only for the given paths.
function existsSet(paths) {
  const set = new Set(paths)
  return (p) => set.has(p)
}

test('ATSTARTER_BIN wins over PATH and known locations', () => {
  const found = locateAtstarterBinary({
    platform: 'linux',
    env: { ATSTARTER_BIN: '/opt/custom/atstarter', PATH: '/usr/bin' },
    existsSync: existsSet(['/opt/custom/atstarter', '/usr/bin/atstarter']),
    delimiter: ':',
  })
  assert.equal(found, '/opt/custom/atstarter')
})

test('ATSTARTER_BIN pointing at a missing file is ignored, falls through to PATH', () => {
  const found = locateAtstarterBinary({
    platform: 'linux',
    env: { ATSTARTER_BIN: '/opt/missing/atstarter', PATH: '/usr/bin' },
    existsSync: existsSet(['/usr/bin/atstarter']),
    delimiter: ':',
  })
  assert.equal(found, '/usr/bin/atstarter')
})

test('linux: found on PATH', () => {
  const found = locateAtstarterBinary({
    platform: 'linux',
    env: { PATH: '/home/me/.local/bin:/usr/bin' },
    existsSync: existsSet(['/usr/bin/atstarter']),
    delimiter: ':',
  })
  assert.equal(found, '/usr/bin/atstarter')
})

test('linux: falls back to /usr/bin when PATH misses', () => {
  const found = locateAtstarterBinary({
    platform: 'linux',
    env: { PATH: '/some/empty/dir' },
    existsSync: existsSet(['/usr/bin/atstarter']),
    delimiter: ':',
  })
  assert.equal(found, '/usr/bin/atstarter')
})

test('linux: falls back to ~/.local/bin (portable install)', () => {
  const found = locateAtstarterBinary({
    platform: 'linux',
    env: { PATH: '', HOME: '/home/me' },
    existsSync: existsSet(['/home/me/.local/bin/atstarter']),
    delimiter: ':',
  })
  assert.equal(found, '/home/me/.local/bin/atstarter')
})

test('darwin: falls back to /usr/local/bin symlink', () => {
  const found = locateAtstarterBinary({
    platform: 'darwin',
    env: { PATH: '' },
    existsSync: existsSet(['/usr/local/bin/atstarter']),
    delimiter: ':',
  })
  assert.equal(found, '/usr/local/bin/atstarter')
})

test('darwin: falls back to app bundle executable', () => {
  const found = locateAtstarterBinary({
    platform: 'darwin',
    env: { PATH: '' },
    existsSync: existsSet(['/Applications/AT Starter.app/Contents/MacOS/atstarter']),
    delimiter: ':',
  })
  assert.equal(found, '/Applications/AT Starter.app/Contents/MacOS/atstarter')
})

test('win32: uses atstarter.exe on PATH', () => {
  const found = locateAtstarterBinary({
    platform: 'win32',
    env: { PATH: 'C:\\Program Files\\AT Starter' },
    existsSync: existsSet(['C:\\Program Files\\AT Starter\\atstarter.exe']),
    delimiter: ';',
  })
  assert.equal(found, 'C:\\Program Files\\AT Starter\\atstarter.exe')
})

test('win32: falls back to Program Files install dir', () => {
  const found = locateAtstarterBinary({
    platform: 'win32',
    env: {
      PATH: '',
      ProgramFiles: 'C:\\Program Files',
    },
    existsSync: existsSet(['C:\\Program Files\\AT Starter\\atstarter.exe']),
    delimiter: ';',
  })
  assert.equal(found, 'C:\\Program Files\\AT Starter\\atstarter.exe')
})

test('returns null when nothing is found anywhere', () => {
  const found = locateAtstarterBinary({
    platform: 'linux',
    env: { PATH: '/nope' },
    existsSync: () => false,
    delimiter: ':',
  })
  assert.equal(found, null)
})

test('skips empty PATH segments without crashing', () => {
  const found = locateAtstarterBinary({
    platform: 'linux',
    env: { PATH: '::/usr/bin:' },
    existsSync: existsSet(['/usr/bin/atstarter']),
    delimiter: ':',
  })
  assert.equal(found, '/usr/bin/atstarter')
})
