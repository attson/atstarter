// gitBranches:分支选择器的纯逻辑。后端 internal/git 返回 Branches
// ({ repo, branch, detached, dirty, head, local[], remote[] }),这里只做
// 过滤、分组、命名校验和稳定的提醒代码。

// 与 internal/git 的 validateName 保持同一套规则。前端先挡一道是为了即时反馈,
// 后端那道才是真正的闸 —— 两边都不能省。
const INVALID_CHARS = /[ \t~^:?*[\\]/

export function validateBranchName(name, existing = []) {
  const trimmed = (name || '').trim()
  if (!trimmed) return { code: 'empty', values: {} }
  if (trimmed.startsWith('-')) return { code: 'leadingDash', values: {} }
  if (INVALID_CHARS.test(trimmed)) return { code: 'invalidCharacters', values: {} }
  if (trimmed.startsWith('/') || trimmed.endsWith('/') || trimmed.includes('//')) {
    return { code: 'slash', values: {} }
  }
  if (trimmed.startsWith('.') || trimmed.includes('..') || trimmed.endsWith('.lock')) {
    return { code: 'dot', values: {} }
  }
  if (trimmed.includes('@{')) return { code: 'reflog', values: {} }
  if (existing.includes(trimmed)) return { code: 'exists', values: { name: trimmed } }
  return ''
}

// branchGroups 把后端数据摊成「本地 / 远端」两组,按查询词过滤。
// 远端组里的分支切过去时 git 会自动建立跟踪分支,所以不需要额外的 UI 区分动作。
export function branchGroups(data, query = '') {
  const q = query.trim().toLowerCase()
  const match = (name) => !q || name.toLowerCase().includes(q)
  const groups = []
  const local = (data?.local || []).filter(match).map((name) => ({
    name,
    kind: 'local',
    current: name === data?.branch,
  }))
  const remote = (data?.remote || []).filter(match).map((name) => ({
    name,
    kind: 'remote',
    current: false,
  }))
  if (local.length) groups.push({ id: 'local', items: local })
  if (remote.length) groups.push({ id: 'remote', items: remote })
  return groups
}

// allBranchNames 是「新建分支」重名校验的依据:本地已有的都算占用,
// 远端同名的不算(切过去会建同名本地分支,那是 checkout 不是 create)。
export function allBranchNames(data) {
  return [...(data?.local || [])]
}

// branchPillLabel 决定分支标签上显示什么。非仓库返回空串,前端据此整个隐藏。
export function branchPillLabel(status) {
  if (!status?.repo) return ''
  if (status.detached) return status.head ? `detached @ ${status.head}` : 'detached'
  return status.branch || ''
}

// checkoutWarnings 返回切换前值得提醒的事项。不阻止操作 —— git 自己会拒绝
// 真正危险的切换,这里只是让用户心里有数。
export function checkoutWarnings(status, running) {
  const warnings = []
  if (status?.dirty) warnings.push('dirty')
  if (running) warnings.push('running')
  return warnings
}

// canCreateFromQuery:搜索框里敲了一个不存在的合法分支名时,列表底部给一个
// 「新建分支」入口。这是 VS Code / lazygit 的通行做法,省掉一个额外按钮。
export function canCreateFromQuery(data, query) {
  const trimmed = (query || '').trim()
  if (!trimmed) return false
  return validateBranchName(trimmed, allBranchNames(data)) === ''
}

// Wails may reject with a string instead of an Error. Never trim or wrap Git stderr.
export function gitErrorDetail(error) {
  if (typeof error === 'string') return error
  return typeof error?.message === 'string' ? error.message : ''
}
