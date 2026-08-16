import assert from 'node:assert/strict'
import test from 'node:test'
import { createFileClipboard, planDrop, planPaste } from './fileClipboard.js'

test('剪贴板记住来源项目,支持跨项目粘贴', () => {
  const clip = createFileClipboard()
  assert.equal(clip.read(), null)
  clip.copy('proj-a', [{ path: 'sub/a.txt', isDir: false }])
  assert.deepEqual(clip.read(), {
    projectId: 'proj-a',
    mode: 'copy',
    items: [{ path: 'sub/a.txt', isDir: false }],
  })
  clip.cut('proj-a', [{ path: 'sub', isDir: true }])
  assert.equal(clip.read().mode, 'cut')
  clip.clear()
  assert.equal(clip.read(), null)
})

test('剪贴板忽略空条目', () => {
  const clip = createFileClipboard()
  clip.copy('proj-a', [])
  assert.equal(clip.read(), null)
  clip.copy('', [{ path: 'a.txt' }])
  assert.equal(clip.read(), null)
})

test('订阅者能收到剪贴板变化', () => {
  const clip = createFileClipboard()
  const seen = []
  const off = clip.subscribe((entry) => seen.push(entry?.mode ?? null))
  clip.copy('p', [{ path: 'a.txt' }])
  clip.clear()
  off()
  clip.copy('p', [{ path: 'b.txt' }])
  assert.deepEqual(seen, ['copy', null])
})

test('planPaste 跨项目时生成两端项目 ID', () => {
  const entry = { projectId: 'proj-a', mode: 'copy', items: [{ path: 'sub/a.txt', isDir: false }] }
  const { ops, error } = planPaste(entry, 'proj-b', 'vendor')
  assert.equal(error, '')
  assert.deepEqual(ops, [{
    mode: 'copy',
    srcProjectId: 'proj-a',
    srcPath: 'sub/a.txt',
    dstProjectId: 'proj-b',
    dstPath: 'vendor/a.txt',
  }])
})

test('planPaste 拒绝把目录粘进它自己内部', () => {
  const entry = { projectId: 'p', mode: 'copy', items: [{ path: 'sub', isDir: true }] }
  const { ops, error } = planPaste(entry, 'p', 'sub/deep')
  assert.deepEqual(ops, [])
  assert.match(error, /自己内部/)
})

test('planPaste 同项目复制到原目录保留为副本', () => {
  const entry = { projectId: 'p', mode: 'copy', items: [{ path: 'sub/a.txt', isDir: false }] }
  const { ops } = planPaste(entry, 'p', 'sub')
  assert.equal(ops.length, 1, '复制到原目录 = 生成副本,后端会自动改名')
  assert.equal(ops[0].dstPath, 'sub/a.txt')
})

test('planPaste 同项目剪切到原目录是空操作', () => {
  const entry = { projectId: 'p', mode: 'cut', items: [{ path: 'sub/a.txt', isDir: false }] }
  const { ops, error } = planPaste(entry, 'p', 'sub')
  assert.deepEqual(ops, [])
  assert.equal(error, '')
})

test('planPaste 跨项目时同名目录不算落进自己', () => {
  const entry = { projectId: 'proj-a', mode: 'copy', items: [{ path: 'sub', isDir: true }] }
  const { ops, error } = planPaste(entry, 'proj-b', 'sub/deep')
  assert.equal(error, '')
  assert.equal(ops.length, 1)
})

test('planDrop 生成移动 op 并跳过原地拖拽', () => {
  const dropped = planDrop([{ path: 'sub/a.txt', isDir: false }], 'p', 'sub')
  assert.deepEqual(dropped.ops, [])

  const moved = planDrop([{ path: 'sub/a.txt', isDir: false }], 'p', 'other')
  assert.deepEqual(moved.ops, [{
    mode: 'cut',
    srcProjectId: 'p',
    srcPath: 'sub/a.txt',
    dstProjectId: 'p',
    dstPath: 'other/a.txt',
  }])
})

test('planDrop 拒绝把目录拖进自己的子孙', () => {
  const { ops, error } = planDrop([{ path: 'sub', isDir: true }], 'p', 'sub/deep')
  assert.deepEqual(ops, [])
  assert.match(error, /自己内部/)
})
