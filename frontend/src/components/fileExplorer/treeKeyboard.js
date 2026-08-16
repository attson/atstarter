// treeKeyboard:文件树的键盘导航与多选。
// 纯函数,只吃「扁平化后的可见行」和一个按键描述,吐出一个动作;
// 真正改 DOM/发请求由 FileTree.vue 做。

import { parentDir } from './pathOps.js'

// flattenVisible 把树摊平成当前肉眼可见的行序列(折叠的目录不展开其子项)。
// 这是键盘上下移动和 Shift 范围选择的唯一顺序依据。
export function flattenVisible(nodes, level = 0, out = []) {
  for (const node of nodes || []) {
    out.push({ path: node.path, isDir: node.isDir, expanded: !!node.expanded, level })
    if (node.isDir && node.expanded && node.children) {
      flattenVisible(node.children, level + 1, out)
    }
  }
  return out
}

export function findRow(flat, path) {
  return flat.find((row) => row.path === path) || null
}

// moveFocus 返回沿可见行移动 delta 之后的路径。越界时停在两端(不回绕:
// 文件树里回绕会让人以为跳错了地方)。当前路径不在可见行里时从首行开始。
export function moveFocus(flat, current, delta) {
  if (!flat.length) return ''
  const index = flat.findIndex((row) => row.path === current)
  if (index === -1) return flat[0].path
  const next = index + delta
  if (next < 0) return flat[0].path
  if (next >= flat.length) return flat[flat.length - 1].path
  return flat[next].path
}

// rangePaths 返回 anchor 到 target 之间(含两端)所有可见行的路径,
// 用于 Shift 点击 / Shift+↑↓ 的范围选择。
export function rangePaths(flat, anchor, target) {
  const a = flat.findIndex((row) => row.path === anchor)
  const b = flat.findIndex((row) => row.path === target)
  if (a === -1 || b === -1) return target ? [target] : []
  const [from, to] = a <= b ? [a, b] : [b, a]
  return flat.slice(from, to + 1).map((row) => row.path)
}

// toggleSelection 是 Ctrl/Cmd 点击的语义:已选则取消,未选则加入。
export function toggleSelection(selected, path) {
  const next = new Set(selected)
  if (next.has(path)) next.delete(path)
  else next.add(path)
  return next
}

// treeKeyAction 把一次 keydown 翻译成动作。row 是当前聚焦行(可能为 null)。
// 返回 null 表示这个按键与文件树无关,交给浏览器默认行为。
export function treeKeyAction(event, row) {
  const key = event.key
  const mod = event.ctrlKey || event.metaKey

  if (mod) {
    switch (key.toLowerCase()) {
      case 'c': return { type: 'copy' }
      case 'x': return { type: 'cut' }
      case 'v': return { type: 'paste' }
      case 'd': return { type: 'duplicate' }
      case 'a': return { type: 'selectAll' }
      default: return null
    }
  }

  switch (key) {
    case 'ArrowDown': return { type: 'move', delta: 1, extend: event.shiftKey }
    case 'ArrowUp': return { type: 'move', delta: -1, extend: event.shiftKey }
    case 'ArrowRight':
      if (!row || !row.isDir) return null
      return row.expanded ? { type: 'move', delta: 1, extend: false } : { type: 'expand' }
    case 'ArrowLeft':
      if (row && row.isDir && row.expanded) return { type: 'collapse' }
      if (row && parentDir(row.path) !== '') return { type: 'focus', path: parentDir(row.path) }
      return null
    case 'Enter': return row ? { type: row.isDir ? 'toggle' : 'open' } : null
    case 'F2': return row ? { type: 'rename' } : null
    case 'Delete': return row ? { type: 'delete' } : null
    case 'Escape': return { type: 'clearSelection' }
    default: return null
  }
}
