// pathOps:文件树用的 relPath 纯函数。
// 路径体系与后端 filetree 一致:相对项目根、"/" 分隔、空串 "" 表示根。
// 这些函数原本散落在 FileTree.vue / FileTreeNode.vue 里各写一份,
// 拷贝/移动/拖拽引入更多路径判断后集中到这里,便于 node:test 覆盖。

export function joinPath(parent, name) {
  if (!parent) return name // root(空 relPath)下的子项不带前导 /
  return parent.endsWith('/') ? parent + name : parent + '/' + name
}

export function parentDir(p) {
  const i = p.lastIndexOf('/')
  return i < 0 ? '' : p.slice(0, i)
}

export function baseName(p) {
  const i = p.lastIndexOf('/')
  return i === -1 ? p : p.slice(i + 1)
}

// isDescendant 判断 path 是否严格位于 parent 之下。parent 为 "" 时代表根,
// 除根自身外的一切路径都是它的子孙。
export function isDescendant(path, parent) {
  if (path === parent) return false
  if (parent === '') return path !== ''
  return path.startsWith(parent + '/')
}

// targetDirFor 把一个树节点解析成「新建/粘贴应该落到哪个目录」:
// 目录节点落到自身,文件节点落到它的父目录,空节点(空白区/根)落到根。
export function targetDirFor(node) {
  if (!node) return ''
  return node.isDir ? node.path : parentDir(node.path)
}

// isSameDirMove 判断把 path 挪到 targetDir 是不是原地移动(什么都不用做)。
export function isSameDirMove(path, targetDir) {
  return parentDir(path) === targetDir
}

// isIntoSelf 判断把 path 拷/移到 targetDir 会不会落进它自己内部。
// 只有目录才可能出现这种情况,但函数本身不关心类型,由调用方判断。
export function isIntoSelf(path, targetDir) {
  return targetDir === path || isDescendant(targetDir, path)
}

// rewritePrefix 在某个路径被重命名/移动后,把受影响的旧路径改写成新路径。
// path 等于 from 或位于 from 之下时改写,其余原样返回。
export function rewritePrefix(path, from, to) {
  if (path === from) return to
  if (isDescendant(path, from)) return to + path.slice(from.length)
  return path
}
