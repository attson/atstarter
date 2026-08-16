// fileClipboard:应用内的文件剪贴板(复制/剪切/粘贴)。
//
// 剪贴板必须活在组件之外:FileBrowser 是按项目挂载的,跨项目粘贴意味着
// 「在 A 项目复制 → 切到 B 项目 → 粘贴」,组件早就被卸载重建了。
// 这里导出一个模块级单例,并保留工厂函数供测试隔离。
//
// 剪贴板只存 { projectId, relPath },真正的拷贝由后端 CopyProjectPath /
// MoveProjectPath 完成,两端各自过自己项目根的 guard。

import { baseName, isIntoSelf, isSameDirMove, joinPath } from './pathOps.js'

function normalize(projectId, items, mode) {
  const clean = (items || [])
    .filter((it) => it && typeof it.path === 'string' && it.path !== '')
    .map((it) => ({ path: it.path, isDir: !!it.isDir }))
  if (!projectId || !clean.length) return null
  return { projectId, mode, items: clean }
}

export function createFileClipboard() {
  let entry = null
  const listeners = new Set()

  function emit() {
    for (const fn of [...listeners]) fn(entry)
  }

  return {
    copy(projectId, items) {
      entry = normalize(projectId, items, 'copy')
      emit()
    },
    cut(projectId, items) {
      entry = normalize(projectId, items, 'cut')
      emit()
    },
    read() {
      return entry
    },
    clear() {
      entry = null
      emit()
    },
    subscribe(fn) {
      listeners.add(fn)
      return () => listeners.delete(fn)
    },
  }
}

export const fileClipboard = createFileClipboard()

// planPaste 把一次粘贴展开成后端调用列表。返回 { ops, error }:
// error 非空表示整次粘贴不合法,不执行任何一条 op(要么全做要么不做)。
//
// 规则:
//   - 把目录粘到它自己或子孙里 → 拒绝(后端也会拒,这里提前给出中文提示)。
//   - 剪切到原目录 → 跳过该条(不是错误,只是无事可做)。
//   - 复制到原目录 → 保留,后端 UniqueName 会生成「x copy」,这正是「副本」语义。
export function planPaste(entry, targetProjectId, targetDir) {
  if (!entry || !entry.items.length) return { ops: [], error: '' }
  const sameProject = entry.projectId === targetProjectId
  const ops = []
  for (const item of entry.items) {
    if (sameProject && item.isDir && isIntoSelf(item.path, targetDir)) {
      return { ops: [], error: `不能把「${baseName(item.path)}」粘贴到它自己内部` }
    }
    if (entry.mode === 'cut' && sameProject && isSameDirMove(item.path, targetDir)) continue
    ops.push({
      mode: entry.mode,
      srcProjectId: entry.projectId,
      srcPath: item.path,
      dstProjectId: targetProjectId,
      dstPath: joinPath(targetDir, baseName(item.path)),
    })
  }
  return { ops, error: '' }
}

// planDrop 把一次拖拽落点展开成移动 op 列表。拖拽永远是移动(同项目内)。
export function planDrop(paths, projectId, targetDir) {
  const ops = []
  for (const item of paths) {
    if (item.isDir && isIntoSelf(item.path, targetDir)) {
      return { ops: [], error: `不能把「${baseName(item.path)}」移动到它自己内部` }
    }
    if (isSameDirMove(item.path, targetDir)) continue
    ops.push({
      mode: 'cut',
      srcProjectId: projectId,
      srcPath: item.path,
      dstProjectId: projectId,
      dstPath: joinPath(targetDir, baseName(item.path)),
    })
  }
  return { ops, error: '' }
}
