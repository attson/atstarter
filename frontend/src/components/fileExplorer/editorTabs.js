// editorTabs:编辑器标签页的状态机。
//
// 契约(brainstorm 定的):切走的标签把内容留在内存里,只在「关闭标签」时
// 才拦截未保存改动。所以这里只管标签的增删改和 dirty 标记,真正的文档状态
// 活在每个标签各自的 CodeEditor 实例里(FileBrowser 用 v-show 保活)。
//
// 全部是纯函数:接收 state 返回新 state,方便 node:test 覆盖。

import { rewritePrefix } from './pathOps.js'

// MAX_TABS 是同时保活的标签上限。每个标签一个 CodeMirror 实例,上限防止
// 连点几十个文件后内存和 DOM 失控。超限时淘汰最旧的「干净且非当前」标签;
// 全都是脏的就不淘汰(宁可多占内存,也不能悄悄丢用户的改动)。
export const MAX_TABS = 12

export function createTabsState() {
  return { tabs: [], activePath: '' }
}

function cloneTabs(state) {
  return state.tabs.map((t) => ({ ...t }))
}

function withTabs(tabs, activePath) {
  return { tabs, activePath }
}

function evictIfNeeded(tabs, activePath) {
  if (tabs.length <= MAX_TABS) return tabs
  const victim = tabs.findIndex((t) => !t.dirty && t.path !== activePath)
  if (victim === -1) return tabs
  const next = tabs.slice()
  next.splice(victim, 1)
  return next
}

export function openTab(state, path, viewMode = 'code') {
  if (!path) return state
  if (state.tabs.some((t) => t.path === path)) {
    return withTabs(cloneTabs(state), path)
  }
  const tabs = cloneTabs(state)
  tabs.push({ path, dirty: false, viewMode })
  return withTabs(evictIfNeeded(tabs, path), path)
}

export function activateTab(state, path) {
  if (!state.tabs.some((t) => t.path === path)) return state
  return withTabs(cloneTabs(state), path)
}

// closeTab 关掉一个标签。新的活动标签优先取右邻,没有右邻取左邻。
export function closeTab(state, path) {
  const index = state.tabs.findIndex((t) => t.path === path)
  if (index === -1) return state
  const tabs = cloneTabs(state)
  tabs.splice(index, 1)
  let activePath = state.activePath
  if (activePath === path) {
    const next = tabs[index] || tabs[index - 1]
    activePath = next ? next.path : ''
  }
  return withTabs(tabs, activePath)
}

export function closeOtherTabs(state, path) {
  if (!state.tabs.some((t) => t.path === path)) return state
  return withTabs(state.tabs.filter((t) => t.path === path).map((t) => ({ ...t })), path)
}

export function setTabDirty(state, path, dirty) {
  const tab = state.tabs.find((t) => t.path === path)
  if (!tab || tab.dirty === dirty) return state
  return withTabs(
    state.tabs.map((t) => (t.path === path ? { ...t, dirty } : { ...t })),
    state.activePath,
  )
}

export function setTabViewMode(state, path, viewMode) {
  const tab = state.tabs.find((t) => t.path === path)
  if (!tab || tab.viewMode === viewMode) return state
  return withTabs(
    state.tabs.map((t) => (t.path === path ? { ...t, viewMode } : { ...t })),
    state.activePath,
  )
}

// renamePath 在文件/目录被重命名或移动后改写受影响的标签路径。
// 目录改名要连它下面所有已打开的文件一起改。
export function renamePath(state, from, to) {
  const tabs = state.tabs.map((t) => ({ ...t, path: rewritePrefix(t.path, from, to) }))
  return withTabs(tabs, rewritePrefix(state.activePath, from, to))
}

// removePath 在文件/目录被删除后关掉它以及它下面的所有标签。
export function removePath(state, path) {
  let next = state
  for (const tab of state.tabs) {
    if (tab.path === path || tab.path.startsWith(path + '/')) {
      next = closeTab(next, tab.path)
    }
  }
  return next
}

// neighborTab 返回相对当前标签偏移 delta 的标签路径(用于 Ctrl+Tab / Alt+←→),
// 到头回绕。没有标签时返回空串。
export function neighborTab(state, delta) {
  if (!state.tabs.length) return ''
  const index = state.tabs.findIndex((t) => t.path === state.activePath)
  if (index === -1) return state.tabs[0].path
  const size = state.tabs.length
  return state.tabs[(((index + delta) % size) + size) % size].path
}

export function dirtyPaths(state) {
  return state.tabs.filter((t) => t.dirty).map((t) => t.path)
}

export function findTab(state, path) {
  return state.tabs.find((t) => t.path === path) || null
}
