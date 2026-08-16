import assert from 'node:assert/strict'
import test from 'node:test'
import {
  findRow,
  flattenVisible,
  moveFocus,
  rangePaths,
  toggleSelection,
  treeKeyAction,
} from './treeKeyboard.js'

// root/
//   sub/          (展开)
//     deep/       (折叠,子项不可见)
//       hidden.txt
//     b.txt
//   a.txt
const tree = [
  {
    path: 'sub',
    isDir: true,
    expanded: true,
    children: [
      {
        path: 'sub/deep',
        isDir: true,
        expanded: false,
        children: [{ path: 'sub/deep/hidden.txt', isDir: false, children: null }],
      },
      { path: 'sub/b.txt', isDir: false, children: null },
    ],
  },
  { path: 'a.txt', isDir: false, children: null },
]

test('flattenVisible 不摊开折叠目录的子项', () => {
  const flat = flattenVisible(tree)
  assert.deepEqual(flat.map((r) => r.path), ['sub', 'sub/deep', 'sub/b.txt', 'a.txt'])
  assert.deepEqual(flat.map((r) => r.level), [0, 1, 1, 0])
})

test('moveFocus 在两端停住而不回绕', () => {
  const flat = flattenVisible(tree)
  assert.equal(moveFocus(flat, 'sub', 1), 'sub/deep')
  assert.equal(moveFocus(flat, 'a.txt', 1), 'a.txt')
  assert.equal(moveFocus(flat, 'sub', -1), 'sub')
  assert.equal(moveFocus(flat, 'unknown', 1), 'sub', '未知路径从首行开始')
  assert.equal(moveFocus([], 'sub', 1), '')
})

test('rangePaths 两个方向都取闭区间', () => {
  const flat = flattenVisible(tree)
  assert.deepEqual(rangePaths(flat, 'sub/deep', 'a.txt'), ['sub/deep', 'sub/b.txt', 'a.txt'])
  assert.deepEqual(rangePaths(flat, 'a.txt', 'sub/deep'), ['sub/deep', 'sub/b.txt', 'a.txt'])
  assert.deepEqual(rangePaths(flat, 'nope', 'a.txt'), ['a.txt'])
})

test('toggleSelection 增删互斥', () => {
  const once = toggleSelection(new Set(), 'a.txt')
  assert.deepEqual([...once], ['a.txt'])
  assert.deepEqual([...toggleSelection(once, 'a.txt')], [])
})

test('findRow 找得到可见行', () => {
  const flat = flattenVisible(tree)
  assert.equal(findRow(flat, 'sub/b.txt').isDir, false)
  assert.equal(findRow(flat, 'sub/deep/hidden.txt'), null, '折叠目录里的行不可见')
})

function key(k, mods = {}) {
  return { key: k, ctrlKey: false, metaKey: false, shiftKey: false, ...mods }
}

test('方向键在目录上先展开、展开后才下移', () => {
  const collapsed = { path: 'sub/deep', isDir: true, expanded: false }
  assert.deepEqual(treeKeyAction(key('ArrowRight'), collapsed), { type: 'expand' })

  const expanded = { path: 'sub', isDir: true, expanded: true }
  assert.deepEqual(treeKeyAction(key('ArrowRight'), expanded), { type: 'move', delta: 1, extend: false })

  assert.deepEqual(treeKeyAction(key('ArrowLeft'), expanded), { type: 'collapse' })
  assert.deepEqual(
    treeKeyAction(key('ArrowLeft'), { path: 'sub/b.txt', isDir: false }),
    { type: 'focus', path: 'sub' },
  )
  assert.equal(treeKeyAction(key('ArrowLeft'), { path: 'a.txt', isDir: false }), null)
})

test('Shift+方向键带上 extend 标记', () => {
  assert.deepEqual(
    treeKeyAction(key('ArrowDown', { shiftKey: true }), { path: 'a.txt', isDir: false }),
    { type: 'move', delta: 1, extend: true },
  )
})

test('Enter 对目录是展开收起,对文件是打开', () => {
  assert.deepEqual(treeKeyAction(key('Enter'), { path: 'sub', isDir: true }), { type: 'toggle' })
  assert.deepEqual(treeKeyAction(key('Enter'), { path: 'a.txt', isDir: false }), { type: 'open' })
  assert.equal(treeKeyAction(key('Enter'), null), null)
})

test('F2 / Delete / Escape', () => {
  const row = { path: 'a.txt', isDir: false }
  assert.deepEqual(treeKeyAction(key('F2'), row), { type: 'rename' })
  assert.deepEqual(treeKeyAction(key('Delete'), row), { type: 'delete' })
  assert.deepEqual(treeKeyAction(key('Escape'), row), { type: 'clearSelection' })
  assert.equal(treeKeyAction(key('F2'), null), null)
})

test('Ctrl/Cmd 组合键映射到剪贴板动作', () => {
  const row = { path: 'a.txt', isDir: false }
  assert.deepEqual(treeKeyAction(key('c', { ctrlKey: true }), row), { type: 'copy' })
  assert.deepEqual(treeKeyAction(key('X', { metaKey: true }), row), { type: 'cut' })
  assert.deepEqual(treeKeyAction(key('v', { metaKey: true }), row), { type: 'paste' })
  assert.deepEqual(treeKeyAction(key('d', { ctrlKey: true }), row), { type: 'duplicate' })
  assert.deepEqual(treeKeyAction(key('a', { ctrlKey: true }), row), { type: 'selectAll' })
  assert.equal(treeKeyAction(key('z', { ctrlKey: true }), row), null)
})
