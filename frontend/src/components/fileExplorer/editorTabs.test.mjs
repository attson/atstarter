import assert from 'node:assert/strict'
import test from 'node:test'
import {
  MAX_TABS,
  activateTab,
  closeOtherTabs,
  closeTab,
  createTabsState,
  dirtyPaths,
  findTab,
  neighborTab,
  openTab,
  removePath,
  renamePath,
  setTabDirty,
  setTabViewMode,
} from './editorTabs.js'

function withTabs(paths, active = paths[paths.length - 1]) {
  let state = createTabsState()
  for (const p of paths) state = openTab(state, p)
  return activateTab(state, active)
}

test('openTab 新建并激活,重复打开只激活不重复添加', () => {
  let state = openTab(createTabsState(), 'a.txt')
  assert.deepEqual(state.tabs.map((t) => t.path), ['a.txt'])
  assert.equal(state.activePath, 'a.txt')

  state = openTab(state, 'b.txt')
  state = openTab(state, 'a.txt')
  assert.deepEqual(state.tabs.map((t) => t.path), ['a.txt', 'b.txt'])
  assert.equal(state.activePath, 'a.txt')
})

test('openTab 忽略空路径', () => {
  const state = openTab(createTabsState(), '')
  assert.deepEqual(state.tabs, [])
})

test('closeTab 之后激活右邻,没有右邻则激活左邻', () => {
  let state = withTabs(['a', 'b', 'c'], 'b')
  state = closeTab(state, 'b')
  assert.equal(state.activePath, 'c')

  state = closeTab(state, 'c')
  assert.equal(state.activePath, 'a')

  state = closeTab(state, 'a')
  assert.equal(state.activePath, '')
  assert.deepEqual(state.tabs, [])
})

test('关掉非当前标签不改变当前标签', () => {
  let state = withTabs(['a', 'b', 'c'], 'a')
  state = closeTab(state, 'c')
  assert.equal(state.activePath, 'a')
})

test('setTabDirty 只改目标标签', () => {
  let state = withTabs(['a', 'b'])
  state = setTabDirty(state, 'a', true)
  assert.equal(findTab(state, 'a').dirty, true)
  assert.equal(findTab(state, 'b').dirty, false)
  assert.deepEqual(dirtyPaths(state), ['a'])
})

test('setTabViewMode 记住每个标签自己的视图', () => {
  let state = withTabs(['a.md', 'b.md'])
  state = setTabViewMode(state, 'a.md', 'render')
  assert.equal(findTab(state, 'a.md').viewMode, 'render')
  assert.equal(findTab(state, 'b.md').viewMode, 'code')
})

test('renamePath 跟随目录改名改写子孙标签', () => {
  let state = withTabs(['old/a.txt', 'old/deep/b.txt', 'keep.txt'], 'old/a.txt')
  state = renamePath(state, 'old', 'new')
  assert.deepEqual(state.tabs.map((t) => t.path), ['new/a.txt', 'new/deep/b.txt', 'keep.txt'])
  assert.equal(state.activePath, 'new/a.txt')
})

test('removePath 关掉被删目录下的所有标签', () => {
  let state = withTabs(['gone/a.txt', 'gone/b.txt', 'stay.txt'], 'gone/a.txt')
  state = removePath(state, 'gone')
  assert.deepEqual(state.tabs.map((t) => t.path), ['stay.txt'])
  assert.equal(state.activePath, 'stay.txt')
})

test('neighborTab 到头回绕', () => {
  const state = withTabs(['a', 'b', 'c'], 'c')
  assert.equal(neighborTab(state, 1), 'a')
  assert.equal(neighborTab(state, -1), 'b')
  assert.equal(neighborTab(createTabsState(), 1), '')
})

test('closeOtherTabs 只留一个', () => {
  const state = closeOtherTabs(withTabs(['a', 'b', 'c'], 'a'), 'b')
  assert.deepEqual(state.tabs.map((t) => t.path), ['b'])
  assert.equal(state.activePath, 'b')
})

test('超过上限时淘汰最旧的干净标签', () => {
  let state = createTabsState()
  for (let i = 0; i < MAX_TABS; i++) state = openTab(state, `f${i}.txt`)
  state = openTab(state, 'extra.txt')
  assert.equal(state.tabs.length, MAX_TABS)
  assert.equal(findTab(state, 'f0.txt'), null, '最旧的干净标签被淘汰')
  assert.ok(findTab(state, 'extra.txt'))
})

test('全部标签都脏时不淘汰,宁可超上限也不丢改动', () => {
  let state = createTabsState()
  for (let i = 0; i < MAX_TABS; i++) {
    state = openTab(state, `f${i}.txt`)
    state = setTabDirty(state, `f${i}.txt`, true)
  }
  state = openTab(state, 'extra.txt')
  assert.equal(state.tabs.length, MAX_TABS + 1)
  assert.equal(dirtyPaths(state).length, MAX_TABS)
})
