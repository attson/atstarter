// dockerState.js — 容器快照的分组与筛选纯函数。

// groupContainers 把容器按 compose 归属分组。
export function groupContainers(list) {
  const compose = {}
  const standalone = []
  for (const c of list || []) {
    if (c.compose) {
      ;(compose[c.compose] ||= []).push(c)
    } else {
      standalone.push(c)
    }
  }
  return { compose, standalone }
}

// filterContainers 按名字子串(大小写不敏感)筛选;空关键字返回全部。
export function filterContainers(list, keyword) {
  const kw = (keyword || '').trim().toLowerCase()
  if (!kw) return list || []
  return (list || []).filter((c) => (c.name || '').toLowerCase().includes(kw))
}

export function summarizeContainers(list) {
  const images = []
  const seenImages = new Set()
  let running = 0
  let exited = 0
  let workingDir = ''
  for (const c of list || []) {
    if (c.state === 'running') running++
    if (c.state === 'exited' || c.state === 'dead') exited++
    if (!workingDir && c.composeWorkingDir) workingDir = c.composeWorkingDir
    if (c.image && !seenImages.has(c.image)) {
      seenImages.add(c.image)
      images.push(c.image)
    }
  }
  return {
    total: (list || []).length,
    running,
    exited,
    workingDir,
    images,
  }
}

export function normalizeComposeProjectName(name) {
  return String(name || '')
    .toLowerCase()
    .split('')
    .filter((ch) => (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch === '_' || ch === '-')
    .join('')
}

function basename(path) {
  const parts = String(path || '').split(/[\\/]/).filter(Boolean)
  return parts.length ? parts[parts.length - 1] : ''
}

export function findComposeProject(projects, composeName) {
  const normalized = normalizeComposeProjectName(composeName)
  return (projects || []).find((p) => {
    if (p.detectedType !== 'compose') return false
    return normalizeComposeProjectName(basename(p.path)) === normalized
  }) || null
}
