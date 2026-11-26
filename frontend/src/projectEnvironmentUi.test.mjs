import test from 'node:test'
import assert from 'node:assert/strict'
import {
  normalizeEnvironmentDocument,
  repositoryStatus,
  environmentSummary,
  toggleRepositorySelection,
  selectableRepositoryIDs,
  fieldRows,
} from './projectEnvironmentUi.js'

test('normalizes Go-shaped and missing environment document values', () => {
  const got = normalizeEnvironmentDocument({
    Config: { Version: 1, Defaults: null, Presets: null, Rules: [{ ID: 'rule' }] },
    Revision: 'rev',
  })
  assert.equal(got.revision, 'rev')
  assert.deepEqual(got.config.defaults, { presetId: '', git: {} })
  assert.deepEqual(got.config.presets, [])
  assert.equal(got.config.rules[0].id, 'rule')
})

test('derives repository status with blockers taking precedence', () => {
  assert.equal(repositoryStatus({ status: 'consistent' }), 'consistent')
  assert.equal(repositoryStatus({ lastApplyResult: 'failed' }), 'failed')
  assert.equal(repositoryStatus({ status: 'pending', blockers: [{ code: 'missing_key' }] }), 'blocked')
  assert.equal(repositoryStatus({}), 'pending')
})

test('summarizes rules repositories pending changes and blockers', () => {
  const document = normalizeEnvironmentDocument({ config: { version: 1, rules: [{ id: 'one' }, { id: 'two' }] } })
  const state = {
    repositories: [
      { id: 'a', status: 'pending' },
      { id: 'b', status: 'blocked', blockers: [{ code: 'x' }, { code: 'y' }] },
      { id: 'c', status: 'consistent' },
    ],
  }
  assert.deepEqual(environmentSummary(document, state, [{ changes: [{ operation: 'set' }, { operation: 'unchanged' }] }]), {
    rules: 2,
    repositories: 3,
    pending: 1,
    changes: 1,
    blockers: 2,
  })
})

test('blocked repositories cannot be selected', () => {
  const repos = [{ id: 'ok', status: 'pending' }, { id: 'bad', status: 'blocked' }]
  assert.deepEqual(toggleRepositorySelection(new Set(), 'bad', repos), new Set())
  assert.deepEqual([...toggleRepositorySelection(new Set(), 'ok', repos)], ['ok'])
})

test('selectable IDs include each canonical target once', () => {
  const repos = [
    { id: 'alias-a', targetId: 'shared', status: 'pending' },
    { id: 'alias-b', targetId: 'shared', status: 'pending' },
    { id: 'blocked', targetId: 'other', status: 'blocked' },
  ]
  assert.deepEqual(selectableRepositoryIDs(repos), ['shared'])
})

test('field rows always expose the six managed Git fields', () => {
  const rows = fieldRows({ 'user.name': { mode: 'set', value: 'Ada' } })
  assert.deepEqual(rows.map((row) => row.key), [
    'user.name', 'user.email', 'user.signingKey',
    'commit.gpgSign', 'tag.gpgSign', 'gpg.format',
  ])
  assert.deepEqual(rows[0].intent, { mode: 'set', value: 'Ada' })
  assert.deepEqual(rows[1].intent, { mode: 'inherit', value: '' })
})
