export const environmentGitKeys = [
  'user.name',
  'user.email',
  'user.signingKey',
  'commit.gpgSign',
  'tag.gpgSign',
  'gpg.format',
]

const valueFor = (value, lower, upper, fallback) => value?.[lower] ?? value?.[upper] ?? fallback

function normalizeIntent(intent) {
  return {
    mode: valueFor(intent, 'mode', 'Mode', 'inherit') || 'inherit',
    value: valueFor(intent, 'value', 'Value', '') || '',
  }
}

function normalizeFields(fields) {
  const result = {}
  for (const [key, intent] of Object.entries(fields || {})) result[key] = normalizeIntent(intent)
  return result
}

function normalizePreset(preset) {
  return {
    id: valueFor(preset, 'id', 'ID', ''),
    name: valueFor(preset, 'name', 'Name', ''),
    description: valueFor(preset, 'description', 'Description', ''),
    git: normalizeFields(valueFor(preset, 'git', 'Git', {})),
  }
}

function normalizeRule(rule) {
  return {
    id: valueFor(rule, 'id', 'ID', ''),
    name: valueFor(rule, 'name', 'Name', ''),
    path: valueFor(rule, 'path', 'Path', ''),
    recursive: Boolean(valueFor(rule, 'recursive', 'Recursive', false)),
    autoDiscover: Boolean(valueFor(rule, 'autoDiscover', 'AutoDiscover', false)),
    presetId: valueFor(rule, 'presetId', 'PresetID', ''),
    git: normalizeFields(valueFor(rule, 'git', 'Git', {})),
  }
}

export function normalizeEnvironmentDocument(document = {}) {
  const config = valueFor(document, 'config', 'Config', {}) || {}
  const defaults = valueFor(config, 'defaults', 'Defaults', {}) || {}
  return {
    revision: valueFor(document, 'revision', 'Revision', ''),
    config: {
      version: valueFor(config, 'version', 'Version', 1) || 1,
      defaults: {
        presetId: valueFor(defaults, 'presetId', 'PresetID', ''),
        git: normalizeFields(valueFor(defaults, 'git', 'Git', {})),
      },
      presets: (valueFor(config, 'presets', 'Presets', []) || []).map(normalizePreset),
      rules: (valueFor(config, 'rules', 'Rules', []) || []).map(normalizeRule),
    },
  }
}

export function repositoryStatus(repository = {}) {
  const blockers = valueFor(repository, 'blockers', 'Blockers', []) || []
  if (blockers.length) return 'blocked'
  const status = valueFor(repository, 'status', 'Status', '')
  if (['pending', 'consistent', 'drifted', 'blocked', 'failed'].includes(status)) return status
  const apply = valueFor(repository, 'lastApplyResult', 'LastApplyResult', '')
  if (apply === 'failed') return 'failed'
  if (apply === 'blocked' || apply === 'stale') return 'blocked'
  if (apply === 'success' || apply === 'unchanged') return 'consistent'
  return 'pending'
}

export function environmentSummary(document, state = {}, plans = []) {
  const normalized = normalizeEnvironmentDocument(document)
  const repositories = valueFor(state, 'repositories', 'Repositories', []) || []
  return {
    rules: normalized.config.rules.length,
    repositories: repositories.length,
    pending: repositories.filter((repository) => repositoryStatus(repository) === 'pending').length,
    changes: (plans || []).flatMap((plan) => valueFor(plan, 'changes', 'Changes', []) || [])
      .filter((change) => !['unchanged', 'unmanaged'].includes(valueFor(change, 'operation', 'Operation', ''))).length,
    blockers: repositories.reduce((total, repository) => total + (valueFor(repository, 'blockers', 'Blockers', []) || []).length, 0),
  }
}

function repositoryID(repository) {
  return valueFor(repository, 'targetId', 'TargetID', '') || valueFor(repository, 'id', 'ID', '')
}

export function toggleRepositorySelection(selection, id, repositories) {
  const next = new Set(selection || [])
  const repository = (repositories || []).find((item) => repositoryID(item) === id || valueFor(item, 'id', 'ID', '') === id)
  if (!repository || repositoryStatus(repository) === 'blocked') return next
  const canonicalID = repositoryID(repository)
  if (next.has(canonicalID)) next.delete(canonicalID)
  else next.add(canonicalID)
  return next
}

export function selectableRepositoryIDs(repositories) {
  const ids = []
  const seen = new Set()
  for (const repository of repositories || []) {
    if (repositoryStatus(repository) === 'blocked') continue
    const id = repositoryID(repository)
    if (!id || seen.has(id)) continue
    seen.add(id)
    ids.push(id)
  }
  return ids
}

export function fieldRows(fields = {}) {
  return environmentGitKeys.map((key) => ({
    key,
    intent: normalizeIntent(fields[key]),
  }))
}
