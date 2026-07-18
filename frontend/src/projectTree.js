function commandLine(project) {
  return [project.command, ...(project.args || [])].filter(Boolean).join(' ')
}

function displayPath(path) {
  if (!path) return ''
  return path
    .replace(/^\/home\/[^/]+(?=\/)/, '~')
    .replace(/^\/Users\/[^/]+(?=\/)/, '~')
}

function pathSegments(path) {
  return displayPath(path).split('/').filter(Boolean)
}

function projectSearchText(project) {
  return [
    project.name,
    project.path,
    project.detectedType,
    commandLine(project),
  ].filter(Boolean).join(' ').toLowerCase()
}

function makeDirectory(label, path) {
  return {
    type: 'directory',
    id: 'dir:' + path,
    label,
    path,
    count: 0,
    children: [],
  }
}

function makeProjectNode(project, status) {
  return {
    type: 'project',
    id: project.id,
    project,
    status: status || {},
    commandLine: commandLine(project),
    children: [],
    count: 0,
  }
}

function worktreeInfo(path) {
  const display = displayPath(path)
  for (const marker of ['/.claude/worktrees/', '/.worktrees/']) {
    const idx = display.indexOf(marker)
    if (idx !== -1) {
      return {
        parentPath: display.slice(0, idx),
        marker,
      }
    }
  }
  return null
}

function ensureImmediateParent(rootMap, project) {
  const segments = pathSegments(project.path)
  segments.pop()
  const parentLabel = segments[segments.length - 1] || 'Projects'
  const parentPath = segments.join('/') || parentLabel

  if (!rootMap.has(parentPath)) {
    rootMap.set(parentPath, makeDirectory(parentLabel, parentPath))
  }
  const parent = rootMap.get(parentPath)
  parent.count += 1
  return parent
}

function findProjectNode(nodes, projectPath) {
  for (const node of nodes) {
    if (node.type === 'project' && displayPath(node.project.path) === projectPath) return node
    if (node.children) {
      const found = findProjectNode(node.children, projectPath)
      if (found) return found
    }
  }
  return null
}

function attachWorktrees(rootMap, worktrees) {
  for (const item of worktrees) {
    const parent = findProjectNode(Array.from(rootMap.values()), item.info.parentPath)
    if (!parent) {
      addNormalProject(rootMap, item.project, item.status)
      continue
    }
    parent.children.push(makeProjectNode(item.project, item.status))
    parent.count = parent.children.length
  }
}

function addProject(rootMap, project, status) {
  const info = worktreeInfo(project.path)
  if (info) return { project, status, info }
  addNormalProject(rootMap, project, status)
  return null
}

function addNormalProject(rootMap, project, status) {
  const parent = ensureImmediateParent(rootMap, project)
  parent.children.push(makeProjectNode(project, status))
}

function sortTree(nodes) {
  nodes.sort((a, b) => {
    if (a.type !== b.type) return a.type === 'directory' ? -1 : 1
    const aLabel = a.type === 'directory' ? a.label : a.project.name
    const bLabel = b.type === 'directory' ? b.label : b.project.name
    return aLabel.localeCompare(bLabel)
  })
  for (const node of nodes) {
    if (node.children) sortTree(node.children)
  }
}

export function buildProjectTree(projects = [], statuses = {}, query = '') {
  const normalizedQuery = query.trim().toLowerCase()
  const rootMap = new Map()
  const worktrees = []
  const normalProjects = []

  for (const project of projects) {
    const info = worktreeInfo(project.path)
    if (info) {
      if (!normalizedQuery || projectSearchText(project).includes(normalizedQuery)) {
        worktrees.push({ project, status: statuses[project.id], info })
      }
      continue
    }
    normalProjects.push(project)
    if (!normalizedQuery || projectSearchText(project).includes(normalizedQuery)) {
      addNormalProject(rootMap, project, statuses[project.id])
    }
  }

  if (normalizedQuery && worktrees.length) {
    for (const item of worktrees) {
      const parent = normalProjects.find((project) => displayPath(project.path) === item.info.parentPath)
      if (parent && !findProjectNode(Array.from(rootMap.values()), item.info.parentPath)) {
        addNormalProject(rootMap, parent, statuses[parent.id])
      }
    }
  }
  attachWorktrees(rootMap, worktrees)

  const roots = Array.from(rootMap.values())
  sortTree(roots)
  return roots
}
