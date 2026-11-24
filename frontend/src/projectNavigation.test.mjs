import test from 'node:test'
import assert from 'node:assert/strict'
import {
  advanceBranchRevision,
  applyBranchEntry,
  DEFAULT_PROJECT_CARD_FIELDS,
  DEFAULT_PROJECT_LIST_VIEW,
  PROJECT_CARD_FIELDS_KEY,
  PROJECT_LIST_VIEW_KEY,
  loadProjectBranches,
  loadVisibleProjectBranches,
  normalizeProjectCardFields,
  normalizeProjectListView,
  readProjectCardFields,
  readProjectListView,
  writeProjectCardFields,
  writeProjectListView,
} from './projectNavigation.js'

function memoryStorage(initial = {}) {
  const values = new Map(Object.entries(initial))
  return {
    getItem(key) { return values.has(key) ? values.get(key) : null },
    setItem(key, value) { values.set(key, value) },
  }
}

test('project list view persists only list and card values', () => {
  const storage = memoryStorage()

  assert.equal(DEFAULT_PROJECT_LIST_VIEW, 'list')
  assert.equal(normalizeProjectListView('card'), 'card')
  assert.equal(normalizeProjectListView('tiles'), 'list')
  assert.equal(readProjectListView(storage), 'list')

  assert.equal(writeProjectListView(storage, 'card'), 'card')
  assert.equal(storage.getItem(PROJECT_LIST_VIEW_KEY), 'card')
  assert.equal(readProjectListView(storage), 'card')
})

test('project list view storage failures fall back without breaking navigation', () => {
  const broken = {
    getItem() { throw new Error('blocked') },
    setItem() { throw new Error('blocked') },
  }

  assert.equal(readProjectListView(broken), 'list')
  assert.equal(writeProjectListView(broken, 'card'), 'card')
})

test('project card fields persist valid booleans and default each invalid field to visible', () => {
  const storage = memoryStorage({
    [PROJECT_CARD_FIELDS_KEY]: JSON.stringify({ type: false, branch: true, path: 'no', extra: false }),
  })

  assert.deepEqual(DEFAULT_PROJECT_CARD_FIELDS, {
    type: true,
    branch: true,
    path: true,
    command: true,
  })
  assert.deepEqual(readProjectCardFields(storage), {
    type: false,
    branch: true,
    path: true,
    command: true,
  })
  assert.deepEqual(normalizeProjectCardFields(null), DEFAULT_PROJECT_CARD_FIELDS)

  const next = writeProjectCardFields(storage, {
    type: false,
    branch: false,
    path: true,
    command: false,
  })
  assert.deepEqual(next, { type: false, branch: false, path: true, command: false })
  assert.equal(storage.getItem(PROJECT_CARD_FIELDS_KEY), JSON.stringify(next))
})

test('project card field storage failures fall back without breaking navigation', () => {
  const broken = {
    getItem() { throw new Error('blocked') },
    setItem() { throw new Error('blocked') },
  }
  const malformed = memoryStorage({ [PROJECT_CARD_FIELDS_KEY]: '{' })

  assert.deepEqual(readProjectCardFields(broken), DEFAULT_PROJECT_CARD_FIELDS)
  assert.deepEqual(readProjectCardFields(malformed), DEFAULT_PROJECT_CARD_FIELDS)
  assert.deepEqual(
    writeProjectCardFields(broken, { type: false, branch: false, path: false, command: false }),
    { type: false, branch: false, path: false, command: false },
  )
})

test('branch loading limits concurrency and isolates project failures', async () => {
  const projects = Array.from({ length: 7 }, (_, index) => ({
    id: `p${index}`,
    path: `/workspace/p${index}`,
  }))
  let active = 0
  let maxActive = 0
  const seen = []

  const entries = await loadProjectBranches(projects, async (path) => {
    active += 1
    maxActive = Math.max(maxActive, active)
    await new Promise((resolve) => setTimeout(resolve, 2))
    active -= 1
    if (path.endsWith('/p3')) throw new Error('git unavailable')
    if (path.endsWith('/p5')) return ''
    return `feature/${path.slice(-2)}`
  }, (id, entry) => seen.push([id, entry]), 3)

  assert.equal(maxActive, 3)
  assert.deepEqual(entries.p0, { state: 'ready', branch: 'feature/p0' })
  assert.deepEqual(entries.p3, { state: 'error', branch: '' })
  assert.deepEqual(entries.p5, { state: 'ready', branch: '' })
  assert.equal(seen.length, 7)
})

test('branch loading de-duplicates repeated project nodes by project id', async () => {
  let calls = 0
  const entries = await loadProjectBranches([
    { id: 'shared', path: '/workspace/shared' },
    { id: 'shared', path: '/workspace/shared' },
  ], async () => {
    calls += 1
    return 'main'
  })

  assert.equal(calls, 1)
  assert.deepEqual(entries, { shared: { state: 'ready', branch: 'main' } })
})

test('branch reads run only for a visible branch field and ignore results hidden in flight', async () => {
  const projects = [
    { id: 'p1', path: '/workspace/p1' },
    { id: 'p2', path: '/workspace/p2' },
  ]
  let calls = 0
  const published = []
  const getBranch = async () => {
    calls += 1
    return 'main'
  }

  await loadVisibleProjectBranches(projects, 'list', { branch: true }, getBranch)
  await loadVisibleProjectBranches(projects, 'card', { branch: false }, getBranch)
  assert.equal(calls, 0)

  let resolveBranch
  let visible = true
  const pending = loadVisibleProjectBranches(
    projects,
    'card',
    { branch: true },
    (path) => {
      calls += 1
      if (path.endsWith('/p2')) return 'feature/queued'
      return new Promise((resolve) => { resolveBranch = resolve })
    },
    (id, entry) => published.push([id, entry]),
    () => visible,
    1,
  )
  await Promise.resolve()
  visible = false
  resolveBranch('feature/old')
  await pending

  assert.equal(calls, 1)
  assert.deepEqual(published, [])

  await loadVisibleProjectBranches(
    projects,
    'card',
    { branch: true },
    getBranch,
    (id, entry) => published.push([id, entry]),
  )
  assert.equal(calls, 3)
  assert.deepEqual(published, [
    ['p1', { state: 'ready', branch: 'main' }],
    ['p2', { state: 'ready', branch: 'main' }],
  ])
})

test('a branch switch prevents an older branch read from overwriting the card', () => {
  const initial = { project: { state: 'loading', branch: '' } }
  const revisions = advanceBranchRevision({}, 'project')
  const switched = applyBranchEntry(
    initial,
    revisions,
    'project',
    { state: 'ready', branch: 'feature/new' },
  )

  const afterStaleRead = applyBranchEntry(
    switched,
    revisions,
    'project',
    { state: 'ready', branch: 'main' },
    0,
  )

  assert.equal(revisions.project, 1)
  assert.equal(afterStaleRead, switched)
  assert.deepEqual(afterStaleRead.project, { state: 'ready', branch: 'feature/new' })
})
