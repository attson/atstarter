export const PROJECT_LIST_VIEW_KEY = 'projectList.viewMode'
export const DEFAULT_PROJECT_LIST_VIEW = 'list'

export function normalizeProjectListView(value) {
  return value === 'card' ? 'card' : DEFAULT_PROJECT_LIST_VIEW
}

export function readProjectListView(storage) {
  if (!storage) return DEFAULT_PROJECT_LIST_VIEW
  try {
    return normalizeProjectListView(storage.getItem(PROJECT_LIST_VIEW_KEY))
  } catch {
    return DEFAULT_PROJECT_LIST_VIEW
  }
}

export function writeProjectListView(storage, value) {
  const next = normalizeProjectListView(value)
  if (storage) {
    try {
      storage.setItem(PROJECT_LIST_VIEW_KEY, next)
    } catch {
      // A display preference must not break project navigation.
    }
  }
  return next
}

export function advanceBranchRevision(revisions, projectId) {
  if (!projectId) return revisions
  return {
    ...revisions,
    [projectId]: (revisions[projectId] || 0) + 1,
  }
}

export function applyBranchEntry(entries, revisions, projectId, entry, expectedRevision = null) {
  if (!projectId) return entries
  if (expectedRevision !== null && (revisions[projectId] || 0) !== expectedRevision) {
    return entries
  }
  return { ...entries, [projectId]: entry }
}

export async function loadProjectBranches(projects, getBranch, onEntry = () => {}, concurrency = 4) {
  const unique = []
  const seen = new Set()
  for (const project of projects || []) {
    if (!project?.id || seen.has(project.id)) continue
    seen.add(project.id)
    unique.push(project)
  }

  const entries = {}
  const limit = Math.max(1, Math.min(unique.length || 1, Math.floor(Number(concurrency)) || 1))
  let cursor = 0

  async function worker() {
    while (cursor < unique.length) {
      const project = unique[cursor]
      cursor += 1
      let entry
      try {
        const branch = await getBranch(project.path)
        entry = { state: 'ready', branch: String(branch || '').trim() }
      } catch {
        entry = { state: 'error', branch: '' }
      }
      entries[project.id] = entry
      onEntry(project.id, entry)
    }
  }

  await Promise.all(Array.from({ length: limit }, () => worker()))
  return entries
}
