import test from 'node:test'
import assert from 'node:assert/strict'
import {
  normalizeEnvironmentDocument,
  repositoryStatus,
  environmentSummary,
  toggleRepositorySelection,
  selectableRepositoryIDs,
  fieldRows,
  editFieldMode,
  editFieldValue,
  presetUsage,
  validatePresetDraft,
  validateRuleDraft,
  isEnvironmentStaleError,
  filterEnvironmentRepositories,
  selectAllRepositories,
  dedupedTargetCount,
  planChangeCount,
  summarizeApplyResults,
  stalePlanTargetIDs,
  sharedTargetConflict,
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

test('field mode and value edits are immutable and clear stale values', () => {
  const original = { 'user.name': { mode: 'set', value: 'Ada' } }
  const cleared = editFieldMode(original, 'user.name', 'clear')
  assert.deepEqual(cleared['user.name'], { mode: 'clear' })
  assert.deepEqual(original['user.name'], { mode: 'set', value: 'Ada' })
  assert.deepEqual(editFieldMode(original, 'user.name', 'inherit')['user.name'], { mode: 'inherit' })
  assert.deepEqual(editFieldValue(cleared, 'user.name', 'Grace')['user.name'], { mode: 'set', value: 'Grace' })
})

test('preset usage includes defaults rules and repository source IDs', () => {
  const document = normalizeEnvironmentDocument({ config: {
    version: 1,
    defaults: { presetId: 'company' },
    rules: [{ id: 'work', presetId: 'company' }, { id: 'other', presetId: 'personal' }],
  } })
  const state = { repositories: [
    { id: 'one', winningSourceIds: ['company'] },
    { id: 'two', WinningSourceIDs: ['personal'] },
  ] }
  assert.deepEqual(presetUsage(document, state, 'company'), {
    defaults: true,
    rules: ['work'],
    repositories: ['one'],
    total: 3,
  })
})

test('draft validation requires names paths and set values', () => {
  assert.equal(validatePresetDraft({ name: '', git: {} }).valid, false)
  assert.equal(validatePresetDraft({ name: 'Company', git: { 'user.email': { mode: 'set', value: '' } } }).valid, false)
  assert.deepEqual(validatePresetDraft({ name: 'Company', git: { 'user.email': { mode: 'set', value: 'a@example.com' } } }), { valid: true, errors: {} })
  assert.equal(validateRuleDraft({ name: 'Work', path: '', git: {} }).valid, false)
  assert.equal(validateRuleDraft({ name: 'Work', path: '/work', git: {} }).valid, true)
})

test('stale revision mapping accepts structured and Wails string errors', () => {
  assert.equal(isEnvironmentStaleError({ code: 'environment_stale' }), true)
  assert.equal(isEnvironmentStaleError(new Error('environment_stale: reload and retry')), true)
  assert.equal(isEnvironmentStaleError(new Error('permission denied')), false)
})

test('repository filters combine status rule and path search', () => {
  const repositories = [
    { id: 'one', status: 'pending', aliases: [{ path: '/work/api', ruleIds: ['work'] }] },
    { id: 'two', status: 'consistent', aliases: [{ path: '/personal/site', ruleIds: ['personal'] }] },
    { id: 'three', status: 'blocked', aliases: [{ path: '/work/web', ruleIds: ['work'] }] },
  ]
  assert.deepEqual(filterEnvironmentRepositories(repositories, { status: 'pending', ruleId: 'work', search: 'api' }).map((item) => item.id), ['one'])
})

test('select all excludes blocked rows and deduplicates targets', () => {
  const repositories = [
    { id: 'a', targetId: 'shared', status: 'pending' },
    { id: 'b', targetId: 'shared', status: 'consistent' },
    { id: 'c', targetId: 'blocked', status: 'blocked' },
  ]
  assert.deepEqual([...selectAllRepositories(repositories)], ['shared'])
  assert.equal(dedupedTargetCount(repositories), 2)
})

test('plan change counts exclude unchanged and unmanaged operations', () => {
  const plans = [{ changes: [
    { operation: 'set' }, { operation: 'replace' }, { operation: 'unchanged' }, { operation: 'unmanaged' },
  ] }]
  assert.equal(planChangeCount(plans), 2)
})

test('apply result summary and stale targets preserve target identity', () => {
  const results = [
    { target: { id: 'a' }, status: 'success' },
    { target: { id: 'b' }, status: 'stale' },
    { target: { id: 'c' }, status: 'failed' },
  ]
  assert.deepEqual(summarizeApplyResults(results), { success: 1, unchanged: 0, stale: 1, blocked: 0, failed: 1, total: 3 })
  assert.deepEqual(stalePlanTargetIDs(results), ['b'])
})

test('shared target conflict returns all aliases for display', () => {
  const plan = {
    aliases: [{ path: '/work/main' }, { path: '/work/linked' }],
    blockers: [{ code: 'shared_target_conflict', detail: '/git/common' }],
  }
  assert.deepEqual(sharedTargetConflict(plan), {
    target: '/git/common',
    paths: ['/work/main', '/work/linked'],
  })
})
