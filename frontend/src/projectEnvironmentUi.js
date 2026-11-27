export const environmentGitKeys = [
  'user.name',
  'user.email',
  'user.signingKey',
  'commit.gpgSign',
  'tag.gpgSign',
  'gpg.format',
]

export const environmentCodexKeys = [
  'model',
  'model_reasoning_effort',
  'sandbox_mode',
  'approval_policy',
  'web_search',
]

const codexFieldMetadata = {
  model: { label: 'Model', description: 'Project model', control: 'input', suggestions: [] },
  model_reasoning_effort: {
    label: 'Reasoning effort',
    description: 'Project reasoning effort',
    control: 'input',
    suggestions: ['low', 'medium', 'high', 'xhigh'],
  },
  sandbox_mode: {
    label: 'Sandbox',
    description: 'Filesystem and process isolation',
    control: 'select',
    options: [
      { value: 'read-only', label: 'Read only' },
      { value: 'workspace-write', label: 'Workspace write' },
      { value: 'danger-full-access', label: 'Danger full access' },
    ],
  },
  approval_policy: {
    label: 'Approval policy',
    description: 'Command approval behavior',
    control: 'select',
    options: [
      { value: 'untrusted', label: 'Untrusted' },
      { value: 'on-failure', label: 'On failure' },
      { value: 'on-request', label: 'On request' },
      { value: 'granular', label: 'Granular' },
      { value: 'never', label: 'Never' },
    ],
  },
  web_search: {
    label: 'Web search',
    description: 'Search data source',
    control: 'select',
    options: [
      { value: 'disabled', label: 'Disabled' },
      { value: 'cached', label: 'Cached' },
      { value: 'indexed', label: 'Indexed' },
      { value: 'live', label: 'Live' },
    ],
  },
}

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
    codex: normalizeFields(valueFor(preset, 'codex', 'Codex', {})),
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
    codex: normalizeFields(valueFor(rule, 'codex', 'Codex', {})),
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
        codex: normalizeFields(valueFor(defaults, 'codex', 'Codex', {})),
      },
      presets: (valueFor(config, 'presets', 'Presets', []) || []).map(normalizePreset),
      rules: (valueFor(config, 'rules', 'Rules', []) || []).map(normalizeRule),
    },
  }
}

export function repositoryStatus(repository = {}) {
  const adapters = valueFor(repository, 'adapters', 'Adapters', null)
  if (adapters && Object.keys(adapters).length) {
    const statuses = Object.values(adapters).flatMap((adapter) => [
      valueFor(adapter, 'status', 'Status', 'pending'),
      ...((valueFor(adapter, 'blockers', 'Blockers', []) || []).length ? ['blocked'] : []),
    ])
    const severity = ['failed', 'blocked', 'drifted', 'pending', 'consistent']
    return severity.find((status) => statuses.includes(status)) || 'pending'
  }
  if (repositoryBlockers(repository).length) return 'blocked'
  const status = valueFor(repository, 'status', 'Status', '')
  if (['pending', 'consistent', 'drifted', 'blocked', 'failed'].includes(status)) return status
  const apply = valueFor(repository, 'lastApplyResult', 'LastApplyResult', '')
  if (apply === 'failed') return 'failed'
  if (apply === 'blocked' || apply === 'stale') return 'blocked'
  if (apply === 'success' || apply === 'unchanged') return 'consistent'
  return 'pending'
}

export function repositoryBlockers(repository = {}) {
  const adapters = valueFor(repository, 'adapters', 'Adapters', null)
  if (adapters && Object.keys(adapters).length) {
    return Object.values(adapters).flatMap((adapter) => valueFor(adapter, 'blockers', 'Blockers', []) || [])
  }
  return valueFor(repository, 'blockers', 'Blockers', []) || []
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
    blockers: repositories.reduce((total, repository) => total + repositoryBlockers(repository).length, 0),
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

export function codexFieldRows(fields = {}) {
  return environmentCodexKeys.map((key) => ({
    key,
    ...codexFieldMetadata[key],
    options: codexFieldMetadata[key].options || [],
    suggestions: codexFieldMetadata[key].suggestions || [],
    intent: normalizeIntent(fields[key]),
  }))
}

function addCodexModelSuggestion(suggestions, seen, intent) {
  const normalized = normalizeIntent(intent)
  const value = normalized.mode === 'set' ? normalized.value.trim() : ''
  if (!value || seen.has(value)) return
  seen.add(value)
  suggestions.push(value)
}

export function codexModelSuggestions(document, plans = []) {
  const normalized = normalizeEnvironmentDocument(document)
  const suggestions = []
  const seen = new Set()
  addCodexModelSuggestion(suggestions, seen, normalized.config.defaults.codex.model)
  for (const preset of normalized.config.presets) addCodexModelSuggestion(suggestions, seen, preset.codex.model)
  for (const rule of normalized.config.rules) addCodexModelSuggestion(suggestions, seen, rule.codex.model)
  for (const plan of plans || []) {
    const adapter = valueFor(plan, 'adapter', 'Adapter', '')
    if (adapter && adapter !== 'codex') continue
    const desired = valueFor(plan, 'desired', 'Desired', {}) || {}
    addCodexModelSuggestion(suggestions, seen, desired.model)
    const snapshot = valueFor(plan, 'snapshot', 'Snapshot', {}) || {}
    const values = valueFor(snapshot, 'values', 'Values', {}) || {}
    for (const origin of values.model || []) {
      addCodexModelSuggestion(suggestions, seen, { mode: 'set', value: valueFor(origin, 'value', 'Value', '') })
    }
  }
  return suggestions
}

function effectiveIntent(field) {
  return valueFor(field, 'intent', 'Intent', field) || {}
}

export function dangerousCodexCombination(fields = {}) {
  const sandbox = normalizeIntent(effectiveIntent(fields.sandbox_mode))
  const approval = normalizeIntent(effectiveIntent(fields.approval_policy))
  return sandbox.mode === 'set' && sandbox.value === 'danger-full-access' &&
    approval.mode === 'set' && approval.value === 'never'
}

export function mergeCodexFields(...layers) {
  const result = {}
  for (const layer of layers) {
    for (const key of environmentCodexKeys) {
      if (!layer?.[key]) continue
      const intent = normalizeIntent(layer[key])
      if (intent.mode === 'inherit') continue
      result[key] = intent
    }
  }
  return result
}

export function editFieldMode(fields = {}, key, mode) {
  const next = { ...fields }
  next[key] = mode === 'set'
    ? { mode: 'set', value: fields[key]?.value || '' }
    : { mode }
  return next
}

export function editFieldValue(fields = {}, key, value) {
  return { ...fields, [key]: { mode: 'set', value } }
}

export function presetUsage(document, state = {}, presetID) {
  const normalized = normalizeEnvironmentDocument(document)
  const rules = normalized.config.rules.filter((rule) => rule.presetId === presetID).map((rule) => rule.id)
  const repositories = (valueFor(state, 'repositories', 'Repositories', []) || [])
    .filter((repository) => (valueFor(repository, 'winningSourceIds', 'WinningSourceIDs', []) || []).includes(presetID))
    .map((repository) => valueFor(repository, 'id', 'ID', ''))
    .filter(Boolean)
  const defaults = normalized.config.defaults.presetId === presetID
  return { defaults, rules, repositories, total: Number(defaults) + rules.length + repositories.length }
}

function validateFields(fields, errors) {
  for (const [key, intent] of Object.entries(fields || {})) {
    if (intent?.mode === 'set' && !String(intent.value || '').trim()) errors[key] = 'A value is required'
  }
}

function validateCodexFields(fields, errors) {
  validateFields(fields, errors)
  for (const [key, intent] of Object.entries(fields || {})) {
    if (intent?.mode !== 'set') continue
    const options = codexFieldMetadata[key]?.options
    if (options && !options.some((option) => option.value === intent.value)) {
      errors[key] = 'Select a supported value'
    }
  }
}

export function validatePresetDraft(draft = {}) {
  const errors = {}
  if (!String(draft.name || '').trim()) errors.name = 'Name is required'
  validateFields(draft.git, errors)
  validateCodexFields(draft.codex, errors)
  return { valid: Object.keys(errors).length === 0, errors }
}

export function validateRuleDraft(draft = {}) {
  const errors = {}
  if (!String(draft.name || '').trim()) errors.name = 'Name is required'
  if (!String(draft.path || '').trim()) errors.path = 'Directory is required'
  validateFields(draft.git, errors)
  validateCodexFields(draft.codex, errors)
  return { valid: Object.keys(errors).length === 0, errors }
}

export function isEnvironmentStaleError(error) {
  const code = error?.code ?? error?.Code ?? error?.error?.code
  return code === 'environment_stale' || String(error?.message || error || '').includes('environment_stale') || String(error?.message || error || '').includes('configuration changed')
}

export function filterEnvironmentRepositories(repositories, filters = {}) {
  const search = String(filters.search || '').toLowerCase()
  return (repositories || []).filter((repository) => {
    if (filters.status && filters.status !== 'all' && repositoryStatus(repository) !== filters.status) return false
    const aliases = valueFor(repository, 'aliases', 'Aliases', []) || []
    if (filters.ruleId && !aliases.some((alias) => (valueFor(alias, 'ruleIds', 'RuleIDs', []) || []).includes(filters.ruleId))) return false
    if (search) {
      const haystack = [valueFor(repository, 'id', 'ID', ''), valueFor(repository, 'targetId', 'TargetID', ''), ...aliases.map((alias) => valueFor(alias, 'path', 'Path', ''))].join(' ').toLowerCase()
      if (!haystack.includes(search)) return false
    }
    return true
  })
}

export function selectAllRepositories(repositories) {
  return new Set(selectableRepositoryIDs(repositories))
}

export function dedupedTargetCount(repositories) {
  return new Set((repositories || []).map(repositoryID).filter(Boolean)).size
}

export function planChangeCount(plans) {
  return (plans || []).flatMap((plan) => valueFor(plan, 'changes', 'Changes', []) || [])
    .filter((change) => !['unchanged', 'unmanaged'].includes(valueFor(change, 'operation', 'Operation', ''))).length
}

export function adapterChangeCounts(plans) {
  const counts = {}
  for (const plan of plans || []) {
    const adapter = valueFor(plan, 'adapter', 'Adapter', 'unknown')
    counts[adapter] = (counts[adapter] || 0) + planChangeCount([plan])
  }
  return counts
}

export function plansHaveIssueCode(plans, code) {
  return (plans || []).some((plan) => [
    ...(valueFor(plan, 'warnings', 'Warnings', []) || []),
    ...(valueFor(plan, 'blockers', 'Blockers', []) || []),
  ].some((issue) => valueFor(issue, 'code', 'Code', '') === code))
}

export function summarizeApplyResults(results) {
  const summary = { success: 0, unchanged: 0, stale: 0, blocked: 0, failed: 0, total: (results || []).length }
  for (const result of results || []) {
    const status = valueFor(result, 'status', 'Status', '')
    if (Object.hasOwn(summary, status)) summary[status] += 1
  }
  return summary
}

export function stalePlanTargetIDs(results) {
  return (results || [])
    .filter((result) => valueFor(result, 'status', 'Status', '') === 'stale')
    .map((result) => valueFor(valueFor(result, 'target', 'Target', {}), 'id', 'ID', ''))
    .filter(Boolean)
}

export function sharedTargetConflict(plan) {
  const blocker = (valueFor(plan, 'blockers', 'Blockers', []) || []).find((issue) => valueFor(issue, 'code', 'Code', '') === 'shared_target_conflict')
  if (!blocker) return null
  return {
    target: valueFor(blocker, 'detail', 'Detail', ''),
    paths: (valueFor(plan, 'aliases', 'Aliases', []) || []).map((alias) => valueFor(alias, 'path', 'Path', '')).filter(Boolean),
  }
}
