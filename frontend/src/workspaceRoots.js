export function parseWorkspaceRoots(text) {
  const seen = new Set()
  const roots = []
  for (const line of String(text || '').split('\n')) {
    const root = line.trim()
    if (!root || seen.has(root)) continue
    seen.add(root)
    roots.push(root)
  }
  return roots
}

function parentDir(path) {
  const clean = String(path || '').replace(/[\\/]+$/, '')
  const i = clean.lastIndexOf('/')
  if (i <= 0) return ''
  return clean.slice(0, i)
}

export function inferWorkspaceRoots(projects) {
  const counts = new Map()
  const order = []
  for (const project of projects || []) {
    const path = project.path || ''
    if (path.includes('/.worktrees/') || path.includes('/.claude/worktrees/')) continue
    const parent = parentDir(path)
    if (!parent) continue
    if (!counts.has(parent)) order.push(parent)
    counts.set(parent, (counts.get(parent) || 0) + 1)
  }
  const common = order.filter((root) => counts.get(root) > 1)
  if (common.length) return common
  return order.slice(0, 1)
}
