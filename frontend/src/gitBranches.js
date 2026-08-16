// gitBranches:分支选择器的纯逻辑。后端 internal/git 返回 Branches
// ({ repo, branch, detached, dirty, head, local[], remote[] }),这里只做
// 过滤、分组、命名校验和提醒文案。

// 与 internal/git 的 validateName 保持同一套规则。前端先挡一道是为了即时反馈,
// 后端那道才是真正的闸 —— 两边都不能省。
const INVALID_CHARS = /[ \t~^:?*[\\]/

export function validateBranchName(name, existing = []) {
  const trimmed = (name || '').trim()
  if (!trimmed) return '分支名不能为空'
  if (trimmed.startsWith('-')) return '分支名不能以 - 开头'
  if (INVALID_CHARS.test(trimmed)) return '分支名不能包含空格或 ~ ^ : ? * [ \\'
  if (trimmed.startsWith('/') || trimmed.endsWith('/') || trimmed.includes('//')) {
    return '分支名不能以 / 开头或结尾,也不能包含 //'
  }
  if (trimmed.startsWith('.') || trimmed.includes('..') || trimmed.endsWith('.lock')) {
    return '分支名不能以 . 开头、包含 .. 或以 .lock 结尾'
  }
  if (trimmed.includes('@{')) return '分支名不能包含 @{'
  if (existing.includes(trimmed)) return '分支已存在'
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
  if (local.length) groups.push({ id: 'local', label: '本地分支', items: local })
  if (remote.length) groups.push({ id: 'remote', label: '远端分支', items: remote })
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
  if (status?.dirty) warnings.push('工作区有未提交改动,git 可能拒绝切换。')
  if (running) warnings.push('项目正在运行,切换分支不会自动重启进程。')
  return warnings
}

// canCreateFromQuery:搜索框里敲了一个不存在的合法分支名时,列表底部给一个
// 「新建分支」入口。这是 VS Code / lazygit 的通行做法,省掉一个额外按钮。
export function canCreateFromQuery(data, query) {
  const trimmed = (query || '').trim()
  if (!trimmed) return false
  return validateBranchName(trimmed, allBranchNames(data)) === ''
}
