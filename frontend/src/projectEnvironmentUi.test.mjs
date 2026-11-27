import test from 'node:test'
import assert from 'node:assert/strict'
import {
  environmentCodexKeys,
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
  codexFieldRows,
  codexModelSuggestions,
  dangerousCodexCombination,
  mergeCodexFields,
  adapterChangeCounts,
  plansHaveIssueCode,
  repositoryBlockers,
} from './projectEnvironmentUi.js'

test('normalizes Go-shaped and missing environment document values', () => {
  const got = normalizeEnvironmentDocument({
    Config: { Version: 1, Defaults: null, Presets: null, Rules: [{ ID: 'rule' }] },
    Revision: 'rev',
  })
  assert.equal(got.revision, 'rev')
  assert.deepEqual(got.config.defaults, { presetId: '', git: {}, codex: {} })
  assert.deepEqual(got.config.presets, [])
  assert.equal(got.config.rules[0].id, 'rule')
})

test('normalizes Codex fields independently from Git', () => {
  const document = normalizeEnvironmentDocument({ Config: { Version: 1, Presets: [{
    ID: 'p', Name: 'Company',
    Git: { 'user.name': { Mode: 'set', Value: 'Ada' } },
    Codex: { model: { Mode: 'set', Value: 'custom-model' } },
  }] } })
  assert.equal(document.config.presets[0].git['user.name'].value, 'Ada')
  assert.equal(document.config.presets[0].codex.model.value, 'custom-model')
  assert.deepEqual(environmentCodexKeys, [
    'model', 'model_reasoning_effort', 'sandbox_mode', 'approval_policy', 'web_search',
  ])
})

test('Codex field rows retain free-form reasoning and expose fixed enums', () => {
  const rows = codexFieldRows({ model_reasoning_effort: { mode: 'set', value: 'ultra' } })
  assert.equal(rows.find((row) => row.key === 'model_reasoning_effort').intent.value, 'ultra')
  assert.deepEqual(rows.find((row) => row.key === 'sandbox_mode').options.map((option) => option.value), [
    'read-only', 'workspace-write', 'danger-full-access',
  ])
  assert.deepEqual(rows.find((row) => row.key === 'approval_policy').options.map((option) => option.value), [
    'untrusted', 'on-failure', 'on-request', 'granular', 'never',
  ])
})

test('Codex metadata exposes stable presentation keys while preserving native values and suggestions', () => {
  const rows = codexFieldRows({})
  assert.deepEqual(rows.find((row) => row.key === 'sandbox_mode').options, [
    { value: 'read-only', labelKey: 'environments.fields.sandbox.options.readOnly' },
    { value: 'workspace-write', labelKey: 'environments.fields.sandbox.options.workspaceWrite' },
    { value: 'danger-full-access', labelKey: 'environments.fields.sandbox.options.dangerFullAccess' },
  ])
  assert.equal(rows.find((row) => row.key === 'model').labelKey, 'environments.fields.model.label')
  assert.deepEqual(rows.map((row) => row.descriptionKey), [
    'environments.fields.model.description', 'environments.fields.reasoning.description',
    'environments.fields.sandbox.description', 'environments.fields.approval.description',
    'environments.fields.webSearch.description',
  ])
  for (const row of rows) {
    assert.equal(Object.hasOwn(row, 'label'), false)
    assert.equal(Object.hasOwn(row, 'description'), false)
    for (const option of row.options) {
      assert.equal(Object.hasOwn(option, 'label'), false)
      assert.ok(option.labelKey.startsWith('environments.fields.'))
      assert.deepEqual(editFieldValue({}, row.key, option.value), { [row.key]: { mode: 'set', value: option.value } })
    }
  }
  assert.deepEqual(rows.find((row) => row.key === 'model_reasoning_effort').suggestions, ['low', 'medium', 'high', 'xhigh'])
  assert.deepEqual(rows.find((row) => row.key === 'web_search').options.map((option) => option.value), ['disabled', 'cached', 'indexed', 'live'])
})

test('recent Codex model suggestions are deduplicated from managed documents and plans', () => {
  const document = normalizeEnvironmentDocument({ config: {
    version: 1,
    defaults: { codex: { model: { mode: 'set', value: 'default-model' } } },
    presets: [{ id: 'p', name: 'One', codex: { model: { mode: 'set', value: 'custom-model' } } }],
    rules: [{ id: 'r', name: 'Rule', codex: { model: { mode: 'set', value: 'custom-model' } } }],
  } })
  const plans = [{
    adapter: 'codex',
    desired: { model: { mode: 'set', value: 'planned-model' } },
    snapshot: { values: { model: [{ value: 'inspected-model' }, { value: 'custom-model' }] } },
  }]
  assert.deepEqual(codexModelSuggestions(document, plans), [
    'default-model', 'custom-model', 'planned-model', 'inspected-model',
  ])
})

test('Codex validation accepts free-form text and rejects invalid fixed enum values', () => {
  assert.equal(validatePresetDraft({
    name: 'Company',
    codex: { model_reasoning_effort: { mode: 'set', value: 'ultra' } },
  }).valid, true)
  const result = validateRuleDraft({
    name: 'Work', path: '/work',
    codex: { sandbox_mode: { mode: 'set', value: 'unrestricted' } },
  })
  assert.equal(result.valid, false)
  assert.equal(result.errors.sandbox_mode, 'Select a supported value')
})

test('dangerous Codex combination requires the effective sandbox and approval pair', () => {
  assert.equal(dangerousCodexCombination({
    sandbox_mode: { mode: 'set', value: 'danger-full-access' },
    approval_policy: { mode: 'set', value: 'never' },
  }), true)
  assert.equal(dangerousCodexCombination({
    sandbox_mode: { mode: 'set', value: 'danger-full-access' },
    approval_policy: { mode: 'inherit' },
  }), false)
  assert.equal(dangerousCodexCombination({
    sandbox_mode: { intent: { mode: 'set', value: 'workspace-write' } },
    approval_policy: { intent: { mode: 'set', value: 'never' } },
  }), false)
})

test('Codex field layers preserve inherit and replace with set or clear', () => {
  const preset = {
    sandbox_mode: { mode: 'set', value: 'danger-full-access' },
    approval_policy: { mode: 'set', value: 'on-request' },
  }
  const effective = mergeCodexFields(preset, {
    sandbox_mode: { mode: 'inherit' },
    approval_policy: { mode: 'set', value: 'never' },
  })
  assert.equal(dangerousCodexCombination(effective), true)
  assert.deepEqual(mergeCodexFields(effective, {
    sandbox_mode: { mode: 'clear' },
  }).sandbox_mode, { mode: 'clear', value: '' })
})

test('derives repository status with blockers taking precedence', () => {
  assert.equal(repositoryStatus({ status: 'consistent' }), 'consistent')
  assert.equal(repositoryStatus({ lastApplyResult: 'failed' }), 'failed')
  assert.equal(repositoryStatus({ status: 'pending', blockers: [{ code: 'missing_key' }] }), 'blocked')
  assert.equal(repositoryStatus({}), 'pending')
})

test('version 2 repository status and blockers aggregate adapter states', () => {
  const repository = {
    status: 'consistent',
    adapters: {
      git: { status: 'consistent', blockers: [] },
      codex: { status: 'drifted', blockers: [{ code: 'invalid_codex_config' }] },
    },
  }
  assert.equal(repositoryStatus(repository), 'blocked')
  assert.deepEqual(repositoryBlockers(repository), [{ code: 'invalid_codex_config' }])
  assert.equal(repositoryStatus({ adapters: { git: { status: 'failed', blockers: [{ code: 'apply_failed' }] }, codex: { status: 'blocked' } } }), 'failed')
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

test('adapter change counts and dangerous plans use structured plan fields', () => {
  const plans = [
    { adapter: 'git', changes: [{ operation: 'set' }] },
    { adapter: 'codex', changes: [{ operation: 'replace' }, { operation: 'unchanged' }], warnings: [{ code: 'dangerous_codex_permissions' }] },
  ]
  assert.deepEqual(adapterChangeCounts(plans), { git: 1, codex: 1 })
  assert.equal(plansHaveIssueCode(plans, 'dangerous_codex_permissions'), true)
  assert.equal(plansHaveIssueCode(plans, 'danger-full-access'), false)
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
