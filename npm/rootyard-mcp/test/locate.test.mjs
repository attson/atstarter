import assert from 'node:assert/strict'
import { test } from 'node:test'

import { locateRootyardBinary } from '../lib/locate.mjs'

// existsSet builds an existsSync stub that returns true only for the given paths.
function existsSet(paths) {
  const set = new Set(paths)
  return (p) => set.has(p)
}

test('ROOTYARD_BIN wins over PATH and known locations', () => {
  const found = locateRootyardBinary({
    platform: 'linux',
    env: { ROOTYARD_BIN: '/opt/custom/rootyard', PATH: '/usr/bin' },
    existsSync: existsSet(['/opt/custom/rootyard', '/usr/bin/rootyard']),
    delimiter: ':',
  })
  assert.equal(found, '/opt/custom/rootyard')
})

test('ROOTYARD_BIN pointing at a missing file is ignored, falls through to PATH', () => {
  const found = locateRootyardBinary({
    platform: 'linux',
    env: { ROOTYARD_BIN: '/opt/missing/rootyard', PATH: '/usr/bin' },
    existsSync: existsSet(['/usr/bin/rootyard']),
    delimiter: ':',
  })
  assert.equal(found, '/usr/bin/rootyard')
})

test('legacy override is ignored', () => {
  const found = locateRootyardBinary({
    platform: 'linux',
    env: { ATSTARTER_BIN: '/opt/legacy/atstarter', PATH: '/usr/bin' },
    existsSync: existsSet(['/opt/legacy/atstarter', '/usr/bin/rootyard']),
    delimiter: ':',
  })
  assert.equal(found, '/usr/bin/rootyard')
})

test('linux: found on PATH', () => {
  const found = locateRootyardBinary({
    platform: 'linux',
    env: { PATH: '/home/me/.local/bin:/usr/bin' },
    existsSync: existsSet(['/usr/bin/rootyard']),
    delimiter: ':',
  })
  assert.equal(found, '/usr/bin/rootyard')
})

test('linux: falls back to /usr/bin when PATH misses', () => {
  const found = locateRootyardBinary({
    platform: 'linux',
    env: { PATH: '/some/empty/dir' },
    existsSync: existsSet(['/usr/bin/rootyard']),
    delimiter: ':',
  })
  assert.equal(found, '/usr/bin/rootyard')
})

test('linux: falls back to ~/.local/bin (portable install)', () => {
  const found = locateRootyardBinary({
    platform: 'linux',
    env: { PATH: '', HOME: '/home/me' },
    existsSync: existsSet(['/home/me/.local/bin/rootyard']),
    delimiter: ':',
  })
  assert.equal(found, '/home/me/.local/bin/rootyard')
})

test('darwin: falls back to /usr/local/bin symlink', () => {
  const found = locateRootyardBinary({
    platform: 'darwin',
    env: { PATH: '' },
    existsSync: existsSet(['/usr/local/bin/rootyard']),
    delimiter: ':',
  })
  assert.equal(found, '/usr/local/bin/rootyard')
})

test('darwin: falls back to app bundle executable', () => {
  const found = locateRootyardBinary({
    platform: 'darwin',
    env: { PATH: '' },
    existsSync: existsSet(['/Applications/Rootyard.app/Contents/MacOS/rootyard']),
    delimiter: ':',
  })
  assert.equal(found, '/Applications/Rootyard.app/Contents/MacOS/rootyard')
})

test('win32: uses rootyard.exe on PATH', () => {
  const found = locateRootyardBinary({
    platform: 'win32',
    env: { PATH: 'C:\\Program Files\\Rootyard' },
    existsSync: existsSet(['C:\\Program Files\\Rootyard\\rootyard.exe']),
    delimiter: ';',
  })
  assert.equal(found, 'C:\\Program Files\\Rootyard\\rootyard.exe')
})

test('win32: falls back to Program Files install dir', () => {
  const found = locateRootyardBinary({
    platform: 'win32',
    env: {
      PATH: '',
      ProgramFiles: 'C:\\Program Files',
    },
    existsSync: existsSet(['C:\\Program Files\\Rootyard\\rootyard.exe']),
    delimiter: ';',
  })
  assert.equal(found, 'C:\\Program Files\\Rootyard\\rootyard.exe')
})

test('returns null when nothing is found anywhere', () => {
  const found = locateRootyardBinary({
    platform: 'linux',
    env: { PATH: '/nope' },
    existsSync: () => false,
    delimiter: ':',
  })
  assert.equal(found, null)
})

test('skips empty PATH segments without crashing', () => {
  const found = locateRootyardBinary({
    platform: 'linux',
    env: { PATH: '::/usr/bin:' },
    existsSync: existsSet(['/usr/bin/rootyard']),
    delimiter: ':',
  })
  assert.equal(found, '/usr/bin/rootyard')
})
